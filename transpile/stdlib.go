package transpile

// stdlibModules lists Python stdlib modules we know how to translate.
// Each entry maps Python attribute/method names to Go expressions and
// records the Go import path needed.
//
// Three flavours of entry:
//
//   - Attrs: attribute access  (e.g. `sys.argv`)        — emit GoExpr
//   - Funcs: function/method call (e.g. `os.getenv(k)`) — emit GoFunc
//   - Subs:  nested module/class (e.g. `datetime.datetime`) — resolved
//            through alias maps or two-level Attribute chains
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
			"time":  {GoFunc: "__gopy_time_now_seconds", GoImport: "time", Helper: helperTimeNowSeconds},
			"sleep": {GoFunc: "__gopy_time_sleep", GoImport: "time", Helper: helperTimeSleep},
		},
	},
	"json": {
		Funcs: map[string]stdlibFunc{
			"dumps": {GoFunc: "__gopy_json_dumps", GoImport: "encoding/json", Helper: helperJSONDumps, HelperImports: []string{"strings"}},
			"loads": {GoFunc: "__gopy_json_loads", GoImport: "encoding/json", Helper: helperJSONLoads},
		},
	},
	"datetime": {
		Subs: map[string]stdlibModule{
			"datetime": {
				Funcs: map[string]stdlibFunc{
					"now": {GoFunc: "__gopy_datetime_now", GoImport: "time", Helper: helperDatetimeNow},
				},
			},
		},
	},
}

type stdlibModule struct {
	Attrs map[string]stdlibAttr
	Funcs map[string]stdlibFunc
	Subs  map[string]stdlibModule
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
	// HelperImports lists additional Go imports the Helper body relies on.
	// They are added to the output only when this function is used.
	HelperImports []string
}

const helperTimeNowSeconds = `func __gopy_time_now_seconds() float64 { return float64(time.Now().UnixNano()) / 1e9 }`

const helperTimeSleep = `func __gopy_time_sleep(seconds float64) { time.Sleep(time.Duration(seconds * float64(time.Second))) }`

// helperJSONDumps mirrors CPython's json.dumps default separators of
// `, ` and `: `. Go's encoding/json emits compact JSON, so we reformat
// the result outside of any string literal.
const helperJSONDumps = `func __gopy_json_dumps(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return __gopy_json_pythonize(string(b))
}

func __gopy_json_pythonize(s string) string {
	var out strings.Builder
	inStr := false
	escape := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escape {
			out.WriteByte(c)
			escape = false
			continue
		}
		if inStr && c == '\\' {
			out.WriteByte(c)
			escape = true
			continue
		}
		if c == '"' {
			inStr = !inStr
			out.WriteByte(c)
			continue
		}
		if !inStr && c == ',' {
			out.WriteString(", ")
			continue
		}
		if !inStr && c == ':' {
			out.WriteString(": ")
			continue
		}
		out.WriteByte(c)
	}
	return out.String()
}`

const helperJSONLoads = `func __gopy_json_loads(s string) any {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		panic(err)
	}
	return v
}`

// helperDatetimeNow returns Python's datetime.datetime.now() in a form
// that round-trips through CPython for ISO-format printing. The format
// matches CPython's default `datetime.now()` __str__ output to one
// microsecond precision.
const helperDatetimeNow = `func __gopy_datetime_now() string { return time.Now().Format("2006-01-02 15:04:05.000000") }`

// isStdlibModule reports whether name refers to a stdlib module we recognize.
func isStdlibModule(name string) bool {
	_, ok := stdlibModules[name]
	return ok
}

// stdlibPathOf walks an IR expression that should look like a chain of
// Attribute(...) nodes terminating in a Name, and returns the dotted
// stdlib path if every component resolves under stdlibModules — applying
// the given alias map at the leaf. Examples (aliases empty):
//
//	Name("os")                              → "os", true
//	Attribute(Name("datetime"), "datetime") → "datetime.datetime", true
//
// With aliases {"datetime": "datetime.datetime"}, a bare Name("datetime")
// also resolves to "datetime.datetime".
//
// To avoid importing the ir package here (we already do in transpile.go),
// the actual implementation lives in transpile.go alongside its use site.


// lookupStdlibFunc resolves a dotted path like "datetime.datetime.now"
// (where the prefix may be an aliased name) to its stdlibFunc entry.
// Returns nil when the path does not resolve.
func lookupStdlibFunc(path, method string) *stdlibFunc {
	parts := splitDotted(path)
	cur, ok := stdlibModules[parts[0]]
	if !ok {
		return nil
	}
	for _, p := range parts[1:] {
		sub, ok := cur.Subs[p]
		if !ok {
			return nil
		}
		cur = sub
	}
	if fn, ok := cur.Funcs[method]; ok {
		return &fn
	}
	return nil
}

func splitDotted(s string) []string {
	var parts []string
	cur := ""
	for _, r := range s {
		if r == '.' {
			parts = append(parts, cur)
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		parts = append(parts, cur)
	}
	return parts
}
