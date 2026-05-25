package ir

import (
	"fmt"

	"github.com/rroblf01/gopy/parser"
)

// Lower converts a Python AST Module node into an IR Module.
// Unsupported constructs return an error so we fail loudly rather than
// silently producing garbage Go.
func Lower(modName string, root parser.Node) (*Module, error) {
	if root.Type() != "Module" {
		return nil, fmt.Errorf("expected Module root, got %q", root.Type())
	}
	moduleScopeReset()
	m := &Module{Name: modName}
	// Pre-pass: capture imports nested inside functions / classes too,
	// so `from itertools import chain` inside `def main():` still
	// registers the alias before name resolution runs.
	collectNestedImports(root, m)
	for _, stmt := range root.Children("body") {
		if isMainGuard(stmt) {
			// `if __name__ == "__main__":` — Go runs main() automatically,
			// so we drop the guarded block entirely.
			continue
		}
		// Capture imports so transpile can resolve `from datetime import datetime`
		// style aliases. Skip emitting them as decls.
		if stmt.Type() == "Import" {
			for _, alias := range stmt.Children("names") {
				m.Imports = append(m.Imports, Import{
					Names: []ImportName{{Name: alias.Str("name"), Alias: alias.Str("asname")}},
				})
			}
			continue
		}
		if stmt.Type() == "ImportFrom" {
			from := stmt.Str("module")
			imp := Import{From: from}
			for _, alias := range stmt.Children("names") {
				imp.Names = append(imp.Names, ImportName{Name: alias.Str("name"), Alias: alias.Str("asname")})
			}
			m.Imports = append(m.Imports, imp)
			continue
		}
		decls, err := lowerTopLevel(stmt)
		if err != nil {
			// Prefix the originating module name (typically the .py
			// filename) so multi-file build output points at the
			// offending source. The "line N: ..." prefix from the
			// inner Errorf is preserved verbatim.
			return nil, fmt.Errorf("%s: %v", modName, err)
		}
		if decls == nil {
			continue
		}
		m.Decls = append(m.Decls, decls...)
	}
	return m, nil
}

// collectNestedImports walks function / class bodies looking for Import
// and ImportFrom statements, mirroring them into m.Imports so the alias
// resolver picks them up regardless of where in the source they appear.
func collectNestedImports(n parser.Node, m *Module) {
	if n == nil {
		return
	}
	walkImports(n, m, true)
}

func walkImports(n parser.Node, m *Module, isRoot bool) {
	if n == nil {
		return
	}
	t := n.Type()
	// Module root: descend without recording top-level imports here (the
	// main loop already does that). For nested nodes, record each Import.
	if !isRoot {
		if t == "Import" {
			for _, alias := range n.Children("names") {
				m.Imports = append(m.Imports, Import{
					Names: []ImportName{{Name: alias.Str("name"), Alias: alias.Str("asname")}},
				})
			}
			return
		}
		if t == "ImportFrom" {
			from := n.Str("module")
			imp := Import{From: from}
			for _, alias := range n.Children("names") {
				imp.Names = append(imp.Names, ImportName{Name: alias.Str("name"), Alias: alias.Str("asname")})
			}
			m.Imports = append(m.Imports, imp)
			return
		}
	}
	for _, k := range []string{"body", "orelse", "finalbody", "handlers"} {
		for _, c := range n.Children(k) {
			walkImports(c, m, false)
		}
	}
}

// isMainGuard matches the Python idiom `if __name__ == "__main__": ...`.
// The transpiler skips it because Go auto-invokes main() at startup.
func isMainGuard(n parser.Node) bool {
	if n.Type() != "If" {
		return false
	}
	test := n.Child("test")
	if test.Type() != "Compare" {
		return false
	}
	left := test.Child("left")
	ops := test.Children("ops")
	comps := test.Children("comparators")
	if left == nil || len(ops) != 1 || len(comps) != 1 {
		return false
	}
	if left.Type() != "Name" || left.Str("id") != "__name__" {
		return false
	}
	if ops[0].Type() != "Eq" {
		return false
	}
	c := comps[0]
	if c.Type() != "Constant" {
		return false
	}
	v, _ := c["value"].(string)
	return v == "__main__"
}

func lowerTopLevel(n parser.Node) ([]Decl, error) {
	switch n.Type() {
	case "FunctionDef", "AsyncFunctionDef":
		f, err := lowerFunc(n)
		if err != nil {
			return nil, err
		}
		if f == nil {
			// @overload stub — skip emission so the real impl that
			// follows wins at codegen.
			return nil, nil
		}
		return []Decl{f}, nil
	case "ClassDef":
		return lowerClass(n)
	case "ImportFrom", "Import":
		// F3: every transpiled .py in the same directory lands in the same
		// Go package, so cross-module names resolve naturally without
		// qualifiers. We drop the import statement here. Stdlib imports
		// (re, json, etc.) are not yet supported and will produce
		// undefined-name errors downstream.
		return nil, nil
	case "Assign":
		// Top-level `name = expr`. We only support single-target Name
		// assignments here; tuple unpack / chained at module level land
		// later if needed.
		targets := n.Children("targets")
		if len(targets) != 1 || targets[0].Type() != "Name" {
			return nil, fmt.Errorf("line %d: module-level assignment must be `name = expr`", n.Lineno())
		}
		name := targets[0].Str("id")
		// `T = TypeVar("T")` — register the name as a module-level type
		// variable. Subsequent function annotations referencing T will
		// pick it up as a Go generic type parameter; no Decl is emitted.
		if val := n.Child("value"); val != nil && val.Type() == "Call" {
			if fn := val.Child("func"); fn != nil && fn.Type() == "Name" {
				if id := fn.Str("id"); id == "TypeVar" || id == "ParamSpec" || id == "TypeVarTuple" {
					moduleTypeVars[name] = true
					return nil, nil
				}
			}
		}
		// `Point = namedtuple("Point", ["x", "y"])` → synthesize a Class
		// with one struct field per name (all typed `any` since the
		// per-field types aren't known at declaration time). Constructors
		// route through the standard Class call site.
		if nt, ok := namedtupleDecl(n.Child("value"), name); ok {
			return []Decl{nt}, nil
		}
		sc := newScope()
		val, err := lowerExpr(n.Child("value"), sc)
		if err != nil {
			return nil, err
		}
		ty := val.TypeOf()
		moduleScopeDeclare(name, ty)
		return []Decl{&Var{Name: name, Ty: ty, Value: val}}, nil
	case "AnnAssign":
		tgt := n.Child("target")
		if tgt.Type() != "Name" {
			return nil, fmt.Errorf("line %d: module-level annotated assignment must target a Name", n.Lineno())
		}
		name := tgt.Str("id")
		ann := n.Child("annotation")
		ty, err := lowerAnnotation(ann)
		if err != nil {
			return nil, err
		}
		var val Expr
		if v := n.Child("value"); v != nil {
			sc := newScope()
			val, err = lowerExpr(v, sc)
			if err != nil {
				return nil, err
			}
		}
		moduleScopeDeclare(name, ty)
		return []Decl{&Var{Name: name, Ty: ty, Value: val}}, nil
	case "If":
		// Guard `if __name__ == "__main__":` at module top — already filtered
		// at the module-walk site, but defensively skip here too.
		if isMainGuard(n) {
			return nil, nil
		}
		return nil, fmt.Errorf("line %d: unsupported top-level `if` (only `if __name__ == \"__main__\":` is recognized)", n.Lineno())
	default:
		return nil, fmt.Errorf("line %d: unsupported top-level node %q", n.Lineno(), n.Type())
	}
}

// decodeDataclassField recognizes `field(default=..., default_factory=...)`
// inside a @dataclass annotation and returns the equivalent default IR
// expression. Returns nil when the value isn't a field(...) call.
func decodeDataclassField(n parser.Node, ty *Type) Expr {
	if n == nil || n.Type() != "Call" {
		return nil
	}
	fn := n.Child("func")
	if fn == nil {
		return nil
	}
	// Recognized forms: bare `field` or `dataclasses.field`.
	switch {
	case fn.Type() == "Name" && fn.Str("id") == "field":
	case fn.Type() == "Attribute" && fn.Str("attr") == "field":
		recv := fn.Child("value")
		if recv == nil || recv.Type() != "Name" || recv.Str("id") != "dataclasses" {
			return nil
		}
	default:
		return nil
	}
	for _, kw := range n.Children("keywords") {
		switch kw.Str("arg") {
		case "default":
			v, err := lowerExpr(kw.Child("value"), newScope())
			if err == nil {
				return v
			}
		case "default_factory":
			factory := kw.Child("value")
			if factory == nil || factory.Type() != "Name" {
				return nil
			}
			switch factory.Str("id") {
			case "list":
				elem := &Type{Kind: TyAny}
				if ty != nil && ty.Kind == TyList && ty.Elem != nil {
					elem = ty.Elem
				}
				return &ListLit{Elems: nil, ElemTy: elem, Ty: &Type{Kind: TyList, Elem: elem}}
			case "dict":
				kt := &Type{Kind: TyAny}
				vt := &Type{Kind: TyAny}
				if ty != nil && ty.Kind == TyDict {
					if ty.Key != nil {
						kt = ty.Key
					}
					if ty.Val != nil {
						vt = ty.Val
					}
				}
				return &DictLit{Keys: nil, Vals: nil, KeyTy: kt, ValTy: vt, Ty: &Type{Kind: TyDict, Key: kt, Val: vt}}
			case "set":
				elem := &Type{Kind: TyAny}
				if ty != nil && ty.Kind == TyList && ty.Elem != nil {
					elem = ty.Elem
				}
				return &ListLit{Elems: nil, ElemTy: elem, Ty: &Type{Kind: TyList, Elem: elem}}
			}
		}
	}
	return nil
}

// namedtupleDecl recognizes the `Name = namedtuple("Name", ["f1", ...])`
// pattern and synthesizes a Class decl with one any-typed field per name.
// Returns (nil, false) when the RHS isn't a namedtuple call. The class
// name is taken from the assignment target so that `Foo = namedtuple("Bar", ...)`
// still produces a Go type called Foo.
func namedtupleDecl(val parser.Node, lhsName string) (*Class, bool) {
	if val == nil || val.Type() != "Call" {
		return nil, false
	}
	fn := val.Child("func")
	if fn == nil {
		return nil, false
	}
	isNT := false
	switch fn.Type() {
	case "Name":
		if fn.Str("id") == "namedtuple" {
			isNT = true
		}
	case "Attribute":
		recv := fn.Child("value")
		if recv != nil && recv.Type() == "Name" && recv.Str("id") == "collections" && fn.Str("attr") == "namedtuple" {
			isNT = true
		}
	}
	if !isNT {
		return nil, false
	}
	args := val.Children("args")
	if len(args) < 2 {
		return nil, false
	}
	// Second arg may be a list of name strings or a single space-separated str.
	var fieldNames []string
	switch args[1].Type() {
	case "List", "Tuple":
		for _, e := range args[1].Children("elts") {
			if e.Type() != "Constant" {
				return nil, false
			}
			s, _ := e["value"].(string)
			fieldNames = append(fieldNames, s)
		}
	case "Constant":
		s, _ := args[1]["value"].(string)
		for _, p := range splitFields(s) {
			fieldNames = append(fieldNames, p)
		}
	default:
		return nil, false
	}
	cls := &Class{Name: lhsName, HasInit: true}
	for _, f := range fieldNames {
		ty := &Type{Kind: TyAny}
		cls.Fields = append(cls.Fields, Param{Name: f, Ty: ty})
		cls.InitArgs = append(cls.InitArgs, Param{Name: f, Ty: ty})
		cls.InitBody = append(cls.InitBody, &AssignAttr{
			Target: &Name{N: "self", Ty: &Type{Kind: TyNamed, Name: lhsName}},
			Name:   f,
			Value:  &Name{N: f, Ty: ty},
		})
	}
	return cls, true
}

