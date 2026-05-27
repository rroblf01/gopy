//go:build cgo

// Package bridge embeds a CPython interpreter inside the gopy binary via
// cgo so transpiled code can call into real Python libraries (Pydantic,
// psycopg2, etc.) instead of relying on a Go-native shim per module.
//
// It is the "slow path" of gopy's hybrid model: pure code keeps compiling
// to native Go, and only operations that touch the embedded interpreter
// cross this boundary. Each crossing marshals values Go<->PyObject and
// acquires the GIL, so the bridge is deliberately coarse-grained.
//
// Build tag: requires cgo (CGO_ENABLED=1) and a libpython discoverable via
// the `python3-embed` pkg-config file.
package bridge

/*
#cgo pkg-config: python3-embed
#include <Python.h>
#include <stdlib.h>

// Helper shims: cgo can't call variadic C functions, and several CPython
// entry points are macros that cgo won't see. Wrap the ones we need.

static PyObject* gopy_import(const char* name) {
    return PyImport_ImportModule(name);
}

static PyObject* gopy_getattr(PyObject* o, const char* name) {
    return PyObject_GetAttrString(o, name);
}

static PyObject* gopy_call(PyObject* fn, PyObject* args) {
    return PyObject_CallObject(fn, args);
}

// Call with positional args tuple + keyword args dict (either may be NULL,
// though CPython requires args to be a tuple — callers pass an empty tuple).
static PyObject* gopy_call_kw(PyObject* fn, PyObject* args, PyObject* kw) {
    return PyObject_Call(fn, args, kw);
}

static PyObject* gopy_getitem(PyObject* o, PyObject* key) {
    return PyObject_GetItem(o, key);
}

static Py_ssize_t gopy_len(PyObject* o) { return PyObject_Length(o); }
static int gopy_is_true_obj(PyObject* o) { return PyObject_IsTrue(o); }
static PyObject* gopy_repr(PyObject* o) { return PyObject_Repr(o); }
static PyObject* gopy_str(PyObject* o) { return PyObject_Str(o); }

static PyObject* gopy_none() {
    Py_RETURN_NONE;
}

static int gopy_is_none(PyObject* o) { return o == Py_None; }
static int gopy_is_true(PyObject* o) { return o == Py_True; }

// Type predicates are macros in CPython headers; cgo can't reference macros,
// so wrap each in a real function.
static int gopy_bool_check(PyObject* o)    { return PyBool_Check(o); }
static int gopy_long_check(PyObject* o)     { return PyLong_Check(o); }
static int gopy_float_check(PyObject* o)    { return PyFloat_Check(o); }
static int gopy_unicode_check(PyObject* o)  { return PyUnicode_Check(o); }
static int gopy_list_check(PyObject* o)     { return PyList_Check(o); }
static int gopy_tuple_check(PyObject* o)    { return PyTuple_Check(o); }
static int gopy_dict_check(PyObject* o)     { return PyDict_Check(o); }

// Fetch + clear the current exception, returning its str() as a C string the
// caller must free. Returns NULL when no exception is set.
static char* gopy_err_fetch() {
    if (!PyErr_Occurred()) return NULL;
    PyObject *t, *v, *tb;
    PyErr_Fetch(&t, &v, &tb);
    PyErr_NormalizeException(&t, &v, &tb);
    PyObject* s = v ? PyObject_Str(v) : PyUnicode_FromString("<no value>");
    const char* utf = s ? PyUnicode_AsUTF8(s) : "<unprintable>";
    char* out = utf ? strdup(utf) : strdup("<unprintable>");
    Py_XDECREF(s);
    Py_XDECREF(t); Py_XDECREF(v); Py_XDECREF(tb);
    return out;
}
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"
)

// Object wraps a borrowed-or-owned *PyObject. The bridge always holds owned
// (new) references and releases them via (*Object).DecRef. Callers must not
// retain the underlying pointer past DecRef.
type Object struct {
	p *C.PyObject
}

var (
	initOnce  sync.Once
	initErr   error
	gilMu     sync.Mutex // serializes interpreter access from goroutines
	mainState *C.PyThreadState
)

// Init starts the embedded interpreter exactly once. Safe to call repeatedly.
// After init the GIL is released so other goroutines can re-acquire it per
// call via the gilMu + PyGILState dance in withGIL.
func Init() error {
	initOnce.Do(func() {
		C.Py_Initialize()
		if C.Py_IsInitialized() == 0 {
			initErr = fmt.Errorf("bridge: Py_Initialize failed")
			return
		}
		// Release the GIL held by the initializing thread; per-call code
		// re-acquires it. Save the main thread state so Finalize can restore.
		mainState = C.PyEval_SaveThread()
	})
	return initErr
}

// withGIL runs fn while holding the GIL. It serializes all interpreter
// access through gilMu (coarse but correct) and acquires/releases the GIL
// with the PyGILState API, which is goroutine-thread-safe. Callers return
// their results via captured variables in the closure.
func withGIL(fn func()) {
	gilMu.Lock()
	defer gilMu.Unlock()
	gil := C.PyGILState_Ensure()
	defer C.PyGILState_Release(gil)
	fn()
}

// lastError returns the current Python exception as a Go error, or nil.
// Must be called while holding the GIL.
func lastError() error {
	cs := C.gopy_err_fetch()
	if cs == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(cs))
	return fmt.Errorf("python: %s", C.GoString(cs))
}

// Import imports a module by dotted name (e.g. "pydantic_core").
func Import(name string) (*Object, error) {
	if err := Init(); err != nil {
		return nil, err
	}
	cn := C.CString(name)
	defer C.free(unsafe.Pointer(cn))
	var obj *Object
	var err error
	withGIL(func() {
		p := C.gopy_import(cn)
		if p == nil {
			err = lastError()
			return
		}
		obj = &Object{p: p}
	})
	return obj, err
}

// Attr fetches obj.name.
func (o *Object) Attr(name string) (*Object, error) {
	cn := C.CString(name)
	defer C.free(unsafe.Pointer(cn))
	var obj *Object
	var err error
	withGIL(func() {
		p := C.gopy_getattr(o.p, cn)
		if p == nil {
			err = lastError()
			return
		}
		obj = &Object{p: p}
	})
	return obj, err
}

// Call invokes a callable Object with the given Go args (converted to Python).
func (o *Object) Call(args ...any) (*Object, error) {
	var obj *Object
	var err error
	withGIL(func() {
		tup := C.PyTuple_New(C.Py_ssize_t(len(args)))
		if tup == nil {
			err = lastError()
			return
		}
		defer C.Py_DecRef(tup)
		for i, a := range args {
			pa := toPy(a)
			if pa == nil {
				err = fmt.Errorf("bridge: cannot convert arg %d (%T) to Python", i, a)
				return
			}
			// PyTuple_SetItem steals the reference — no DecRef on pa.
			C.PyTuple_SetItem(tup, C.Py_ssize_t(i), pa)
		}
		res := C.gopy_call(o.p, tup)
		if res == nil {
			err = lastError()
			return
		}
		obj = &Object{p: res}
	})
	return obj, err
}

// CallMethod fetches obj.name and calls it with args.
func (o *Object) CallMethod(name string, args ...any) (*Object, error) {
	m, err := o.Attr(name)
	if err != nil {
		return nil, err
	}
	defer m.DecRef()
	return m.Call(args...)
}

// CallKw invokes a callable with positional args plus keyword args. kwargs
// keys become Python keyword names; values are converted like positional
// args. A nil/empty kwargs behaves like Call.
func (o *Object) CallKw(args []any, kwargs map[string]any) (*Object, error) {
	var obj *Object
	var err error
	withGIL(func() {
		tup := C.PyTuple_New(C.Py_ssize_t(len(args)))
		if tup == nil {
			err = lastError()
			return
		}
		defer C.Py_DecRef(tup)
		for i, a := range args {
			pa := toPy(a)
			if pa == nil {
				err = fmt.Errorf("bridge: cannot convert arg %d (%T) to Python", i, a)
				return
			}
			C.PyTuple_SetItem(tup, C.Py_ssize_t(i), pa) // steals
		}
		var kw *C.PyObject
		if len(kwargs) > 0 {
			kw = toPy(map[string]any(kwargs))
			if kw == nil {
				err = fmt.Errorf("bridge: cannot convert kwargs to Python")
				return
			}
			defer C.Py_DecRef(kw)
		}
		res := C.gopy_call_kw(o.p, tup, kw)
		if res == nil {
			err = lastError()
			return
		}
		obj = &Object{p: res}
	})
	return obj, err
}

// CallMethodKw fetches obj.name and calls it with args + kwargs.
func (o *Object) CallMethodKw(name string, args []any, kwargs map[string]any) (*Object, error) {
	m, err := o.Attr(name)
	if err != nil {
		return nil, err
	}
	defer m.DecRef()
	return m.CallKw(args, kwargs)
}

// GetItem implements obj[key] for any indexable/subscriptable Python object.
func (o *Object) GetItem(key any) (*Object, error) {
	var obj *Object
	var err error
	withGIL(func() {
		pk := toPy(key)
		if pk == nil {
			err = fmt.Errorf("bridge: cannot convert key (%T) to Python", key)
			return
		}
		defer C.Py_DecRef(pk)
		p := C.gopy_getitem(o.p, pk)
		if p == nil {
			err = lastError()
			return
		}
		obj = &Object{p: p}
	})
	return obj, err
}

// Len returns len(obj). Errors when the object has no length.
func (o *Object) Len() (int, error) {
	var n int
	var err error
	withGIL(func() {
		r := C.gopy_len(o.p)
		if r < 0 {
			err = lastError()
			return
		}
		n = int(r)
	})
	return n, err
}

// Bool returns the Python truthiness of the object (bool(obj)).
func (o *Object) Bool() (bool, error) {
	var b bool
	var err error
	withGIL(func() {
		r := C.gopy_is_true_obj(o.p)
		if r < 0 {
			err = lastError()
			return
		}
		b = r != 0
	})
	return b, err
}

// Str returns str(obj). Repr returns repr(obj).
func (o *Object) Str() (string, error)  { return o.strLike(false) }
func (o *Object) Repr() (string, error) { return o.strLike(true) }

func (o *Object) strLike(repr bool) (string, error) {
	var s string
	var err error
	withGIL(func() {
		var p *C.PyObject
		if repr {
			p = C.gopy_repr(o.p)
		} else {
			p = C.gopy_str(o.p)
		}
		if p == nil {
			err = lastError()
			return
		}
		defer C.Py_DecRef(p)
		s = C.GoString(C.PyUnicode_AsUTF8(p))
	})
	return s, err
}

// Go converts the wrapped PyObject back to a native Go value.
func (o *Object) Go() (any, error) {
	var v any
	var err error
	withGIL(func() {
		v, err = fromPy(o.p)
	})
	return v, err
}

// DecRef releases the underlying reference. Idempotent-ish: nil pointer is a
// no-op. Not safe to call concurrently with use of the same Object.
func (o *Object) DecRef() {
	if o == nil || o.p == nil {
		return
	}
	withGIL(func() {
		C.Py_DecRef(o.p)
	})
	o.p = nil
}

// toPy converts a Go value to a new (owned) *PyObject. Must hold the GIL.
// Returns nil on unsupported types.
func toPy(v any) *C.PyObject {
	switch x := v.(type) {
	case nil:
		return C.gopy_none()
	case bool:
		var b C.long
		if x {
			b = 1
		}
		return C.PyBool_FromLong(b)
	case int:
		return C.PyLong_FromLongLong(C.longlong(x))
	case int64:
		return C.PyLong_FromLongLong(C.longlong(x))
	case float64:
		return C.PyFloat_FromDouble(C.double(x))
	case string:
		cs := C.CString(x)
		defer C.free(unsafe.Pointer(cs))
		return C.PyUnicode_FromStringAndSize(cs, C.Py_ssize_t(len(x)))
	case []any:
		lst := C.PyList_New(C.Py_ssize_t(len(x)))
		if lst == nil {
			return nil
		}
		for i, e := range x {
			pe := toPy(e)
			if pe == nil {
				C.Py_DecRef(lst)
				return nil
			}
			C.PyList_SetItem(lst, C.Py_ssize_t(i), pe) // steals ref
		}
		return lst
	case map[string]any:
		d := C.PyDict_New()
		if d == nil {
			return nil
		}
		for k, val := range x {
			pk := toPy(k)
			pv := toPy(val)
			if pk == nil || pv == nil {
				C.Py_XDECREF(pk)
				C.Py_XDECREF(pv)
				C.Py_DecRef(d)
				return nil
			}
			C.PyDict_SetItem(d, pk, pv) // does NOT steal — decref ours
			C.Py_DecRef(pk)
			C.Py_DecRef(pv)
		}
		return d
	}
	return nil
}

// fromPy converts a *PyObject to a Go value. Must hold the GIL.
func fromPy(p *C.PyObject) (any, error) {
	if p == nil {
		return nil, fmt.Errorf("bridge: nil PyObject")
	}
	if C.gopy_is_none(p) != 0 {
		return nil, nil
	}
	if C.gopy_is_true(p) != 0 {
		return true, nil
	}
	if C.gopy_bool_check(p) != 0 {
		return false, nil
	}
	if C.gopy_long_check(p) != 0 {
		return int64(C.PyLong_AsLongLong(p)), nil
	}
	if C.gopy_float_check(p) != 0 {
		return float64(C.PyFloat_AsDouble(p)), nil
	}
	if C.gopy_unicode_check(p) != 0 {
		return C.GoString(C.PyUnicode_AsUTF8(p)), nil
	}
	if C.gopy_list_check(p) != 0 || C.gopy_tuple_check(p) != 0 {
		n := int(C.PySequence_Size(p))
		out := make([]any, n)
		for i := 0; i < n; i++ {
			item := C.PySequence_GetItem(p, C.Py_ssize_t(i)) // new ref
			gv, err := fromPy(item)
			C.Py_DecRef(item)
			if err != nil {
				return nil, err
			}
			out[i] = gv
		}
		return out, nil
	}
	if C.gopy_dict_check(p) != 0 {
		out := map[string]any{}
		keys := C.PyDict_Keys(p) // new ref
		defer C.Py_DecRef(keys)
		n := int(C.PySequence_Size(keys))
		for i := 0; i < n; i++ {
			k := C.PySequence_GetItem(keys, C.Py_ssize_t(i))
			v := C.PyObject_GetItem(p, k)
			ks := C.GoString(C.PyUnicode_AsUTF8(k))
			gv, err := fromPy(v)
			C.Py_DecRef(k)
			C.Py_DecRef(v)
			if err != nil {
				return nil, err
			}
			out[ks] = gv
		}
		return out, nil
	}
	// Fallback: str() the object so callers at least see something.
	s := C.PyObject_Str(p)
	if s == nil {
		return nil, lastError()
	}
	defer C.Py_DecRef(s)
	return C.GoString(C.PyUnicode_AsUTF8(s)), nil
}
