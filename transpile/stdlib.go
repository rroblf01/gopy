package transpile

// stdlibModules lists Python stdlib modules we know how to translate.
// Each entry maps Python attribute/method names to Go expressions and
// records the Go import path needed.
//
// Two flavours of entry:
//
//   - attribute access  (e.g. `sys.argv`):       use stdlibAttr.GoExpr
//   - function/method call (e.g. `os.getenv(k)`): use stdlibFunc.GoFunc + arity
//
// The codegen consults this table before falling back to "emit receiver
// dot method" so unrecognized references fail fast with a clear error.
var stdlibModules = map[string]stdlibModule{
	"sys": {
		Attrs: map[string]stdlibAttr{
			"argv": {GoExpr: "os.Args", GoImport: "os"},
		},
		Funcs: map[string]stdlibFunc{
			"exit": {GoFunc: "os.Exit", GoImport: "os", IntArg0: true},
		},
	},
	"os": {
		Funcs: map[string]stdlibFunc{
			"getenv": {GoFunc: "os.Getenv", GoImport: "os"},
		},
	},
	"time": {
		Funcs: map[string]stdlibFunc{
			// time.time() → seconds since epoch as float
			"time": {GoFunc: "__gopy_time_now_seconds", GoImport: "time", Helper: helperTimeNowSeconds},
			// time.sleep(s) where s is seconds (float or int)
			"sleep": {GoFunc: "__gopy_time_sleep", GoImport: "time", Helper: helperTimeSleep},
		},
	},
}

type stdlibModule struct {
	Attrs map[string]stdlibAttr
	Funcs map[string]stdlibFunc
}

type stdlibAttr struct {
	GoExpr   string
	GoImport string
}

type stdlibFunc struct {
	GoFunc   string
	GoImport string
	// IntArg0, if set, wraps the first arg in int(...) — useful for things
	// like sys.exit(n) where the Python n is int64 but os.Exit expects int.
	IntArg0 bool
	// Helper is the source of an inline helper function that the generated
	// program must include in order to call GoFunc. Empty means no helper.
	Helper string
}

const helperTimeNowSeconds = `func __gopy_time_now_seconds() float64 { return float64(time.Now().UnixNano()) / 1e9 }`

const helperTimeSleep = `func __gopy_time_sleep(seconds float64) { time.Sleep(time.Duration(seconds * float64(time.Second))) }`

// isStdlibModule reports whether name refers to a stdlib module we recognize.
func isStdlibModule(name string) bool {
	_, ok := stdlibModules[name]
	return ok
}