// splitFields breaks a namedtuple field-string on whitespace and commas.
func splitFields(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ' ' || r == '\t' || r == ',' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// moduleScopeVars tracks top-level Name → Type so function-body scopes
// can find globals during read-after-write inference. The map is reset
// at each call to Module so multi-file tests don't bleed types.
var moduleScopeVars = map[string]*Type{}

// starUnpackCounter generates fresh temp-variable names for star-LHS
// destructuring (`first, *rest = xs`) so the RHS evaluates exactly once
// per assignment. Reset alongside other module-scope state via
// moduleScopeReset.
var starUnpackCounter int

func moduleScopeDeclare(name string, ty *Type) {
	moduleScopeVars[name] = ty
}

func moduleScopeLookup(name string) (*Type, bool) {
	t, ok := moduleScopeVars[name]
	return t, ok
}

func moduleScopeReset() {
	moduleScopeVars = map[string]*Type{}
	moduleTypeVars = map[string]bool{}
	starUnpackCounter = 0
}

// moduleTypeVars tracks TypeVar declarations at module level so subsequent
// function signatures can pick them up as Go generic type parameters.
var moduleTypeVars = map[string]bool{}

func isModuleTypeVar(name string) bool { return moduleTypeVars[name] }

// currentLoweringClass tracks the class whose body is currently being
// lowered. `typing.Self` annotations resolve to this class — outside a
// class, the type still collapses to `any`.
var currentLoweringClass string

// lowerClass emits a Class decl plus one Func per method (with Receiver set).
// __init__ becomes the constructor body; its self-attribute assignments are
// scanned to derive struct fields.
func lowerClass(n parser.Node) ([]Decl, error) {
	name := n.Str("name")
	// Register the class name in module scope so subsequent `Class(...)`
	// constructor calls infer TyNamed as their static return type.
	moduleScopeDeclare(name, &Type{Kind: TyNamed, Name: name})
	prevClass := currentLoweringClass
	currentLoweringClass = name
	defer func() { currentLoweringClass = prevClass }()
	class := &Class{Name: name}
	// First pass: collect all base names (including any registered plugin
	// markers like `Model`). Plugin bases are consumed by the lower hook
	// — they don't appear as Go-level embeds.
	var rawBases []string
	for _, b := range n.Children("bases") {
		// Accept Attribute bases like `abc.ABC` or subscripted bases like
		// `Generic[T]` / `Protocol[T]` as no-op markers — they carry no
		// codegen consequence and would otherwise block parsing of code
		// that mirrors common Python typing patterns.
		if b.Type() == "Attribute" {
			attr := b.Str("attr")
			if attr == "ABC" || attr == "ABCMeta" || attr == "Protocol" || attr == "Generic" {
				continue
			}
			return nil, fmt.Errorf("class %s: complex base expressions not supported", name)
		}
		if b.Type() == "Subscript" {
			val := b.Child("value")
			if val != nil && val.Type() == "Name" {
				switch val.Str("id") {
				case "Generic", "Protocol":
					continue
				}
			}
			return nil, fmt.Errorf("class %s: complex base expressions not supported", name)
		}
		if b.Type() != "Name" {
			return nil, fmt.Errorf("class %s: complex base expressions not supported", name)
		}
		bn := b.Str("id")
		switch bn {
		case "object", "Protocol", "Generic":
			continue
		case "ABC", "ABCMeta":
			// ABC marker: collect for later interface promotion. Stripped
			// from the embed list either way.
			class.IsInterface = true
			continue
		}
		rawBases = append(rawBases, bn)
	}
	// Enum detection: collapse `class Color(Enum):` into a typed
	// integer-aliased constant set. The base name is consumed, not
	// embedded.
	for i, b := range rawBases {
		if b == "Enum" || b == "IntEnum" || b == "Flag" || b == "IntFlag" || b == "StrEnum" {
			class.IsEnum = true
			rawBases = append(rawBases[:i], rawBases[i+1:]...)
			break
		}
	}
	class.Bases = append(class.Bases, rawBases...)
	if class.IsEnum {
		nextAuto := int64(1)
		for _, m := range n.Children("body") {
			if m.Type() != "Assign" {
				return nil, fmt.Errorf("enum %s: body must contain only `NAME = value` declarations", name)
			}
			tgts := m.Children("targets")
			if len(tgts) != 1 || tgts[0].Type() != "Name" {
				return nil, fmt.Errorf("enum %s: each member needs a single Name target", name)
			}
			val := m.Child("value")
			memberName := tgts[0].Str("id")
			// `auto()` — assign next sequential int starting at 1 (Python
			// default), mirroring enum.auto() semantics.
			if val.Type() == "Call" {
				fn := val.Child("func")
				if fn != nil && ((fn.Type() == "Name" && fn.Str("id") == "auto") ||
					(fn.Type() == "Attribute" && fn.Str("attr") == "auto")) {
					class.EnumMembers = append(class.EnumMembers, EnumMember{Name: memberName, Value: nextAuto})
					nextAuto++
					continue
				}
			}
			if val.Type() != "Constant" {
				return nil, fmt.Errorf("enum %s.%s: value must be an int literal or auto()", name, memberName)
			}
			fv, ok := val["value"].(float64)
			if !ok {
				return nil, fmt.Errorf("enum %s.%s: value must be an int literal", name, memberName)
			}
			iv := int64(fv)
			class.EnumMembers = append(class.EnumMembers, EnumMember{Name: memberName, Value: iv})
			nextAuto = iv + 1
		}
		return []Decl{class}, nil
	}
	// @dataclass / @dataclasses.dataclass synthesizes __init__ from
	// class-level annotated fields. Detect the decorator(s) here so the
	// body walk can pick up the AnnAssign nodes that would otherwise
	// trip the "only methods supported" rule.
	isDataclass := false
	for _, d := range n.Children("decorator_list") {
		if d.Type() == "Name" {
			switch d.Str("id") {
			case "dataclass":
				isDataclass = true
			case "final", "runtime_checkable", "unique", "total_ordering":
				continue
			}
		}
		if d.Type() == "Attribute" {
			recv := d.Child("value")
			attr := d.Str("attr")
			if recv != nil && recv.Type() == "Name" {
				switch recv.Str("id") {
				case "dataclasses":
					if attr == "dataclass" {
						isDataclass = true
					}
				case "typing":
					if attr == "final" || attr == "runtime_checkable" {
						continue
					}
				case "enum":
					if attr == "unique" {
						continue
					}
				case "functools":
					if attr == "total_ordering" {
						continue
					}
				}
			}
		}
		if d.Type() == "Call" {
			fn := d.Child("func")
			if fn != nil && fn.Type() == "Name" && fn.Str("id") == "dataclass" {
				isDataclass = true
			}
		}
	}
	bodyNodes := n.Children("body")
	var decls []Decl
	// Synthesized __init__ state used by the dataclass path. We append
	// to class.InitArgs / InitBody / Fields directly so the existing
	// class-codegen pipeline emits the constructor.
	dataclassDone := false
	if isDataclass {
		class.IsDataclass = true
		class.HasInit = true
		dcSc := newScope()
		dcSc.declare("self", &Type{Kind: TyNamed, Name: name})
		for _, m := range bodyNodes {
			if m.Type() == "AnnAssign" {
				tgt := m.Child("target")
				if tgt.Type() != "Name" {
					return nil, fmt.Errorf("@dataclass %s: field decl needs a single Name target", name)
				}
				ty, err := lowerAnnotation(m.Child("annotation"))
				if err != nil {
					return nil, fmt.Errorf("@dataclass %s.%s: %w", name, tgt.Str("id"), err)
				}
				fieldName := tgt.Str("id")
				class.Fields = append(class.Fields, Param{Name: fieldName, Ty: ty})
				p := Param{Name: fieldName, Ty: ty}
				if def := m.Child("value"); def != nil {
					// `field(default_factory=list)` / `field(default=...)`:
					// substitute an empty container literal (or the explicit
					// default value) before lowering so each call site builds
					// a fresh instance.
					if dvExpr := decodeDataclassField(def, ty); dvExpr != nil {
						p.Default = dvExpr
					} else {
						dv, err := lowerExpr(def, dcSc)
						if err != nil {
							return nil, err
						}
						p.Default = dv
					}
				}
				class.InitArgs = append(class.InitArgs, p)
				dcSc.declare(fieldName, ty)
				class.InitBody = append(class.InitBody, &AssignAttr{
					Target: &Name{N: "self", Ty: &Type{Kind: TyNamed, Name: name}},
					Name:   fieldName,
					Value:  &Name{N: fieldName, Ty: ty},
				})
			}
		}
		dataclassDone = true
	}

	type classFieldDefault struct {
		Name  string
		Value Expr
	}
	var classFieldDefaults []classFieldDefault
	var userInitBody []Stmt // user-written __init__ before defaults prepend

	for _, m := range bodyNodes {
		if dataclassDone && m.Type() == "AnnAssign" {
			continue // already consumed above
		}
		// `__slots__ = (...)` / `__slots__ = [...]` / `_ = ...` class-body
		// statements are CPython hints with no Go equivalent — drop them
		// rather than rejecting the class as having a non-method body.
		if m.Type() == "Assign" {
			targets := m.Children("targets")
			if len(targets) == 1 && targets[0].Type() == "Name" {
				switch targets[0].Str("id") {
				case "__slots__", "__match_args__", "__all__", "_":
					continue
				}
			}
		}
		// Class-level field annotations on a regular class: `class Foo:\n
		// items: dict[str, int]\n def __init__(self): ...`. Register the
		// field on the class struct so the type is known. If a default
		// value is given (`x: int = 5`), stash it so it gets applied at
		// the head of __init__'s body, mirroring Python's class-level
		// initializer semantics. ClassVar declarations are dropped from
		// struct fields by convention.
		if !isDataclass && m.Type() == "AnnAssign" {
			tgt := m.Child("target")
			if tgt.Type() != "Name" {
				return nil, fmt.Errorf("line %d: class %s: field declaration target must be a Name", m.Lineno(), name)
			}
			annNode := m.Child("annotation")
			if annNode != nil && annNode.Type() == "Subscript" {
				if base := annNode.Child("value"); base != nil && base.Type() == "Name" && base.Str("id") == "ClassVar" {
					// ClassVar[T] — accepted as a class-level constant
					// annotation but not added as a struct field. Drop.
					continue
				}
			}
			ty, err := lowerAnnotation(annNode)
			if err != nil {
				return nil, fmt.Errorf("class %s.%s: %w", name, tgt.Str("id"), err)
			}
			fieldName := tgt.Str("id")
			// Avoid duplicates: a class may also assign self.<name> in
			// __init__, which a later pass will pick up. We register the
			// declared type so attribute access sees it before __init__
			// runs.
			already := false
			for _, f := range class.Fields {
				if f.Name == fieldName {
					already = true
					break
				}
			}
			if !already {
				class.Fields = append(class.Fields, Param{Name: fieldName, Ty: ty})
			}
			if def := m.Child("value"); def != nil {
				dsc := newScope()
				dsc.declare("self", &Type{Kind: TyNamed, Name: name})
				dv, err := lowerExpr(def, dsc)
				if err != nil {
					return nil, fmt.Errorf("class %s.%s default: %w", name, fieldName, err)
				}
				classFieldDefaults = append(classFieldDefaults, classFieldDefault{Name: fieldName, Value: dv})
			}
			continue
		}
		// `pass`, bare docstring, ellipsis literals and other no-op
		// statements are accepted but not lowered — they let
		// `class MyError(Exception): pass` and abstract stubs compile
		// without F2 erroring.
		if m.Type() == "Pass" {
			continue
		}
		if m.Type() == "Expr" {
			val := m.Child("value")
			if val != nil {
				switch val.Type() {
				case "Constant", "Str", "Ellipsis":
					continue
				}
			}
		}
		if m.Type() != "FunctionDef" && m.Type() != "AsyncFunctionDef" {
			return nil, fmt.Errorf("line %d: class %s: only methods supported (F2)", m.Lineno(), name)
		}
		// Accepted method decorators:
		//   @property     — call sites emit `.attr()`
		//   @classmethod  — lowered to a free `<Class>_<method>` Go function
		//                   so it doesn't need a `*Class` receiver
		isProperty := false
		isClassMethod := false
		isStaticMethod := false
		isAbstract := false
		propSetterFor := ""
		for _, d := range m.Children("decorator_list") {
			if d.Type() == "Name" {
				switch d.Str("id") {
				case "property":
					isProperty = true
					continue
				case "classmethod":
					isClassMethod = true
					continue
				case "abstractmethod":
					isAbstract = true
					continue
				case "staticmethod":
					isStaticMethod = true
					continue
				case "final", "override", "overload",
					"deprecated", "cached_property", "no_type_check":
					continue
				}
			}
			// Attribute-style decorator, e.g. @abc.abstractmethod.
			if d.Type() == "Attribute" {
				attr := d.Str("attr")
				if attr == "abstractmethod" || attr == "abstractclassmethod" || attr == "abstractstaticmethod" || attr == "abstractproperty" {
					isAbstract = true
					continue
				}
				// @<name>.setter — mark this method as the property
				// setter for <name>. The method body becomes Set<Name>.
				if attr == "setter" {
					recv := d.Child("value")
					if recv != nil && recv.Type() == "Name" {
						propSetterFor = recv.Str("id")
						continue
					}
				}
				// @<name>.deleter — accepted but ignored (no Go equivalent).
				if attr == "deleter" {
					recv := d.Child("value")
					if recv != nil && recv.Type() == "Name" {
						continue
					}
				}
				recv := d.Child("value")
				if recv != nil && recv.Type() == "Name" {
					switch recv.Str("id") {
					case "typing":
						switch attr {
						case "final", "overload", "no_type_check":
							continue
						}
					case "functools":
						if attr == "cached_property" {
							continue
						}
					case "warnings":
						if attr == "deprecated" {
							continue
						}
					}
				}
			}
			// Call-form decorators like @overload() / @deprecated("...").
			if d.Type() == "Call" {
				fn := d.Child("func")
				if fn != nil {
					if fn.Type() == "Name" {
						switch fn.Str("id") {
						case "overload", "deprecated":
							continue
						}
					}
					if fn.Type() == "Attribute" {
						attr := fn.Str("attr")
						recv := fn.Child("value")
						if recv != nil && recv.Type() == "Name" {
							switch recv.Str("id") {
							case "typing":
								if attr == "overload" || attr == "deprecated" {
									continue
								}
							case "warnings":
								if attr == "deprecated" {
									continue
								}
							}
						}
					}
				}
			}
			// Unrecognized decorator on a method: accept as a passthrough
			// so files leaning on annotation-only decorators compile.
			// Same caveat as free functions — real behavior (logging,
			// caching) won't run. We accept @name, @mod.attr, and the
			// call-form @name(args) / @mod.attr(args).
			if d.Type() == "Name" || d.Type() == "Attribute" {
				continue
			}
			if d.Type() == "Call" {
				fn := d.Child("func")
				if fn != nil && (fn.Type() == "Name" || fn.Type() == "Attribute") {
					continue
				}
			}
			var dname string
			if d.Type() == "Name" {
				dname = d.Str("id")
			} else {
				dname = d.Type()
			}
			return nil, fmt.Errorf("line %d: class %s: method decorator %q not supported", m.Lineno(), name, dname)
		}
		if isProperty && isClassMethod {
			return nil, fmt.Errorf("line %d: class %s: cannot combine @property and @classmethod", m.Lineno(), name)
		}
		methName := m.Str("name")
		// `@<name>.setter` def has the same name as the property. Rename
		// the Go method so it doesn't collide with the getter, and
		// register it on the class's PropertySetters map.
		if propSetterFor != "" {
			titled := propSetterFor
			if len(titled) > 0 {
				titled = string(titled[0]-32) + titled[1:]
			}
			rename := "Set" + titled
			if class.PropertySetters == nil {
				class.PropertySetters = map[string]string{}
			}
			class.PropertySetters[propSetterFor] = rename
			methName = rename
		}
		args := m.Child("args").Children("args")
		if !isStaticMethod && len(args) == 0 {
			return nil, fmt.Errorf("class %s.%s: method must have at least self parameter", name, methName)
		}
		// Drop self; we add it back as Receiver. @staticmethod methods
		// have no implicit self, so keep all args.
		var params []parser.Node
		if isStaticMethod {
			params = args
		} else {
			params = args[1:]
		}

		if methName == "__init__" {
			class.HasInit = true
			for _, p := range params {
				ty, err := lowerAnnotation(p.Child("annotation"))
				if err != nil {
					return nil, fmt.Errorf("class %s __init__ param %s: %w", name, p.Str("arg"), err)
				}
				class.InitArgs = append(class.InitArgs, Param{Name: p.Str("arg"), Ty: ty})
			}
			// Lower body in a scope that knows about init params + self.
			sc := newScope()
			sc.declare("self", &Type{Kind: TyNamed, Name: name})
			for _, p := range class.InitArgs {
				sc.declare(p.Name, p.Ty)
			}
			body, err := lowerBody(m.Children("body"), sc)
			if err != nil {
				return nil, err
			}
			// Preserve a slice of the user-written __init__ body before
			// we prepend the class-level field defaults; the ClassVars
			// detection below uses it to tell which fields are written
			// only by the auto-prelude (eligible to hoist).
			userInitBody = body
			// Prepend class-level field defaults so they run before the
			// user-written __init__ body. If __init__ already sets the
			// same field, the user's assignment wins (executed second).
			if len(classFieldDefaults) > 0 {
				prefix := make([]Stmt, 0, len(classFieldDefaults))
				for _, d := range classFieldDefaults {
					prefix = append(prefix, &AssignAttr{
						Target: &Name{N: "self", Ty: &Type{Kind: TyNamed, Name: name}},
						Name:   d.Name,
						Value:  d.Value,
					})
				}
				body = append(prefix, body...)
			}
			class.InitBody = body
			// Scan body for `self.x = expr` to derive fields (preserve order).
			// Skip fields already registered via class-level annotation so
			// they keep their declared type rather than picking up the
			// (often-untyped) initializer type.
			seen := map[string]bool{}
			for _, f := range class.Fields {
				seen[f.Name] = true
			}
			for _, st := range body {
				aa, ok := st.(*AssignAttr)
				if !ok {
					continue
				}
				recv, ok := aa.Target.(*Name)
				if !ok || recv.N != "self" {
					continue
				}
				if seen[aa.Name] {
					continue
				}
				seen[aa.Name] = true
				ty := aa.AnnTy
				if ty == nil {
					ty = aa.Value.TypeOf()
				}
				class.Fields = append(class.Fields, Param{Name: aa.Name, Ty: ty})
			}
			continue
		}

		// Regular method (or @classmethod / @staticmethod). For
		// @classmethod / @staticmethod we emit a free function
		// `<Class>_<method>` so it doesn't take a *Class receiver; calls
		// like `Class.method(args)` are rewritten at codegen time.
		// References to `cls` inside the body are rewritten to the class
		// name, matching Python's semantics for `cls(...)`.
		fn := &Func{Name: methName, Line: m.Lineno()}
		if !isClassMethod && !isStaticMethod {
			fn.Receiver = &Param{Name: "self", Ty: &Type{Kind: TyNamed, Name: name}}
		} else {
			fn.Name = name + "_" + methName
		}
		for _, p := range params {
			ty, err := lowerAnnotation(p.Child("annotation"))
			if err != nil {
				return nil, fmt.Errorf("class %s.%s param %s: %w", name, methName, p.Str("arg"), err)
			}
			fn.Params = append(fn.Params, Param{Name: p.Str("arg"), Ty: ty})
		}
		// Method defaults live under m.args.defaults, aligned to the
		// trailing params (Python excludes `self`/`cls` from defaults).
		mArgs := m.Child("args")
		if mArgs != nil {
			mdefs := mArgs.Children("defaults")
			if n := len(mdefs); n > 0 {
				off := len(fn.Params) - n
				dsc := newScope()
				for _, p := range fn.Params[:off] {
					dsc.declare(p.Name, p.Ty)
				}
				for i, dn := range mdefs {
					d, err := lowerExpr(dn, dsc)
					if err != nil {
						return nil, fmt.Errorf("default for class %s.%s param %q: %w", name, methName, fn.Params[off+i].Name, err)
					}
					fn.Params[off+i].Default = d
				}
			}
			// Method *args / **kwargs.
			if va := mArgs.Child("vararg"); va != nil {
				elemTy := &Type{Kind: TyAny}
				if ann := va.Child("annotation"); ann != nil {
					if t, err := lowerAnnotation(ann); err == nil {
						elemTy = t
					}
				}
				fn.Vararg = &Param{Name: va.Str("arg"), Ty: &Type{Kind: TyList, Elem: elemTy}}
			}
			if kw := mArgs.Child("kwarg"); kw != nil {
				fn.Kwarg = &Param{Name: kw.Str("arg"), Ty: &Type{Kind: TyDict, Key: &Type{Kind: TyStr}, Val: &Type{Kind: TyAny}}}
			}
		}
		ret, err := lowerAnnotation(m.Child("returns"))
		if err != nil {
			return nil, err
		}
		fn.Ret = ret

		sc := newScope()
		if isClassMethod {
			// `cls` is a stand-in for the class itself; treat it as a
			// declared name with the class's named type so lookups don't
			// trip the undeclared-identifier path.
			sc.declare("cls", &Type{Kind: TyNamed, Name: name})
		} else if !isStaticMethod {
			sc.declare("self", fn.Receiver.Ty)
		}
		for _, p := range fn.Params {
			sc.declare(p.Name, p.Ty)
		}
		if fn.Vararg != nil {
			sc.declare(fn.Vararg.Name, fn.Vararg.Ty)
		}
		if fn.Kwarg != nil {
			sc.declare(fn.Kwarg.Name, fn.Kwarg.Ty)
		}
		body, err := lowerBody(m.Children("body"), sc)
		if err != nil {
			return nil, err
		}
		if isClassMethod {
			// Rewrite every `cls` Name in the body to the class's own name.
			// `cls(...)` then routes through the constructor; `cls.attr`
			// becomes `<Class>.attr` (undefined unless attr is a registered
			// class method).
			substituteName(body, "cls", name)
		}
		fn.Body = body
		decls = append(decls, fn)
		if isClassMethod || isStaticMethod {
			// Both flavors emit as free functions; reuse the ClassMethods
			// set as the call-site dispatch hook. @staticmethod skips the
			// self/cls bind; the call site form `Class.name(...)` rewrites
			// to `Class_name(...)` identically.
			if class.ClassMethods == nil {
				class.ClassMethods = map[string]bool{}
			}
			class.ClassMethods[methName] = true
			continue
		}
		class.MethodNames = append(class.MethodNames, methName)
		if isProperty {
			if class.Properties == nil {
				class.Properties = map[string]bool{}
			}
			class.Properties[methName] = true
		}
		if isAbstract {
			class.InterfaceMethods = append(class.InterfaceMethods, InterfaceMethod{
				Name:   methName,
				Params: append([]Param(nil), fn.Params...),
				Ret:    fn.Ret,
			})
		}
	}

	// Detect class-level fields that are *only* assigned at class level
	// (default initializer) and never via `self.<field>` in any method.
	// These are class-shared state in Python — promote them to Go
	// module-level vars so `Class.<field>` reads/writes hit a single
	// storage cell rather than per-instance struct fields.
	{
		writtenViaSelf := map[string]bool{}
		walkAssignAttrTargets(decls, "self", writtenViaSelf)
		// Only the user-written portion of __init__ counts — the auto
		// prepended `self.<f> = <default>` block doesn't disqualify a
		// field from becoming a class var.
		walkAssignAttrTargetsStmts(userInitBody, "self", writtenViaSelf)
		defaultByName := map[string]Expr{}
		for _, d := range classFieldDefaults {
			defaultByName[d.Name] = d.Value
		}
		var keptFields []Param
		for _, f := range class.Fields {
			defVal, hasDef := defaultByName[f.Name]
			if hasDef && !writtenViaSelf[f.Name] {
				if class.ClassVars == nil {
					class.ClassVars = map[string]ClassVar{}
				}
				class.ClassVars[f.Name] = ClassVar{Ty: f.Ty, Default: defVal}
				continue
			}
			keptFields = append(keptFields, f)
		}
		class.Fields = keptFields
		// Drop the corresponding default-init entries so the constructor
		// doesn't try to write `self.<field> = <default>` for a hoisted
		// class var (the field no longer exists on the struct).
		if len(class.ClassVars) > 0 {
			var keptDefaults []classFieldDefault
			for _, d := range classFieldDefaults {
				if _, ok := class.ClassVars[d.Name]; ok {
					continue
				}
				keptDefaults = append(keptDefaults, d)
			}
			classFieldDefaults = keptDefaults
			// Also peel any leading default-init statements from InitBody.
			var keptInit []Stmt
			for _, st := range class.InitBody {
				if aa, ok := st.(*AssignAttr); ok {
					if recv, ok := aa.Target.(*Name); ok && recv.N == "self" {
						if _, isCV := class.ClassVars[aa.Name]; isCV {
							continue
						}
					}
				}
				keptInit = append(keptInit, st)
			}
			class.InitBody = keptInit
		}
	}

	// Synthesize a constructor body for class-level field defaults when
	// the class has no explicit __init__ — otherwise the defaults would
	// vanish at codegen time.
	if !class.HasInit && len(classFieldDefaults) > 0 {
		class.HasInit = true
		for _, d := range classFieldDefaults {
			class.InitBody = append(class.InitBody, &AssignAttr{
				Target: &Name{N: "self", Ty: &Type{Kind: TyNamed, Name: name}},
				Name:   d.Name,
				Value:  d.Value,
			})
		}
	}

	// Class first, then methods, so codegen order is type → methods.
	return append([]Decl{class}, decls...), nil
}

// walkAssignAttrTargets visits every statement / nested block looking for
// `<receiver>.<name> = ...` writes, recording the attribute name into
// `out`. Used by lowerClass to decide which fields are pure class-level
// (never touched per-instance) and thus eligible for module-var hoisting.
func walkAssignAttrTargets(decls []Decl, receiver string, out map[string]bool) {
	var walkStmts func([]Stmt)
	walkStmts = func(ss []Stmt) {
		for _, s := range ss {
			switch x := s.(type) {
			case *AssignAttr:
				if recv, ok := x.Target.(*Name); ok && recv.N == receiver {
					out[x.Name] = true
				}
			case *If:
				walkStmts(x.Then)
				walkStmts(x.Else)
			case *While:
				walkStmts(x.Body)
			case *ForEach:
				walkStmts(x.Body)
			case *ForRange:
				walkStmts(x.Body)
			case *Try:
				walkStmts(x.Body)
				walkStmts(x.Finally)
				for _, h := range x.Handlers {
					walkStmts(h.Body)
				}
			}
		}
	}
	for _, d := range decls {
		fn, ok := d.(*Func)
		if !ok {
			continue
		}
		walkStmts(fn.Body)
	}
}

// walkAssignAttrTargetsStmts is an overload for raw stmt slices (init body).
func walkAssignAttrTargetsStmts(ss []Stmt, receiver string, out map[string]bool) {
	for _, s := range ss {
		if aa, ok := s.(*AssignAttr); ok {
			if recv, ok := aa.Target.(*Name); ok && recv.N == receiver {
				out[aa.Name] = true
			}
		}
	}
}

func lowerFunc(n parser.Node) (*Func, error) {
	// `@overload` / `@typing.overload` — the decorated function is a
	// type-check stub. Drop it entirely; the real implementation that
	// follows under the same name does the actual work.
	for _, d := range n.Children("decorator_list") {
		if d.Type() == "Name" && d.Str("id") == "overload" {
			return nil, nil
		}
		if d.Type() == "Attribute" && d.Str("attr") == "overload" {
			recv := d.Child("value")
			if recv != nil && recv.Type() == "Name" && recv.Str("id") == "typing" {
				return nil, nil
			}
		}
	}
	// Accepted decorators (silently ignored — they don't change Go
	// codegen): `@staticmethod` matches a free function with no self;
	// `@functools.lru_cache` / `@lru_cache` / `@lru_cache(...)` is a
	// performance hint we currently ignore (the function still runs,
	// uncached). User-defined name-form decorators (`@my_decorator`)
	// are accepted as passthroughs; the decorator function is not
	// invoked at runtime, so any behavior beyond identity (caching,
	// logging) won't be reproduced.
	var userDecorators []string
	for _, d := range n.Children("decorator_list") {
		if d.Type() == "Name" {
			switch d.Str("id") {
			case "staticmethod", "lru_cache", "cache", "cached_property",
				"wraps", "singledispatch", "final", "override", "overload",
				"deprecated", "no_type_check", "runtime_checkable":
				continue
			}
		}
		if d.Type() == "Attribute" {
			recv := d.Child("value")
			attr := d.Str("attr")
			if recv != nil && recv.Type() == "Name" {
				switch recv.Str("id") {
				case "functools":
					switch attr {
					case "lru_cache", "cache", "cached_property", "wraps", "singledispatch":
						continue
					}
				case "typing":
					switch attr {
					case "final", "overload", "no_type_check", "runtime_checkable":
						continue
					}
				case "warnings":
					if attr == "deprecated" {
						continue
					}
				}
			}
		}
		if d.Type() == "Call" {
			fn := d.Child("func")
			if fn != nil {
				if fn.Type() == "Name" {
					switch fn.Str("id") {
					case "lru_cache", "cache", "wraps", "deprecated", "overload":
						continue
					}
				}
				if fn.Type() == "Attribute" {
					recv := fn.Child("value")
					attr := fn.Str("attr")
					if recv != nil && recv.Type() == "Name" {
						switch recv.Str("id") {
						case "functools":
							switch attr {
							case "lru_cache", "cache", "wraps":
								continue
							}
						case "typing":
							if attr == "overload" || attr == "deprecated" {
								continue
							}
						case "warnings":
							if attr == "deprecated" {
								continue
							}
						}
					}
				}
			}
		}
		// Unrecognized decorator. Accept as a passthrough when it's a
		// shape gopy can ignore safely:
		//   @name                 → bare user decorator
		//   @name(args...)        → parametrized user decorator
		//   @mod.attr             → dotted name (treated as no-op)
		//   @mod.attr(args...)    → dotted call form
		// The decorator body never runs, so anything beyond identity
		// (caching, retry, logging) won't be reproduced. This trades
		// strict rejection for the ability to transpile real-world
		// Python files that lean on annotation-only decorators.
		if d.Type() == "Name" {
			userDecorators = append(userDecorators, d.Str("id"))
			continue
		}
		if d.Type() == "Attribute" {
			userDecorators = append(userDecorators, d.Str("attr"))
			continue
		}
		if d.Type() == "Call" {
			fn := d.Child("func")
			if fn != nil && (fn.Type() == "Name" || fn.Type() == "Attribute") {
				if fn.Type() == "Name" {
					userDecorators = append(userDecorators, fn.Str("id"))
				} else {
					userDecorators = append(userDecorators, fn.Str("attr"))
				}
				continue
			}
		}
		var name string
		if d.Type() == "Name" {
			name = d.Str("id")
		} else {
			name = d.Type()
		}
		return nil, fmt.Errorf("line %d: decorator %q not supported", n.Lineno(), name)
	}
	f := &Func{Name: n.Str("name"), UserDecorators: userDecorators, Line: n.Lineno()}
	for _, tp := range n.Children("type_params") {
		if tp.Type() == "TypeVar" {
			f.TypeParams = append(f.TypeParams, tp.Str("name"))
		}
	}
	args := n.Child("args")
	if args == nil {
		return nil, fmt.Errorf("FunctionDef %q missing args", f.Name)
	}
	paramNodes := args.Children("args")
	for _, a := range paramNodes {
		ty, err := lowerAnnotation(a.Child("annotation"))
		if err != nil {
			return nil, fmt.Errorf("param %q: %w", a.Str("arg"), err)
		}
		f.Params = append(f.Params, Param{Name: a.Str("arg"), Ty: ty})
	}
	// *args (single python `arg` node under args.vararg) becomes a []T
	// slice — T from the annotation when present, []any otherwise.
	if va := args.Child("vararg"); va != nil {
		elemTy := &Type{Kind: TyAny}
		if ann := va.Child("annotation"); ann != nil {
			if t, err := lowerAnnotation(ann); err == nil {
				elemTy = t
			}
		}
		f.Vararg = &Param{Name: va.Str("arg"), Ty: &Type{Kind: TyList, Elem: elemTy}}
	}
	// **kwargs (single python `arg` node under args.kwarg) becomes map[string]any.
	if kw := args.Child("kwarg"); kw != nil {
		f.Kwarg = &Param{Name: kw.Str("arg"), Ty: &Type{Kind: TyDict, Key: &Type{Kind: TyStr}, Val: &Type{Kind: TyAny}}}
	}
	// Keyword-only parameters: `def f(a, *, b=1, c=2)`. Treat them as
	// regular Params with defaults. The call site already routes kwargs
	// by name. Defaults come from args.kw_defaults aligned 1:1 with
	// kwonlyargs (None entries mean no default — still required).
	kwonlyNodes := args.Children("kwonlyargs")
	kwDefaults := args.Children("kw_defaults")
	if len(kwonlyNodes) > 0 {
		dsc := newScope()
		for _, p := range f.Params {
			dsc.declare(p.Name, p.Ty)
		}
		for i, a := range kwonlyNodes {
			ty, err := lowerAnnotation(a.Child("annotation"))
			if err != nil {
				return nil, fmt.Errorf("kwonly param %q: %w", a.Str("arg"), err)
			}
			p := Param{Name: a.Str("arg"), Ty: ty}
			if i < len(kwDefaults) && kwDefaults[i] != nil {
				// `None` placeholder in kw_defaults means no default.
				dn := kwDefaults[i]
				if !(dn.Type() == "Constant" && dn["value"] == nil) {
					d, err := lowerExpr(dn, dsc)
					if err != nil {
						return nil, fmt.Errorf("default for kwonly param %q: %w", p.Name, err)
					}
					p.Default = d
				}
			}
			f.Params = append(f.Params, p)
		}
	}
	// Python aligns `args.defaults` to the END of the positional params:
	// for `def f(a, b, c=1, d=2)`, defaults == [1, 2] aligns to c, d.
	defaults := args.Children("defaults")
	if n := len(defaults); n > 0 {
		off := len(f.Params) - n
		// Defaults are evaluated in a scope that already has the earlier
		// params declared (Python evaluates them at def-time in the
		// enclosing scope; we approximate by using an empty scope here
		// since defaults are typically literals or stable references).
		dsc := newScope()
		for _, p := range f.Params[:off] {
			dsc.declare(p.Name, p.Ty)
		}
		for i, dn := range defaults {
			d, err := lowerExpr(dn, dsc)
			if err != nil {
				return nil, fmt.Errorf("default for param %q: %w", f.Params[off+i].Name, err)
			}
			f.Params[off+i].Default = d
		}
	}
	ret, err := lowerAnnotation(n.Child("returns"))
	if err != nil {
		return nil, fmt.Errorf("return annotation: %w", err)
	}
	f.Ret = ret
	scope := newScope()
	for _, p := range f.Params {
		scope.declare(p.Name, p.Ty)
	}
	if f.Vararg != nil {
		scope.declare(f.Vararg.Name, f.Vararg.Ty)
	}
	if f.Kwarg != nil {
		scope.declare(f.Kwarg.Name, f.Kwarg.Ty)
	}
	body, err := lowerBody(n.Children("body"), scope)
	if err != nil {
		return nil, err
	}
	f.Body = body
	// Promote function to generator if any Yield appears in the body.
	if ty := findYieldType(body); ty != nil {
		f.IsGenerator = true
		f.YieldType = ty
	}
	// Module-level TypeVars referenced in this function's annotations
	// promote to Go generic type parameters. Walk param / return / vararg
	// types and collect any TyNamed name matching a registered TypeVar.
	collectTypeVarParams(f)
	return f, nil
}

func collectTypeVarParams(f *Func) {
	seen := map[string]bool{}
	for _, n := range f.TypeParams {
		seen[n] = true
	}
	add := func(t *Type) {
		walkTypeVars(t, func(name string) {
			if isModuleTypeVar(name) && !seen[name] {
				seen[name] = true
				f.TypeParams = append(f.TypeParams, name)
			}
		})
	}
	for _, p := range f.Params {
		add(p.Ty)
	}
	if f.Vararg != nil {
		add(f.Vararg.Ty)
	}
	add(f.Ret)
}

func walkTypeVars(t *Type, visit func(string)) {
	if t == nil {
		return
	}
	if t.Kind == TyNamed {
		visit(t.Name)
	}
	walkTypeVars(t.Elem, visit)
	walkTypeVars(t.Key, visit)
	walkTypeVars(t.Val, visit)
	for _, x := range t.Tuple {
		walkTypeVars(x, visit)
	}
	for _, p := range t.FuncParams {
		walkTypeVars(p, visit)
	}
	walkTypeVars(t.FuncRet, visit)
}

// findYieldType walks a function body recursively, returning the inferred
// element type of the first Yield encountered. Returns nil if the function
// is not a generator.
func findYieldType(ss []Stmt) *Type {
	for _, s := range ss {
		if t := yieldTypeStmt(s); t != nil {
			return t
		}
	}
	return nil
}

func yieldTypeStmt(s Stmt) *Type {
	switch x := s.(type) {
	case *Yield:
		if x.X != nil {
			return x.X.TypeOf()
		}
		return &Type{Kind: TyUnknown}
	case *YieldFrom:
		// The forwarded iterable's element type drives the outer
		// generator's yield type.
		if t := x.Iter.TypeOf(); t != nil {
			if t.Kind == TyList {
				return t.Elem
			}
			return t
		}
		return &Type{Kind: TyUnknown}
	case *If:
		if t := findYieldType(x.Then); t != nil {
			return t
		}
		return findYieldType(x.Else)
	case *While:
		return findYieldType(x.Body)
	case *ForRange:
		return findYieldType(x.Body)
	case *ForEach:
		return findYieldType(x.Body)
	case *Try:
		if t := findYieldType(x.Body); t != nil {
			return t
		}
		for _, h := range x.Handlers {
			if t := findYieldType(h.Body); t != nil {
				return t
			}
		}
		return findYieldType(x.Finally)
	case *WithFile:
		return findYieldType(x.Body)
	}
	return nil
}

func lowerAnnotation(n parser.Node) (*Type, error) {
	if n == nil {
		return &Type{Kind: TyUnknown}, nil
	}
	switch n.Type() {
	case "Name":
		switch n.Str("id") {
		case "int":
			return &Type{Kind: TyInt}, nil
		case "float":
			return &Type{Kind: TyFloat}, nil
		case "complex":
			return &Type{Kind: TyComplex}, nil
		case "str":
			return &Type{Kind: TyStr}, nil
		case "bool":
			return &Type{Kind: TyBool}, nil
		case "None":
			return &Type{Kind: TyNone}, nil
		case "list":
			// Bare `list` annotation (no element type) — accept and lower
			// to `list[any]` so it round-trips through Go without naming
			// a nonexistent type.
			return &Type{Kind: TyList, Elem: &Type{Kind: TyAny}}, nil
		case "dict":
			return &Type{Kind: TyDict, Key: &Type{Kind: TyAny}, Val: &Type{Kind: TyAny}}, nil
		case "tuple":
			return &Type{Kind: TyList, Elem: &Type{Kind: TyAny}}, nil
		case "set", "frozenset":
			return &Type{Kind: TyList, Elem: &Type{Kind: TyAny}}, nil
		case "Self":
			// PEP 673 `typing.Self`: refers to the enclosing class.
			// When lowered inside `lowerClass`, currentLoweringClass
			// holds the name. Free-function context falls back to `any`.
			if currentLoweringClass != "" {
				return &Type{Kind: TyNamed, Name: currentLoweringClass}, nil
			}
			return &Type{Kind: TyAny}, nil
		case "Any", "Callable", "Iterable", "Iterator", "Sequence",
			"Mapping", "MutableMapping", "MutableSequence", "Collection",
			"Hashable", "Reversible", "Container", "Sized", "object",
			"Final", "ClassVar", "TypeAlias", "Never", "NoReturn":
			// typing bare aliases — lower to `any` so user code can pass
			// values around without further annotation.
			return &Type{Kind: TyAny}, nil
		case "bytes":
			// gopy treats bytes as Go's string.
			return &Type{Kind: TyStr}, nil
		default:
			return &Type{Kind: TyNamed, Name: n.Str("id")}, nil
		}
	case "Constant":
		// `-> None` parses as Constant(None); forward-reference string
		// annotations like `-> "Foo"` parse as Constant("Foo") and resolve
		// to a named user type.
		switch v := n["value"].(type) {
		case nil:
			return &Type{Kind: TyNone}, nil
		case string:
			switch v {
			case "int":
				return &Type{Kind: TyInt}, nil
			case "float":
				return &Type{Kind: TyFloat}, nil
			case "str":
				return &Type{Kind: TyStr}, nil
			case "bool":
				return &Type{Kind: TyBool}, nil
			}
			return &Type{Kind: TyNamed, Name: v}, nil
		}
		return nil, fmt.Errorf("unsupported constant annotation")
	case "BinOp":
		// `int | str` style unions desugar to `any`. `T | None` (or
		// `None | T`) where T is a class narrows to T so the nullable
		// pointer carries the class type for attribute access.
		op := n.Child("op")
		if op != nil && op.Type() == "BitOr" {
			left := n.Child("left")
			right := n.Child("right")
			leftTy, lErr := lowerAnnotation(left)
			rightTy, rErr := lowerAnnotation(right)
			if lErr == nil && rErr == nil && leftTy != nil && rightTy != nil {
				if leftTy.Kind == TyNone && rightTy.Kind == TyNamed {
					return rightTy, nil
				}
				if rightTy.Kind == TyNone && leftTy.Kind == TyNamed {
					return leftTy, nil
				}
			}
			return &Type{Kind: TyAny}, nil
		}
		return nil, fmt.Errorf("unsupported annotation BinOp %q", op.Type())
	case "Subscript":
		// list[T] / dict[K, V] / tuple[...] / Optional[T] / Union[...]
		base := n.Child("value")
		if base.Type() != "Name" {
			return nil, fmt.Errorf("generic annotation: base must be Name, got %q", base.Type())
		}
		switch base.Str("id") {
		case "Optional":
			// typing.Optional[T] — lowered to T when T is a class (Go's
			// pointer-to-struct can be nil), otherwise to `any` so None
			// remains assignable.
			inner, err := lowerAnnotation(n.Child("slice"))
			if err == nil && inner != nil && inner.Kind == TyNamed {
				return inner, nil
			}
			return &Type{Kind: TyAny}, nil
		case "Union":
			// typing.Union[...] — same lowering as the `|` operator
			// form: collapse to any. Components are not tracked.
			return &Type{Kind: TyAny}, nil
		case "List", "Iterable", "Sequence", "Set", "FrozenSet",
			"MutableSequence", "Collection", "Iterator", "Reversible":
			elem, err := lowerAnnotation(n.Child("slice"))
			if err != nil {
				return nil, err
			}
			return &Type{Kind: TyList, Elem: elem}, nil
		case "Final", "ClassVar", "Annotated":
			// `Final[T]` / `ClassVar[T]` / `Annotated[T, ...]` carry T as
			// the first subscript arg and metadata thereafter — gopy keeps
			// only T, dropping the wrapper.
			sl := n.Child("slice")
			if sl == nil {
				return &Type{Kind: TyAny}, nil
			}
			// Annotated takes (T, metadata...) — pick first elt.
			if sl.Type() == "Tuple" {
				elts := sl.Children("elts")
				if len(elts) == 0 {
					return &Type{Kind: TyAny}, nil
				}
				return lowerAnnotation(elts[0])
			}
			return lowerAnnotation(sl)
		case "Type", "type":
			// `Type[Foo]` — used as a runtime class object. gopy doesn't
			// reify classes, so lower to `any`.
			return &Type{Kind: TyAny}, nil
		case "Mapping", "MutableMapping":
			// typing.Mapping[K, V] / MutableMapping[K, V] — both lower
			// like Dict[K, V] so dict.get() etc. dispatch normally.
			sl := n.Child("slice")
			if sl == nil || sl.Type() != "Tuple" {
				return &Type{Kind: TyAny}, nil
			}
			elts := sl.Children("elts")
			if len(elts) != 2 {
				return &Type{Kind: TyAny}, nil
			}
			kt, err := lowerAnnotation(elts[0])
			if err != nil {
				return nil, err
			}
			vt, err := lowerAnnotation(elts[1])
			if err != nil {
				return nil, err
			}
			return &Type{Kind: TyDict, Key: kt, Val: vt}, nil
		case "Tuple":
			// typing.Tuple[T, U, ...] — match the lowercase tuple[...] path.
			sl := n.Child("slice")
			if sl == nil {
				return nil, fmt.Errorf("Tuple annotation requires component types")
			}
			var elts []parser.Node
			if sl.Type() == "Tuple" {
				elts = sl.Children("elts")
			} else {
				elts = []parser.Node{sl}
			}
			out := &Type{Kind: TyTuple}
			for _, e := range elts {
				et, err := lowerAnnotation(e)
				if err != nil {
					return nil, err
				}
				out.Tuple = append(out.Tuple, et)
			}
			return out, nil
		case "Callable":
			// `Callable[[A, B], R]` — preserve the signature as TyFunc so
			// lambda assignments can re-lower their bodies with concrete
			// param types and codegen can emit the matching `func(...)`
			// type. Args = Tuple([params-list, ret]) where the first elt
			// is a List literal of param types and the second is the ret.
			sl := n.Child("slice")
			if sl == nil || sl.Type() != "Tuple" {
				return &Type{Kind: TyAny}, nil
			}
			elts := sl.Children("elts")
			if len(elts) != 2 {
				return &Type{Kind: TyAny}, nil
			}
			out := &Type{Kind: TyFunc}
			paramsNode := elts[0]
			if paramsNode.Type() == "List" {
				for _, p := range paramsNode.Children("elts") {
					pt, err := lowerAnnotation(p)
					if err != nil {
						return nil, err
					}
					out.FuncParams = append(out.FuncParams, pt)
				}
			} else if paramsNode.Type() == "Constant" {
				// `Callable[..., R]` (Ellipsis params) — leave params empty.
			}
			ret, err := lowerAnnotation(elts[1])
			if err != nil {
				return nil, err
			}
			out.FuncRet = ret
			return out, nil
		case "Dict":
			sl := n.Child("slice")
			if sl.Type() != "Tuple" {
				return nil, fmt.Errorf("Dict annotation requires two type args")
			}
			elts := sl.Children("elts")
			if len(elts) != 2 {
				return nil, fmt.Errorf("Dict annotation requires 2 args")
			}
			kt, err := lowerAnnotation(elts[0])
			if err != nil {
				return nil, err
			}
			vt, err := lowerAnnotation(elts[1])
			if err != nil {
				return nil, err
			}
			return &Type{Kind: TyDict, Key: kt, Val: vt}, nil
		case "list":
			elem, err := lowerAnnotation(n.Child("slice"))
			if err != nil {
				return nil, fmt.Errorf("list[...]: %w", err)
			}
			return &Type{Kind: TyList, Elem: elem}, nil
		case "set", "frozenset", "MutableSet":
			// `set[T]` / `frozenset[T]` — gopy collapses to a Go slice
			// since there's no native set type. Membership / iteration
			// behave like a list; dedup is the caller's responsibility.
			elem, err := lowerAnnotation(n.Child("slice"))
			if err != nil {
				return nil, fmt.Errorf("set[...]: %w", err)
			}
			return &Type{Kind: TyList, Elem: elem}, nil
		case "dict", "defaultdict", "OrderedDict", "Counter":
			sl := n.Child("slice")
			if sl.Type() != "Tuple" {
				return nil, fmt.Errorf("dict annotation requires two type args")
			}
			elts := sl.Children("elts")
			if len(elts) != 2 {
				return nil, fmt.Errorf("dict annotation requires exactly 2 args, got %d", len(elts))
			}
			kt, err := lowerAnnotation(elts[0])
			if err != nil {
				return nil, fmt.Errorf("dict key: %w", err)
			}
			vt, err := lowerAnnotation(elts[1])
			if err != nil {
				return nil, fmt.Errorf("dict val: %w", err)
			}
			return &Type{Kind: TyDict, Key: kt, Val: vt}, nil
		case "tuple":
			// `tuple[T, U, ...]` carries a typed component list. On a
			// function return slot the transpiler emits Go multi-value
			// return; in other positions it falls back to a slice of any.
			sl := n.Child("slice")
			if sl == nil {
				return nil, fmt.Errorf("tuple annotation requires component types")
			}
			var elts []parser.Node
			if sl.Type() == "Tuple" {
				elts = sl.Children("elts")
			} else {
				elts = []parser.Node{sl}
			}
			out := &Type{Kind: TyTuple}
			for _, e := range elts {
				t, err := lowerAnnotation(e)
				if err != nil {
					return nil, fmt.Errorf("tuple component: %w", err)
				}
				out.Tuple = append(out.Tuple, t)
			}
			return out, nil
		case "Literal", "LiteralString":
			// Literal["a", "b", ...] — accepted as `any` since gopy
			// doesn't enforce the literal-value constraint at runtime.
			return &Type{Kind: TyAny}, nil
		case "TypedDict":
			return &Type{Kind: TyDict, Key: &Type{Kind: TyStr}, Val: &Type{Kind: TyAny}}, nil
		case "Required", "NotRequired", "ReadOnly":
			// Wrapper hints — unwrap to the inner annotation.
			return lowerAnnotation(n.Child("slice"))
		default:
			return nil, fmt.Errorf("unsupported generic annotation base %q", base.Str("id"))
		}
	default:
		return nil, fmt.Errorf("unsupported annotation kind %q", n.Type())
	}
}

// scope tracks variable declarations so the transpiler can pick `:=` vs `=`.
type scope struct{ vars map[string]*Type }

func newScope() *scope { return &scope{vars: map[string]*Type{}} }

func (s *scope) declare(name string, ty *Type) bool {
	if _, ok := s.vars[name]; ok {
		return false
	}
	s.vars[name] = ty
	return true
}

func (s *scope) lookup(name string) (*Type, bool) {
	t, ok := s.vars[name]
	return t, ok
}

func lowerBody(stmts []parser.Node, sc *scope) ([]Stmt, error) {
	out := make([]Stmt, 0, len(stmts))
	for _, s := range stmts {
		st, err := lowerStmt(s, sc)
		if err != nil {
			return nil, err
		}
		if st == nil {
			continue // e.g. `pass`
		}
		// Synthetic Block expansion: a single Python statement that
		// lowered to multiple IR statements (e.g. `a = b = 0`).
		if blk, ok := st.(*Block); ok {
			out = append(out, blk.Body...)
			continue
		}
		out = append(out, st)
	}
	return out, nil
}

// lowerStarUnpack synthesizes the assigns for `a, *b, c = xs` and the
// `a, *b = xs` / `*a, b = xs` variants. The RHS is bound to a temp
// `__star_unpack_N` so it evaluates once even when the assignment
// destructures into many names.
//
// The lowering produces a Block of:
//
//	__star_unpack_N := xs
//	a              := __star_unpack_N[0]
//	b              := __star_unpack_N[1 : len(__star_unpack_N)-postCount]
//	c              := __star_unpack_N[len(__star_unpack_N)-postCount+0]
//	...
//
// Each subscript / slice expression uses the IR's Subscript / Slice
// nodes so existing codegen handles them. `c.left` for the post-star
// names is computed at runtime via `len(...) - postCount + j`.
func lowerStarUnpack(n parser.Node, sc *scope, elts []parser.Node, starIdx int) (Stmt, error) {
	rhs, err := lowerExpr(n.Child("value"), sc)
	if err != nil {
		return nil, err
	}
	starUnpackCounter++
	tmp := fmt.Sprintf("__star_unpack_%d", starUnpackCounter)
	sc.declare(tmp, nil)
	stmts := []Stmt{&Assign{Target: tmp, Value: rhs, Decl: true}}

	preCount := starIdx
	postCount := len(elts) - starIdx - 1
	// Pre-star Names: __tmp[i].
	for i := 0; i < preCount; i++ {
		t := elts[i]
		if t.Type() != "Name" {
			return nil, fmt.Errorf("line %d: nested star-unpack not supported", n.Lineno())
		}
		name := t.Str("id")
		decl := sc.declare(name, nil)
		stmts = append(stmts, &Assign{
			Target: name,
			Value: &Subscript{
				Value: &Name{N: tmp},
				Index: &IntLit{V: int64(i)},
			},
			Decl: decl,
		})
	}
	// Star target: __tmp[preCount : len(__tmp)-postCount]. Names emitted
	// in source order so the star binding lands between pre and post.
	starInner := elts[starIdx].Child("value")
	if starInner == nil || starInner.Type() != "Name" {
		return nil, fmt.Errorf("line %d: starred LHS must wrap a Name", n.Lineno())
	}
	starName := starInner.Str("id")
	if starName != "_" {
		decl := sc.declare(starName, nil)
		var lowExpr, highExpr Expr
		if preCount > 0 {
			lowExpr = &IntLit{V: int64(preCount)}
		}
		if postCount > 0 {
			highExpr = &BinOp{
				Op: "-",
				L:  &Call{Func: &Name{N: "len"}, Args: []Expr{&Name{N: tmp}}},
				R:  &IntLit{V: int64(postCount)},
			}
		}
		stmts = append(stmts, &Assign{
			Target: starName,
			Value: &Slice{
				Value: &Name{N: tmp},
				Low:   lowExpr,
				High:  highExpr,
			},
			Decl: decl,
		})
	}
	// Post-star Names: __tmp[len(__tmp)-postCount+j].
	for j := 0; j < postCount; j++ {
		t := elts[starIdx+1+j]
		if t.Type() != "Name" {
			return nil, fmt.Errorf("line %d: nested star-unpack not supported", n.Lineno())
		}
		name := t.Str("id")
		decl := sc.declare(name, nil)
		// idx = len(__tmp) - postCount + j  → as len(__tmp) - (postCount - j)
		offset := postCount - j
		idx := &BinOp{
			Op: "-",
			L:  &Call{Func: &Name{N: "len"}, Args: []Expr{&Name{N: tmp}}},
			R:  &IntLit{V: int64(offset)},
		}
		stmts = append(stmts, &Assign{
			Target: name,
			Value: &Subscript{
				Value: &Name{N: tmp},
				Index: idx,
			},
			Decl: decl,
		})
	}
	return &Block{Body: stmts}, nil
}

func lowerStmt(n parser.Node, sc *scope) (Stmt, error) {
	switch n.Type() {
	case "Import", "ImportFrom":
		// Function-level imports are no-ops in gopy — names resolve via
		// the module-level alias table that was already populated when
		// the file was loaded. Drop the statement.
		return nil, nil
	case "Expr":
		// `yield X` parses as Expr(value=Yield(value=X)). Catch it here so
		// we can treat it as a control-flow statement rather than a
		// value-producing expression — generators don't return per-yield.
		if v := n.Child("value"); v != nil && v.Type() == "Yield" {
			var x Expr
			if val := v.Child("value"); val != nil {
				e, err := lowerExpr(val, sc)
				if err != nil {
					return nil, err
				}
				x = e
			}
			return &Yield{X: x}, nil
		}
		// `yield from X` parses as Expr(value=YieldFrom(value=X)).
		if v := n.Child("value"); v != nil && v.Type() == "YieldFrom" {
			inner, err := lowerExpr(v.Child("value"), sc)
			if err != nil {
				return nil, err
			}
			return &YieldFrom{Iter: inner}, nil
		}
		x, err := lowerExpr(n.Child("value"), sc)
		if err != nil {
			return nil, err
		}
		// Drop bare `None` / `...` placeholder expression statements (the
		// canonical body for an `@abstractmethod` stub, or a docstring-
		// style standalone constant). Keeping them emits invalid Go.
		if _, ok := x.(*NoneLit); ok {
			return nil, nil
		}
		if _, ok := x.(*StrLit); ok {
			// Bare string literal statements are docstrings — skip.
			return nil, nil
		}
		return &ExprStmt{X: x}, nil
	case "Return":
		v := n.Child("value")
		if v == nil {
			return &Return{X: nil}, nil
		}
		x, err := lowerExpr(v, sc)
		if err != nil {
			return nil, err
		}
		return &Return{X: x}, nil
	case "Assign":
		targets := n.Children("targets")
		// Chained assignment: `a = b = 0`. Python parses this as a single
		// Assign with multiple targets sharing one value. Lower to a
		// sequence of standalone assigns; the value is re-emitted at each
		// call site, so callers that pass side-effecting expressions
		// should hoist them first.
		if len(targets) > 1 {
			val, err := lowerExpr(n.Child("value"), sc)
			if err != nil {
				return nil, err
			}
			var assigns []Stmt
			for _, t := range targets {
				if t.Type() != "Name" {
					return nil, fmt.Errorf("line %d: chained assignment requires bare-name targets", n.Lineno())
				}
				name := t.Str("id")
				decl := sc.declare(name, val.TypeOf())
				assigns = append(assigns, &Assign{Target: name, Value: val, Decl: decl})
			}
			// Wrap multiple assigns in a synthetic Try-less block. We don't
			// have a generic Block IR; emit them as separate statements
			// through a small "fan-out" wrapper. Cheaper: return the
			// first and rely on lowerBody flattening — but lowerBody
			// expects single Stmt return. Use a ChainedAssign helper.
			return &Block{Body: assigns}, nil
		}
		tgt := targets[0]
		// Tuple unpacking: `a, b = x, y`. Requires the RHS to be a Tuple
		// literal of the same arity; arbitrary iterables would need
		// per-call runtime help we don't emit yet.
		if tgt.Type() == "Tuple" {
			elts := tgt.Children("elts")
			// Star-expr LHS detection: `first, *rest = xs` or
			// `*head, last = xs` or `a, *mid, z = xs`. Synthesize per-
			// position assigns over a shared temp so the RHS is
			// evaluated exactly once.
			starIdx := -1
			for i, t := range elts {
				if t.Type() == "Starred" {
					if starIdx >= 0 {
						return nil, fmt.Errorf("line %d: only one starred target allowed", n.Lineno())
					}
					starIdx = i
				}
			}
			if starIdx >= 0 {
				return lowerStarUnpack(n, sc, elts, starIdx)
			}
			var names []string
			for _, t := range elts {
				if t.Type() != "Name" {
					return nil, fmt.Errorf("line %d: tuple unpack target must be a Name", n.Lineno())
				}
				names = append(names, t.Str("id"))
			}
			valNode := n.Child("value")
			// `a, b = f()` where f returns tuple[T, U]: keep one
			// Value in the IR and let codegen emit Go multi-return.
			// Tuple-literal RHS still expands per-element.
			if valNode.Type() == "Tuple" {
				rhs := valNode.Children("elts")
				if len(rhs) != len(elts) {
					return nil, fmt.Errorf("line %d: tuple unpack arity mismatch (%d vs %d)", n.Lineno(), len(elts), len(rhs))
				}
				var values []Expr
				for _, e := range rhs {
					ve, err := lowerExpr(e, sc)
					if err != nil {
						return nil, err
					}
					values = append(values, ve)
				}
				declAll := true
				for i, nm := range names {
					if !sc.declare(nm, values[i].TypeOf()) {
						declAll = false
					}
				}
				return &MultiAssign{Targets: names, Values: values, Decl: declAll}, nil
			}
			// Non-tuple RHS: must be callable returning multi-value at codegen.
			ve, err := lowerExpr(valNode, sc)
			if err != nil {
				return nil, err
			}
			declAll := true
			for _, nm := range names {
				if !sc.declare(nm, nil) {
					declAll = false
				}
			}
			return &MultiAssign{Targets: names, Values: []Expr{ve}, Decl: declAll}, nil
		}
		val, err := lowerExpr(n.Child("value"), sc)
		if err != nil {
			return nil, err
		}
		switch tgt.Type() {
		case "Name":
			name := tgt.Str("id")
			decl := sc.declare(name, val.TypeOf())
			return &Assign{Target: name, Value: val, Decl: decl}, nil
		case "Subscript":
			// Slice assignment `xs[a:b] = ys` rewrites to
			// `xs = append(append(xs[:a], ys...), xs[b:]...)`. We detect
			// it by inspecting the Subscript's slice child — if it's a
			// Slice node, route through a synthesized Assign over a
			// ListLit star-spread expression.
			sliceChild := tgt.Child("slice")
			if sliceChild != nil && sliceChild.Type() == "Slice" {
				name, ok := tgt.Child("value"), false
				if name != nil && name.Type() == "Name" {
					ok = true
				}
				if !ok {
					return nil, fmt.Errorf("line %d: slice assignment requires a bare-name target", n.Lineno())
				}
				targetName := name.Str("id")
				// Build low/high bounds (defaulting to 0 / len(target)).
				var lowExpr, highExpr Expr
				if l := sliceChild.Child("lower"); l != nil {
					lowExpr, err = lowerExpr(l, sc)
					if err != nil {
						return nil, err
					}
				}
				if h := sliceChild.Child("upper"); h != nil {
					highExpr, err = lowerExpr(h, sc)
					if err != nil {
						return nil, err
					}
				}
				if sliceChild.Child("step") != nil {
					return nil, fmt.Errorf("line %d: slice assignment with step is not supported", n.Lineno())
				}
				if lowExpr == nil {
					lowExpr = &IntLit{V: 0}
				}
				if highExpr == nil {
					highExpr = &Call{Func: &Name{N: "len"}, Args: []Expr{&Name{N: targetName}}}
				}
				// New value: list literal whose first chunk is xs[:low],
				// then spread of RHS, then xs[high:].
				prefix := &Slice{Value: &Name{N: targetName}, High: lowExpr}
				suffix := &Slice{Value: &Name{N: targetName}, Low: highExpr}
				rebuilt := &ListLit{Elems: []Expr{
					&Starred{Value: prefix},
					&Starred{Value: val},
					&Starred{Value: suffix},
				}}
				return &Assign{Target: targetName, Value: rebuilt, Decl: false}, nil
			}
			obj, err := lowerExpr(tgt.Child("value"), sc)
			if err != nil {
				return nil, err
			}
			idx, err := lowerExpr(sliceChild, sc)
			if err != nil {
				return nil, err
			}
			return &AssignSub{Target: obj, Index: idx, Value: val}, nil
		case "Attribute":
			recv, err := lowerExpr(tgt.Child("value"), sc)
			if err != nil {
				return nil, err
			}
			return &AssignAttr{Target: recv, Name: tgt.Str("attr"), Value: val}, nil
		default:
			return nil, fmt.Errorf("line %d: unsupported assignment target %q", n.Lineno(), tgt.Type())
		}
	case "AugAssign":
		// x += y  →  x = x + y. Supports Name, Attribute, and Subscript targets.
		opNode := n.Child("op")
		op, err := lowerBinOpKind(opNode.Type())
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", n.Lineno(), err)
		}
		rhs, err := lowerExpr(n.Child("value"), sc)
		if err != nil {
			return nil, err
		}
		tgt := n.Child("target")
		switch tgt.Type() {
		case "Name":
			name := tgt.Str("id")
			lhsTy, _ := sc.lookup(name)
			lhs := &Name{N: name, Ty: lhsTy}
			bin := &BinOp{Op: op, L: lhs, R: rhs, Ty: promoteOp(op, lhs.Ty, rhs.TypeOf()), InPlace: true}
			return &Assign{Target: name, Value: bin, Decl: false}, nil
		case "Attribute":
			recv, err := lowerExpr(tgt.Child("value"), sc)
			if err != nil {
				return nil, err
			}
			attrName := tgt.Str("attr")
			lhs := &Attribute{Recv: recv, Name: attrName, Ty: rhs.TypeOf()}
			bin := &BinOp{Op: op, L: lhs, R: rhs, Ty: promoteOp(op, lhs.Ty, rhs.TypeOf()), InPlace: true}
			return &AssignAttr{Target: recv, Name: attrName, Value: bin}, nil
		case "Subscript":
			obj, err := lowerExpr(tgt.Child("value"), sc)
			if err != nil {
				return nil, err
			}
			idx, err := lowerExpr(tgt.Child("slice"), sc)
			if err != nil {
				return nil, err
			}
			var elemTy *Type
			if t := obj.TypeOf(); t != nil {
				switch t.Kind {
				case TyList:
					elemTy = t.Elem
				case TyDict:
					elemTy = t.Val
				}
			}
			lhs := &Subscript{Value: obj, Index: idx, Ty: elemTy}
			bin := &BinOp{Op: op, L: lhs, R: rhs, Ty: promoteOp(op, elemTy, rhs.TypeOf()), InPlace: true}
			return &AssignSub{Target: obj, Index: idx, Value: bin}, nil
		default:
			return nil, fmt.Errorf("line %d: unsupported AugAssign target %q", n.Lineno(), tgt.Type())
		}
	case "AnnAssign":
		tgt := n.Child("target")
		ty, err := lowerAnnotation(n.Child("annotation"))
		if err != nil {
			return nil, err
		}
		var val Expr
		if v := n.Child("value"); v != nil {
			val, err = lowerExpr(v, sc)
			if err != nil {
				return nil, err
			}
			// Lambda RHS with Callable LHS: re-lower the body with the
			// concrete param types so x*2 etc. resolve as int64 rather
			// than the default any.
			if lam, ok := val.(*Lambda); ok && ty != nil && ty.Kind == TyFunc {
				body, lerr := LowerLambdaBody(lam, ty.FuncParams)
				if lerr == nil {
					lam.Body = body
					lam.Ty = ty
					if len(lam.Params) == len(ty.FuncParams) {
						for i, p := range ty.FuncParams {
							lam.Params[i].Ty = p
						}
					}
				}
			}
		}
		switch tgt.Type() {
		case "Name":
			name := tgt.Str("id")
			decl := sc.declare(name, ty)
			if val == nil {
				// `name: T` with no initializer — register the type in
				// the local scope so later references see it, but don't
				// emit a statement (Go forbids declaring a typed var
				// without using it, and Python's bare annotation is a
				// declaration-only hint).
				_ = decl
				return nil, nil
			}
			return &Assign{Target: name, Ty: ty, Value: val, Decl: decl}, nil
		case "Attribute":
			// `self.items: list[T] = []` etc. Treat as a plain attribute
			// store — the annotation is parsed (so the field gets the
			// declared type during class-body field discovery) but the
			// statement itself emits as AssignAttr.
			recv, err := lowerExpr(tgt.Child("value"), sc)
			if err != nil {
				return nil, err
			}
			if val == nil {
				return nil, nil
			}
			return &AssignAttr{Target: recv, Name: tgt.Str("attr"), Value: val, AnnTy: ty}, nil
		default:
			return nil, fmt.Errorf("line %d: AnnAssign target %q not supported", n.Lineno(), tgt.Type())
		}
	case "If":
		cond, err := lowerExpr(n.Child("test"), sc)
		if err != nil {
			return nil, err
		}
		// Walrus hoist: any `name := value` inside the condition is
		// peeled off into a preceding Assign so the binding survives
		// into the if body (and Go's expression-level constraints).
		pre, condExpr := hoistNamedExprs(cond)
		thenS, err := lowerBody(n.Children("body"), sc)
		if err != nil {
			return nil, err
		}
		elseS, err := lowerBody(n.Children("orelse"), sc)
		if err != nil {
			return nil, err
		}
		ifStmt := &If{Cond: condExpr, Then: thenS, Else: elseS}
		if len(pre) == 0 {
			return ifStmt, nil
		}
		return &Block{Body: append(pre, ifStmt)}, nil
	case "While":
		cond, err := lowerExpr(n.Child("test"), sc)
		if err != nil {
			return nil, err
		}
		body, err := lowerBody(n.Children("body"), sc)
		if err != nil {
			return nil, err
		}
		var orelse []Stmt
		if oe := n.Children("orelse"); len(oe) > 0 {
			ob, err := lowerBody(oe, sc)
			if err != nil {
				return nil, err
			}
			orelse = ob
		}
		// Same walrus hoist as `if`. Note: the assigned name is
		// re-bound every iteration because the hoisted Assign sits
		// before the loop header in the IR — we replicate it back
		// into the body's tail to maintain Python semantics.
		pre, condExpr := hoistNamedExprs(cond)
		whileStmt := &While{Cond: condExpr, Body: body, OrElse: orelse}
		if len(pre) == 0 {
			return whileStmt, nil
		}
		// Re-evaluate the hoisted assignment at the end of each
		// iteration so subsequent condition checks see fresh values.
		// The re-evaluation is a reassignment, never a re-declaration,
		// to avoid shadowing the outer binding.
		for _, s := range pre {
			if a, ok := s.(*Assign); ok {
				whileStmt.Body = append(whileStmt.Body, &Assign{Target: a.Target, Value: a.Value, Decl: false, Ty: a.Ty})
			}
		}
		return &Block{Body: append(pre, whileStmt)}, nil
	case "For", "AsyncFor":
		// gopy strips async semantics: `async for x in it` lowers as the
		// regular for-loop. Real concurrency requires hand-written
		// goroutines — this is a compat shim so libraries using async-
		// style APIs compile.
		return lowerFor(n, sc)
	case "Try":
		body, err := lowerBody(n.Children("body"), sc)
		if err != nil {
			return nil, err
		}
		var handlers []ExceptHandler
		for _, h := range n.Children("handlers") {
			if h.Type() != "ExceptHandler" {
				return nil, fmt.Errorf("line %d: unexpected handler %q", h.Lineno(), h.Type())
			}
			eh := ExceptHandler{VarName: h.Str("name")}
			if t := h.Child("type"); t != nil {
				switch t.Type() {
				case "Name":
					eh.ClassName = t.Str("id")
					eh.ClassNames = []string{t.Str("id")}
				case "Tuple":
					// `except (A, B, C) as e:` — collect each class name.
					for _, e := range t.Children("elts") {
						if e.Type() != "Name" {
							return nil, fmt.Errorf("line %d: complex except type not supported (F3)", h.Lineno())
						}
						eh.ClassNames = append(eh.ClassNames, e.Str("id"))
					}
					if len(eh.ClassNames) > 0 {
						eh.ClassName = eh.ClassNames[0]
					}
				default:
					return nil, fmt.Errorf("line %d: complex except type not supported (F3)", h.Lineno())
				}
			}
			hSc := &scope{vars: copyVars(sc.vars)}
			if eh.VarName != "" {
				hSc.declare(eh.VarName, &Type{Kind: TyNamed, Name: eh.ClassName})
			}
			hb, err := lowerBody(h.Children("body"), hSc)
			if err != nil {
				return nil, err
			}
			eh.Body = hb
			handlers = append(handlers, eh)
		}
		finally, err := lowerBody(n.Children("finalbody"), sc)
		if err != nil {
			return nil, err
		}
		var orelse []Stmt
		if oe := n.Children("orelse"); len(oe) > 0 {
			ob, err := lowerBody(oe, sc)
			if err != nil {
				return nil, err
			}
			orelse = ob
		}
		return &Try{Body: body, Handlers: handlers, Finally: finally, OrElse: orelse}, nil
	case "Raise":
		excNode := n.Child("exc")
		causeNode := n.Child("cause")
		var cause Expr
		if causeNode != nil {
			c, err := lowerExpr(causeNode, sc)
			if err != nil {
				return nil, err
			}
			cause = c
		}
		if excNode == nil {
			return &Raise{Exc: nil, Cause: cause}, nil
		}
		exc, err := lowerExpr(excNode, sc)
		if err != nil {
			return nil, err
		}
		return &Raise{Exc: exc, Cause: cause}, nil
	case "Pass":
		return nil, nil
	case "Global":
		// Python's `global x` opts subsequent writes into the module-level
		// binding. Mark the names as already-declared in the local scope
		// using the module-level type so future Assigns lower with the
		// existing-binding shape; codegen then routes them to `=` against
		// the package var.
		if raw, ok := n["names"].([]any); ok {
			for _, x := range raw {
				name, _ := x.(string)
				if name == "" {
					continue
				}
				t, _ := moduleScopeLookup(name)
				if t == nil {
					t = &Type{Kind: TyUnknown}
				}
				sc.declare(name, t)
			}
		}
		return nil, nil
	case "Nonlocal":
		// Same shape: opt-in to an enclosing-scope binding. We don't track
		// enclosing-fn scopes explicitly, so this is best-effort — declare
		// the names as TyUnknown and let later inference fill in.
		if raw, ok := n["names"].([]any); ok {
			for _, x := range raw {
				name, _ := x.(string)
				if name == "" {
					continue
				}
				sc.declare(name, &Type{Kind: TyUnknown})
			}
		}
		return nil, nil
	case "Match":
		return lowerMatch(n, sc)
	case "FunctionDef":
		// Nested function inside another body: lower it as a regular
		// Func and wrap in a LocalFunc statement so codegen can emit
		// `name := func(...) ret { ... }`.
		fn, err := lowerFunc(n)
		if err != nil {
			return nil, err
		}
		if fn.IsGenerator {
			return nil, fmt.Errorf("line %d: nested generators are not yet supported", n.Lineno())
		}
		// Make the nested name visible to subsequent statements so
		// recursive references and post-decl uses see it. Type is
		// loose (TyUnknown) — Go infers from the closure literal.
		sc.declare(fn.Name, &Type{Kind: TyUnknown})
		return &LocalFunc{Fn: fn}, nil
	case "Break":
		return &Break{}, nil
	case "Continue":
		return &Continue{}, nil
	case "With", "AsyncWith":
		// Async-with lowers identically to sync with — gopy drops the
		// async layer (see also: async def / await collapsing in lowerFunc).
		return lowerWith(n, sc)
	case "Assert":
		cond, err := lowerExpr(n.Child("test"), sc)
		if err != nil {
			return nil, err
		}
		var msg Expr
		if m := n.Child("msg"); m != nil {
			msg, err = lowerExpr(m, sc)
			if err != nil {
				return nil, err
			}
		}
		return &Assert{Cond: cond, Msg: msg}, nil
	case "Delete":
		var targets []Expr
		for _, t := range n.Children("targets") {
			e, err := lowerExpr(t, sc)
			if err != nil {
				return nil, err
			}
			targets = append(targets, e)
		}
		return &Del{Targets: targets}, nil
	default:
		return nil, fmt.Errorf("line %d: unsupported statement %q", n.Lineno(), n.Type())
	}
}

