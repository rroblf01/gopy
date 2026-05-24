// Package ir defines the typed intermediate representation produced from the
// Python AST. The transpiler consumes IR (not raw AST) so the lowering pass
// is the single place where Python semantics are made explicit.
package ir

import "github.com/rroblf01/gopy/parser"

// Type is the resolved type of an expression or variable. F1 only models a
// small set; richer kinds (generics, structs, interfaces) land in later phases.
type Type struct {
	Kind       TypeKind
	Elem       *Type   // for List/Optional
	Key, Val   *Type   // for Dict
	Name       string  // for Named
	Tuple      []*Type // for TyTuple
	FuncParams []*Type // for TyFunc
	FuncRet    *Type   // for TyFunc
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
	TyComplex
	TyFunc
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
	// Vararg captures `*args`. When non-nil, callers may pass extra
	// positional arguments after Params; codegen exposes them as
	// `<name> []any` inside the body.
	Vararg *Param
	// Kwarg captures `**kwargs`. When non-nil, callers may pass extra
	// keyword arguments not matching any Param; codegen exposes them
	// as `<name> map[string]any` inside the body.
	Kwarg *Param
	// TypeParams holds PEP 695 type parameter names declared with the
	// `def f[T, U](...)` syntax. Codegen emits them as Go generics
	// constrained to `any`.
	TypeParams []string
	// UserDecorators carries the names of user-defined decorators applied
	// to the function (`@my_wrap`). Built-in / stdlib decorators that
	// gopy already understands (e.g. @staticmethod, @lru_cache, @property)
	// are handled inline and never appear here. Codegen treats these as
	// identity wrappers — semantics that depend on the decorator body
	// (caching, logging, …) are not reproduced; the decorated function
	// runs as written. This is a best-effort accept-and-ignore so files
	// that lean on annotation-only decorators still compile.
	UserDecorators []string
}

func (*Func) declNode() {}

// Var is a module-level (package-scope) variable declaration: produced
// by lowering `name = expr` or `name: T = expr` at module top level.
// Inside function bodies, writes to Name without a `global` declaration
// still create a Go-local — codegen consults the gen.globals map to
// detect cross-function writes and switch to `=` instead of `:=`.
type Var struct {
	Name  string
	Ty    *Type
	Value Expr
}

func (*Var) declNode() {}

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
	// PropertySetters maps property name → setter method Go name (e.g.
	// "Setn"). `obj.<name> = v` codegen emits `obj.Setn(v)` for these.
	PropertySetters map[string]string
	// MethodNames lists every regular method defined directly on this
	// class (excluding __init__). The transpiler uses this to catch
	// diamond-inheritance conflicts when a subclass with multiple bases
	// inherits the same method name from two of them without overriding.
	MethodNames []string
	// ClassMethods is the set of method names declared with @classmethod.
	// They are emitted as free functions named `<Class>_<method>`; call
	// sites of the form `Class.method(args)` rewrite to that free call.
	ClassMethods map[string]bool
	// IsEnum marks subclasses of `Enum`. The transpiler emits the class
	// as a `type <Name> int64` plus one untyped constant per declared
	// member; `Class.MEMBER` attribute accesses rewrite to `<Class><MEMBER>`.
	IsEnum bool
	// EnumMembers lists the declared name / value pairs in source order.
	EnumMembers []EnumMember
	// IsInterface marks classes that inherit from abc.ABC with at least
	// one @abstractmethod and no concrete state — emitted as a Go
	// `interface` so subclasses satisfy it structurally.
	IsInterface bool
	// InterfaceMethods captures the method signatures (excluding self)
	// for the interface emission: name + params + ret type.
	InterfaceMethods []InterfaceMethod
	// ClassVars is the set of fields declared at the class level with
	// a default value and never assigned via `self.<field>` in any
	// method. They emit as module-level Go vars named `<Class>_<field>`;
	// `<Class>.<field>` / `cls.<field>` access at codegen rewrites to
	// that name, mirroring Python's shared class-state semantics.
	ClassVars map[string]ClassVar
}

// ClassVar holds the default value and IR type of a hoisted class-level
// variable. The codegen uses this to emit a module-level `var
// <Class>_<field> <Ty> = <Default>` declaration once per class.
type ClassVar struct {
	Ty      *Type
	Default Expr
}

type InterfaceMethod struct {
	Name   string
	Params []Param
	Ret    *Type
}

type EnumMember struct {
	Name  string
	Value int64
}

func (*Class) declNode() {}

