// Package ir defines the typed intermediate representation produced from the
// Python AST. The transpiler consumes IR (not raw AST) so the lowering pass
// is the single place where Python semantics are made explicit.
package ir

// Type is the resolved type of an expression or variable. F1 only models a
// small set; richer kinds (generics, structs, interfaces) land in later phases.
type Type struct {
	Kind     TypeKind
	Elem     *Type   // for List/Optional
	Key, Val *Type   // for Dict
	Name     string  // for Named
}

type TypeKind int

const (
	TyUnknown TypeKind = iota
	TyNone
	TyBool
	TyInt
	TyFloat
	TyStr
	TyList
	TyDict
	TyTuple
	TyNamed
	TyAny
)

func (t *Type) IsZero() bool { return t == nil || t.Kind == TyUnknown }

// Module is the root: a flat list of top-level declarations.
type Module struct {
	Name    string
	Decls   []Decl
	Imports []Import // collected at lower time so codegen can resolve aliases
}

// Import captures one `import X` or `from X import Y [as Z]`.
// For plain `import X`, From == "" and Names == [{Name: X}].
// For `from X import Y`, From == X and Names == [{Name: Y, Alias: ""}].
// For `from X import Y as Z`, Alias = Z.
type Import struct {
	From  string       // "" for plain `import X`
	Names []ImportName
}

type ImportName struct {
	Name  string
	Alias string
}

type Decl interface{ declNode() }

type Func struct {
	Name   string
	Params []Param
	Ret    *Type
	Body   []Stmt
	// Receiver is non-nil for methods. The transpiler emits
	//   func (self *T) name(...) {...}
	Receiver *Param
	// IsGenerator is true when the function body contains `yield`.
	// Codegen routes the body through a goroutine and exposes a
	// `<-chan YieldType` to the caller.
	IsGenerator bool
	YieldType   *Type
}

func (*Func) declNode() {}

// Class lowers a Python `class` into a Go struct plus a constructor.
// Methods land in the module's Decls as separate Func entries with Receiver set.
type Class struct {
	Name     string
	Bases    []string // direct superclasses, in MRO order. F3 supports single base only.
	Fields   []Param  // ordered struct fields declared in this class (not inherited)
	HasInit  bool
	InitArgs []Param // params of __init__ excluding self (drives constructor sig)
	InitBody []Stmt  // __init__ body (self.x = expr, etc.) used as constructor body
	// Properties is the set of methods marked with @property: at call sites,
	// `instance.prop` should emit `instance.prop()` rather than a field load.
	Properties map[string]bool
}

func (*Class) declNode() {}

type Param struct {
	Name string
	Ty   *Type
}

// Stmt is any statement in a function body or module top level.
type Stmt interface{ stmtNode() }

type ExprStmt struct{ X Expr }
type Assign struct {
	Target string
	Ty     *Type // optional, from annotation
	Value  Expr
	Decl   bool  // first assignment in scope (emit `:=` / `var`)
}
type Return struct{ X Expr }
type If struct {
	Cond Expr
	Then []Stmt
	Else []Stmt
}
type While struct {
	Cond Expr
	Body []Stmt
}

// ForRange is a numeric loop produced from `for i in range(...)`.
// Start/Stop/Step are typed int expressions.
type ForRange struct {
	Var   string
	Start Expr
	Stop  Expr
	Step  Expr // nil means +1
	Body  []Stmt
}

// ForEach iterates a list/dict/string. ElemTy is the inferred element type.
type ForEach struct {
	Var    string
	Iter   Expr
	ElemTy *Type
	Body   []Stmt
}

// AssignSub assigns to an indexed expression: `target[index] = value`.
type AssignSub struct {
	Target Expr
	Index  Expr
	Value  Expr
}

// AssignAttr assigns to an attribute: `target.name = value`.
type AssignAttr struct {
	Target Expr
	Name   string
	Value  Expr
}

// Try is `try: ... except E [as v]: ... finally: ...`.
// Codegen lowers this to an IIFE with `defer recover()`.
type Try struct {
	Body    []Stmt
	Handlers []ExceptHandler
	Finally []Stmt
}

// ExceptHandler is one `except ClassName [as varname]:` clause.
// ClassName == "" means bare `except:` (catch-all).
type ExceptHandler struct {
	ClassName string
	VarName   string
	Body      []Stmt
}

// Raise is `raise X(args...)` or bare `raise` (re-raise).
type Raise struct {
	Exc Expr // nil for bare re-raise
}

// Yield emits one value from a generator function. Codegen lowers it
// to `__ch <- value` inside the goroutine that wraps the function body.
type Yield struct {
	X Expr
}

// WithFile is the lowered form of `with open(path, mode) as name: body`.
// F4 only supports file context managers; arbitrary __enter__/__exit__
// objects are rejected at lower time.
type WithFile struct {
	VarName string
	Path    Expr
	Mode    string // "r" (default) or "w"
	Body    []Stmt
}

