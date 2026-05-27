//go:build cgo

// Introspection surface: expose a transpiled Go function to Python *with a
// real signature and annotations*, so a Python framework can introspect it
// the way it would a native `def`. FastAPI / Pydantic / click and friends
// drive everything off `inspect.signature(handler)` + `get_type_hints` — a
// bare PyCFunction (what RegisterFunc gives) reports no usable signature, so
// frameworks can't build routes / schemas from it.
//
// Mechanism: register the Go callback as a raw reverse-bridge callable, then
// hand it to a small Python factory (bootstrapped once) that wraps it in a
// real Python function carrying a synthesized `__signature__` and
// `__annotations__`. The wrapper binds/forwards arguments to the raw callable.

package bridge

import (
	"fmt"
	"sync"
)

// Param describes one parameter of a Go function being exposed to Python.
// Annotation is a Python type name the factory resolves ("int", "str",
// "float", "bool", "list", "dict", "bytes", "None", or "" / "any" for no
// annotation). HasDefault + Default supply an optional default value.
type Param struct {
	Name       string
	Annotation string
	HasDefault bool
	Default    any
}

// Field describes one attribute of a Go-backed Python class.
type Field struct {
	Name       string
	Annotation string
}

var (
	bootstrapOnce sync.Once
	bootstrapNS   *Object // namespace dict holding the factory functions
	bootstrapErr  error
)

// pyTypedFactorySrc bootstraps the Python wrapper factory. It builds a real
// function with an explicit inspect.Signature and __annotations__, so
// `inspect.signature(f)` and `typing.get_type_hints(f)` both report the Go
// function's declared shape.
const pyTypedFactorySrc = `
import inspect
import typing

# Namespace for evaluating annotation strings. Covers scalars, the typing
# generics (Optional / List / Dict / Union / Any), and the lowercase builtin
# generics (list[int], dict[str, int]). Extra names (e.g. Go-declared model
# classes) can be registered via _gopy_register_type.
_GOPY_EVAL_NS = {
    'int': int, 'float': float, 'str': str, 'bool': bool,
    'list': list, 'dict': dict, 'set': set, 'tuple': tuple,
    'bytes': bytes, 'None': None, 'NoneType': type(None), 'object': object,
    'Optional': typing.Optional, 'List': typing.List, 'Dict': typing.Dict,
    'Set': typing.Set, 'Tuple': typing.Tuple, 'Union': typing.Union,
    'Any': typing.Any,
}

def _gopy_register_type(name, obj):
    _GOPY_EVAL_NS[name] = obj

def _gopy_resolve(tn):
    if not tn or tn == 'any' or tn == 'Any':
        return inspect.Parameter.empty
    try:
        return eval(tn, dict(_GOPY_EVAL_NS))
    except Exception:
        return object

def _gopy_make_typed(raw, name, params, ret):
    sig_params = []
    ann = {}
    for p in params:
        pname, tn, has_def, dflt = p[0], p[1], p[2], p[3]
        annotation = _gopy_resolve(tn)
        default = dflt if has_def else inspect.Parameter.empty
        sig_params.append(inspect.Parameter(
            pname, inspect.Parameter.POSITIONAL_OR_KEYWORD,
            default=default, annotation=annotation))
        if annotation is not inspect.Parameter.empty:
            ann[pname] = annotation
    retann = _gopy_resolve(ret)
    sig = inspect.Signature(sig_params, return_annotation=retann)

    def wrapper(*args, **kwargs):
        bound = sig.bind(*args, **kwargs)
        bound.apply_defaults()
        return raw(*bound.args, **bound.kwargs)

    wrapper.__name__ = name
    wrapper.__qualname__ = name
    wrapper.__signature__ = sig
    if retann is not inspect.Parameter.empty:
        ann['return'] = retann
    wrapper.__annotations__ = ann
    return wrapper

def _gopy_make_class(name, bases, fields):
    ann = {}
    for f in fields:
        ann[f[0]] = _gopy_resolve(f[1])
    bases = tuple(bases) if bases else (object,)
    ns = {'__annotations__': ann, '__module__': 'gopy_bridge'}
    # Respect the most-derived metaclass among the bases (e.g. Pydantic's
    # ModelMetaclass) so class creation runs the framework's machinery.
    meta = type(bases[0])
    for b in bases[1:]:
        m = type(b)
        if issubclass(m, meta):
            meta = m
    return meta(name, bases, ns)
`