type Param struct {
	Name string
	Ty   *Type
	// Default is the expression used when a call site omits this argument.
	// nil when no default was declared. Note: unlike CPython, defaults
	// here are re-evaluated at every call site, so users avoid the
	// classic "mutable default argument" trap.
	Default Expr
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
	Cond   Expr
	Body   []Stmt
	OrElse []Stmt // `while X: ... else: ...` runs only if loop exits normally
}

// ForRange is a numeric loop produced from `for i in range(...)`.
// Start/Stop/Step are typed int expressions.
type ForRange struct {
	Var    string
	Start  Expr
	Stop   Expr
	Step   Expr // nil means +1
	Body   []Stmt
	OrElse []Stmt // `for i in range(...): ... else: ...` runs only if loop exits normally
}

// ForEach iterates a list/dict/string. ElemTy is the inferred element type.
// For tuple-targeted loops the IR carries Var2 (second name) and an
// optional Kind hint to drive codegen:
//
//	""        — single-var iteration (default)
//	"dict"    — iterate (key, value) pairs of a map
//	"enum"    — iterate (index, value) pairs of a slice; Iter is the slice
//	"zip"     — iterate paired (a, b) from two slices; Iter is __ZipIter holder
type ForEach struct {
	Var    string
	Var2   string
	Iter   Expr
	Iter2  Expr // second iterable for zip
	ElemTy *Type
	Kind   string
	Body   []Stmt
	OrElse []Stmt // `for x in xs: ... else: ...` runs only if loop exits normally
}

// MultiAssign is `a, b = x, y` where both sides have matching arity.
// Codegen emits Go parallel assignment (`a, b := x, y` for first
// declaration, `a, b = x, y` for reassignment). Mixed decl is detected
// at lower time and rejected.
type MultiAssign struct {
	Targets []string
	Values  []Expr
	Decl    bool
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
	// AnnTy carries the declared annotation, if the statement came from
	// an `AnnAssign` (`self.x: T = v`). Used by class field discovery to
	// pick a precise field type when the RHS value alone is too narrow
	// (e.g. `None` for an `Optional[T]` field).
	AnnTy *Type
}

// Try is `try: ... except E [as v]: ... finally: ...`.
// Codegen lowers this to an IIFE with `defer recover()`.
type Try struct {
	Body     []Stmt
	Handlers []ExceptHandler
	Finally  []Stmt
	OrElse   []Stmt // `try: ... else: ...` runs only if no handler fired
}

// ExceptHandler is one `except ClassName [as varname]:` clause.
// ClassName == "" means bare `except:` (catch-all). ClassNames carries
// the names from `except (A, B, C):` tuple-typed clauses; ClassName is
// the first name for back-compat with single-type emit paths.
type ExceptHandler struct {
	ClassName  string
	ClassNames []string
	VarName    string
	Body       []Stmt
}

// Raise is `raise X(args...)` or bare `raise` (re-raise).
type Raise struct {
	Exc   Expr // nil for bare re-raise
	Cause Expr // `raise X from Y` — Y carried for chaining, nil otherwise
}

// Yield emits one value from a generator function. Codegen lowers it
// to `__ch <- value` inside the goroutine that wraps the function body.
type Yield struct {
	X Expr
}

// YieldFrom delegates yielding to an inner iterable (channel from another
// generator, or a slice). Lowered as a `for __v := range expr` that
// forwards each value to the outer channel.
type YieldFrom struct {
	Iter Expr
}

// Match is Python's structural-match. F+ supports only literal patterns
// (MatchValue / MatchSingleton) plus a wildcard catch-all; richer
// destructuring patterns (sequence / mapping / class) are rejected at
// lower time.
type Match struct {
	Subject Expr
	Cases   []MatchCase
}

// MatchCase: Patterns lists each `case X:` literal value (multiple for
// `case 1 | 2:`). Wildcard is the empty Patterns list. Guard is the
// optional `if <cond>` filter.
type MatchCase struct {
	Patterns []Expr
	Guard    Expr
	Body     []Stmt
	// ClassPat is non-nil for `case ClassName(field=value, ...):` arms.
	// When set, Patterns is empty.
	ClassPat *MatchClassPat
	// SeqPat is non-nil for `case [v1, v2, ...]:` arms. When set, the
	// other pattern fields are empty.
	SeqPat *MatchSeqPat
	// MapPat is non-nil for `case {"k": v, ...}:` arms.
	MapPat *MatchMapPat
	// Capture is the optional `as name` binding for class/wildcard arms.
	// For `case x:` (bare-name pattern) Capture holds the name and all
	// other pattern fields are empty (the arm acts as a default).
	Capture string
}

