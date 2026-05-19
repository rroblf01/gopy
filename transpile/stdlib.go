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
	"math": {
		Attrs: map[string]stdlibAttr{
			"pi": {GoExpr: "math.Pi", GoImport: "math"},
			"e":  {GoExpr: "math.E", GoImport: "math"},
			"inf": {GoExpr: "math.Inf(1)", GoImport: "math"},
		},
		Funcs: map[string]stdlibFunc{
			"sqrt":  {GoFunc: "math.Sqrt", GoImport: "math"},
			"floor": {GoFunc: "__gopy_math_floor", GoImport: "math", Helper: helperMathFloor},
			"ceil":  {GoFunc: "__gopy_math_ceil", GoImport: "math", Helper: helperMathCeil},
			"log":   {GoFunc: "math.Log", GoImport: "math"},
			"log2":  {GoFunc: "math.Log2", GoImport: "math"},
			"log10": {GoFunc: "math.Log10", GoImport: "math"},
			"exp":   {GoFunc: "math.Exp", GoImport: "math"},
			"sin":   {GoFunc: "math.Sin", GoImport: "math"},
			"cos":   {GoFunc: "math.Cos", GoImport: "math"},
			"tan":   {GoFunc: "math.Tan", GoImport: "math"},
			"atan":  {GoFunc: "math.Atan", GoImport: "math"},
			"atan2": {GoFunc: "math.Atan2", GoImport: "math"},
			"pow":   {GoFunc: "math.Pow", GoImport: "math"},
		},
	},
	"hashlib": {
		Funcs: map[string]stdlibFunc{
			"sha256": {GoFunc: "__gopy_hashlib_sha256", GoImport: "crypto/sha256", Helper: helperHashlibSha256, RetTag: "__Hasher", ExtraHelpers: map[string]string{"__Hasher": helperHasherType}, HelperImports: []string{"encoding/hex", "crypto/md5"}},
			"md5":    {GoFunc: "__gopy_hashlib_md5", GoImport: "crypto/md5", Helper: helperHashlibMd5, RetTag: "__Hasher", ExtraHelpers: map[string]string{"__Hasher": helperHasherType}, HelperImports: []string{"encoding/hex", "crypto/sha256"}},
		},
	},
	"base64": {
		Funcs: map[string]stdlibFunc{
			"b64encode": {GoFunc: "__gopy_b64encode", GoImport: "encoding/base64", Helper: helperB64Encode, RetKind: "str"},
			"b64decode": {GoFunc: "__gopy_b64decode", GoImport: "encoding/base64", Helper: helperB64Decode, RetKind: "str"},
		},
	},
	"urllib": {
		Subs: map[string]stdlibModule{
			"parse": {
				Funcs: map[string]stdlibFunc{
					"quote":     {GoFunc: "__gopy_url_quote", GoImport: "net/url", Helper: helperURLQuote, HelperImports: []string{"strings", "fmt"}, RetKind: "str"},
					"unquote":   {GoFunc: "__gopy_url_unquote", GoImport: "net/url", Helper: helperURLUnquote, RetKind: "str"},
					"urlencode": {GoFunc: "__gopy_url_urlencode", GoImport: "net/url", Helper: helperURLUrlencode, HelperImports: []string{"strings"}, RetKind: "str"},
					"urlparse":  {GoFunc: "__gopy_url_urlparse", GoImport: "net/url", Helper: helperURLUrlparse, RetTag: "__URLParseResult", ExtraHelpers: map[string]string{"__URLParseResult": helperURLParseResultType}},
				},
			},
		},
	},
	"string": {
		Attrs: map[string]stdlibAttr{
			"ascii_lowercase": {GoExpr: `"abcdefghijklmnopqrstuvwxyz"`},
			"ascii_uppercase": {GoExpr: `"ABCDEFGHIJKLMNOPQRSTUVWXYZ"`},
			"ascii_letters":   {GoExpr: `"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"`},
			"digits":          {GoExpr: `"0123456789"`},
			"hexdigits":       {GoExpr: `"0123456789abcdefABCDEF"`},
			"octdigits":       {GoExpr: `"01234567"`},
			"punctuation":     {GoExpr: "\"!\\\"#$%&'()*+,-./:;<=>?@[\\\\]^_`{|}~\""},
			"whitespace":      {GoExpr: `" \t\n\r\f\v"`},
		},
	},
	"collections": {
		// Counter and defaultdict are handled by per-arg-type builders
		// in transpile.go; entries below are stubs so alias resolution
		// succeeds for the call expressions.
		Funcs: map[string]stdlibFunc{
			"Counter":     {GoFunc: "__gopy_counter_unused"},
			"defaultdict": {GoFunc: "__gopy_defaultdict_unused"},
			"OrderedDict": {GoFunc: "__gopy_ordereddict_unused"},
			"deque":       {GoFunc: "__gopy_deque_unused", RetTag: "__Deque"},
		},
	},
	"subprocess": {
		// run() needs to ignore Python kwargs (capture_output, text, ...)
		// that don't have a Go equivalent. Dispatch lives in transpile.go.
		Funcs: map[string]stdlibFunc{
			"run": {GoFunc: "__gopy_subprocess_run_unused", RetTag: "__CompletedProcess"},
		},
	},
	"functools": {
		Funcs: map[string]stdlibFunc{
			// reduce uses an inline lambda for the binary op; dispatch
			// lives in transpile.go's call() builder.
			"reduce":  {GoFunc: "__gopy_reduce_unused"},
			"partial": {GoFunc: "__gopy_partial_unused"},
		},
	},
	"logging": {
		Funcs: map[string]stdlibFunc{
			"debug":       {GoFunc: "__gopy_log_debug", GoImport: "fmt", Helper: helperLogDebug, HelperImports: []string{"os"}},
			"info":        {GoFunc: "__gopy_log_info", GoImport: "fmt", Helper: helperLogInfo, HelperImports: []string{"os"}},
			"warning":     {GoFunc: "__gopy_log_warning", GoImport: "fmt", Helper: helperLogWarning, HelperImports: []string{"os"}},
			"error":       {GoFunc: "__gopy_log_error", GoImport: "fmt", Helper: helperLogError, HelperImports: []string{"os"}},
			"critical":    {GoFunc: "__gopy_log_critical", GoImport: "fmt", Helper: helperLogCritical, HelperImports: []string{"os"}},
			"basicConfig": {GoFunc: "__gopy_log_basicConfig", Helper: helperLogBasicConfig},
		},
	},
	"itertools": {
		Funcs: map[string]stdlibFunc{
			"chain":        {GoFunc: "__gopy_chain_unused"},
			"accumulate":   {GoFunc: "__gopy_accumulate_unused"},
			"takewhile":    {GoFunc: "__gopy_takewhile_unused"},
			"dropwhile":    {GoFunc: "__gopy_dropwhile_unused"},
			"combinations": {GoFunc: "__gopy_combinations_unused"},
			"product":      {GoFunc: "__gopy_product_unused"},
			"groupby":      {GoFunc: "__gopy_groupby_unused"},
		},
	},
	"random": {
		Funcs: map[string]stdlibFunc{
			"random":  {GoFunc: "__gopy_random", GoImport: "math/rand", Helper: helperRandomFloat},
			"randint": {GoFunc: "__gopy_randint", GoImport: "math/rand", Helper: helperRandint},
			"seed":    {GoFunc: "__gopy_random_seed", GoImport: "math/rand", Helper: helperRandomSeed},
		},
	},
	"re": {
		Funcs: map[string]stdlibFunc{
			"findall": {GoFunc: "__gopy_re_findall", GoImport: "regexp", Helper: helperReFindall},
			"search":  {GoFunc: "__gopy_re_search", GoImport: "regexp", Helper: helperReSearch, RetTag: "__Match", ExtraHelpers: map[string]string{"__Match": helperMatchType}},
			"match":   {GoFunc: "__gopy_re_match", GoImport: "regexp", Helper: helperReMatch, RetTag: "__Match", ExtraHelpers: map[string]string{"__Match": helperMatchType}},
			"sub":     {GoFunc: "__gopy_re_sub", GoImport: "regexp", Helper: helperReSub},
			"compile": {GoFunc: "__gopy_re_compile", GoImport: "regexp", Helper: helperReCompile, RetTag: "__Pattern", ExtraHelpers: map[string]string{"__Pattern": helperPatternType, "__Match": helperMatchType}},
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
	// RetKind is the IR type kind returned by this stdlib function for
	// primitives (str / int / float / bool). Tagged types use RetTag
	// instead. Empty means unknown / no special handling.
	RetKind string
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
// Methods mirror a thin subset of Python's Match: Group accepts an int or
// a string (named group); Groups() and Groupdict() expose the captures by
// position and by name respectively. Strings round through fmt so callers
// can pass Match directly to __gopy_print.
const helperMatchType = `type __Match struct {
	full   string
	groups []string
	names  []string
}

func (m *__Match) Group(args ...any) string {
	if len(args) == 0 {
		return m.full
	}
	switch a := args[0].(type) {
	case int:
		if a == 0 {
			return m.full
		}
		if a < 1 || a > len(m.groups) {
			return ""
		}
		return m.groups[a-1]
	case int64:
		i := int(a)
		if i == 0 {
			return m.full
		}
		if i < 1 || i > len(m.groups) {
			return ""
		}
		return m.groups[i-1]
	case string:
		for i, n := range m.names {
			if n == a && i >= 1 && i <= len(m.groups) {
				return m.groups[i-1]
			}
		}
		return ""
	}
	return ""
}

func (m *__Match) Groups() []string { return m.groups }

func (m *__Match) Groupdict() map[string]string {
	out := map[string]string{}
	for i, n := range m.names {
		if n == "" || i < 1 || i > len(m.groups) {
			continue
		}
		out[n] = m.groups[i-1]
	}
	return out
}

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
	return &__Match{full: parts[0], groups: parts[1:], names: r.SubexpNames()}
}`

// helperReMatch anchors the pattern to the start of the string, matching
// Python's re.match semantics. Returns nil on miss.
const helperReMatch = `func __gopy_re_match(pattern, s string) *__Match {
	r := regexp.MustCompile("^(?:" + pattern + ")")
	parts := r.FindStringSubmatch(s)
	if parts == nil {
		return nil
	}
	return &__Match{full: parts[0], groups: parts[1:], names: r.SubexpNames()}
}`

// helperReSub replaces every match of pattern with repl.
const helperReSub = `func __gopy_re_sub(pattern, repl, s string) string {
	r := regexp.MustCompile(pattern)
	return r.ReplaceAllString(s, repl)
}`

// helperPatternType wraps a compiled regexp so re.compile(p).match(s)
// and friends share one re-usable underlying *regexp.Regexp. Method
// names match the (already-renamed) Match/Search/Findall/Sub forms.
const helperPatternType = `type __Pattern struct {
	r       *regexp.Regexp
	anchor  *regexp.Regexp
}

func (p *__Pattern) Match(s string) *__Match {
	m := p.anchor.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	return &__Match{full: m[0], groups: m[1:], names: p.r.SubexpNames()}
}

func (p *__Pattern) Search(s string) *__Match {
	m := p.r.FindStringSubmatch(s)
	if m == nil {
		return nil
	}
	return &__Match{full: m[0], groups: m[1:], names: p.r.SubexpNames()}
}

func (p *__Pattern) Findall(s string) []string {
	out := p.r.FindAllString(s, -1)
	if out == nil {
		return []string{}
	}
	return out
}

func (p *__Pattern) Sub(repl, s string) string {
	return p.r.ReplaceAllString(s, repl)
}`

const helperReCompile = `func __gopy_re_compile(pattern string) *__Pattern {
	return &__Pattern{
		r:      regexp.MustCompile(pattern),
		anchor: regexp.MustCompile("^(?:" + pattern + ")"),
	}
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

// helperHasherType bridges hashlib's hash objects. Both sha256 and md5
// build the same shape; the algo string drives Hexdigest's dispatch so
// fixtures can compare hex strings across CPython and Go.
const helperHasherType = `type __Hasher struct {
	data []byte
	algo string
}

func (h *__Hasher) Hexdigest() string {
	switch h.algo {
	case "sha256":
		sum := sha256.Sum256(h.data)
		return hex.EncodeToString(sum[:])
	case "md5":
		sum := md5.Sum(h.data)
		return hex.EncodeToString(sum[:])
	}
	return ""
}`

const helperHashlibSha256 = `func __gopy_hashlib_sha256(data string) *__Hasher {
	return &__Hasher{data: []byte(data), algo: "sha256"}
}`

const helperHashlibMd5 = `func __gopy_hashlib_md5(data string) *__Hasher {
	return &__Hasher{data: []byte(data), algo: "md5"}
}`

// helperB64Encode / helperB64Decode mirror Python's base64.b64encode /
// b64decode for str inputs. Python's API returns/accepts bytes; the
// gopy shim treats both ends as str so fixtures don't need a bytes type.
const helperB64Encode = `func __gopy_b64encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}`

const helperB64Decode = `func __gopy_b64decode(s string) string {
	out, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return string(out)
}`

// helperURLQuote mirrors CPython's urllib.parse.quote default safe=/:
// only ASCII letters, digits, and `_.-~/` pass through unescaped; every
// other byte renders as %XX. Go's net/url functions either turn space
// into `+` (QueryEscape) or leave `&` unescaped (PathEscape), so we
// roll our own.
const helperURLQuote = `func __gopy_url_quote(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		safe := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '_' || c == '.' || c == '-' || c == '~' || c == '/'
		if safe {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}`

// helperURLUrlencode mirrors CPython's urllib.parse.urlencode for the
// common dict[str, str] input shape. Output is &-joined key=value pairs
// with each part quoted under our `+` semantics (Python's default).
const helperURLUrlencode = `func __gopy_url_urlencode(d map[string]string) string {
	keys := make([]string, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}
	// urlencode iteration order in CPython follows insertion, which Go
	// maps don't preserve. Sort for deterministic output so fixtures
	// match across runtimes.
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(url.QueryEscape(k))
		b.WriteByte('=')
		b.WriteString(url.QueryEscape(d[k]))
	}
	return b.String()
}`

// helperURLParseResultType + helperURLUrlparse mirror CPython's
// ParseResult shape for the fields most fixtures care about. The
// `params` slot is always empty since Go's url.URL doesn't expose
// the RFC 3986 path-parameter component separately.
const helperURLParseResultType = `type __URLParseResult struct {
	Scheme   string
	Netloc   string
	Path     string
	Params   string
	Query    string
	Fragment string
}`

const helperURLUrlparse = `func __gopy_url_urlparse(s string) *__URLParseResult {
	u, err := url.Parse(s)
	if err != nil {
		return &__URLParseResult{}
	}
	return &__URLParseResult{
		Scheme:   u.Scheme,
		Netloc:   u.Host,
		Path:     u.Path,
		Query:    u.RawQuery,
		Fragment: u.Fragment,
	}
}`

// helperURLUnquote uses url.PathUnescape because Python's unquote
// preserves `+` (only QueryUnescape converts `+` to space).
const helperURLUnquote = `func __gopy_url_unquote(s string) string {
	v, err := url.PathUnescape(s)
	if err != nil {
		panic(err)
	}
	return v
}`

// helperLog* mimic the logging module's level-prefixed stderr output.
// CPython's default formatter is `LEVEL:root:msg`; our shim uses the
// same shape so fixtures comparing stderr can round-trip. basicConfig
// is a no-op because we don't honor log levels yet — every call writes.
const helperLogDebug = `func __gopy_log_debug(msg string) { fmt.Fprintln(os.Stderr, "DEBUG:root:"+msg) }`
const helperLogInfo = `func __gopy_log_info(msg string) { fmt.Fprintln(os.Stderr, "INFO:root:"+msg) }`
const helperLogWarning = `func __gopy_log_warning(msg string) { fmt.Fprintln(os.Stderr, "WARNING:root:"+msg) }`
const helperLogError = `func __gopy_log_error(msg string) { fmt.Fprintln(os.Stderr, "ERROR:root:"+msg) }`
const helperLogCritical = `func __gopy_log_critical(msg string) { fmt.Fprintln(os.Stderr, "CRITICAL:root:"+msg) }`
const helperLogBasicConfig = `func __gopy_log_basicConfig() {}`

// helperCompletedProcessType + helperSubprocessRun bridge Python's
// subprocess.run to Go's os/exec. We always capture stdout / stderr;
// kwargs like capture_output / text are accepted at the call site and
// silently ignored because Go's exec semantics are equivalent.
const helperCompletedProcessType = `type __CompletedProcess struct {
	Returncode int64
	Stdout     string
	Stderr     string
}`

const helperSubprocessRun = `func __gopy_subprocess_run(args []string) *__CompletedProcess {
	if len(args) == 0 {
		return &__CompletedProcess{Returncode: -1}
	}
	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.Output()
	r := &__CompletedProcess{Stdout: string(out)}
	if ee, ok := err.(*exec.ExitError); ok {
		r.Returncode = int64(ee.ExitCode())
		r.Stderr = string(ee.Stderr)
	} else if err != nil {
		r.Returncode = -1
	}
	return r
}`

// helperMathFloor / helperMathCeil match Python 3's math.floor / math.ceil
// signature — they return an int (int64), not a float, even though the
// underlying Go math package operates on float64.
const helperMathFloor = `func __gopy_math_floor(x float64) int64 { return int64(math.Floor(x)) }`
const helperMathCeil = `func __gopy_math_ceil(x float64) int64 { return int64(math.Ceil(x)) }`

// helperRandomFloat / helperRandint / helperRandomSeed bridge Python's
// random module to Go's math/rand. We use the package-level rand source
// so callers can seed deterministically.
const helperRandomFloat = `func __gopy_random() float64 { return rand.Float64() }`

const helperRandint = `func __gopy_randint(a, b int64) int64 {
	// Python's random.randint is inclusive on both ends.
	return a + rand.Int63n(b-a+1)
}`

const helperRandomSeed = `func __gopy_random_seed(s int64) { rand.Seed(s) }`

// helperPathType is the runtime Path struct used by pathlib.Path.
// Mirrors a handful of Path methods sufficient for "open this, check
// existence, read/write text" workflows.
const helperPathType = `type __Path struct{ p string }

func (p *__Path) Join(other string) *__Path {
	if p.p == "" {
		return &__Path{p: other}
	}
	if len(p.p) > 0 && p.p[len(p.p)-1] == '/' {
		return &__Path{p: p.p + other}
	}
	return &__Path{p: p.p + "/" + other}
}

func (p *__Path) Name() string {
	for i := len(p.p) - 1; i >= 0; i-- {
		if p.p[i] == '/' {
			return p.p[i+1:]
		}
	}
	return p.p
}

func (p *__Path) Parent() *__Path {
	for i := len(p.p) - 1; i >= 0; i-- {
		if p.p[i] == '/' {
			if i == 0 {
				return &__Path{p: "/"}
			}
			return &__Path{p: p.p[:i]}
		}
	}
	return &__Path{p: "."}
}

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
