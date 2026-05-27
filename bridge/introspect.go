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

var (
	typedFactoryOnce sync.Once
	typedFactory     *Object
	typedFactoryErr  error
)

// pyTypedFactorySrc bootstraps the Python wrapper factory. It builds a real
// function with an explicit inspect.Signature and __annotations__, so
// `inspect.signature(f)` and `typing.get_type_hints(f)` both report the Go
// function's declared shape.
const pyTypedFactorySrc = `
import inspect

_GOPY_TYPES = {
    'int': int, 'float': float, 'str': str, 'bool': bool,
    'list': list, 'dict': dict, 'bytes': bytes, 'none': type(None),
}

def _gopy_resolve(tn):
    if not tn or tn == 'any':
        return inspect.Parameter.empty
    return _GOPY_TYPES.get(tn.lower(), object)

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
`

// loadTypedFactory bootstraps and caches the Python wrapper factory function.
func loadTypedFactory() (*Object, error) {
	typedFactoryOnce.Do(func() {
		builtins, err := Import("builtins")
		if err != nil {
			typedFactoryErr = err
			return
		}
		defer builtins.DecRef()
		ns, err := builtins.CallMethod("dict")
		if err != nil {
			typedFactoryErr = err
			return
		}
		defer ns.DecRef()
		if _, err := builtins.CallMethod("exec", pyTypedFactorySrc, ns); err != nil {
			typedFactoryErr = err
			return
		}
		f, err := ns.GetItem("_gopy_make_typed")
		if err != nil {
			typedFactoryErr = err
			return
		}
		typedFactory = f // kept alive for the process lifetime
	})
	return typedFactory, typedFactoryErr
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

	factory, err := loadTypedFactory()
	if err != nil {
		return nil, err
	}
	if factory == nil {
		return nil, fmt.Errorf("bridge: typed-func factory unavailable")
	}

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
