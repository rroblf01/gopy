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
	g := &gen{opt: opt, classes: map[string]*ir.Class{}, funcs: map[string]*ir.Func{}, methods: map[string]map[string]*ir.Func{}, helpers: map[string]string{}, fileVars: map[string]bool{}, generators: map[string]bool{}, aliases: map[string]string{}, varTypes: map[string]string{}, localVarTypes: map[string]*ir.Type{}}
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
			if err := g.fn(x); err != nil {
				return nil, err
			}
		case *ir.Class:
			if err := g.class(x); err != nil {
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

	// Emit struct type.
	g.writef("type %s struct {\n", c.Name)
	g.indent++
	// Inheritance: embed *Base as an anonymous field. Field name is the base
	// type name, so attribute access on inherited fields works transparently.
	for _, b := range c.Bases {
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
	// primary base later, overriding this stub.
	for _, b := range c.Bases {
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
	g.writef("%s(", fn.Name)
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
					if path, hit := g.aliases[n.N]; hit && path == "collections.defaultdict" {
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
		}
		// Track stdlib-call return tags so later method dispatch and
		// truthy checks see the right type. We do this regardless of
		// whether the declaration carries an explicit annotation.
		if tag := g.stdlibCallRetTag(x.Value); tag != "" {
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
		switch {
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
		if err := g.expr(x.Value); err != nil {
			return err
		}
		g.writef("\n")
		return nil
	case *ir.AssignSub:
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

// builtinNext receives the next value off a generator's channel. Bare
// `next(it)` panics on exhaustion (mirroring Python's StopIteration);
// `next(it, default)` returns the default in that case.
const helperNoDefault = "panic(NewException(\"StopIteration\"))"

// helperGopyInt mirrors Python's int(x) for the common cases: numeric
// types are truncated to int64, strings are parsed as base-10, bools
// become 0/1. Used when the static type isn't known to be numeric
// (e.g. values pulled out of **kwargs or json.loads).
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
		if len(mc.Patterns) == 0 && mc.Guard == nil {
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
		// matches Python's integer model.
		g.writef("%s := int64(__i); _ = %s\n", x.Var, x.Var)
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
	if single {
		g.writef("for %s := range ", x.Var)
	} else {
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
		g.writef("(")
		if err := g.expr(x.L); err != nil {
			return err
		}
		g.writef(" %s ", op)
		if err := g.expr(x.R); err != nil {
			return err
		}
		g.writef(")")
	case *ir.CmpOp:
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
		switch x.Op {
		case "-":
			g.writef("(-")
		case "+":
			g.writef("(+")
		case "not":
			g.writef("(!")
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
		// Standalone-lambda fallback: emit `func(p any) any { return body }`
		// using the IR Body lowered with TyAny params. Body operations
		// that rely on concrete types will fail to compile — that's a
		// known limitation; specialized call sites (map / filter /
		// sorted with key=) re-lower with proper types.
		g.writef("func(")
		for i, p := range x.Params {
			if i > 0 {
				g.writef(", ")
			}
			g.writef("%s any", p.Name)
		}
		g.writef(") any { return ")
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
// user code; the user's loop variable keeps its Python name.
func (g *gen) listComp(c *ir.ListComp) error {
	elem := g.goType(c.ElemTy)
	if elem == "" || elem == "any" {
		elem = "any"
	}
	g.writef("func() []%s {\n", elem)
	g.indent++
	g.writeIndent()
	g.writef("__out := []%s{}\n", elem)
	g.writeIndent()
	iterTy := c.Iter.TypeOf()
	if iterTy != nil && iterTy.Kind == ir.TyDict {
		g.writef("for %s := range ", c.Var)
	} else {
		g.writef("for _, %s := range ", c.Var)
	}
	if err := g.expr(c.Iter); err != nil {
		return err
	}
	g.writef(" {\n")
	g.indent++
	if c.Cond != nil {
		g.writeIndent()
		g.writef("if !(")
		if err := g.expr(c.Cond); err != nil {
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
	g.writeIndent()
	g.writef("__out = append(__out, ")
	if err := g.expr(c.Elt); err != nil {
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
	g.writeIndent()
	iterTy := c.Iter.TypeOf()
	if iterTy != nil && iterTy.Kind == ir.TyDict {
		g.writef("for %s := range ", c.Var)
	} else {
		g.writef("for _, %s := range ", c.Var)
	}
	if err := g.expr(c.Iter); err != nil {
		return err
	}
	g.writef(" {\n")
	g.indent++
	if c.Cond != nil {
		g.writeIndent()
		g.writef("if !(")
		if err := g.expr(c.Cond); err != nil {
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
	var args []ir.Expr
	for _, p := range f.Parts {
		if p.Expr != nil {
			fmtBuf.WriteString("%v")
			args = append(args, p.Expr)
		} else {
			// Escape literal % inside the format string.
			fmtBuf.WriteString(strings.ReplaceAll(p.Lit, "%", "%%"))
		}
	}
	g.writef("fmt.Sprintf(%s", strconv.Quote(fmtBuf.String()))
	for _, a := range args {
		g.writef(", ")
		if err := g.expr(a); err != nil {
			return err
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
			case "subprocess.run":
				return g.builtinSubprocessRun(c)
			case "collections.deque":
				return g.builtinDeque(c)
			case "functools.reduce":
				return g.builtinReduceFn(c)
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
		// Class constructor: rewrite Foo(...) → NewFoo(...).
		if _, ok := g.classes[name.N]; ok {
			g.writef("New%s(", name.N)
			for i, a := range c.Args {
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
		case "float":
			if len(c.Args) != 1 {
				return fmt.Errorf("float() takes exactly 1 argument")
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
		}
	}
	// User-defined free function: resolve kwargs/defaults if any.
	if name, ok := c.Func.(*ir.Name); ok {
		if fn, ok := g.funcs[name.N]; ok {
			return g.userFuncCall(fn, c)
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
		"group":  "Group",
		"groups": "Groups",
	},
	"__Path": {
		"exists":     "Exists",
		"is_file":    "IsFile",
		"is_dir":     "IsDir",
		"read_text":  "ReadText",
		"write_text": "WriteText",
	},
	"__Datetime": {
		"year":      "Year",
		"month":     "Month",
		"day":       "Day",
		"hour":      "Hour",
		"minute":    "Minute",
		"second":    "Second",
		"isoformat": "Isoformat",
	},
	"__Hasher": {
		"hexdigest": "Hexdigest",
	},
	"__Deque": {
		"append":     "Append",
		"appendleft": "Appendleft",
		"pop":        "Pop",
		"popleft":    "Popleft",
	},
}

// taggedAttrInfo describes one tagged-value field: the Go field name to
// emit at the call site plus the IR type its access yields so chained
// expressions can dispatch correctly (e.g. `result.stdout.strip()`).
type taggedAttrInfo struct {
	GoName string
	Ty     *ir.Type
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
}

func (g *gen) methodCall(m *ir.MethodCall) error {
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
		case "collections.Counter":
			return g.builtinCounter(synth)
		case "itertools.chain":
			return g.builtinChain(synth)
		case "itertools.accumulate":
			return g.builtinAccumulate(synth)
		case "subprocess.run":
			return g.builtinSubprocessRun(synth)
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
	g.writef(".%s(", m.Method)
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
	t := c.Args[0].TypeOf()
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
func (g *gen) builtinRound(c *ir.Call) error {
	if len(c.Args) != 1 || len(c.Keywords) != 0 {
		return fmt.Errorf("round() in F11+ takes one positional argument (digits not supported)")
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

func (g *gen) builtinReduce(c *ir.Call, kind string) error {
	if len(c.Keywords) != 0 {
		return fmt.Errorf("%s(): keyword arguments not supported", kind)
	}
	// Multi-arg form (min/max only): `min(a, b)` / `min(a, b, c)`.
	// Reject for any / all / sum which only make sense over an iterable.
	if len(c.Args) > 1 {
		if kind != "min" && kind != "max" {
			return fmt.Errorf("%s() takes one iterable; got %d arguments", kind, len(c.Args))
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
	case "startswith":
		if len(m.Args) != 1 {
			return true, fmt.Errorf("str.startswith() takes one argument")
		}
		g.addImport("strings")
		g.writef("strings.HasPrefix(")
		if err := g.expr(m.Recv); err != nil {
			return true, err
		}
		g.writef(", ")
		if err := g.expr(m.Args[0]); err != nil {
			return true, err
		}
		g.writef(")")
		return true, nil
	case "endswith":
		if len(m.Args) != 1 {
			return true, fmt.Errorf("str.endswith() takes one argument")
		}
		g.addImport("strings")
		g.writef("strings.HasSuffix(")
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
	}
	return false, nil
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

const helperStrFormat = `func __gopy_str_format(s string, args ...any) string {
	var b strings.Builder
	argi := 0
	for i := 0; i < len(s); i++ {
		if i+1 < len(s) && s[i] == '{' && s[i+1] == '}' {
			if argi < len(args) {
				b.WriteString(fmt.Sprint(args[argi]))
				argi++
			}
			i++
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}`

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
	case *ir.Call, *ir.MethodCall:
		return g.stdlibCallRetTag(e.(ir.Expr))
	}
	return ""
}

// effectiveType returns the strongest type known for an expression.
// Resolution order: static IR type, local-var inference (for bare Names),
// stdlib return-type registry (for Call / MethodCall whose target maps
// to a stdlibFunc.RetKind). Returns nil only when no signal exists.
func (g *gen) effectiveType(e ir.Expr) *ir.Type {
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
		// User-defined classes are referenced by pointer.
		if _, ok := g.classes[t.Name]; ok {
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