func (*ExprStmt) stmtNode()   {}
func (*Assign) stmtNode()     {}
func (*Return) stmtNode()     {}
func (*If) stmtNode()         {}
func (*While) stmtNode()      {}
func (*ForRange) stmtNode()   {}
func (*ForEach) stmtNode()    {}
func (*AssignSub) stmtNode()  {}
func (*AssignAttr) stmtNode() {}
func (*Try) stmtNode()        {}
func (*Raise) stmtNode()      {}
func (*WithFile) stmtNode()   {}
func (*Yield) stmtNode()      {}

// Expr is any value-producing node.
type Expr interface {
	exprNode()
	TypeOf() *Type
}

type IntLit struct {
	V  int64
	Ty *Type
}
type FloatLit struct {
	V  float64
	Ty *Type
}
type StrLit struct {
	V  string
	Ty *Type
}
type BoolLit struct {
	V  bool
	Ty *Type
}
type NoneLit struct{ Ty *Type }
type Name struct {
	N  string
	Ty *Type
}
type BinOp struct {
	Op       string // "+", "-", "*", "/", "//", "%"
	L, R     Expr
	Ty       *Type
}
type CmpOp struct {
	Op   string // "==", "!=", "<", "<=", ">", ">="
	L, R Expr
	Ty   *Type
}
type BoolOp struct {
	Op   string // "and", "or"
	L, R Expr
	Ty   *Type
}
type UnaryOp struct {
	Op string // "-", "not"
	X  Expr
	Ty *Type
}
type Call struct {
	Func Expr
	Args []Expr
	Ty   *Type
}

// MethodCall is `recv.method(args)`. Kept distinct from Call to make
// receiver-aware codegen straightforward (e.g. list.append → append()).
type MethodCall struct {
	Recv   Expr
	Method string
	Args   []Expr
	Ty     *Type
}

// Attribute is `recv.name` used as a value (not assignment target).
type Attribute struct {
	Recv Expr
	Name string
	Ty   *Type
}

// Subscript is `value[index]`.
type Subscript struct {
	Value Expr
	Index Expr
	Ty    *Type
}

// ListLit is `[e1, e2, ...]`. ElemTy is inferred (or TyAny if mixed/unknown).
type ListLit struct {
	Elems  []Expr
	ElemTy *Type
	Ty     *Type
}

// DictLit is `{k1: v1, k2: v2}`. KeyTy/ValTy are inferred.
type DictLit struct {
	Keys, Vals []Expr
	KeyTy      *Type
	ValTy      *Type
	Ty         *Type
}

// FStr is a Python f-string lowered to a list of literal / expression parts.
// Codegen emits fmt.Sprintf with %v for each expression part.
type FStr struct {
	Parts []FStrPart
	Ty    *Type
}

// FStrPart: exactly one of Lit or Expr is set.
type FStrPart struct {
	Lit  string
	Expr Expr
}

func (*IntLit) exprNode()     {}
func (*FloatLit) exprNode()   {}
func (*StrLit) exprNode()     {}
func (*BoolLit) exprNode()    {}
func (*NoneLit) exprNode()    {}
func (*Name) exprNode()       {}
func (*BinOp) exprNode()      {}
func (*CmpOp) exprNode()      {}
func (*BoolOp) exprNode()     {}
func (*UnaryOp) exprNode()    {}
func (*Call) exprNode()       {}
func (*MethodCall) exprNode() {}
func (*Attribute) exprNode()  {}
func (*Subscript) exprNode()  {}
func (*ListLit) exprNode()    {}
func (*DictLit) exprNode()    {}
func (*FStr) exprNode()       {}

func (e *IntLit) TypeOf() *Type     { return e.Ty }
func (e *FloatLit) TypeOf() *Type   { return e.Ty }
func (e *StrLit) TypeOf() *Type     { return e.Ty }
func (e *BoolLit) TypeOf() *Type    { return e.Ty }
func (e *NoneLit) TypeOf() *Type    { return e.Ty }
func (e *Name) TypeOf() *Type       { return e.Ty }
func (e *BinOp) TypeOf() *Type      { return e.Ty }
func (e *CmpOp) TypeOf() *Type      { return e.Ty }
func (e *BoolOp) TypeOf() *Type     { return e.Ty }
func (e *UnaryOp) TypeOf() *Type    { return e.Ty }
func (e *Call) TypeOf() *Type       { return e.Ty }
func (e *MethodCall) TypeOf() *Type { return e.Ty }
func (e *Attribute) TypeOf() *Type  { return e.Ty }
func (e *Subscript) TypeOf() *Type  { return e.Ty }
func (e *ListLit) TypeOf() *Type    { return e.Ty }
func (e *DictLit) TypeOf() *Type    { return e.Ty }
func (e *FStr) TypeOf() *Type       { return e.Ty }