// MatchClassPat captures a `case ClassName(kw=pat, ...)` arm. Positional
// captures (`Class(a, b)`) bind names to fields by declaration order.
type MatchClassPat struct {
	ClassName   string
	KwdAttrs    []string
	KwdValues   []Expr
	PosCaptures []string // empty string entries = wildcard at that position
}

// MatchSeqPat captures `case [v1, v2, ...]:` — sequence match.
// Each Elements entry is either a literal value to match against
// (Capture == "") or a name to bind that position to (Capture != "",
// LitVal == nil). When Star != "", the pattern matches sequences of
// length >= len(Elements) + len(Tail); Star binds the middle slice;
// Tail elements (if any) bind the final elements.
type MatchSeqPat struct {
	Elements []MatchSeqElt
	Star     string // "" when no `*name`; "_" allowed as anonymous
	HasStar  bool
	Tail     []MatchSeqElt
}

// MatchSeqElt: exactly one of LitVal / Capture is set.
type MatchSeqElt struct {
	LitVal  Expr
	Capture string
}

// MatchMapPat captures `case {"k": v, ...}:` — checks each (key, value)
// pair is present and matches. Additional keys in the subject are
// allowed (matches CPython's partial-match semantics). `**rest` capture
// isn't supported.
type MatchMapPat struct {
	Keys   []Expr
	Values []Expr
}

// Break and Continue map directly to Go's break / continue inside the
// nearest enclosing loop. No payload needed.
type Break struct{}
type Continue struct{}

// Block is a synthetic statement carrying a sequence that should be
// inlined into the enclosing body. Used when one Python statement
// expands to several IR statements (e.g. chained assignment).
type Block struct {
	Body []Stmt
}

