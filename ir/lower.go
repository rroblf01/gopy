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
			return nil, err
		}
		if decls == nil {
			continue
		}
		m.Decls = append(m.Decls, decls...)
	}
	return m, nil
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
	case "FunctionDef":
		f, err := lowerFunc(n)
		if err != nil {
			return nil, err
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

func moduleScopeDeclare(name string, ty *Type) {
	moduleScopeVars[name] = ty
}

func moduleScopeLookup(name string) (*Type, bool) {
	t, ok := moduleScopeVars[name]
	return t, ok
}

func moduleScopeReset() {
	moduleScopeVars = map[string]*Type{}
}

// lowerClass emits a Class decl plus one Func per method (with Receiver set).
// __init__ becomes the constructor body; its self-attribute assignments are
// scanned to derive struct fields.
func lowerClass(n parser.Node) ([]Decl, error) {
	name := n.Str("name")
	class := &Class{Name: name}
	// First pass: collect all base names (including any registered plugin
	// markers like `Model`). Plugin bases are consumed by the lower hook
	// — they don't appear as Go-level embeds.
	var rawBases []string
	for _, b := range n.Children("bases") {
		if b.Type() != "Name" {
			return nil, fmt.Errorf("class %s: complex base expressions not supported", name)
		}
		bn := b.Str("id")
		if bn == "object" {
			continue
		}
		rawBases = append(rawBases, bn)
	}
	// Enum detection: collapse `class Color(Enum):` into a typed
	// integer-aliased constant set. The base name is consumed, not
	// embedded.
	for i, b := range rawBases {
		if b == "Enum" || b == "IntEnum" {
			class.IsEnum = true
			rawBases = append(rawBases[:i], rawBases[i+1:]...)
			break
		}
	}
	class.Bases = append(class.Bases, rawBases...)
	if class.IsEnum {
		for _, m := range n.Children("body") {
			if m.Type() != "Assign" {
				return nil, fmt.Errorf("enum %s: body must contain only `NAME = int` declarations", name)
			}
			tgts := m.Children("targets")
			if len(tgts) != 1 || tgts[0].Type() != "Name" {
				return nil, fmt.Errorf("enum %s: each member needs a single Name target", name)
			}
			val := m.Child("value")
			if val.Type() != "Constant" {
				return nil, fmt.Errorf("enum %s.%s: value must be an integer literal", name, tgts[0].Str("id"))
			}
			fv, ok := val["value"].(float64)
			if !ok {
				return nil, fmt.Errorf("enum %s.%s: value must be an integer literal", name, tgts[0].Str("id"))
			}
			class.EnumMembers = append(class.EnumMembers, EnumMember{Name: tgts[0].Str("id"), Value: int64(fv)})
		}
		return []Decl{class}, nil
	}
	// @dataclass / @dataclasses.dataclass synthesizes __init__ from
	// class-level annotated fields. Detect the decorator(s) here so the
	// body walk can pick up the AnnAssign nodes that would otherwise
	// trip the "only methods supported" rule.
	isDataclass := false
	for _, d := range n.Children("decorator_list") {
		if d.Type() == "Name" && d.Str("id") == "dataclass" {
			isDataclass = true
		}
		if d.Type() == "Attribute" {
			recv := d.Child("value")
			if recv != nil && recv.Type() == "Name" && recv.Str("id") == "dataclasses" && d.Str("attr") == "dataclass" {
				isDataclass = true
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
					dv, err := lowerExpr(def, dcSc)
					if err != nil {
						return nil, err
					}
					p.Default = dv
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

	for _, m := range bodyNodes {
		if dataclassDone && m.Type() == "AnnAssign" {
			continue // already consumed above
		}
		if m.Type() != "FunctionDef" {
			return nil, fmt.Errorf("line %d: class %s: only methods supported (F2)", m.Lineno(), name)
		}
		// Accepted method decorators:
		//   @property     — call sites emit `.attr()`
		//   @classmethod  — lowered to a free `<Class>_<method>` Go function
		//                   so it doesn't need a `*Class` receiver
		isProperty := false
		isClassMethod := false
		for _, d := range m.Children("decorator_list") {
			if d.Type() == "Name" {
				switch d.Str("id") {
				case "property":
					isProperty = true
					continue
				case "classmethod":
					isClassMethod = true
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
		args := m.Child("args").Children("args")
		if len(args) == 0 {
			return nil, fmt.Errorf("class %s.%s: method must have at least self parameter", name, methName)
		}
		// Drop self; we add it back as Receiver.
		params := args[1:]

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
			class.InitBody = body
			// Scan body for `self.x = expr` to derive fields (preserve order).
			seen := map[string]bool{}
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
				class.Fields = append(class.Fields, Param{Name: aa.Name, Ty: aa.Value.TypeOf()})
			}
			continue
		}

		// Regular method (or @classmethod). For @classmethod we emit a
		// free function `<Class>_<method>` so it doesn't take a *Class
		// receiver; calls like `Class.method(args)` are rewritten at
		// codegen time. References to `cls` inside the body are rewritten
		// to the class name, matching Python's semantics for `cls(...)`.
		fn := &Func{Name: methName}
		if !isClassMethod {
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
				fn.Vararg = &Param{Name: va.Str("arg"), Ty: &Type{Kind: TyList, Elem: &Type{Kind: TyAny}}}
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
		} else {
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
		if isClassMethod {
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
	}

	// Class first, then methods, so codegen order is type → methods.
	return append([]Decl{class}, decls...), nil
}

func lowerFunc(n parser.Node) (*Func, error) {
	// Accepted decorators (silently ignored — they don't change Go
	// codegen): `@staticmethod` matches a free function with no self;
	// `@functools.lru_cache` / `@lru_cache` / `@lru_cache(...)` is a
	// performance hint we currently ignore (the function still runs,
	// uncached). Anything else is rejected so wrong-looking output
	// doesn't slip through quietly.
	for _, d := range n.Children("decorator_list") {
		if d.Type() == "Name" {
			switch d.Str("id") {
			case "staticmethod", "lru_cache", "cache", "cached_property":
				continue
			}
		}
		if d.Type() == "Attribute" {
			recv := d.Child("value")
			if recv != nil && recv.Type() == "Name" && recv.Str("id") == "functools" {
				switch d.Str("attr") {
				case "lru_cache", "cache", "cached_property":
					continue
				}
			}
		}
		if d.Type() == "Call" {
			fn := d.Child("func")
			if fn != nil {
				if fn.Type() == "Name" {
					switch fn.Str("id") {
					case "lru_cache", "cache":
						continue
					}
				}
				if fn.Type() == "Attribute" {
					recv := fn.Child("value")
					if recv != nil && recv.Type() == "Name" && recv.Str("id") == "functools" {
						switch fn.Str("attr") {
						case "lru_cache", "cache":
							continue
						}
					}
				}
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
	f := &Func{Name: n.Str("name")}
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
	// *args (single python `arg` node under args.vararg) becomes a []any slice.
	if va := args.Child("vararg"); va != nil {
		f.Vararg = &Param{Name: va.Str("arg"), Ty: &Type{Kind: TyList, Elem: &Type{Kind: TyAny}}}
	}
	// **kwargs (single python `arg` node under args.kwarg) becomes map[string]any.
	if kw := args.Child("kwarg"); kw != nil {
		f.Kwarg = &Param{Name: kw.Str("arg"), Ty: &Type{Kind: TyDict, Key: &Type{Kind: TyStr}, Val: &Type{Kind: TyAny}}}
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
	return f, nil
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
		case "str":
			return &Type{Kind: TyStr}, nil
		case "bool":
			return &Type{Kind: TyBool}, nil
		case "None":
			return &Type{Kind: TyNone}, nil
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
		// `int | str` style unions desugar to `any`. We don't track the
		// alternative types separately — every operand winds up boxed
		// when passed through, and explicit `isinstance` covers the
		// reverse direction.
		op := n.Child("op")
		if op != nil && op.Type() == "BitOr" {
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
			// typing.Optional[T] — lowered to `any`. The wrapped type
			// is recorded as the elem in case a future pass wants to
			// narrow it, but for now we accept None alongside T values.
			return &Type{Kind: TyAny}, nil
		case "Union":
			// typing.Union[...] — same lowering as the `|` operator
			// form: collapse to any. Components are not tracked.
			return &Type{Kind: TyAny}, nil
		case "List":
			elem, err := lowerAnnotation(n.Child("slice"))
			if err != nil {
				return nil, err
			}
			return &Type{Kind: TyList, Elem: elem}, nil
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
		case "dict":
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

func lowerStmt(n parser.Node, sc *scope) (Stmt, error) {
	switch n.Type() {
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
			obj, err := lowerExpr(tgt.Child("value"), sc)
			if err != nil {
				return nil, err
			}
			idx, err := lowerExpr(tgt.Child("slice"), sc)
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
			bin := &BinOp{Op: op, L: lhs, R: rhs, Ty: promoteOp(op, lhs.Ty, rhs.TypeOf())}
			return &Assign{Target: name, Value: bin, Decl: false}, nil
		case "Attribute":
			recv, err := lowerExpr(tgt.Child("value"), sc)
			if err != nil {
				return nil, err
			}
			attrName := tgt.Str("attr")
			lhs := &Attribute{Recv: recv, Name: attrName, Ty: rhs.TypeOf()}
			bin := &BinOp{Op: op, L: lhs, R: rhs, Ty: promoteOp(op, lhs.Ty, rhs.TypeOf())}
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
			bin := &BinOp{Op: op, L: lhs, R: rhs, Ty: promoteOp(op, elemTy, rhs.TypeOf())}
			return &AssignSub{Target: obj, Index: idx, Value: bin}, nil
		default:
			return nil, fmt.Errorf("line %d: unsupported AugAssign target %q", n.Lineno(), tgt.Type())
		}
	case "AnnAssign":
		tgt := n.Child("target")
		if tgt.Type() != "Name" {
			return nil, fmt.Errorf("line %d: only Name annotated-assignment supported (F1)", n.Lineno())
		}
		ty, err := lowerAnnotation(n.Child("annotation"))
		if err != nil {
			return nil, err
		}
		val, err := lowerExpr(n.Child("value"), sc)
		if err != nil {
			return nil, err
		}
		name := tgt.Str("id")
		decl := sc.declare(name, ty)
		return &Assign{Target: name, Ty: ty, Value: val, Decl: decl}, nil
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
		if len(n.Children("orelse")) > 0 {
			return nil, fmt.Errorf("line %d: while-else not supported", n.Lineno())
		}
		// Same walrus hoist as `if`. Note: the assigned name is
		// re-bound every iteration because the hoisted Assign sits
		// before the loop header in the IR — we replicate it back
		// into the body's tail to maintain Python semantics.
		pre, condExpr := hoistNamedExprs(cond)
		whileStmt := &While{Cond: condExpr, Body: body}
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
	case "For":
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
				if t.Type() != "Name" {
					return nil, fmt.Errorf("line %d: complex except type not supported (F3)", h.Lineno())
				}
				eh.ClassName = t.Str("id")
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
		if len(n.Children("orelse")) > 0 {
			return nil, fmt.Errorf("line %d: try-else not supported (F3)", n.Lineno())
		}
		return &Try{Body: body, Handlers: handlers, Finally: finally}, nil
	case "Raise":
		excNode := n.Child("exc")
		if excNode == nil {
			return &Raise{Exc: nil}, nil
		}
		exc, err := lowerExpr(excNode, sc)
		if err != nil {
			return nil, err
		}
		return &Raise{Exc: exc}, nil
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
	case "With":
		return lowerWith(n, sc)
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
		return &BinOp{Op: op, L: l, R: r, Ty: promoteOp(op, l.TypeOf(), r.TypeOf())}, nil
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
			acc = &BoolOp{Op: op, L: acc, R: r, Ty: &Type{Kind: TyBool}}
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
					}
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
	case "DictComp":
		return lowerDictComp(n, sc)
	case "Starred":
		inner, err := lowerExpr(n.Child("value"), sc)
		if err != nil {
			return nil, err
		}
		return &Starred{Value: inner, Ty: inner.TypeOf()}, nil
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
				if fs := p.Child("format_spec"); fs != nil && fs.Type() == "JoinedStr" {
					// format_spec is itself a JoinedStr; flatten its
					// literal pieces into a single spec string. Nested
					// expressions inside the spec are uncommon — skip.
					for _, sp := range fs.Children("values") {
						if sp.Type() == "Constant" {
							s, _ := sp["value"].(string)
							spec += s
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
				parts = append(parts, FStrPart{Expr: x, Spec: spec, Conv: conv})
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
		case "upper", "lower", "strip", "replace":
			return &Type{Kind: TyStr}
		case "startswith", "endswith":
			return &Type{Kind: TyBool}
		case "find":
			return &Type{Kind: TyInt}
		case "split":
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
		if tgt.Type() != "Name" {
			return nil, fmt.Errorf("line %d: comprehension target must be a single name", n.Lineno())
		}
		iter, err := lowerExpr(g.Child("iter"), innerSc)
		if err != nil {
			return nil, err
		}
		varName := tgt.Str("id")
		var elemTy *Type
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
		innerSc.declare(varName, elemTy)
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
		gen := CompGen{Var: varName, Iter: iter, Cond: cond, ElemTy: elemTy}
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
		if tgt.Type() != "Name" {
			return nil, fmt.Errorf("line %d: comprehension target must be a single name", n.Lineno())
		}
		iter, err := lowerExpr(g.Child("iter"), innerSc)
		if err != nil {
			return nil, err
		}
		varName := tgt.Str("id")
		var elemTy *Type
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
		innerSc.declare(varName, elemTy)
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
		gen := CompGen{Var: varName, Iter: iter, Cond: cond, ElemTy: elemTy}
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
		patterns, err := lowerMatchPattern(pat, sc)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", caseNode.Lineno(), err)
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
		m.Cases = append(m.Cases, MatchCase{Patterns: patterns, Guard: guard, Body: body})
	}
	return m, nil
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

// lowerWith handles only `with open(path[, mode]) as name: body` in F4.
// Other context managers raise an explicit "unsupported" error so users see
// a precise failure mode rather than mysterious downstream Go errors.
func lowerWith(n parser.Node, sc *scope) (Stmt, error) {
	items := n.Children("items")
	if len(items) != 1 {
		return nil, fmt.Errorf("line %d: multi-item `with` not supported (F4)", n.Lineno())
	}
	item := items[0]
	ctx := item.Child("context_expr")
	if ctx.Type() != "Call" {
		return nil, fmt.Errorf("line %d: only `with open(...)` supported (F4)", n.Lineno())
	}
	fn := ctx.Child("func")
	if fn.Type() != "Name" || fn.Str("id") != "open" {
		return nil, fmt.Errorf("line %d: only `with open(...)` supported (F4)", n.Lineno())
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
	if len(n.Children("orelse")) > 0 {
		return nil, fmt.Errorf("line %d: for-else not supported", n.Lineno())
	}
	iter := n.Child("iter")
	// Two-name target: `for k, v in pairs`. Three patterns supported:
	//   `for k, v in d.items()`         — Go-native map range
	//   `for i, x in enumerate(xs)`     — Go-native slice index+value range
	//   `for a, b in zip(xs, ys)`       — paired iteration via parallel index
	if tgt.Type() == "Tuple" {
		elts := tgt.Children("elts")
		if len(elts) != 2 || elts[0].Type() != "Name" || elts[1].Type() != "Name" {
			return nil, fmt.Errorf("line %d: only two Name targets supported in for-loop unpacking", n.Lineno())
		}
		v1, v2 := elts[0].Str("id"), elts[1].Str("id")
		return lowerForTuple(n, sc, v1, v2, iter)
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
			return &ForRange{Var: varName, Start: start, Stop: stop, Step: step, Body: body}, nil
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
	return &ForEach{Var: varName, Iter: iterE, ElemTy: elemTy, Body: body}, nil
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
	return t
}