func lowerExpr(n parser.Node, sc *scope) (Expr, error) {
	switch n.Type() {
	case "Constant":
		return lowerConstant(n)
	case "Name":
		name := n.Str("id")
		t, ok := sc.lookup(name)
		if !ok {
			if gt, gok := moduleScopeLookup(name); gok {
				t = gt
			}
		}
		return &Name{N: name, Ty: t}, nil
	case "BinOp":
		l, err := lowerExpr(n.Child("left"), sc)
		if err != nil {
			return nil, err
		}
		r, err := lowerExpr(n.Child("right"), sc)
		if err != nil {
			return nil, err
		}
		opNode := n.Child("op")
		op, err := lowerBinOpKind(opNode.Type())
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", n.Lineno(), err)
		}
		ty := promoteOp(op, l.TypeOf(), r.TypeOf())
		// `int ** negative_literal` returns a float in CPython
		// (`2 ** -3 == 0.125`). Detect the literal-negative-exponent case
		// at lower time so the BinOp's static type reads as TyFloat —
		// otherwise downstream casts would clamp the math.Pow result.
		if op == "**" && ty != nil && ty.Kind == TyInt {
			isNegLit := false
			if lit, ok := r.(*IntLit); ok && lit.V < 0 {
				isNegLit = true
			} else if u, ok := r.(*UnaryOp); ok && u.Op == "-" {
				if lit, ok := u.X.(*IntLit); ok && lit.V > 0 {
					isNegLit = true
				}
			}
			if isNegLit {
				ty = &Type{Kind: TyFloat}
			}
		}
		return &BinOp{Op: op, L: l, R: r, Ty: ty}, nil
	case "Compare":
		ops := n.Children("ops")
		comps := n.Children("comparators")
		if len(ops) == 0 || len(ops) != len(comps) {
			return nil, fmt.Errorf("line %d: malformed Compare node", n.Lineno())
		}
		// Single comparison — produce a plain CmpOp.
		left, err := lowerExpr(n.Child("left"), sc)
		if err != nil {
			return nil, err
		}
		if len(ops) == 1 {
			r, err := lowerExpr(comps[0], sc)
			if err != nil {
				return nil, err
			}
			op, err := lowerCmpKind(ops[0].Type())
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", n.Lineno(), err)
			}
			return &CmpOp{Op: op, L: left, R: r, Ty: &Type{Kind: TyBool}}, nil
		}
		// Chained comparison `a < b < c` → `(a < b) and (b < c)`, with the
		// middle operand evaluated once. We approximate "once" by binding
		// the rendered expression to a local in codegen; at the IR level
		// we just fold to nested BoolOp(And).
		var chain Expr
		prev := left
		for i, opNode := range ops {
			r, err := lowerExpr(comps[i], sc)
			if err != nil {
				return nil, err
			}
			op, err := lowerCmpKind(opNode.Type())
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", n.Lineno(), err)
			}
			step := &CmpOp{Op: op, L: prev, R: r, Ty: &Type{Kind: TyBool}}
			if chain == nil {
				chain = step
			} else {
				chain = &BoolOp{Op: "and", L: chain, R: step, Ty: &Type{Kind: TyBool}}
			}
			prev = r
		}
		return chain, nil
	case "BoolOp":
		vals := n.Children("values")
		if len(vals) < 2 {
			return nil, fmt.Errorf("line %d: BoolOp with <2 values", n.Lineno())
		}
		opNode := n.Child("op")
		var op string
		switch opNode.Type() {
		case "And":
			op = "and"
		case "Or":
			op = "or"
		default:
			return nil, fmt.Errorf("line %d: unsupported BoolOp %q", n.Lineno(), opNode.Type())
		}
		// Left-fold into binary tree.
		acc, err := lowerExpr(vals[0], sc)
		if err != nil {
			return nil, err
		}
		for _, v := range vals[1:] {
			r, err := lowerExpr(v, sc)
			if err != nil {
				return nil, err
			}
			lt := acc.TypeOf()
			rt := r.TypeOf()
			ty := &Type{Kind: TyBool}
			if lt != nil && rt != nil && lt.Kind == rt.Kind &&
				lt.Kind != TyBool && lt.Kind != TyUnknown && lt.Kind != TyAny {
				ty = lt
			}
			acc = &BoolOp{Op: op, L: acc, R: r, Ty: ty}
		}
		return acc, nil
	case "UnaryOp":
		x, err := lowerExpr(n.Child("operand"), sc)
		if err != nil {
			return nil, err
		}
		opNode := n.Child("op")
		var op string
		switch opNode.Type() {
		case "USub":
			op = "-"
		case "Not":
			op = "not"
		case "UAdd":
			op = "+"
		case "Invert":
			op = "~"
		default:
			return nil, fmt.Errorf("line %d: unsupported UnaryOp %q", n.Lineno(), opNode.Type())
		}
		ty := x.TypeOf()
		if op == "not" {
			ty = &Type{Kind: TyBool}
		}
		return &UnaryOp{Op: op, X: x, Ty: ty}, nil
	case "Call":
		var args []Expr
		for _, a := range n.Children("args") {
			x, err := lowerExpr(a, sc)
			if err != nil {
				return nil, err
			}
			args = append(args, x)
		}
		var kws []Keyword
		for _, kw := range n.Children("keywords") {
			name := kw.Str("arg")
			v, err := lowerExpr(kw.Child("value"), sc)
			if err != nil {
				return nil, err
			}
			// `name == ""` marks a `**dict` splat; we propagate it
			// through the IR with the sentinel name "**" so codegen
			// can spot it without a separate enum.
			if name == "" {
				name = "**"
			}
			kws = append(kws, Keyword{Name: name, Value: v})
		}
		// Method call: callee is Attribute(value, attr).
		if fnNode := n.Child("func"); fnNode.Type() == "Attribute" {
			recv, err := lowerExpr(fnNode.Child("value"), sc)
			if err != nil {
				return nil, err
			}
			method := fnNode.Str("attr")
			retTy := inferMethodRet(recv, method)
			return &MethodCall{Recv: recv, Method: method, Args: args, Keywords: kws, Ty: retTy}, nil
		}
		callee, err := lowerExpr(n.Child("func"), sc)
		if err != nil {
			return nil, err
		}
		retTy := &Type{Kind: TyUnknown}
		// Builtin return types we can pin down without a full inference pass.
		// Useful so list/dict comprehensions over `len(x)` know to type
		// their element/value as int rather than `any`.
		if name, ok := callee.(*Name); ok {
			switch name.N {
			case "len":
				retTy = &Type{Kind: TyInt}
			case "int":
				retTy = &Type{Kind: TyInt}
			case "float":
				retTy = &Type{Kind: TyFloat}
			case "str":
				retTy = &Type{Kind: TyStr}
			case "bool":
				retTy = &Type{Kind: TyBool}
			case "set", "frozenset", "list", "sorted", "reversed":
				// Pass through the input list's element type so later
				// `sorted(set(xs))` keeps a typed slice rather than `any`.
				if len(args) > 0 {
					if t := args[0].TypeOf(); t != nil && t.Kind == TyList {
						retTy = &Type{Kind: TyList, Elem: t.Elem}
					} else if name.N == "list" {
						// `list(range(N))` and `list(range(a, b))` materialize
						// an int slice — recognize so downstream `sorted` /
						// `islice` see a typed list rather than TyUnknown.
						if call, ok := args[0].(*Call); ok {
							if inner, ok := call.Func.(*Name); ok && inner.N == "range" {
								retTy = &Type{Kind: TyList, Elem: &Type{Kind: TyInt}}
							}
						}
					}
				}
			case "chr":
				retTy = &Type{Kind: TyStr}
			case "ord", "hash", "id":
				retTy = &Type{Kind: TyInt}
			case "hex", "oct", "bin", "repr":
				retTy = &Type{Kind: TyStr}
			default:
				// `Foo(...)` where Foo is a registered class returns a
				// TyNamed instance — surfaced here so ListLit / DictLit
				// element-type inference sees the concrete class type.
				if ty, ok := moduleScopeLookup(name.N); ok && ty != nil && ty.Kind == TyNamed {
					retTy = ty
				}
			}
		}
		return &Call{Func: callee, Args: args, Keywords: kws, Ty: retTy}, nil
	case "Attribute":
		recv, err := lowerExpr(n.Child("value"), sc)
		if err != nil {
			return nil, err
		}
		return &Attribute{Recv: recv, Name: n.Str("attr"), Ty: &Type{Kind: TyUnknown}}, nil
	case "Subscript":
		val, err := lowerExpr(n.Child("value"), sc)
		if err != nil {
			return nil, err
		}
		sliceNode := n.Child("slice")
		// Python AST: `xs[a:b]` parses slice as Slice(lower, upper, step);
		// `xs[k]` parses slice as a plain expression. Disambiguate here.
		if sliceNode != nil && sliceNode.Type() == "Slice" {
			var low, high, step Expr
			if l := sliceNode.Child("lower"); l != nil {
				e, err := lowerExpr(l, sc)
				if err != nil {
					return nil, err
				}
				low = e
			}
			if h := sliceNode.Child("upper"); h != nil {
				e, err := lowerExpr(h, sc)
				if err != nil {
					return nil, err
				}
				high = e
			}
			if s := sliceNode.Child("step"); s != nil {
				e, err := lowerExpr(s, sc)
				if err != nil {
					return nil, err
				}
				step = e
			}
			// Slice preserves the container type for list/str; unknown otherwise.
			ty := val.TypeOf()
			return &Slice{Value: val, Low: low, High: high, Step: step, Ty: ty}, nil
		}
		idx, err := lowerExpr(sliceNode, sc)
		if err != nil {
			return nil, err
		}
		var elemTy *Type
		if t := val.TypeOf(); t != nil {
			switch t.Kind {
			case TyList:
				elemTy = t.Elem
			case TyDict:
				elemTy = t.Val
			case TyStr:
				elemTy = &Type{Kind: TyStr}
			}
		}
		return &Subscript{Value: val, Index: idx, Ty: elemTy}, nil
	case "Tuple":
		// Lowered as a list literal: same Go shape (slice), same access
		// pattern. Trade-off: we lose Python's immutability semantics.
		fallthrough
	case "Set":
		// `{1, 2, 3}` lowers to the same slice shape. Membership via
		// `in` works because we walk the slice; uniqueness is not
		// enforced — users that need that should call list(set(...)).
		fallthrough
	case "List":
		elts := n.Children("elts")
		var elems []Expr
		var elemTy *Type
		for _, e := range elts {
			x, err := lowerExpr(e, sc)
			if err != nil {
				return nil, err
			}
			elems = append(elems, x)
			if elemTy == nil {
				elemTy = x.TypeOf()
			} else if !sameType(elemTy, x.TypeOf()) {
				elemTy = &Type{Kind: TyAny}
			}
		}
		if elemTy == nil {
			elemTy = &Type{Kind: TyAny}
		}
		return &ListLit{Elems: elems, ElemTy: elemTy, Ty: &Type{Kind: TyList, Elem: elemTy}}, nil
	case "Dict":
		keys := n.Children("keys")
		vals := n.Children("values")
		var ks, vs []Expr
		var kt, vt *Type
		for i, k := range keys {
			ke, err := lowerExpr(k, sc)
			if err != nil {
				return nil, err
			}
			ve, err := lowerExpr(vals[i], sc)
			if err != nil {
				return nil, err
			}
			ks = append(ks, ke)
			vs = append(vs, ve)
			if kt == nil {
				kt = ke.TypeOf()
			} else if !sameType(kt, ke.TypeOf()) {
				kt = &Type{Kind: TyAny}
			}
			if vt == nil {
				vt = ve.TypeOf()
			} else if !sameType(vt, ve.TypeOf()) {
				vt = &Type{Kind: TyAny}
			}
		}
		if kt == nil {
			kt = &Type{Kind: TyAny}
		}
		if vt == nil {
			vt = &Type{Kind: TyAny}
		}
		return &DictLit{Keys: ks, Vals: vs, KeyTy: kt, ValTy: vt, Ty: &Type{Kind: TyDict, Key: kt, Val: vt}}, nil
	case "Lambda":
		args := n.Child("args")
		var params []Param
		for _, a := range args.Children("args") {
			params = append(params, Param{Name: a.Str("arg"), Ty: &Type{Kind: TyAny}})
		}
		// Lower body once with TyAny-typed params for the fallback path
		// (lambda used as a value). Specialized call sites re-lower
		// through LowerLambdaBody to inject concrete types.
		bodySc := newScope()
		for _, p := range params {
			bodySc.declare(p.Name, p.Ty)
		}
		bodyNode := n.Child("body")
		body, err := lowerExpr(bodyNode, bodySc)
		if err != nil {
			return nil, fmt.Errorf("line %d: lambda body: %w", n.Lineno(), err)
		}
		return &Lambda{Params: params, Body: body, BodyAST: bodyNode, Ty: &Type{Kind: TyUnknown}}, nil
	case "NamedExpr":
		// Walrus assignment. The lower pass only generates this node;
		// surrounding statement-level lowering hoists it into a
		// preceding Assign. Direct use outside a hoist context is not
		// supported and errors at codegen.
		tgt := n.Child("target")
		if tgt.Type() != "Name" {
			return nil, fmt.Errorf("walrus target must be a Name")
		}
		v, err := lowerExpr(n.Child("value"), sc)
		if err != nil {
			return nil, err
		}
		sc.declare(tgt.Str("id"), v.TypeOf())
		return &NamedExpr{Name: tgt.Str("id"), Value: v, Ty: v.TypeOf()}, nil
	case "IfExp":
		cond, err := lowerExpr(n.Child("test"), sc)
		if err != nil {
			return nil, err
		}
		thenE, err := lowerExpr(n.Child("body"), sc)
		if err != nil {
			return nil, err
		}
		elseE, err := lowerExpr(n.Child("orelse"), sc)
		if err != nil {
			return nil, err
		}
		ty := thenE.TypeOf()
		if ty == nil || ty.Kind == TyUnknown {
			ty = elseE.TypeOf()
		}
		return &IfExpr{Cond: cond, Then: thenE, Else: elseE, Ty: ty}, nil
	case "ListComp":
		return lowerListComp(n, sc)
	case "GeneratorExp":
		// `(expr for var in iter [if cond])` — same shape as ListComp,
		// just immutable in Python. We materialize eagerly to a slice
		// since we don't have a lazy generator-of-expressions runtime.
		return lowerListComp(n, sc)
	case "SetComp":
		// `{expr for var in iter [if cond]}` — same shape as ListComp.
		// Lower to a ListComp; the receiver's `set[T]` annotation drives
		// later set-style codegen (membership via `in`, etc.).
		return lowerListComp(n, sc)
	case "DictComp":
		return lowerDictComp(n, sc)
	case "Starred":
		inner, err := lowerExpr(n.Child("value"), sc)
		if err != nil {
			return nil, err
		}
		return &Starred{Value: inner, Ty: inner.TypeOf()}, nil
	case "Await":
		// gopy doesn't model real async — `await expr` collapses to
		// `expr` (synchronous evaluation). Lets libraries with async
		// APIs compile and behave correctly when the awaitable is a
		// simple value rather than a deferred computation.
		return lowerExpr(n.Child("value"), sc)
	case "JoinedStr":
		// f"..." — list of Constant(str) and FormattedValue.
		var parts []FStrPart
		for _, p := range n.Children("values") {
			switch p.Type() {
			case "Constant":
				s, _ := p["value"].(string)
				parts = append(parts, FStrPart{Lit: s})
			case "FormattedValue":
				x, err := lowerExpr(p.Child("value"), sc)
				if err != nil {
					return nil, err
				}
				spec := ""
				var specExprs []Expr
				if fs := p.Child("format_spec"); fs != nil && fs.Type() == "JoinedStr" {
					// format_spec is itself a JoinedStr. Static segments
					// are appended to the spec verbatim; nested
					// FormattedValue placeholders (`f"{x:>{width}}"`)
					// emit a `%v` marker and push the expr into
					// specExprs so codegen can fmt.Sprintf the dynamic
					// spec at runtime.
					for _, sp := range fs.Children("values") {
						switch sp.Type() {
						case "Constant":
							s, _ := sp["value"].(string)
							spec += s
						case "FormattedValue":
							sx, err := lowerExpr(sp.Child("value"), sc)
							if err != nil {
								return nil, err
							}
							spec += "%v"
							specExprs = append(specExprs, sx)
						}
					}
				}
				var conv byte
				if cv, ok := p["conversion"].(float64); ok && cv >= 0 {
					ci := int(cv)
					if ci == int('r') || ci == int('s') || ci == int('a') {
						conv = byte(ci)
					}
				}
				parts = append(parts, FStrPart{Expr: x, Spec: spec, SpecExprs: specExprs, Conv: conv})
			default:
				return nil, fmt.Errorf("line %d: unsupported f-string part %q", n.Lineno(), p.Type())
			}
		}
		return &FStr{Parts: parts, Ty: &Type{Kind: TyStr}}, nil
	default:
		return nil, fmt.Errorf("line %d: unsupported expression %q", n.Lineno(), n.Type())
	}
}

