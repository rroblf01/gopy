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
	g := &gen{opt: opt, classes: map[string]*ir.Class{}, helpers: map[string]string{}, fileVars: map[string]bool{}, generators: map[string]bool{}, aliases: map[string]string{}}
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
		}
	}
	// Scan for usage of the builtin `Exception` type so we know whether to
	// emit the embedded definition (keeps generated programs self-contained
	// without forcing a runtime import).
	g.detectExceptionUsage(m)

	if g.needsException {
		g.emitExceptionBase()
		g.writef("\n")
	}
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
		mod, ok := stdlibModules[imp.From]
		if !ok {
			continue // unknown module; let downstream produce the error
		}
		for _, n := range imp.Names {
			local := n.Alias
			if local == "" {
				local = n.Name
			}
			// If the imported name is a submodule/class registered under
			// stdlibModules[<from>].Subs, the alias resolves to the dotted path.
			if _, ok := mod.Subs[n.Name]; ok {
				g.aliases[local] = imp.From + "." + n.Name
				continue
			}
			// Otherwise it's a function imported directly (e.g. from os import getenv).
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

// emitExceptionBase writes the inline Exception base type used by raised /
// caught exceptions. Kept inline rather than imported from the runtime
// package so transpiled binaries have no module-path dependency.
func (g *gen) emitExceptionBase() {
	g.writef("type Exception struct {\n\tMsg string\n}\n\n")
	g.writef("func NewException(msg string) *Exception { return &Exception{Msg: msg} }\n\n")
	g.writef("func (e *Exception) Error() string { return e.Msg }\n\n")
	g.writef("func (e *Exception) String() string { return e.Msg }\n")
}

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
			g.scanStmtsForException(x.Then)
			g.scanStmtsForException(x.Else)
		case *ir.While:
			g.scanStmtsForException(x.Body)
		case *ir.ForRange:
			g.scanStmtsForException(x.Body)
		case *ir.ForEach:
			g.scanStmtsForException(x.Body)
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
	classes        map[string]*ir.Class // class name → decl (for super() lookup)
	needsException bool                 // module references the builtin Exception type
	currentClass   *ir.Class            // set while emitting a method body, used for super()
	helpers        map[string]string    // inline runtime helpers emitted once at module end
	fileVars       map[string]bool      // names currently bound to *os.File inside an active `with` block
	generators     map[string]bool      // function names that return a channel (Python generators)
	// aliases maps a local Python name (introduced by `from X import Y` or
	// `import X as Y`) to a dotted stdlib path the codegen knows about.
	// Example: `from datetime import datetime` → aliases["datetime"] = "datetime.datetime".
	aliases map[string]string
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
	g.writef(")")
	if fn.Ret != nil && fn.Ret.Kind != ir.TyUnknown && fn.Ret.Kind != ir.TyNone {
		g.writef(" %s", g.goType(fn.Ret))
	}
	g.writef(" {\n")
	g.indent++
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
		g.writeIndent()
		switch {
		case x.Decl && x.Ty != nil && x.Ty.Kind != ir.TyUnknown:
			g.writef("var %s %s = ", x.Target, g.goType(x.Ty))
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
		g.writef("return ")
		if err := g.expr(x.X); err != nil {
			return err
		}
		g.writef("\n")
		return nil
	case *ir.If:
		g.writeIndent()
		g.writef("if ")
		if err := g.expr(x.Cond); err != nil {
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
		if err := g.expr(x.Cond); err != nil {
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
// helperPyPrint imitates Python's print(): bools render as "True"/"False",
// nil renders as "None", everything else falls through to fmt.Print.
// Items are space-separated and the line is newline-terminated.
const helperPyPrint = `func __gopy_print(args ...any) {
	for i, a := range args {
		if i > 0 {
			fmt.Print(" ")
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
		default:
			fmt.Print(a)
		}
	}
	fmt.Println()
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
	g.writeIndent()
	// Choose single- vs two-variable range form based on iterable kind:
	//   - dict:      `for k := range m`           — Python iterates keys
	//   - generator: `for v := range ch`          — channel receive
	//   - list/str:  `for _, v := range slice`    — index discarded
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
		// @property: receiver is an instance of a user class that registers
		// this attribute as a property. Emit `recv.x()` (method call) rather
		// than `recv.x` (field load).
		if ty := x.Recv.TypeOf(); ty != nil && ty.Kind == ir.TyNamed {
			if cls, ok := g.classes[ty.Name]; ok && cls.Properties[x.Name] {
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
	default:
		return fmt.Errorf("transpile: unsupported expr %T", e)
	}
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
			segs := splitDotted(path)
			if mod, ok := stdlibModules[segs[0]]; ok && len(segs) == 2 {
				if fn, ok := mod.Funcs[segs[1]]; ok {
					if fn.GoImport != "" {
						g.addImport(fn.GoImport)
					}
					if fn.Helper != "" {
						g.helpers[fn.GoFunc] = fn.Helper
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
			// Route through a small helper so bool prints as "True"/"False"
			// and None as "None", matching Python's repr.
			g.addImport("fmt")
			g.helpers["__gopy_print"] = helperPyPrint
			g.writef("__gopy_print(")
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
			// F2 supports int(x) only for numeric truncation; full str→int parse comes later.
			if len(c.Args) != 1 {
				return fmt.Errorf("int() takes exactly 1 argument")
			}
			g.writef("int64(")
			if err := g.expr(c.Args[0]); err != nil {
				return err
			}
			g.writef(")")
			return nil
		case "float":
			if len(c.Args) != 1 {
				return fmt.Errorf("float() takes exactly 1 argument")
			}
			g.writef("float64(")
			if err := g.expr(c.Args[0]); err != nil {
				return err
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
	g.writef(")")
	return nil
}

func (g *gen) methodCall(m *ir.MethodCall) error {
	// Stdlib resolution that crosses module aliases or nested submodules
	// (e.g. `datetime.datetime.now()` and the aliased
	// `from datetime import datetime` form). Build a dotted path from the
	// receiver expression and try the registry; if it resolves we emit
	// the helper call directly without falling through.
	if path, ok := stdlibPathOf(m.Recv, g.aliases); ok {
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