// LocalFunc is a `def name(...):` declared inside another function. The
// transpiler emits it as a function-typed local assigned to `name`,
// so the closure can be referenced from later statements in the same
// scope. Generators / methods are rejected at lower time.
type LocalFunc struct {
	Fn *Func
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

// WithCM is the lowered form of `with <expr> as name: body` for a
// user-class context manager. Codegen resolves __enter__/__exit__ on
// the class to emit a deferred Exit call wrapped in an IIFE.
type WithCM struct {
	VarName string // "" when there is no `as` clause
	Ctx     Expr
	Body    []Stmt
}

// Assert is `assert cond[, msg]`. Codegen panics with an AssertionError
// exception when cond evaluates falsy.
type Assert struct {
	Cond Expr
	Msg  Expr // optional
}

// Del is `del target`. Target shapes: Name (no-op + scope drop),
// Subscript (`del d[k]` → delete(d, k)), Attribute (`del obj.attr`,
// not yet supported on user classes).
type Del struct {
	Targets []Expr
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
func (*WithCM) stmtNode()     {}
func (*Assert) stmtNode()     {}
func (*Del) stmtNode()        {}
func (*Yield) stmtNode()      {}
func (*YieldFrom) stmtNode()  {}
func (*Break) stmtNode()      {}
func (*Continue) stmtNode()   {}
func (*Block) stmtNode()      {}
func (*Match) stmtNode()      {}
func (*LocalFunc) stmtNode()  {}
func (*MultiAssign) stmtNode() {}

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
type ComplexLit struct {
	Real float64
	Imag float64
	Ty   *Type
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
	Op      string // "+", "-", "*", "/", "//", "%"
	L, R    Expr
	Ty      *Type
	InPlace bool // true when produced by an AugAssign lowering
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
	Func     Expr
	Args     []Expr
	Keywords []Keyword
	Ty       *Type
}

// Keyword is one `name=value` pair in a call expression.
type Keyword struct {
	Name  string
	Value Expr
}

// MethodCall is `recv.method(args)`. Kept distinct from Call to make
// receiver-aware codegen straightforward (e.g. list.append → append()).
type MethodCall struct {
	Recv     Expr
	Method   string
	Args     []Expr
	Keywords []Keyword
	Ty       *Type
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

// Slice is `value[low:high]` or `value[low:high:step]`. Any of Low/High/Step
// may be nil to mark an omitted bound (e.g. `xs[1:]`). Step is not yet
// supported at codegen time and is rejected when non-nil.
type Slice struct {
	Value Expr
	Low   Expr
	High  Expr
	Step  Expr
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

// ListComp is `[ Elt for Var in Iter [if Cond] ]`. F7 supports one
// generator and at most one filter expression.
type ListComp struct {
	Elt    Expr
	Var    string
	Var2   string // when set, target is two-name tuple unpack over dict.items()
	Iter   Expr
	Cond   Expr // optional filter
	ElemTy *Type
	Ty     *Type
	// Extra holds any additional `for V in ITER [if COND]` generators
	// beyond the primary one above; codegen nests one loop per entry.
	Extra []CompGen
}

// CompGen is one `for V in ITER [if COND]` clause inside a comprehension.
type CompGen struct {
	Var    string
	Var2   string
	Iter   Expr
	Cond   Expr
	ElemTy *Type
}

// DictComp is `{ Key: Val for Var in Iter [if Cond] }`. Same restrictions
// as ListComp; Extra carries nested `for V in I [if C]` clauses.
type DictComp struct {
	Key   Expr
	Val   Expr
	Var   string
	Var2  string
	Iter  Expr
	Cond  Expr
	KeyTy *Type
	ValTy *Type
	Ty    *Type
	Extra []CompGen
}

// IfExpr is the ternary `then if cond else else_`. Codegen emits an IIFE
// that picks between the two branches; Go has no expression-level if.
type IfExpr struct {
	Cond Expr
	Then Expr
	Else Expr
	Ty   *Type
}

// NamedExpr is Python's walrus `(name := value)`. The lower pass hoists
// occurrences inside `if` / `while` conditions into a preceding Assign
// (so the binding survives into the body), and substitutes a plain
// Name in the condition expression. Standalone use outside a control-
// flow header is not yet supported.
type NamedExpr struct {
	Name  string
	Value Expr
	Ty    *Type
}

// Starred marks `*xs` in a call's positional-args list. The wrapped
// value is the list to splat; codegen forwards it as a Go variadic
// spread when the target function has a Vararg.
type Starred struct {
	Value Expr
	Ty    *Type
}

// Lambda is `lambda x, y: body`. Body is lowered against a scope where
// the params carry TyAny so the IR alone compiles to an `any`-typed Go
// closure as fallback. Sites with stronger type knowledge (map / filter
// / sorted's key=) re-lower BodyAST through LowerLambdaBody with the
// concrete param types they can infer.
type Lambda struct {
	Params  []Param
	Body    Expr        // body lowered with TyAny params (fallback)
	BodyAST parser.Node // raw AST for re-lowering at specialized call sites
	Ty      *Type
}

// FStr is a Python f-string lowered to a list of literal / expression parts.
// Codegen emits fmt.Sprintf with %v for each expression part.
type FStr struct {
	Parts []FStrPart
	Ty    *Type
}

// FStrPart: exactly one of Lit or Expr is set. Spec carries the optional
// Python format spec (e.g. ".2f" / ">5d"). Conv carries the `!r` / `!s`
// conversion flag if any (`r`, `s`, `a` → repr / str / ascii).
type FStrPart struct {
	Lit  string
	Expr Expr
	Spec string
	Conv byte
}

func (*IntLit) exprNode()     {}
func (*ComplexLit) exprNode() {}
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
func (*Slice) exprNode()      {}
func (*ListLit) exprNode()    {}
func (*DictLit) exprNode()    {}
func (*FStr) exprNode()       {}
func (*ListComp) exprNode()   {}
func (*DictComp) exprNode()   {}
func (*IfExpr) exprNode()     {}
func (*Lambda) exprNode()     {}
func (*Starred) exprNode()    {}
func (*NamedExpr) exprNode()  {}

func (e *IntLit) TypeOf() *Type     { return e.Ty }
func (e *ComplexLit) TypeOf() *Type { return e.Ty }
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
func (e *Slice) TypeOf() *Type      { return e.Ty }
func (e *ListLit) TypeOf() *Type    { return e.Ty }
func (e *DictLit) TypeOf() *Type    { return e.Ty }
func (e *FStr) TypeOf() *Type       { return e.Ty }
func (e *ListComp) TypeOf() *Type   { return e.Ty }
func (e *DictComp) TypeOf() *Type   { return e.Ty }
func (e *IfExpr) TypeOf() *Type     { return e.Ty }
func (e *Lambda) TypeOf() *Type     { return e.Ty }
func (e *Starred) TypeOf() *Type    { return e.Ty }
func (e *NamedExpr) TypeOf() *Type  { return e.Ty }