// sameType returns true when two types are structurally equal for inference.
// Nil is treated as "unknown" and only matches nil.
func sameType(a, b *Type) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case TyList:
		return sameType(a.Elem, b.Elem)
	case TyDict:
		return sameType(a.Key, b.Key) && sameType(a.Val, b.Val)
	case TyNamed:
		return a.Name == b.Name
	}
	return true
}

func lowerConstant(n parser.Node) (Expr, error) {
	v := n["value"]
	// Prefer the explicit _const_kind tag the AST dumper adds — it lets us
	// distinguish `1` (Python int) from `1.0` (Python float), which both
	// land here as float64 after JSON decode.
	kind, _ := n["_const_kind"].(string)
	switch x := v.(type) {
	case bool:
		return &BoolLit{V: x, Ty: &Type{Kind: TyBool}}, nil
	case float64:
		if kind == "float" {
			return &FloatLit{V: x, Ty: &Type{Kind: TyFloat}}, nil
		}
		if kind == "int" {
			return &IntLit{V: int64(x), Ty: &Type{Kind: TyInt}}, nil
		}
		// Fallback when the dumper didn't tag (older trees): integer-valued
		// floats become int, anything else stays float.
		if x == float64(int64(x)) {
			return &IntLit{V: int64(x), Ty: &Type{Kind: TyInt}}, nil
		}
		return &FloatLit{V: x, Ty: &Type{Kind: TyFloat}}, nil
	case string:
		return &StrLit{V: x, Ty: &Type{Kind: TyStr}}, nil
	case nil:
		return &NoneLit{Ty: &Type{Kind: TyNone}}, nil
	case map[string]any:
		if kind == "complex" {
			re, _ := x["real"].(float64)
			im, _ := x["imag"].(float64)
			return &ComplexLit{Real: re, Imag: im, Ty: &Type{Kind: TyComplex}}, nil
		}
		return nil, fmt.Errorf("line %d: unsupported constant map shape", n.Lineno())
	default:
		return nil, fmt.Errorf("line %d: unsupported constant %T", n.Lineno(), v)
	}
}

