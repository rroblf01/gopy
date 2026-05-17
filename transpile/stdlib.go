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
	"re": {
		Funcs: map[string]stdlibFunc{
			"findall": {GoFunc: "__gopy_re_findall", GoImport: "regexp", Helper: helperReFindall},
			"search":  {GoFunc: "__gopy_re_search", GoImport: "regexp", Helper: helperReSearch, RetTag: "__Match", ExtraHelpers: map[string]string{"__Match": helperMatchType}},
			"match":   {GoFunc: "__gopy_re_match", GoImport: "regexp", Helper: helperReMatch, RetTag: "__Match", ExtraHelpers: map[string]string{"__Match": helperMatchType}},
			"sub":     {GoFunc: "__gopy_re_sub", GoImport: "regexp", Helper: helperReSub},
		},
	},
	"csv": {
		Funcs: map[string]stdlibFunc{
			"reader": {GoFunc: "__gopy_csv_reader", GoImport: "encoding/csv", Helper: helperCSVReader, HelperImports: []string{"strings"}},
		},
	},
	"pathlib": {
		Funcs: map[string]stdlibFunc{
			"Path": {GoFunc: "__gopy_path_new", GoImport: "os", Helper: helperPathNew, RetTag: "__Path", ExtraHelpers: map[string]string{"__Path": helperPathType}, HelperImports: []string{"os"}},
		},
	},
	"datetime": {
		Funcs: map[string]stdlibFunc{
			"timedelta": {GoFunc: "__gopy_timedelta_new", GoImport: "time", Helper: helperTimedeltaNew, RetTag: "__Timedelta", ExtraHelpers: map[string]string{"__Timedelta": helperTimedeltaType}, HelperImports: []string{"fmt"}},
		},
		Subs: map[string]stdlibModule{
			"datetime": {
				Funcs: map[string]stdlibFunc{
					// __Datetime methods reference __Timedelta (for Add/Sub),
					// so we always emit both types whenever datetime.now() is
					// used; otherwise Go would error on the undefined type.
					"now": {GoFunc: "__gopy_datetime_now", GoImport: "time", Helper: helperDatetimeNow, RetTag: "__Datetime", ExtraHelpers: map[string]string{"__Datetime": helperDatetimeType, "__Timedelta": helperTimedeltaType}, HelperImports: []string{"fmt"}},
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
	// RetTag is a stable tag for the function's return type. When non-empty,
	// the codegen records it under the assigned variable so subsequent
	// MethodCall / If / `is None` checks can dispatch by type. Tags:
	//   "__Match"     — re.search / re.match
	//   "__Path"      — pathlib.Path constructor
	//   "__Timedelta" — datetime.timedelta constructor
	RetTag string
	// ExtraHelpers lists additional helper definitions and matching keys to
	// emit when this function is used. Each helper is keyed by its own
	// stable name to avoid duplication.
	ExtraHelpers map[string]string
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

// helperReFindall mirrors Python's re.findall(pattern, string): returns
// every non-overlapping match as a []string. Go's regexp uses RE2 syntax
// so user patterns relying on backrefs / lookarounds will fail at compile.
const helperReFindall = `func __gopy_re_findall(pattern, s string) []string {
	r := regexp.MustCompile(pattern)
	out := r.FindAllString(s, -1)
	if out == nil {
		return []string{}
	}
	return out
}`

// helperMatchType is the runtime Match struct used by re.search / re.match.
// Methods mirror a (very) thin subset of Python's Match: Group(n) returns
// the full match for n=0 or empty, and capture n otherwise. Strings round
// through fmt as the match text so callers can pass Match directly to
// __gopy_print.
const helperMatchType = `type __Match struct {
	full   string
	groups []string
}

func (m *__Match) Group(n ...int) string {
	if len(n) == 0 || n[0] == 0 {
		return m.full
	}
	if n[0] < 1 || n[0] > len(m.groups) {
		return ""
	}
	return m.groups[n[0]-1]
}

func (m *__Match) Groups() []string { return m.groups }

func (m *__Match) String() string { return m.full }`

// helperReSearch returns a *__Match on hit, nil on miss — mirroring
// Python's re.search semantics. Truthy / `is None` checks at call sites
// work because the codegen rewrites them to a nil comparison.
const helperReSearch = `func __gopy_re_search(pattern, s string) *__Match {
	r := regexp.MustCompile(pattern)
	parts := r.FindStringSubmatch(s)
	if parts == nil {
		return nil
	}
	return &__Match{full: parts[0], groups: parts[1:]}
}`

// helperReMatch anchors the pattern to the start of the string, matching
// Python's re.match semantics. Returns nil on miss.
const helperReMatch = `func __gopy_re_match(pattern, s string) *__Match {
	r := regexp.MustCompile("^(?:" + pattern + ")")
	parts := r.FindStringSubmatch(s)
	if parts == nil {
		return nil
	}
	return &__Match{full: parts[0], groups: parts[1:]}
}`

// helperReSub replaces every match of pattern with repl.
const helperReSub = `func __gopy_re_sub(pattern, repl, s string) string {
	r := regexp.MustCompile(pattern)
	return r.ReplaceAllString(s, repl)
}`

// helperCSVReader materializes Python's `csv.reader(iterable_of_lines)`
// as a list of rows. CPython returns an iterator; we return a slice to
// keep the shim simple and to match common idioms (`for row in
// csv.reader(lines)`, `list(csv.reader(lines))`). Pass already-split
// lines (each without a trailing newline) for parity with CPython.
const helperCSVReader = `func __gopy_csv_reader(lines []string) [][]string {
	r := csv.NewReader(strings.NewReader(strings.Join(lines, "\n")))
	rows, err := r.ReadAll()
	if err != nil {
		panic(err)
	}
	return rows
}`

// helperCSVWriter renders a list of rows to a single CSV-formatted string.
// CPython's csv.writer is stateful and bound to a file-like object; the
// gopy shim takes the rows directly to avoid pulling in the StringIO
// machinery.
const helperCSVWriter = `func __gopy_csv_writer(rows [][]string) string {
	var b strings.Builder
	w := csv.NewWriter(&b)
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			panic(err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		panic(err)
	}
	return b.String()
}`

// helperPathType is the runtime Path struct used by pathlib.Path.
// Mirrors a handful of Path methods sufficient for "open this, check
// existence, read/write text" workflows.
const helperPathType = `type __Path struct{ p string }

func (p *__Path) Exists() bool {
	_, err := os.Stat(p.p)
	return err == nil
}

func (p *__Path) IsFile() bool {
	i, err := os.Stat(p.p)
	return err == nil && !i.IsDir()
}

func (p *__Path) IsDir() bool {
	i, err := os.Stat(p.p)
	return err == nil && i.IsDir()
}

func (p *__Path) ReadText() string {
	b, err := os.ReadFile(p.p)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (p *__Path) WriteText(s string) {
	if err := os.WriteFile(p.p, []byte(s), 0o644); err != nil {
		panic(err)
	}
}

func (p *__Path) String() string { return p.p }`

const helperPathNew = `func __gopy_path_new(s string) *__Path { return &__Path{p: s} }`

// helperTimedeltaType mirrors Python's str(timedelta(days=...)) output
// so cross-runtime fixtures can print the value directly. Supports
// only the positional-days constructor in F6-fix; richer kwargs land
// once general kwargs support exists.
const helperTimedeltaType = `type __Timedelta struct{ d time.Duration }

func (t *__Timedelta) String() string {
	d := t.d
	days := int(d / (24 * time.Hour))
	rem := d - time.Duration(days)*24*time.Hour
	h := int(rem / time.Hour)
	m := int((rem % time.Hour) / time.Minute)
	s := int((rem % time.Minute) / time.Second)
	if days == 1 {
		return fmt.Sprintf("1 day, %d:%02d:%02d", h, m, s)
	}
	if days != 0 {
		return fmt.Sprintf("%d days, %d:%02d:%02d", days, h, m, s)
	}
	return fmt.Sprintf("%d:%02d:%02d", h, m, s)
}`

const helperTimedeltaNew = `func __gopy_timedelta_new(days int64) *__Timedelta {
	return &__Timedelta{d: time.Duration(days) * 24 * time.Hour}
}`

// helperDatetimeType is the runtime Datetime struct used by
// datetime.datetime.now(). String() matches CPython's default __str__
// (microsecond precision), so f-strings and `str(dt)` round-trip across
// runtimes. Add/Sub support timedelta arithmetic via BinOp rewriting.
const helperDatetimeType = `type __Datetime struct{ t time.Time }

func (d *__Datetime) String() string {
	return d.t.Format("2006-01-02 15:04:05.000000")
}

func (d *__Datetime) Add(td *__Timedelta) *__Datetime {
	return &__Datetime{t: d.t.Add(td.d)}
}

func (d *__Datetime) Sub(other *__Datetime) *__Timedelta {
	return &__Timedelta{d: d.t.Sub(other.t)}
}

func (d *__Datetime) SubTimedelta(td *__Timedelta) *__Datetime {
	return &__Datetime{t: d.t.Add(-td.d)}
}

func (d *__Datetime) Year() int64   { return int64(d.t.Year()) }
func (d *__Datetime) Month() int64  { return int64(d.t.Month()) }
func (d *__Datetime) Day() int64    { return int64(d.t.Day()) }
func (d *__Datetime) Hour() int64   { return int64(d.t.Hour()) }
func (d *__Datetime) Minute() int64 { return int64(d.t.Minute()) }
func (d *__Datetime) Second() int64 { return int64(d.t.Second()) }

func (d *__Datetime) Isoformat() string {
	return d.t.Format("2006-01-02T15:04:05.000000")
}`

// helperDatetimeNow returns Python's datetime.datetime.now() as a
// *__Datetime so it can take part in timedelta arithmetic.
const helperDatetimeNow = `func __gopy_datetime_now() *__Datetime { return &__Datetime{t: time.Now()} }`

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