// loadBootstrap execs the factory source once into a namespace dict (kept
// alive for the process) and returns it; factory functions are fetched from
// it by name via bootstrapFunc.
func loadBootstrap() (*Object, error) {
	bootstrapOnce.Do(func() {
		builtins, err := Import("builtins")
		if err != nil {
			bootstrapErr = err
			return
		}
		defer builtins.DecRef()
		ns, err := builtins.CallMethod("dict")
		if err != nil {
			bootstrapErr = err
			return
		}
		if _, err := builtins.CallMethod("exec", pyTypedFactorySrc, ns); err != nil {
			ns.DecRef()
			bootstrapErr = err
			return
		}
		bootstrapNS = ns // kept alive; holds the factory functions
	})
	return bootstrapNS, bootstrapErr
}

// bootstrapFunc fetches a factory function from the bootstrap namespace.
func bootstrapFunc(name string) (*Object, error) {
	ns, err := loadBootstrap()
	if err != nil {
		return nil, err
	}
	if ns == nil {
		return nil, fmt.Errorf("bridge: bootstrap namespace unavailable")
	}
	return ns.GetItem(name)
}

// RegisterTypedFunc exposes a Go callback to Python as a function with a real
// signature: `inspect.signature` reports the given params (names, annotations,
// defaults) and return annotation, and `typing.get_type_hints` sees the
// annotations. The returned *Object is the wrapped Python callable to hand to
// a framework. Calls forward to fn(args, kwargs) through the reverse bridge.
func RegisterTypedFunc(name string, params []Param, retAnnotation string, fn revFunc) (*Object, error) {
	raw, err := RegisterFunc(fn)
	if err != nil {
		return nil, err
	}
	defer raw.DecRef()

	factory, err := bootstrapFunc("_gopy_make_typed")
	if err != nil {
		return nil, err
	}
	defer factory.DecRef()

	// Marshal params as a list of [name, annotation, has_default, default].
	pyParams := make([]any, len(params))
	for i, p := range params {
		var dflt any
		if p.HasDefault {
			dflt = p.Default
		}
		pyParams[i] = []any{p.Name, p.Annotation, p.HasDefault, dflt}
	}
	return factory.Call(raw, name, pyParams, retAnnotation)
}

// RegisterType makes a Python type object resolvable by `name` in annotation
// strings passed to RegisterTypedFunc / MakeClass — e.g. register a
// Go-declared Pydantic model as "User" so a handler annotated `"User"` binds
// to it. obj is typically the *Object returned by MakeClass.
func RegisterType(name string, obj *Object) error {
	reg, err := bootstrapFunc("_gopy_register_type")
	if err != nil {
		return err
	}
	defer reg.DecRef()
	r, err := reg.Call(name, obj)
	if err != nil {
		return err
	}
	r.DecRef()
	return nil
}

// MakeClass creates a Python class named `name` inheriting the given bases,
// with the given annotated fields, and returns the class object. Because the
// factory builds the class through the most-derived base metaclass, inheriting
// a framework base (e.g. pydantic.BaseModel) runs that framework's class
// machinery — so a Pydantic model can be defined entirely from Go. bases may
// be empty (the class then derives from object). Field annotations resolve the
// same names as RegisterTypedFunc ("int", "str", ...).
func MakeClass(name string, bases []*Object, fields []Field) (*Object, error) {
	if err := Init(); err != nil {
		return nil, err
	}
	factory, err := bootstrapFunc("_gopy_make_class")
	if err != nil {
		return nil, err
	}
	defer factory.DecRef()

	pyBases := make([]any, len(bases))
	for i, b := range bases {
		pyBases[i] = b
	}
	pyFields := make([]any, len(fields))
	for i, f := range fields {
		pyFields[i] = []any{f.Name, f.Annotation}
	}
	return factory.Call(name, pyBases, pyFields)
}