func lowerBinOpKind(s string) (string, error) {
	switch s {
	case "Add":
		return "+", nil
	case "Sub":
		return "-", nil
	case "Mult":
		return "*", nil
	case "Div":
		return "/", nil
	case "FloorDiv":
		return "//", nil
	case "Mod":
		return "%", nil
	case "BitOr":
		return "|", nil
	case "BitAnd":
		return "&", nil
	case "BitXor":
		return "^", nil
	case "LShift":
		return "<<", nil
	case "RShift":
		return ">>", nil
	case "MatMult":
		return "@", nil
	case "Pow":
		return "**", nil
	}
	return "", fmt.Errorf("unsupported BinOp %q", s)
}

func lowerCmpKind(s string) (string, error) {
	switch s {
	case "Eq":
		return "==", nil
	case "NotEq":
		return "!=", nil
	case "Lt":
		return "<", nil
	case "LtE":
		return "<=", nil
	case "Gt":
		return ">", nil
	case "GtE":
		return ">=", nil
	// `x is y` / `x is not y` lower to plain equality. The compared side
	// is typically `None` (i.e. a NoneLit) and codegen handles that as
	// a nil check; for non-None operands Go's `==` still gives the right
	// answer for pointer identity.
	case "Is":
		return "==", nil
	case "IsNot":
		return "!=", nil
	// `in` / `not in` need runtime-shape knowledge (str/list/dict/set);
	// codegen looks at the right operand's type to pick the right form.
	case "In":
		return "in", nil
	case "NotIn":
		return "notin", nil
	}
	return "", fmt.Errorf("unsupported Compare op %q", s)
}

