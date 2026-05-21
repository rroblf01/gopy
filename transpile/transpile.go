// Package transpile lowers the IR into Go source text.
// It writes the code through go/format so the output is always gofmt-clean.
package transpile

import (
	"fmt"
	"go/format"
	"sort"
	"strconv"
	"strings"

	"github.com/rroblf01/gopy/ir"
)

// Options controls code generation.
type Options struct {
	PackageName string // defaults to "main"
	RuntimePath string // import path of the gopy runtime; "" disables import
}

// Module renders an IR module as gofmt-formatted Go source.
func Module(m *ir.Module, opt Options) ([]byte, error) {
	if opt.PackageName == "" {
		opt.PackageName = "main"
	}
	g := &gen{opt: opt, classes: map[string]*ir.Class{}, funcs: map[string]*ir.Func{}, methods: map[string]map[string]*ir.Func{}, helpers: map[string]string{}, fileVars: map[string]bool{}, generators: map[string]bool{}, aliases: map[string]string{}, varTypes: map[string]string{}, localVarTypes: map[string]*ir.Type{}, globals: map[string]*ir.Type{}}
	g.buildAliases(m)
	// First pass: register class names so call-site lowering can rewrite
	// `Foo(...)` → `NewFoo(...)`, and so method codegen can look up bases
	// for super() dispatch.
	for _, d := range m.Decls {
		switch x := d.(type) {
		case *ir.Class:
			g.classes[x.Name] = x
		case *ir.Func:
			if x.IsGenerator {
				g.generators[x.Name] = true
			}
			if x.Receiver == nil {
				g.funcs[x.Name] = x
			} else {
				cls := x.Receiver.Ty.Name
				if g.methods[cls] == nil {
					g.methods[cls] = map[string]*ir.Func{}
				}
				g.methods[cls][x.Name] = x
			}
		case *ir.Var:
			g.globals[x.Name] = x.Ty
		}
	}
	// Diamond-inheritance method conflict check. Go's embedding rules will
	// reject ambiguous selectors at compile time with a cryptic message
	// ("ambiguous selector"); surfacing the same condition here with the
	// Python-level class and method names is much friendlier.
	if err := g.detectDiamondConflicts(); err != nil {
		return nil, err
	}
	// Up-front scan for `Exception` usage. Anything we miss here gets
	// caught after codegen too — see the final pass below that promotes
	// the base type into the helpers map when any emitted helper
	// references NewException.
	g.detectExceptionUsage(m)
	for _, d := range m.Decls {
		switch x := d.(type) {
		case *ir.Func:
			// Skip emission for methods of interface-shaped classes —
			// the interface declaration already lists the signatures.
			if x.Receiver != nil {
				if cls, ok := g.classes[x.Receiver.Ty.Name]; ok && cls.IsInterface && len(cls.InterfaceMethods) > 0 && len(cls.Fields) == 0 && !cls.HasInit && len(cls.MethodNames) == len(cls.InterfaceMethods) {
					continue
				}
			}
			if err := g.fn(x); err != nil {
				return nil, err
			}
		case *ir.Class:
			if err := g.class(x); err != nil {
				return nil, err
			}
		case *ir.Var:
			if err := g.moduleVar(x); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("transpile: unsupported decl %T", d)
		}
		g.writef("\n")
	}
	// Post-codegen Exception promotion: any helper that calls
	// NewException pulls in the inline base type. This catches the
	// shims we don't explicitly track at scan time (deque, hashlib's
	// Get, ...).
	if !g.needsException {
		for _, src := range g.helpers {
			if strings.Contains(src, "NewException(") {
				g.needsException = true
				break
			}
		}
	}
	if g.needsException {
		g.helpers["__Exception"] = exceptionBaseSource
	}
	// Emit any inline runtime helpers (e.g. time.time shim) once at module end.
	for _, names := range sortedKeys(g.helpers) {
		g.writef("\n%s\n", g.helpers[names])
	}

	src := assembleSource(opt.PackageName, g.collectImports(), g.body.String())
	formatted, err := format.Source([]byte(src))
	if err != nil {
		return []byte(src), fmt.Errorf("gofmt: %w\n---\n%s", err, src)
	}
	return formatted, nil
}

// moduleVar emits a Go package-scope variable declaration from a Var Decl.
// Untyped vars (no annotation) get their type from the initializer; typed
// ones honor the annotation so untyped numeric literals don't drift to Go
// defaults that disagree with Python's int64 / float64 model.
func (g *gen) moduleVar(v *ir.Var) error {
	hasTy := v.Ty != nil && v.Ty.Kind != ir.TyUnknown && v.Ty.Kind != ir.TyNone
	switch {
	case hasTy:
		g.writef("var %s %s", v.Name, g.goType(v.Ty))
	default:
		g.writef("var %s", v.Name)
	}
	if v.Value != nil {
		g.writef(" = ")
		if err := g.expr(v.Value); err != nil {
			return err
		}
	}
	g.writef("\n")
	return nil
}

// buildAliases populates g.aliases from the module's import statements.
// Only `from <stdlib> import <name>` forms are recorded — bare `import X`
// already maps `X.attr` to stdlibModules[X] without help, and unknown
// modules surface as undefined-name errors downstream.
func (g *gen) buildAliases(m *ir.Module) {
	for _, imp := range m.Imports {
		if imp.From == "" {
			// `import X` or `import X as Y`. Record an alias when `as` was used.
			for _, n := range imp.Names {
				if n.Alias != "" {
					g.aliases[n.Alias] = n.Name
				}
			}
			continue
		}
		// Walk dotted import paths: `from urllib.parse import quote` needs
		// urllib → parse → quote. Resolve segment by segment through Subs.
		segs := splitDotted(imp.From)
		mod, ok := stdlibModules[segs[0]]
		if !ok {
			continue
		}
		for _, p := range segs[1:] {
			sub, ok := mod.Subs[p]
			if !ok {
				mod = stdlibModule{}
				break
			}
			mod = sub
		}
		for _, n := range imp.Names {
			local := n.Alias
			if local == "" {
				local = n.Name
			}
			if _, ok := mod.Subs[n.Name]; ok {
				g.aliases[local] = imp.From + "." + n.Name
				continue
			}
			if _, ok := mod.Funcs[n.Name]; ok {
				g.aliases[local] = imp.From + "." + n.Name
				continue
			}
		}
	}
}

// resolveAlias returns the canonical stdlib path for a local name, or the
// name unchanged if no alias is registered.
func (g *gen) resolveAlias(name string) string {
	if v, ok := g.aliases[name]; ok {
		return v
	}
	return name
}

// exceptionBaseSource is the inline Exception base emitted via the
// helpers map. Keeping it as a helper (rather than a hand-rolled top-
// of-file emit) lets late discovery during codegen still pull the type
// in, and matches the existing helper-import dedup semantics.
const exceptionBaseSource = `type Exception struct {
	Msg string
}

func NewException(msg string) *Exception { return &Exception{Msg: msg} }

func (e *Exception) Error() string { return e.Msg }

func (e *Exception) String() string { return e.Msg }`

// detectExceptionUsage walks the IR looking for any place where the bare
// builtin `Exception` is referenced (as a class base, an except clause, or
// as the callee of a raise).
func (g *gen) detectExceptionUsage(m *ir.Module) {
	for _, d := range m.Decls {
		switch x := d.(type) {
		case *ir.Class:
			for _, b := range x.Bases {
				if b == "Exception" {
					g.needsException = true
				}
			}
			g.scanStmtsForException(x.InitBody)
		case *ir.Func:
			g.scanStmtsForException(x.Body)
		}
	}
}

func (g *gen) scanStmtsForException(ss []ir.Stmt) {
	for _, s := range ss {
		switch x := s.(type) {
		case *ir.Try:
			for _, h := range x.Handlers {
				if h.ClassName == "Exception" {
					g.needsException = true
				}
			}
			g.scanStmtsForException(x.Body)
			for _, h := range x.Handlers {
				g.scanStmtsForException(h.Body)
			}
			g.scanStmtsForException(x.Finally)
		case *ir.Raise:
			if c, ok := x.Exc.(*ir.Call); ok {
				if n, ok := c.Func.(*ir.Name); ok && n.N == "Exception" {
					g.needsException = true
				}
			}
		case *ir.If:
			g.scanExprForException(x.Cond)
			g.scanStmtsForException(x.Then)
			g.scanStmtsForException(x.Else)
		case *ir.While:
			g.scanExprForException(x.Cond)
			g.scanStmtsForException(x.Body)
		case *ir.ForRange:
			g.scanStmtsForException(x.Body)
		case *ir.ForEach:
			g.scanStmtsForException(x.Body)
		case *ir.ExprStmt:
			g.scanExprForException(x.X)
		case *ir.Assign:
			g.scanExprForException(x.Value)
		case *ir.Return:
			g.scanExprForException(x.X)
		}
	}
}

// scanExprForException walks an expression tree looking for constructs
// that need the inline Exception base — currently the two-arg
// `getattr(obj, name)` form, which panics with `NewException(...)` when
// the attribute is missing.
func (g *gen) scanExprForException(e ir.Expr) {
	if e == nil {
		return
	}
	switch x := e.(type) {
	case *ir.Call:
		if n, ok := x.Func.(*ir.Name); ok {
			switch {
			case n.N == "getattr" && len(x.Args) == 2:
				g.needsException = true
			case n.N == "next" && len(x.Args) == 1:
				g.needsException = true
			}
		}
		g.scanExprForException(x.Func)
		for _, a := range x.Args {
			g.scanExprForException(a)
		}
		for _, kw := range x.Keywords {
			g.scanExprForException(kw.Value)
		}
	case *ir.MethodCall:
		g.scanExprForException(x.Recv)
		for _, a := range x.Args {
			g.scanExprForException(a)
		}
	case *ir.BinOp:
		g.scanExprForException(x.L)
		g.scanExprForException(x.R)
	case *ir.CmpOp:
		g.scanExprForException(x.L)
		g.scanExprForException(x.R)
	case *ir.BoolOp:
		g.scanExprForException(x.L)
		g.scanExprForException(x.R)
	case *ir.UnaryOp:
		g.scanExprForException(x.X)
	case *ir.Attribute:
		g.scanExprForException(x.Recv)
	case *ir.Subscript:
		g.scanExprForException(x.Value)
		g.scanExprForException(x.Index)
	case *ir.Slice:
		g.scanExprForException(x.Value)
		g.scanExprForException(x.Low)
		g.scanExprForException(x.High)
		g.scanExprForException(x.Step)
	case *ir.IfExpr:
		g.scanExprForException(x.Cond)
		g.scanExprForException(x.Then)
		g.scanExprForException(x.Else)
	case *ir.ListLit:
		for _, el := range x.Elems {
			g.scanExprForException(el)
		}
	case *ir.DictLit:
		for i := range x.Keys {
			g.scanExprForException(x.Keys[i])
			g.scanExprForException(x.Vals[i])
		}
	case *ir.FStr:
		for _, p := range x.Parts {
			if p.Expr != nil {
				g.scanExprForException(p.Expr)
			}
		}
	}
}

func assembleSource(pkg string, imports []string, body string) string {
	var b strings.Builder
	b.WriteString("package ")
	b.WriteString(pkg)
	b.WriteString("\n\n")
	if len(imports) > 0 {
		b.WriteString("import (\n")
		for _, imp := range imports {
			b.WriteString("\t")
			b.WriteString(strconv.Quote(imp))
			b.WriteString("\n")
		}
		b.WriteString(")\n\n")
	}
	b.WriteString(body)
	return b.String()
}

type gen struct {
	opt            Options
	body           strings.Builder
	imports        map[string]bool
	indent         int
	classes        map[string]*ir.Class            // class name → decl (for super() lookup)
	funcs          map[string]*ir.Func             // free function name → decl (for kwarg/default resolution)
	methods        map[string]map[string]*ir.Func  // class name → method name → method decl
	needsException bool                 // module references the builtin Exception type
	currentClass   *ir.Class            // set while emitting a method body, used for super()
	helpers        map[string]string    // inline runtime helpers emitted once at module end
	currentFn      *ir.Func             // current function being emitted (used for multi-return Return)
	fileVars       map[string]bool      // names currently bound to *os.File inside an active `with` block
	generators     map[string]bool      // function names that return a channel (Python generators)
	// aliases maps a local Python name (introduced by `from X import Y` or
	// `import X as Y`) to a dotted stdlib path the codegen knows about.
	// Example: `from datetime import datetime` → aliases["datetime"] = "datetime.datetime".
	aliases map[string]string
	// varTypes records the runtime tag of a local variable when it was
	// assigned the result of a typed stdlib call (Match, Path, Timedelta).
	// Codegen consults this for method dispatch (e.g. m.group()) and for
	// nil-safety on `if m:` truthy checks. Cleared between functions.
	varTypes map[string]string
	// localVarTypes carries IR types learned from typed assignments
	// inside the current function (e.g. `u = create_user()` where the
	// function's declared return type is `User`). Used by Name/Attribute
	// codegen to dispatch user-class methods when no annotation exists.
	localVarTypes map[string]*ir.Type
	// globals tracks module-level variable names (from `ir.Var` decls)
	// so writes inside function bodies emit `name = expr` (package-var
	// reassignment) rather than `name := expr` (which would shadow with
	// a local). Populated up-front from the IR.
	globals map[string]*ir.Type
}

func (g *gen) writef(f string, a ...any) { fmt.Fprintf(&g.body, f, a...) }

func (g *gen) writeIndent() {
	for i := 0; i < g.indent; i++ {
		g.body.WriteByte('\t')
	}
}

func (g *gen) addImport(path string) {
	if g.imports == nil {
		g.imports = map[string]bool{}
	}
	g.imports[path] = true
}

