//go:build cgo

// Reverse bridge: expose Go functions to the embedded CPython interpreter as
// callable Python objects. This is the direction frameworks need — a Python
// library (a web framework, a callback registry, a plugin host) calling back
// into user code. The forward bridge (bridge.go) is Go→Python; this is
// Python→Go.
//
// Mechanism: every registered Go callback is appended to a registry and gets
// a small integer id. A single C trampoline (a PyCFunction) is wrapped in a
// PyCFunction object whose bound `self` carries that id as a PyLong. When
// Python calls it, the trampoline reads the id from `self` and hands the
// argument tuple to an //export-ed Go dispatcher, which runs the callback and
// converts the result back to a PyObject.

package bridge

/*
#include <Python.h>

// Forward declaration of the Go dispatcher (see //export below).
extern PyObject* gopyRevDispatch(long id, PyObject* args);

// The trampoline every exposed Go function shares. `self` is the PyLong id
// bound when the callable was created; `args` is the positional tuple.
static PyObject* gopy_trampoline(PyObject* self, PyObject* args) {
    long id = PyLong_AsLong(self);
    return gopyRevDispatch(id, args);
}

// One static method definition is enough — all exposed callbacks share the
// same trampoline and METH_VARARGS calling convention; the per-function id
// lives in the bound `self`, not here.
static PyMethodDef gopy_revdef = {"gopy_callback", gopy_trampoline, METH_VARARGS, "gopy reverse-bridge callback"};

static PyObject* gopy_make_callable(long id) {
    PyObject* idObj = PyLong_FromLong(id);
    if (!idObj) return NULL;
    PyObject* fn = PyCFunction_New(&gopy_revdef, idObj);
    Py_DECREF(idObj); // PyCFunction_New takes its own reference to self
    return fn;
}
*/
import "C"

import (
	"fmt"
	"sync"
	"unsafe"
)

// revFunc is a registered Go callback. It receives the Python positional args
// (already converted to native Go values) and returns a value to convert back
// to Python. Panicking inside is caught and surfaced as a Python exception.
type revFunc func(args []any) any

var (
	revMu    sync.Mutex
	revFuncs []revFunc
)

// RegisterFunc exposes a Go callback to Python and returns it as a callable
// *Object — pass it to Python code, store it in a module, hand it to a
// framework. The returned object is callable from Python with positional
// arguments.
func RegisterFunc(fn revFunc) (*Object, error) {
	if err := Init(); err != nil {
		return nil, err
	}
	revMu.Lock()
	id := len(revFuncs)
	revFuncs = append(revFuncs, fn)
	revMu.Unlock()

	var obj *Object
	withGIL(func() {
		p := C.gopy_make_callable(C.long(id))
		if p != nil {
			obj = &Object{p: p}
		}
	})
	if obj == nil {
		return nil, lastError()
	}
	return obj, nil
}

//export gopyRevDispatch
func gopyRevDispatch(id C.long, args *C.PyObject) *C.PyObject {
	// Runs while CPython holds the GIL (we're called synchronously from the
	// trampoline inside a Python call), so we must not re-acquire it.
	revMu.Lock()
	var fn revFunc
	if int(id) >= 0 && int(id) < len(revFuncs) {
		fn = revFuncs[id]
	}
	revMu.Unlock()
	if fn == nil {
		setPyError("gopy reverse-bridge: unknown callback id")
		return nil
	}

	// Convert the positional args tuple to native Go values.
	n := int(C.PySequence_Size(args))
	goArgs := make([]any, n)
	for i := 0; i < n; i++ {
		item := C.PySequence_GetItem(args, C.Py_ssize_t(i)) // new ref
		v, err := fromPy(item)
		C.Py_DecRef(item)
		if err != nil {
			setPyError("gopy reverse-bridge: arg conversion: " + err.Error())
			return nil
		}
		goArgs[i] = v
	}

	// Run the callback, turning a Go panic into a Python exception rather
	// than crashing the interpreter.
	var result any
	if err := callSafely(fn, goArgs, &result); err != nil {
		setPyError("gopy reverse-bridge: " + err.Error())
		return nil
	}
	out := toPy(result)
	if out == nil {
		setPyError("gopy reverse-bridge: cannot convert return value to Python")
		return nil
	}
	return out
}

// callSafely runs fn(args), recovering any panic into an error.
func callSafely(fn revFunc, args []any, result *any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = panicError(r)
		}
	}()
	*result = fn(args)
	return nil
}

// panicError renders a recovered panic value as an error.
func panicError(r any) error {
	if e, ok := r.(error); ok {
		return e
	}
	return fmt.Errorf("%v", r)
}

// setPyError raises a Python RuntimeError with the given message. Must be
// called while holding the GIL (the dispatcher always is).
func setPyError(msg string) {
	cs := C.CString(msg)
	defer C.free(unsafe.Pointer(cs))
	C.PyErr_SetString(C.PyExc_RuntimeError, cs)
}