// hoistNamedExprs walks an expression looking for top-level NamedExpr
// nodes (walrus assignments). Each is peeled off into a preceding
// Assign statement; the returned expression substitutes a plain Name
// where the walrus stood. Only handles NamedExpr appearances at the
// top of common operator trees — deeper nestings (lambda bodies,
// function call args, comprehensions) fall through unchanged and rely
// on the codegen-level error path.
// HoistNamedExprs is the exported form used by transpile codegen for
// listcomp / dictcomp lowering — peels NamedExpr nodes out of an
// expression into prior Assign statements and returns the rewritten
// expression with plain Name references in place of the walrus.
func HoistNamedExprs(e Expr) ([]Stmt, Expr) {
	return hoistNamedExprs(e)
}

func hoistNamedExprs(e Expr) ([]Stmt, Expr) {
	var pre []Stmt
	var visit func(Expr) Expr
	visit = func(x Expr) Expr {
		switch n := x.(type) {
		case *NamedExpr:
			pre = append(pre, &Assign{Target: n.Name, Value: n.Value, Decl: true, Ty: n.Ty})
			return &Name{N: n.Name, Ty: n.Ty}
		case *BinOp:
			n.L = visit(n.L)
			n.R = visit(n.R)
		case *CmpOp:
			n.L = visit(n.L)
			n.R = visit(n.R)
		case *BoolOp:
			n.L = visit(n.L)
			n.R = visit(n.R)
		case *UnaryOp:
			n.X = visit(n.X)
		}
		return x
	}
	out := visit(e)
	return pre, out
}

// inferMethodRet returns the static return type of a method call when
// the receiver type + method name unambiguously imply it. Used so
// chained calls like `s.strip().replace(...)` know `strip()` yields a
// `str` and can dispatch `replace` against TyStr. Returns TyUnknown
// when no inference applies.
func inferMethodRet(recv Expr, method string) *Type {
	rt := recv.TypeOf()
	if rt == nil {
		return &Type{Kind: TyUnknown}
	}
	switch rt.Kind {
	case TyStr:
		switch method {
		case "upper", "lower", "strip", "lstrip", "rstrip", "replace",
			"title", "capitalize", "swapcase", "casefold", "center",
			"ljust", "rjust", "zfill", "expandtabs", "translate",
			"encode", "removeprefix", "removesuffix", "format":
			return &Type{Kind: TyStr}
		case "startswith", "endswith", "isalpha", "isdigit", "isalnum",
			"isspace", "isupper", "islower", "isdecimal", "isnumeric",
			"isidentifier", "isprintable", "istitle", "isascii":
			return &Type{Kind: TyBool}
		case "find", "rfind", "index", "rindex", "count":
			return &Type{Kind: TyInt}
		case "split", "rsplit", "splitlines", "partition", "rpartition":
			return &Type{Kind: TyList, Elem: &Type{Kind: TyStr}}
		case "join":
			return &Type{Kind: TyStr}
		}
	case TyList:
		switch method {
		case "append":
			return &Type{Kind: TyNone}
		}
	case TyDict:
		switch method {
		case "get":
			return rt.Val
		case "keys":
			return &Type{Kind: TyList, Elem: rt.Key}
		case "values":
			return &Type{Kind: TyList, Elem: rt.Val}
		case "items":
			// (k, v) pairs lowered to list-of-list-any; standalone use
			// is rare and best paired with for-loop unpacking, which is
			// special-cased earlier in lowerFor.
			return &Type{Kind: TyList, Elem: &Type{Kind: TyAny}}
		}
	}
	return &Type{Kind: TyUnknown}
}

// substituteName rewrites every Name(from) anywhere in ss to Name(to).
// Used so @classmethod bodies can refer to `cls` while we emit the
// class's actual name in Go output.
func substituteName(ss []Stmt, from, to string) {
	for _, s := range ss {
		subStmt(s, from, to)
	}
}

func subStmt(s Stmt, from, to string) {
	switch x := s.(type) {
	case *ExprStmt:
		x.X = subExpr(x.X, from, to)
	case *Assign:
		x.Value = subExpr(x.Value, from, to)
	case *AssignSub:
		x.Target = subExpr(x.Target, from, to)
		x.Index = subExpr(x.Index, from, to)
		x.Value = subExpr(x.Value, from, to)
	case *AssignAttr:
		x.Target = subExpr(x.Target, from, to)
		x.Value = subExpr(x.Value, from, to)
	case *Return:
		if x.X != nil {
			x.X = subExpr(x.X, from, to)
		}
	case *If:
		x.Cond = subExpr(x.Cond, from, to)
		substituteName(x.Then, from, to)
		substituteName(x.Else, from, to)
	case *While:
		x.Cond = subExpr(x.Cond, from, to)
		substituteName(x.Body, from, to)
	case *ForRange:
		x.Start = subExpr(x.Start, from, to)
		x.Stop = subExpr(x.Stop, from, to)
		if x.Step != nil {
			x.Step = subExpr(x.Step, from, to)
		}
		substituteName(x.Body, from, to)
	case *ForEach:
		x.Iter = subExpr(x.Iter, from, to)
		substituteName(x.Body, from, to)
	case *Try:
		substituteName(x.Body, from, to)
		for i := range x.Handlers {
			substituteName(x.Handlers[i].Body, from, to)
		}
		substituteName(x.Finally, from, to)
	case *Raise:
		if x.Exc != nil {
			x.Exc = subExpr(x.Exc, from, to)
		}
	case *WithFile:
		x.Path = subExpr(x.Path, from, to)
		substituteName(x.Body, from, to)
	case *Yield:
		if x.X != nil {
			x.X = subExpr(x.X, from, to)
		}
	}
}

