//go:build cgo

package bridge

import (
	"sync"
	"testing"
)

// counterClassSrc defines a small stateful Python class used to test rich
// object handles and re-entrant bridge calls.
const counterClassSrc = `
class Counter:
    def __init__(self, start=0):
        self.n = start
    def inc(self):
        self.n += 1
        return self.n
`

func makeCounter(t *testing.T, start int64) *Object {
	t.Helper()
	builtins, err := Import("builtins")
	if err != nil {
		t.Fatalf("import builtins: %v", err)
	}
	defer builtins.DecRef()
	ns, err := builtins.CallMethod("dict")
	if err != nil {
		t.Fatalf("dict(): %v", err)
	}
	defer ns.DecRef()
	if _, err := builtins.CallMethod("exec", counterClassSrc, ns); err != nil {
		t.Fatalf("exec Counter: %v", err)
	}
	cls, err := ns.GetItem("Counter")
	if err != nil {
		t.Fatalf("ns[Counter]: %v", err)
	}
	defer cls.DecRef()
	c, err := cls.Call(start)
	if err != nil {
		t.Fatalf("Counter(%d): %v", start, err)
	}
	return c
}

// TestRichObjectHandle confirms an unrecognized Python object round-trips
// through fromPy as a live *Object that Go can keep operating on.
func TestRichObjectHandle(t *testing.T) {
	c := makeCounter(t, 41)
	defer c.DecRef()

	// .Go() on a rich object yields an *Object handle, not a string.
	r, err := c.CallMethod("inc")
	if err != nil {
		t.Fatalf("c.inc(): %v", err)
	}
	defer r.DecRef()
	v, _ := r.Go()
	if v != int64(42) {
		t.Fatalf("c.inc() = %v, want 42", v)
	}
}

// TestReentrantBridge is the case the old non-reentrant gilMu deadlocked on:
// a Go reverse callback that, while being invoked from Python, calls back into
// the forward bridge (here: it calls .inc() on the Counter object it was
// handed). Python → Go → Python must not deadlock.
func TestReentrantBridge(t *testing.T) {
	c := makeCounter(t, 0)
	defer c.DecRef()

	// Go callback receives the Counter as a rich *Object and calls .inc()
	// on it twice — re-entering the forward bridge from inside a reverse call.
	bump, err := RegisterFunc(func(args []any, kwargs map[string]any) any {
		obj, ok := args[0].(*Object)
		if !ok {
			t.Errorf("callback arg is %T, want *Object", args[0])
			return nil
		}
		r1, err := obj.CallMethod("inc")
		if err != nil {
			t.Errorf("reentrant inc 1: %v", err)
			return nil
		}
		defer r1.DecRef()
		r2, err := obj.CallMethod("inc")
		if err != nil {
			t.Errorf("reentrant inc 2: %v", err)
			return nil
		}
		defer r2.DecRef()
		v, _ := r2.Go()
		return v
	})
	if err != nil {
		t.Fatalf("RegisterFunc: %v", err)
	}
	defer bump.DecRef()

	// Invoke from Python: bump(counter) → counter.inc() twice → 2.
	res, err := bump.Call(c)
	if err != nil {
		t.Fatalf("bump(counter): %v", err)
	}
	defer res.DecRef()
	v, _ := res.Go()
	if v != int64(2) {
		t.Fatalf("reentrant bump = %v, want 2", v)
	}
}

// TestConcurrentGIL hammers the bridge from many goroutines to confirm the
// GIL-only serialization (no Go mutex) stays correct under contention.
func TestConcurrentGIL(t *testing.T) {
	math, err := Import("math")
	if err != nil {
		t.Fatalf("import math: %v", err)
	}
	defer math.DecRef()

	var wg sync.WaitGroup
	errs := make(chan error, 50)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := math.CallMethod("sqrt", 144.0)
			if err != nil {
				errs <- err
				return
			}
			v, _ := r.Go()
			r.DecRef()
			if v != 12.0 {
				errs <- &stringErr{"sqrt(144) != 12"}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatalf("concurrent: %v", e)
	}
}

type stringErr struct{ s string }

func (e *stringErr) Error() string { return e.s }