func (g *gen) collectImports() []string {
	out := make([]string, 0, len(g.imports))
	for k := range g.imports {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// sortedKeys returns the keys of m in lexical order.
func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (g *gen) class(c *ir.Class) error {
	prev := g.currentClass
	g.currentClass = c
	defer func() { g.currentClass = prev }()

	// Enum classes get a typed-int alias + one constant per declared
	// member. No struct, no methods. Member access through
	// `Color.RED` is rewritten elsewhere to `ColorRED`.
	if c.IsEnum {
		g.writef("type %s int64\n", c.Name)
		g.writef("const (\n")
		g.indent++
		for _, m := range c.EnumMembers {
			g.writeIndent()
			g.writef("%s%s %s = %d\n", c.Name, m.Name, c.Name, m.Value)
		}
		g.indent--
		g.writef(")\n")
		return nil
	}

	// Pure-abstract classes (inherit from ABC + only @abstractmethod
	// methods + no fields + no __init__) emit as a Go interface. Any
	// subclass whose concrete method set covers InterfaceMethods will
	// structurally satisfy it, so a function annotated with the base
	// type can take any subclass instance.
	if c.IsInterface && len(c.InterfaceMethods) > 0 && len(c.Fields) == 0 && !c.HasInit && len(c.MethodNames) == len(c.InterfaceMethods) {
		g.writef("type %s interface {\n", c.Name)
		g.indent++
		for _, m := range c.InterfaceMethods {
			g.writeIndent()
			g.writef("%s(", m.Name)
			for i, p := range m.Params {
				if i > 0 {
					g.writef(", ")
				}
				g.writef("%s %s", p.Name, g.goType(p.Ty))
			}
			g.writef(")")
			if m.Ret != nil && m.Ret.Kind != ir.TyNone && m.Ret.Kind != ir.TyUnknown {
				g.writef(" %s", g.goType(m.Ret))
			}
			g.writef("\n")
		}
		g.indent--
		g.writef("}\n\n")
		return nil
	}

	// Emit struct type.
	g.writef("type %s struct {\n", c.Name)
	g.indent++
	// Inheritance: embed *Base as an anonymous field. Field name is the base
	// type name, so attribute access on inherited fields works transparently.
	// Interface-shaped bases aren't embedded — the subclass's method set
	// satisfies the interface structurally.
	for _, b := range c.Bases {
		if base, ok := g.classes[b]; ok && base.IsInterface && len(base.InterfaceMethods) > 0 && len(base.Fields) == 0 && !base.HasInit && len(base.MethodNames) == len(base.InterfaceMethods) {
			continue
		}
		g.writeIndent()
		g.writef("*%s\n", b)
	}
	for _, f := range c.Fields {
		g.writeIndent()
		g.writef("%s %s\n", f.Name, g.goType(f.Ty))
	}
	g.indent--
	g.writef("}\n\n")

	// Per-class accessor helpers used by setattr / getattr / hasattr.
	// Emitting these unconditionally is cheap and keeps the dynamic
	// builtins type-safe without falling back to runtime reflection.
	if err := g.emitClassAccessors(c); err != nil {
		return err
	}

	// Emit constructor: New<Class>(args...) *<Class>
	g.writef("func New%s(", c.Name)
	for i, p := range c.InitArgs {
		if i > 0 {
			g.writef(", ")
		}
		g.writef("%s %s", p.Name, g.goType(p.Ty))
	}
	g.writef(") *%s {\n", c.Name)
	g.indent++
	g.writeIndent()
	g.writef("self := &%s{}\n", c.Name)
	// Zero-initialize every embedded base. This keeps stateless mixin bases
	// usable without an explicit `super().__init__()` call. The user's
	// `super().__init__(args)` (which targets Bases[0]) will reassign the
	// primary base later, overriding this stub. Interface-typed bases are
	// skipped (no embed, no init).
	for _, b := range c.Bases {
		if base, ok := g.classes[b]; ok && base.IsInterface && len(base.InterfaceMethods) > 0 && len(base.Fields) == 0 && !base.HasInit && len(base.MethodNames) == len(base.InterfaceMethods) {
			continue
		}
		g.writeIndent()
		g.writef("self.%s = &%s{}\n", b, b)
	}
	if err := g.stmts(c.InitBody); err != nil {
		return err
	}
	g.writeIndent()
	g.writef("return self\n")
	g.indent--
	g.writef("}\n")
	return nil
}

func (g *gen) fn(fn *ir.Func) error {
	if fn.Receiver != nil {
		// Track the enclosing class so super() inside the body resolves.
		prev := g.currentClass
		g.currentClass = g.classes[fn.Receiver.Ty.Name]
		defer func() { g.currentClass = prev }()
	}
	// Var-type tracking is function-local: a name reused across two
	// functions could plausibly hold different stdlib types in each, so
	// we start fresh and restore on exit.
	prevFn := g.currentFn
	g.currentFn = fn
	prevVars := g.varTypes
	g.varTypes = map[string]string{}
	prevLocal := g.localVarTypes
	g.localVarTypes = map[string]*ir.Type{}
	// Seed function parameters so attribute access on them dispatches
	// against the right class without needing assignment-side inference.
	for _, p := range fn.Params {
		if p.Ty != nil {
			g.localVarTypes[p.Name] = p.Ty
		}
	}
	if fn.Receiver != nil {
		g.localVarTypes[fn.Receiver.Name] = fn.Receiver.Ty
	}
	defer func() {
		g.varTypes = prevVars
		g.localVarTypes = prevLocal
		g.currentFn = prevFn
	}()
	if fn.IsGenerator {
		return g.generatorFn(fn)
	}
	g.writef("func ")
	if fn.Receiver != nil {
		g.writef("(%s *%s) ", fn.Receiver.Name, fn.Receiver.Ty.Name)
	}
	// Python dunder methods that map onto Go interfaces — rename so the
	// generated method matches what fmt / sort / etc. dispatch through.
	methodName := fn.Name
	if fn.Receiver != nil {
		switch fn.Name {
		case "__str__":
			methodName = "String"
		case "__repr__":
			methodName = "Repr"
		case "__len__":
			methodName = "Len"
		case "__hash__":
			methodName = "Hash"
		default:
			if mapped := exportedDunder(fn.Name); mapped != fn.Name {
				methodName = mapped
			}
		}
	}
	g.writef("%s", methodName)
	if len(fn.TypeParams) > 0 && fn.Receiver == nil {
		// Go generics: `func name[T any, U any](...)`. Only free
		// functions can be generic — Go methods can't introduce new
		// type parameters separately from their receiver.
		g.writef("[")
		for i, tp := range fn.TypeParams {
			if i > 0 {
				g.writef(", ")
			}
			g.writef("%s any", tp)
		}
		g.writef("]")
	}
	g.writef("(")
	for i, p := range fn.Params {
		if i > 0 {
			g.writef(", ")
		}
		g.writef("%s %s", p.Name, g.goType(p.Ty))
	}
	if fn.Vararg != nil {
		if len(fn.Params) > 0 {
			g.writef(", ")
		}
		g.writef("%s []any", fn.Vararg.Name)
	}
	if fn.Kwarg != nil {
		if len(fn.Params) > 0 || fn.Vararg != nil {
			g.writef(", ")
		}
		g.writef("%s map[string]any", fn.Kwarg.Name)
	}
	g.writef(")")
	if fn.Ret != nil && fn.Ret.Kind == ir.TyTuple {
		g.writef(" (")
		for i, t := range fn.Ret.Tuple {
			if i > 0 {
				g.writef(", ")
			}
			g.writef("%s", g.goType(t))
		}
		g.writef(")")
	} else if fn.Ret != nil && fn.Ret.Kind != ir.TyUnknown && fn.Ret.Kind != ir.TyNone {
		g.writef(" %s", g.goType(fn.Ret))
	}
	g.writef(" {\n")
	g.indent++
	// `_ = args / _ = kwargs` keeps unused captures from breaking the build.
	if fn.Vararg != nil {
		g.writeIndent()
		g.writef("_ = %s\n", fn.Vararg.Name)
	}
	if fn.Kwarg != nil {
		g.writeIndent()
		g.writef("_ = %s\n", fn.Kwarg.Name)
	}
	if err := g.stmts(fn.Body); err != nil {
		return err
	}
	// Abstract / stub bodies lowered to nothing still need to satisfy Go's
	// "missing return" rule when the signature has a non-void return.
	// Emit a panic so the method exists at runtime but loudly refuses to
	// be called — matches Python's "abstract method not implemented" model.
	if len(fn.Body) == 0 && fn.Ret != nil && fn.Ret.Kind != ir.TyNone {
		g.needsException = true
		g.writeIndent()
		g.writef("%s", "panic(NewException(\"NotImplementedError: abstract method "+fn.Name+"\"))\n")
	}
	g.indent--
	g.writef("}\n")
	return nil
}

// generatorFn lowers a Python generator (function with yield) to a Go
// function that returns a receive-only channel. The body runs in a
// goroutine; each `yield X` becomes `__yield <- X`; the channel closes
// when the goroutine returns.
//
// Limitations: no `send`/`throw`, no `return value` from generator
// (return without value ends the goroutine).
func (g *gen) generatorFn(fn *ir.Func) error {
	if fn.Receiver != nil {
		return fmt.Errorf("generator methods not supported (F4)")
	}
	elem := g.goType(fn.YieldType)
	if elem == "" || elem == "any" {
		elem = "any"
	}
	g.writef("func %s(", fn.Name)
	for i, p := range fn.Params {
		if i > 0 {
			g.writef(", ")
		}
		g.writef("%s %s", p.Name, g.goType(p.Ty))
	}
	g.writef(") <-chan %s {\n", elem)
	g.indent++
	g.writeIndent()
	g.writef("__yield := make(chan %s)\n", elem)
	g.writeIndent()
	g.writef("go func() {\n")
	g.indent++
	g.writeIndent()
	g.writef("defer close(__yield)\n")
	if err := g.stmts(fn.Body); err != nil {
		return err
	}
	g.indent--
	g.writeIndent()
	g.writef("}()\n")
	g.writeIndent()
	g.writef("return __yield\n")
	g.indent--
	g.writef("}\n")
	return nil
}

func (g *gen) stmts(ss []ir.Stmt) error {
	for _, s := range ss {
		if err := g.stmt(s); err != nil {
			return err
		}
	}
	return nil
}

func (g *gen) stmt(s ir.Stmt) error {
	switch x := s.(type) {
	case *ir.ExprStmt:
		g.writeIndent()
		if err := g.expr(x.X); err != nil {
			return err
		}
		g.writef("\n")
		return nil
	case *ir.Assign:
		// `defaultdict(factory)` at the RHS of a typed assignment: the
		// factory is ignored and we emit a plain empty map of the
		// declared (K, V). Untyped assignments can't infer K, so they
		// fall through to the default codegen and error there.
		if x.Decl && x.Ty != nil && x.Ty.Kind == ir.TyDict {
			if call, ok := x.Value.(*ir.Call); ok {
				if n, ok := call.Func.(*ir.Name); ok {
					if path, hit := g.aliases[n.N]; hit && (path == "collections.defaultdict" || path == "collections.OrderedDict") {
						g.writeIndent()
						g.writef("var %s %s = %s{}\n", x.Target, g.goType(x.Ty), g.goType(x.Ty))
						g.localVarTypes[x.Target] = x.Ty
						return nil
					}
				}
			}
		}
		// Augmented list-concat: `lst += other` lowered to `lst = lst + other`
		// where both sides have TyList. Python extends in place; Go has no
		// `+` for slices, so rewrite as `lst = append(lst, other...)`.
		if !x.Decl {
			if bin, ok := x.Value.(*ir.BinOp); ok && bin.Op == "+" {
				lt, rt := bin.L.TypeOf(), bin.R.TypeOf()
				if lt != nil && rt != nil && lt.Kind == ir.TyList && rt.Kind == ir.TyList {
					g.writeIndent()
					g.writef("%s = append(%s, ", x.Target, x.Target)
					if err := g.expr(bin.R); err != nil {
						return err
					}
					g.writef("...)\n")
					return nil
				}
			}
			// `d |= other` (Python 3.9+ dict merge) — rewrite as a key-by-key
			// copy, since Go has no `|` over maps.
			if bin, ok := x.Value.(*ir.BinOp); ok && bin.Op == "|" {
				lt, rt := bin.L.TypeOf(), bin.R.TypeOf()
				if lt != nil && rt != nil && lt.Kind == ir.TyDict && rt.Kind == ir.TyDict {
					g.writeIndent()
					g.writef("for __k, __v := range ")
					if err := g.expr(bin.R); err != nil {
						return err
					}
					g.writef(" { %s[__k] = __v }\n", x.Target)
					return nil
				}
			}
		}
		// Track stdlib-call return tags so later method dispatch and
		// truthy checks see the right type. We do this regardless of
		// whether the declaration carries an explicit annotation.
		if tag := g.exprTag(x.Value); tag != "" {
			g.varTypes[x.Target] = tag
		}
		// Propagate types so later attribute / method dispatch resolves
		// without requiring an explicit annotation. Sources, in order:
		//   1. The annotation on this Assign, if any.
		//   2. The return type of a user-defined function/method call.
		//   3. The static IR type the value itself carries.
		switch {
		case x.Ty != nil && x.Ty.Kind != ir.TyUnknown:
			g.localVarTypes[x.Target] = x.Ty
		case g.userCallRetType(x.Value) != nil:
			g.localVarTypes[x.Target] = g.userCallRetType(x.Value)
		default:
			if vt := x.Value.TypeOf(); vt != nil && vt.Kind != ir.TyUnknown {
				g.localVarTypes[x.Target] = vt
			}
		}
		g.writeIndent()
		// Writes targeting a module-level (package-scope) var must use
		// `=`, never `:=`, otherwise the function body would shadow the
		// global with a new local. The IR doesn't carry `global` info,
		// so we override here based on the registered globals.
		isGlobal := g.globals[x.Target] != nil
		switch {
		case isGlobal:
			g.writef("%s = ", x.Target)
		case x.Decl && x.Ty != nil && x.Ty.Kind != ir.TyUnknown:
			g.writef("var %s %s = ", x.Target, g.goType(x.Ty))
		case x.Decl && g.varTypes[x.Target] != "":
			// Tagged var (stdlib return): let Go infer the pointer type from RHS.
			g.writef("%s := ", x.Target)
		case x.Decl:
			g.writef("%s := ", x.Target)
		default:
			g.writef("%s = ", x.Target)
		}
		// Empty list/dict literals on the RHS need the target's typed
		// shape; otherwise the IR's TyAny element type would produce
		// `[]any{}` which won't assign to `[]string`.
		if x.Ty != nil && x.Ty.Kind == ir.TyList {
			if ll, ok := x.Value.(*ir.ListLit); ok && len(ll.Elems) == 0 {
				g.writef("%s{}\n", g.goType(x.Ty))
				return nil
			}
			// Non-empty list literal with a heterogeneous element type
			// (e.g. `list[Shape]` holding `Square` / `Circle`): emit with
			// the declared element type so Go's structural typing converts
			// each entry into the interface value implicitly.
			if ll, ok := x.Value.(*ir.ListLit); ok && len(ll.Elems) > 0 && x.Ty.Elem != nil {
				if cls, ok := g.classes[x.Ty.Elem.Name]; ok && cls.IsInterface && len(cls.InterfaceMethods) > 0 && len(cls.Fields) == 0 && !cls.HasInit && len(cls.MethodNames) == len(cls.InterfaceMethods) {
					g.writef("%s{", g.goType(x.Ty))
					for i, e := range ll.Elems {
						if i > 0 {
							g.writef(", ")
						}
						if err := g.expr(e); err != nil {
							return err
						}
					}
					g.writef("}\n")
					return nil
				}
			}
		}
		if x.Ty != nil && x.Ty.Kind == ir.TyDict {
			if dl, ok := x.Value.(*ir.DictLit); ok && len(dl.Keys) == 0 {
				g.writef("%s{}\n", g.goType(x.Ty))
				return nil
			}
		}
		if err := g.expr(x.Value); err != nil {
			return err
		}
		g.writef("\n")
		return nil
	case *ir.AssignSub:
		// User-class __setitem__: route `recv[k] = v` to `recv.Setitem(k, v)`.
		if tTy := g.effectiveType(x.Target); tTy != nil && tTy.Kind == ir.TyNamed {
			if fn := g.lookupMethod(tTy.Name, "__setitem__"); fn != nil {
				_ = fn
				g.writeIndent()
				if err := g.expr(x.Target); err != nil {
					return err
				}
				g.writef(".Setitem(")
				if err := g.expr(x.Index); err != nil {
					return err
				}
				g.writef(", ")
				if err := g.expr(x.Value); err != nil {
					return err
				}
				g.writef(")\n")
				return nil
			}
		}
		g.writeIndent()
		if err := g.expr(x.Target); err != nil {
			return err
		}
		g.writef("[")
		if err := g.expr(x.Index); err != nil {
			return err
		}
		g.writef("] = ")
		if err := g.expr(x.Value); err != nil {
			return err
		}
		g.writef("\n")
		return nil
	case *ir.AssignAttr:
		g.writeIndent()
		if err := g.expr(x.Target); err != nil {
			return err
		}
		g.writef(".%s = ", x.Name)
		// If the LHS field has a known concrete type (registered on the
		// class) and the RHS is an untyped empty literal, emit the typed
		// empty constructor so Go accepts the assignment.
		if fieldTy := g.attrFieldType(x.Target, x.Name); fieldTy != nil {
			if ll, ok := x.Value.(*ir.ListLit); ok && len(ll.Elems) == 0 && fieldTy.Kind == ir.TyList {
				g.writef("%s{}\n", g.goType(fieldTy))
				return nil
			}
			if dl, ok := x.Value.(*ir.DictLit); ok && len(dl.Keys) == 0 && fieldTy.Kind == ir.TyDict {
				g.writef("%s{}\n", g.goType(fieldTy))
				return nil
			}
		}
		if err := g.expr(x.Value); err != nil {
			return err
		}
		g.writef("\n")
		return nil
	case *ir.Return:
		g.writeIndent()
		if x.X == nil {
			g.writef("return\n")
			return nil
		}
		// Multi-return Return: enclosing func declared `tuple[T, U, ...]`
		// and the returned value is a list/tuple literal that matches.
		if g.currentFn != nil && g.currentFn.Ret != nil && g.currentFn.Ret.Kind == ir.TyTuple {
			if ll, ok := x.X.(*ir.ListLit); ok && len(ll.Elems) == len(g.currentFn.Ret.Tuple) {
				g.writef("return ")
				for i, el := range ll.Elems {
					if i > 0 {
						g.writef(", ")
					}
					if err := g.expr(el); err != nil {
						return err
					}
				}
				g.writef("\n")
				return nil
			}
		}
		g.writef("return ")
		if err := g.expr(x.X); err != nil {
			return err
		}
		g.writef("\n")
		return nil
	case *ir.If:
		// `if isinstance(name, Class):` — narrow `name` to *Class
		// inside the Then body by shadowing with a typed assertion.
		// Method / attribute access against the narrowed name dispatches
		// through the class's full method set.
		if narrow, ok := g.isinstanceNarrow(x.Cond); ok {
			g.writeIndent()
			g.writef("if %s, __isnok := any(%s).(%s); __isnok {\n", narrow.Var, narrow.Var, narrow.GoType)
			g.indent++
			g.writeIndent()
			g.writef("_ = %s\n", narrow.Var)
			prev, hadPrev := g.localVarTypes[narrow.Var]
			g.localVarTypes[narrow.Var] = narrow.Ty
			if err := g.stmts(x.Then); err != nil {
				return err
			}
			if hadPrev {
				g.localVarTypes[narrow.Var] = prev
			} else {
				delete(g.localVarTypes, narrow.Var)
			}
			g.indent--
			g.writeIndent()
			g.writef("}")
			if len(x.Else) > 0 {
				g.writef(" else {\n")
				g.indent++
				if err := g.stmts(x.Else); err != nil {
					return err
				}
				g.indent--
				g.writeIndent()
				g.writef("}")
			}
			g.writef("\n")
			return nil
		}
		g.writeIndent()
		g.writef("if ")
		if err := g.boolExpr(x.Cond); err != nil {
			return err
		}
		g.writef(" {\n")
		g.indent++
		if err := g.stmts(x.Then); err != nil {
			return err
		}
		g.indent--
		g.writeIndent()
		g.writef("}")
		if len(x.Else) > 0 {
			g.writef(" else {\n")
			g.indent++
			if err := g.stmts(x.Else); err != nil {
				return err
			}
			g.indent--
			g.writeIndent()
			g.writef("}")
		}
		g.writef("\n")
		return nil
	case *ir.While:
		g.writeIndent()
		g.writef("for ")
		if err := g.boolExpr(x.Cond); err != nil {
			return err
		}
		g.writef(" {\n")
		g.indent++
		if err := g.stmts(x.Body); err != nil {
			return err
		}
		g.indent--
		g.writeIndent()
		g.writef("}\n")
		return nil
	case *ir.ForRange:
		return g.forRange(x)
	case *ir.ForEach:
		return g.forEach(x)
	case *ir.Try:
		return g.try(x)
	case *ir.Raise:
		return g.raise(x)
	case *ir.WithFile:
		return g.withFile(x)
	case *ir.WithCM:
		return g.withCM(x)
	case *ir.Assert:
		return g.assertStmt(x)
	case *ir.Match:
		return g.matchStmt(x)
	case *ir.LocalFunc:
		return g.localFunc(x)
	case *ir.Break:
		g.writeIndent()
		g.writef("break\n")
		return nil
	case *ir.Continue:
		g.writeIndent()
		g.writef("continue\n")
		return nil
	case *ir.MultiAssign:
		g.writeIndent()
		for i, t := range x.Targets {
			if i > 0 {
				g.writef(", ")
			}
			g.writef("%s", t)
		}
		if x.Decl {
			g.writef(" := ")
		} else {
			g.writef(" = ")
		}
		// Multi-return single-call shorthand: `a, b = f()` where f
		// returns tuple[T, U]. Emitted as `a, b := f()` directly.
		if len(x.Values) == 1 {
			if call, ok := x.Values[0].(*ir.Call); ok {
				if t := g.userCallRetType(call); t != nil && t.Kind == ir.TyTuple && len(t.Tuple) == len(x.Targets) {
					if err := g.expr(call); err != nil {
						return err
					}
					g.writef("\n")
					return nil
				}
			}
		}
		for i, v := range x.Values {
			if i > 0 {
				g.writef(", ")
			}
			if err := g.expr(v); err != nil {
				return err
			}
		}
		g.writef("\n")
		return nil
	case *ir.YieldFrom:
		g.writeIndent()
		g.writef("for __v := range ")
		if err := g.expr(x.Iter); err != nil {
			return err
		}
		g.writef(" { __yield <- __v }\n")
		return nil
	case *ir.Yield:
		g.writeIndent()
		g.writef("__yield <- ")
		if x.X == nil {
			g.writef("0\n") // bare `yield` is rare; emit something compilable
			return nil
		}
		if err := g.expr(x.X); err != nil {
			return err
		}
		g.writef("\n")
		return nil
	}
	return fmt.Errorf("transpile: unsupported stmt %T", s)
}

// withFile emits the IIFE pattern for `with open(path, mode) as fh: body`.
// It opens the file, defers Close, and tags the file variable so method
// calls like fh.read() / fh.write(s) inside the body translate to the
// right Go expressions.
// helperCmpStr backs string-typed ORM field comparisons. ops:
//   eq (default), contains, startswith, endswith
// Numeric ops (lt/gt) on strings fall through to false to keep the
// dispatch table closed.
const helperCmpStr = `func __gopy_cmp_str(a string, expected any, op string) bool {
	e, _ := expected.(string)
	switch op {
	case "eq":
		return a == e
	case "contains":
		return strings.Contains(a, e)
	case "startswith":
		return strings.HasPrefix(a, e)
	case "endswith":
		return strings.HasSuffix(a, e)
	}
	return false
}`

// helperCmpInt backs integer-typed ORM field comparisons. Accepts the
// usual numeric operators in addition to plain equality.
const helperCmpInt = `func __gopy_cmp_int(a int64, expected any, op string) bool {
	var e int64
	switch x := expected.(type) {
	case int64:
		e = x
	case int:
		e = int64(x)
	case float64:
		e = int64(x)
	default:
		return false
	}
	switch op {
	case "eq":
		return a == e
	case "lt":
		return a < e
	case "lte":
		return a <= e
	case "gt":
		return a > e
	case "gte":
		return a >= e
	}
	return false
}`

// helperGopyID returns a stable integer derived from the value's
// pointer when possible, falling back to its string representation.
// CPython's id() returns the object's memory address; the gopy
// shim matches the spirit (different addresses ↔ different ids) but
// not specific values, so cross-runtime fixtures should only compare
// id-equality, never literal id values.
const helperGopyID = `func __gopy_id(v any) int64 {
	return int64(__gopy_hash(v))
}`

const helperGopyHash = `func __gopy_hash(v any) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(fmt.Sprintf("%v", v)))
	return int64(h.Sum64())
}`

const helperGopyHex = `func __gopy_hex(n int64) string {
	if n < 0 {
		return fmt.Sprintf("-0x%x", -n)
	}
	return fmt.Sprintf("0x%x", n)
}`

const helperGopyOct = `func __gopy_oct(n int64) string {
	if n < 0 {
		return fmt.Sprintf("-0o%o", -n)
	}
	return fmt.Sprintf("0o%o", n)
}`

const helperGopyBin = `func __gopy_bin(n int64) string {
	if n < 0 {
		return fmt.Sprintf("-0b%b", -n)
	}
	return fmt.Sprintf("0b%b", n)
}`

// helperGopyCallable mirrors Python's callable() for runtime values: a
// reflect.Func kind matches, everything else returns false. Classes
// (user-defined types) hit at compile time before this helper is reached.
const helperGopyCallable = `func __gopy_callable(v any) bool {
	if v == nil {
		return false
	}
	return reflect.TypeOf(v).Kind() == reflect.Func
}`

// helperGopyAscii mirrors Python's ascii(): wraps the string-ish repr in
// single quotes (or 'b"..."' for bytes), escaping non-ASCII as \uXXXX.
// Non-string inputs route through fmt.Sprintf("%#v") for a debug-style
// dump (Python uses repr() too).
const helperGopyAscii = `func __gopy_ascii(v any) string {
	s, ok := v.(string)
	if !ok {
		s = fmt.Sprintf("%#v", v)
		return s
	}
	var b []byte
	b = append(b, '\'')
	for _, r := range s {
		switch {
		case r == '\'':
			b = append(b, '\\', '\'')
		case r == '\\':
			b = append(b, '\\', '\\')
		case r == '\n':
			b = append(b, '\\', 'n')
		case r == '\t':
			b = append(b, '\\', 't')
		case r == '\r':
			b = append(b, '\\', 'r')
		case r >= 0x20 && r <= 0x7E:
			b = append(b, byte(r))
		case r < 0x100:
			b = append(b, []byte(fmt.Sprintf("\\x%02x", r))...)
		case r < 0x10000:
			b = append(b, []byte(fmt.Sprintf("\\u%04x", r))...)
		default:
			b = append(b, []byte(fmt.Sprintf("\\U%08x", r))...)
		}
	}
	b = append(b, '\'')
	return string(b)
}`

// helperStrMaketrans builds the rune→string mapping used by str.translate.
// 2-arg form: pair from[i] → to[i]. 3-arg form: chars in delete map to "".
const helperStrMaketrans = `func __gopy_str_maketrans(from, to string, del ...string) map[rune]string {
	out := map[rune]string{}
	fr := []rune(from)
	tr := []rune(to)
	for i := 0; i < len(fr); i++ {
		var rep string
		if i < len(tr) {
			rep = string(tr[i])
		}
		out[fr[i]] = rep
	}
	if len(del) > 0 {
		for _, r := range del[0] {
			out[r] = ""
		}
	}
	return out
}`

// helperStrTranslate applies the maketrans table to s. Runes missing from
// the table pass through unchanged; runes mapped to "" are dropped.
const helperStrTranslate = `func __gopy_str_translate(s string, table map[rune]string) string {
	var b strings.Builder
	for _, r := range s {
		if rep, ok := table[r]; ok {
			b.WriteString(rep)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}`

// helperGopyType returns the Python class name of v as a string. Covers
// primitives directly; for everything else we strip the package prefix
// off Go's %T formatting so user classes render as plain "Point" rather
// than "main.Point". Pointer types and slices/maps map to "list"/"dict"
// like CPython's repr-style names.
const helperGopyType = `type __Type struct {
	Name  string
	Qname string
}

func (t *__Type) String() string {
	if t.Qname != "" {
		return "<class '" + t.Qname + "'>"
	}
	return "<class '" + t.Name + "'>"
}

func __gopy_type(v any) *__Type {
	switch v.(type) {
	case nil:
		return &__Type{Name: "NoneType"}
	case bool:
		return &__Type{Name: "bool"}
	case int, int32, int64:
		return &__Type{Name: "int"}
	case float32, float64:
		return &__Type{Name: "float"}
	case string:
		return &__Type{Name: "str"}
	}
	s := fmt.Sprintf("%T", v)
	if len(s) >= 2 && s[0] == '[' && s[1] == ']' {
		return &__Type{Name: "list"}
	}
	if len(s) >= 4 && s[:4] == "map[" {
		return &__Type{Name: "dict"}
	}
	if len(s) >= 1 && s[0] == '*' {
		s = s[1:]
	}
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			short := s[i+1:]
			return &__Type{Name: short, Qname: "__main__." + short}
		}
	}
	return &__Type{Name: s}
}`

// builtinNext receives the next value off a generator's channel. Bare
// `next(it)` panics on exhaustion (mirroring Python's StopIteration);
// `next(it, default)` returns the default in that case.
const helperNoDefault = "panic(NewException(\"StopIteration\"))"

// helperGopyInt mirrors Python's int(x) for the common cases: numeric
// types are truncated to int64, strings are parsed as base-10, bools
// become 0/1. Used when the static type isn't known to be numeric
// (e.g. values pulled out of **kwargs or json.loads).
// helperGopyBool mirrors Python's truthiness rules for runtime-typed
// values: nil / 0 / "" / empty containers / false → false; everything
// else → true.
const helperGopyBool = `func __gopy_bool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case nil:
		return false
	case int64:
		return x != 0
	case int:
		return x != 0
	case float64:
		return x != 0
	case string:
		return len(x) > 0
	}
	return true
}`

const helperGopyInt = `func __gopy_int(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case bool:
		if x {
			return 1
		}
		return 0
	case string:
		n, err := strconv.ParseInt(x, 10, 64)
		if err != nil {
			panic(err)
		}
		return n
	}
	panic("int(): unsupported type")
}`

const helperGopyFloat = `func __gopy_float(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int64:
		return float64(x)
	case int:
		return float64(x)
	case string:
		n, err := strconv.ParseFloat(x, 64)
		if err != nil {
			panic(err)
		}
		return n
	}
	panic("float(): unsupported type")
}`

// helperPyPrint imitates Python's print(): bools render as "True"/"False",
// nil renders as "None", everything else falls through to fmt.Print.
// Items are space-separated and the line is newline-terminated.
const helperPyPrint = `func __gopy_print(sep string, end string, args ...any) {
	for i, a := range args {
		if i > 0 {
			fmt.Print(sep)
		}
		switch v := a.(type) {
		case bool:
			if v {
				fmt.Print("True")
			} else {
				fmt.Print("False")
			}
		case nil:
			fmt.Print("None")
			_ = v
		case float64:
			s := strconv.FormatFloat(v, 'g', -1, 64)
			// Python always renders whole-valued floats with a trailing
			// .0; Go's default omits it. Add the suffix when neither a
			// decimal point nor an exponent is present.
			has := false
			for j := 0; j < len(s); j++ {
				if s[j] == '.' || s[j] == 'e' || s[j] == 'E' {
					has = true
					break
				}
			}
			if !has {
				s += ".0"
			}
			fmt.Print(s)
		case complex128:
			// Match Python's complex repr: (real+imagj) with parens only
			// when the real part is nonzero. Pure-imaginary values print
			// without parens; uses 'j' (Python) rather than Go's 'i'.
			re, im := real(v), imag(v)
			fmtFloat := func(f float64) string {
				s := strconv.FormatFloat(f, 'g', -1, 64)
				ok := false
				for j := 0; j < len(s); j++ {
					if s[j] == '.' || s[j] == 'e' || s[j] == 'E' {
						ok = true
						break
					}
				}
				if !ok {
					s += ""
				}
				return s
			}
			if re == 0 {
				fmt.Print(fmtFloat(im) + "j")
			} else {
				sign := "+"
				if im < 0 {
					sign = "-"
					im = -im
				}
				fmt.Print("(" + fmtFloat(re) + sign + fmtFloat(im) + "j)")
			}
		default:
			fmt.Print(a)
		}
	}
	fmt.Print(end)
}`

const helperFileReadAll = `func __gopy_fh_read(f *os.File) string {
	b, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}
	return string(b)
}`

const helperFileWrite = `func __gopy_fh_write(f *os.File, s string) {
	if _, err := f.WriteString(s); err != nil {
		panic(err)
	}
}`

// withCM emits a user-class context manager: instantiates the ctx
// expression, calls .Enter() to bind the as-var, defers .Exit() so the
// teardown runs even on panic, then emits the body. Wrapping the whole
// block in an IIFE scopes the defer to the `with` block rather than the
// enclosing function.
// assertStmt emits `assert cond[, msg]`. The condition is run through
// the same truthiness helper used by `if`, then we panic with an
// AssertionError-tagged Exception so existing try/except wiring catches
// it. CPython disables asserts under `python -O`; gopy keeps them on
// since there's no equivalent compile-mode switch.
func (g *gen) assertStmt(a *ir.Assert) error {
	g.writeIndent()
	g.writef("if !(")
	if err := g.emitTruthy(a.Cond); err != nil {
		return err
	}
	g.writef(") {\n")
	g.indent++
	g.writeIndent()
	if a.Msg != nil {
		g.addImport("fmt")
		g.writef("panic(NewException(fmt.Sprintf(\"%%v\", ")
		if err := g.boxedExpr(a.Msg); err != nil {
			return err
		}
		g.writef(")))\n")
	} else {
		g.writef("panic(NewException(\"\"))\n")
	}
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	return nil
}

// emitTruthy emits a Go bool expression for a value, applying Python's
// truthiness rules: empty containers/strings → false, zero numbers →
// false, nil → false, anything else → true. Already-bool expressions
// pass through unchanged.
func (g *gen) emitTruthy(e ir.Expr) error {
	t := g.effectiveType(e)
	if t == nil {
		return g.boolExpr(e)
	}
	switch t.Kind {
	case ir.TyBool:
		return g.boolExpr(e)
	case ir.TyInt, ir.TyFloat:
		if err := g.expr(e); err != nil {
			return err
		}
		g.writef(" != 0")
		return nil
	case ir.TyStr, ir.TyList, ir.TyDict:
		g.writef("len(")
		if err := g.expr(e); err != nil {
			return err
		}
		g.writef(") > 0")
		return nil
	case ir.TyNamed:
		if fn := g.lookupMethod(t.Name, "__bool__"); fn != nil {
			_ = fn
			if err := g.expr(e); err != nil {
				return err
			}
			g.writef(".Bool()")
			return nil
		}
		if err := g.expr(e); err != nil {
			return err
		}
		g.writef(" != nil")
		return nil
	}
	return g.boolExpr(e)
}

func (g *gen) withCM(w *ir.WithCM) error {
	t := g.effectiveType(w.Ctx)
	if t == nil || t.Kind != ir.TyNamed {
		t = g.userCallRetType(w.Ctx)
	}
	if t == nil || t.Kind != ir.TyNamed {
		return fmt.Errorf("with: context expression has no resolvable class type")
	}
	enter := g.lookupMethod(t.Name, "__enter__")
	exit := g.lookupMethod(t.Name, "__exit__")
	if enter == nil || exit == nil {
		return fmt.Errorf("with: class %s must define both __enter__ and __exit__", t.Name)
	}
	g.writeIndent()
	g.writef("func() {\n")
	g.indent++
	g.writeIndent()
	g.writef("__cm := ")
	if err := g.expr(w.Ctx); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("defer __cm.Exit(nil, nil, nil)\n")
	if w.VarName != "" {
		g.writeIndent()
		g.writef("%s := __cm.Enter()\n", w.VarName)
		g.writeIndent()
		g.writef("_ = %s\n", w.VarName)
		if enter.Ret != nil && enter.Ret.Kind == ir.TyNamed {
			g.varTypes[w.VarName] = enter.Ret.Name
		}
	} else {
		g.writeIndent()
		g.writef("__cm.Enter()\n")
	}
	if err := g.stmts(w.Body); err != nil {
		return err
	}
	g.indent--
	g.writeIndent()
	g.writef("}()\n")
	return nil
}

func (g *gen) withFile(w *ir.WithFile) error {
	g.addImport("os")
	g.writeIndent()
	g.writef("func() {\n")
	g.indent++
	g.writeIndent()
	if w.Mode == "w" {
		g.writef("%s, __err := os.Create(", w.VarName)
	} else {
		g.writef("%s, __err := os.Open(", w.VarName)
	}
	if err := g.expr(w.Path); err != nil {
		return err
	}
	g.writef(")\n")
	g.writeIndent()
	g.writef("if __err != nil {\n")
	g.indent++
	g.writeIndent()
	g.writef("panic(__err)\n")
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("defer %s.Close()\n", w.VarName)

	// Mark the var as a file handle for the duration of the body so
	// method-call codegen can route fh.read() / fh.write() to helpers.
	prev := g.fileVars[w.VarName]
	g.fileVars[w.VarName] = true
	defer func() {
		if prev {
			g.fileVars[w.VarName] = true
		} else {
			delete(g.fileVars, w.VarName)
		}
	}()

	if err := g.stmts(w.Body); err != nil {
		return err
	}
	g.indent--
	g.writeIndent()
	g.writef("}()\n")
	return nil
}

// localFunc emits a nested function definition as a function-typed
// local: `name := func(p T) U { body }`. Closures over enclosing scope
// work because Go function literals capture surrounding names by
// reference, matching Python's late-binding semantics.
func (g *gen) localFunc(lf *ir.LocalFunc) error {
	fn := lf.Fn
	// Reset varTypes / localVarTypes inside the nested body so it
	// doesn't accidentally inherit shadowing assumptions; capture the
	// outer maps for restoration.
	prevVars := g.varTypes
	g.varTypes = map[string]string{}
	prevLocal := g.localVarTypes
	g.localVarTypes = map[string]*ir.Type{}
	for _, p := range fn.Params {
		if p.Ty != nil {
			g.localVarTypes[p.Name] = p.Ty
		}
	}
	defer func() {
		g.varTypes = prevVars
		g.localVarTypes = prevLocal
	}()

	g.writeIndent()
	g.writef("%s := func(", fn.Name)
	for i, p := range fn.Params {
		if i > 0 {
			g.writef(", ")
		}
		g.writef("%s %s", p.Name, g.goType(p.Ty))
	}
	if fn.Vararg != nil {
		if len(fn.Params) > 0 {
			g.writef(", ")
		}
		g.writef("%s []any", fn.Vararg.Name)
	}
	if fn.Kwarg != nil {
		if len(fn.Params) > 0 || fn.Vararg != nil {
			g.writef(", ")
		}
		g.writef("%s map[string]any", fn.Kwarg.Name)
	}
	g.writef(")")
	if fn.Ret != nil && fn.Ret.Kind != ir.TyUnknown && fn.Ret.Kind != ir.TyNone {
		g.writef(" %s", g.goType(fn.Ret))
	}
	g.writef(" {\n")
	g.indent++
	if fn.Vararg != nil {
		g.writeIndent()
		g.writef("_ = %s\n", fn.Vararg.Name)
	}
	if fn.Kwarg != nil {
		g.writeIndent()
		g.writef("_ = %s\n", fn.Kwarg.Name)
	}
	if err := g.stmts(fn.Body); err != nil {
		return err
	}
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	// Silence unused-warning when the closure is defined but never
	// referenced in the rest of the body (rare but lets the file
	// compile under the same `_ = ...` convention used elsewhere).
	g.writeIndent()
	g.writef("_ = %s\n", fn.Name)
	return nil
}

// matchStmt lowers Python's `match` to an if/else-if chain rather than
// a Go switch — switches can't combine guarded wildcards with an
// unconditional default, and exhaustive return analysis works equally
// well on a chained if. The subject is evaluated once into a local.
func (g *gen) matchStmt(m *ir.Match) error {
	g.writeIndent()
	g.writef("{\n")
	g.indent++
	g.writeIndent()
	g.writef("__subj := ")
	if err := g.expr(m.Subject); err != nil {
		return err
	}
	g.writef("\n")
	hadUnconditionalDefault := false
	for i, mc := range m.Cases {
		g.writeIndent()
		if i > 0 {
			g.writef("} else ")
		}
		if mc.MapPat != nil {
			mp := mc.MapPat
			g.writef("if ")
			if len(mp.Keys) == 0 {
				g.writef("true")
			}
			for j, k := range mp.Keys {
				if j > 0 {
					g.writef(" && ")
				}
				g.writef("func() bool { __mv, __mok := __subj[")
				if err := g.expr(k); err != nil {
					return err
				}
				g.writef("]; return __mok && __mv == ")
				if err := g.expr(mp.Values[j]); err != nil {
					return err
				}
				g.writef(" }()")
			}
			if mc.Guard != nil {
				g.writef(" && (")
				if err := g.boolExpr(mc.Guard); err != nil {
					return err
				}
				g.writef(")")
			}
			g.writef(" {\n")
		} else if mc.SeqPat != nil {
			sp := mc.SeqPat
			lenOp := "=="
			minLen := len(sp.Elements) + len(sp.Tail)
			if sp.HasStar {
				lenOp = ">="
			}
			g.writef("if len(__subj) %s %d", lenOp, minLen)
			for j, e := range sp.Elements {
				if e.LitVal != nil {
					g.writef(" && __subj[%d] == ", j)
					if err := g.expr(e.LitVal); err != nil {
						return err
					}
				}
			}
			for j, e := range sp.Tail {
				if e.LitVal != nil {
					g.writef(" && __subj[len(__subj)-%d] == ", len(sp.Tail)-j)
					if err := g.expr(e.LitVal); err != nil {
						return err
					}
				}
			}
			if mc.Guard != nil {
				g.writef(" && (")
				if err := g.boolExpr(mc.Guard); err != nil {
					return err
				}
				g.writef(")")
			}
			g.writef(" {\n")
			g.indent++
			for j, e := range sp.Elements {
				if e.Capture != "" && e.Capture != "_" {
					g.writeIndent()
					g.writef("%s := __subj[%d]\n", e.Capture, j)
					g.writeIndent()
					g.writef("_ = %s\n", e.Capture)
				}
			}
			if sp.HasStar && sp.Star != "_" {
				g.writeIndent()
				g.writef("%s := __subj[%d:len(__subj)-%d]\n", sp.Star, len(sp.Elements), len(sp.Tail))
				g.writeIndent()
				g.writef("_ = %s\n", sp.Star)
			}
			for j, e := range sp.Tail {
				if e.Capture != "" && e.Capture != "_" {
					g.writeIndent()
					g.writef("%s := __subj[len(__subj)-%d]\n", e.Capture, len(sp.Tail)-j)
					g.writeIndent()
					g.writef("_ = %s\n", e.Capture)
				}
			}
			g.indent--
		} else if mc.ClassPat != nil {
			// `case ClassName(field=value, ...)` — type-assert __subj
			// against the class pointer (or Go primitive for int/str/
			// float/bool), then check each named field.
			cp := mc.ClassPat
			primitive, goPrim := "", ""
			switch cp.ClassName {
			case "int":
				primitive, goPrim = "int", "int64"
			case "float":
				primitive, goPrim = "float", "float64"
			case "str":
				primitive, goPrim = "str", "string"
			case "bool":
				primitive, goPrim = "bool", "bool"
			}
			if primitive != "" {
				g.writef("if __cm, __cmok := any(__subj).(%s); __cmok", goPrim)
				if len(cp.KwdAttrs) > 0 {
					return fmt.Errorf("match class pattern: %s() takes no field patterns", primitive)
				}
			} else {
				g.writef("if __cm, __cmok := any(__subj).(*%s); __cmok", cp.ClassName)
				for j, attr := range cp.KwdAttrs {
					g.writef(" && __cm.%s == ", attr)
					if err := g.expr(cp.KwdValues[j]); err != nil {
						return err
					}
				}
			}
			if mc.Guard != nil {
				g.writef(" && (")
				if err := g.boolExpr(mc.Guard); err != nil {
					return err
				}
				g.writef(")")
			}
			g.writef(" {\n")
			g.indent++
			g.writeIndent()
			g.writef("_ = __cm\n")
			if mc.Capture != "" {
				g.writeIndent()
				g.writef("%s := __cm\n", mc.Capture)
				g.writeIndent()
				g.writef("_ = %s\n", mc.Capture)
			}
			g.indent--
		} else if mc.Capture != "" && len(mc.Patterns) == 0 && mc.Guard == nil {
			// `case name:` — bind name to subject, act as default arm.
			g.writef("{\n")
			hadUnconditionalDefault = true
			g.indent++
			g.writeIndent()
			g.writef("%s := __subj\n", mc.Capture)
			g.writeIndent()
			g.writef("_ = %s\n", mc.Capture)
			g.indent--
		} else if len(mc.Patterns) == 0 && mc.Guard == nil {
			// Bare wildcard — open an else with no condition.
			g.writef("{\n")
			hadUnconditionalDefault = true
		} else {
			g.writef("if ")
			needAnd := false
			if len(mc.Patterns) > 0 {
				g.writef("(")
				for j, p := range mc.Patterns {
					if j > 0 {
						g.writef(" || ")
					}
					g.writef("__subj == ")
					if err := g.expr(p); err != nil {
						return err
					}
				}
				g.writef(")")
				needAnd = true
			}
			if mc.Guard != nil {
				if needAnd {
					g.writef(" && (")
				} else {
					g.writef("(")
				}
				if err := g.boolExpr(mc.Guard); err != nil {
					return err
				}
				g.writef(")")
			}
			g.writef(" {\n")
		}
		g.indent++
		if err := g.stmts(mc.Body); err != nil {
			return err
		}
		g.indent--
	}
	g.writeIndent()
	g.writef("}\n")
	// Mute potential unused-variable warning if no case used __subj.
	if !hadUnconditionalDefault {
		g.writeIndent()
		g.writef("_ = __subj\n")
	}
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	return nil
}

// try emits a try/except/finally as an IIFE so the deferred recover()
// is lexically scoped to just the try block. Note: returning from inside
// the try body is NOT supported in F3 — it would only return from the IIFE.
func (g *gen) try(t *ir.Try) error {
	g.writeIndent()
	g.writef("func() {\n")
	g.indent++
	if len(t.Finally) > 0 {
		g.writeIndent()
		g.writef("defer func() {\n")
		g.indent++
		if err := g.stmts(t.Finally); err != nil {
			return err
		}
		g.indent--
		g.writeIndent()
		g.writef("}()\n")
	}
	if len(t.Handlers) > 0 {
		g.writeIndent()
		g.writef("defer func() {\n")
		g.indent++
		g.writeIndent()
		g.writef("r := recover()\n")
		g.writeIndent()
		g.writef("if r == nil {\n\t\treturn\n\t}\n")
		for _, h := range t.Handlers {
			if h.ClassName == "" {
				// bare except — catches anything.
				if h.VarName != "" {
					g.writeIndent()
					g.writef("%s := r\n", h.VarName)
					g.writeIndent()
					g.writef("_ = %s\n", h.VarName)
				}
				if err := g.stmts(h.Body); err != nil {
					return err
				}
				g.writeIndent()
				g.writef("return\n")
				continue
			}
			// Typed except — type-assert against *ClassName.
			g.writeIndent()
			if h.VarName != "" {
				g.writef("if %s, ok := r.(*%s); ok {\n", h.VarName, h.ClassName)
			} else {
				g.writef("if _, ok := r.(*%s); ok {\n", h.ClassName)
			}
			g.indent++
			if h.VarName != "" {
				g.writeIndent()
				g.writef("_ = %s\n", h.VarName)
			}
			if err := g.stmts(h.Body); err != nil {
				return err
			}
			g.writeIndent()
			g.writef("return\n")
			g.indent--
			g.writeIndent()
			g.writef("}\n")
		}
		// No handler matched: re-panic so outer scopes see it.
		g.writeIndent()
		g.writef("panic(r)\n")
		g.indent--
		g.writeIndent()
		g.writef("}()\n")
	}
	if err := g.stmts(t.Body); err != nil {
		return err
	}
	g.indent--
	g.writeIndent()
	g.writef("}()\n")
	return nil
}

func (g *gen) raise(r *ir.Raise) error {
	g.writeIndent()
	if r.Exc == nil {
		// Bare `raise` is only valid inside an except handler. We emit
		// `panic(r)` and assume an enclosing handler has `r` in scope —
		// the current codegen does name it `r`. Documented F3 limitation
		// otherwise.
		g.writef("panic(r)\n")
		return nil
	}
	g.writef("panic(")
	if err := g.expr(r.Exc); err != nil {
		return err
	}
	g.writef(")\n")
	return nil
}

func (g *gen) forRange(x *ir.ForRange) error {
	g.writeIndent()
	// Force int64 on the counter so the loop tolerates mixed-typed bounds
	// (e.g. untyped IntLit 1 alongside an int64 parameter).
	g.writef("for %s := int64(", x.Var)
	if err := g.expr(x.Start); err != nil {
		return err
	}
	g.writef("); %s < int64(", x.Var)
	if err := g.expr(x.Stop); err != nil {
		return err
	}
	g.writef("); ")
	if x.Step == nil {
		g.writef("%s++", x.Var)
	} else {
		g.writef("%s += int64(", x.Var)
		if err := g.expr(x.Step); err != nil {
			return err
		}
		g.writef(")")
	}
	g.writef(" {\n")
	g.indent++
	if err := g.stmts(x.Body); err != nil {
		return err
	}
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	return nil
}

func (g *gen) forEach(x *ir.ForEach) error {
	// Two-name forms emitted from tuple-target lowering. Each writes its
	// own header + extra rebindings, then falls through to forEachBody to
	// emit the user body + closing brace.
	switch x.Kind {
	case "dict":
		g.writeIndent()
		g.writef("for %s, %s := range ", x.Var, x.Var2)
		if err := g.expr(x.Iter); err != nil {
			return err
		}
		g.writef(" {\n")
		return g.forEachBody(x)
	case "enum":
		g.writeIndent()
		g.writef("for __i, %s := range ", x.Var2)
		if err := g.expr(x.Iter); err != nil {
			return err
		}
		g.writef(" {\n")
		g.indent++
		g.writeIndent()
		// Promote Go's int index to int64 so downstream arithmetic
		// matches Python's integer model. The optional `start=` kwarg
		// (parked in Iter2) shifts the index without an extra var.
		if x.Iter2 != nil {
			g.writef("%s := int64(__i) + ", x.Var)
			if err := g.expr(x.Iter2); err != nil {
				return err
			}
			g.writef("; _ = %s\n", x.Var)
		} else {
			g.writef("%s := int64(__i); _ = %s\n", x.Var, x.Var)
		}
		g.indent--
		return g.forEachBody(x)
	case "zip":
		g.writeIndent()
		g.writef("for __i := 0; __i < len(")
		if err := g.expr(x.Iter); err != nil {
			return err
		}
		g.writef(") && __i < len(")
		if err := g.expr(x.Iter2); err != nil {
			return err
		}
		g.writef("); __i++ {\n")
		g.indent++
		g.writeIndent()
		g.writef("%s := ", x.Var)
		if err := g.expr(x.Iter); err != nil {
			return err
		}
		g.writef("[__i]\n")
		g.writeIndent()
		g.writef("%s := ", x.Var2)
		if err := g.expr(x.Iter2); err != nil {
			return err
		}
		g.writef("[__i]\n")
		g.writeIndent()
		g.writef("_ = %s; _ = %s\n", x.Var, x.Var2)
		g.indent--
		return g.forEachBody(x)
	case "groupby":
		g.writeIndent()
		g.writef("for _, __gb := range ")
		if err := g.expr(x.Iter); err != nil {
			return err
		}
		g.writef(" {\n")
		g.indent++
		g.writeIndent()
		g.writef("%s := __gb.Key; _ = %s\n", x.Var, x.Var)
		g.writeIndent()
		g.writef("%s := __gb.Group; _ = %s\n", x.Var2, x.Var2)
		g.indent--
		return g.forEachBody(x)
	}
	// File-handle iteration: `for line in fh` over an *os.File bound by
	// the enclosing `with open(...) as fh:` block. Uses bufio.Scanner so
	// each iteration yields one stripped line as a string.
	if n, ok := x.Iter.(*ir.Name); ok && g.fileVars[n.N] {
		g.addImport("bufio")
		g.writeIndent()
		g.writef("__sc := bufio.NewScanner(%s)\n", n.N)
		g.writeIndent()
		g.writef("for __sc.Scan() {\n")
		g.indent++
		g.writeIndent()
		g.writef("%s := __sc.Text()\n", x.Var)
		g.writeIndent()
		g.writef("_ = %s\n", x.Var)
		g.indent--
		// Fall through to forEachBody for the user body + closing brace.
		return g.forEachBody(x)
	}
	// User-class iteration: `for v in obj` where obj's class defines
	// __iter__ returning a list/iterable. Emit `for _, v := range obj.Iter()`
	// after binding the loop var's element type for downstream attr access.
	if t := g.effectiveType(x.Iter); t != nil && t.Kind == ir.TyNamed {
		if fn := g.lookupMethod(t.Name, "__iter__"); fn != nil {
			if fn.Ret != nil && fn.Ret.Kind == ir.TyList && fn.Ret.Elem != nil {
				if fn.Ret.Elem.Kind == ir.TyNamed && x.Var != "_" {
					g.varTypes[x.Var] = fn.Ret.Elem.Name
				}
				g.writeIndent()
				switch x.Var {
				case "_":
					g.writef("for range ")
				default:
					g.writef("for _, %s := range ", x.Var)
				}
				if err := g.expr(x.Iter); err != nil {
					return err
				}
				g.writef(".Iter() {\n")
				return g.forEachBody(x)
			}
		}
	}
	// Default single-var range. Dict iterates keys (Python semantics);
	// channels (generators) take the value side.
	g.writeIndent()
	iterTy := x.Iter.TypeOf()
	single := false
	if iterTy != nil && iterTy.Kind == ir.TyDict {
		single = true
	}
	if c, ok := x.Iter.(*ir.Call); ok {
		if n, ok := c.Func.(*ir.Name); ok && g.generators[n.N] {
			single = true
		}
	}
	// Tag propagation: `for child in recv.iterdir()` should bind child
	// with the same tag the slice elements carry, so `child.name` etc.
	// dispatch through taggedPropAttrs.
	if mc, ok := x.Iter.(*ir.MethodCall); ok && x.Var != "_" {
		if recvTag := g.exprTag(mc.Recv); recvTag != "" {
			if elemTags, ok := taggedMethodElemTag[recvTag]; ok {
				if tag, ok := elemTags[mc.Method]; ok {
					g.varTypes[x.Var] = tag
				}
			}
		}
	}
	switch {
	case x.Var == "_":
		g.writef("for range ")
	case single:
		g.writef("for %s := range ", x.Var)
	default:
		g.writef("for _, %s := range ", x.Var)
	}
	if err := g.expr(x.Iter); err != nil {
		return err
	}
	g.writef(" {\n")
	return g.forEachBody(x)
}

// forEachBody renders the user body + closing brace shared by every
// ForEach codegen path. Caller must already have written the `for ... {`
// header line.
func (g *gen) forEachBody(x *ir.ForEach) error {
	g.indent++
	if err := g.stmts(x.Body); err != nil {
		return err
	}
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	return nil
}

func (g *gen) expr(e ir.Expr) error {
	switch x := e.(type) {
	case *ir.IntLit:
		g.writef("%d", x.V)
	case *ir.FloatLit:
		g.writef("%g", x.V)
	case *ir.BoolLit:
		if x.V {
			g.writef("true")
		} else {
			g.writef("false")
		}
	case *ir.NoneLit:
		g.writef("nil")
	case *ir.StrLit:
		g.writef("%s", strconv.Quote(x.V))
	case *ir.Name:
		g.writef("%s", x.N)
	case *ir.BinOp:
		// Operator overloading for the tagged datetime / timedelta types.
		// Python uses + / - on these objects; Go can't, so we route to
		// the helper methods defined on __Datetime.
		lt := g.exprTag(x.L)
		rt := g.exprTag(x.R)
		switch {
		case lt == "__Datetime" && rt == "__Timedelta":
			if x.Op == "+" {
				return g.emitMethodOp(x.L, "Add", x.R)
			}
			if x.Op == "-" {
				return g.emitMethodOp(x.L, "SubTimedelta", x.R)
			}
		case lt == "__Datetime" && rt == "__Datetime":
			if x.Op == "-" {
				return g.emitMethodOp(x.L, "Sub", x.R)
			}
		case lt == "__Timedelta" && rt == "__Timedelta":
			if x.Op == "+" {
				return g.emitMethodOp(x.L, "Add", x.R)
			}
			if x.Op == "-" {
				return g.emitMethodOp(x.L, "Sub", x.R)
			}
		case lt == "__Timedelta" && x.R.TypeOf() != nil && x.R.TypeOf().Kind == ir.TyInt:
			if x.Op == "*" {
				return g.emitMethodOp(x.L, "Mul", x.R)
			}
			if x.Op == "/" || x.Op == "//" {
				return g.emitMethodOp(x.L, "DivInt", x.R)
			}
		case x.L.TypeOf() != nil && x.L.TypeOf().Kind == ir.TyInt && rt == "__Timedelta":
			if x.Op == "*" {
				return g.emitMethodOp(x.R, "Mul", x.L)
			}
		case lt == "__Path" && x.Op == "/":
			return g.emitMethodOp(x.L, "Join", x.R)
		}
		// User-class operator overloading: when L's effective type is a
		// registered TyNamed class with a matching __op__ method, route
		// the binop through it. Augmented-assignment forms (`x += y`)
		// try `__iadd__` etc. first and fall back to `__add__`.
		if lTy := g.effectiveType(x.L); lTy != nil && lTy.Kind == ir.TyNamed {
			if x.InPlace {
				if iop := iopDunderName(x.Op); iop != "" {
					if fn := g.lookupMethod(lTy.Name, iop); fn != nil {
						return g.emitMethodOp(x.L, exportedDunder(iop), x.R)
					}
				}
			}
			dunder := opDunderName(x.Op)
			if dunder != "" {
				if fn := g.lookupMethod(lTy.Name, dunder); fn != nil {
					return g.emitMethodOp(x.L, exportedDunder(dunder), x.R)
				}
			}
		}
		// `s % args` printf-style string formatting. Python's % format
		// codes mostly overlap with Go's fmt; we pass the string through
		// unchanged and rely on Go fmt to do the substitution.
		if x.Op == "%" {
			lTy := x.L.TypeOf()
			if lTy != nil && lTy.Kind == ir.TyStr {
				g.addImport("fmt")
				g.writef("fmt.Sprintf(")
				if err := g.expr(x.L); err != nil {
					return err
				}
				if ll, ok := x.R.(*ir.ListLit); ok {
					for _, e := range ll.Elems {
						g.writef(", ")
						if err := g.boxedExpr(e); err != nil {
							return err
						}
					}
				} else {
					g.writef(", ")
					if err := g.boxedExpr(x.R); err != nil {
						return err
					}
				}
				g.writef(")")
				return nil
			}
		}
		// `a | b` over dicts → merged dict (b wins on key collision).
		if x.Op == "|" {
			lTy, rTy := x.L.TypeOf(), x.R.TypeOf()
			if lTy != nil && rTy != nil && lTy.Kind == ir.TyDict && rTy.Kind == ir.TyDict {
				mapGo := g.goType(lTy)
				g.writef("func() %s {\n", mapGo)
				g.indent++
				g.writeIndent()
				g.writef("__out := %s{}\n", mapGo)
				g.writeIndent()
				g.writef("for __k, __v := range ")
				if err := g.expr(x.L); err != nil {
					return err
				}
				g.writef(" { __out[__k] = __v }\n")
				g.writeIndent()
				g.writef("for __k, __v := range ")
				if err := g.expr(x.R); err != nil {
					return err
				}
				g.writef(" { __out[__k] = __v }\n")
				g.writeIndent()
				g.writef("return __out\n")
				g.indent--
				g.writeIndent()
				g.writef("}()")
				return nil
			}
		}
		// `"ab" * 3` / `3 * "ab"` → strings.Repeat. Same shape for list*int
		// would need element-type knowledge; skip for now.
		if x.Op == "*" {
			lTy, rTy := x.L.TypeOf(), x.R.TypeOf()
			if lTy != nil && rTy != nil {
				if lTy.Kind == ir.TyStr && rTy.Kind == ir.TyInt {
					g.addImport("strings")
					g.writef("strings.Repeat(")
					if err := g.expr(x.L); err != nil {
						return err
					}
					g.writef(", int(")
					if err := g.expr(x.R); err != nil {
						return err
					}
					g.writef("))")
					return nil
				}
				if lTy.Kind == ir.TyInt && rTy.Kind == ir.TyStr {
					g.addImport("strings")
					g.writef("strings.Repeat(")
					if err := g.expr(x.R); err != nil {
						return err
					}
					g.writef(", int(")
					if err := g.expr(x.L); err != nil {
						return err
					}
					g.writef("))")
					return nil
				}
				if lTy.Kind == ir.TyList && rTy.Kind == ir.TyInt {
					return g.emitListRepeat(x.L, x.R, lTy.Elem)
				}
				if lTy.Kind == ir.TyInt && rTy.Kind == ir.TyList {
					return g.emitListRepeat(x.R, x.L, rTy.Elem)
				}
			}
		}
		op := x.Op
		if op == "//" {
			if x.L.TypeOf() != nil && x.L.TypeOf().Kind == ir.TyInt &&
				x.R.TypeOf() != nil && x.R.TypeOf().Kind == ir.TyInt {
				op = "/"
			} else {
				return fmt.Errorf("// on non-int operands not supported")
			}
		}
		// True division: Python's `/` always returns float. Go's `/` on
		// int64 truncates, so when both operands are int we cast each
		// to float64 first to preserve the fraction.
		if x.Op == "/" {
			lTy, rTy := x.L.TypeOf(), x.R.TypeOf()
			if lTy != nil && rTy != nil && lTy.Kind == ir.TyInt && rTy.Kind == ir.TyInt {
				g.writef("(float64(")
				if err := g.expr(x.L); err != nil {
					return err
				}
				g.writef(") / float64(")
				if err := g.expr(x.R); err != nil {
					return err
				}
				g.writef("))")
				return nil
			}
		}
		// Mixed int/float arithmetic: Go won't auto-promote, so wrap
		// the int side in float64(...) when the other side is float.
		lTy, rTy := x.L.TypeOf(), x.R.TypeOf()
		castL, castR := false, false
		if lTy != nil && rTy != nil {
			if lTy.Kind == ir.TyInt && rTy.Kind == ir.TyFloat {
				castL = true
			}
			if lTy.Kind == ir.TyFloat && rTy.Kind == ir.TyInt {
				castR = true
			}
		}
		g.writef("(")
		if castL {
			g.writef("float64(")
		}
		if err := g.expr(x.L); err != nil {
			return err
		}
		if castL {
			g.writef(")")
		}
		g.writef(" %s ", op)
		if castR {
			g.writef("float64(")
		}
		if err := g.expr(x.R); err != nil {
			return err
		}
		if castR {
			g.writef(")")
		}
		g.writef(")")
	case *ir.CmpOp:
		if x.Op == "in" || x.Op == "notin" {
			return g.emitInOp(x)
		}
		// User-class comparison dispatch: route through __lt__ / __eq__
		// etc. when the LHS is a known TyNamed class with the method.
		if lTy := g.effectiveType(x.L); lTy != nil && lTy.Kind == ir.TyNamed {
			if dunder := opDunderName(x.Op); dunder != "" {
				if fn := g.lookupMethod(lTy.Name, dunder); fn != nil {
					return g.emitMethodOp(x.L, exportedDunder(dunder), x.R)
				}
			}
		}
		g.writef("(")
		if err := g.expr(x.L); err != nil {
			return err
		}
		g.writef(" %s ", x.Op)
		if err := g.expr(x.R); err != nil {
			return err
		}
		g.writef(")")
	case *ir.BoolOp:
		op := "&&"
		if x.Op == "or" {
			op = "||"
		}
		g.writef("(")
		if err := g.expr(x.L); err != nil {
			return err
		}
		g.writef(" %s ", op)
		if err := g.expr(x.R); err != nil {
			return err
		}
		g.writef(")")
	case *ir.UnaryOp:
		// User-class unary dispatch: `-obj` / `+obj` / `~obj` route through
		// __neg__ / __pos__ / __invert__ when the operand is a TyNamed
		// class instance with the method defined.
		if t := g.effectiveType(x.X); t != nil && t.Kind == ir.TyNamed {
			var dunder string
			switch x.Op {
			case "-":
				dunder = "__neg__"
			case "+":
				dunder = "__pos__"
			case "~":
				dunder = "__invert__"
			}
			if dunder != "" {
				if fn := g.lookupMethod(t.Name, dunder); fn != nil {
					_ = fn
					if err := g.expr(x.X); err != nil {
						return err
					}
					g.writef(".%s()", exportedDunder(dunder))
					return nil
				}
			}
		}
		switch x.Op {
		case "-":
			g.writef("(-")
		case "+":
			g.writef("(+")
		case "not":
			g.writef("(!")
		case "~":
			g.writef("(^")
		}
		if err := g.expr(x.X); err != nil {
			return err
		}
		g.writef(")")
	case *ir.Call:
		return g.call(x)
	case *ir.MethodCall:
		return g.methodCall(x)
	case *ir.Attribute:
		// `type(x).__name__` in CPython yields the class name as a str.
		// __gopy_type(...) returns a *__Type with .Name = the class name.
		if x.Name == "__name__" {
			if call, ok := x.Recv.(*ir.Call); ok {
				if n, ok := call.Func.(*ir.Name); ok && n.N == "type" {
					if err := g.expr(x.Recv); err != nil {
						return err
					}
					g.writef(".Name")
					return nil
				}
			}
			// `obj.__class__.__name__` — the inner __class__ already emits
			// `__gopy_type(obj)` (a *__Type), so we just need its .Name.
			if attr, ok := x.Recv.(*ir.Attribute); ok && attr.Name == "__class__" {
				if err := g.expr(x.Recv); err != nil {
					return err
				}
				g.writef(".Name")
				return nil
			}
			// `Foo.__name__` where Foo is a registered class name → "Foo".
			if n, ok := x.Recv.(*ir.Name); ok {
				if _, ok := g.classes[n.N]; ok {
					g.writef("%q", n.N)
					return nil
				}
			}
		}
		// `c.real` / `c.imag` on a complex128 — map to Go's real() / imag()
		// builtins so the access returns a float64.
		if x.Name == "real" || x.Name == "imag" {
			if t := g.effectiveType(x.Recv); t != nil && t.Kind == ir.TyComplex {
				g.writef("%s(", x.Name)
				if err := g.expr(x.Recv); err != nil {
					return err
				}
				g.writef(")")
				return nil
			}
		}
		// `obj.__class__` is equivalent to `type(obj)` — returns the
		// gopy __Type tag wrapper so `.__name__` / `.__qualname__` work.
		if x.Name == "__class__" {
			g.addImport("fmt")
			g.helpers["__gopy_type"] = helperGopyType
			g.writef("__gopy_type(")
			if err := g.boxedExpr(x.Recv); err != nil {
				return err
			}
			g.writef(")")
			return nil
		}
		// Enum member access: `Color.RED` → `ColorRED`.
		if n, ok := x.Recv.(*ir.Name); ok {
			if cls, ok := g.classes[n.N]; ok && cls.IsEnum {
				for _, m := range cls.EnumMembers {
					if m.Name == x.Name {
						g.writef("%s%s", cls.Name, m.Name)
						return nil
					}
				}
				return fmt.Errorf("enum %s has no member %q", cls.Name, x.Name)
			}
		}
		// Stdlib module attribute: sys.argv, etc.
		if n, ok := x.Recv.(*ir.Name); ok {
			if mod, ok := stdlibModules[n.N]; ok {
				attr, ok := mod.Attrs[x.Name]
				if !ok {
					return fmt.Errorf("unsupported stdlib attribute %s.%s", n.N, x.Name)
				}
				if attr.GoImport != "" {
					g.addImport(attr.GoImport)
				}
				if attr.Helper != "" {
					key := attr.HelperName
					if key == "" {
						key = attr.GoExpr
					}
					g.helpers[key] = attr.Helper
					for _, imp := range attr.HelperImports {
						g.addImport(imp)
					}
				}
				g.writef("%s", attr.GoExpr)
				return nil
			}
		}
		// Tagged-attribute dispatch (e.g. CompletedProcess.stdout).
		if tag := g.exprTag(x.Recv); tag != "" {
			if attrs, ok := taggedAttrs[tag]; ok {
				if info, ok := attrs[x.Name]; ok {
					if err := g.expr(x.Recv); err != nil {
						return err
					}
					g.writef(".%s", info.GoName)
					return nil
				}
			}
			if attrs, ok := taggedPropAttrs[tag]; ok {
				if info, ok := attrs[x.Name]; ok {
					if err := g.expr(x.Recv); err != nil {
						return err
					}
					g.writef(".%s()", info.GoName)
					return nil
				}
			}
		}
		// @property: receiver is an instance of a user class that registers
		// this attribute as a property (in itself or in any base). Emit
		// `recv.x()` (method call) rather than `recv.x` (field load).
		if ty := g.effectiveType(x.Recv); ty != nil && ty.Kind == ir.TyNamed {
			if g.hasProperty(ty.Name, x.Name) {
				if err := g.expr(x.Recv); err != nil {
					return err
				}
				g.writef(".%s()", x.Name)
				return nil
			}
		}
		if err := g.expr(x.Recv); err != nil {
			return err
		}
		g.writef(".%s", x.Name)
	case *ir.Subscript:
		// User-class `__getitem__` dispatch — emit `recv.Getitem(idx)`
		// instead of `recv[idx]` when the receiver is a known class with
		// the method.
		if vTy := g.effectiveType(x.Value); vTy != nil && vTy.Kind == ir.TyNamed {
			if fn := g.lookupMethod(vTy.Name, "__getitem__"); fn != nil {
				_ = fn
				if err := g.expr(x.Value); err != nil {
					return err
				}
				g.writef(".Getitem(")
				if err := g.expr(x.Index); err != nil {
					return err
				}
				g.writef(")")
				return nil
			}
		}
		if err := g.expr(x.Value); err != nil {
			return err
		}
		g.writef("[")
		if err := g.expr(x.Index); err != nil {
			return err
		}
		g.writef("]")
	case *ir.Slice:
		// Fast path: bounds are non-negative literal ints and no step.
		// Anything fancier (negative bound, step, runtime expression we
		// can't statically check) routes through the generic helper.
		if x.Step == nil && sliceBoundSafe(x.Low) && sliceBoundSafe(x.High) {
			if err := g.expr(x.Value); err != nil {
				return err
			}
			g.writef("[")
			if x.Low != nil {
				if err := g.expr(x.Low); err != nil {
					return err
				}
			}
			g.writef(":")
			if x.High != nil {
				if err := g.expr(x.High); err != nil {
					return err
				}
			}
			g.writef("]")
			return nil
		}
		return g.sliceWithHelper(x)
	case *ir.ListLit:
		g.writef("[]%s{", g.goType(x.ElemTy))
		for i, e := range x.Elems {
			if i > 0 {
				g.writef(", ")
			}
			if err := g.expr(e); err != nil {
				return err
			}
		}
		g.writef("}")
	case *ir.DictLit:
		g.writef("map[%s]%s{", g.goType(x.KeyTy), g.goType(x.ValTy))
		for i := range x.Keys {
			if i > 0 {
				g.writef(", ")
			}
			if err := g.expr(x.Keys[i]); err != nil {
				return err
			}
			g.writef(": ")
			if err := g.expr(x.Vals[i]); err != nil {
				return err
			}
		}
		g.writef("}")
	case *ir.FStr:
		return g.fstring(x)
	case *ir.Lambda:
		// Standalone-lambda emission. When the lambda has been retyped
		// via a Callable annotation (TyFunc target), use the concrete
		// param / return types so body ops compile; otherwise fall back
		// to `func(p any) any` and rely on call-site re-lowering for
		// builtins like sorted/map/filter to specialize properly.
		typedParams := false
		var retTy *ir.Type
		if x.Ty != nil && x.Ty.Kind == ir.TyFunc {
			typedParams = true
			retTy = x.Ty.FuncRet
		}
		g.writef("func(")
		for i, p := range x.Params {
			if i > 0 {
				g.writef(", ")
			}
			if typedParams && p.Ty != nil {
				g.writef("%s %s", p.Name, g.goType(p.Ty))
			} else {
				g.writef("%s any", p.Name)
			}
		}
		if typedParams && retTy != nil && retTy.Kind != ir.TyNone && retTy.Kind != ir.TyUnknown {
			g.writef(") %s { return ", g.goType(retTy))
		} else {
			g.writef(") any { return ")
		}
		if err := g.expr(x.Body); err != nil {
			return err
		}
		g.writef(" }")
	case *ir.IfExpr:
		// Go has no expression-level if; wrap both branches in an IIFE
		// whose return type comes from the inferred IR type. Branches must
		// share a static type (or `any`) for Go to compile the function.
		ret := g.goType(x.Ty)
		if ret == "" {
			ret = "any"
		}
		g.writef("func() %s {\n", ret)
		g.indent++
		g.writeIndent()
		g.writef("if ")
		if err := g.boolExpr(x.Cond); err != nil {
			return err
		}
		g.writef(" {\n")
		g.indent++
		g.writeIndent()
		g.writef("return ")
		if err := g.expr(x.Then); err != nil {
			return err
		}
		g.writef("\n")
		g.indent--
		g.writeIndent()
		g.writef("}\n")
		g.writeIndent()
		g.writef("return ")
		if err := g.expr(x.Else); err != nil {
			return err
		}
		g.writef("\n")
		g.indent--
		g.writeIndent()
		g.writef("}()")
	case *ir.ListComp:
		return g.listComp(x)
	case *ir.DictComp:
		return g.dictComp(x)
	default:
		return fmt.Errorf("transpile: unsupported expr %T", e)
	}
	return nil
}

// listComp emits an IIFE that builds the result slice in-place. The
// element-collection variable is named __out so it never collides with
// user code; the user's loop variable keeps its Python name. Multiple
// `for V in ITER` clauses (Extra) nest the loops in source order.
func (g *gen) listComp(c *ir.ListComp) error {
	elem := g.goType(c.ElemTy)
	if elem == "" || elem == "any" {
		elem = "any"
	}
	g.writef("func() []%s {\n", elem)
	g.indent++
	g.writeIndent()
	g.writef("__out := []%s{}\n", elem)
	openLoop := func(varName string, cond ir.Expr, iter ir.Expr) error {
		g.writeIndent()
		iterTy := iter.TypeOf()
		if iterTy != nil && iterTy.Kind == ir.TyDict {
			g.writef("for %s := range ", varName)
		} else {
			g.writef("for _, %s := range ", varName)
		}
		if err := g.expr(iter); err != nil {
			return err
		}
		g.writef(" {\n")
		g.indent++
		if cond != nil {
			g.writeIndent()
			g.writef("if !(")
			if err := g.expr(cond); err != nil {
				return err
			}
			g.writef(") {\n")
			g.indent++
			g.writeIndent()
			g.writef("continue\n")
			g.indent--
			g.writeIndent()
			g.writef("}\n")
		}
		return nil
	}
	if err := openLoop(c.Var, c.Cond, c.Iter); err != nil {
		return err
	}
	for _, gn := range c.Extra {
		if err := openLoop(gn.Var, gn.Cond, gn.Iter); err != nil {
			return err
		}
	}
	g.writeIndent()
	g.writef("__out = append(__out, ")
	if err := g.expr(c.Elt); err != nil {
		return err
	}
	g.writef(")\n")
	for range c.Extra {
		g.indent--
		g.writeIndent()
		g.writef("}\n")
	}
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// dictComp is the dict analogue of listComp.
func (g *gen) dictComp(c *ir.DictComp) error {
	kt := g.goType(c.KeyTy)
	vt := g.goType(c.ValTy)
	if kt == "" || kt == "any" {
		kt = "any"
	}
	if vt == "" || vt == "any" {
		vt = "any"
	}
	g.writef("func() map[%s]%s {\n", kt, vt)
	g.indent++
	g.writeIndent()
	g.writef("__out := map[%s]%s{}\n", kt, vt)
	openLoop := func(varName string, cond ir.Expr, iter ir.Expr) error {
		g.writeIndent()
		iterTy := iter.TypeOf()
		if iterTy != nil && iterTy.Kind == ir.TyDict {
			g.writef("for %s := range ", varName)
		} else {
			g.writef("for _, %s := range ", varName)
		}
		if err := g.expr(iter); err != nil {
			return err
		}
		g.writef(" {\n")
		g.indent++
		if cond != nil {
			g.writeIndent()
			g.writef("if !(")
			if err := g.expr(cond); err != nil {
				return err
			}
			g.writef(") {\n")
			g.indent++
			g.writeIndent()
			g.writef("continue\n")
			g.indent--
			g.writeIndent()
			g.writef("}\n")
		}
		return nil
	}
	if err := openLoop(c.Var, c.Cond, c.Iter); err != nil {
		return err
	}
	for _, gn := range c.Extra {
		if err := openLoop(gn.Var, gn.Cond, gn.Iter); err != nil {
			return err
		}
	}
	g.writeIndent()
	g.writef("__out[")
	if err := g.expr(c.Key); err != nil {
		return err
	}
	g.writef("] = ")
	if err := g.expr(c.Val); err != nil {
		return err
	}
	g.writef("\n")
	for range c.Extra {
		g.indent--
		g.writeIndent()
		g.writef("}\n")
	}
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

func (g *gen) fstring(f *ir.FStr) error {
	g.addImport("fmt")
	var fmtBuf strings.Builder
	type fstrArg struct {
		expr ir.Expr
		spec string
		conv byte
	}
	var args []fstrArg
	for _, p := range f.Parts {
		if p.Expr != nil {
			// User-class __format__ always wins when the expression is a
			// TyNamed instance with the dunder defined — CPython calls
			// __format__ even for the empty spec.
			hasUserFormat := false
			if p.Conv == 0 {
				if t := g.effectiveType(p.Expr); t != nil && t.Kind == ir.TyNamed {
					if fn := g.lookupMethod(t.Name, "__format__"); fn != nil {
						_ = fn
						hasUserFormat = true
					}
				}
			}
			if p.Spec != "" || p.Conv != 0 || hasUserFormat {
				// Spec / conv / user dispatch all yield a fully-formatted
				// string. Placeholder becomes %s.
				fmtBuf.WriteString("%s")
			} else {
				fmtBuf.WriteString("%v")
			}
			args = append(args, fstrArg{expr: p.Expr, spec: p.Spec, conv: p.Conv})
		} else {
			fmtBuf.WriteString(strings.ReplaceAll(p.Lit, "%", "%%"))
		}
	}
	g.writef("fmt.Sprintf(%s", strconv.Quote(fmtBuf.String()))
	for _, a := range args {
		g.writef(", ")
		// User-class __format__ dispatch: `f"{obj:spec}"` →
		// `obj.Format(spec)`. The conversion (!s / !r) still applies
		// first when present, so chain in the same order CPython does.
		// Empty spec also routes here so the dunder runs even for
		// `f"{obj}"`, matching CPython.
		if a.conv == 0 {
			if t := g.effectiveType(a.expr); t != nil && t.Kind == ir.TyNamed {
				if fn := g.lookupMethod(t.Name, "__format__"); fn != nil {
					_ = fn
					if err := g.expr(a.expr); err != nil {
						return err
					}
					g.writef(".Format(%s)", strconv.Quote(a.spec))
					continue
				}
			}
		}
		if a.spec != "" || a.conv != 0 {
			// __gopy_fmt_spec is defined alongside __gopy_str_format in the
			// same helper const, so adding one pulls in both. Helper body
			// uses strings.Builder / strings.Repeat.
			g.helpers["__gopy_str_format"] = helperStrFormat
			g.addImport("strings")
			g.writef("__gopy_fmt_spec(%s, ", strconv.Quote(a.spec))
			switch a.conv {
			case 'r':
				g.writef("fmt.Sprintf(\"%%#v\", ")
				if err := g.boxedExpr(a.expr); err != nil {
					return err
				}
				g.writef(")")
			case 's':
				g.writef("fmt.Sprint(")
				if err := g.boxedExpr(a.expr); err != nil {
					return err
				}
				g.writef(")")
			default:
				if err := g.boxedExpr(a.expr); err != nil {
					return err
				}
			}
			g.writef(")")
		} else {
			if err := g.expr(a.expr); err != nil {
				return err
			}
		}
	}
	g.writef(")")
	return nil
}

func (g *gen) call(c *ir.Call) error {
	if name, ok := c.Func.(*ir.Name); ok {
		// Alias from `from X import Y` — e.g. `getenv("PATH")` after
		// `from os import getenv` resolves to os.Getenv.
		if path, hit := g.aliases[name.N]; hit {
			// Specials that need per-arg-type code generation rather than
			// a static helper call: collections.Counter / itertools.chain /
			// itertools.accumulate. Each emits an IIFE specialized to the
			// argument's static element type.
			switch path {
			case "collections.Counter":
				return g.builtinCounter(c)
			case "itertools.chain":
				return g.builtinChain(c)
			case "itertools.accumulate":
				return g.builtinAccumulate(c)
			case "itertools.takewhile":
				return g.builtinTakeWhile(c)
			case "itertools.dropwhile":
				return g.builtinDropWhile(c)
			case "itertools.combinations":
				return g.builtinCombinations(c)
			case "itertools.product":
				return g.builtinProduct(c)
			case "itertools.permutations":
				return g.builtinPermutations(c)
			case "itertools.islice":
				return g.builtinIslice(c)
			case "itertools.repeat":
				return g.builtinRepeat(c)
			case "itertools.starmap":
				return g.builtinStarmap(c)
			case "itertools.filterfalse":
				return g.builtinFilterfalse(c)
			case "itertools.compress":
				return g.builtinCompress(c)
			case "itertools.count":
				return g.builtinCount(c)
			case "itertools.zip_longest":
				return g.builtinZipLongest(c)
			case "itertools.pairwise":
				return g.builtinPairwise(c)
			case "itertools.batched":
				return g.builtinBatched(c)
			case "heapq.heappush":
				return g.builtinHeappush(c)
			case "heapq.heappop":
				return g.builtinHeappop(c)
			case "heapq.heapify":
				return g.builtinHeapify(c)
			case "heapq.heappushpop":
				return g.builtinHeappushpop(c)
			case "heapq.nsmallest":
				return g.builtinNsmallest(c, false)
			case "heapq.nlargest":
				return g.builtinNsmallest(c, true)
			case "heapq.merge":
				return g.builtinHeapqMerge(c)
			case "bisect.bisect_left":
				return g.builtinBisect(c, false)
			case "bisect.bisect_right", "bisect.bisect":
				return g.builtinBisect(c, true)
			case "bisect.insort", "bisect.insort_left", "bisect.insort_right":
				return g.builtinInsort(c)
			case "subprocess.run":
				return g.builtinSubprocessRun(c)
			case "collections.deque":
				return g.builtinDeque(c)
			case "functools.reduce":
				return g.builtinReduceFn(c)
			case "functools.partial":
				return g.builtinPartial(c)
			case "datetime.timedelta":
				return g.builtinTimedelta(c)
			case "json.dumps":
				return g.builtinJSONDumps(c)
			case "dataclasses.asdict":
				return g.builtinAsdict(c)
			case "dataclasses.astuple":
				return g.builtinAstuple(c)
			case "dataclasses.replace":
				return g.builtinReplace(c)
			case "dataclasses.fields":
				return g.builtinFields(c)
			case "typing.cast":
				if len(c.Args) != 2 {
					return fmt.Errorf("typing.cast() takes (type, value)")
				}
				return g.expr(c.Args[1])
			case "asyncio.run":
				// gopy treats async as sync, so `asyncio.run(coro)` is
				// just the result of evaluating coro.
				if len(c.Args) != 1 {
					return fmt.Errorf("asyncio.run() takes 1 argument")
				}
				return g.expr(c.Args[0])
			case "asyncio.sleep":
				// No-op under sync semantics.
				g.writef("nil")
				return nil
			case "random.choice":
				return g.builtinRandomChoice(c)
			case "random.shuffle":
				return g.builtinRandomShuffle(c)
			case "random.sample":
				return g.builtinRandomSample(c)
			case "itertools.groupby":
				return g.builtinGroupBy(c)
			case "logging.basicConfig":
				// kwargs accepted and ignored.
				g.helpers["__gopy_log_basicConfig"] = helperLogBasicConfig
				g.writef("__gopy_log_basicConfig()")
				return nil
			}
			segs := splitDotted(path)
			if len(segs) >= 2 {
				modPath := strings.Join(segs[:len(segs)-1], ".")
				method := segs[len(segs)-1]
				// User-class numeric dunder dispatch for ceil/floor/trunc
				// imported via `from math import ceil`.
				if modPath == "math" && len(c.Args) == 1 {
					var dunder string
					switch method {
					case "ceil":
						dunder = "__ceil__"
					case "floor":
						dunder = "__floor__"
					case "trunc":
						dunder = "__trunc__"
					}
					if dunder != "" {
						if t := g.effectiveType(c.Args[0]); t != nil && t.Kind == ir.TyNamed {
							if fn := g.lookupMethod(t.Name, dunder); fn != nil {
								_ = fn
								if err := g.expr(c.Args[0]); err != nil {
									return err
								}
								g.writef(".%s()", exportedDunder(dunder))
								return nil
							}
						}
					}
				}
				if fn := lookupStdlibFunc(modPath, method); fn != nil {
					if fn.GoImport != "" {
						g.addImport(fn.GoImport)
					}
					if fn.Helper != "" {
						g.helpers[fn.GoFunc] = fn.Helper
						for _, imp := range fn.HelperImports {
							g.addImport(imp)
						}
					}
					for k, v := range fn.ExtraHelpers {
						g.helpers[k] = v
					}
					g.writef("%s(", fn.GoFunc)
					for i, a := range c.Args {
						if i > 0 {
							g.writef(", ")
						}
						if i == 0 && fn.IntArg0 {
							g.writef("int(")
							if err := g.expr(a); err != nil {
								return err
							}
							g.writef(")")
						} else {
							if err := g.expr(a); err != nil {
								return err
							}
						}
					}
					g.writef(")")
					return nil
				}
			}
		}
		// Class constructor: rewrite Foo(...) → NewFoo(...). Defaults
		// declared on the class's __init__ params (typically via
		// @dataclass) fill in missing trailing args.
		if cls, ok := g.classes[name.N]; ok {
			// Keyword arguments → match against InitArgs by name.
			kwIdx := map[string]ir.Expr{}
			for _, kw := range c.Keywords {
				kwIdx[kw.Name] = kw.Value
			}
			g.writef("New%s(", name.N)
			for i, p := range cls.InitArgs {
				if i > 0 {
					g.writef(", ")
				}
				switch {
				case i < len(c.Args):
					if err := g.expr(c.Args[i]); err != nil {
						return err
					}
				case kwIdx[p.Name] != nil:
					if err := g.expr(kwIdx[p.Name]); err != nil {
						return err
					}
					delete(kwIdx, p.Name)
				case p.Default != nil:
					if err := g.expr(p.Default); err != nil {
						return err
					}
				default:
					return fmt.Errorf("New%s: missing argument for %q", name.N, p.Name)
				}
			}
			g.writef(")")
			return nil
		}
		// Builtins.
		switch name.N {
		case "print":
			// Route through a small helper so bool prints as "True"/"False",
			// None as "None", and floats keep their trailing `.0`.
			// `sep=` / `end=` kwargs override the defaults.
			g.addImport("fmt")
			g.addImport("strconv")
			g.helpers["__gopy_print"] = helperPyPrint
			var sepExpr, endExpr ir.Expr
			for _, kw := range c.Keywords {
				switch kw.Name {
				case "sep":
					sepExpr = kw.Value
				case "end":
					endExpr = kw.Value
				default:
					return fmt.Errorf("print(): unsupported kwarg %q", kw.Name)
				}
			}
			g.writef("__gopy_print(")
			if sepExpr != nil {
				if err := g.expr(sepExpr); err != nil {
					return err
				}
			} else {
				g.writef("\" \"")
			}
			g.writef(", ")
			if endExpr != nil {
				if err := g.expr(endExpr); err != nil {
					return err
				}
			} else {
				g.writef("\"\\n\"")
			}
			for _, a := range c.Args {
				g.writef(", ")
				if err := g.boxedExpr(a); err != nil {
					return err
				}
			}
			g.writef(")")
			return nil
		case "len":
			if len(c.Args) != 1 {
				return fmt.Errorf("len() takes exactly 1 argument")
			}
			// User-class `__len__` dispatch: `len(obj)` → `obj.Len()`.
			if t := g.effectiveType(c.Args[0]); t != nil && t.Kind == ir.TyNamed {
				if fn := g.lookupMethod(t.Name, "__len__"); fn != nil {
					_ = fn
					if err := g.expr(c.Args[0]); err != nil {
						return err
					}
					g.writef(".Len()")
					return nil
				}
			}
			g.writef("int64(len(")
			if err := g.expr(c.Args[0]); err != nil {
				return err
			}
			g.writef("))")
			return nil
		case "str":
			if len(c.Args) != 1 {
				return fmt.Errorf("str() takes exactly 1 argument")
			}
			// `str(obj)` on a user class with `__str__` defined → call
			// `obj.String()`. fmt.Sprintf("%v", obj) does this too via
			// the Stringer interface, but going direct is clearer in the
			// emitted code.
			if t := g.effectiveType(c.Args[0]); t != nil && t.Kind == ir.TyNamed {
				if fn := g.lookupMethod(t.Name, "__str__"); fn != nil {
					_ = fn
					if err := g.expr(c.Args[0]); err != nil {
						return err
					}
					g.writef(".String()")
					return nil
				}
			}
			g.addImport("fmt")
			g.writef("fmt.Sprintf(\"%%v\", ")
			if err := g.expr(c.Args[0]); err != nil {
				return err
			}
			g.writef(")")
			return nil
		case "int":
			if len(c.Args) != 1 {
				return fmt.Errorf("int() takes exactly 1 argument")
			}
			// `int(obj)` → `obj.Int()` when user class has `__int__`.
			if t := g.effectiveType(c.Args[0]); t != nil && t.Kind == ir.TyNamed {
				if fn := g.lookupMethod(t.Name, "__int__"); fn != nil {
					_ = fn
					if err := g.expr(c.Args[0]); err != nil {
						return err
					}
					g.writef(".Int()")
					return nil
				}
			}
			// If the arg's IR type is concretely numeric, the simple Go
			// cast wins. Otherwise (any from **kwargs, a bare interface
			// from json.loads, etc.) we route through a helper that
			// type-switches over the common numeric/string forms.
			if t := c.Args[0].TypeOf(); t != nil &&
				(t.Kind == ir.TyInt || t.Kind == ir.TyFloat || t.Kind == ir.TyBool) {
				g.writef("int64(")
				if err := g.expr(c.Args[0]); err != nil {
					return err
				}
				g.writef(")")
				return nil
			}
			g.addImport("strconv")
			g.helpers["__gopy_int"] = helperGopyInt
			g.writef("__gopy_int(")
			if err := g.expr(c.Args[0]); err != nil {
				return err
			}
			g.writef(")")
			return nil
		case "bool":
			if len(c.Args) != 1 {
				return fmt.Errorf("bool() takes exactly 1 argument")
			}
			// User-class `__bool__` dispatch: `bool(obj)` → `obj.Bool()`.
			if t := g.effectiveType(c.Args[0]); t != nil && t.Kind == ir.TyNamed {
				if fn := g.lookupMethod(t.Name, "__bool__"); fn != nil {
					_ = fn
					if err := g.expr(c.Args[0]); err != nil {
						return err
					}
					g.writef(".Bool()")
					return nil
				}
			}
			t := c.Args[0].TypeOf()
			if t != nil && t.Kind == ir.TyBool {
				return g.expr(c.Args[0])
			}
			// Numeric / string truthy: emit a small inline check.
			if t != nil && (t.Kind == ir.TyInt || t.Kind == ir.TyFloat) {
				g.writef("(")
				if err := g.expr(c.Args[0]); err != nil {
					return err
				}
				g.writef(" != 0)")
				return nil
			}
			if t != nil && t.Kind == ir.TyStr {
				g.writef("(len(")
				if err := g.expr(c.Args[0]); err != nil {
					return err
				}
				g.writef(") > 0)")
				return nil
			}
			if t != nil && t.Kind == ir.TyList {
				g.writef("(len(")
				if err := g.expr(c.Args[0]); err != nil {
					return err
				}
				g.writef(") > 0)")
				return nil
			}
			// Fallback: route through a helper.
			g.helpers["__gopy_bool"] = helperGopyBool
			g.writef("__gopy_bool(")
			if err := g.boxedExpr(c.Args[0]); err != nil {
				return err
			}
			g.writef(")")
			return nil
		case "float":
			if len(c.Args) != 1 {
				return fmt.Errorf("float() takes exactly 1 argument")
			}
			// `float(obj)` → `obj.Float()` when user class has `__float__`.
			if t := g.effectiveType(c.Args[0]); t != nil && t.Kind == ir.TyNamed {
				if fn := g.lookupMethod(t.Name, "__float__"); fn != nil {
					_ = fn
					if err := g.expr(c.Args[0]); err != nil {
						return err
					}
					g.writef(".Float()")
					return nil
				}
			}
			if t := c.Args[0].TypeOf(); t != nil &&
				(t.Kind == ir.TyInt || t.Kind == ir.TyFloat) {
				g.writef("float64(")
				if err := g.expr(c.Args[0]); err != nil {
					return err
				}
				g.writef(")")
				return nil
			}
			g.addImport("strconv")
			g.helpers["__gopy_float"] = helperGopyFloat
			g.writef("__gopy_float(")
			if err := g.expr(c.Args[0]); err != nil {
				return err
			}
			g.writef(")")
			return nil
		case "sorted":
			return g.builtinSorted(c)
		case "map":
			return g.builtinMap(c)
		case "filter":
			return g.builtinFilter(c)
		case "reversed":
			return g.builtinReversed(c)
		case "abs":
			return g.builtinAbs(c)
		case "round":
			return g.builtinRound(c)
		case "isinstance":
			return g.builtinIsInstance(c)
		case "getattr":
			return g.builtinGetattr(c)
		case "setattr":
			return g.builtinSetattr(c)
		case "hasattr":
			return g.builtinHasattr(c)
		case "issubclass":
			return g.builtinIsSubclass(c)
		case "list":
			// `list(iter)` materializes an iterator. In the gopy shim,
			// iterables that map to slices already are slices, so this
			// is a pass-through. Strings could feasibly be supported by
			// splitting into runes, but we punt for now.
			if len(c.Args) != 1 {
				return fmt.Errorf("list() takes 1 argument")
			}
			return g.expr(c.Args[0])
		case "id":
			if len(c.Args) != 1 {
				return fmt.Errorf("id() takes 1 argument")
			}
			g.addImport("fmt")
			g.helpers["__gopy_id"] = helperGopyID
			g.writef("__gopy_id(")
			if err := g.boxedExpr(c.Args[0]); err != nil {
				return err
			}
			g.writef(")")
			return nil
		case "hash":
			if len(c.Args) != 1 {
				return fmt.Errorf("hash() takes 1 argument")
			}
			// `hash(obj)` → `obj.Hash()` when user class defines __hash__.
			if t := g.effectiveType(c.Args[0]); t != nil && t.Kind == ir.TyNamed {
				if fn := g.lookupMethod(t.Name, "__hash__"); fn != nil {
					_ = fn
					if err := g.expr(c.Args[0]); err != nil {
						return err
					}
					g.writef(".Hash()")
					return nil
				}
			}
			g.addImport("fmt")
			g.addImport("hash/fnv")
			g.helpers["__gopy_hash"] = helperGopyHash
			g.writef("__gopy_hash(")
			if err := g.boxedExpr(c.Args[0]); err != nil {
				return err
			}
			g.writef(")")
			return nil
		case "iter":
			// CPython returns an iterator; the gopy shim returns the
			// argument unchanged so `for x in iter(xs)` works the same
			// way `for x in xs` does.
			if len(c.Args) != 1 {
				return fmt.Errorf("iter() takes 1 argument")
			}
			return g.expr(c.Args[0])
		case "next":
			return g.builtinNext(c)
		case "pow":
			return g.builtinPow(c)
		case "chr":
			return g.builtinChr(c)
		case "ord":
			return g.builtinOrd(c)
		case "repr":
			return g.builtinRepr(c)
		case "divmod":
			return g.builtinDivmod(c)
		case "any":
			return g.builtinReduce(c, "any")
		case "all":
			return g.builtinReduce(c, "all")
		case "sum":
			return g.builtinReduce(c, "sum")
		case "min":
			return g.builtinReduce(c, "min")
		case "max":
			return g.builtinReduce(c, "max")
		case "set", "frozenset":
			return g.builtinSet(c)
		case "type":
			return g.builtinType(c)
		case "complex":
			return g.builtinComplex(c)
		case "enumerate":
			return g.builtinEnumerate(c)
		case "zip":
			return g.builtinZipPairs(c)
		case "format":
			return g.builtinFormat(c)
		case "hex":
			return g.builtinHex(c)
		case "oct":
			return g.builtinOct(c)
		case "bin":
			return g.builtinBin(c)
		case "callable":
			return g.builtinCallable(c)
		case "ascii":
			return g.builtinAscii(c)
		case "vars":
			return g.builtinVars(c)
		case "dir":
			return g.builtinDir(c)
		case "eval", "exec", "compile":
			return fmt.Errorf("%s() is not supported by gopy (no runtime Python interpreter)", name.N)
		}
	}
	// User-defined free function: resolve kwargs/defaults if any.
	if name, ok := c.Func.(*ir.Name); ok {
		if fn, ok := g.funcs[name.N]; ok {
			return g.userFuncCall(fn, c)
		}
	}
	// Callable instance: `obj(args)` where obj is a class instance with
	// __call__ defined dispatches to obj.Call(args).
	if t := g.effectiveType(c.Func); t != nil && t.Kind == ir.TyNamed {
		if fn := g.lookupMethod(t.Name, "__call__"); fn != nil {
			_ = fn
			if err := g.expr(c.Func); err != nil {
				return err
			}
			g.writef(".Call(")
			for i, a := range c.Args {
				if i > 0 {
					g.writef(", ")
				}
				if err := g.expr(a); err != nil {
					return err
				}
			}
			if len(c.Keywords) > 0 {
				return fmt.Errorf("kwargs not supported on __call__")
			}
			g.writef(")")
			return nil
		}
	}
	if err := g.expr(c.Func); err != nil {
		return err
	}
	g.writef("(")
	for i, a := range c.Args {
		if i > 0 {
			g.writef(", ")
		}
		if err := g.expr(a); err != nil {
			return err
		}
	}
	if len(c.Keywords) > 0 {
		return fmt.Errorf("kwargs not supported on this call target")
	}
	g.writef(")")
	return nil
}

// userFuncCall emits a call to a known free function, resolving the call's
// positional arguments and keyword arguments against the function's
// parameter list. Excess positional args feed Vararg (*args); unmatched
// keywords feed Kwarg (**kwargs). Missing trailing positionals are filled
// from each parameter's Default, evaluated at the call site.
func (g *gen) userFuncCall(fn *ir.Func, c *ir.Call) error {
	if len(c.Args) > len(fn.Params) && fn.Vararg == nil {
		return fmt.Errorf("%s: too many positional arguments (got %d, expected %d)", fn.Name, len(c.Args), len(fn.Params))
	}
	kwIdx := map[string]ir.Expr{}
	for _, kw := range c.Keywords {
		if _, dup := kwIdx[kw.Name]; dup {
			return fmt.Errorf("%s: duplicate keyword %q", fn.Name, kw.Name)
		}
		kwIdx[kw.Name] = kw.Value
	}
	g.writef("%s(", fn.Name)
	for i, p := range fn.Params {
		if i > 0 {
			g.writef(", ")
		}
		// When the param is typed `any`, the argument must be a
		// concretely-typed Go value — otherwise untyped literals would
		// box as `int` / `float64` rather than the IR's int64 / float64.
		emit := g.expr
		if p.Ty != nil && p.Ty.Kind == ir.TyAny {
			emit = g.boxedExpr
		}
		// Empty list / dict literals carry TyAny element types. When
		// the parameter is a typed collection, emit the empty literal
		// with the target type so Go accepts it without a conversion.
		if p.Ty != nil && (p.Ty.Kind == ir.TyList || p.Ty.Kind == ir.TyDict) {
			tgt := p.Ty
			emit = func(e ir.Expr) error {
				if ll, ok := e.(*ir.ListLit); ok && len(ll.Elems) == 0 && tgt.Kind == ir.TyList {
					g.writef("%s{}", g.goType(tgt))
					return nil
				}
				if dl, ok := e.(*ir.DictLit); ok && len(dl.Keys) == 0 && tgt.Kind == ir.TyDict {
					g.writef("%s{}", g.goType(tgt))
					return nil
				}
				return g.expr(e)
			}
		}
		switch {
		case i < len(c.Args):
			if _, dup := kwIdx[p.Name]; dup {
				return fmt.Errorf("%s: keyword %q clashes with positional", fn.Name, p.Name)
			}
			if err := emit(c.Args[i]); err != nil {
				return err
			}
		case kwIdx[p.Name] != nil:
			if err := emit(kwIdx[p.Name]); err != nil {
				return err
			}
			delete(kwIdx, p.Name)
		case p.Default != nil:
			if err := emit(p.Default); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s: missing argument for %q", fn.Name, p.Name)
		}
	}
	if fn.Vararg != nil {
		if len(fn.Params) > 0 {
			g.writef(", ")
		}
		extras := c.Args[min(len(c.Args), len(fn.Params)):]
		// Splat: `f(*xs)` for a Vararg-accepting function. Convert the
		// typed input slice to `[]any` (the Vararg's actual Go type) so
		// the assignment typechecks regardless of the inner element
		// type.
		if len(extras) == 1 {
			if st, ok := extras[0].(*ir.Starred); ok {
				g.writef("func() []any { __r := []any{}; for _, __v := range ")
				if err := g.expr(st.Value); err != nil {
					return err
				}
				g.writef(" { __r = append(__r, __v) }; return __r }()")
				goto kwargsBlock
			}
		}
		g.writef("[]any{")
		for i, a := range extras {
			if i > 0 {
				g.writef(", ")
			}
			if err := g.boxedExpr(a); err != nil {
				return err
			}
		}
		g.writef("}")
	}
kwargsBlock:
	if fn.Kwarg != nil {
		if len(fn.Params) > 0 || fn.Vararg != nil {
			g.writef(", ")
		}
		// Splat **kwargs values from the kwIdx — the lower stage tags
		// them with the sentinel name "**".
		splats := kwIdx["**"]
		delete(kwIdx, "**")
		if splats == nil && len(kwIdx) == 0 {
			g.writef("map[string]any{}")
		} else if splats == nil {
			g.writef("map[string]any{")
			first := true
			for k, v := range kwIdx {
				if !first {
					g.writef(", ")
				}
				first = false
				g.writef("%q: ", k)
				if err := g.boxedExpr(v); err != nil {
					return err
				}
			}
			g.writef("}")
		} else {
			// Build the kwargs map dynamically: seed with the splatted
			// dict, overwrite with any explicit keyword arguments.
			g.writef("func() map[string]any {\n")
			g.indent++
			g.writeIndent()
			g.writef("__r := map[string]any{}\n")
			g.writeIndent()
			g.writef("for __k, __v := range ")
			if err := g.expr(splats); err != nil {
				return err
			}
			g.writef(" { __r[__k] = __v }\n")
			for k, v := range kwIdx {
				g.writeIndent()
				g.writef("__r[%q] = ", k)
				if err := g.boxedExpr(v); err != nil {
					return err
				}
				g.writef("\n")
			}
			g.writeIndent()
			g.writef("return __r\n")
			g.indent--
			g.writeIndent()
			g.writef("}()")
		}
		// All remaining kwargs went into Kwarg; clear to skip the
		// "unknown keyword" error below.
		kwIdx = nil
	}
	for k := range kwIdx {
		return fmt.Errorf("%s: unknown keyword %q", fn.Name, k)
	}
	g.writef(")")
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// taggedMethodRename maps Python-style method names on stdlib-wrapped
// objects to the exported Go method names emitted by the inline helpers.
// snake_case → CamelCase, mostly.
var taggedMethodRename = map[string]map[string]string{
	"__Match": {
		"group":     "Group",
		"groups":    "Groups",
		"groupdict": "Groupdict",
		"start":     "Start",
		"end":       "End",
		"span":      "Span",
	},
	"__Path": {
		"exists":     "Exists",
		"is_file":    "IsFile",
		"is_dir":     "IsDir",
		"read_text":  "ReadText",
		"write_text": "WriteText",
		"iterdir":    "Iterdir",
		"mkdir":      "Mkdir",
		"unlink":     "Unlink",
		"glob":       "Glob",
	},
	"__Datetime": {
		"year":       "Year",
		"month":      "Month",
		"day":        "Day",
		"hour":       "Hour",
		"minute":     "Minute",
		"second":     "Second",
		"isoformat":  "Isoformat",
		"strftime":   "Strftime",
		"weekday":    "Weekday",
		"isoweekday": "Isoweekday",
		"timestamp":  "Timestamp",
	},
	"__Timedelta": {
		"total_seconds": "TotalSeconds",
	},
	"__Hasher": {
		"hexdigest": "Hexdigest",
	},
	"__Hmac": {
		"hexdigest": "Hexdigest",
		"update":    "Update",
	},
	"__Logger": {
		"debug":    "Debug",
		"info":     "Info",
		"warning":  "Warning",
		"error":    "Error",
		"critical": "Critical",
	},
	"__CSVWriter": {
		"writerow":  "Writerow",
		"writerows": "Writerows",
	},
	"__CSVDictWriter": {
		"writeheader": "Writeheader",
		"writerow":    "Writerow",
		"writerows":   "Writerows",
	},
	"__Lock": {
		"acquire": "Acquire",
		"release": "Release",
		"locked":  "Locked",
	},
	"__Deque": {
		"append":     "Append",
		"appendleft": "Appendleft",
		"pop":        "Pop",
		"popleft":    "Popleft",
	},
	"__Pattern": {
		"match":     "Match",
		"search":    "Search",
		"findall":   "Findall",
		"sub":       "Sub",
		"subn":      "Subn",
		"split":     "Split",
		"fullmatch": "Fullmatch",
	},
	"__Date": {
		"isoformat":  "Isoformat",
		"strftime":   "Strftime",
		"weekday":    "Weekday",
		"isoweekday": "Isoweekday",
	},
	"__Time": {
		"isoformat": "Isoformat",
	},
	"__Template": {
		"substitute":      "Substitute",
		"safe_substitute": "SafeSubstitute",
	},
	"__HTTPResponse": {
		"read":    "Read",
		"close":   "Close",
		"getcode": "Getcode",
	},
	"__HTTPClient": {
		"request":     "Request",
		"getresponse": "Getresponse",
		"close":       "Close",
	},
	"__XMLElement": {
		"find":    "Find",
		"findall": "Findall",
		"iter":    "Iter",
		"get":     "Get",
	},
	"__URLRequest": {
		"add_header": "Add_header",
	},
	"__Queue": {
		"put":   "Put",
		"get":   "Get",
		"qsize": "Qsize",
		"empty": "Empty",
		"full":  "Full",
	},
	"__Fraction": {
		"add":     "Add",
		"sub":     "Sub",
		"mul":     "Mul",
		"truediv": "Truediv",
		"eq":      "Eq",
		"lt":      "Lt",
	},
	"__ArgParser": {
		"add_argument": "AddArgument",
		"parse_args":   "ParseArgs",
	},
	"__ConfigParser": {
		"read":        "Read",
		"get":         "Get",
		"sections":    "Sections",
		"has_section": "Has_section",
		"has_option":  "Has_option",
	},
	"__ArgNamespace": {
		"get": "Get",
	},
	"__StringIO": {
		"write":    "Write",
		"getvalue": "Getvalue",
		"read":     "Read",
		"seek":     "Seek",
		"tell":     "Tell",
		"truncate": "Truncate",
		"close":    "Close",
	},
	"__Socket": {
		"connect":    "Connect",
		"bind":       "Bind",
		"listen":     "Listen",
		"accept":     "Accept",
		"send":       "Send",
		"sendall":    "Sendall",
		"recv":       "Recv",
		"close":      "Close",
		"setsockopt": "Setsockopt",
		"settimeout": "Settimeout",
	},
}

// taggedMethodRetTag tracks the tag of a tagged-method call's return
// value so chained dispatch (e.g. `re.compile(p).match(s).group()`) keeps
// resolving through exprTag.
var taggedMethodRetTag = map[string]map[string]string{
	"__Pattern": {
		"match":     "__Match",
		"search":    "__Match",
		"fullmatch": "__Match",
	},
	"__Datetime": {
		"__add__": "__Datetime",
		"replace": "__Datetime",
	},
	"__Date": {
		"replace": "__Date",
	},
}

// taggedMethodElemTag tracks methods that return a slice of tagged
// values. Used to propagate the element tag onto a `for x in recv.m()`
// loop variable so chained attribute access (e.g. `child.name`) keeps
// dispatching through the tag tables.
var taggedMethodElemTag = map[string]map[string]string{
	"__Path": {
		"iterdir": "__Path",
		"glob":    "__Path",
	},
	"__XMLElement": {
		"findall": "__XMLElement",
		"iter":    "__XMLElement",
	},
}

// taggedAttrInfo describes one tagged-value field: the Go field name to
// emit at the call site plus the IR type its access yields so chained
// expressions can dispatch correctly (e.g. `result.stdout.strip()`).
type taggedAttrInfo struct {
	GoName string
	Ty     *ir.Type
}

// taggedPropAttrs is the property-style equivalent of taggedAttrs: an
// attribute access on the tagged receiver emits a *method* call rather
// than a field load. Maps tag → python-name → {GoName, Ty}.
var taggedPropAttrs = map[string]map[string]taggedAttrInfo{
	"__Path": {
		"name":   {GoName: "Name", Ty: &ir.Type{Kind: ir.TyStr}},
		"parent": {GoName: "Parent", Ty: nil},
		"suffix": {GoName: "Suffix", Ty: &ir.Type{Kind: ir.TyStr}},
		"stem":   {GoName: "Stem", Ty: &ir.Type{Kind: ir.TyStr}},
	},
	"__Date": {
		"year":  {GoName: "Year", Ty: &ir.Type{Kind: ir.TyInt}},
		"month": {GoName: "Month", Ty: &ir.Type{Kind: ir.TyInt}},
		"day":   {GoName: "Day", Ty: &ir.Type{Kind: ir.TyInt}},
	},
	"__Time": {
		"hour":   {GoName: "Hour", Ty: &ir.Type{Kind: ir.TyInt}},
		"minute": {GoName: "Minute", Ty: &ir.Type{Kind: ir.TyInt}},
		"second": {GoName: "Second", Ty: &ir.Type{Kind: ir.TyInt}},
	},
	"__Timedelta": {
		"days":    {GoName: "Days", Ty: &ir.Type{Kind: ir.TyInt}},
		"seconds": {GoName: "Seconds", Ty: &ir.Type{Kind: ir.TyInt}},
	},
	"__Datetime": {
		"year":   {GoName: "Year", Ty: &ir.Type{Kind: ir.TyInt}},
		"month":  {GoName: "Month", Ty: &ir.Type{Kind: ir.TyInt}},
		"day":    {GoName: "Day", Ty: &ir.Type{Kind: ir.TyInt}},
		"hour":   {GoName: "Hour", Ty: &ir.Type{Kind: ir.TyInt}},
		"minute": {GoName: "Minute", Ty: &ir.Type{Kind: ir.TyInt}},
		"second": {GoName: "Second", Ty: &ir.Type{Kind: ir.TyInt}},
	},
}

var taggedAttrs = map[string]map[string]taggedAttrInfo{
	"__CompletedProcess": {
		"returncode": {GoName: "Returncode", Ty: &ir.Type{Kind: ir.TyInt}},
		"stdout":     {GoName: "Stdout", Ty: &ir.Type{Kind: ir.TyStr}},
		"stderr":     {GoName: "Stderr", Ty: &ir.Type{Kind: ir.TyStr}},
	},
	"__URLParseResult": {
		"scheme":   {GoName: "Scheme", Ty: &ir.Type{Kind: ir.TyStr}},
		"netloc":   {GoName: "Netloc", Ty: &ir.Type{Kind: ir.TyStr}},
		"path":     {GoName: "Path", Ty: &ir.Type{Kind: ir.TyStr}},
		"params":   {GoName: "Params", Ty: &ir.Type{Kind: ir.TyStr}},
		"query":    {GoName: "Query", Ty: &ir.Type{Kind: ir.TyStr}},
		"fragment": {GoName: "Fragment", Ty: &ir.Type{Kind: ir.TyStr}},
	},
	"__HTTPResponse": {
		"status":  {GoName: "Status", Ty: &ir.Type{Kind: ir.TyInt}},
		"headers": {GoName: "Headers", Ty: &ir.Type{Kind: ir.TyDict, Key: &ir.Type{Kind: ir.TyStr}, Val: &ir.Type{Kind: ir.TyStr}}},
	},
	"__Fraction": {
		"numerator":   {GoName: "Num", Ty: &ir.Type{Kind: ir.TyInt}},
		"denominator": {GoName: "Den", Ty: &ir.Type{Kind: ir.TyInt}},
	},
	"__XMLElement": {
		"tag":  {GoName: "Tag", Ty: &ir.Type{Kind: ir.TyStr}},
		"text": {GoName: "Text", Ty: &ir.Type{Kind: ir.TyStr}},
		"attrib": {GoName: "Attrib", Ty: &ir.Type{Kind: ir.TyDict, Key: &ir.Type{Kind: ir.TyStr}, Val: &ir.Type{Kind: ir.TyStr}}},
	},
}

func (g *gen) methodCall(m *ir.MethodCall) error {
	// `dict.fromkeys(iter, value)` — classmethod-like; build a typed map
	// from the iterable. Value type comes from the second arg or defaults
	// to None. Key type comes from the iterable's element type.
	if n, ok := m.Recv.(*ir.Name); ok && n.N == "dict" && m.Method == "fromkeys" {
		return g.builtinDictFromkeys(m)
	}
	// `str.maketrans(...)` — classmethod-like; build a rune→string map.
	if n, ok := m.Recv.(*ir.Name); ok && n.N == "str" && m.Method == "maketrans" {
		return g.builtinStrMaketrans(m)
	}
	// `chain.from_iterable(xs)` — flatten one level. Receiver Name "chain"
	// resolves via the itertools alias.
	if n, ok := m.Recv.(*ir.Name); ok && m.Method == "from_iterable" {
		if path, hit := g.aliases[n.N]; hit && path == "itertools.chain" {
			return g.builtinChainFromIterable(m)
		}
	}
	// `dt.replace(year=..., ...)` / `d.replace(year=..., ...)` — kwargs
	// can't ride the standard tagged-method dispatch, so intercept.
	if m.Method == "replace" {
		if tag := g.exprTag(m.Recv); tag == "__Datetime" {
			return g.builtinDatetimeReplace(m)
		}
		if tag := g.exprTag(m.Recv); tag == "__Date" {
			return g.builtinDateReplace(m)
		}
	}
	// Tagged-receiver method dispatch (Match.group, Path.exists, ...).
	// Tag may come from a Name (varTypes) or from a Call / MethodCall
	// whose declared stdlib return tag is recorded in the registry.
	if tag := g.exprTag(m.Recv); tag != "" {
		if rename, ok := taggedMethodRename[tag]; ok {
			if goName, ok := rename[m.Method]; ok {
				if err := g.expr(m.Recv); err != nil {
					return err
				}
				g.writef(".%s(", goName)
				for i, a := range m.Args {
					if i > 0 {
						g.writef(", ")
					}
					if err := g.expr(a); err != nil {
						return err
					}
				}
				g.writef(")")
				return nil
			}
			return fmt.Errorf("method .%s not supported on %s-tagged value", m.Method, tag)
		}
	}
	// Stdlib resolution that crosses module aliases or nested submodules
	// (e.g. `datetime.datetime.now()` and the aliased
	// `from datetime import datetime` form). Build a dotted path from the
	// receiver expression and try the registry; if it resolves we emit
	// the helper call directly without falling through.
	if path, ok := stdlibPathOf(m.Recv, g.aliases); ok {
		// Per-call-shape specials (same set as the Call branch).
		fullPath := path + "." + m.Method
		// Synthesize a fake Call so we can reuse the builders.
		synth := &ir.Call{Args: m.Args, Keywords: m.Keywords}
		switch fullPath {
		case "asyncio.run":
			if len(synth.Args) != 1 {
				return fmt.Errorf("asyncio.run() takes 1 argument")
			}
			return g.expr(synth.Args[0])
		case "asyncio.sleep":
			g.writef("nil")
			return nil
		case "collections.Counter":
			return g.builtinCounter(synth)
		case "itertools.chain":
			return g.builtinChain(synth)
		case "itertools.accumulate":
			return g.builtinAccumulate(synth)
		case "subprocess.run":
			return g.builtinSubprocessRun(synth)
		case "json.dumps":
			return g.builtinJSONDumps(synth)
		case "datetime.timedelta":
			return g.builtinTimedelta(synth)
		case "random.choice":
			return g.builtinRandomChoice(synth)
		case "random.shuffle":
			return g.builtinRandomShuffle(synth)
		case "random.sample":
			return g.builtinRandomSample(synth)
		case "heapq.heappush":
			return g.builtinHeappush(synth)
		case "heapq.heappop":
			return g.builtinHeappop(synth)
		case "heapq.heapify":
			return g.builtinHeapify(synth)
		case "heapq.heappushpop":
			return g.builtinHeappushpop(synth)
		case "heapq.nsmallest":
			return g.builtinNsmallest(synth, false)
		case "heapq.nlargest":
			return g.builtinNsmallest(synth, true)
		case "heapq.merge":
			return g.builtinHeapqMerge(synth)
		case "bisect.bisect_left":
			return g.builtinBisect(synth, false)
		case "bisect.bisect_right", "bisect.bisect":
			return g.builtinBisect(synth, true)
		case "bisect.insort", "bisect.insort_left", "bisect.insort_right":
			return g.builtinInsort(synth)
		case "itertools.pairwise":
			return g.builtinPairwise(synth)
		case "itertools.batched":
			return g.builtinBatched(synth)
		}
		// User-class numeric dunder dispatch for math.ceil / math.floor /
		// math.trunc: when the lone argument is a user class instance with
		// the corresponding dunder, route through the method instead of
		// the math stdlib helper.
		if path == "math" && len(m.Args) == 1 {
			var dunder string
			switch m.Method {
			case "ceil":
				dunder = "__ceil__"
			case "floor":
				dunder = "__floor__"
			case "trunc":
				dunder = "__trunc__"
			}
			if dunder != "" {
				if t := g.effectiveType(m.Args[0]); t != nil && t.Kind == ir.TyNamed {
					if fn := g.lookupMethod(t.Name, dunder); fn != nil {
						_ = fn
						if err := g.expr(m.Args[0]); err != nil {
							return err
						}
						g.writef(".%s()", exportedDunder(dunder))
						return nil
					}
				}
			}
		}
		if fn := lookupStdlibFunc(path, m.Method); fn != nil {
			if fn.GoImport != "" {
				g.addImport(fn.GoImport)
			}
			if fn.Helper != "" {
				g.helpers[fn.GoFunc] = fn.Helper
				for _, imp := range fn.HelperImports {
					g.addImport(imp)
				}
			}
			for k, v := range fn.ExtraHelpers {
				g.helpers[k] = v
			}
			g.writef("%s(", fn.GoFunc)
			for i, a := range m.Args {
				if i > 0 {
					g.writef(", ")
				}
				if i == 0 && fn.IntArg0 {
					g.writef("int(")
					if err := g.expr(a); err != nil {
						return err
					}
					g.writef(")")
				} else {
					if err := g.expr(a); err != nil {
						return err
					}
				}
			}
			g.writef(")")
			return nil
		}
	}
	// String methods: dispatched whenever the receiver resolves to a
	// TyStr — covers bare-Name strings, chained stdlib calls that return
	// a string (base64, urllib), and previously-typed locals.
	if rt := g.effectiveType(m.Recv); rt != nil && rt.Kind == ir.TyStr {
		if handled, err := g.stringMethod(m); handled || err != nil {
			return err
		}
	}
	// File handle methods inside a `with open(...) as fh:` block.
	if n, ok := m.Recv.(*ir.Name); ok && g.fileVars[n.N] {
		switch m.Method {
		case "read":
			g.addImport("os")
			g.addImport("io")
			g.helpers["__gopy_fh_read"] = helperFileReadAll
			g.writef("__gopy_fh_read(%s)", n.N)
			return nil
		case "write":
			if len(m.Args) != 1 {
				return fmt.Errorf("file.write() takes exactly 1 argument")
			}
			g.addImport("os")
			g.helpers["__gopy_fh_write"] = helperFileWrite
			g.writef("__gopy_fh_write(%s, ", n.N)
			if err := g.expr(m.Args[0]); err != nil {
				return err
			}
			g.writef(")")
			return nil
		}
		return fmt.Errorf("file.%s() not supported (F4: read/write only)", m.Method)
	}
	// Stdlib module function: os.getenv(...), time.time(), sys.exit(...)
	if n, ok := m.Recv.(*ir.Name); ok {
		if mod, ok := stdlibModules[n.N]; ok {
			fn, ok := mod.Funcs[m.Method]
			if !ok {
				return fmt.Errorf("unsupported stdlib function %s.%s", n.N, m.Method)
			}
			if fn.GoImport != "" {
				g.addImport(fn.GoImport)
			}
			if fn.Helper != "" {
				g.helpers[fn.GoFunc] = fn.Helper
				for _, imp := range fn.HelperImports {
					g.addImport(imp)
				}
			}
			for k, v := range fn.ExtraHelpers {
				g.helpers[k] = v
			}
			g.writef("%s(", fn.GoFunc)
			for i, a := range m.Args {
				if i > 0 {
					g.writef(", ")
				}
				if i == 0 && fn.IntArg0 {
					g.writef("int(")
					if err := g.expr(a); err != nil {
						return err
					}
					g.writef(")")
				} else {
					if err := g.expr(a); err != nil {
						return err
					}
				}
			}
			g.writef(")")
			return nil
		}
	}
	// Class-level method call: `Class.method(args)` when method is a
	// @classmethod becomes a free `<Class>_<method>(args)` call. Triggered
	// when Recv is Name(ClassName) and ClassName has the method registered.
	if n, ok := m.Recv.(*ir.Name); ok {
		if cls, ok := g.classes[n.N]; ok && cls.ClassMethods[m.Method] {
			g.writef("%s_%s(", n.N, m.Method)
			for i, a := range m.Args {
				if i > 0 {
					g.writef(", ")
				}
				if err := g.expr(a); err != nil {
					return err
				}
			}
			g.writef(")")
			return nil
		}
	}
	// super().X(...) → resolve against the current class's base.
	if isSuperCall(m.Recv) {
		if g.currentClass == nil || len(g.currentClass.Bases) == 0 {
			return fmt.Errorf("super() used outside a subclass method")
		}
		base := g.currentClass.Bases[0]
		if m.Method == "__init__" {
			// super().__init__(args) → self.<Base> = New<Base>(args)
			g.writef("self.%s = New%s(", base, base)
			for i, a := range m.Args {
				if i > 0 {
					g.writef(", ")
				}
				if err := g.expr(a); err != nil {
					return err
				}
			}
			g.writef(")")
			return nil
		}
		// Regular method: self.<Base>.method(args).
		g.writef("self.%s.%s(", base, m.Method)
		for i, a := range m.Args {
			if i > 0 {
				g.writef(", ")
			}
			if err := g.expr(a); err != nil {
				return err
			}
		}
		g.writef(")")
		return nil
	}
	// Dict view methods: keys() / values() materialize as IIFE-built
	// slices; items() is only viable in for-loop tuple-unpack form,
	// which is handled before we ever reach this branch.
	if rt := g.effectiveType(m.Recv); rt != nil && rt.Kind == ir.TyDict {
		switch m.Method {
		case "keys":
			g.writef("func() []%s {\n", g.goType(rt.Key))
			g.indent++
			g.writeIndent()
			g.writef("__out := make([]%s, 0, len(", g.goType(rt.Key))
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("))\n")
			g.writeIndent()
			g.writef("for k := range ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef(" { __out = append(__out, k) }\n")
			g.writeIndent()
			g.writef("return __out\n")
			g.indent--
			g.writeIndent()
			g.writef("}()")
			return nil
		case "values":
			g.writef("func() []%s {\n", g.goType(rt.Val))
			g.indent++
			g.writeIndent()
			g.writef("__out := make([]%s, 0, len(", g.goType(rt.Val))
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("))\n")
			g.writeIndent()
			g.writef("for _, v := range ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef(" { __out = append(__out, v) }\n")
			g.writeIndent()
			g.writef("return __out\n")
			g.indent--
			g.writeIndent()
			g.writef("}()")
			return nil
		case "items":
			// Standalone .items() returns a slice of {Key, Value} pairs.
			// For-loop tuple-unpack form (`for k, v in d.items()`) is
			// handled earlier via Kind="dict" before we ever reach this
			// branch.
			kGo, vGo := g.goType(rt.Key), g.goType(rt.Val)
			g.writef("func() []struct{ Key %s; Value %s } {\n", kGo, vGo)
			g.indent++
			g.writeIndent()
			g.writef("__out := make([]struct{ Key %s; Value %s }, 0, len(", kGo, vGo)
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("))\n")
			g.writeIndent()
			g.writef("for __k, __v := range ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef(" { __out = append(__out, struct{ Key %s; Value %s }{Key: __k, Value: __v}) }\n", kGo, vGo)
			g.writeIndent()
			g.writef("return __out\n")
			g.indent--
			g.writeIndent()
			g.writef("}()")
			return nil
		}
	}
	// dict.get(k, default) — emit a small inline ternary so missing keys
	// return the default rather than the zero value silently.
	if m.Method == "get" {
		if rt := m.Recv.TypeOf(); rt != nil && rt.Kind == ir.TyDict {
			if len(m.Args) != 2 {
				return fmt.Errorf("dict.get() requires (key, default) — F6 doesn't support single-arg form")
			}
			g.writef("func() %s {\n", g.goType(rt.Val))
			g.indent++
			g.writeIndent()
			g.writef("if __v, __ok := ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("[")
			if err := g.expr(m.Args[0]); err != nil {
				return err
			}
			g.writef("]; __ok {\n")
			g.indent++
			g.writeIndent()
			g.writef("return __v\n")
			g.indent--
			g.writeIndent()
			g.writef("}\n")
			g.writeIndent()
			g.writef("return ")
			if err := g.expr(m.Args[1]); err != nil {
				return err
			}
			g.writef("\n")
			g.indent--
			g.writeIndent()
			g.writef("}()")
			return nil
		}
	}
	// list mutators that need reassignment of the receiver expression.
	if rt := m.Recv.TypeOf(); rt != nil && rt.Kind == ir.TyList && rt.Elem != nil {
		switch m.Method {
		case "extend":
			if len(m.Args) != 1 {
				return fmt.Errorf("list.extend() takes 1 argument")
			}
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef(" = append(")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef(", ")
			if err := g.expr(m.Args[0]); err != nil {
				return err
			}
			g.writef("...)")
			return nil
		case "insert":
			if len(m.Args) != 2 {
				return fmt.Errorf("list.insert() takes 2 arguments")
			}
			elemGo := g.goType(rt.Elem)
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef(" = append(")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("[:")
			if err := g.expr(m.Args[0]); err != nil {
				return err
			}
			g.writef("], append([]%s{", elemGo)
			if err := g.expr(m.Args[1]); err != nil {
				return err
			}
			g.writef("}, ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("[")
			if err := g.expr(m.Args[0]); err != nil {
				return err
			}
			g.writef(":]...)...)")
			return nil
		case "remove":
			if len(m.Args) != 1 {
				return fmt.Errorf("list.remove() takes 1 argument")
			}
			elemGo := g.goType(rt.Elem)
			g.needsException = true
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef(" = func() []%s {\n", elemGo)
			g.indent++
			g.writeIndent()
			g.writef("__src := ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("\n")
			g.writeIndent()
			g.writef("var __target %s = ", elemGo)
			if err := g.expr(m.Args[0]); err != nil {
				return err
			}
			g.writef("\n")
			g.writeIndent()
			g.writef("for __i, __v := range __src { if __v == __target { return append(__src[:__i], __src[__i+1:]...) } }\n")
			g.writeIndent()
			g.writef("panic(NewException(\"ValueError: not in list\"))\n")
			g.indent--
			g.writeIndent()
			g.writef("}()")
			return nil
		case "clear":
			if len(m.Args) != 0 {
				return fmt.Errorf("list.clear() takes no arguments")
			}
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef(" = ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("[:0]")
			return nil
		case "copy":
			if len(m.Args) != 0 {
				return fmt.Errorf("list.copy() takes no arguments")
			}
			elemGo := g.goType(rt.Elem)
			g.writef("func() []%s { __src := ", elemGo)
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("; __out := make([]%s, len(__src)); copy(__out, __src); return __out }()", elemGo)
			return nil
		case "pop":
			if len(m.Args) > 1 {
				return fmt.Errorf("list.pop() takes 0 or 1 arguments")
			}
			elemGo := g.goType(rt.Elem)
			g.needsException = true
			g.writef("func() %s {\n", elemGo)
			g.indent++
			g.writeIndent()
			g.writef("__src := ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("\n")
			g.writeIndent()
			g.writef("if len(__src) == 0 { panic(NewException(\"IndexError: pop from empty list\")) }\n")
			g.writeIndent()
			g.writef("__i := len(__src) - 1\n")
			if len(m.Args) == 1 {
				g.writeIndent()
				g.writef("__i = int(")
				if err := g.expr(m.Args[0]); err != nil {
					return err
				}
				g.writef(")\n")
				g.writeIndent()
				g.writef("if __i < 0 { __i += len(__src) }\n")
				g.writeIndent()
				g.writef("if __i < 0 || __i >= len(__src) { panic(NewException(\"IndexError: pop index out of range\")) }\n")
			}
			g.writeIndent()
			g.writef("__v := __src[__i]\n")
			g.writeIndent()
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef(" = append(__src[:__i], __src[__i+1:]...)\n")
			g.writeIndent()
			g.writef("return __v\n")
			g.indent--
			g.writeIndent()
			g.writef("}()")
			return nil
		case "sort":
			// Only naive ascending sort. reverse=True flips the comparator.
			reverse := false
			var keyLam *ir.Lambda
			for _, kw := range m.Keywords {
				if kw.Name == "reverse" {
					if b, ok := kw.Value.(*ir.BoolLit); ok {
						reverse = b.V
					} else {
						return fmt.Errorf("list.sort(reverse=...): expected literal True/False")
					}
				} else if kw.Name == "key" {
					lam, ok := kw.Value.(*ir.Lambda)
					if !ok {
						return fmt.Errorf("list.sort(key=...): only inline lambda supported")
					}
					if len(lam.Params) != 1 {
						return fmt.Errorf("list.sort(key=...): lambda must take 1 arg")
					}
					keyLam = lam
				} else {
					return fmt.Errorf("list.sort(): unknown keyword %q", kw.Name)
				}
			}
			op := "<"
			if reverse {
				op = ">"
			}
			g.addImport("sort")
			if keyLam != nil {
				body, err := ir.LowerLambdaBody(keyLam, []*ir.Type{rt.Elem})
				if err != nil {
					return err
				}
				g.writef("sort.Slice(")
				if err := g.expr(m.Recv); err != nil {
					return err
				}
				g.writef(", func(i, j int) bool {\n")
				g.indent++
				g.writeIndent()
				g.writef("%s := ", keyLam.Params[0].Name)
				if err := g.expr(m.Recv); err != nil {
					return err
				}
				g.writef("[i]\n")
				g.writeIndent()
				g.writef("__ki := ")
				if err := g.expr(body); err != nil {
					return err
				}
				g.writef("\n")
				g.writeIndent()
				g.writef("%s = ", keyLam.Params[0].Name)
				if err := g.expr(m.Recv); err != nil {
					return err
				}
				g.writef("[j]\n")
				g.writeIndent()
				g.writef("__kj := ")
				if err := g.expr(body); err != nil {
					return err
				}
				g.writef("\n")
				g.writeIndent()
				g.writef("return __ki %s __kj\n", op)
				g.indent--
				g.writeIndent()
				g.writef("})")
				return nil
			}
			if rt.Elem.Kind != ir.TyInt && rt.Elem.Kind != ir.TyFloat && rt.Elem.Kind != ir.TyStr {
				return fmt.Errorf("list.sort(): only int/float/str element types supported")
			}
			g.writef("sort.Slice(")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef(", func(i, j int) bool { return ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("[i] %s ", op)
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("[j] })")
			return nil
		case "reverse":
			if len(m.Args) != 0 {
				return fmt.Errorf("list.reverse() takes no arguments")
			}
			g.writef("for __i, __j := 0, len(")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef(")-1; __i < __j; __i, __j = __i+1, __j-1 { ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("[__i], ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("[__j] = ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("[__j], ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("[__i] }")
			return nil
		case "count":
			if len(m.Args) != 1 {
				return fmt.Errorf("list.count() takes 1 argument")
			}
			elemGo := g.goType(rt.Elem)
			g.writef("func() int64 {\n")
			g.indent++
			g.writeIndent()
			g.writef("__src := ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("\n")
			g.writeIndent()
			g.writef("var __target %s = ", elemGo)
			if err := g.expr(m.Args[0]); err != nil {
				return err
			}
			g.writef("\n")
			g.writeIndent()
			g.writef("__n := int64(0)\n")
			g.writeIndent()
			g.writef("for _, __v := range __src { if __v == __target { __n++ } }\n")
			g.writeIndent()
			g.writef("return __n\n")
			g.indent--
			g.writeIndent()
			g.writef("}()")
			return nil
		case "index":
			if len(m.Args) != 1 {
				return fmt.Errorf("list.index() takes 1 argument")
			}
			elemGo := g.goType(rt.Elem)
			g.needsException = true
			g.writef("func() int64 {\n")
			g.indent++
			g.writeIndent()
			g.writef("__src := ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("\n")
			g.writeIndent()
			g.writef("var __target %s = ", elemGo)
			if err := g.expr(m.Args[0]); err != nil {
				return err
			}
			g.writef("\n")
			g.writeIndent()
			g.writef("for __i, __v := range __src { if __v == __target { return int64(__i) } }\n")
			g.writeIndent()
			g.writef("panic(NewException(\"ValueError: not in list\"))\n")
			g.indent--
			g.writeIndent()
			g.writef("}()")
			return nil
		}
	}
	// Dict mutators: update / pop / setdefault / clear.
	if rt := m.Recv.TypeOf(); rt != nil && rt.Kind == ir.TyDict {
		switch m.Method {
		case "update":
			if len(m.Args) != 1 {
				return fmt.Errorf("dict.update() takes 1 argument")
			}
			g.writef("for __k, __v := range ")
			if err := g.expr(m.Args[0]); err != nil {
				return err
			}
			g.writef(" { ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("[__k] = __v }")
			return nil
		case "pop":
			if len(m.Args) < 1 || len(m.Args) > 2 {
				return fmt.Errorf("dict.pop() takes (key[, default])")
			}
			valGo := g.goType(rt.Val)
			g.writef("func() %s {\n", valGo)
			g.indent++
			g.writeIndent()
			g.writef("__d := ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("\n")
			g.writeIndent()
			g.writef("__k := ")
			if err := g.expr(m.Args[0]); err != nil {
				return err
			}
			g.writef("\n")
			g.writeIndent()
			g.writef("if __v, __ok := __d[__k]; __ok { delete(__d, __k); return __v }\n")
			g.writeIndent()
			if len(m.Args) == 2 {
				g.writef("return ")
				if err := g.expr(m.Args[1]); err != nil {
					return err
				}
				g.writef("\n")
			} else {
				g.needsException = true
				g.writef("panic(NewException(\"KeyError\"))\n")
			}
			g.indent--
			g.writeIndent()
			g.writef("}()")
			return nil
		case "setdefault":
			if len(m.Args) != 2 {
				return fmt.Errorf("dict.setdefault() takes (key, default)")
			}
			valGo := g.goType(rt.Val)
			g.writef("func() %s {\n", valGo)
			g.indent++
			g.writeIndent()
			g.writef("__d := ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("\n")
			g.writeIndent()
			g.writef("__k := ")
			if err := g.expr(m.Args[0]); err != nil {
				return err
			}
			g.writef("\n")
			g.writeIndent()
			g.writef("if __v, __ok := __d[__k]; __ok { return __v }\n")
			g.writeIndent()
			g.writef("var __dv %s = ", valGo)
			if err := g.expr(m.Args[1]); err != nil {
				return err
			}
			g.writef("\n")
			g.writeIndent()
			g.writef("__d[__k] = __dv\n")
			g.writeIndent()
			g.writef("return __dv\n")
			g.indent--
			g.writeIndent()
			g.writef("}()")
			return nil
		case "clear":
			if len(m.Args) != 0 {
				return fmt.Errorf("dict.clear() takes no arguments")
			}
			g.writef("for __k := range ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef(" { delete(")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef(", __k) }")
			return nil
		case "copy":
			if len(m.Args) != 0 {
				return fmt.Errorf("dict.copy() takes no arguments")
			}
			mapGo := g.goType(rt)
			g.writef("func() %s { __src := ", mapGo)
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("; __out := make(%s, len(__src)); for __k, __v := range __src { __out[__k] = __v }; return __out }()", mapGo)
			return nil
		case "popitem":
			if len(m.Args) != 0 {
				return fmt.Errorf("dict.popitem() takes no arguments")
			}
			keyGo, valGo := g.goType(rt.Key), g.goType(rt.Val)
			g.needsException = true
			g.writef("func() []any {\n")
			g.indent++
			g.writeIndent()
			g.writef("__d := ")
			if err := g.expr(m.Recv); err != nil {
				return err
			}
			g.writef("\n")
			g.writeIndent()
			g.writef("if len(__d) == 0 { panic(NewException(\"KeyError: dictionary is empty\")) }\n")
			g.writeIndent()
			g.writef("var __k %s\n", keyGo)
			g.writeIndent()
			g.writef("var __v %s\n", valGo)
			g.writeIndent()
			g.writef("for __k, __v = range __d { break }\n")
			g.writeIndent()
			g.writef("delete(__d, __k)\n")
			g.writeIndent()
			g.writef("return []any{__k, __v}\n")
			g.indent--
			g.writeIndent()
			g.writef("}()")
			return nil
		}
	}
	// list.append(x) — Python mutates in place; Go's append returns a new slice
	// and we must reassign. This is only safe when the receiver is an addressable
	// expression like a Name or attribute; F2 enforces that.
	if m.Method == "append" {
		if len(m.Args) != 1 {
			return fmt.Errorf("append() takes exactly 1 argument")
		}
		if err := g.expr(m.Recv); err != nil {
			return err
		}
		g.writef(" = append(")
		if err := g.expr(m.Recv); err != nil {
			return err
		}
		g.writef(", ")
		if err := g.expr(m.Args[0]); err != nil {
			return err
		}
		g.writef(")")
		return nil
	}
	// User-defined instance method with kwargs/defaults: resolve via class
	// method registry. Triggers only when the receiver has a known
	// user-class type AND the method is declared on that class (or a base).
	if rt := g.effectiveType(m.Recv); rt != nil && rt.Kind == ir.TyNamed {
		if fn := g.lookupMethod(rt.Name, m.Method); fn != nil && (len(m.Keywords) > 0 || hasDefault(fn)) {
			return g.userMethodCall(m, fn)
		}
	}
	if len(m.Keywords) > 0 {
		return fmt.Errorf("keyword arguments not supported on this call target (.%s)", m.Method)
	}
	// Default: emit as recv.Method(args). Works for user-defined class methods.
	if err := g.expr(m.Recv); err != nil {
		return err
	}
	mname := m.Method
	switch mname {
	case "__str__":
		mname = "String"
	case "__repr__":
		mname = "Repr"
	case "__len__":
		mname = "Len"
	case "__hash__":
		mname = "Hash"
	default:
		if mapped := exportedDunder(mname); mapped != mname {
			mname = mapped
		}
	}
	g.writef(".%s(", mname)
	for i, a := range m.Args {
		if i > 0 {
			g.writef(", ")
		}
		if err := g.expr(a); err != nil {
			return err
		}
	}
	g.writef(")")
	return nil
}

// isinstanceNarrowInfo holds the bits the If codegen needs to emit the
// shadowing type assertion + register the narrowed local type.
type isinstanceNarrowInfo struct {
	Var    string
	GoType string
	Ty     *ir.Type
}

// isinstanceNarrow detects `isinstance(name, ClassOrPrim)` as the *entire*
// condition. Tuple-of-classes forms aren't narrowed (no single target
// type). Returns (info, true) on match.
func (g *gen) isinstanceNarrow(cond ir.Expr) (isinstanceNarrowInfo, bool) {
	call, ok := cond.(*ir.Call)
	if !ok {
		return isinstanceNarrowInfo{}, false
	}
	fn, ok := call.Func.(*ir.Name)
	if !ok || fn.N != "isinstance" || len(call.Args) != 2 {
		return isinstanceNarrowInfo{}, false
	}
	nameExpr, ok := call.Args[0].(*ir.Name)
	if !ok {
		return isinstanceNarrowInfo{}, false
	}
	clsName, ok := call.Args[1].(*ir.Name)
	if !ok {
		return isinstanceNarrowInfo{}, false
	}
	switch clsName.N {
	case "int":
		return isinstanceNarrowInfo{Var: nameExpr.N, GoType: "int64", Ty: &ir.Type{Kind: ir.TyInt}}, true
	case "float":
		return isinstanceNarrowInfo{Var: nameExpr.N, GoType: "float64", Ty: &ir.Type{Kind: ir.TyFloat}}, true
	case "str":
		return isinstanceNarrowInfo{Var: nameExpr.N, GoType: "string", Ty: &ir.Type{Kind: ir.TyStr}}, true
	case "bool":
		return isinstanceNarrowInfo{Var: nameExpr.N, GoType: "bool", Ty: &ir.Type{Kind: ir.TyBool}}, true
	}
	if _, ok := g.classes[clsName.N]; !ok {
		return isinstanceNarrowInfo{}, false
	}
	ty := &ir.Type{Kind: ir.TyNamed, Name: clsName.N}
	return isinstanceNarrowInfo{Var: nameExpr.N, GoType: g.goType(ty), Ty: ty}, true
}

// attrFieldType returns the declared IR type of `recv.attr` when recv is
// a Name whose effective type is a registered class with that field.
// Used by AssignAttr codegen to cast empty literals to the field type.
func (g *gen) attrFieldType(recv ir.Expr, attr string) *ir.Type {
	ty := g.effectiveType(recv)
	if ty == nil || ty.Kind != ir.TyNamed {
		if n, ok := recv.(*ir.Name); ok && n.N == "self" && g.currentClass != nil {
			ty = &ir.Type{Kind: ir.TyNamed, Name: g.currentClass.Name}
		}
	}
	if ty == nil || ty.Kind != ir.TyNamed {
		return nil
	}
	cls, ok := g.classes[ty.Name]
	if !ok {
		return nil
	}
	for _, f := range cls.Fields {
		if f.Name == attr {
			return f.Ty
		}
	}
	return nil
}

// lookupMethod walks className and its bases for a method named meth.
func (g *gen) lookupMethod(className, meth string) *ir.Func {
	visited := map[string]bool{}
	var walk func(string) *ir.Func
	walk = func(n string) *ir.Func {
		if visited[n] {
			return nil
		}
		visited[n] = true
		if mm, ok := g.methods[n]; ok {
			if fn, ok := mm[meth]; ok {
				return fn
			}
		}
		cls, ok := g.classes[n]
		if !ok {
			return nil
		}
		for _, b := range cls.Bases {
			if fn := walk(b); fn != nil {
				return fn
			}
		}
		return nil
	}
	return walk(className)
}

func hasDefault(fn *ir.Func) bool {
	for _, p := range fn.Params {
		if p.Default != nil {
			return true
		}
	}
	return false
}

// userMethodCall emits a call to a known user-defined method, resolving
// positional + keyword arguments against the method's parameter list and
// filling missing trailing arguments from each parameter's Default.
func (g *gen) userMethodCall(m *ir.MethodCall, fn *ir.Func) error {
	if len(m.Args) > len(fn.Params) {
		return fmt.Errorf("%s.%s: too many positional arguments (got %d, expected %d)", fn.Receiver.Ty.Name, fn.Name, len(m.Args), len(fn.Params))
	}
	kwIdx := map[string]ir.Expr{}
	for _, kw := range m.Keywords {
		if _, dup := kwIdx[kw.Name]; dup {
			return fmt.Errorf("%s: duplicate keyword %q", fn.Name, kw.Name)
		}
		kwIdx[kw.Name] = kw.Value
	}
	if err := g.expr(m.Recv); err != nil {
		return err
	}
	g.writef(".%s(", fn.Name)
	for i, p := range fn.Params {
		if i > 0 {
			g.writef(", ")
		}
		switch {
		case i < len(m.Args):
			if _, dup := kwIdx[p.Name]; dup {
				return fmt.Errorf("%s: keyword %q clashes with positional", fn.Name, p.Name)
			}
			if err := g.expr(m.Args[i]); err != nil {
				return err
			}
		case kwIdx[p.Name] != nil:
			if err := g.expr(kwIdx[p.Name]); err != nil {
				return err
			}
			delete(kwIdx, p.Name)
		case p.Default != nil:
			if err := g.expr(p.Default); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s: missing argument for %q", fn.Name, p.Name)
		}
	}
	for k := range kwIdx {
		return fmt.Errorf("%s: unknown keyword %q", fn.Name, k)
	}
	g.writef(")")
	return nil
}

// stdlibPathOf resolves a receiver expression to a dotted stdlib module
// path (e.g. "datetime.datetime") if all components are stdlib modules
// or submodules. It honors the alias map for the root name so that
// `from datetime import datetime` followed by `datetime.now()` looks up
// `datetime.datetime` rather than `datetime`.
//
// Returns ("", false) when the expression is not a stdlib path — for
// instance when the receiver is an instance variable.
func stdlibPathOf(e ir.Expr, aliases map[string]string) (string, bool) {
	parts, ok := collectAttrChain(e)
	if !ok {
		return "", false
	}
	// Apply alias at the root, then split (an alias can resolve to a
	// dotted path itself, e.g. datetime → datetime.datetime).
	root := parts[0]
	if v, ok := aliases[root]; ok {
		root = v
	}
	full := root
	for _, p := range parts[1:] {
		full += "." + p
	}
	// Verify the path resolves under stdlibModules so unrelated chains
	// like `self.user.name` don't accidentally match.
	segs := splitDotted(full)
	cur, ok := stdlibModules[segs[0]]
	if !ok {
		return "", false
	}
	for _, p := range segs[1:] {
		sub, ok := cur.Subs[p]
		if !ok {
			// Could be a function or attr leaf; that's fine for callers
			// who keep walking via Funcs map after we return.
			return full, true
		}
		cur = sub
	}
	return full, true
}

// collectAttrChain unrolls Attribute(Attribute(Name, _), _) into a slice
// [name, attr1, attr2, ...]. Anything else returns nil, false.
func collectAttrChain(e ir.Expr) ([]string, bool) {
	var parts []string
	cur := e
	for {
		switch x := cur.(type) {
		case *ir.Name:
			parts = append([]string{x.N}, parts...)
			return parts, true
		case *ir.Attribute:
			parts = append([]string{x.Name}, parts...)
			cur = x.Recv
		default:
			return nil, false
		}
	}
}

// detectDiamondConflicts walks each multi-base class and rejects cases
// where two distinct bases provide methods with the same name AND the
// subclass does not override it. Single-base classes are fine; subclasses
// that explicitly override the shadowed method are also fine.
func (g *gen) detectDiamondConflicts() error {
	for _, cls := range g.classes {
		if len(cls.Bases) < 2 {
			continue
		}
		// Collect inherited method names with their source base.
		seen := map[string]string{}
		own := map[string]bool{}
		for _, name := range cls.MethodNames {
			own[name] = true
		}
		for _, b := range cls.Bases {
			base, ok := g.classes[b]
			if !ok {
				continue
			}
			for _, name := range g.collectInheritedMethods(base) {
				if own[name] {
					continue // subclass overrides; no conflict
				}
				if prev, hit := seen[name]; hit && prev != b {
					return fmt.Errorf("class %s inherits %q from both %s and %s; override it explicitly in %s to disambiguate", cls.Name, name, prev, b, cls.Name)
				}
				seen[name] = b
			}
		}
	}
	return nil
}

// collectInheritedMethods returns every method name defined on the given
// class or any of its ancestors. Walks the base chain transitively.
func (g *gen) collectInheritedMethods(cls *ir.Class) []string {
	visited := map[string]bool{}
	var out []string
	var walk func(*ir.Class)
	walk = func(c *ir.Class) {
		if c == nil || visited[c.Name] {
			return
		}
		visited[c.Name] = true
		out = append(out, c.MethodNames...)
		for _, b := range c.Bases {
			walk(g.classes[b])
		}
	}
	walk(cls)
	return out
}

// emitClassAccessors writes the per-class getter / setter / has helpers
// used by getattr / setattr / hasattr. Each helper switches over the
// declared field name and routes to the actual Go field; unknown names
// return the supplied default (getter), succeed silently (setter), or
// report false (has).
func (g *gen) emitClassAccessors(c *ir.Class) error {
	// Getter: returns (any, bool). False ok means "no such field".
	g.writef("func __%s_get(self *%s, name string) (any, bool) {\n", c.Name, c.Name)
	g.indent++
	g.writeIndent()
	g.writef("switch name {\n")
	for _, f := range c.Fields {
		g.writeIndent()
		g.writef("case %q:\n", f.Name)
		g.indent++
		g.writeIndent()
		g.writef("return self.%s, true\n", f.Name)
		g.indent--
	}
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return nil, false\n")
	g.indent--
	g.writef("}\n\n")

	// Setter: returns bool indicating whether the field was found.
	g.writef("func __%s_set(self *%s, name string, value any) bool {\n", c.Name, c.Name)
	g.indent++
	g.writeIndent()
	g.writef("switch name {\n")
	for _, f := range c.Fields {
		g.writeIndent()
		g.writef("case %q:\n", f.Name)
		g.indent++
		g.writeIndent()
		g.writef("self.%s = value.(%s)\n", f.Name, g.goType(f.Ty))
		g.writeIndent()
		g.writef("return true\n")
		g.indent--
	}
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return false\n")
	g.indent--
	g.writef("}\n\n")

	return nil
}

// hasProperty walks className and its base chain looking for a @property
// method named attr. The class registry is keyed by Python class name and
// bases are also Python names, so the lookup is uniform.
func (g *gen) hasProperty(className, attr string) bool {
	visited := map[string]bool{}
	var walk func(string) bool
	walk = func(n string) bool {
		if visited[n] {
			return false
		}
		visited[n] = true
		cls, ok := g.classes[n]
		if !ok {
			return false
		}
		if cls.Properties[attr] {
			return true
		}
		for _, b := range cls.Bases {
			if walk(b) {
				return true
			}
		}
		return false
	}
	return walk(className)
}

// boolExpr emits e as a boolean condition. If e is a Name bound to a
// nullable stdlib type (Match, Path, Timedelta), Go won't accept the bare
// variable as a condition, so we rewrite to a nil comparison. Same for
// UnaryOp(Not, Name) where the Name is tagged.
func (g *gen) boolExpr(e ir.Expr) error {
	switch x := e.(type) {
	case *ir.Name:
		if g.varTypes[x.N] != "" {
			g.writef("%s != nil", x.N)
			return nil
		}
	case *ir.UnaryOp:
		if x.Op == "not" {
			if n, ok := x.X.(*ir.Name); ok && g.varTypes[n.N] != "" {
				g.writef("%s == nil", n.N)
				return nil
			}
		}
	}
	return g.expr(e)
}


// boxedExpr emits an expression wrapped in a Go conversion that produces
// a concretely-typed value (int64 / float64 / string / bool). Used when
// the value is about to be stored in an `any` slot — otherwise an untyped
// integer literal would land in the slot as `int`, not the `int64` the
// ORM later asserts against.
func (g *gen) boxedExpr(e ir.Expr) error {
	t := e.TypeOf()
	if t == nil {
		return g.expr(e)
	}
	switch t.Kind {
	case ir.TyInt:
		g.writef("int64(")
		if err := g.expr(e); err != nil {
			return err
		}
		g.writef(")")
		return nil
	case ir.TyFloat:
		g.writef("float64(")
		if err := g.expr(e); err != nil {
			return err
		}
		g.writef(")")
		return nil
	}
	return g.expr(e)
}

// builtinSorted emits an IIFE that returns a sorted copy of the input
// slice. Supports an optional `key=` lambda (re-lowered with the
// element type) and an optional `reverse=` bool.
func (g *gen) builtinSorted(c *ir.Call) error {
	if len(c.Args) != 1 {
		return fmt.Errorf("sorted() takes one positional argument")
	}
	var keyLambda *ir.Lambda
	reverse := false
	for _, kw := range c.Keywords {
		switch kw.Name {
		case "key":
			lam, ok := kw.Value.(*ir.Lambda)
			if !ok {
				return fmt.Errorf("sorted(key=...): only inline lambda supported")
			}
			if len(lam.Params) != 1 {
				return fmt.Errorf("sorted(key=...): lambda must take one argument")
			}
			keyLambda = lam
		case "reverse":
			b, ok := kw.Value.(*ir.BoolLit)
			if !ok {
				return fmt.Errorf("sorted(reverse=...): must be a bool literal")
			}
			reverse = b.V
		default:
			return fmt.Errorf("sorted(): unknown keyword %q", kw.Name)
		}
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("sorted(): %w", err)
	}
	g.addImport("sort")
	elemGo := g.goType(elem)
	g.writef("func() []%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__out := make([]%s, len(__src))\n", elemGo)
	g.writeIndent()
	g.writef("copy(__out, __src)\n")
	g.writeIndent()
	op := "<"
	if reverse {
		op = ">"
	}
	if keyLambda == nil {
		g.writef("sort.Slice(__out, func(i, j int) bool { return __out[i] %s __out[j] })\n", op)
	} else {
		// Re-lower the lambda body with the element type so arithmetic
		// and field access in the key expression typecheck.
		body, err := ir.LowerLambdaBody(keyLambda, []*ir.Type{elem})
		if err != nil {
			return fmt.Errorf("sorted(key=...): %w", err)
		}
		paramName := keyLambda.Params[0].Name
		g.writef("sort.Slice(__out, func(__i, __j int) bool {\n")
		g.indent++
		// Bind the lambda's parameter to out[i], compute keyI; then to
		// out[j], compute keyJ. We rebind (`=`) rather than redeclare.
		g.writeIndent()
		g.writef("%s := __out[__i]\n", paramName)
		g.writeIndent()
		g.writef("__ki := ")
		if err := g.expr(body); err != nil {
			return err
		}
		g.writef("\n")
		g.writeIndent()
		g.writef("%s = __out[__j]\n", paramName)
		g.writeIndent()
		g.writef("__kj := ")
		if err := g.expr(body); err != nil {
			return err
		}
		g.writef("\n")
		g.writeIndent()
		g.writef("_ = %s\n", paramName)
		g.writeIndent()
		g.writef("return __ki %s __kj\n", op)
		g.indent--
		g.writeIndent()
		g.writef("})\n")
	}
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinMap emits an IIFE that walks the iterable and applies the
// lambda to each element, materializing the result as a typed slice.
// First argument must be an inline lambda — function-value passing
// without lambdas is not yet supported.
func (g *gen) builtinMap(c *ir.Call) error {
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("map() takes (fn, iterable)")
	}
	lam, ok := c.Args[0].(*ir.Lambda)
	if !ok {
		return fmt.Errorf("map(): first argument must be an inline lambda")
	}
	if len(lam.Params) != 1 {
		return fmt.Errorf("map(): lambda must take one argument")
	}
	elem, err := listElemTypeOf(c.Args[1])
	if err != nil {
		return fmt.Errorf("map(): %w", err)
	}
	body, err := ir.LowerLambdaBody(lam, []*ir.Type{elem})
	if err != nil {
		return fmt.Errorf("map(): %w", err)
	}
	resTy := body.TypeOf()
	if resTy == nil || resTy.Kind == ir.TyUnknown {
		resTy = &ir.Type{Kind: ir.TyAny}
	}
	resGo := g.goType(resTy)
	paramName := lam.Params[0].Name
	g.writef("func() []%s {\n", resGo)
	g.indent++
	g.writeIndent()
	g.writef("__out := []%s{}\n", resGo)
	g.writeIndent()
	g.writef("for _, %s := range ", paramName)
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef(" {\n")
	g.indent++
	g.writeIndent()
	g.writef("__out = append(__out, ")
	if err := g.expr(body); err != nil {
		return err
	}
	g.writef(")\n")
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinFilter emits an IIFE that walks the iterable and keeps every
// element for which the lambda predicate returns true. Element type
// comes from the iterable; the lambda body must yield a bool.
func (g *gen) builtinFilter(c *ir.Call) error {
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("filter() takes (fn, iterable)")
	}
	lam, ok := c.Args[0].(*ir.Lambda)
	if !ok {
		return fmt.Errorf("filter(): first argument must be an inline lambda")
	}
	if len(lam.Params) != 1 {
		return fmt.Errorf("filter(): lambda must take one argument")
	}
	elem, err := listElemTypeOf(c.Args[1])
	if err != nil {
		return fmt.Errorf("filter(): %w", err)
	}
	body, err := ir.LowerLambdaBody(lam, []*ir.Type{elem})
	if err != nil {
		return fmt.Errorf("filter(): %w", err)
	}
	elemGo := g.goType(elem)
	paramName := lam.Params[0].Name
	g.writef("func() []%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__out := []%s{}\n", elemGo)
	g.writeIndent()
	g.writef("for _, %s := range ", paramName)
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef(" {\n")
	g.indent++
	g.writeIndent()
	g.writef("if ")
	if err := g.expr(body); err != nil {
		return err
	}
	g.writef(" { __out = append(__out, %s) }\n", paramName)
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinDivmod returns Python's (quotient, remainder) pair as a
// two-element slice. Both args must be int — float divmod yields
// different semantics (floor-div semantics) and is not yet supported.
func (g *gen) builtinDivmod(c *ir.Call) error {
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("divmod() takes two positional arguments")
	}
	at, bt := c.Args[0].TypeOf(), c.Args[1].TypeOf()
	if at == nil || bt == nil || at.Kind != ir.TyInt || bt.Kind != ir.TyInt {
		return fmt.Errorf("divmod() requires (int, int) for now")
	}
	g.writef("func() []int64 {\n")
	g.indent++
	g.writeIndent()
	g.writef("var __a int64 = ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("var __b int64 = ")
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("return []int64{__a / __b, __a %% __b}\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinPow lowers `pow(a, b)` to integer/float exponentiation. Float
// arguments route through math.Pow; integer arguments use a loop that
// keeps the result as int64 (matching Python's int**int → int).
func (g *gen) builtinPow(c *ir.Call) error {
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("pow() takes two positional arguments")
	}
	at := c.Args[0].TypeOf()
	bt := c.Args[1].TypeOf()
	floatish := (at != nil && at.Kind == ir.TyFloat) || (bt != nil && bt.Kind == ir.TyFloat)
	if floatish {
		g.addImport("math")
		g.writef("math.Pow(float64(")
		if err := g.expr(c.Args[0]); err != nil {
			return err
		}
		g.writef("), float64(")
		if err := g.expr(c.Args[1]); err != nil {
			return err
		}
		g.writef("))")
		return nil
	}
	g.writef("func() int64 {\n")
	g.indent++
	g.writeIndent()
	g.writef("var __r int64 = 1\n")
	g.writeIndent()
	g.writef("var __b int64 = ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("var __e int64 = ")
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("for __i := int64(0); __i < __e; __i++ { __r *= __b }\n")
	g.writeIndent()
	g.writef("return __r\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinChr lowers `chr(n)` to a single-rune string. Matches Python's
// codepoint-based behavior for the BMP and beyond.
func (g *gen) builtinChr(c *ir.Call) error {
	if len(c.Args) != 1 {
		return fmt.Errorf("chr() takes one positional argument")
	}
	g.writef("string(rune(")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("))")
	return nil
}

// builtinOrd lowers `ord(s)` to the first rune's codepoint as int64.
func (g *gen) builtinOrd(c *ir.Call) error {
	if len(c.Args) != 1 {
		return fmt.Errorf("ord() takes one positional argument")
	}
	g.writef("int64([]rune(")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(")[0])")
	return nil
}

// builtinRepr lowers `repr(x)` to fmt.Sprintf with the %#v verb. This is
// only an approximation of Python's repr — string quotes match, but
// container shapes (`[1, 2]` vs Go's `[]int64{1, 2}`) diverge. Documented
// limitation.
func (g *gen) builtinRepr(c *ir.Call) error {
	if len(c.Args) != 1 {
		return fmt.Errorf("repr() takes one positional argument")
	}
	g.addImport("fmt")
	g.writef("fmt.Sprintf(%q, ", "%#v")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(")")
	return nil
}

// sliceBoundSafe reports whether a slice bound expression is safe to
// emit as a Go-native index — i.e. it's nil (omitted) or a non-negative
// integer literal. Anything else (variables, negative literals,
// arithmetic) needs the helper that wraps negative indices and clamps
// out-of-range values like Python does.
func sliceBoundSafe(e ir.Expr) bool {
	if e == nil {
		return true
	}
	if lit, ok := e.(*ir.IntLit); ok {
		return lit.V >= 0
	}
	return false
}

// sliceWithHelper emits a call to the generic __gopy_slice_T helper for
// the value's element type, threading the (low, high, step) tuple as
// nullable int64 pointers so the helper can apply Python's semantics
// (negative indices, omitted bounds, signed step).
func (g *gen) sliceWithHelper(x *ir.Slice) error {
	containerTy := x.Value.TypeOf()
	if containerTy == nil {
		return fmt.Errorf("slicing: receiver type unknown; add an annotation")
	}
	switch containerTy.Kind {
	case ir.TyList:
		elemGo := g.goType(containerTy.Elem)
		helperKey := "__gopy_slice_" + sanitizeHelper(elemGo)
		g.helpers[helperKey] = sliceHelperFor(elemGo, helperKey)
		g.writef("%s(", helperKey)
	case ir.TyStr:
		g.helpers["__gopy_slice_str"] = sliceStrHelper
		g.writef("__gopy_slice_str(")
	default:
		return fmt.Errorf("slicing: unsupported container type %s", g.goType(containerTy))
	}
	if err := g.expr(x.Value); err != nil {
		return err
	}
	g.writef(", ")
	if err := g.sliceBoundArg(x.Low); err != nil {
		return err
	}
	g.writef(", ")
	if err := g.sliceBoundArg(x.High); err != nil {
		return err
	}
	g.writef(", ")
	if x.Step == nil {
		g.writef("1")
	} else {
		g.writef("int64(")
		if err := g.expr(x.Step); err != nil {
			return err
		}
		g.writef(")")
	}
	g.writef(")")
	return nil
}

func (g *gen) sliceBoundArg(e ir.Expr) error {
	g.helpers["__gopy_slice_some"] = sliceSomeHelper
	if e == nil {
		g.writef("__gopy_slice_none")
		g.helpers["__gopy_slice_none"] = sliceNoneHelper
		return nil
	}
	g.writef("__gopy_slice_some(int64(")
	if err := g.expr(e); err != nil {
		return err
	}
	g.writef("))")
	return nil
}

// sliceHelperFor returns the source of a slice helper specialized for
// the given Go element type. Generated once per element type in use.
func sliceHelperFor(elemGo, helperKey string) string {
	return "func " + helperKey + "(xs []" + elemGo + ", low, high *int64, step int64) []" + elemGo + ` {
	n := int64(len(xs))
	resolve := func(b *int64, fallback int64) int64 {
		if b == nil {
			return fallback
		}
		if *b < 0 {
			r := *b + n
			if r < 0 {
				r = 0
			}
			return r
		}
		if *b > n {
			return n
		}
		return *b
	}
	if step == 0 {
		step = 1
	}
	var lo, hi int64
	if step > 0 {
		lo = resolve(low, 0)
		hi = resolve(high, n)
	} else {
		lo = resolve(low, n-1)
		hi = resolve(high, -1)
	}
	var out []` + elemGo + `
	if step > 0 {
		for i := lo; i < hi; i += step {
			out = append(out, xs[i])
		}
	} else {
		for i := lo; i > hi; i += step {
			if i >= 0 && i < n {
				out = append(out, xs[i])
			}
		}
	}
	if out == nil {
		out = []` + elemGo + `{}
	}
	return out
}`
}

const sliceStrHelper = `func __gopy_slice_str(s string, low, high *int64, step int64) string {
	rs := []rune(s)
	n := int64(len(rs))
	resolve := func(b *int64, fallback int64) int64 {
		if b == nil {
			return fallback
		}
		if *b < 0 {
			r := *b + n
			if r < 0 {
				r = 0
			}
			return r
		}
		if *b > n {
			return n
		}
		return *b
	}
	if step == 0 {
		step = 1
	}
	var lo, hi int64
	if step > 0 {
		lo = resolve(low, 0)
		hi = resolve(high, n)
	} else {
		lo = resolve(low, n-1)
		hi = resolve(high, -1)
	}
	var out []rune
	if step > 0 {
		for i := lo; i < hi; i += step {
			out = append(out, rs[i])
		}
	} else {
		for i := lo; i > hi; i += step {
			if i >= 0 && i < n {
				out = append(out, rs[i])
			}
		}
	}
	return string(out)
}`

const sliceSomeHelper = `func __gopy_slice_some(v int64) *int64 { return &v }`
const sliceNoneHelper = "var __gopy_slice_none *int64 = nil"

// sanitizeHelper turns a Go type expression like `[]int64` or `*Foo`
// into a name fragment safe to splice into an identifier.
func sanitizeHelper(s string) string {
	var b []rune
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_':
			b = append(b, r)
		default:
			b = append(b, '_')
		}
	}
	return string(b)
}

// builtinGetattr emits `__<Class>_get(obj, name)` with an optional
// default value when the field doesn't exist. The class is taken from
// the first argument's effective IR type; getattr on objects with
// unknown class type is rejected at transpile time.
func (g *gen) builtinGetattr(c *ir.Call) error {
	if len(c.Args) < 2 || len(c.Args) > 3 {
		return fmt.Errorf("getattr() takes (obj, name[, default])")
	}
	cls, err := g.lookupUserClass(c.Args[0])
	if err != nil {
		return err
	}
	g.writef("func() any {\n")
	g.indent++
	g.writeIndent()
	g.writef("__v, __ok := __%s_get(", cls.Name)
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(", ")
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef(")\n")
	g.writeIndent()
	g.writef("if __ok { return __v }\n")
	g.writeIndent()
	if len(c.Args) == 3 {
		g.writef("return ")
		if err := g.expr(c.Args[2]); err != nil {
			return err
		}
		g.writef("\n")
	} else {
		g.helpers["__gopy_attr_err"] = `func __gopy_attr_err(name string) { panic(NewException("AttributeError: " + name)) }`
		g.needsException = true
		g.writef("__gopy_attr_err(")
		if err := g.expr(c.Args[1]); err != nil {
			return err
		}
		g.writef("); return nil\n")
	}
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinSetattr emits a call to the class's setter helper. The value
// is wrapped in any to match the helper's signature.
func (g *gen) builtinSetattr(c *ir.Call) error {
	if len(c.Args) != 3 {
		return fmt.Errorf("setattr() takes (obj, name, value)")
	}
	cls, err := g.lookupUserClass(c.Args[0])
	if err != nil {
		return err
	}
	g.writef("__%s_set(", cls.Name)
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(", ")
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef(", ")
	if err := g.boxedExpr(c.Args[2]); err != nil {
		return err
	}
	g.writef(")")
	return nil
}

// builtinHasattr returns true when the class's getter helper reports
// success — the field's value is discarded.
func (g *gen) builtinHasattr(c *ir.Call) error {
	if len(c.Args) != 2 {
		return fmt.Errorf("hasattr() takes (obj, name)")
	}
	cls, err := g.lookupUserClass(c.Args[0])
	if err != nil {
		return err
	}
	g.writef("func() bool { _, __ok := __%s_get(", cls.Name)
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(", ")
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef("); return __ok }()")
	return nil
}

// lookupUserClass returns the class declaration whose instance is
// the given expression. Resolves through effectiveType so locals and
// chained calls work alongside annotated parameters.
func (g *gen) lookupUserClass(e ir.Expr) (*ir.Class, error) {
	ty := g.effectiveType(e)
	if ty == nil || ty.Kind != ir.TyNamed {
		return nil, fmt.Errorf("dynamic-attribute builtin: receiver type must be a known user class")
	}
	cls, ok := g.classes[ty.Name]
	if !ok {
		return nil, fmt.Errorf("dynamic-attribute builtin: %s is not a user class", ty.Name)
	}
	return cls, nil
}

// builtinReduceFn emits a left-fold over the iterable using the inline
// binary lambda. Supports the two- and three-arg forms (initial value
// optional). The lambda body is re-lowered with both param types so
// arithmetic typechecks; result type comes from the body.
func (g *gen) builtinReduceFn(c *ir.Call) error {
	if len(c.Args) != 2 && len(c.Args) != 3 {
		return fmt.Errorf("reduce() takes (fn, iterable[, initializer])")
	}
	lam, ok := c.Args[0].(*ir.Lambda)
	if !ok {
		return fmt.Errorf("reduce(): first argument must be an inline lambda")
	}
	if len(lam.Params) != 2 {
		return fmt.Errorf("reduce(): lambda must take two arguments")
	}
	elem, err := listElemTypeOf(c.Args[1])
	if err != nil {
		return fmt.Errorf("reduce(): %w", err)
	}
	// Re-lower body with (acc, elem) types. Acc type defaults to elem
	// when no initializer; with initializer we'd need the init's type
	// — use elem for the simple case.
	body, err := ir.LowerLambdaBody(lam, []*ir.Type{elem, elem})
	if err != nil {
		return fmt.Errorf("reduce(): %w", err)
	}
	resTy := body.TypeOf()
	if resTy == nil || resTy.Kind == ir.TyUnknown {
		resTy = elem
	}
	resGo := g.goType(resTy)
	accName, elemName := lam.Params[0].Name, lam.Params[1].Name
	g.writef("func() %s {\n", resGo)
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	if len(c.Args) == 3 {
		g.writef("var %s %s = ", accName, resGo)
		if err := g.expr(c.Args[2]); err != nil {
			return err
		}
		g.writef("\n")
		g.writeIndent()
		g.writef("for _, %s := range __src {\n", elemName)
		g.indent++
		g.writeIndent()
		g.writef("_ = %s\n", elemName)
	} else {
		// Two-arg form: seed acc from first element, iterate the rest.
		g.writef("if len(__src) == 0 { panic(\"reduce() of empty sequence with no initial value\") }\n")
		g.writeIndent()
		g.writef("var %s %s = __src[0]\n", accName, resGo)
		g.writeIndent()
		g.writef("for __i := 1; __i < len(__src); __i++ {\n")
		g.indent++
		g.writeIndent()
		g.writef("%s := __src[__i]\n", elemName)
		g.writeIndent()
		g.writef("_ = %s\n", elemName)
	}
	g.writeIndent()
	g.writef("%s = ", accName)
	if err := g.expr(body); err != nil {
		return err
	}
	g.writef("\n")
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return %s\n", accName)
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinVars emits `map[string]any{...}` populated from the instance's
// fields. Same shape as `dataclasses.asdict`, since gopy doesn't have a
// real __dict__ — the static class registry stands in.
func (g *gen) builtinVars(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("vars() takes 1 positional argument")
	}
	cls, err := g.dataclassFor(c.Args[0])
	if err != nil {
		return err
	}
	g.writef("func() map[string]any {\n")
	g.indent++
	g.writeIndent()
	g.writef("__obj := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("return map[string]any{\n")
	g.indent++
	for _, f := range cls.Fields {
		g.writeIndent()
		g.writef("%q: __obj.%s,\n", f.Name, f.Name)
	}
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinDir emits `[]string{...}` listing the instance's field and
// method names. Order: declared fields first, then declared methods.
func (g *gen) builtinDir(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("dir() takes 1 positional argument")
	}
	var cls *ir.Class
	if n, ok := c.Args[0].(*ir.Name); ok {
		if k, isClass := g.classes[n.N]; isClass {
			cls = k
		}
	}
	if cls == nil {
		k, err := g.dataclassFor(c.Args[0])
		if err != nil {
			return err
		}
		cls = k
	}
	g.writef("[]string{")
	first := true
	emit := func(s string) {
		if !first {
			g.writef(", ")
		}
		first = false
		g.writef("%q", s)
	}
	for _, f := range cls.Fields {
		emit(f.Name)
	}
	for _, mname := range cls.MethodNames {
		emit(mname)
	}
	g.writef("}")
	return nil
}

// builtinAscii emits the ASCII-safe repr — string-like with non-ASCII
// characters replaced by `\uXXXX` escapes. Mirrors Python's ascii(s).
func (g *gen) builtinAscii(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("ascii() takes exactly 1 positional argument")
	}
	g.helpers["__gopy_ascii"] = helperGopyAscii
	g.addImport("fmt")
	g.writef("__gopy_ascii(")
	if err := g.boxedExpr(c.Args[0]); err != nil {
		return err
	}
	g.writef(")")
	return nil
}

// builtinCallable resolves `callable(x)` statically when x is a known
// function or class name, otherwise routes through a reflect-based
// helper. Methods on instances and bound methods are not supported.
func (g *gen) builtinCallable(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("callable() takes exactly 1 positional argument")
	}
	if n, ok := c.Args[0].(*ir.Name); ok {
		if _, isFn := g.funcs[n.N]; isFn {
			g.writef("true")
			return nil
		}
		if _, isCls := g.classes[n.N]; isCls {
			g.writef("true")
			return nil
		}
	}
	g.helpers["__gopy_callable"] = helperGopyCallable
	g.addImport("reflect")
	g.writef("__gopy_callable(")
	if err := g.boxedExpr(c.Args[0]); err != nil {
		return err
	}
	g.writef(")")
	return nil
}

// builtinHex / builtinOct / builtinBin mirror Python's prefixed-string
// converters for ints. Negative numbers get a leading minus before the
// prefix (e.g. `-0xff`), matching CPython.
func (g *gen) builtinHex(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("hex() takes exactly 1 positional argument")
	}
	g.helpers["__gopy_hex"] = helperGopyHex
	g.addImport("fmt")
	g.writef("__gopy_hex(")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(")")
	return nil
}

func (g *gen) builtinOct(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("oct() takes exactly 1 positional argument")
	}
	g.helpers["__gopy_oct"] = helperGopyOct
	g.addImport("fmt")
	g.writef("__gopy_oct(")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(")")
	return nil
}

func (g *gen) builtinBin(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("bin() takes exactly 1 positional argument")
	}
	g.helpers["__gopy_bin"] = helperGopyBin
	g.addImport("fmt")
	g.writef("__gopy_bin(")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(")")
	return nil
}

// builtinFormat emits `format(value[, spec])` — single-value formatter.
// Routes through the same __gopy_fmt_spec helper used by f-strings and
// str.format. With no spec, returns the default string representation.
func (g *gen) builtinFormat(c *ir.Call) error {
	if len(c.Keywords) != 0 {
		return fmt.Errorf("format() takes no keyword arguments")
	}
	if len(c.Args) < 1 || len(c.Args) > 2 {
		return fmt.Errorf("format() takes (value[, spec])")
	}
	g.helpers["__gopy_str_format"] = helperStrFormat
	g.addImport("strings")
	g.addImport("fmt")
	g.writef("__gopy_fmt_spec(")
	if len(c.Args) == 2 {
		if err := g.expr(c.Args[1]); err != nil {
			return err
		}
	} else {
		g.writef(`""`)
	}
	g.writef(", ")
	if err := g.boxedExpr(c.Args[0]); err != nil {
		return err
	}
	g.writef(")")
	return nil
}

// builtinType emits `__gopy_type(x)` which returns the Python class name
// of x as a string. Mirrors `type(x).__name__` for ordinary primitives and
// user classes; accessing `.__name__` on the result is a no-op (see the
// Attribute codegen). Full class-handle comparison (`type(x) is int`) is
// not supported.
// builtinComplex emits Go's complex(real, imag) builtin. Both args are
// cast to float64 so int / mixed-type calls match CPython's coercion
// rules. Zero / one-arg forms supply a 0.0 imaginary part to match
// `complex(2.5)` → `(2.5+0j)`.
func (g *gen) builtinComplex(c *ir.Call) error {
	if len(c.Keywords) != 0 {
		return fmt.Errorf("complex() takes no keyword arguments")
	}
	if len(c.Args) > 2 {
		return fmt.Errorf("complex() takes at most 2 arguments")
	}
	g.writef("complex(")
	if len(c.Args) >= 1 {
		g.writef("float64(")
		if err := g.expr(c.Args[0]); err != nil {
			return err
		}
		g.writef(")")
	} else {
		g.writef("float64(0)")
	}
	g.writef(", ")
	if len(c.Args) == 2 {
		g.writef("float64(")
		if err := g.expr(c.Args[1]); err != nil {
			return err
		}
		g.writef(")")
	} else {
		g.writef("float64(0)")
	}
	g.writef(")")
	return nil
}

// builtinEnumerate emits a standalone `enumerate(xs[, start])` as an IIFE
// returning `[][]any` of [index, value] pairs. Matches gopy's tuple
// convention (slice of any). The 2-arg form starts the index at the
// given offset; defaults to 0 like CPython.
func (g *gen) builtinEnumerate(c *ir.Call) error {
	if len(c.Args) < 1 || len(c.Args) > 2 {
		return fmt.Errorf("enumerate() takes 1 or 2 arguments")
	}
	g.writef("func() [][]any {\n")
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__start := int64(0)\n")
	if len(c.Args) == 2 {
		g.writeIndent()
		g.writef("__start = ")
		if err := g.expr(c.Args[1]); err != nil {
			return err
		}
		g.writef("\n")
	}
	g.writeIndent()
	g.writef("__out := make([][]any, 0, len(__src))\n")
	g.writeIndent()
	g.writef("for __i, __v := range __src {\n")
	g.indent++
	g.writeIndent()
	g.writef("__out = append(__out, []any{__start + int64(__i), __v})\n")
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinZipPairs emits standalone `zip(a, b)` as an IIFE returning
// `[][]any` of paired elements, stopping at the shorter input. Matches
// CPython's strict=False default.
func (g *gen) builtinZipPairs(c *ir.Call) error {
	if len(c.Args) != 2 {
		return fmt.Errorf("zip() takes exactly 2 iterables (more arities not yet supported)")
	}
	g.writef("func() [][]any {\n")
	g.indent++
	g.writeIndent()
	g.writef("__a := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__b := ")
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__n := len(__a)\n")
	g.writeIndent()
	g.writef("if len(__b) < __n { __n = len(__b) }\n")
	g.writeIndent()
	g.writef("__out := make([][]any, 0, __n)\n")
	g.writeIndent()
	g.writef("for __i := 0; __i < __n; __i++ {\n")
	g.indent++
	g.writeIndent()
	g.writef("__out = append(__out, []any{__a[__i], __b[__i]})\n")
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

func (g *gen) builtinType(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("type() takes exactly 1 positional argument")
	}
	g.addImport("fmt")
	g.helpers["__gopy_type"] = helperGopyType
	g.writef("__gopy_type(")
	if err := g.boxedExpr(c.Args[0]); err != nil {
		return err
	}
	g.writef(")")
	return nil
}

// builtinSet handles both `set(iter)` and `frozenset(iter)`. Returns a
// deduplicated slice preserving insertion order. Python sets are unordered
// but most fixtures sort the result before printing, so this is a safe
// approximation that lets `in` / iteration work over the value.
func (g *gen) builtinSet(c *ir.Call) error {
	if len(c.Keywords) != 0 {
		return fmt.Errorf("set()/frozenset() take no keyword arguments")
	}
	if len(c.Args) == 0 {
		// `set()` with no args: caller needs an explicit target type.
		// Without one we can't pick an element type — fall back to []any.
		g.writef("[]any{}")
		return nil
	}
	if len(c.Args) != 1 {
		return fmt.Errorf("set()/frozenset() take at most 1 argument")
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("set(): %w", err)
	}
	elemGo := g.goType(elem)
	g.writef("func() []%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__seen := map[%s]bool{}\n", elemGo)
	g.writeIndent()
	g.writef("__out := []%s{}\n", elemGo)
	g.writeIndent()
	g.writef("for _, __v := range __src { if !__seen[__v] { __seen[__v] = true; __out = append(__out, __v) } }\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinTimedelta resolves Python's `timedelta(days, seconds, ...)`
// constructor against the 7 named parameters and emits a call to
// __gopy_timedelta_new with each slot filled (0 default). Both positional
// and keyword forms are supported, mixed with defaults.
func (g *gen) builtinTimedelta(c *ir.Call) error {
	g.addImport("time")
	g.helpers["__gopy_timedelta_new"] = helperTimedeltaNew
	g.helpers["__Timedelta"] = helperTimedeltaType
	g.addImport("fmt")
	params := []string{"days", "seconds", "microseconds", "milliseconds", "minutes", "hours", "weeks"}
	values := make([]ir.Expr, len(params))
	if len(c.Args) > len(params) {
		return fmt.Errorf("timedelta() takes at most 7 positional arguments")
	}
	for i, a := range c.Args {
		values[i] = a
	}
	for _, kw := range c.Keywords {
		idx := -1
		for i, p := range params {
			if p == kw.Name {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fmt.Errorf("timedelta(): unknown keyword %q", kw.Name)
		}
		if values[idx] != nil {
			return fmt.Errorf("timedelta(): duplicate keyword %q (also passed positionally)", kw.Name)
		}
		values[idx] = kw.Value
	}
	g.writef("__gopy_timedelta_new(")
	for i, v := range values {
		if i > 0 {
			g.writef(", ")
		}
		if v == nil {
			g.writef("0")
			continue
		}
		// Cast int operands to float64 so the variadic helper signature
		// stays consistent.
		t := v.TypeOf()
		needCast := t == nil || t.Kind == ir.TyInt
		if needCast {
			g.writef("float64(")
		}
		if err := g.expr(v); err != nil {
			return err
		}
		if needCast {
			g.writef(")")
		}
	}
	g.writef(")")
	return nil
}

// builtinRandomChoice emits an IIFE that picks a random element from
// the typed slice. Panics IndexError on empty input.
func (g *gen) builtinRandomChoice(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("random.choice() takes one positional argument")
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("random.choice(): %w", err)
	}
	elemGo := g.goType(elem)
	g.addImport("math/rand")
	g.needsException = true
	g.writef("func() %s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("if len(__src) == 0 { panic(NewException(\"IndexError: cannot choose from empty sequence\")) }\n")
	g.writeIndent()
	g.writef("return __src[rand.Intn(len(__src))]\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinRandomShuffle emits an in-place Fisher-Yates shuffle. Like the
// other list mutators, the receiver must be an addressable expression.
func (g *gen) builtinRandomShuffle(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("random.shuffle() takes one positional argument")
	}
	if _, err := listElemTypeOf(c.Args[0]); err != nil {
		return fmt.Errorf("random.shuffle(): %w", err)
	}
	g.addImport("math/rand")
	g.writef("for __i := len(")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(") - 1; __i > 0; __i-- {\n")
	g.indent++
	g.writeIndent()
	g.writef("__j := rand.Intn(__i + 1)\n")
	g.writeIndent()
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__i], ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__j] = ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__j], ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__i]\n")
	g.indent--
	g.writeIndent()
	g.writef("}")
	return nil
}

// builtinRandomSample emits k unique elements drawn from the input list.
// Copies the slice, partial-shuffles, returns the first k.
func (g *gen) builtinRandomSample(c *ir.Call) error {
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("random.sample() takes (population, k)")
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("random.sample(): %w", err)
	}
	elemGo := g.goType(elem)
	g.addImport("math/rand")
	g.needsException = true
	g.writef("func() []%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__k := int(")
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef(")\n")
	g.writeIndent()
	g.writef("if __k < 0 || __k > len(__src) { panic(NewException(\"ValueError: sample larger than population\")) }\n")
	g.writeIndent()
	g.writef("__cp := make([]%s, len(__src))\n", elemGo)
	g.writeIndent()
	g.writef("copy(__cp, __src)\n")
	g.writeIndent()
	g.writef("for __i := len(__cp) - 1; __i > len(__cp)-__k-1 && __i > 0; __i-- {\n")
	g.indent++
	g.writeIndent()
	g.writef("__j := rand.Intn(__i + 1)\n")
	g.writeIndent()
	g.writef("__cp[__i], __cp[__j] = __cp[__j], __cp[__i]\n")
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __cp[len(__cp)-__k:]\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinFields emits a `[]string` of field names declared on the class.
// CPython returns a tuple of Field objects with `.name` attribute; gopy
// returns the names directly so iteration patterns like `for f in fields(C)`
// stay simple. Accepts a class name or an instance.
func (g *gen) builtinFields(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("dataclasses.fields() takes 1 positional argument")
	}
	var cls *ir.Class
	if n, ok := c.Args[0].(*ir.Name); ok {
		if k, isClass := g.classes[n.N]; isClass {
			cls = k
		}
	}
	if cls == nil {
		k, err := g.dataclassFor(c.Args[0])
		if err != nil {
			return err
		}
		cls = k
	}
	g.writef("[]string{")
	for i, f := range cls.Fields {
		if i > 0 {
			g.writef(", ")
		}
		g.writef("%q", f.Name)
	}
	g.writef("}")
	return nil
}

// builtinAsdict emits a `map[string]any{...}` populated from the
// instance's typed fields. Requires the receiver to be a Name with a
// known user-class type (or any expression whose effectiveType is TyNamed
// and resolves to a registered class).
func (g *gen) builtinAsdict(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("dataclasses.asdict() takes 1 positional argument")
	}
	cls, err := g.dataclassFor(c.Args[0])
	if err != nil {
		return err
	}
	g.writef("func() map[string]any {\n")
	g.indent++
	g.writeIndent()
	g.writef("__obj := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("return map[string]any{\n")
	g.indent++
	for _, f := range cls.Fields {
		g.writeIndent()
		g.writef("%q: __obj.%s,\n", f.Name, exportedField(f.Name))
	}
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinAstuple emits an `[]any{...}` with the instance's fields in
// declaration order.
func (g *gen) builtinAstuple(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("dataclasses.astuple() takes 1 positional argument")
	}
	cls, err := g.dataclassFor(c.Args[0])
	if err != nil {
		return err
	}
	g.writef("func() []any {\n")
	g.indent++
	g.writeIndent()
	g.writef("__obj := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("return []any{")
	for i, f := range cls.Fields {
		if i > 0 {
			g.writef(", ")
		}
		g.writef("__obj.%s", exportedField(f.Name))
	}
	g.writef("}\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinReplace emits a fresh constructor call seeded from the existing
// instance, overriding any fields listed in the kwargs.
func (g *gen) builtinReplace(c *ir.Call) error {
	if len(c.Args) != 1 {
		return fmt.Errorf("dataclasses.replace() takes (instance, **kwargs)")
	}
	cls, err := g.dataclassFor(c.Args[0])
	if err != nil {
		return err
	}
	kw := map[string]ir.Expr{}
	for _, k := range c.Keywords {
		kw[k.Name] = k.Value
	}
	g.writef("func() *%s {\n", cls.Name)
	g.indent++
	g.writeIndent()
	g.writef("__obj := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("return New%s(", cls.Name)
	for i, f := range cls.Fields {
		if i > 0 {
			g.writef(", ")
		}
		if v, ok := kw[f.Name]; ok {
			if err := g.expr(v); err != nil {
				return err
			}
		} else {
			g.writef("__obj.%s", exportedField(f.Name))
		}
	}
	g.writef(")\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// dataclassFor resolves an expression to its underlying user-class decl.
// Returns an error when the expression has no statically-known TyNamed
// type or the class isn't registered.
func (g *gen) dataclassFor(e ir.Expr) (*ir.Class, error) {
	t := g.effectiveType(e)
	if t == nil || t.Kind != ir.TyNamed {
		return nil, fmt.Errorf("dataclasses helper: receiver type unknown — add a class annotation")
	}
	cls, ok := g.classes[t.Name]
	if !ok {
		return nil, fmt.Errorf("dataclasses helper: %s is not a registered class", t.Name)
	}
	return cls, nil
}

// exportedField returns the Go field name for a Python attribute. gopy
// emits struct fields with the original Python casing, so this is the
// identity function — wrapped to make intent explicit at call sites.
func exportedField(name string) string { return name }

// opDunderName maps a Python binary operator to the dunder method name
// CPython would dispatch through. Empty string when the operator has no
// dunder analogue we surface.
func opDunderName(op string) string {
	switch op {
	case "+":
		return "__add__"
	case "-":
		return "__sub__"
	case "*":
		return "__mul__"
	case "/":
		return "__truediv__"
	case "//":
		return "__floordiv__"
	case "%":
		return "__mod__"
	case "**":
		return "__pow__"
	case "<":
		return "__lt__"
	case "<=":
		return "__le__"
	case ">":
		return "__gt__"
	case ">=":
		return "__ge__"
	case "==":
		return "__eq__"
	case "!=":
		return "__ne__"
	case "|":
		return "__or__"
	case "&":
		return "__and__"
	case "^":
		return "__xor__"
	case "<<":
		return "__lshift__"
	case ">>":
		return "__rshift__"
	case "@":
		return "__matmul__"
	}
	return ""
}

// iopDunderName maps a binary operator to the in-place dunder method
// CPython dispatches through during augmented assignment (`x += y` →
// `__iadd__`). Empty string when no in-place form exists.
func iopDunderName(op string) string {
	switch op {
	case "+":
		return "__iadd__"
	case "-":
		return "__isub__"
	case "*":
		return "__imul__"
	case "/":
		return "__itruediv__"
	case "//":
		return "__ifloordiv__"
	case "%":
		return "__imod__"
	case "**":
		return "__ipow__"
	case "|":
		return "__ior__"
	case "&":
		return "__iand__"
	case "^":
		return "__ixor__"
	case "<<":
		return "__ilshift__"
	case ">>":
		return "__irshift__"
	case "@":
		return "__imatmul__"
	}
	return ""
}

// exportedDunder returns the Go method name gopy emits for a Python
// dunder method. Matches the renames performed at method-def time.
func exportedDunder(name string) string {
	switch name {
	case "__add__":
		return "Add"
	case "__sub__":
		return "Sub"
	case "__mul__":
		return "Mul"
	case "__truediv__":
		return "Truediv"
	case "__floordiv__":
		return "Floordiv"
	case "__mod__":
		return "Mod"
	case "__pow__":
		return "Pow"
	case "__lt__":
		return "Lt"
	case "__le__":
		return "Le"
	case "__gt__":
		return "Gt"
	case "__ge__":
		return "Ge"
	case "__eq__":
		return "Eq"
	case "__ne__":
		return "Ne"
	case "__contains__":
		return "Contains"
	case "__getitem__":
		return "Getitem"
	case "__setitem__":
		return "Setitem"
	case "__bool__":
		return "Bool"
	case "__iter__":
		return "Iter"
	case "__next__":
		return "Next"
	case "__abs__":
		return "Abs"
	case "__neg__":
		return "Neg"
	case "__pos__":
		return "Pos"
	case "__int__":
		return "Int"
	case "__float__":
		return "Float"
	case "__reversed__":
		return "Reversed"
	case "__call__":
		return "Call"
	case "__or__":
		return "Or"
	case "__and__":
		return "And"
	case "__xor__":
		return "Xor"
	case "__lshift__":
		return "Lshift"
	case "__rshift__":
		return "Rshift"
	case "__matmul__":
		return "Matmul"
	case "__invert__":
		return "Invert"
	case "__enter__":
		return "Enter"
	case "__exit__":
		return "Exit"
	case "__iadd__":
		return "Iadd"
	case "__isub__":
		return "Isub"
	case "__imul__":
		return "Imul"
	case "__itruediv__":
		return "Itruediv"
	case "__ifloordiv__":
		return "Ifloordiv"
	case "__imod__":
		return "Imod"
	case "__ipow__":
		return "Ipow"
	case "__ior__":
		return "Ior"
	case "__iand__":
		return "Iand"
	case "__ixor__":
		return "Ixor"
	case "__ilshift__":
		return "Ilshift"
	case "__irshift__":
		return "Irshift"
	case "__imatmul__":
		return "Imatmul"
	case "__format__":
		return "Format"
	case "__round__":
		return "Round"
	case "__ceil__":
		return "Ceil"
	case "__floor__":
		return "Floor"
	case "__trunc__":
		return "Trunc"
	}
	return name
}

// builtinJSONDumps emits a json.dumps call optionally honoring the
// indent= kwarg. Other kwargs (sort_keys, separators, default) are not
// supported yet — they error at the call site so users notice rather
// than silently get the default formatting.
func (g *gen) builtinJSONDumps(c *ir.Call) error {
	if len(c.Args) != 1 {
		return fmt.Errorf("json.dumps() takes 1 positional argument")
	}
	var indent ir.Expr
	for _, kw := range c.Keywords {
		if kw.Name != "indent" {
			return fmt.Errorf("json.dumps(): unsupported keyword %q", kw.Name)
		}
		indent = kw.Value
	}
	g.addImport("encoding/json")
	g.addImport("strings")
	g.helpers["__gopy_json_dumps"] = helperJSONDumps
	g.writef("__gopy_json_dumps(")
	if err := g.boxedExpr(c.Args[0]); err != nil {
		return err
	}
	if indent != nil {
		g.writef(", int64(")
		if err := g.expr(indent); err != nil {
			return err
		}
		g.writef(")")
	}
	g.writef(")")
	return nil
}

// builtinChainFromIterable emits `chain.from_iterable(xs)` — flattens one
// level. Input must be a list-of-lists with a known inner element type.
func (g *gen) builtinChainFromIterable(m *ir.MethodCall) error {
	if len(m.Args) != 1 || len(m.Keywords) != 0 {
		return fmt.Errorf("chain.from_iterable() takes one iterable")
	}
	outer, err := listElemTypeOf(m.Args[0])
	if err != nil {
		return fmt.Errorf("chain.from_iterable(): %w", err)
	}
	if outer.Kind != ir.TyList {
		return fmt.Errorf("chain.from_iterable(): outer iterable must be a list of lists")
	}
	innerGo := g.goType(outer.Elem)
	g.writef("func() []%s {\n", innerGo)
	g.indent++
	g.writeIndent()
	g.writef("__out := []%s{}\n", innerGo)
	g.writeIndent()
	g.writef("for _, __sub := range ")
	if err := g.expr(m.Args[0]); err != nil {
		return err
	}
	g.writef(" { __out = append(__out, __sub...) }\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinDatetimeReplace emits a fresh *__Datetime built from the receiver
// with year / month / day / hour / minute / second overridden by kwargs.
// Unrecognized kwargs error rather than silently drop.
func (g *gen) builtinDatetimeReplace(m *ir.MethodCall) error {
	if len(m.Args) != 0 {
		return fmt.Errorf("datetime.replace() takes keyword arguments only")
	}
	overrides := map[string]ir.Expr{}
	for _, kw := range m.Keywords {
		switch kw.Name {
		case "year", "month", "day", "hour", "minute", "second":
			overrides[kw.Name] = kw.Value
		default:
			return fmt.Errorf("datetime.replace(): unsupported keyword %q", kw.Name)
		}
	}
	g.writef("func() *__Datetime {\n")
	g.indent++
	g.writeIndent()
	g.writef("__old := ")
	if err := g.expr(m.Recv); err != nil {
		return err
	}
	g.writef(".t\n")
	emit := func(name, defExpr string) error {
		g.writeIndent()
		if v, ok := overrides[name]; ok {
			g.writef("__%s := int(", name)
			if err := g.expr(v); err != nil {
				return err
			}
			g.writef(")\n")
		} else {
			g.writef("__%s := %s\n", name, defExpr)
		}
		return nil
	}
	if err := emit("year", "__old.Year()"); err != nil {
		return err
	}
	if err := emit("month", "int(__old.Month())"); err != nil {
		return err
	}
	if err := emit("day", "__old.Day()"); err != nil {
		return err
	}
	if err := emit("hour", "__old.Hour()"); err != nil {
		return err
	}
	if err := emit("minute", "__old.Minute()"); err != nil {
		return err
	}
	if err := emit("second", "__old.Second()"); err != nil {
		return err
	}
	g.writeIndent()
	g.addImport("time")
	g.writef("return &__Datetime{t: time.Date(__year, time.Month(__month), __day, __hour, __minute, __second, __old.Nanosecond(), __old.Location())}\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinDateReplace mirrors builtinDatetimeReplace for *__Date.
func (g *gen) builtinDateReplace(m *ir.MethodCall) error {
	if len(m.Args) != 0 {
		return fmt.Errorf("date.replace() takes keyword arguments only")
	}
	overrides := map[string]ir.Expr{}
	for _, kw := range m.Keywords {
		switch kw.Name {
		case "year", "month", "day":
			overrides[kw.Name] = kw.Value
		default:
			return fmt.Errorf("date.replace(): unsupported keyword %q", kw.Name)
		}
	}
	g.writef("func() *__Date {\n")
	g.indent++
	g.writeIndent()
	g.writef("__old := ")
	if err := g.expr(m.Recv); err != nil {
		return err
	}
	g.writef("\n")
	emit := func(name, defExpr string) error {
		g.writeIndent()
		if v, ok := overrides[name]; ok {
			g.writef("__%s := int64(", name)
			if err := g.expr(v); err != nil {
				return err
			}
			g.writef(")\n")
		} else {
			g.writef("__%s := %s\n", name, defExpr)
		}
		return nil
	}
	if err := emit("year", "__old.Y"); err != nil {
		return err
	}
	if err := emit("month", "__old.M"); err != nil {
		return err
	}
	if err := emit("day", "__old.D"); err != nil {
		return err
	}
	g.writeIndent()
	g.writef("return &__Date{Y: __year, M: __month, D: __day}\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinStrMaketrans emits a `str.maketrans(...)` call producing a
// map[rune]string. Supports both 2-arg (from, to) and 3-arg (from, to,
// delete) forms — the from/to must be string literals or string-typed
// expressions of the same length.
func (g *gen) builtinStrMaketrans(m *ir.MethodCall) error {
	if len(m.Args) < 1 || len(m.Args) > 3 || len(m.Keywords) != 0 {
		return fmt.Errorf("str.maketrans() takes (from, to[, delete])")
	}
	g.helpers["__gopy_str_maketrans"] = helperStrMaketrans
	g.writef("__gopy_str_maketrans(")
	if err := g.expr(m.Args[0]); err != nil {
		return err
	}
	if len(m.Args) >= 2 {
		g.writef(", ")
		if err := g.expr(m.Args[1]); err != nil {
			return err
		}
	} else {
		g.writef(", \"\"")
	}
	if len(m.Args) == 3 {
		g.writef(", ")
		if err := g.expr(m.Args[2]); err != nil {
			return err
		}
	}
	g.writef(")")
	return nil
}

// builtinDictFromkeys emits `dict.fromkeys(iter, value)` as a typed
// map literal built from the iterable's elements paired with value.
// One-arg form defaults value to None (mapped to int64(0) for typed
// dicts; users that want None should pass an explicit None literal).
func (g *gen) builtinDictFromkeys(m *ir.MethodCall) error {
	if len(m.Args) < 1 || len(m.Args) > 2 || len(m.Keywords) != 0 {
		return fmt.Errorf("dict.fromkeys() takes (iterable[, value])")
	}
	elem, err := listElemTypeOf(m.Args[0])
	if err != nil {
		return fmt.Errorf("dict.fromkeys(): %w", err)
	}
	keyGo := g.goType(elem)
	var valTy *ir.Type
	if len(m.Args) == 2 {
		valTy = m.Args[1].TypeOf()
	}
	if valTy == nil || valTy.Kind == ir.TyUnknown || valTy.Kind == ir.TyNone {
		valTy = &ir.Type{Kind: ir.TyAny}
	}
	valGo := g.goType(valTy)
	g.writef("func() map[%s]%s {\n", keyGo, valGo)
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(m.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__out := make(map[%s]%s, len(__src))\n", keyGo, valGo)
	g.writeIndent()
	if len(m.Args) == 2 {
		g.writef("var __v %s = ", valGo)
		if err := g.expr(m.Args[1]); err != nil {
			return err
		}
		g.writef("\n")
		g.writeIndent()
		g.writef("for _, __k := range __src { __out[__k] = __v }\n")
	} else {
		g.writef("var __v %s\n", valGo)
		g.writeIndent()
		g.writef("for _, __k := range __src { __out[__k] = __v }\n")
	}
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinZipLongest emits a list of 2-element pairs from two iterables.
// Shorter sequence padded with the fillvalue kwarg (or the element type's
// zero value when fillvalue is absent). Returned shape matches starmap's
// expected pair-list input.
// builtinPairwise emits `itertools.pairwise(xs)` as an eager `[][]any`
// of (x[i], x[i+1]) pairs. Result is empty for sequences of length < 2.
func (g *gen) builtinPairwise(c *ir.Call) error {
	if len(c.Args) != 1 {
		return fmt.Errorf("pairwise() takes 1 argument")
	}
	g.writef("func() [][]any {\n")
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__out := make([][]any, 0)\n")
	g.writeIndent()
	g.writef("for __i := 0; __i+1 < len(__src); __i++ {\n")
	g.indent++
	g.writeIndent()
	g.writef("__out = append(__out, []any{__src[__i], __src[__i+1]})\n")
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinBatched emits `itertools.batched(xs, n)` as an eager `[][]any`
// of n-sized chunks (last chunk may be shorter). n must be >= 1.
func (g *gen) builtinBatched(c *ir.Call) error {
	if len(c.Args) != 2 {
		return fmt.Errorf("batched() takes (iterable, n)")
	}
	g.needsException = true
	g.writef("func() [][]any {\n")
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__n := int(")
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef(")\n")
	g.writeIndent()
	g.writef("if __n < 1 { panic(NewException(\"ValueError: batched(): n must be >= 1\")) }\n")
	g.writeIndent()
	g.writef("__out := make([][]any, 0)\n")
	g.writeIndent()
	g.writef("for __i := 0; __i < len(__src); __i += __n {\n")
	g.indent++
	g.writeIndent()
	g.writef("__end := __i + __n\n")
	g.writeIndent()
	g.writef("if __end > len(__src) { __end = len(__src) }\n")
	g.writeIndent()
	g.writef("__chunk := make([]any, 0, __end-__i)\n")
	g.writeIndent()
	g.writef("for __j := __i; __j < __end; __j++ { __chunk = append(__chunk, __src[__j]) }\n")
	g.writeIndent()
	g.writef("__out = append(__out, __chunk)\n")
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

func (g *gen) builtinZipLongest(c *ir.Call) error {
	if len(c.Args) != 2 {
		return fmt.Errorf("zip_longest() takes (a, b); fillvalue is a kwarg")
	}
	var fill ir.Expr
	for _, kw := range c.Keywords {
		if kw.Name != "fillvalue" {
			return fmt.Errorf("zip_longest(): unknown keyword %q", kw.Name)
		}
		fill = kw.Value
	}
	elemA, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("zip_longest(): %w", err)
	}
	elemB, err := listElemTypeOf(c.Args[1])
	if err != nil {
		return fmt.Errorf("zip_longest(): %w", err)
	}
	// Both sides must share element type so the pair-list has a single
	// element type. Mismatched types degrade to any.
	elem := elemA
	if elemA == nil || elemB == nil || elemA.Kind != elemB.Kind {
		elem = &ir.Type{Kind: ir.TyAny}
	}
	elemGo := g.goType(elem)
	g.writef("func() [][]%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__a := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__b := ")
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__n := len(__a)\n")
	g.writeIndent()
	g.writef("if len(__b) > __n { __n = len(__b) }\n")
	g.writeIndent()
	g.writef("var __fill %s\n", elemGo)
	if fill != nil {
		g.writeIndent()
		g.writef("__fill = ")
		if err := g.expr(fill); err != nil {
			return err
		}
		g.writef("\n")
	}
	g.writeIndent()
	g.writef("__out := make([][]%s, 0, __n)\n", elemGo)
	g.writeIndent()
	g.writef("for __i := 0; __i < __n; __i++ {\n")
	g.indent++
	g.writeIndent()
	g.writef("__pa := __fill\n")
	g.writeIndent()
	g.writef("__pb := __fill\n")
	g.writeIndent()
	g.writef("if __i < len(__a) { __pa = __a[__i] }\n")
	g.writeIndent()
	g.writef("if __i < len(__b) { __pb = __b[__i] }\n")
	g.writeIndent()
	g.writef("__out = append(__out, []%s{__pa, __pb})\n", elemGo)
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinFilterfalse emits filter(not pred(x), xs) — keeps elements for
// which the lambda predicate returns false. Lambda re-lowered with the
// iterable's element type.
func (g *gen) builtinFilterfalse(c *ir.Call) error {
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("filterfalse() takes (predicate, iterable)")
	}
	lam, ok := c.Args[0].(*ir.Lambda)
	if !ok {
		return fmt.Errorf("filterfalse(): first arg must be an inline lambda")
	}
	if len(lam.Params) != 1 {
		return fmt.Errorf("filterfalse(): lambda must take 1 argument")
	}
	elem, err := listElemTypeOf(c.Args[1])
	if err != nil {
		return fmt.Errorf("filterfalse(): %w", err)
	}
	body, err := ir.LowerLambdaBody(lam, []*ir.Type{elem})
	if err != nil {
		return fmt.Errorf("filterfalse(): %w", err)
	}
	elemGo := g.goType(elem)
	g.writef("func() []%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__out := []%s{}\n", elemGo)
	g.writeIndent()
	g.writef("for _, %s := range ", lam.Params[0].Name)
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef(" {\n")
	g.indent++
	g.writeIndent()
	g.writef("if !(")
	if err := g.expr(body); err != nil {
		return err
	}
	g.writef(") { __out = append(__out, %s) }\n", lam.Params[0].Name)
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinCompress emits Python's compress(data, selectors) — keeps each
// data[i] whose corresponding selectors[i] is truthy.
func (g *gen) builtinCompress(c *ir.Call) error {
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("compress() takes (data, selectors)")
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("compress(): %w", err)
	}
	selElem, err := listElemTypeOf(c.Args[1])
	if err != nil {
		return fmt.Errorf("compress(): %w", err)
	}
	elemGo := g.goType(elem)
	g.writef("func() []%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__data := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__sel := ")
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__n := len(__data)\n")
	g.writeIndent()
	g.writef("if len(__sel) < __n { __n = len(__sel) }\n")
	g.writeIndent()
	g.writef("__out := []%s{}\n", elemGo)
	g.writeIndent()
	g.writef("for __i := 0; __i < __n; __i++ {\n")
	g.indent++
	g.writeIndent()
	switch selElem.Kind {
	case ir.TyBool:
		g.writef("if __sel[__i] { __out = append(__out, __data[__i]) }\n")
	case ir.TyInt:
		g.writef("if __sel[__i] != 0 { __out = append(__out, __data[__i]) }\n")
	default:
		g.writef("if __sel[__i] != 0 { __out = append(__out, __data[__i]) }\n")
	}
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinCount emits a bounded form of itertools.count: count(start, step, n).
// CPython's count is infinite; gopy requires an explicit `n` element limit
// (passed as the third positional arg) since consumers we support all
// materialize.
func (g *gen) builtinCount(c *ir.Call) error {
	if len(c.Args) < 1 || len(c.Args) > 3 || len(c.Keywords) != 0 {
		return fmt.Errorf("count() takes (start, step, n); gopy requires explicit n")
	}
	var startE, stepE, nE ir.Expr
	startE = c.Args[0]
	if len(c.Args) >= 2 {
		stepE = c.Args[1]
	}
	if len(c.Args) == 3 {
		nE = c.Args[2]
	}
	if nE == nil {
		return fmt.Errorf("count(): unbounded form not supported — pass n as the third arg")
	}
	// Pick int64 / float64 from start's type.
	tStart := startE.TypeOf()
	if tStart == nil || (tStart.Kind != ir.TyInt && tStart.Kind != ir.TyFloat) {
		return fmt.Errorf("count(): start must be int or float")
	}
	elemGo := g.goType(tStart)
	g.writef("func() []%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("var __start %s = ", elemGo)
	if err := g.expr(startE); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("var __step %s = ", elemGo)
	if stepE != nil {
		if err := g.expr(stepE); err != nil {
			return err
		}
	} else {
		g.writef("1")
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__n := int(")
	if err := g.expr(nE); err != nil {
		return err
	}
	g.writef(")\n")
	g.writeIndent()
	g.writef("if __n < 0 { __n = 0 }\n")
	g.writeIndent()
	g.writef("__out := make([]%s, __n)\n", elemGo)
	g.writeIndent()
	g.writef("__v := __start\n")
	g.writeIndent()
	g.writef("for __i := 0; __i < __n; __i++ { __out[__i] = __v; __v += __step }\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinNsmallest emits an IIFE that sorts a copy of the iterable and
// returns the first n elements. `largest=true` reverses the sort to give
// the n largest. Matches CPython's heapq.nsmallest / nlargest output for
// typed int/float/str slices.
// builtinHeapqMerge emits `heapq.merge(a, b)` for two pre-sorted slices
// as a merged-output `[]any`. CPython supports N inputs and a key=; gopy
// supports 2 unkeyed inputs and uses a generic less-than via fmt-string
// compare when types disagree. The result is eagerly materialized.
func (g *gen) builtinHeapqMerge(c *ir.Call) error {
	if len(c.Args) != 2 {
		return fmt.Errorf("heapq.merge() takes (a, b)")
	}
	g.writef("func() []any {\n")
	g.indent++
	g.writeIndent()
	g.writef("__a := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__b := ")
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__out := make([]any, 0, len(__a)+len(__b))\n")
	g.writeIndent()
	g.writef("__i, __j := 0, 0\n")
	g.writeIndent()
	g.addImport("fmt")
	g.writef("for __i < len(__a) && __j < len(__b) {\n")
	g.indent++
	g.writeIndent()
	g.writef("if fmt.Sprintf(\"%%v\", __a[__i]) <= fmt.Sprintf(\"%%v\", __b[__j]) { __out = append(__out, __a[__i]); __i++ } else { __out = append(__out, __b[__j]); __j++ }\n")
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("for __i < len(__a) { __out = append(__out, __a[__i]); __i++ }\n")
	g.writeIndent()
	g.writef("for __j < len(__b) { __out = append(__out, __b[__j]); __j++ }\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

func (g *gen) builtinNsmallest(c *ir.Call, largest bool) error {
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("heapq.%s() takes (n, iterable)", map[bool]string{false: "nsmallest", true: "nlargest"}[largest])
	}
	elem, err := listElemTypeOf(c.Args[1])
	if err != nil {
		return fmt.Errorf("heapq.nsmallest(): %w", err)
	}
	if !heapOrderable(elem) {
		return fmt.Errorf("heapq.nsmallest(): element type must be int/float/str")
	}
	elemGo := g.goType(elem)
	g.addImport("sort")
	g.writef("func() []%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__n := int(")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(")\n")
	g.writeIndent()
	g.writef("if __n < 0 { __n = 0 }\n")
	g.writeIndent()
	g.writef("if __n > len(__src) { __n = len(__src) }\n")
	g.writeIndent()
	g.writef("__cp := make([]%s, len(__src))\n", elemGo)
	g.writeIndent()
	g.writef("copy(__cp, __src)\n")
	g.writeIndent()
	op := "<"
	if largest {
		op = ">"
	}
	g.writef("sort.Slice(__cp, func(i, j int) bool { return __cp[i] %s __cp[j] })\n", op)
	g.writeIndent()
	g.writef("return __cp[:__n]\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinHeappush emits an inline min-heap push: appends, then sifts up.
// The receiver must be an addressable typed list whose element type is
// comparable with `<` (int / float / str).
func (g *gen) builtinHeappush(c *ir.Call) error {
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("heapq.heappush() takes (heap, item)")
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("heapq.heappush(): %w", err)
	}
	if !heapOrderable(elem) {
		return fmt.Errorf("heapq.heappush(): element type must be int/float/str")
	}
	g.writef("func() {\n")
	g.indent++
	g.writeIndent()
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(" = append(")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(", ")
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef(")\n")
	g.writeIndent()
	g.writef("__i := len(")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(") - 1\n")
	g.writeIndent()
	g.writef("for __i > 0 {\n")
	g.indent++
	g.writeIndent()
	g.writef("__p := (__i - 1) / 2\n")
	g.writeIndent()
	g.writef("if !(")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__i] < ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__p]) { break }\n")
	g.writeIndent()
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__i], ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__p] = ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__p], ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__i]\n")
	g.writeIndent()
	g.writef("__i = __p\n")
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinHeappop emits an IIFE that pops the smallest element and sifts
// the replacement down.
func (g *gen) builtinHeappop(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("heapq.heappop() takes (heap)")
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("heapq.heappop(): %w", err)
	}
	if !heapOrderable(elem) {
		return fmt.Errorf("heapq.heappop(): element type must be int/float/str")
	}
	g.needsException = true
	elemGo := g.goType(elem)
	g.writef("func() %s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("if len(")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(") == 0 { panic(NewException(\"IndexError: heappop from empty heap\")) }\n")
	g.writeIndent()
	g.writef("__top := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[0]\n")
	g.writeIndent()
	g.writef("__last := len(")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(") - 1\n")
	g.writeIndent()
	g.writef("if __last == 0 {\n")
	g.indent++
	g.writeIndent()
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(" = ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[:0]\n")
	g.writeIndent()
	g.writef("return __top\n")
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[0] = ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__last]\n")
	g.writeIndent()
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(" = ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[:__last]\n")
	g.writeIndent()
	g.writef("__i := 0\n")
	g.writeIndent()
	g.writef("__n := len(")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(")\n")
	g.writeIndent()
	g.writef("for {\n")
	g.indent++
	g.writeIndent()
	g.writef("__l := 2*__i + 1\n")
	g.writeIndent()
	g.writef("__r := 2*__i + 2\n")
	g.writeIndent()
	g.writef("__best := __i\n")
	g.writeIndent()
	g.writef("if __l < __n && ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__l] < ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__best] { __best = __l }\n")
	g.writeIndent()
	g.writef("if __r < __n && ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__r] < ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__best] { __best = __r }\n")
	g.writeIndent()
	g.writef("if __best == __i { break }\n")
	g.writeIndent()
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__i], ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__best] = ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__best], ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__i]\n")
	g.writeIndent()
	g.writef("__i = __best\n")
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __top\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinHeapify emits an inline sort.Slice-based heapify — simplest
// correct approach for our limited heap surface. Sorting in ascending
// order is a valid min-heap ordering; subsequent heappush/heappop fixes
// the invariant.
func (g *gen) builtinHeapify(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("heapq.heapify() takes (heap)")
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("heapq.heapify(): %w", err)
	}
	if !heapOrderable(elem) {
		return fmt.Errorf("heapq.heapify(): element type must be int/float/str")
	}
	g.addImport("sort")
	g.writef("sort.Slice(")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(", func(i, j int) bool { return ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[i] < ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[j] })")
	return nil
}

// builtinHeappushpop emits a push-then-pop, returning the smaller of the
// new item and the heap's current min. Faster than push+pop for hot loops.
func (g *gen) builtinHeappushpop(c *ir.Call) error {
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("heapq.heappushpop() takes (heap, item)")
	}
	if err := g.builtinHeappush(c); err != nil {
		return err
	}
	g.writef("; ")
	popSynth := &ir.Call{Args: c.Args[:1]}
	return g.builtinHeappop(popSynth)
}

// builtinBisect emits a binary search returning the insertion index for
// `item` in the sorted slice. `right` selects bisect_right semantics.
func (g *gen) builtinBisect(c *ir.Call, right bool) error {
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("bisect_left/right() takes (a, x)")
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("bisect(): %w", err)
	}
	if !heapOrderable(elem) {
		return fmt.Errorf("bisect(): element type must be int/float/str")
	}
	elemGo := g.goType(elem)
	g.writef("func() int64 {\n")
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("var __x %s = ", elemGo)
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__lo, __hi := 0, len(__src)\n")
	g.writeIndent()
	g.writef("for __lo < __hi {\n")
	g.indent++
	g.writeIndent()
	g.writef("__mid := (__lo + __hi) / 2\n")
	g.writeIndent()
	if right {
		g.writef("if __x < __src[__mid] { __hi = __mid } else { __lo = __mid + 1 }\n")
	} else {
		g.writef("if __src[__mid] < __x { __lo = __mid + 1 } else { __hi = __mid }\n")
	}
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return int64(__lo)\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinInsort emits the insort_right form: finds index via bisect_right
// then splices the new element into the slice.
func (g *gen) builtinInsort(c *ir.Call) error {
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("bisect.insort() takes (a, x)")
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("insort(): %w", err)
	}
	if !heapOrderable(elem) {
		return fmt.Errorf("insort(): element type must be int/float/str")
	}
	elemGo := g.goType(elem)
	g.writef("func() {\n")
	g.indent++
	g.writeIndent()
	g.writef("var __x %s = ", elemGo)
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__lo, __hi := 0, len(")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(")\n")
	g.writeIndent()
	g.writef("for __lo < __hi {\n")
	g.indent++
	g.writeIndent()
	g.writef("__mid := (__lo + __hi) / 2\n")
	g.writeIndent()
	g.writef("if __x < ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__mid] { __hi = __mid } else { __lo = __mid + 1 }\n")
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(" = append(")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[:__lo], append([]%s{__x}, ", elemGo)
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("[__lo:]...)...)\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// heapOrderable returns true when the element type supports `<` directly
// in Go (int / float / str).
func heapOrderable(t *ir.Type) bool {
	if t == nil {
		return false
	}
	switch t.Kind {
	case ir.TyInt, ir.TyFloat, ir.TyStr:
		return true
	}
	return false
}

// builtinStarmap emits `starmap(fn, iterable)` where iterable is a list
// of pair-lists (tuples lowered to slices). The lambda body is re-lowered
// with the inner element type for both params so arithmetic typechecks.
// Only 2-arg lambdas are supported — matches our tuple-as-slice shape.
func (g *gen) builtinStarmap(c *ir.Call) error {
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("starmap() takes (fn, iterable)")
	}
	lam, ok := c.Args[0].(*ir.Lambda)
	if !ok {
		return fmt.Errorf("starmap(): first arg must be an inline lambda")
	}
	if len(lam.Params) != 2 {
		return fmt.Errorf("starmap(): lambda must take 2 arguments (pair-tuples only)")
	}
	outer, err := listElemTypeOf(c.Args[1])
	if err != nil {
		return fmt.Errorf("starmap(): %w", err)
	}
	if outer.Kind != ir.TyList {
		return fmt.Errorf("starmap(): iterable must be a list of pair-tuples")
	}
	innerElem := outer.Elem
	body, err := ir.LowerLambdaBody(lam, []*ir.Type{innerElem, innerElem})
	if err != nil {
		return fmt.Errorf("starmap(): %w", err)
	}
	retTy := body.TypeOf()
	if retTy == nil || retTy.Kind == ir.TyUnknown {
		retTy = innerElem
	}
	retGo := g.goType(retTy)
	p0, p1 := lam.Params[0].Name, lam.Params[1].Name
	g.writef("func() []%s {\n", retGo)
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__out := make([]%s, 0, len(__src))\n", retGo)
	g.writeIndent()
	g.writef("for _, __pair := range __src {\n")
	g.indent++
	g.writeIndent()
	g.writef("%s := __pair[0]\n", p0)
	g.writeIndent()
	g.writef("%s := __pair[1]\n", p1)
	g.writeIndent()
	g.writef("_, _ = %s, %s\n", p0, p1)
	g.writeIndent()
	g.writef("__out = append(__out, ")
	if err := g.expr(body); err != nil {
		return err
	}
	g.writef(")\n")
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinPermutations emits an IIFE producing every r-length ordered
// arrangement of the input list. F+ supports the fixed r=2 form to match
// builtinCombinations' shape; richer r values need recursion that would
// blow up the helper surface.
func (g *gen) builtinPermutations(c *ir.Call) error {
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("permutations() takes (iterable, r); F+ accepts r=2 only")
	}
	rLit, ok := c.Args[1].(*ir.IntLit)
	if !ok || rLit.V != 2 {
		return fmt.Errorf("permutations(): r must be the literal 2")
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("permutations(): %w", err)
	}
	elemGo := g.goType(elem)
	g.writef("func() [][]%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__out := [][]%s{}\n", elemGo)
	g.writeIndent()
	g.writef("for __i := 0; __i < len(__src); __i++ {\n")
	g.indent++
	g.writeIndent()
	g.writef("for __j := 0; __j < len(__src); __j++ {\n")
	g.indent++
	g.writeIndent()
	g.writef("if __i == __j { continue }\n")
	g.writeIndent()
	g.writef("__out = append(__out, []%s{__src[__i], __src[__j]})\n", elemGo)
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinIslice slices an already-materialized iterable. Supports both
// `islice(it, stop)` and `islice(it, start, stop[, step])` — step defaults
// to 1, all bounds are int literals or names with int type.
func (g *gen) builtinIslice(c *ir.Call) error {
	if len(c.Args) < 2 || len(c.Args) > 4 || len(c.Keywords) != 0 {
		return fmt.Errorf("islice() takes (iterable, stop) or (iterable, start, stop[, step])")
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("islice(): %w", err)
	}
	elemGo := g.goType(elem)
	var startExpr, stopExpr, stepExpr ir.Expr
	if len(c.Args) == 2 {
		stopExpr = c.Args[1]
	} else {
		startExpr = c.Args[1]
		stopExpr = c.Args[2]
		if len(c.Args) == 4 {
			stepExpr = c.Args[3]
		}
	}
	g.writef("func() []%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("var __start int64 = 0\n")
	if startExpr != nil {
		g.writeIndent()
		g.writef("__start = int64(")
		if err := g.expr(startExpr); err != nil {
			return err
		}
		g.writef(")\n")
	}
	g.writeIndent()
	g.writef("var __stop int64 = int64(")
	if err := g.expr(stopExpr); err != nil {
		return err
	}
	g.writef(")\n")
	g.writeIndent()
	g.writef("var __step int64 = 1\n")
	if stepExpr != nil {
		g.writeIndent()
		g.writef("__step = int64(")
		if err := g.expr(stepExpr); err != nil {
			return err
		}
		g.writef(")\n")
	}
	g.writeIndent()
	g.writef("if __step <= 0 { panic(NewException(\"islice(): step must be > 0\")) }\n")
	g.needsException = true
	g.writeIndent()
	g.writef("__n := int64(len(__src))\n")
	g.writeIndent()
	g.writef("if __stop > __n { __stop = __n }\n")
	g.writeIndent()
	g.writef("__out := []%s{}\n", elemGo)
	g.writeIndent()
	g.writef("for __i := __start; __i < __stop; __i += __step { __out = append(__out, __src[__i]) }\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinRepeat emits a slice of n copies of the value. CPython's
// itertools.repeat is unbounded if `n` is omitted; gopy requires `n`
// since the consumers we support all materialize the result.
func (g *gen) builtinRepeat(c *ir.Call) error {
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("repeat() takes (value, n); unbounded form not supported")
	}
	elemTy := c.Args[0].TypeOf()
	if elemTy == nil {
		elemTy = &ir.Type{Kind: ir.TyAny}
	}
	elemGo := g.goType(elemTy)
	g.writef("func() []%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__v := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__n := int(")
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef(")\n")
	g.writeIndent()
	g.writef("if __n <= 0 { return []%s{} }\n", elemGo)
	g.writeIndent()
	g.writef("__out := make([]%s, __n)\n", elemGo)
	g.writeIndent()
	g.writef("for __i := 0; __i < __n; __i++ { __out[__i] = __v }\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// emitListRepeat emits an IIFE that repeats `list` (the typed slice
// expression) `n` times, mirroring Python's `xs * 3`. Element type comes
// from the static IR type so the returned slice keeps its concrete shape.
func (g *gen) emitListRepeat(list, n ir.Expr, elem *ir.Type) error {
	elemGo := "any"
	if elem != nil {
		elemGo = g.goType(elem)
	}
	g.writef("func() []%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(list); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__n := int(")
	if err := g.expr(n); err != nil {
		return err
	}
	g.writef(")\n")
	g.writeIndent()
	g.writef("if __n <= 0 { return []%s{} }\n", elemGo)
	g.writeIndent()
	g.writef("__out := make([]%s, 0, len(__src)*__n)\n", elemGo)
	g.writeIndent()
	g.writef("for __i := 0; __i < __n; __i++ { __out = append(__out, __src...) }\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinPartial emits a closure that binds the first N positional
// arguments of a known user function. The remaining params keep their
// declared types so Go infers the closure's signature correctly. Only
// free functions are supported; partial(method) would require capturing
// a bound receiver which the IR doesn't expose at this site.
func (g *gen) builtinPartial(c *ir.Call) error {
	if len(c.Args) < 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("partial() takes a function followed by positional prefilled args")
	}
	fnName, ok := c.Args[0].(*ir.Name)
	if !ok {
		return fmt.Errorf("partial(): first argument must be a function name")
	}
	fn, ok := g.funcs[fnName.N]
	if !ok {
		return fmt.Errorf("partial(): unknown function %q", fnName.N)
	}
	if fn.Receiver != nil {
		return fmt.Errorf("partial(): methods not supported (free functions only)")
	}
	pre := c.Args[1:]
	if len(pre) > len(fn.Params) {
		return fmt.Errorf("partial(%s): too many prefilled args (got %d, max %d)", fn.Name, len(pre), len(fn.Params))
	}
	rest := fn.Params[len(pre):]
	hasRet := fn.Ret != nil && fn.Ret.Kind != ir.TyUnknown && fn.Ret.Kind != ir.TyNone
	g.writef("func(")
	for i, p := range rest {
		if i > 0 {
			g.writef(", ")
		}
		g.writef("%s %s", p.Name, g.goType(p.Ty))
	}
	if hasRet {
		g.writef(") %s { return %s(", g.goType(fn.Ret), fn.Name)
	} else {
		g.writef(") { %s(", fn.Name)
	}
	for i, a := range pre {
		if i > 0 {
			g.writef(", ")
		}
		if err := g.expr(a); err != nil {
			return err
		}
	}
	for i, p := range rest {
		if i > 0 || len(pre) > 0 {
			g.writef(", ")
		}
		g.writef("%s", p.Name)
	}
	g.writef(") }")
	return nil
}

// builtinGroupBy emits a slice of {Key, Group} pairs grouping consecutive
// elements of the iterable that share the same key. Supports an optional
// `key=` lambda; absent it, the element itself is the key. Mirrors
// CPython's itertools.groupby semantics (groups only run-length, not
// global) so callers should sort the input first if they want one bucket
// per distinct key.
func (g *gen) builtinGroupBy(c *ir.Call) error {
	if len(c.Args) != 1 {
		return fmt.Errorf("groupby() takes (iterable[, key=lambda])")
	}
	var keyLam *ir.Lambda
	for _, kw := range c.Keywords {
		if kw.Name != "key" {
			return fmt.Errorf("groupby(): unknown keyword %q", kw.Name)
		}
		lam, ok := kw.Value.(*ir.Lambda)
		if !ok {
			return fmt.Errorf("groupby(key=...): only inline lambda supported")
		}
		if len(lam.Params) != 1 {
			return fmt.Errorf("groupby(key=...): lambda must take one argument")
		}
		keyLam = lam
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("groupby(): %w", err)
	}
	keyTy := elem
	var keyBody ir.Expr
	if keyLam != nil {
		keyBody, err = ir.LowerLambdaBody(keyLam, []*ir.Type{elem})
		if err != nil {
			return fmt.Errorf("groupby(): %w", err)
		}
		if t := keyBody.TypeOf(); t != nil && t.Kind != ir.TyUnknown {
			keyTy = t
		}
	}
	elemGo := g.goType(elem)
	keyGo := g.goType(keyTy)
	g.writef("func() []struct{ Key %s; Group []%s } {\n", keyGo, elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__out := []struct{ Key %s; Group []%s }{}\n", keyGo, elemGo)
	g.writeIndent()
	g.writef("var __k %s\n", keyGo)
	g.writeIndent()
	g.writef("var __cur []%s\n", elemGo)
	g.writeIndent()
	g.writef("__started := false\n")
	g.writeIndent()
	elemName := "__v"
	if keyLam != nil {
		elemName = keyLam.Params[0].Name
	}
	g.writef("for _, %s := range __src {\n", elemName)
	g.indent++
	g.writeIndent()
	g.writef("__nk := ")
	if keyLam != nil {
		if err := g.expr(keyBody); err != nil {
			return err
		}
	} else {
		g.writef("%s", elemName)
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("if !__started || __nk != __k {\n")
	g.indent++
	g.writeIndent()
	g.writef("if __started { __out = append(__out, struct{ Key %s; Group []%s }{Key: __k, Group: __cur}) }\n", keyGo, elemGo)
	g.writeIndent()
	g.writef("__k = __nk\n")
	g.writeIndent()
	g.writef("__cur = []%s{}\n", elemGo)
	g.writeIndent()
	g.writef("__started = true\n")
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("__cur = append(__cur, %s)\n", elemName)
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("if __started { __out = append(__out, struct{ Key %s; Group []%s }{Key: __k, Group: __cur}) }\n", keyGo, elemGo)
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinNext receives the next value off an iterator. For Python
// generator channels (function-call expressions referencing a known
// generator) we emit `<-ch` directly. With a default, the receive form
// `v, ok := <-ch` lets us fall back when the channel closes.
func (g *gen) builtinNext(c *ir.Call) error {
	if len(c.Args) < 1 || len(c.Args) > 2 {
		return fmt.Errorf("next() takes (iterator[, default])")
	}
	if len(c.Args) == 1 {
		g.needsException = true
		g.writef("func() any {\n")
		g.indent++
		g.writeIndent()
		g.writef("__v, __ok := <-")
		if err := g.expr(c.Args[0]); err != nil {
			return err
		}
		g.writef("\n")
		g.writeIndent()
		g.writef("if !__ok { panic(NewException(\"StopIteration\")) }\n")
		g.writeIndent()
		g.writef("return __v\n")
		g.indent--
		g.writeIndent()
		g.writef("}()")
		return nil
	}
	g.writef("func() any {\n")
	g.indent++
	g.writeIndent()
	g.writef("__v, __ok := <-")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("if !__ok { return ")
	if err := g.boxedExpr(c.Args[1]); err != nil {
		return err
	}
	g.writef(" }\n")
	g.writeIndent()
	g.writef("return __v\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinCombinations emits an IIFE producing every 2-element subset
// (i < j) of the input slice. Only r=2 is supported — variable-r
// combinations need recursion that adds little user-facing value.
func (g *gen) builtinCombinations(c *ir.Call) error {
	if len(c.Args) != 2 {
		return fmt.Errorf("combinations() takes (iterable, r); F+ accepts r=2 only")
	}
	rLit, ok := c.Args[1].(*ir.IntLit)
	if !ok || rLit.V != 2 {
		return fmt.Errorf("combinations(): r must be the literal 2")
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("combinations(): %w", err)
	}
	elemGo := g.goType(elem)
	g.writef("func() [][]%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__out := [][]%s{}\n", elemGo)
	g.writeIndent()
	g.writef("for __i := 0; __i < len(__src); __i++ {\n")
	g.indent++
	g.writeIndent()
	g.writef("for __j := __i + 1; __j < len(__src); __j++ {\n")
	g.indent++
	g.writeIndent()
	g.writef("__out = append(__out, []%s{__src[__i], __src[__j]})\n", elemGo)
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinProduct emits the cartesian product of two same-typed slices.
// Only the 2-iterable form is supported in F+.
func (g *gen) builtinProduct(c *ir.Call) error {
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("product() takes two iterables (F+: 2-way product only)")
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("product(): %w", err)
	}
	elemGo := g.goType(elem)
	g.writef("func() [][]%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__a := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__b := ")
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__out := [][]%s{}\n", elemGo)
	g.writeIndent()
	g.writef("for _, __x := range __a {\n")
	g.indent++
	g.writeIndent()
	g.writef("for _, __y := range __b {\n")
	g.indent++
	g.writeIndent()
	g.writef("__out = append(__out, []%s{__x, __y})\n", elemGo)
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinTakeWhile / builtinDropWhile emit IIFE loops that consume the
// predicate lambda's static return type from the body inference. Like
// filter, the first argument must be an inline lambda.
func (g *gen) builtinTakeWhile(c *ir.Call) error { return g.builtinWhile(c, true) }
func (g *gen) builtinDropWhile(c *ir.Call) error { return g.builtinWhile(c, false) }

func (g *gen) builtinWhile(c *ir.Call, take bool) error {
	name := "takewhile"
	if !take {
		name = "dropwhile"
	}
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("%s() takes (predicate, iterable)", name)
	}
	lam, ok := c.Args[0].(*ir.Lambda)
	if !ok {
		return fmt.Errorf("%s(): first argument must be an inline lambda", name)
	}
	if len(lam.Params) != 1 {
		return fmt.Errorf("%s(): lambda must take one argument", name)
	}
	elem, err := listElemTypeOf(c.Args[1])
	if err != nil {
		return fmt.Errorf("%s(): %w", name, err)
	}
	body, err := ir.LowerLambdaBody(lam, []*ir.Type{elem})
	if err != nil {
		return fmt.Errorf("%s(): %w", name, err)
	}
	elemGo := g.goType(elem)
	param := lam.Params[0].Name
	g.writef("func() []%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__out := []%s{}\n", elemGo)
	g.writeIndent()
	if take {
		g.writef("for _, %s := range ", param)
	} else {
		g.writef("__taking := false\n")
		g.writeIndent()
		g.writef("for _, %s := range ", param)
	}
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef(" {\n")
	g.indent++
	if take {
		g.writeIndent()
		g.writef("if !(")
		if err := g.expr(body); err != nil {
			return err
		}
		g.writef(") { break }\n")
		g.writeIndent()
		g.writef("__out = append(__out, %s)\n", param)
	} else {
		g.writeIndent()
		g.writef("if !__taking {\n")
		g.indent++
		g.writeIndent()
		g.writef("if ")
		if err := g.expr(body); err != nil {
			return err
		}
		g.writef(" { continue }\n")
		g.writeIndent()
		g.writef("__taking = true\n")
		g.indent--
		g.writeIndent()
		g.writef("}\n")
		g.writeIndent()
		g.writef("__out = append(__out, %s)\n", param)
	}
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinDeque emits `deque()` / `deque([1,2,3])` as a new typed
// __Deque[T] backed by a slice. Element type is inferred from the
// argument's element type, defaulting to int64 when called without
// arguments (rare but harmless — the user can re-annotate).
func (g *gen) builtinDeque(c *ir.Call) error {
	if len(c.Args) > 1 {
		return fmt.Errorf("deque() takes at most one iterable")
	}
	var elem *ir.Type
	if len(c.Args) == 1 {
		e, err := listElemTypeOf(c.Args[0])
		if err != nil {
			return fmt.Errorf("deque(): %w", err)
		}
		elem = e
	} else {
		elem = &ir.Type{Kind: ir.TyAny}
	}
	g.helpers["__Deque"] = helperDequeType
	elemGo := g.goType(elem)
	g.writef("func() *__Deque {\n")
	g.indent++
	g.writeIndent()
	g.writef("__d := &__Deque{items: []any{}}\n")
	if len(c.Args) == 1 {
		g.writeIndent()
		g.writef("for _, __v := range ")
		if err := g.expr(c.Args[0]); err != nil {
			return err
		}
		g.writef(" { __d.items = append(__d.items, __v) }\n")
	}
	_ = elemGo
	g.writeIndent()
	g.writef("return __d\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// helperDequeType is the runtime Deque struct. Element type is `any`
// to keep the helper generic — the user's type info would have to be
// threaded through every method otherwise. The cost is a runtime cast
// at each `popleft` / `pop` site that wants a specific Go type.
const helperDequeType = `type __Deque struct {
	items []any
}

func (d *__Deque) Append(v any) {
	d.items = append(d.items, v)
}

func (d *__Deque) Appendleft(v any) {
	d.items = append([]any{v}, d.items...)
}

func (d *__Deque) Pop() any {
	if len(d.items) == 0 {
		panic(NewException("IndexError: pop from empty deque"))
	}
	v := d.items[len(d.items)-1]
	d.items = d.items[:len(d.items)-1]
	return v
}

func (d *__Deque) Popleft() any {
	if len(d.items) == 0 {
		panic(NewException("IndexError: popleft from empty deque"))
	}
	v := d.items[0]
	d.items = d.items[1:]
	return v
}

func (d *__Deque) Len() int64 { return int64(len(d.items)) }`

// builtinSubprocessRun emits a call to the inline __gopy_subprocess_run
// helper. The first positional argument is the command list; any kwargs
// at the call site (capture_output=True, text=True, ...) are silently
// ignored — the helper always captures stdout / stderr / returncode.
func (g *gen) builtinSubprocessRun(c *ir.Call) error {
	if len(c.Args) < 1 {
		return fmt.Errorf("subprocess.run() needs the command list as the first positional argument")
	}
	g.addImport("os/exec")
	g.helpers["__gopy_subprocess_run"] = helperSubprocessRun
	g.helpers["__CompletedProcess"] = helperCompletedProcessType
	g.writef("__gopy_subprocess_run(")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(")")
	return nil
}

// builtinCounter emits an IIFE that walks the input list and tallies
// occurrences into a typed map. Element type comes from the static IR
// type of the argument, so the resulting `dict[T, int]` can be indexed
// directly without `any`-assertions.
func (g *gen) builtinCounter(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("Counter() takes one positional iterable")
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("Counter(): %w", err)
	}
	elemGo := g.goType(elem)
	g.writef("func() map[%s]int64 {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__out := map[%s]int64{}\n", elemGo)
	g.writeIndent()
	g.writef("for _, __v := range ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(" { __out[__v]++ }\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinChain concatenates two lists of the same element type.
func (g *gen) builtinChain(c *ir.Call) error {
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("chain() takes two list arguments (F+: only 2-way chain)")
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("chain(): %w", err)
	}
	elemGo := g.goType(elem)
	g.writef("func() []%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__a := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__b := ")
	if err := g.expr(c.Args[1]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__out := make([]%s, 0, len(__a)+len(__b))\n", elemGo)
	g.writeIndent()
	g.writef("__out = append(__out, __a...)\n")
	g.writeIndent()
	g.writef("__out = append(__out, __b...)\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinAccumulate emits running totals of a numeric list. Matches
// Python's itertools.accumulate default (operator.add).
func (g *gen) builtinAccumulate(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("accumulate() takes one positional argument (key/func not supported)")
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("accumulate(): %w", err)
	}
	if elem.Kind != ir.TyInt && elem.Kind != ir.TyFloat {
		return fmt.Errorf("accumulate(): only int / float elements supported")
	}
	elemGo := g.goType(elem)
	zero := "0"
	if elem.Kind == ir.TyFloat {
		zero = "0.0"
	}
	g.writef("func() []%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__out := make([]%s, 0, len(__src))\n", elemGo)
	g.writeIndent()
	g.writef("var __acc %s = %s\n", elemGo, zero)
	g.writeIndent()
	g.writef("for __i, __v := range __src { _ = __i; __acc += __v; __out = append(__out, __acc) }\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinReversed emits an IIFE that returns a reversed copy of the
// input list. Slice element type comes from the static IR type.
func (g *gen) builtinReversed(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("reversed() takes one positional argument")
	}
	// `reversed(obj)` → `obj.Reversed()` when user class has `__reversed__`.
	if t := g.effectiveType(c.Args[0]); t != nil && t.Kind == ir.TyNamed {
		if fn := g.lookupMethod(t.Name, "__reversed__"); fn != nil {
			_ = fn
			if err := g.expr(c.Args[0]); err != nil {
				return err
			}
			g.writef(".Reversed()")
			return nil
		}
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("reversed(): %w", err)
	}
	elemGo := g.goType(elem)
	g.writef("func() []%s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("__src := ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("__out := make([]%s, len(__src))\n", elemGo)
	g.writeIndent()
	g.writef("for __i, __v := range __src { __out[len(__src)-1-__i] = __v }\n")
	g.writeIndent()
	g.writef("return __out\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinAbs maps to Go's math.Abs for floats or a sign-flip for ints.
// Static IR type drives the dispatch.
func (g *gen) builtinAbs(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("abs() takes one positional argument")
	}
	// abs(timedelta) → call the helper's Abs method, returning a fresh
	// __Timedelta with non-negative duration.
	if tag := g.exprTag(c.Args[0]); tag == "__Timedelta" {
		g.writef("func() *__Timedelta { __td := ")
		if err := g.expr(c.Args[0]); err != nil {
			return err
		}
		g.writef("; if __td.d < 0 { return &__Timedelta{d: -__td.d} }; return __td }()")
		return nil
	}
	// User-class `__abs__` dispatch: `abs(obj)` → `obj.Abs()`.
	if t := g.effectiveType(c.Args[0]); t != nil && t.Kind == ir.TyNamed {
		if fn := g.lookupMethod(t.Name, "__abs__"); fn != nil {
			_ = fn
			if err := g.expr(c.Args[0]); err != nil {
				return err
			}
			g.writef(".Abs()")
			return nil
		}
	}
	t := c.Args[0].TypeOf()
	if t != nil && t.Kind == ir.TyComplex {
		// abs(complex) → magnitude (float64) via Go's complex builtins.
		g.addImport("math")
		g.writef("math.Hypot(real(")
		if err := g.expr(c.Args[0]); err != nil {
			return err
		}
		g.writef("), imag(")
		if err := g.expr(c.Args[0]); err != nil {
			return err
		}
		g.writef("))")
		return nil
	}
	if t == nil || (t.Kind != ir.TyInt && t.Kind != ir.TyFloat) {
		return fmt.Errorf("abs() requires int or float, got %s", g.goType(t))
	}
	if t.Kind == ir.TyFloat {
		g.addImport("math")
		g.writef("math.Abs(")
		if err := g.expr(c.Args[0]); err != nil {
			return err
		}
		g.writef(")")
		return nil
	}
	// int: emit IIFE with sign flip. Wrap the value in int64 so an
	// untyped literal like `-7` doesn't collapse to Go's plain int.
	g.writef("func() int64 { var __v int64 = ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("; if __v < 0 { return -__v }; return __v }()")
	return nil
}

// builtinRound emits math.Round semantics for floats; ints pass through.
// The 2-arg form `round(x, ndigits)` rounds to ndigits decimal places and
// returns a float (Python returns float when ndigits is given).
func (g *gen) builtinRound(c *ir.Call) error {
	if len(c.Keywords) != 0 {
		return fmt.Errorf("round() takes no keyword arguments")
	}
	// User-class __round__: `round(obj)` → `obj.Round()`,
	// `round(obj, n)` → `obj.Round(n)`. When the method takes an `n`
	// parameter and the call omits it, fill in the parameter's declared
	// default (or 0) so the Go signature stays satisfied.
	if len(c.Args) >= 1 {
		if t := g.effectiveType(c.Args[0]); t != nil && t.Kind == ir.TyNamed {
			if fn := g.lookupMethod(t.Name, "__round__"); fn != nil {
				if err := g.expr(c.Args[0]); err != nil {
					return err
				}
				g.writef(".Round(")
				switch {
				case len(c.Args) == 2:
					if err := g.expr(c.Args[1]); err != nil {
						return err
					}
				case len(fn.Params) >= 1:
					if fn.Params[0].Default != nil {
						if err := g.expr(fn.Params[0].Default); err != nil {
							return err
						}
					} else {
						g.writef("int64(0)")
					}
				}
				g.writef(")")
				return nil
			}
		}
	}
	if len(c.Args) == 2 {
		g.addImport("math")
		g.writef("func() float64 { __m := math.Pow(10, float64(")
		if err := g.expr(c.Args[1]); err != nil {
			return err
		}
		g.writef(")); return math.Round(")
		t := c.Args[0].TypeOf()
		if t != nil && t.Kind == ir.TyInt {
			g.writef("float64(")
			if err := g.expr(c.Args[0]); err != nil {
				return err
			}
			g.writef(")")
		} else {
			if err := g.expr(c.Args[0]); err != nil {
				return err
			}
		}
		g.writef(" * __m) / __m }()")
		return nil
	}
	if len(c.Args) != 1 {
		return fmt.Errorf("round() takes 1 or 2 arguments")
	}
	t := c.Args[0].TypeOf()
	if t != nil && t.Kind == ir.TyInt {
		return g.expr(c.Args[0])
	}
	g.addImport("math")
	g.writef("int64(math.Round(")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("))")
	return nil
}

// builtinIsSubclass compiles `issubclass(C, Base)` to a static lookup
// against the class registry's recorded base chain. Both arguments must
// be bare class names; runtime class objects aren't tracked.
func (g *gen) builtinIsSubclass(c *ir.Call) error {
	if len(c.Args) != 2 {
		return fmt.Errorf("issubclass() takes (class, base)")
	}
	clsN, ok := c.Args[0].(*ir.Name)
	if !ok {
		return fmt.Errorf("issubclass(): first arg must be a bare class name")
	}
	bases, err := isInstanceClasses(c.Args[1])
	if err != nil {
		return err
	}
	// Compute the answer at transpile time — class hierarchy is fixed.
	result := false
	visited := map[string]bool{}
	var walk func(string) []string
	walk = func(name string) []string {
		if visited[name] {
			return nil
		}
		visited[name] = true
		out := []string{name}
		if cl, ok := g.classes[name]; ok {
			for _, b := range cl.Bases {
				out = append(out, walk(b)...)
			}
		}
		return out
	}
	chain := walk(clsN.N)
	for _, b := range bases {
		for _, c := range chain {
			if c == b {
				result = true
				break
			}
		}
		if result {
			break
		}
	}
	if result {
		g.writef("true")
	} else {
		g.writef("false")
	}
	return nil
}

// builtinIsInstance compiles `isinstance(obj, Cls)` and `isinstance(obj,
// (Cls1, Cls2, ...))` to short-circuited Go type assertions. The list
// element-type case (`isinstance(x, list)`) is still unsupported.
func (g *gen) builtinIsInstance(c *ir.Call) error {
	if len(c.Args) != 2 || len(c.Keywords) != 0 {
		return fmt.Errorf("isinstance() takes (obj, Class[s])")
	}
	classes, err := isInstanceClasses(c.Args[1])
	if err != nil {
		return err
	}
	g.writef("func() bool { __v := any(")
	if err := g.boxedExpr(c.Args[0]); err != nil {
		return err
	}
	g.writef("); _ = __v\n")
	for _, name := range classes {
		switch {
		case g.classes[name] != nil:
			g.writef("\tif _, __ok := __v.(*%s); __ok { return true }\n", name)
		case name == "int":
			g.writef("\tif _, __ok := __v.(int64); __ok { return true }\n")
		case name == "float":
			g.writef("\tif _, __ok := __v.(float64); __ok { return true }\n")
		case name == "str":
			g.writef("\tif _, __ok := __v.(string); __ok { return true }\n")
		case name == "bool":
			g.writef("\tif _, __ok := __v.(bool); __ok { return true }\n")
		default:
			return fmt.Errorf("isinstance() against %q not supported", name)
		}
	}
	g.writef("\treturn false }()")
	return nil
}

// isInstanceClasses extracts the class names from the second argument
// of isinstance(). Accepts either a bare Name or a tuple/list literal
// of Names.
func isInstanceClasses(e ir.Expr) ([]string, error) {
	switch x := e.(type) {
	case *ir.Name:
		return []string{x.N}, nil
	case *ir.ListLit:
		var names []string
		for _, el := range x.Elems {
			n, ok := el.(*ir.Name)
			if !ok {
				return nil, fmt.Errorf("isinstance(): tuple of classes must contain bare class names")
			}
			names = append(names, n.N)
		}
		return names, nil
	}
	return nil, fmt.Errorf("isinstance(): second argument must be a class name or tuple of class names")
}

// builtinReduce handles single-pass list reductions: any/all/sum/min/max.
// All take exactly one list argument; the element type guides the
// accumulator and comparator.
// builtinMinMaxArgs handles `min(a, b, ...)` / `max(a, b, ...)`. All
// args must share a numeric / string IR type so Go's `<` / `>` operators
// work. Emits an IIFE that picks the first arg then iterates the rest.
func (g *gen) builtinMinMaxArgs(c *ir.Call, kind string) error {
	first := c.Args[0].TypeOf()
	if first == nil || (first.Kind != ir.TyInt && first.Kind != ir.TyFloat && first.Kind != ir.TyStr) {
		return fmt.Errorf("%s(): args must share a numeric or string type", kind)
	}
	elemGo := g.goType(first)
	op := "<"
	if kind == "max" {
		op = ">"
	}
	g.writef("func() %s {\n", elemGo)
	g.indent++
	g.writeIndent()
	g.writef("var __acc %s = ", elemGo)
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef("\n")
	for _, a := range c.Args[1:] {
		g.writeIndent()
		g.writef("if __v := %s(", elemGo)
		if err := g.expr(a); err != nil {
			return err
		}
		g.writef("); __v %s __acc { __acc = __v }\n", op)
	}
	g.writeIndent()
	g.writef("return __acc\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

// builtinSumStart handles `sum(xs, start)` — accumulate from `start`
// instead of zero. Type follows the start expression so callers can sum
// floats into an int-typed list etc.
func (g *gen) builtinSumStart(c *ir.Call) error {
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("sum(): %w", err)
	}
	if elem.Kind != ir.TyInt && elem.Kind != ir.TyFloat {
		return fmt.Errorf("sum(): list must contain int or float")
	}
	startTy := c.Args[1].TypeOf()
	if startTy == nil || (startTy.Kind != ir.TyInt && startTy.Kind != ir.TyFloat) {
		return fmt.Errorf("sum(): start must be int or float")
	}
	retTy := startTy
	if elem.Kind == ir.TyFloat || startTy.Kind == ir.TyFloat {
		retTy = &ir.Type{Kind: ir.TyFloat}
	}
	retGo := g.goType(retTy)
	g.writef("func() %s {\n", retGo)
	g.indent++
	g.writeIndent()
	g.writef("var __acc %s = ", retGo)
	if retTy.Kind == ir.TyFloat && startTy.Kind == ir.TyInt {
		g.writef("float64(")
		if err := g.expr(c.Args[1]); err != nil {
			return err
		}
		g.writef(")")
	} else {
		if err := g.expr(c.Args[1]); err != nil {
			return err
		}
	}
	g.writef("\n")
	g.writeIndent()
	g.writef("for _, __v := range ")
	if err := g.expr(c.Args[0]); err != nil {
		return err
	}
	g.writef(" {\n")
	g.indent++
	g.writeIndent()
	if retTy.Kind == ir.TyFloat && elem.Kind == ir.TyInt {
		g.writef("__acc += float64(__v)\n")
	} else {
		g.writef("__acc += __v\n")
	}
	g.indent--
	g.writeIndent()
	g.writef("}\n")
	g.writeIndent()
	g.writef("return __acc\n")
	g.indent--
	g.writeIndent()
	g.writef("}()")
	return nil
}

func (g *gen) builtinReduce(c *ir.Call, kind string) error {
	// `min(xs, key=lambda x: ...)` / `max(...)` re-lower the lambda body with
	// the iterable's element type and pick the element with the min/max key.
	var keyLam *ir.Lambda
	for _, kw := range c.Keywords {
		if (kind == "min" || kind == "max") && kw.Name == "key" {
			lam, ok := kw.Value.(*ir.Lambda)
			if !ok {
				return fmt.Errorf("%s(key=...): only inline lambda supported", kind)
			}
			if len(lam.Params) != 1 {
				return fmt.Errorf("%s(key=...): lambda must take one argument", kind)
			}
			keyLam = lam
		} else {
			return fmt.Errorf("%s(): keyword arguments not supported", kind)
		}
	}
	// Multi-arg form (min/max only): `min(a, b)` / `min(a, b, c)`.
	// Reject for any / all / sum which only make sense over an iterable.
	// `sum(xs, start)` is the one exception — handled below.
	if kind == "sum" && len(c.Args) == 2 {
		return g.builtinSumStart(c)
	}
	if len(c.Args) > 1 {
		if kind != "min" && kind != "max" {
			return fmt.Errorf("%s() takes one iterable; got %d arguments", kind, len(c.Args))
		}
		if keyLam != nil {
			return fmt.Errorf("%s(): key= cannot combine with multi-positional form", kind)
		}
		return g.builtinMinMaxArgs(c, kind)
	}
	if len(c.Args) != 1 {
		return fmt.Errorf("%s() takes exactly one positional argument", kind)
	}
	elem, err := listElemTypeOf(c.Args[0])
	if err != nil {
		return fmt.Errorf("%s(): %w", kind, err)
	}
	elemGo := g.goType(elem)
	if keyLam != nil && (kind == "min" || kind == "max") {
		body, err := ir.LowerLambdaBody(keyLam, []*ir.Type{elem})
		if err != nil {
			return fmt.Errorf("%s(): %w", kind, err)
		}
		keyTy := body.TypeOf()
		if keyTy == nil || keyTy.Kind == ir.TyUnknown {
			keyTy = elem
		}
		keyGo := g.goType(keyTy)
		paramName := keyLam.Params[0].Name
		op := "<"
		if kind == "max" {
			op = ">"
		}
		g.writef("func() %s {\n", elemGo)
		g.indent++
		g.writeIndent()
		g.writef("__src := ")
		if err := g.expr(c.Args[0]); err != nil {
			return err
		}
		g.writef("\n")
		g.writeIndent()
		g.writef("if len(__src) == 0 { panic(NewException(\"%s() of empty sequence\")) }\n", kind)
		g.needsException = true
		g.writeIndent()
		g.writef("var __best %s = __src[0]\n", elemGo)
		g.writeIndent()
		g.writef("%s := __src[0]\n", paramName)
		g.writeIndent()
		g.writef("var __bestK %s = ", keyGo)
		if err := g.expr(body); err != nil {
			return err
		}
		g.writef("\n")
		g.writeIndent()
		g.writef("for __i := 1; __i < len(__src); __i++ {\n")
		g.indent++
		g.writeIndent()
		g.writef("%s = __src[__i]\n", paramName)
		g.writeIndent()
		g.writef("__k := ")
		if err := g.expr(body); err != nil {
			return err
		}
		g.writef("\n")
		g.writeIndent()
		g.writef("if __k %s __bestK { __bestK = __k; __best = __src[__i] }\n", op)
		g.indent--
		g.writeIndent()
		g.writef("}\n")
		g.writeIndent()
		g.writef("return __best\n")
		g.indent--
		g.writeIndent()
		g.writef("}()")
		return nil
	}
	emit := func(retGo, init, loopBody string) error {
		g.writef("func() %s {\n", retGo)
		g.indent++
		g.writeIndent()
		g.writef("__src := ")
		if err := g.expr(c.Args[0]); err != nil {
			return err
		}
		g.writef("\n")
		g.writeIndent()
		g.writef("%s\n", init)
		g.writeIndent()
		g.writef("for __i, __v := range __src {\n")
		g.indent++
		g.writeIndent()
		g.writef("%s\n", loopBody)
		g.indent--
		g.writeIndent()
		g.writef("}\n")
		g.writeIndent()
		g.writef("return __acc\n")
		g.indent--
		g.writeIndent()
		g.writef("}()")
		return nil
	}
	switch kind {
	case "any":
		return emit("bool", "__acc := false", "_ = __i; if __v { __acc = true; break }")
	case "all":
		return emit("bool", "__acc := true", "_ = __i; if !__v { __acc = false; break }")
	case "sum":
		if elem.Kind != ir.TyInt && elem.Kind != ir.TyFloat {
			return fmt.Errorf("sum(): list must contain int or float, got %s", elemGo)
		}
		zero := "0"
		if elem.Kind == ir.TyFloat {
			zero = "0.0"
		}
		return emit(elemGo, fmt.Sprintf("var __acc %s = %s", elemGo, zero), "_ = __i; __acc += __v")
	case "min":
		return emit(elemGo, "var __acc "+elemGo, "if __i == 0 || __v < __acc { __acc = __v }")
	case "max":
		return emit(elemGo, "var __acc "+elemGo, "if __i == 0 || __v > __acc { __acc = __v }")
	}
	return fmt.Errorf("unknown reduction %q", kind)
}

// listElemTypeOf extracts the element type from a list-typed expression,
// surfacing a descriptive error when the static type isn't a list.
func listElemTypeOf(e ir.Expr) (*ir.Type, error) {
	t := e.TypeOf()
	if t == nil || t.Kind != ir.TyList {
		return nil, fmt.Errorf("argument must be a typed list")
	}
	if t.Elem == nil {
		return &ir.Type{Kind: ir.TyAny}, nil
	}
	return t.Elem, nil
}

// stringMethod handles Python str methods on a TyStr-typed receiver by
// routing to the Go `strings` package. Returns (true, nil) if it handled
// the call, (false, nil) to fall through to default codegen, or an error
// if the method is recognized but argument shape is wrong.
func (g *gen) stringMethod(m *ir.MethodCall) (bool, error) {
	switch m.Method {
	case "upper":
		g.addImport("strings")
		g.writef("strings.ToUpper(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "lower":
		g.addImport("strings")
		g.writef("strings.ToLower(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "strip":
		g.addImport("strings")
		if len(m.Args) == 0 {
			g.writef("strings.TrimSpace(")
			if err := g.expr(m.Recv); err != nil {
				return true, err
			}
			g.writef(")")
			return true, nil
		}
		g.writef("strings.Trim(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(", ")
		if err := g.expr(m.Args[0]); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "split":
		g.addImport("strings")
		if len(m.Args) == 0 {
			// Python's bare split() collapses runs of whitespace; Go's Fields() does too.
			g.writef("strings.Fields(")
			if err := g.expr(m.Recv); err != nil {
				return true, err
			}
			g.writef(")")
			return true, nil
		}
		if len(m.Args) == 2 {
			// split(sep, maxsplit): Go's SplitN takes n = maxsplit+1.
			g.writef("strings.SplitN(")
			if err := g.expr(m.Recv); err != nil {
				return true, err
			}
			g.writef(", ")
			if err := g.expr(m.Args[0]); err != nil {
				return true, err
			}
			g.writef(", int(")
			if err := g.expr(m.Args[1]); err != nil {
				return true, err
			}
			g.writef("+1))")
			return true, nil
		}
		g.writef("strings.Split(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(", ")
		if err := g.expr(m.Args[0]); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "splitlines":
		if len(m.Args) != 0 {
			return true, fmt.Errorf("str.splitlines() takes no arguments")
		}
		g.helpers["__gopy_str_splitlines"] = helperStrSplitlines
		g.writef("__gopy_str_splitlines(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "partition", "rpartition":
		if len(m.Args) != 1 {
			return true, fmt.Errorf("str.%s() takes 1 argument", m.Method)
		}
		helperName := "__gopy_str_" + m.Method
		if m.Method == "partition" {
			g.helpers[helperName] = helperStrPartition
		} else {
			g.helpers[helperName] = helperStrRpartition
		}
		g.addImport("strings")
		g.writef("%s(", helperName)
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(", ")
		if err := g.expr(m.Args[0]); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "join":
		// Python: `sep.join(parts)` — receiver is the separator.
		if len(m.Args) != 1 {
			return true, fmt.Errorf("str.join() takes exactly one argument")
		}
		g.addImport("strings")
		g.writef("strings.Join(")
		if err := g.expr(m.Args[0]); err != nil {
			return true, err
		}
		g.writef(", ")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "replace":
		if len(m.Args) != 2 {
			return true, fmt.Errorf("str.replace() takes (old, new)")
		}
		g.addImport("strings")
		g.writef("strings.ReplaceAll(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(", ")
		if err := g.expr(m.Args[0]); err != nil {
			return true, err
		}
		g.writef(", ")
		if err := g.expr(m.Args[1]); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "startswith", "endswith":
		if len(m.Args) != 1 {
			return true, fmt.Errorf("str.%s() takes one argument", m.Method)
		}
		fn := "HasPrefix"
		if m.Method == "endswith" {
			fn = "HasSuffix"
		}
		g.addImport("strings")
		// Tuple argument: Python tries each candidate; emit a chained ||
		// expression so short-circuit semantics carry over.
		if lit, ok := m.Args[0].(*ir.ListLit); ok {
			// Python's tuple lit lowers to ListLit too.
			g.writef("func() bool {\n")
			g.indent++
			g.writeIndent()
			g.writef("__s := ")
			if err := g.expr(m.Recv); err != nil {
				return true, err
			}
			g.writef("\n")
			for _, e := range lit.Elems {
				g.writeIndent()
				g.writef("if strings.%s(__s, ", fn)
				if err := g.expr(e); err != nil {
					return true, err
				}
				g.writef(") { return true }\n")
			}
			g.writeIndent()
			g.writef("return false\n")
			g.indent--
			g.writeIndent()
			g.writef("}()")
			return true, nil
		}
		g.writef("strings.%s(", fn)
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(", ")
		if err := g.expr(m.Args[0]); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "removeprefix", "removesuffix":
		if len(m.Args) != 1 {
			return true, fmt.Errorf("str.%s() takes one argument", m.Method)
		}
		fn := "TrimPrefix"
		if m.Method == "removesuffix" {
			fn = "TrimSuffix"
		}
		g.addImport("strings")
		g.writef("strings.%s(", fn)
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(", ")
		if err := g.expr(m.Args[0]); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "find":
		if len(m.Args) != 1 {
			return true, fmt.Errorf("str.find() takes one argument")
		}
		g.addImport("strings")
		// Both Python's str.find and Go's strings.Index return -1 when
		// the substring isn't present, so the int64 cast is enough.
		g.writef("int64(strings.Index(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(", ")
		if err := g.expr(m.Args[0]); err != nil {
			return true, err
		}
		g.writef("))")
		return true, nil
	case "encode", "decode":
		// In Python these toggle between str and bytes. The gopy shim
		// treats both ends as `string`, so the call is a no-op — just
		// pass the receiver through.
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		return true, nil
	case "format":
		// Positional-only `{}` placeholder support. Indexed `{0}` and
		// named `{name}` substitutions are not yet implemented.
		g.addImport("strings")
		g.addImport("fmt")
		g.helpers["__gopy_str_format"] = helperStrFormat
		g.writef("__gopy_str_format(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		for _, a := range m.Args {
			g.writef(", ")
			if err := g.boxedExpr(a); err != nil {
				return true, err
			}
		}
		g.writef(")")
		return true, nil
	case "lstrip":
		g.addImport("strings")
		if len(m.Args) == 0 {
			g.writef("strings.TrimLeft(")
			if err := g.expr(m.Recv); err != nil {
				return true, err
			}
			g.writef(", \" \\t\\n\\r\")")
			return true, nil
		}
		g.writef("strings.TrimLeft(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(", ")
		if err := g.expr(m.Args[0]); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "rstrip":
		g.addImport("strings")
		if len(m.Args) == 0 {
			g.writef("strings.TrimRight(")
			if err := g.expr(m.Recv); err != nil {
				return true, err
			}
			g.writef(", \" \\t\\n\\r\")")
			return true, nil
		}
		g.writef("strings.TrimRight(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(", ")
		if err := g.expr(m.Args[0]); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "count":
		if len(m.Args) != 1 {
			return true, fmt.Errorf("str.count() takes one argument")
		}
		g.addImport("strings")
		g.writef("int64(strings.Count(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(", ")
		if err := g.expr(m.Args[0]); err != nil {
			return true, err
		}
		g.writef("))")
		return true, nil
	case "title":
		// Go's strings.Title is deprecated; replicate the Python "Title
		// Case" semantics with a tiny helper rather than pulling in
		// golang.org/x/text/cases.
		g.helpers["__gopy_str_title"] = helperStrTitle
		g.writef("__gopy_str_title(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "capitalize":
		g.helpers["__gopy_str_capitalize"] = helperStrCapitalize
		g.writef("__gopy_str_capitalize(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "center":
		if len(m.Args) < 1 || len(m.Args) > 2 {
			return true, fmt.Errorf("str.center() takes (width[, fillchar])")
		}
		g.helpers["__gopy_str_center"] = helperStrCenter
		g.writef("__gopy_str_center(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(", ")
		if err := g.expr(m.Args[0]); err != nil {
			return true, err
		}
		g.writef(", ")
		if len(m.Args) == 2 {
			if err := g.expr(m.Args[1]); err != nil {
				return true, err
			}
		} else {
			g.writef("\" \"")
		}
		g.writef(")")
		return true, nil
	case "ljust":
		if len(m.Args) < 1 || len(m.Args) > 2 {
			return true, fmt.Errorf("str.ljust() takes (width[, fillchar])")
		}
		g.helpers["__gopy_str_ljust"] = helperStrLjust
		g.writef("__gopy_str_ljust(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(", ")
		if err := g.expr(m.Args[0]); err != nil {
			return true, err
		}
		g.writef(", ")
		if len(m.Args) == 2 {
			if err := g.expr(m.Args[1]); err != nil {
				return true, err
			}
		} else {
			g.writef("\" \"")
		}
		g.writef(")")
		return true, nil
	case "rjust":
		if len(m.Args) < 1 || len(m.Args) > 2 {
			return true, fmt.Errorf("str.rjust() takes (width[, fillchar])")
		}
		g.helpers["__gopy_str_rjust"] = helperStrRjust
		g.writef("__gopy_str_rjust(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(", ")
		if err := g.expr(m.Args[0]); err != nil {
			return true, err
		}
		g.writef(", ")
		if len(m.Args) == 2 {
			if err := g.expr(m.Args[1]); err != nil {
				return true, err
			}
		} else {
			g.writef("\" \"")
		}
		g.writef(")")
		return true, nil
	case "zfill":
		if len(m.Args) != 1 {
			return true, fmt.Errorf("str.zfill() takes (width)")
		}
		g.helpers["__gopy_str_zfill"] = helperStrZfill
		g.writef("__gopy_str_zfill(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(", ")
		if err := g.expr(m.Args[0]); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "casefold":
		if len(m.Args) != 0 {
			return true, fmt.Errorf("str.casefold() takes no arguments")
		}
		g.addImport("strings")
		g.writef("strings.ToLower(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "swapcase":
		if len(m.Args) != 0 {
			return true, fmt.Errorf("str.swapcase() takes no arguments")
		}
		g.helpers["__gopy_str_swapcase"] = helperStrSwapcase
		g.writef("__gopy_str_swapcase(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "expandtabs":
		if len(m.Args) > 1 {
			return true, fmt.Errorf("str.expandtabs() takes 0 or 1 arguments")
		}
		g.helpers["__gopy_str_expandtabs"] = helperStrExpandtabs
		g.addImport("strings")
		g.writef("__gopy_str_expandtabs(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		if len(m.Args) == 1 {
			g.writef(", ")
			if err := g.expr(m.Args[0]); err != nil {
				return true, err
			}
		} else {
			g.writef(", 8")
		}
		g.writef(")")
		return true, nil
	case "translate":
		if len(m.Args) != 1 {
			return true, fmt.Errorf("str.translate() takes 1 argument")
		}
		g.helpers["__gopy_str_translate"] = helperStrTranslate
		g.addImport("strings")
		g.writef("__gopy_str_translate(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(", ")
		if err := g.expr(m.Args[0]); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "isdigit", "isalpha", "isalnum", "isspace", "isupper", "islower", "isnumeric", "isdecimal", "isidentifier", "isprintable", "isascii":
		if len(m.Args) != 0 {
			return true, fmt.Errorf("str.%s() takes no arguments", m.Method)
		}
		helperName := "__gopy_str_" + m.Method
		g.helpers[helperName] = strPredicateHelper(m.Method)
		g.writef("%s(", helperName)
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	}
	return false, nil
}

// strPredicateHelper builds the inline Go body for a str-predicate method.
// Mirrors CPython: empty string → False; isupper/islower additionally
// require at least one cased character.
func strPredicateHelper(kind string) string {
	switch kind {
	case "isidentifier":
		// Python: first char letter or _, rest letters/digits/_. Empty → False.
		return `func __gopy_str_isidentifier(s string) bool {
	if len(s) == 0 { return false }
	for i, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
		if i > 0 {
			ok = ok || (r >= '0' && r <= '9')
		}
		if !ok { return false }
	}
	return true
}`
	case "isprintable":
		// True for empty string. ASCII range 0x20..0x7E plus tab handled
		// as non-printable; matches CPython for ASCII subset.
		return `func __gopy_str_isprintable(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7F { return false }
	}
	return true
}`
	case "isascii":
		// True for empty string. All chars must be < 128.
		return `func __gopy_str_isascii(s string) bool {
	for _, r := range s {
		if r > 0x7F { return false }
	}
	return true
}`
	case "isupper":
		return `func __gopy_str_isupper(s string) bool {
	if len(s) == 0 { return false }
	hasCased := false
	for _, r := range s {
		if r >= 'A' && r <= 'Z' { hasCased = true; continue }
		if r >= 'a' && r <= 'z' { return false }
	}
	return hasCased
}`
	case "islower":
		return `func __gopy_str_islower(s string) bool {
	if len(s) == 0 { return false }
	hasCased := false
	for _, r := range s {
		if r >= 'a' && r <= 'z' { hasCased = true; continue }
		if r >= 'A' && r <= 'Z' { return false }
	}
	return hasCased
}`
	}
	body := ""
	switch kind {
	case "isdigit", "isnumeric", "isdecimal":
		// gopy lumps these together — the Unicode distinctions
		// CPython makes (numeric ⊃ digit ⊃ decimal) require a Unicode
		// table we don't carry. Treats ASCII 0-9 as the in-set.
		body = "r < '0' || r > '9'"
	case "isalpha":
		body = "!((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'))"
	case "isalnum":
		body = "!((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))"
	case "isspace":
		body = "r != ' ' && r != '\\t' && r != '\\n' && r != '\\r' && r != '\\v' && r != '\\f'"
	}
	return "func __gopy_str_" + kind + "(s string) bool {\n" +
		"\tif len(s) == 0 { return false }\n" +
		"\tfor _, r := range s { if " + body + " { return false } }\n" +
		"\treturn true\n" +
		"}"
}

const helperStrTitle = `func __gopy_str_title(s string) string {
	var b []rune
	upNext := true
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' {
			b = append(b, r)
			upNext = true
			continue
		}
		if upNext && r >= 'a' && r <= 'z' {
			b = append(b, r-32)
		} else if !upNext && r >= 'A' && r <= 'Z' {
			b = append(b, r+32)
		} else {
			b = append(b, r)
		}
		upNext = false
	}
	return string(b)
}`

const helperStrCapitalize = `func __gopy_str_capitalize(s string) string {
	rs := []rune(s)
	if len(rs) == 0 {
		return s
	}
	if rs[0] >= 'a' && rs[0] <= 'z' {
		rs[0] -= 32
	}
	for i := 1; i < len(rs); i++ {
		if rs[i] >= 'A' && rs[i] <= 'Z' {
			rs[i] += 32
		}
	}
	return string(rs)
}`

const helperStrCenter = `func __gopy_str_center(s string, width int64, fill string) string {
	n := int64(len([]rune(s)))
	if n >= width {
		return s
	}
	pad := width - n
	// CPython 3 biases the extra char to the left when total pad is odd.
	left := (pad + 1) / 2
	right := pad - left
	fr := []rune(fill)
	if len(fr) == 0 {
		fr = []rune(" ")
	}
	out := []rune{}
	for i := int64(0); i < left; i++ {
		out = append(out, fr[0])
	}
	out = append(out, []rune(s)...)
	for i := int64(0); i < right; i++ {
		out = append(out, fr[0])
	}
	return string(out)
}`

const helperStrLjust = `func __gopy_str_ljust(s string, width int64, fill string) string {
	n := int64(len([]rune(s)))
	if n >= width {
		return s
	}
	fr := []rune(fill)
	if len(fr) == 0 {
		fr = []rune(" ")
	}
	out := []rune(s)
	for i := n; i < width; i++ {
		out = append(out, fr[0])
	}
	return string(out)
}`

const helperStrRjust = `func __gopy_str_rjust(s string, width int64, fill string) string {
	n := int64(len([]rune(s)))
	if n >= width {
		return s
	}
	fr := []rune(fill)
	if len(fr) == 0 {
		fr = []rune(" ")
	}
	out := []rune{}
	for i := n; i < width; i++ {
		out = append(out, fr[0])
	}
	out = append(out, []rune(s)...)
	return string(out)
}`

const helperStrZfill = `func __gopy_str_zfill(s string, width int64) string {
	rs := []rune(s)
	n := int64(len(rs))
	if n >= width {
		return s
	}
	prefix := ""
	if len(rs) > 0 && (rs[0] == '+' || rs[0] == '-') {
		prefix = string(rs[0])
		rs = rs[1:]
	}
	// Target digit count: total width minus the optional sign char.
	digits := width
	if prefix != "" {
		digits = width - 1
	}
	out := []rune(prefix)
	have := int64(len(rs))
	for i := have; i < digits; i++ {
		out = append(out, '0')
	}
	out = append(out, rs...)
	return string(out)
}`

// helperStrSwapcase flips case rune by rune (ASCII subset). Non-letter
// runes pass through unchanged.
const helperStrSwapcase = `func __gopy_str_swapcase(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 'a' && r <= 'z' {
			out = append(out, r-32)
		} else if r >= 'A' && r <= 'Z' {
			out = append(out, r+32)
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}`

// helperStrExpandtabs replaces each tab with spaces aligning to the next
// tabstop. Matches CPython's str.expandtabs(tabsize) semantics including
// resetting the column counter at newlines.
const helperStrExpandtabs = `func __gopy_str_expandtabs(s string, tabsize int64) string {
	var b strings.Builder
	col := int64(0)
	for _, r := range s {
		if r == '\t' {
			step := tabsize
			if tabsize > 0 {
				step = tabsize - (col % tabsize)
			}
			for i := int64(0); i < step; i++ {
				b.WriteByte(' ')
			}
			col += step
			continue
		}
		b.WriteRune(r)
		if r == '\n' || r == '\r' {
			col = 0
		} else {
			col++
		}
	}
	return b.String()
}`

// helperStrSplitlines mirrors Python's str.splitlines: splits on \n, \r,
// \r\n, \v, \f and drops trailing empty element from a final newline.
const helperStrSplitlines = `func __gopy_str_splitlines(s string) []string {
	var out []string
	cur := ""
	i := 0
	for i < len(s) {
		c := s[i]
		if c == '\n' || c == '\v' || c == '\f' {
			out = append(out, cur)
			cur = ""
			i++
			continue
		}
		if c == '\r' {
			out = append(out, cur)
			cur = ""
			i++
			if i < len(s) && s[i] == '\n' {
				i++
			}
			continue
		}
		cur += string(c)
		i++
	}
	if cur != "" {
		out = append(out, cur)
	}
	if out == nil {
		return []string{}
	}
	return out
}`

// helperStrPartition mirrors Python's str.partition: returns the head,
// separator, and tail. When sep is absent, head=s, sep="", tail="".
const helperStrPartition = `func __gopy_str_partition(s, sep string) []string {
	i := strings.Index(s, sep)
	if i < 0 {
		return []string{s, "", ""}
	}
	return []string{s[:i], sep, s[i+len(sep):]}
}`

const helperStrRpartition = `func __gopy_str_rpartition(s, sep string) []string {
	i := strings.LastIndex(s, sep)
	if i < 0 {
		return []string{"", "", s}
	}
	return []string{s[:i], sep, s[i+len(sep):]}
}`

const helperStrFormat = `func __gopy_str_format(s string, args ...any) string {
	var b strings.Builder
	argi := 0
	i := 0
	for i < len(s) {
		if s[i] == '{' && i+1 < len(s) && s[i+1] == '{' {
			b.WriteByte('{')
			i += 2
			continue
		}
		if s[i] == '}' && i+1 < len(s) && s[i+1] == '}' {
			b.WriteByte('}')
			i += 2
			continue
		}
		if s[i] == '{' {
			j := i + 1
			for j < len(s) && s[j] != '}' {
				j++
			}
			if j >= len(s) {
				b.WriteByte(s[i])
				i++
				continue
			}
			spec := s[i+1 : j]
			colon := -1
			for k := 0; k < len(spec); k++ {
				if spec[k] == ':' {
					colon = k
					break
				}
			}
			fspec := ""
			if colon >= 0 {
				fspec = spec[colon+1:]
			}
			if argi < len(args) {
				b.WriteString(__gopy_fmt_spec(fspec, args[argi]))
				argi++
			}
			i = j + 1
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func __gopy_fmt_spec(spec string, v any) string {
	if spec == "" {
		return fmt.Sprint(v)
	}
	fill := byte(' ')
	align := byte(0)
	zero := false
	width := 0
	prec := -1
	typeCh := byte(0)
	n := len(spec)
	i := 0
	if n >= 2 && (spec[1] == '<' || spec[1] == '>' || spec[1] == '^') {
		fill = spec[0]
		align = spec[1]
		i = 2
	}
	if i < n && (spec[i] == '<' || spec[i] == '>' || spec[i] == '^') {
		align = spec[i]
		i++
	}
	if i < n && spec[i] == '0' {
		zero = true
		i++
	}
	for i < n && spec[i] >= '0' && spec[i] <= '9' {
		width = width*10 + int(spec[i]-'0')
		i++
	}
	if i < n && spec[i] == '.' {
		i++
		prec = 0
		for i < n && spec[i] >= '0' && spec[i] <= '9' {
			prec = prec*10 + int(spec[i]-'0')
			i++
		}
	}
	if i < n {
		typeCh = spec[i]
	}
	// Coerce v based on the requested type so Go's fmt doesn't reject it.
	switch typeCh {
	case 'd', 'x', 'X', 'o', 'b':
		switch x := v.(type) {
		case int:
			v = int64(x)
		case int32:
			v = int64(x)
		case int64:
			v = x
		case float64:
			v = int64(x)
		case float32:
			v = int64(x)
		case bool:
			if x {
				v = int64(1)
			} else {
				v = int64(0)
			}
		}
	case 'f', 'F', 'e', 'E', 'g', 'G':
		switch x := v.(type) {
		case int:
			v = float64(x)
		case int32:
			v = float64(x)
		case int64:
			v = float64(x)
		case float32:
			v = float64(x)
		case float64:
			v = x
		}
	case 's':
		v = fmt.Sprint(v)
	}
	customFill := fill != ' ' && !zero && align != 0
	var sb strings.Builder
	sb.WriteByte('%')
	if align == '<' && !customFill {
		sb.WriteByte('-')
	}
	if zero {
		sb.WriteByte('0')
	}
	if width > 0 && align != '^' && !customFill {
		fmt.Fprintf(&sb, "%d", width)
	}
	if prec >= 0 {
		fmt.Fprintf(&sb, ".%d", prec)
	}
	verb := byte('v')
	switch typeCh {
	case 'd':
		verb = 'd'
	case 'f', 'F':
		verb = 'f'
	case 'e', 'E':
		verb = byte(typeCh)
	case 'g', 'G':
		verb = byte(typeCh)
	case 'x', 'X', 'o', 'b':
		verb = typeCh
	case 's':
		verb = 's'
	default:
		switch v.(type) {
		case float32, float64:
			verb = 'g'
		case string:
			verb = 's'
		default:
			verb = 'v'
		}
	}
	sb.WriteByte(verb)
	out := fmt.Sprintf(sb.String(), v)
	if align == '^' && len(out) < width {
		pad := width - len(out)
		left := pad / 2
		right := pad - left
		return strings.Repeat(string(fill), left) + out + strings.Repeat(string(fill), right)
	}
	if customFill && (align == '<' || align == '>') && len(out) < width {
		pad := width - len(out)
		if align == '<' {
			return out + strings.Repeat(string(fill), pad)
		}
		return strings.Repeat(string(fill), pad) + out
	}
	return out
}`

// emitInOp emits Python's `in` / `not in` operators. The right operand's
// IR type drives the shape: strings → strings.Contains, dicts → map
// lookup with comma-ok, lists/sets → slice / map presence helper.
func (g *gen) emitInOp(x *ir.CmpOp) error {
	rt := g.effectiveType(x.R)
	negate := x.Op == "notin"
	// User-class `__contains__` dispatch: `needle in container` routes
	// through `container.Contains(needle)`.
	if rt != nil && rt.Kind == ir.TyNamed {
		if fn := g.lookupMethod(rt.Name, "__contains__"); fn != nil {
			if negate {
				g.writef("(!")
			}
			if err := g.expr(x.R); err != nil {
				return err
			}
			g.writef(".Contains(")
			if err := g.expr(x.L); err != nil {
				return err
			}
			g.writef(")")
			if negate {
				g.writef(")")
			}
			return nil
		}
	}
	if rt != nil {
		switch rt.Kind {
		case ir.TyStr:
			g.addImport("strings")
			if negate {
				g.writef("(!strings.Contains(")
			} else {
				g.writef("strings.Contains(")
			}
			if err := g.expr(x.R); err != nil {
				return err
			}
			g.writef(", ")
			if err := g.expr(x.L); err != nil {
				return err
			}
			if negate {
				g.writef("))")
			} else {
				g.writef(")")
			}
			return nil
		case ir.TyDict:
			g.writef("func() bool { _, __ok := ")
			if err := g.expr(x.R); err != nil {
				return err
			}
			g.writef("[")
			if err := g.expr(x.L); err != nil {
				return err
			}
			if negate {
				g.writef("]; return !__ok }()")
			} else {
				g.writef("]; return __ok }()")
			}
			return nil
		case ir.TyList:
			// Inline membership scan keyed on the slice's static
			// element type. Avoids helpers that would need generics.
			if negate {
				g.writef("(!func() bool {\n")
			} else {
				g.writef("func() bool {\n")
			}
			g.indent++
			g.writeIndent()
			// Cast the target to the element type so untyped literals
			// don't trip the comparison.
			elemGo := g.goType(rt.Elem)
			g.writef("var __target %s = ", elemGo)
			if err := g.expr(x.L); err != nil {
				return err
			}
			g.writef("\n")
			g.writeIndent()
			g.writef("for _, __v := range ")
			if err := g.expr(x.R); err != nil {
				return err
			}
			g.writef(" { if __v == __target { return true } }\n")
			g.writeIndent()
			g.writef("return false\n")
			g.indent--
			g.writeIndent()
			if negate {
				g.writef("}())")
			} else {
				g.writef("}()")
			}
			return nil
		}
	}
	return fmt.Errorf("`in` / `not in` requires receiver with known str / dict / list type")
}

// emitMethodOp renders `recv.Method(arg)` — used by BinOp rewriting when
// operator overloading on tagged stdlib types needs to delegate to a Go
// method.
func (g *gen) emitMethodOp(recv ir.Expr, method string, arg ir.Expr) error {
	if err := g.expr(recv); err != nil {
		return err
	}
	g.writef(".%s(", method)
	if err := g.expr(arg); err != nil {
		return err
	}
	g.writef(")")
	return nil
}

// exprTag returns the type tag of an expression, or "" when none is known.
// Looks through Name → varTypes and Call/MethodCall → stdlibCallRetTag.
func (g *gen) exprTag(e ir.Expr) string {
	switch x := e.(type) {
	case *ir.Name:
		return g.varTypes[x.N]
	case *ir.MethodCall:
		if recvTag := g.exprTag(x.Recv); recvTag != "" {
			if rets, ok := taggedMethodRetTag[recvTag]; ok {
				if tag, ok := rets[x.Method]; ok {
					return tag
				}
			}
		}
		return g.stdlibCallRetTag(e)
	case *ir.Call:
		_ = x
		return g.stdlibCallRetTag(e)
	case *ir.BinOp:
		// Path / str / str / ... chains stay Path-tagged so the next /
		// operator dispatches through emitMethodOp again.
		if x.Op == "/" && g.exprTag(x.L) == "__Path" {
			return "__Path"
		}
		// Timedelta arithmetic preserves the tag.
		lt, rt := g.exprTag(x.L), g.exprTag(x.R)
		if (lt == "__Timedelta" && rt == "__Timedelta") && (x.Op == "+" || x.Op == "-") {
			return "__Timedelta"
		}
		if lt == "__Timedelta" && (x.Op == "*" || x.Op == "/" || x.Op == "//") {
			return "__Timedelta"
		}
		if rt == "__Timedelta" && x.Op == "*" {
			return "__Timedelta"
		}
		if lt == "__Datetime" && rt == "__Timedelta" && (x.Op == "+" || x.Op == "-") {
			return "__Datetime"
		}
	case *ir.Attribute:
		if recvTag := g.exprTag(x.Recv); recvTag != "" {
			if attrs, ok := taggedPropAttrs[recvTag]; ok {
				if info, ok := attrs[x.Name]; ok && info.Ty == nil {
					// Untyped prop = same tag as receiver (e.g. Path.parent → Path).
					return recvTag
				}
			}
		}
	}
	return ""
}

// effectiveType returns the strongest type known for an expression.
// Resolution order: static IR type, local-var inference (for bare Names),
// stdlib return-type registry (for Call / MethodCall whose target maps
// to a stdlibFunc.RetKind). Returns nil only when no signal exists.
func (g *gen) effectiveType(e ir.Expr) *ir.Type {
	// localVarTypes wins for Name expressions: it tracks narrower /
	// inferred types (e.g. inside an `isinstance` branch) than the IR's
	// static type which might still be TyAny from a Union annotation.
	if n, ok := e.(*ir.Name); ok {
		if t, ok := g.localVarTypes[n.N]; ok && t != nil && t.Kind != ir.TyUnknown {
			return t
		}
	}
	if t := e.TypeOf(); t != nil && t.Kind != ir.TyUnknown {
		return t
	}
	if n, ok := e.(*ir.Name); ok {
		if t, ok := g.localVarTypes[n.N]; ok {
			return t
		}
	}
	if t := g.stdlibCallRetType(e); t != nil {
		return t
	}
	// Tagged-attr access: `result.stdout` where result has tag
	// __CompletedProcess and stdout is registered as str.
	if attr, ok := e.(*ir.Attribute); ok {
		if tag := g.exprTag(attr.Recv); tag != "" {
			if attrs, ok := taggedAttrs[tag]; ok {
				if info, ok := attrs[attr.Name]; ok {
					return info.Ty
				}
			}
		}
	}
	return e.TypeOf()
}

// stdlibCallRetType looks up a Call / MethodCall against the stdlib
// registry and returns the IR type derived from stdlibFunc.RetKind (for
// primitives) — TaggedTypes are not handled here; see exprTag for those.
func (g *gen) stdlibCallRetType(e ir.Expr) *ir.Type {
	var path, method string
	switch x := e.(type) {
	case *ir.Call:
		n, ok := x.Func.(*ir.Name)
		if !ok {
			return nil
		}
		p, hit := g.aliases[n.N]
		if !hit {
			return nil
		}
		segs := splitDotted(p)
		if len(segs) < 2 {
			return nil
		}
		path = strings.Join(segs[:len(segs)-1], ".")
		method = segs[len(segs)-1]
	case *ir.MethodCall:
		p, ok := stdlibPathOf(x.Recv, g.aliases)
		if !ok {
			return nil
		}
		path = p
		method = x.Method
	default:
		return nil
	}
	fn := lookupStdlibFunc(path, method)
	if fn == nil {
		return nil
	}
	switch fn.RetKind {
	case "str":
		return &ir.Type{Kind: ir.TyStr}
	case "int":
		return &ir.Type{Kind: ir.TyInt}
	case "float":
		return &ir.Type{Kind: ir.TyFloat}
	case "bool":
		return &ir.Type{Kind: ir.TyBool}
	}
	return nil
}

// userCallRetType returns the declared return type of a user-defined
// function (or method) referenced by the given call, or nil when the
// call doesn't bind to a known function/method.
func (g *gen) userCallRetType(e ir.Expr) *ir.Type {
	switch x := e.(type) {
	case *ir.Call:
		if n, ok := x.Func.(*ir.Name); ok {
			if fn, ok := g.funcs[n.N]; ok {
				return fn.Ret
			}
			// `Point(...)` where Point is a registered class — the
			// constructor returns *Point, surfaced as TyNamed so later
			// attribute access (and dataclass helpers) resolve.
			if _, ok := g.classes[n.N]; ok {
				return &ir.Type{Kind: ir.TyNamed, Name: n.N}
			}
		}
	case *ir.MethodCall:
		if rt := x.Recv.TypeOf(); rt != nil && rt.Kind == ir.TyNamed {
			if fn := g.lookupMethod(rt.Name, x.Method); fn != nil {
				return fn.Ret
			}
		}
	}
	return nil
}

// stdlibCallRetTag returns the RetTag of a stdlib call expression, or "".
// It handles both forms: a bare Call(Name(alias), ...) where the alias
// resolves to a dotted path, and a MethodCall(Recv, method) where the
// receiver is itself a stdlib module/submodule chain.
func (g *gen) stdlibCallRetTag(e ir.Expr) string {
	switch x := e.(type) {
	case *ir.Call:
		n, ok := x.Func.(*ir.Name)
		if !ok {
			return ""
		}
		path, hit := g.aliases[n.N]
		if !hit {
			return ""
		}
		// Path may be 2-seg (e.g. "os.getenv") or longer (e.g.
		// "urllib.parse.urlparse"). Split last segment as method,
		// resolve through Subs.
		segs := splitDotted(path)
		if len(segs) < 2 {
			return ""
		}
		modPath := strings.Join(segs[:len(segs)-1], ".")
		method := segs[len(segs)-1]
		fn := lookupStdlibFunc(modPath, method)
		if fn == nil {
			return ""
		}
		return fn.RetTag
	case *ir.MethodCall:
		path, ok := stdlibPathOf(x.Recv, g.aliases)
		if !ok {
			return ""
		}
		fn := lookupStdlibFunc(path, x.Method)
		if fn == nil {
			return ""
		}
		return fn.RetTag
	}
	return ""
}

// isSuperCall returns true when expr is a call to bare `super()`.
func isSuperCall(e ir.Expr) bool {
	c, ok := e.(*ir.Call)
	if !ok {
		return false
	}
	n, ok := c.Func.(*ir.Name)
	if !ok {
		return false
	}
	return n.N == "super" && len(c.Args) == 0
}

func (g *gen) goType(t *ir.Type) string {
	if t == nil {
		return "any"
	}
	switch t.Kind {
	case ir.TyInt:
		return "int64"
	case ir.TyFloat:
		return "float64"
	case ir.TyComplex:
		return "complex128"
	case ir.TyFunc:
		var b strings.Builder
		b.WriteString("func(")
		for i, p := range t.FuncParams {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(g.goType(p))
		}
		b.WriteString(")")
		if t.FuncRet != nil && t.FuncRet.Kind != ir.TyNone && t.FuncRet.Kind != ir.TyUnknown {
			b.WriteString(" ")
			b.WriteString(g.goType(t.FuncRet))
		}
		return b.String()
	case ir.TyStr:
		return "string"
	case ir.TyBool:
		return "bool"
	case ir.TyNone:
		return ""
	case ir.TyList:
		return "[]" + g.goType(t.Elem)
	case ir.TyDict:
		return "map[" + g.goType(t.Key) + "]" + g.goType(t.Val)
	case ir.TyNamed:
		// User-defined classes are referenced by pointer — except enums,
		// which the codegen aliases to int64 directly, and interface-
		// shaped classes which carry their own pointer / value receiver.
		if cls, ok := g.classes[t.Name]; ok {
			if cls.IsEnum {
				return t.Name
			}
			if cls.IsInterface && len(cls.InterfaceMethods) > 0 && len(cls.Fields) == 0 && !cls.HasInit && len(cls.MethodNames) == len(cls.InterfaceMethods) {
				return t.Name
			}
			return "*" + t.Name
		}
		if t.Name == "Exception" {
			return "*Exception"
		}
		return t.Name
	default:
		return "any"
	}
}