func subExpr(e Expr, from, to string) Expr {
	if e == nil {
		return nil
	}
	switch x := e.(type) {
	case *Name:
		if x.N == from {
			return &Name{N: to, Ty: x.Ty}
		}
		return x
	case *BinOp:
		x.L = subExpr(x.L, from, to)
		x.R = subExpr(x.R, from, to)
	case *CmpOp:
		x.L = subExpr(x.L, from, to)
		x.R = subExpr(x.R, from, to)
	case *BoolOp:
		x.L = subExpr(x.L, from, to)
		x.R = subExpr(x.R, from, to)
	case *UnaryOp:
		x.X = subExpr(x.X, from, to)
	case *Call:
		x.Func = subExpr(x.Func, from, to)
		for i := range x.Args {
			x.Args[i] = subExpr(x.Args[i], from, to)
		}
		for i := range x.Keywords {
			x.Keywords[i].Value = subExpr(x.Keywords[i].Value, from, to)
		}
	case *MethodCall:
		x.Recv = subExpr(x.Recv, from, to)
		for i := range x.Args {
			x.Args[i] = subExpr(x.Args[i], from, to)
		}
	case *Attribute:
		x.Recv = subExpr(x.Recv, from, to)
	case *Subscript:
		x.Value = subExpr(x.Value, from, to)
		x.Index = subExpr(x.Index, from, to)
	case *ListLit:
		for i := range x.Elems {
			x.Elems[i] = subExpr(x.Elems[i], from, to)
		}
	case *DictLit:
		for i := range x.Keys {
			x.Keys[i] = subExpr(x.Keys[i], from, to)
			x.Vals[i] = subExpr(x.Vals[i], from, to)
		}
	case *FStr:
		for i := range x.Parts {
			if x.Parts[i].Expr != nil {
				x.Parts[i].Expr = subExpr(x.Parts[i].Expr, from, to)
			}
		}
	case *ListComp:
		x.Iter = subExpr(x.Iter, from, to)
		x.Elt = subExpr(x.Elt, from, to)
		if x.Cond != nil {
			x.Cond = subExpr(x.Cond, from, to)
		}
	case *DictComp:
		x.Iter = subExpr(x.Iter, from, to)
		x.Key = subExpr(x.Key, from, to)
		x.Val = subExpr(x.Val, from, to)
		if x.Cond != nil {
			x.Cond = subExpr(x.Cond, from, to)
		}
	}
	return e
}

// lowerListComp lowers `[Elt for Var in Iter if Cond]` to a ListComp IR.
// F7 supports exactly one generator and at most one filter.
func lowerListComp(n parser.Node, sc *scope) (Expr, error) {
	gens := n.Children("generators")
	if len(gens) < 1 {
		return nil, fmt.Errorf("line %d: comprehension requires at least one generator", n.Lineno())
	}
	innerSc := &scope{vars: copyVars(sc.vars)}
	var primary CompGen
	var extra []CompGen
	for gi, g := range gens {
		tgt := g.Child("target")
		var varName, varName2 string
		var iter Expr
		var elemTy *Type
		if tgt.Type() == "Tuple" {
			elts := tgt.Children("elts")
			if len(elts) != 2 || elts[0].Type() != "Name" || elts[1].Type() != "Name" {
				return nil, fmt.Errorf("line %d: comprehension tuple target must be two names", n.Lineno())
			}
			itNode := g.Child("iter")
			if itNode.Type() != "Call" || itNode.Child("func").Type() != "Attribute" || itNode.Child("func").Str("attr") != "items" {
				return nil, fmt.Errorf("line %d: comprehension tuple target requires dict.items()", n.Lineno())
			}
			d, err := lowerExpr(itNode.Child("func").Child("value"), innerSc)
			if err != nil {
				return nil, err
			}
			dt := d.TypeOf()
			if dt == nil || dt.Kind != TyDict {
				return nil, fmt.Errorf("line %d: .items() must be called on a typed dict", n.Lineno())
			}
			varName = elts[0].Str("id")
			varName2 = elts[1].Str("id")
			iter = d
			innerSc.declare(varName, dt.Key)
			innerSc.declare(varName2, dt.Val)
		} else {
			if tgt.Type() != "Name" {
				return nil, fmt.Errorf("line %d: comprehension target must be a single name", n.Lineno())
			}
			it, err := lowerExpr(g.Child("iter"), innerSc)
			if err != nil {
				return nil, err
			}
			iter = it
			varName = tgt.Str("id")
			if t := iter.TypeOf(); t != nil {
				switch t.Kind {
				case TyList:
					elemTy = t.Elem
				case TyDict:
					elemTy = t.Key
				case TyStr:
					elemTy = &Type{Kind: TyStr}
				}
			}
			// `range(...)` calls don't carry a TyList type but yield ints —
			// declare the loop var as TyInt so downstream expressions infer.
			if elemTy == nil {
				if call, ok := iter.(*Call); ok {
					if nm, ok := call.Func.(*Name); ok && nm.N == "range" {
						elemTy = &Type{Kind: TyInt}
					}
				}
			}
			innerSc.declare(varName, elemTy)
		}
		var cond Expr
		if ifs := g.Children("ifs"); len(ifs) > 0 {
			if len(ifs) > 1 {
				return nil, fmt.Errorf("line %d: only one if-filter per generator supported", n.Lineno())
			}
			c, err := lowerExpr(ifs[0], innerSc)
			if err != nil {
				return nil, err
			}
			cond = c
		}
		gen := CompGen{Var: varName, Var2: varName2, Iter: iter, Cond: cond, ElemTy: elemTy}
		if gi == 0 {
			primary = gen
		} else {
			extra = append(extra, gen)
		}
	}
	elt, err := lowerExpr(n.Child("elt"), innerSc)
	if err != nil {
		return nil, err
	}
	resultElemTy := elt.TypeOf()
	if resultElemTy == nil {
		resultElemTy = &Type{Kind: TyAny}
	}
	return &ListComp{
		Elt:    elt,
		Var:    primary.Var,
		Var2:   primary.Var2,
		Iter:   primary.Iter,
		Cond:   primary.Cond,
		ElemTy: resultElemTy,
		Ty:     &Type{Kind: TyList, Elem: resultElemTy},
		Extra:  extra,
	}, nil
}

// lowerDictComp lowers `{K: V for Var in Iter if Cond}`.
func lowerDictComp(n parser.Node, sc *scope) (Expr, error) {
	gens := n.Children("generators")
	if len(gens) < 1 {
		return nil, fmt.Errorf("line %d: dict comprehension requires at least one generator", n.Lineno())
	}
	innerSc := &scope{vars: copyVars(sc.vars)}
	var primary CompGen
	var extra []CompGen
	for gi, g := range gens {
		tgt := g.Child("target")
		var varName, varName2 string
		var iter Expr
		var elemTy *Type
		if tgt.Type() == "Tuple" {
			elts := tgt.Children("elts")
			if len(elts) != 2 || elts[0].Type() != "Name" || elts[1].Type() != "Name" {
				return nil, fmt.Errorf("line %d: comprehension tuple target must be two names", n.Lineno())
			}
			itNode := g.Child("iter")
			if itNode.Type() != "Call" || itNode.Child("func").Type() != "Attribute" || itNode.Child("func").Str("attr") != "items" {
				return nil, fmt.Errorf("line %d: comprehension tuple target requires dict.items()", n.Lineno())
			}
			d, err := lowerExpr(itNode.Child("func").Child("value"), innerSc)
			if err != nil {
				return nil, err
			}
			dt := d.TypeOf()
			if dt == nil || dt.Kind != TyDict {
				return nil, fmt.Errorf("line %d: .items() must be called on a typed dict", n.Lineno())
			}
			varName = elts[0].Str("id")
			varName2 = elts[1].Str("id")
			iter = d
			innerSc.declare(varName, dt.Key)
			innerSc.declare(varName2, dt.Val)
		} else {
			if tgt.Type() != "Name" {
				return nil, fmt.Errorf("line %d: comprehension target must be a single name", n.Lineno())
			}
			it, err := lowerExpr(g.Child("iter"), innerSc)
			if err != nil {
				return nil, err
			}
			iter = it
			varName = tgt.Str("id")
			if t := iter.TypeOf(); t != nil {
				switch t.Kind {
				case TyList:
					elemTy = t.Elem
				case TyDict:
					elemTy = t.Key
				case TyStr:
					elemTy = &Type{Kind: TyStr}
				}
			}
			// `range(...)` calls don't carry a TyList type but yield ints —
			// declare the loop var as TyInt so downstream expressions infer.
			if elemTy == nil {
				if call, ok := iter.(*Call); ok {
					if nm, ok := call.Func.(*Name); ok && nm.N == "range" {
						elemTy = &Type{Kind: TyInt}
					}
				}
			}
			innerSc.declare(varName, elemTy)
		}
		var cond Expr
		if ifs := g.Children("ifs"); len(ifs) > 0 {
			if len(ifs) > 1 {
				return nil, fmt.Errorf("line %d: only one if-filter per generator supported", n.Lineno())
			}
			c, err := lowerExpr(ifs[0], innerSc)
			if err != nil {
				return nil, err
			}
			cond = c
		}
		gen := CompGen{Var: varName, Var2: varName2, Iter: iter, Cond: cond, ElemTy: elemTy}
		if gi == 0 {
			primary = gen
		} else {
			extra = append(extra, gen)
		}
	}
	key, err := lowerExpr(n.Child("key"), innerSc)
	if err != nil {
		return nil, err
	}
	val, err := lowerExpr(n.Child("value"), innerSc)
	if err != nil {
		return nil, err
	}
	kt := key.TypeOf()
	if kt == nil {
		kt = &Type{Kind: TyAny}
	}
	vt := val.TypeOf()
	if vt == nil {
		vt = &Type{Kind: TyAny}
	}
	return &DictComp{
		Key:   key,
		Val:   val,
		Var:   primary.Var,
		Var2:  primary.Var2,
		Iter:  primary.Iter,
		Cond:  primary.Cond,
		KeyTy: kt,
		ValTy: vt,
		Ty:    &Type{Kind: TyDict, Key: kt, Val: vt},
		Extra: extra,
	}, nil
}

// lowerMatch handles Python's `match` statement. F+ recognizes only
// literal patterns (MatchValue / MatchSingleton) and the wildcard
// (MatchAs with no pattern), each optionally guarded by `if cond`.
// Sequence / mapping / class patterns are rejected at lower time.
func lowerMatch(n parser.Node, sc *scope) (Stmt, error) {
	subject, err := lowerExpr(n.Child("subject"), sc)
	if err != nil {
		return nil, err
	}
	m := &Match{Subject: subject}
	for _, caseNode := range n.Children("cases") {
		pat := caseNode.Child("pattern")
		var classPat *MatchClassPat
		var seqPat *MatchSeqPat
		var mapPat *MatchMapPat
		var patterns []Expr
		capture := ""
		// `case Foo(...) as b:` / `case [...] as b:` etc. — unwrap to the
		// inner pattern and stash the capture name on the case.
		if pat != nil && pat.Type() == "MatchAs" {
			if inner := pat.Child("pattern"); inner != nil {
				capture = pat.Str("name")
				pat = inner
			} else if name := pat.Str("name"); name != "" {
				// `case name:` / `case name if cond:` — bind name to the
				// subject. Guard parsed below if present, so we still need
				// to honor it here.
				var guard Expr
				if g := caseNode.Child("guard"); g != nil {
					ge, err := lowerExpr(g, sc)
					if err != nil {
						return nil, err
					}
					guard = ge
				}
				body, err := lowerBody(caseNode.Children("body"), sc)
				if err != nil {
					return nil, err
				}
				m.Cases = append(m.Cases, MatchCase{
					Body:    body,
					Capture: name,
					Guard:   guard,
				})
				continue
			}
		}
		switch {
		case pat != nil && pat.Type() == "MatchClass":
			cp, err := lowerMatchClassPattern(pat, sc)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", caseNode.Lineno(), err)
			}
			classPat = cp
		case pat != nil && pat.Type() == "MatchSequence":
			sp, err := lowerMatchSeqPattern(pat, sc)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", caseNode.Lineno(), err)
			}
			seqPat = sp
		case pat != nil && pat.Type() == "MatchMapping":
			mp, err := lowerMatchMapPattern(pat, sc)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", caseNode.Lineno(), err)
			}
			mapPat = mp
		default:
			ps, err := lowerMatchPattern(pat, sc)
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", caseNode.Lineno(), err)
			}
			patterns = ps
		}
		var guard Expr
		if g := caseNode.Child("guard"); g != nil {
			ge, err := lowerExpr(g, sc)
			if err != nil {
				return nil, err
			}
			guard = ge
		}
		body, err := lowerBody(caseNode.Children("body"), sc)
		if err != nil {
			return nil, err
		}
		m.Cases = append(m.Cases, MatchCase{Patterns: patterns, Guard: guard, Body: body, ClassPat: classPat, SeqPat: seqPat, MapPat: mapPat, Capture: capture})
	}
	return m, nil
}

// lowerMatchMapPattern parses `case {"k": v, ...}:` — partial-match
// mapping pattern. Each (key, value) pair must exist in the subject
// and match. `**rest` capture isn't supported.
func lowerMatchMapPattern(p parser.Node, sc *scope) (*MatchMapPat, error) {
	if rest := p["rest"]; rest != nil {
		if s, ok := rest.(string); ok && s != "" {
			return nil, fmt.Errorf("match mapping: `**` rest capture not supported")
		}
	}
	out := &MatchMapPat{}
	keys := p.Children("keys")
	values := p.Children("patterns")
	if len(keys) != len(values) {
		return nil, fmt.Errorf("match mapping: keys/patterns length mismatch")
	}
	for i, k := range keys {
		ke, err := lowerExpr(k, sc)
		if err != nil {
			return nil, err
		}
		ps, err := lowerMatchPattern(values[i], sc)
		if err != nil {
			return nil, err
		}
		if len(ps) != 1 {
			return nil, fmt.Errorf("match mapping: value must be a single literal pattern")
		}
		out.Keys = append(out.Keys, ke)
		out.Values = append(out.Values, ps[0])
	}
	return out, nil
}

// lowerMatchSeqPattern parses `case [v1, v2, ..., *rest, w1, ...]:`.
// Each element is either a literal (MatchValue), a singleton, a bare
// capture (MatchAs with no inner pattern), or the single allowed star.
// Nested patterns are not supported.
func lowerMatchSeqPattern(p parser.Node, sc *scope) (*MatchSeqPat, error) {
	out := &MatchSeqPat{}
	cur := &out.Elements
	for _, sub := range p.Children("patterns") {
		if sub.Type() == "MatchStar" {
			if out.HasStar {
				return nil, fmt.Errorf("match sequence: only one `*` allowed")
			}
			name := sub.Str("name")
			if name == "" {
				name = "_"
			}
			out.Star = name
			out.HasStar = true
			cur = &out.Tail
			continue
		}
		if sub.Type() == "MatchAs" && sub.Child("pattern") == nil {
			name := sub.Str("name")
			if name == "" {
				name = "_"
			}
			*cur = append(*cur, MatchSeqElt{Capture: name})
			continue
		}
		ps, err := lowerMatchPattern(sub, sc)
		if err != nil {
			return nil, err
		}
		if len(ps) != 1 {
			return nil, fmt.Errorf("match sequence element must produce one literal pattern")
		}
		*cur = append(*cur, MatchSeqElt{LitVal: ps[0]})
	}
	return out, nil
}

// lowerMatchClassPattern parses `case ClassName(kw=lit, ...)`. Positional
// captures (`Cls(x, y)`) and nested class patterns aren't supported yet.
func lowerMatchClassPattern(p parser.Node, sc *scope) (*MatchClassPat, error) {
	cls := p.Child("cls")
	if cls == nil || cls.Type() != "Name" {
		return nil, fmt.Errorf("match class pattern: class must be a bare Name")
	}
	out := &MatchClassPat{ClassName: cls.Str("id")}
	for _, pp := range p.Children("patterns") {
		switch pp.Type() {
		case "MatchAs":
			// `case Class(x)` — bare capture; inner pattern must be absent.
			if pp.Child("pattern") != nil {
				return nil, fmt.Errorf("match class pattern: only bare-name positional captures supported")
			}
			name := pp.Str("name")
			out.PosCaptures = append(out.PosCaptures, name)
		default:
			return nil, fmt.Errorf("match class pattern: positional pattern %q not supported", pp.Type())
		}
	}
	attrs := p["kwd_attrs"]
	rawAttrs, _ := attrs.([]any)
	for _, a := range rawAttrs {
		s, _ := a.(string)
		out.KwdAttrs = append(out.KwdAttrs, s)
	}
	for _, kp := range p.Children("kwd_patterns") {
		if kp.Type() != "MatchValue" && kp.Type() != "MatchSingleton" {
			return nil, fmt.Errorf("match class pattern: only literal kwd patterns supported")
		}
		ps, err := lowerMatchPattern(kp, sc)
		if err != nil {
			return nil, err
		}
		if len(ps) != 1 {
			return nil, fmt.Errorf("match class pattern: kwd pattern must produce one value")
		}
		out.KwdValues = append(out.KwdValues, ps[0])
	}
	if len(out.KwdAttrs) != len(out.KwdValues) {
		return nil, fmt.Errorf("match class pattern: kwd attrs / values length mismatch")
	}
	return out, nil
}

// lowerMatchPattern returns the literal-value expressions a `case`
// matches against. Wildcard (MatchAs with no pattern, or MatchValue
// pattern bound to `_`) returns nil to signal the default arm.
func lowerMatchPattern(p parser.Node, sc *scope) ([]Expr, error) {
	switch p.Type() {
	case "MatchValue":
		v, err := lowerExpr(p.Child("value"), sc)
		if err != nil {
			return nil, err
		}
		return []Expr{v}, nil
	case "MatchSingleton":
		v := p["value"]
		switch v.(type) {
		case nil:
			return []Expr{&NoneLit{Ty: &Type{Kind: TyNone}}}, nil
		case bool:
			return []Expr{&BoolLit{V: v.(bool), Ty: &Type{Kind: TyBool}}}, nil
		}
		return nil, fmt.Errorf("match pattern: unsupported singleton %T", v)
	case "MatchAs":
		// `case _` or `case x` — both wildcard for our subset; bare name
		// captures aren't bound to a variable yet.
		if p.Child("pattern") == nil {
			return nil, nil // wildcard
		}
		return nil, fmt.Errorf("match pattern: `as` captures not supported (use a guard or rewrite the case)")
	case "MatchOr":
		var out []Expr
		for _, sub := range p.Children("patterns") {
			one, err := lowerMatchPattern(sub, sc)
			if err != nil {
				return nil, err
			}
			if one == nil {
				return nil, fmt.Errorf("match pattern: `|` cannot include wildcard")
			}
			out = append(out, one...)
		}
		return out, nil
	}
	return nil, fmt.Errorf("match pattern: %q not supported", p.Type())
}

