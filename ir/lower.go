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
	default:
		return nil, fmt.Errorf("line %d: unsupported top-level node %q", n.Lineno(), n.Type())
	}
}

// lowerClass emits a Class decl plus one Func per method (with Receiver set).
// __init__ becomes the constructor body; its self-attribute assignments are
// scanned to derive struct fields.
func lowerClass(n parser.Node) ([]Decl, error) {
	name := n.Str("name")
	class := &Class{Name: name}
	for _, b := range n.Children("bases") {
		if b.Type() != "Name" {
			return nil, fmt.Errorf("class %s: complex base expressions not supported", name)
		}
		bn := b.Str("id")
		if bn == "object" {
			continue // implicit base, ignore
		}
		class.Bases = append(class.Bases, bn)
	}
	var decls []Decl

	for _, m := range n.Children("body") {
		if m.Type() != "FunctionDef" {
			return nil, fmt.Errorf("line %d: class %s: only methods supported (F2)", m.Lineno(), name)
		}
		// F5 accepts `@property` on methods (emitted as a regular method,
		// call sites add the `()`); other decorators are still rejected.
		isProperty := false
		for _, d := range m.Children("decorator_list") {
			if d.Type() == "Name" && d.Str("id") == "property" {
				isProperty = true
				continue
			}
			var dname string
			if d.Type() == "Name" {
				dname = d.Str("id")
			} else {
				dname = d.Type()
			}
			return nil, fmt.Errorf("line %d: class %s: method decorator %q not supported", m.Lineno(), name, dname)
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

		// Regular method.
		fn := &Func{
			Name:     methName,
			Receiver: &Param{Name: "self", Ty: &Type{Kind: TyNamed, Name: name}},
		}
		for _, p := range params {
			ty, err := lowerAnnotation(p.Child("annotation"))
			if err != nil {
				return nil, fmt.Errorf("class %s.%s param %s: %w", name, methName, p.Str("arg"), err)
			}
			fn.Params = append(fn.Params, Param{Name: p.Str("arg"), Ty: ty})
		}
		ret, err := lowerAnnotation(m.Child("returns"))
		if err != nil {
			return nil, err
		}
		fn.Ret = ret

		sc := newScope()
		sc.declare("self", fn.Receiver.Ty)
		for _, p := range fn.Params {
			sc.declare(p.Name, p.Ty)
		}
		body, err := lowerBody(m.Children("body"), sc)
		if err != nil {
			return nil, err
		}
		fn.Body = body
		decls = append(decls, fn)
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
	// F4 accepts only `@staticmethod` (silently ignored — methods don't get a
	// `self`-bound receiver at the call site, which matches Go semantics for
	// plain functions). Other decorators are rejected so we don't quietly
	// produce wrong output.
	for _, d := range n.Children("decorator_list") {
		if d.Type() == "Name" && d.Str("id") == "staticmethod" {
			continue
		}
		var name string
		if d.Type() == "Name" {
			name = d.Str("id")
		} else {
			name = d.Type()
		}
		return nil, fmt.Errorf("line %d: decorator %q not supported (F4: only @staticmethod)", n.Lineno(), name)
	}
	f := &Func{Name: n.Str("name")}
	args := n.Child("args")
	if args == nil {
		return nil, fmt.Errorf("FunctionDef %q missing args", f.Name)
	}
	for _, a := range args.Children("args") {
		ty, err := lowerAnnotation(a.Child("annotation"))
		if err != nil {
			return nil, fmt.Errorf("param %q: %w", a.Str("arg"), err)
		}
		f.Params = append(f.Params, Param{Name: a.Str("arg"), Ty: ty})
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
		// e.g. `-> None` parses as Constant(None)
		if n["value"] == nil {
			return &Type{Kind: TyNone}, nil
		}
		return nil, fmt.Errorf("unsupported constant annotation")
	case "Subscript":
		// list[T] / dict[K, V] / tuple[...]
		base := n.Child("value")
		if base.Type() != "Name" {
			return nil, fmt.Errorf("generic annotation: base must be Name, got %q", base.Type())
		}
		switch base.Str("id") {
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
		if len(targets) != 1 {
			return nil, fmt.Errorf("line %d: multi-target assignment not supported", n.Lineno())
		}
		tgt := targets[0]
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
			bin := &BinOp{Op: op, L: lhs, R: rhs, Ty: promote(lhs.Ty, rhs.TypeOf())}
			return &Assign{Target: name, Value: bin, Decl: false}, nil
		case "Attribute":
			recv, err := lowerExpr(tgt.Child("value"), sc)
			if err != nil {
				return nil, err
			}
			attrName := tgt.Str("attr")
			lhs := &Attribute{Recv: recv, Name: attrName, Ty: rhs.TypeOf()}
			bin := &BinOp{Op: op, L: lhs, R: rhs, Ty: promote(lhs.Ty, rhs.TypeOf())}
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
			bin := &BinOp{Op: op, L: lhs, R: rhs, Ty: promote(elemTy, rhs.TypeOf())}
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
		thenS, err := lowerBody(n.Children("body"), sc)
		if err != nil {
			return nil, err
		}
		elseS, err := lowerBody(n.Children("orelse"), sc)
		if err != nil {
			return nil, err
		}
		return &If{Cond: cond, Then: thenS, Else: elseS}, nil
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
		return &While{Cond: cond, Body: body}, nil
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
		t, _ := sc.lookup(name)
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
		return &BinOp{Op: op, L: l, R: r, Ty: promote(l.TypeOf(), r.TypeOf())}, nil
	case "Compare":
		ops := n.Children("ops")
		comps := n.Children("comparators")
		if len(ops) != 1 || len(comps) != 1 {
			return nil, fmt.Errorf("line %d: chained comparisons not supported (F1)", n.Lineno())
		}
		l, err := lowerExpr(n.Child("left"), sc)
		if err != nil {
			return nil, err
		}
		r, err := lowerExpr(comps[0], sc)
		if err != nil {
			return nil, err
		}
		op, err := lowerCmpKind(ops[0].Type())
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", n.Lineno(), err)
		}
		return &CmpOp{Op: op, L: l, R: r, Ty: &Type{Kind: TyBool}}, nil
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
		default:
			return nil, fmt.Errorf("line %d: unsupported UnaryOp %q", n.Lineno(), opNode.Type())
		}
		ty := x.TypeOf()
		if op == "not" {
			ty = &Type{Kind: TyBool}
		}
		return &UnaryOp{Op: op, X: x, Ty: ty}, nil
	case "Call":
		if kws := n.Children("keywords"); len(kws) > 0 {
			return nil, fmt.Errorf("line %d: keyword arguments not supported", n.Lineno())
		}
		var args []Expr
		for _, a := range n.Children("args") {
			x, err := lowerExpr(a, sc)
			if err != nil {
				return nil, err
			}
			args = append(args, x)
		}
		// Method call: callee is Attribute(value, attr).
		if fnNode := n.Child("func"); fnNode.Type() == "Attribute" {
			recv, err := lowerExpr(fnNode.Child("value"), sc)
			if err != nil {
				return nil, err
			}
			return &MethodCall{Recv: recv, Method: fnNode.Str("attr"), Args: args, Ty: &Type{Kind: TyUnknown}}, nil
		}
		callee, err := lowerExpr(n.Child("func"), sc)
		if err != nil {
			return nil, err
		}
		return &Call{Func: callee, Args: args, Ty: &Type{Kind: TyUnknown}}, nil
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
		idx, err := lowerExpr(n.Child("slice"), sc)
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
	case "JoinedStr":
		// f"..." — list of Constant(str) and FormattedValue.
		var parts []FStrPart
		for _, p := range n.Children("values") {
			switch p.Type() {
			case "Constant":
				s, _ := p["value"].(string)
				parts = append(parts, FStrPart{Lit: s})
			case "FormattedValue":
				// Ignore format_spec/conversion in F2; treat as %v.
				x, err := lowerExpr(p.Child("value"), sc)
				if err != nil {
					return nil, err
				}
				parts = append(parts, FStrPart{Expr: x})
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
	switch x := v.(type) {
	case bool:
		return &BoolLit{V: x, Ty: &Type{Kind: TyBool}}, nil
	case float64:
		// JSON has no int/float distinction. Treat integer-valued floats as int.
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
	}
	return "", fmt.Errorf("unsupported Compare op %q", s)
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
	if tgt.Type() != "Name" {
		return nil, fmt.Errorf("line %d: only single Name loop variable supported", n.Lineno())
	}
	if len(n.Children("orelse")) > 0 {
		return nil, fmt.Errorf("line %d: for-else not supported", n.Lineno())
	}
	varName := tgt.Str("id")
	iter := n.Child("iter")

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