// lowerWith handles `with open(...) as name:` (file form) and the
// generic user-class form `with <expr> as name:` where the class
// implements __enter__ / __exit__. The user-class path produces a
// WithCM IR node; codegen resolves the dunder methods.
func lowerWith(n parser.Node, sc *scope) (Stmt, error) {
	items := n.Children("items")
	if len(items) != 1 {
		return nil, fmt.Errorf("line %d: multi-item `with` not supported (F4)", n.Lineno())
	}
	item := items[0]
	ctx := item.Child("context_expr")
	// Generic context-manager path: anything other than `open(...)`.
	isOpenCall := false
	if ctx.Type() == "Call" {
		fn := ctx.Child("func")
		if fn.Type() == "Name" && fn.Str("id") == "open" {
			isOpenCall = true
		}
	}
	if !isOpenCall {
		ctxExpr, err := lowerExpr(ctx, sc)
		if err != nil {
			return nil, err
		}
		varName := ""
		asNode := item.Child("optional_vars")
		innerSc := &scope{vars: copyVars(sc.vars)}
		if asNode != nil && asNode.Type() == "Name" {
			varName = asNode.Str("id")
			// Best-effort: bind the as-var to the ctx expression's type
			// (typically TyNamed for a class call). Codegen still treats
			// the var as the class instance.
			if t := ctxExpr.TypeOf(); t != nil {
				innerSc.declare(varName, t)
			} else {
				innerSc.declare(varName, &Type{Kind: TyUnknown})
			}
		}
		body, err := lowerBody(n.Children("body"), innerSc)
		if err != nil {
			return nil, err
		}
		return &WithCM{VarName: varName, Ctx: ctxExpr, Body: body}, nil
	}
	args := ctx.Children("args")
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("line %d: open() takes 1 or 2 positional args", n.Lineno())
	}
	pathE, err := lowerExpr(args[0], sc)
	if err != nil {
		return nil, err
	}
	mode := "r"
	if len(args) == 2 {
		if args[1].Type() != "Constant" {
			return nil, fmt.Errorf("line %d: open() mode must be a string literal", n.Lineno())
		}
		s, _ := args[1]["value"].(string)
		if s != "r" && s != "w" {
			return nil, fmt.Errorf("line %d: open() mode %q not supported (F4: only \"r\" or \"w\")", n.Lineno(), s)
		}
		mode = s
	}
	asNode := item.Child("optional_vars")
	if asNode == nil || asNode.Type() != "Name" {
		return nil, fmt.Errorf("line %d: `with open(...) as <name>` required", n.Lineno())
	}
	varName := asNode.Str("id")
	innerSc := &scope{vars: copyVars(sc.vars)}
	// File handle type tracked as a named type; codegen knows what to emit.
	innerSc.declare(varName, &Type{Kind: TyNamed, Name: "__gopy_file"})
	body, err := lowerBody(n.Children("body"), innerSc)
	if err != nil {
		return nil, err
	}
	return &WithFile{VarName: varName, Path: pathE, Mode: mode, Body: body}, nil
}

// lowerFor handles two shapes:
//   for i in range(...): ...   → ForRange (numeric loop)
//   for x in <iterable>: ...   → ForEach (range over slice/map)
func lowerFor(n parser.Node, sc *scope) (Stmt, error) {
	tgt := n.Child("target")
	var orelse []Stmt
	if oe := n.Children("orelse"); len(oe) > 0 {
		ob, err := lowerBody(oe, sc)
		if err != nil {
			return nil, err
		}
		orelse = ob
	}
	iter := n.Child("iter")
	// Two-name target: `for k, v in pairs`. Three patterns supported:
	//   `for k, v in d.items()`         — Go-native map range
	//   `for i, x in enumerate(xs)`     — Go-native slice index+value range
	//   `for a, b in zip(xs, ys)`       — paired iteration via parallel index
	if tgt.Type() == "Tuple" {
		elts := tgt.Children("elts")
		for _, e := range elts {
			if e.Type() != "Name" {
				return nil, fmt.Errorf("line %d: for-loop unpacking targets must be Names", n.Lineno())
			}
		}
		var s Stmt
		var err error
		if len(elts) == 2 {
			v1, v2 := elts[0].Str("id"), elts[1].Str("id")
			s, err = lowerForTuple(n, sc, v1, v2, iter)
		} else {
			// N-tuple unpack (N != 2): iterate once over a synthetic
			// per-element temp and destructure inside the body. Works
			// for any iterable whose static element type is a tuple
			// (heterogeneous: tuple[T0, T1, ...]) or a homogeneous list.
			s, err = lowerForTupleN(n, sc, elts, iter)
		}
		if err != nil {
			return nil, err
		}
		if len(orelse) > 0 {
			switch fe := s.(type) {
			case *ForEach:
				fe.OrElse = orelse
			case *ForRange:
				fe.OrElse = orelse
			}
		}
		return s, nil
	}
	if tgt.Type() != "Name" {
		return nil, fmt.Errorf("line %d: only single Name loop variable supported", n.Lineno())
	}
	varName := tgt.Str("id")

	// Recognize range(...) without evaluating it as a runtime call.
	if iter.Type() == "Call" {
		fn := iter.Child("func")
		if fn.Type() == "Name" && fn.Str("id") == "range" {
			args := iter.Children("args")
			if len(args) < 1 || len(args) > 3 {
				return nil, fmt.Errorf("line %d: range() takes 1..3 args", n.Lineno())
			}
			loopSc := &scope{vars: copyVars(sc.vars)}
			loopSc.declare(varName, &Type{Kind: TyInt})
			body, err := lowerBody(n.Children("body"), loopSc)
			if err != nil {
				return nil, err
			}
			var start, stop, step Expr
			switch len(args) {
			case 1:
				start = &IntLit{V: 0, Ty: &Type{Kind: TyInt}}
				st, err := lowerExpr(args[0], sc)
				if err != nil {
					return nil, err
				}
				stop = st
			case 2, 3:
				s, err := lowerExpr(args[0], sc)
				if err != nil {
					return nil, err
				}
				e, err := lowerExpr(args[1], sc)
				if err != nil {
					return nil, err
				}
				start = s
				stop = e
				if len(args) == 3 {
					sp, err := lowerExpr(args[2], sc)
					if err != nil {
						return nil, err
					}
					step = sp
				}
			}
			return &ForRange{Var: varName, Start: start, Stop: stop, Step: step, Body: body, OrElse: orelse}, nil
		}
	}

	// Generic ForEach.
	iterE, err := lowerExpr(iter, sc)
	if err != nil {
		return nil, err
	}
	var elemTy *Type
	if t := iterE.TypeOf(); t != nil {
		switch t.Kind {
		case TyList:
			elemTy = t.Elem
		case TyDict:
			elemTy = t.Key // Python iterates dict keys
		case TyStr:
			elemTy = &Type{Kind: TyStr}
		case TyNamed:
			// File handles bound by `with open(...) as fh:` carry the
			// synthetic __gopy_file type; iterating yields string lines.
			if t.Name == "__gopy_file" {
				elemTy = &Type{Kind: TyStr}
			}
		}
	}
	loopSc := &scope{vars: copyVars(sc.vars)}
	loopSc.declare(varName, elemTy)
	body, err := lowerBody(n.Children("body"), loopSc)
	if err != nil {
		return nil, err
	}
	return &ForEach{Var: varName, Iter: iterE, ElemTy: elemTy, Body: body, OrElse: orelse}, nil
}

// lowerForTupleN lowers `for a, b, c, ... in iter` for N != 2 by binding
// each loop element to a synthetic temp and destructuring it inside the
// body via Subscript / type assertions. The iter's static element type
// must be a tuple (heterogeneous) or a homogeneous list of length N —
// otherwise the body sees per-position `any` and downstream codegen has
// to coerce.
func lowerForTupleN(n parser.Node, sc *scope, elts []parser.Node, iter parser.Node) (Stmt, error) {
	iterE, err := lowerExpr(iter, sc)
	if err != nil {
		return nil, err
	}
	var elemTy *Type
	if t := iterE.TypeOf(); t != nil && t.Kind == TyList {
		elemTy = t.Elem
	}
	starUnpackCounter++
	tmp := fmt.Sprintf("__tup_unpack_%d", starUnpackCounter)
	loopSc := &scope{vars: copyVars(sc.vars)}
	loopSc.declare(tmp, elemTy)
	// Synthesize destructure assigns at the top of the body.
	var prelude []Stmt
	for i, e := range elts {
		name := e.Str("id")
		var elTy *Type
		if elemTy != nil {
			if elemTy.Kind == TyTuple && i < len(elemTy.Tuple) {
				elTy = elemTy.Tuple[i]
			} else if elemTy.Kind == TyList {
				elTy = elemTy.Elem
			}
		}
		loopSc.declare(name, elTy)
		prelude = append(prelude, &Assign{
			Target: name,
			Value: &Subscript{
				Value: &Name{N: tmp, Ty: elemTy},
				Index: &IntLit{V: int64(i)},
				Ty:    elTy,
			},
			Decl: true,
		})
	}
	body, err := lowerBody(n.Children("body"), loopSc)
	if err != nil {
		return nil, err
	}
	return &ForEach{
		Var:    tmp,
		Iter:   iterE,
		ElemTy: elemTy,
		Body:   append(prelude, body...),
	}, nil
}

// lowerForTuple lowers a two-name `for a, b in iter` form by recognizing
// the three iter shapes we can emit Go-native code for.
func lowerForTuple(n parser.Node, sc *scope, v1, v2 string, iter parser.Node) (Stmt, error) {
	// Pattern: enumerate(xs)
	if iter.Type() == "Call" {
		fn := iter.Child("func")
		if fn.Type() == "Name" && fn.Str("id") == "enumerate" {
			args := iter.Children("args")
			if len(args) < 1 || len(args) > 2 {
				return nil, fmt.Errorf("line %d: enumerate() takes 1 or 2 positional arguments", n.Lineno())
			}
			seq, err := lowerExpr(args[0], sc)
			if err != nil {
				return nil, err
			}
			var startExpr Expr
			if len(args) == 2 {
				startExpr, err = lowerExpr(args[1], sc)
				if err != nil {
					return nil, err
				}
			}
			for _, kw := range iter.Children("keywords") {
				if kw.Str("arg") == "start" {
					startExpr, err = lowerExpr(kw.Child("value"), sc)
					if err != nil {
						return nil, err
					}
				}
			}
			var elemTy *Type
			if t := seq.TypeOf(); t != nil && t.Kind == TyList {
				elemTy = t.Elem
			}
			loopSc := &scope{vars: copyVars(sc.vars)}
			loopSc.declare(v1, &Type{Kind: TyInt})
			loopSc.declare(v2, elemTy)
			body, err := lowerBody(n.Children("body"), loopSc)
			if err != nil {
				return nil, err
			}
			return &ForEach{Var: v1, Var2: v2, Iter: seq, Iter2: startExpr, ElemTy: elemTy, Kind: "enum", Body: body}, nil
		}
		if fn.Type() == "Name" && fn.Str("id") == "zip" {
			args := iter.Children("args")
			if len(args) != 2 {
				return nil, fmt.Errorf("line %d: zip() takes 2 arguments (F12)", n.Lineno())
			}
			a, err := lowerExpr(args[0], sc)
			if err != nil {
				return nil, err
			}
			b, err := lowerExpr(args[1], sc)
			if err != nil {
				return nil, err
			}
			var aElem, bElem *Type
			if t := a.TypeOf(); t != nil && t.Kind == TyList {
				aElem = t.Elem
			}
			if t := b.TypeOf(); t != nil && t.Kind == TyList {
				bElem = t.Elem
			}
			loopSc := &scope{vars: copyVars(sc.vars)}
			loopSc.declare(v1, aElem)
			loopSc.declare(v2, bElem)
			body, err := lowerBody(n.Children("body"), loopSc)
			if err != nil {
				return nil, err
			}
			return &ForEach{Var: v1, Var2: v2, Iter: a, Iter2: b, Kind: "zip", Body: body}, nil
		}
		if fn.Type() == "Name" && fn.Str("id") == "groupby" {
			args := iter.Children("args")
			if len(args) != 1 {
				return nil, fmt.Errorf("line %d: groupby() takes 1 positional argument", n.Lineno())
			}
			src, err := lowerExpr(args[0], sc)
			if err != nil {
				return nil, err
			}
			var elemTy *Type
			if t := src.TypeOf(); t != nil && t.Kind == TyList {
				elemTy = t.Elem
			}
			keyTy := elemTy
			fullIter, err := lowerExpr(iter, sc)
			if err != nil {
				return nil, err
			}
			if call, ok := fullIter.(*Call); ok {
				for _, kw := range call.Keywords {
					if kw.Name == "key" {
						if lam, ok := kw.Value.(*Lambda); ok && len(lam.Params) == 1 {
							body, lerr := LowerLambdaBody(lam, []*Type{elemTy})
							if lerr == nil {
								if t := body.TypeOf(); t != nil && t.Kind != TyUnknown {
									keyTy = t
								}
							}
						}
					}
				}
			}
			loopSc := &scope{vars: copyVars(sc.vars)}
			loopSc.declare(v1, keyTy)
			loopSc.declare(v2, &Type{Kind: TyList, Elem: elemTy})
			body, err := lowerBody(n.Children("body"), loopSc)
			if err != nil {
				return nil, err
			}
			return &ForEach{Var: v1, Var2: v2, Iter: fullIter, ElemTy: elemTy, Kind: "groupby", Body: body}, nil
		}
		// .items() on a dict — recognized when the call's func is
		// Attribute(value, "items") with value typed as dict.
		if fn.Type() == "Attribute" && fn.Str("attr") == "items" {
			d, err := lowerExpr(fn.Child("value"), sc)
			if err != nil {
				return nil, err
			}
			dt := d.TypeOf()
			if dt == nil || dt.Kind != TyDict {
				return nil, fmt.Errorf("line %d: .items() must be called on a typed dict", n.Lineno())
			}
			loopSc := &scope{vars: copyVars(sc.vars)}
			loopSc.declare(v1, dt.Key)
			loopSc.declare(v2, dt.Val)
			body, err := lowerBody(n.Children("body"), loopSc)
			if err != nil {
				return nil, err
			}
			return &ForEach{Var: v1, Var2: v2, Iter: d, Kind: "dict", Body: body}, nil
		}
	}
	// Fallback: iterate a list of 2-tuples by destructuring each element.
	iterE, err := lowerExpr(iter, sc)
	if err == nil {
		if t := iterE.TypeOf(); t != nil && t.Kind == TyList && t.Elem != nil {
			elemTy := t.Elem
			var t0, t1 *Type
			if elemTy.Kind == TyTuple && len(elemTy.Tuple) == 2 {
				t0 = elemTy.Tuple[0]
				t1 = elemTy.Tuple[1]
			} else if elemTy.Kind == TyList {
				// Homogeneous tuple (lowered to a list): both binds share
				// the inner element type.
				t0 = elemTy.Elem
				t1 = elemTy.Elem
			}
			if t0 != nil && t1 != nil {
				loopSc := &scope{vars: copyVars(sc.vars)}
				loopSc.declare(v1, t0)
				loopSc.declare(v2, t1)
				body, berr := lowerBody(n.Children("body"), loopSc)
				if berr != nil {
					return nil, berr
				}
				return &ForEach{Var: v1, Var2: v2, Iter: iterE, ElemTy: elemTy, Kind: "tuple_list", Body: body}, nil
			}
		}
	}
	return nil, fmt.Errorf("line %d: two-name for-loop requires enumerate(xs), zip(a, b), or dict.items()", n.Lineno())
}

// LowerLambdaBody rebuilds a lambda's body with the supplied parameter
// types, replacing the TyAny fallback the lambda was lowered with at
// definition time. Used by call-site specialization (map / filter /
// sorted with key=) where the iterable's element type is known.
//
// paramTypes must be the same length as l.Params; nil entries leave
// that parameter as TyAny.
func LowerLambdaBody(l *Lambda, paramTypes []*Type) (Expr, error) {
	sc := newScope()
	for i, p := range l.Params {
		ty := p.Ty
		if i < len(paramTypes) && paramTypes[i] != nil {
			ty = paramTypes[i]
		}
		sc.declare(p.Name, ty)
	}
	return lowerExpr(l.BodyAST, sc)
}

func copyVars(in map[string]*Type) map[string]*Type {
	out := make(map[string]*Type, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// promote picks the result type of a binary op. F1: numeric promotion only.
func promote(a, b *Type) *Type {
	if a == nil || b == nil {
		return &Type{Kind: TyUnknown}
	}
	if a.Kind == TyStr && b.Kind == TyStr {
		return &Type{Kind: TyStr}
	}
	if a.Kind == TyComplex || b.Kind == TyComplex {
		return &Type{Kind: TyComplex}
	}
	if a.Kind == TyFloat || b.Kind == TyFloat {
		return &Type{Kind: TyFloat}
	}
	if a.Kind == TyInt && b.Kind == TyInt {
		return &Type{Kind: TyInt}
	}
	return &Type{Kind: TyUnknown}
}

// promoteOp is promote that also knows about Python's `/` always yielding
// float — even for two int operands. Used at BinOp construction so the
// IR type of a `/` expression reads as TyFloat downstream.
func promoteOp(op string, a, b *Type) *Type {
	t := promote(a, b)
	if op == "/" && t != nil && t.Kind == TyInt {
		return &Type{Kind: TyFloat}
	}
	if op == "*" && a != nil && b != nil {
		if (a.Kind == TyStr && b.Kind == TyInt) || (a.Kind == TyInt && b.Kind == TyStr) {
			return &Type{Kind: TyStr}
		}
		if a.Kind == TyList && b.Kind == TyInt {
			return &Type{Kind: TyList, Elem: a.Elem}
		}
		if a.Kind == TyInt && b.Kind == TyList {
			return &Type{Kind: TyList, Elem: b.Elem}
		}
	}
	if op == "|" && a != nil && b != nil && a.Kind == TyDict && b.Kind == TyDict {
		return &Type{Kind: TyDict, Key: a.Key, Val: a.Val}
	}
	// Set operations on lists (gopy lowers sets to slices): preserve the
	// element type so downstream uses (sorted, in, etc.) see a typed list.
	if a != nil && b != nil && a.Kind == TyList && b.Kind == TyList {
		switch op {
		case "&", "|", "-", "^":
			return &Type{Kind: TyList, Elem: a.Elem}
		}
	}
	if op == "%" && a != nil && a.Kind == TyStr {
		return &Type{Kind: TyStr}
	}
	return t
}
