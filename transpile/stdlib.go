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
		Attrs: map[string]stdlibAttr{
			"sep":    {GoExpr: `string(os.PathSeparator)`, GoImport: "os"},
			"linesep": {GoExpr: `"\n"`},
		},
		Funcs: map[string]stdlibFunc{
			"getenv":   {GoFunc: "os.Getenv", GoImport: "os"},
			"getcwd":   {GoFunc: "__gopy_os_getcwd", GoImport: "os", Helper: helperOsGetcwd, RetKind: "str"},
			"listdir":  {GoFunc: "__gopy_os_listdir", GoImport: "os", Helper: helperOsListdir},
			"makedirs": {GoFunc: "__gopy_os_makedirs", GoImport: "os", Helper: helperOsMakedirs},
			"remove":   {GoFunc: "__gopy_os_remove", GoImport: "os", Helper: helperOsRemove},
			"rename":   {GoFunc: "__gopy_os_rename", GoImport: "os", Helper: helperOsRename},
			"mkdir":    {GoFunc: "__gopy_os_mkdir", GoImport: "os", Helper: helperOsMkdir},
			"rmdir":    {GoFunc: "__gopy_os_rmdir", GoImport: "os", Helper: helperOsRmdir},
		},
		Subs: map[string]stdlibModule{
			"path": {
				Funcs: map[string]stdlibFunc{
					"join":     {GoFunc: "__gopy_path_join", GoImport: "path/filepath", Helper: helperPathJoin, RetKind: "str"},
					"exists":   {GoFunc: "__gopy_path_exists", GoImport: "os", Helper: helperPathExists, RetKind: "bool"},
					"isfile":   {GoFunc: "__gopy_path_isfile", GoImport: "os", Helper: helperPathIsfile, RetKind: "bool"},
					"isdir":    {GoFunc: "__gopy_path_isdir", GoImport: "os", Helper: helperPathIsdir, RetKind: "bool"},
					"basename": {GoFunc: "filepath.Base", GoImport: "path/filepath", RetKind: "str"},
					"dirname":  {GoFunc: "filepath.Dir", GoImport: "path/filepath", RetKind: "str"},
					"splitext": {GoFunc: "__gopy_path_splitext", GoImport: "path/filepath", Helper: helperPathSplitext},
					"abspath":  {GoFunc: "__gopy_path_abspath", GoImport: "path/filepath", Helper: helperPathAbspath, RetKind: "str"},
					"split":    {GoFunc: "__gopy_path_split", GoImport: "path/filepath", Helper: helperPathSplit},
					"relpath":  {GoFunc: "__gopy_path_relpath", GoImport: "path/filepath", Helper: helperPathRelpath, RetKind: "str"},
					"getsize":  {GoFunc: "__gopy_path_getsize", GoImport: "os", Helper: helperPathGetsize, RetKind: "int"},
				},
			},
		},
	},
	"time": {
		Funcs: map[string]stdlibFunc{
			"time":         {GoFunc: "__gopy_time_now_seconds", GoImport: "time", Helper: helperTimeNowSeconds, RetKind: "float"},
			"sleep":        {GoFunc: "__gopy_time_sleep", GoImport: "time", Helper: helperTimeSleep},
			"monotonic":    {GoFunc: "__gopy_time_monotonic", GoImport: "time", Helper: helperTimeMonotonic, RetKind: "float"},
			"perf_counter": {GoFunc: "__gopy_time_monotonic", GoImport: "time", Helper: helperTimeMonotonic, RetKind: "float"},
			"time_ns":      {GoFunc: "__gopy_time_ns", GoImport: "time", Helper: helperTimeNs, RetKind: "int"},
		},
	},
	"json": {
		Funcs: map[string]stdlibFunc{
			"dumps": {GoFunc: "__gopy_json_dumps", GoImport: "encoding/json", Helper: helperJSONDumps, HelperImports: []string{"strings"}},
			"loads": {GoFunc: "__gopy_json_loads", GoImport: "encoding/json", Helper: helperJSONLoads},
			"load":  {GoFunc: "__gopy_json_load", GoImport: "encoding/json", Helper: helperJSONLoad, HelperImports: []string{"io"}},
			"dump":  {GoFunc: "__gopy_json_dump", GoImport: "encoding/json", Helper: helperJSONDump, HelperImports: []string{"strings"}},
		},
	},
	"math": {
		Attrs: map[string]stdlibAttr{
			"pi":  {GoExpr: "math.Pi", GoImport: "math"},
			"e":   {GoExpr: "math.E", GoImport: "math"},
			"inf": {GoExpr: "math.Inf(1)", GoImport: "math"},
			"nan": {GoExpr: "math.NaN()", GoImport: "math"},
			"tau": {GoExpr: "math.Pi * 2", GoImport: "math"},
		},
		Funcs: map[string]stdlibFunc{
			"sqrt":     {GoFunc: "math.Sqrt", GoImport: "math"},
			"floor":    {GoFunc: "__gopy_math_floor", GoImport: "math", Helper: helperMathFloor, RetKind: "int"},
			"ceil":     {GoFunc: "__gopy_math_ceil", GoImport: "math", Helper: helperMathCeil, RetKind: "int"},
			"log":      {GoFunc: "math.Log", GoImport: "math"},
			"log2":     {GoFunc: "math.Log2", GoImport: "math"},
			"log10":    {GoFunc: "math.Log10", GoImport: "math"},
			"exp":      {GoFunc: "math.Exp", GoImport: "math"},
			"sin":      {GoFunc: "math.Sin", GoImport: "math"},
			"cos":      {GoFunc: "math.Cos", GoImport: "math"},
			"tan":      {GoFunc: "math.Tan", GoImport: "math"},
			"atan":     {GoFunc: "math.Atan", GoImport: "math"},
			"atan2":    {GoFunc: "math.Atan2", GoImport: "math"},
			"pow":      {GoFunc: "math.Pow", GoImport: "math"},
			"trunc":    {GoFunc: "__gopy_math_trunc", GoImport: "math", Helper: helperMathTrunc, RetKind: "int"},
			"fmod":     {GoFunc: "math.Mod", GoImport: "math", RetKind: "float"},
			"gcd":      {GoFunc: "__gopy_math_gcd", Helper: helperMathGcd, RetKind: "int"},
			"isnan":    {GoFunc: "math.IsNaN", GoImport: "math", RetKind: "bool"},
			"isinf":    {GoFunc: "__gopy_math_isinf", GoImport: "math", Helper: helperMathIsInf, RetKind: "bool"},
			"isfinite": {GoFunc: "__gopy_math_isfinite", GoImport: "math", Helper: helperMathIsFinite, RetKind: "bool"},
			"copysign": {GoFunc: "math.Copysign", GoImport: "math"},
			"hypot":    {GoFunc: "math.Hypot", GoImport: "math"},
			"degrees":   {GoFunc: "__gopy_math_degrees", GoImport: "math", Helper: helperMathDegrees},
			"radians":   {GoFunc: "__gopy_math_radians", GoImport: "math", Helper: helperMathRadians},
			"factorial": {GoFunc: "__gopy_math_factorial", Helper: helperMathFactorial, RetKind: "int"},
			"comb":      {GoFunc: "__gopy_math_comb", Helper: helperMathComb, RetKind: "int"},
			"perm":      {GoFunc: "__gopy_math_perm", Helper: helperMathPerm, RetKind: "int"},
			"dist":      {GoFunc: "__gopy_math_dist", GoImport: "math", Helper: helperMathDist, RetKind: "float"},
			"prod":      {GoFunc: "__gopy_math_prod", Helper: helperMathProd, RetKind: "int"},
			"remainder": {GoFunc: "math.Remainder", GoImport: "math", RetKind: "float"},
		},
	},
	"hashlib": {
		Funcs: map[string]stdlibFunc{
			"sha256": {GoFunc: "__gopy_hashlib_sha256", GoImport: "crypto/sha256", Helper: helperHashlibSha256, RetTag: "__Hasher", ExtraHelpers: map[string]string{"__Hasher": helperHasherType}, HelperImports: []string{"encoding/hex", "crypto/md5", "crypto/sha1", "crypto/sha512"}},
			"md5":    {GoFunc: "__gopy_hashlib_md5", GoImport: "crypto/md5", Helper: helperHashlibMd5, RetTag: "__Hasher", ExtraHelpers: map[string]string{"__Hasher": helperHasherType}, HelperImports: []string{"encoding/hex", "crypto/sha256", "crypto/sha1", "crypto/sha512"}},
			"sha1":   {GoFunc: "__gopy_hashlib_sha1", GoImport: "crypto/sha1", Helper: helperHashlibSha1, RetTag: "__Hasher", ExtraHelpers: map[string]string{"__Hasher": helperHasherType}, HelperImports: []string{"encoding/hex", "crypto/md5", "crypto/sha256", "crypto/sha512"}},
			"sha512": {GoFunc: "__gopy_hashlib_sha512", GoImport: "crypto/sha512", Helper: helperHashlibSha512, RetTag: "__Hasher", ExtraHelpers: map[string]string{"__Hasher": helperHasherType}, HelperImports: []string{"encoding/hex", "crypto/md5", "crypto/sha256", "crypto/sha1"}},
		},
	},
	"secrets": {
		Funcs: map[string]stdlibFunc{
			"token_hex":     {GoFunc: "__gopy_secrets_token_hex", GoImport: "crypto/rand", Helper: helperSecretsTokenHex, HelperImports: []string{"encoding/hex"}, RetKind: "str"},
			"token_urlsafe": {GoFunc: "__gopy_secrets_token_urlsafe", GoImport: "crypto/rand", Helper: helperSecretsTokenUrl, HelperImports: []string{"encoding/base64"}, RetKind: "str"},
			"token_bytes":   {GoFunc: "__gopy_secrets_token_bytes", GoImport: "crypto/rand", Helper: helperSecretsTokenBytes, RetKind: "str"},
		},
	},
	"base64": {
		Funcs: map[string]stdlibFunc{
			"b64encode":         {GoFunc: "__gopy_b64encode", GoImport: "encoding/base64", Helper: helperB64Encode, RetKind: "str"},
			"b64decode":         {GoFunc: "__gopy_b64decode", GoImport: "encoding/base64", Helper: helperB64Decode, RetKind: "str"},
			"urlsafe_b64encode": {GoFunc: "__gopy_b64urlencode", GoImport: "encoding/base64", Helper: helperB64URLEncode, RetKind: "str"},
			"urlsafe_b64decode": {GoFunc: "__gopy_b64urldecode", GoImport: "encoding/base64", Helper: helperB64URLDecode, RetKind: "str"},
			"b32encode":         {GoFunc: "__gopy_b32encode", GoImport: "encoding/base32", Helper: helperB32Encode, RetKind: "str"},
			"b32decode":         {GoFunc: "__gopy_b32decode", GoImport: "encoding/base32", Helper: helperB32Decode, RetKind: "str"},
			"b16encode":         {GoFunc: "__gopy_b16encode", GoImport: "encoding/hex", Helper: helperB16Encode, HelperImports: []string{"strings"}, RetKind: "str"},
			"b16decode":         {GoFunc: "__gopy_b16decode", GoImport: "encoding/hex", Helper: helperB16Decode, RetKind: "str"},
		},
	},
	"urllib": {
		Subs: map[string]stdlibModule{
			"parse": {
				Funcs: map[string]stdlibFunc{
					"quote":        {GoFunc: "__gopy_url_quote", GoImport: "net/url", Helper: helperURLQuote, HelperImports: []string{"strings", "fmt"}, RetKind: "str"},
					"quote_plus":   {GoFunc: "__gopy_url_quote_plus", GoImport: "net/url", Helper: helperURLQuotePlus, HelperImports: []string{"strings", "fmt"}, RetKind: "str"},
					"unquote":      {GoFunc: "__gopy_url_unquote", GoImport: "net/url", Helper: helperURLUnquote, RetKind: "str"},
					"unquote_plus": {GoFunc: "__gopy_url_unquote_plus", GoImport: "net/url", Helper: helperURLUnquotePlus, HelperImports: []string{"strings"}, RetKind: "str"},
					"urlencode":    {GoFunc: "__gopy_url_urlencode", GoImport: "net/url", Helper: helperURLUrlencode, HelperImports: []string{"strings"}, RetKind: "str"},
					"urlparse":     {GoFunc: "__gopy_url_urlparse", GoImport: "net/url", Helper: helperURLUrlparse, RetTag: "__URLParseResult", ExtraHelpers: map[string]string{"__URLParseResult": helperURLParseResultType}},
					"parse_qs":     {GoFunc: "__gopy_url_parse_qs", GoImport: "net/url", Helper: helperURLParseQs},
					"parse_qsl":    {GoFunc: "__gopy_url_parse_qsl", GoImport: "net/url", Helper: helperURLParseQsl},
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
		Funcs: map[string]stdlibFunc{
			"Template": {GoFunc: "__gopy_string_template_new", Helper: helperStringTemplateNew, RetTag: "__Template", ExtraHelpers: map[string]string{"__Template": helperStringTemplateType}, HelperImports: []string{"strings", "fmt"}},
			"capwords": {GoFunc: "__gopy_string_capwords", Helper: helperStringCapwords, HelperImports: []string{"strings"}, RetKind: "str"},
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
			"reduce":          {GoFunc: "__gopy_reduce_unused"},
			"partial":         {GoFunc: "__gopy_partial_unused"},
			"cache":           {GoFunc: "__gopy_cache_unused"},
			"cached_property": {GoFunc: "__gopy_cached_prop_unused"},
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
	"heapq": {
		Funcs: map[string]stdlibFunc{
			// Dispatched per-element-type in transpile.go's call() builders.
			"heappush":    {GoFunc: "__gopy_heappush_unused"},
			"heappop":     {GoFunc: "__gopy_heappop_unused"},
			"heapify":     {GoFunc: "__gopy_heapify_unused"},
			"heappushpop": {GoFunc: "__gopy_heappushpop_unused"},
			"nsmallest":   {GoFunc: "__gopy_nsmallest_unused"},
			"nlargest":    {GoFunc: "__gopy_nlargest_unused"},
		},
	},
	"bisect": {
		Funcs: map[string]stdlibFunc{
			"bisect_left":  {GoFunc: "__gopy_bisect_left_unused"},
			"bisect_right": {GoFunc: "__gopy_bisect_right_unused"},
			"insort":       {GoFunc: "__gopy_insort_unused"},
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
			"permutations": {GoFunc: "__gopy_permutations_unused"},
			"islice":       {GoFunc: "__gopy_islice_unused"},
			"repeat":       {GoFunc: "__gopy_repeat_unused"},
			"starmap":      {GoFunc: "__gopy_starmap_unused"},
			"filterfalse":  {GoFunc: "__gopy_filterfalse_unused"},
			"compress":     {GoFunc: "__gopy_compress_unused"},
			"count":        {GoFunc: "__gopy_count_unused"},
		},
	},
	"random": {
		Funcs: map[string]stdlibFunc{
			"random":  {GoFunc: "__gopy_random", GoImport: "math/rand", Helper: helperRandomFloat},
			"randint": {GoFunc: "__gopy_randint", GoImport: "math/rand", Helper: helperRandint},
			"seed":    {GoFunc: "__gopy_random_seed", GoImport: "math/rand", Helper: helperRandomSeed},
			"uniform": {GoFunc: "__gopy_random_uniform", GoImport: "math/rand", Helper: helperRandomUniform, RetKind: "float"},
			// choice / shuffle / sample dispatch per-element type from
			// transpile.go's call() builders below.
			"choice":  {GoFunc: "__gopy_random_choice_unused"},
			"shuffle": {GoFunc: "__gopy_random_shuffle_unused"},
			"sample":  {GoFunc: "__gopy_random_sample_unused"},
		},
	},
	"statistics": {
		Funcs: map[string]stdlibFunc{
			"mean":          {GoFunc: "__gopy_stats_mean", Helper: helperStatsMean, RetKind: "float"},
			"fmean":         {GoFunc: "__gopy_stats_mean", Helper: helperStatsMean, RetKind: "float"},
			"median":        {GoFunc: "__gopy_stats_median", GoImport: "sort", Helper: helperStatsMedian, RetKind: "float"},
			"mode":          {GoFunc: "__gopy_stats_mode", Helper: helperStatsMode, RetKind: "int"},
			"stdev":         {GoFunc: "__gopy_stats_stdev", GoImport: "math", Helper: helperStatsStdev, RetKind: "float"},
			"pstdev":        {GoFunc: "__gopy_stats_pstdev", GoImport: "math", Helper: helperStatsPstdev, RetKind: "float"},
			"variance":      {GoFunc: "__gopy_stats_variance", Helper: helperStatsVariance, RetKind: "float"},
			"median_low":    {GoFunc: "__gopy_stats_median_low", GoImport: "sort", Helper: helperStatsMedianLow, RetKind: "float"},
			"median_high":   {GoFunc: "__gopy_stats_median_high", GoImport: "sort", Helper: helperStatsMedianHigh, RetKind: "float"},
			"harmonic_mean": {GoFunc: "__gopy_stats_harmonic", Helper: helperStatsHarmonic, RetKind: "float"},
			"pvariance":     {GoFunc: "__gopy_stats_pvariance", Helper: helperStatsPvariance, RetKind: "float"},
		},
	},
	"uuid": {
		Funcs: map[string]stdlibFunc{
			"uuid4": {GoFunc: "__gopy_uuid4", GoImport: "crypto/rand", Helper: helperUuid4, RetKind: "str", HelperImports: []string{"fmt"}},
		},
	},
	"textwrap": {
		Funcs: map[string]stdlibFunc{
			"dedent": {GoFunc: "__gopy_textwrap_dedent", Helper: helperTextwrapDedent, RetKind: "str", HelperImports: []string{"strings"}},
			"indent": {GoFunc: "__gopy_textwrap_indent", Helper: helperTextwrapIndent, RetKind: "str", HelperImports: []string{"strings"}},
			"fill":   {GoFunc: "__gopy_textwrap_fill", Helper: helperTextwrapFill, RetKind: "str", HelperImports: []string{"strings"}},
		},
	},
	"re": {
		Funcs: map[string]stdlibFunc{
			"findall":   {GoFunc: "__gopy_re_findall", GoImport: "regexp", Helper: helperReFindall},
			"search":    {GoFunc: "__gopy_re_search", GoImport: "regexp", Helper: helperReSearch, RetTag: "__Match", ExtraHelpers: map[string]string{"__Match": helperMatchType, "__gopy_match_build": helperMatchBuild}},
			"match":     {GoFunc: "__gopy_re_match", GoImport: "regexp", Helper: helperReMatch, RetTag: "__Match", ExtraHelpers: map[string]string{"__Match": helperMatchType, "__gopy_match_build": helperMatchBuild}},
			"fullmatch": {GoFunc: "__gopy_re_fullmatch", GoImport: "regexp", Helper: helperReFullmatch, RetTag: "__Match", ExtraHelpers: map[string]string{"__Match": helperMatchType, "__gopy_match_build": helperMatchBuild}},
			"sub":       {GoFunc: "__gopy_re_sub", GoImport: "regexp", Helper: helperReSub},
			"subn":      {GoFunc: "__gopy_re_subn", GoImport: "regexp", Helper: helperReSubn},
			"split":     {GoFunc: "__gopy_re_split", GoImport: "regexp", Helper: helperReSplit},
			"escape":    {GoFunc: "regexp.QuoteMeta", GoImport: "regexp", RetKind: "str"},
			"compile":   {GoFunc: "__gopy_re_compile", GoImport: "regexp", Helper: helperReCompile, RetTag: "__Pattern", ExtraHelpers: map[string]string{"__Pattern": helperPatternType, "__Match": helperMatchType, "__gopy_match_build": helperMatchBuild}},
		},
	},
	"csv": {
		Funcs: map[string]stdlibFunc{
			"reader": {GoFunc: "__gopy_csv_reader", GoImport: "encoding/csv", Helper: helperCSVReader, HelperImports: []string{"strings"}},
		},
	},
	"pathlib": {
		Funcs: map[string]stdlibFunc{
			"Path": {GoFunc: "__gopy_path_new", GoImport: "os", Helper: helperPathNew, RetTag: "__Path", ExtraHelpers: map[string]string{"__Path": helperPathType}, HelperImports: []string{"os", "path/filepath"}},
		},
	},
	"datetime": {
		Funcs: map[string]stdlibFunc{
			"timedelta": {GoFunc: "__gopy_timedelta_new", GoImport: "time", Helper: helperTimedeltaNew, RetTag: "__Timedelta", ExtraHelpers: map[string]string{"__Timedelta": helperTimedeltaType}, HelperImports: []string{"fmt"}},
			"date":      {GoFunc: "__gopy_date_new", GoImport: "fmt", Helper: helperDateNew, RetTag: "__Date", ExtraHelpers: map[string]string{"__Date": helperDateType, "__gopy_py_time_format": helperPyTimeFormat, "__gopy_datetime_strftime": helperDatetimeStrftime}, HelperImports: []string{"time", "strings"}},
			"time":      {GoFunc: "__gopy_time_new", GoImport: "fmt", Helper: helperTimeNew, RetTag: "__Time", ExtraHelpers: map[string]string{"__Time": helperTimeType}},
		},
		Subs: map[string]stdlibModule{
			"datetime": {
				Funcs: map[string]stdlibFunc{
					// __Datetime methods reference __Timedelta (for Add/Sub),
					// so we always emit both types whenever datetime.now() is
					// used; otherwise Go would error on the undefined type.
					"now":            {GoFunc: "__gopy_datetime_now", GoImport: "time", Helper: helperDatetimeNow, RetTag: "__Datetime", ExtraHelpers: map[string]string{"__Datetime": helperDatetimeType, "__Timedelta": helperTimedeltaType, "__gopy_py_time_format": helperPyTimeFormat, "__gopy_datetime_strftime": helperDatetimeStrftime}, HelperImports: []string{"fmt", "strings"}},
					"strptime":       {GoFunc: "__gopy_datetime_strptime", GoImport: "time", Helper: helperDatetimeStrptime, RetTag: "__Datetime", ExtraHelpers: map[string]string{"__Datetime": helperDatetimeType, "__Timedelta": helperTimedeltaType, "__gopy_py_time_format": helperPyTimeFormat}, HelperImports: []string{"fmt", "strings"}},
					"fromtimestamp":  {GoFunc: "__gopy_datetime_fromts", GoImport: "time", Helper: helperDatetimeFromTs, RetTag: "__Datetime", ExtraHelpers: map[string]string{"__Datetime": helperDatetimeType, "__Timedelta": helperTimedeltaType, "__gopy_py_time_format": helperPyTimeFormat, "__gopy_datetime_strftime": helperDatetimeStrftime}, HelperImports: []string{"fmt", "strings"}},
					"fromisoformat":  {GoFunc: "__gopy_datetime_fromiso", GoImport: "time", Helper: helperDatetimeFromIso, RetTag: "__Datetime", ExtraHelpers: map[string]string{"__Datetime": helperDatetimeType, "__Timedelta": helperTimedeltaType, "__gopy_py_time_format": helperPyTimeFormat, "__gopy_datetime_strftime": helperDatetimeStrftime}, HelperImports: []string{"fmt", "strings"}},
					"utcnow":         {GoFunc: "__gopy_datetime_utcnow", GoImport: "time", Helper: helperDatetimeUtcnow, RetTag: "__Datetime", ExtraHelpers: map[string]string{"__Datetime": helperDatetimeType, "__Timedelta": helperTimedeltaType, "__gopy_py_time_format": helperPyTimeFormat, "__gopy_datetime_strftime": helperDatetimeStrftime}, HelperImports: []string{"fmt", "strings"}},
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

// helperTimeMonotonic / helperTimeNs mirror Python's monotonic clocks.
// Go's time.Now is monotonic by default; we expose the nanosecond reading
// converted to seconds (float) or kept as int64 ns.
const helperTimeMonotonic = `func __gopy_time_monotonic() float64 { return float64(time.Now().UnixNano()) / 1e9 }`

const helperTimeNs = `func __gopy_time_ns() int64 { return time.Now().UnixNano() }`

// helperJSONDumps mirrors CPython's json.dumps default separators of
// `, ` and `: `. Go's encoding/json emits compact JSON, so we reformat
// the result outside of any string literal. The variadic indent param
// matches Python's optional indent=N kwarg (>= 0 enables pretty-print).
const helperJSONDumps = `func __gopy_json_dumps(v any, indent ...int64) string {
	if len(indent) > 0 && indent[0] >= 0 {
		b, err := json.MarshalIndent(v, "", strings.Repeat(" ", int(indent[0])))
		if err != nil {
			panic(err)
		}
		return string(b)
	}
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

// helperJSONLoad reads JSON from an io.Reader (typically *os.File from a
// with-open block). Mirrors json.load(fh).
const helperJSONLoad = `func __gopy_json_load(r io.Reader) any {
	b, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		panic(err)
	}
	return v
}`

// helperJSONDump serializes v and writes the (Python-style separator)
// output to the writer. Variadic indent param matches dumps's signature.
const helperJSONDump = `func __gopy_json_dump(v any, w interface{ Write([]byte) (int, error) }, indent ...int64) {
	out := __gopy_json_dumps(v, indent...)
	if _, err := w.Write([]byte(out)); err != nil {
		panic(err)
	}
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
	idx    []int
}

func (m *__Match) Start(args ...any) int64 {
	n := 0
	if len(args) > 0 {
		switch a := args[0].(type) {
		case int:
			n = a
		case int64:
			n = int(a)
		case string:
			for i, nm := range m.names {
				if nm == a {
					n = i
					break
				}
			}
		}
	}
	if n*2+1 >= len(m.idx) {
		return -1
	}
	return int64(m.idx[n*2])
}

func (m *__Match) End(args ...any) int64 {
	n := 0
	if len(args) > 0 {
		switch a := args[0].(type) {
		case int:
			n = a
		case int64:
			n = int(a)
		case string:
			for i, nm := range m.names {
				if nm == a {
					n = i
					break
				}
			}
		}
	}
	if n*2+1 >= len(m.idx) {
		return -1
	}
	return int64(m.idx[n*2+1])
}

func (m *__Match) Span(args ...any) []int64 {
	return []int64{m.Start(args...), m.End(args...)}
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
	return __gopy_match_build(r, s, false)
}`

// helperReMatch anchors the pattern to the start of the string, matching
// Python's re.match semantics. Returns nil on miss.
const helperReMatch = `func __gopy_re_match(pattern, s string) *__Match {
	r := regexp.MustCompile("^(?:" + pattern + ")")
	return __gopy_match_build(r, s, false)
}`

// helperMatchBuild centralizes the FindStringSubmatchIndex → __Match
// conversion so search / match / compile share the same position data.
const helperMatchBuild = `func __gopy_match_build(r *regexp.Regexp, s string, _ bool) *__Match {
	idx := r.FindStringSubmatchIndex(s)
	if idx == nil {
		return nil
	}
	full := s[idx[0]:idx[1]]
	groups := make([]string, 0, len(idx)/2-1)
	for i := 1; i < len(idx)/2; i++ {
		if idx[i*2] < 0 {
			groups = append(groups, "")
		} else {
			groups = append(groups, s[idx[i*2]:idx[i*2+1]])
		}
	}
	return &__Match{full: full, groups: groups, names: r.SubexpNames(), idx: idx}
}`

// helperReSub replaces every match of pattern with repl.
const helperReSub = `func __gopy_re_sub(pattern, repl, s string) string {
	r := regexp.MustCompile(pattern)
	return r.ReplaceAllString(s, repl)
}`

// helperReSubn returns []any{result_string, n_substitutions} so callers
// can unpack via positional indexing. Mirrors Python's re.subn tuple.
const helperReSubn = `func __gopy_re_subn(pattern, repl, s string) []any {
	r := regexp.MustCompile(pattern)
	count := int64(len(r.FindAllStringIndex(s, -1)))
	return []any{r.ReplaceAllString(s, repl), count}
}`

// helperReFullmatch anchors at both ends, mirroring re.fullmatch.
const helperReFullmatch = `func __gopy_re_fullmatch(pattern, s string) *__Match {
	r := regexp.MustCompile("^(?:" + pattern + ")$")
	return __gopy_match_build(r, s, false)
}`

// helperReSplit splits s on every occurrence of the pattern. Mirrors
// re.split's default form; the maxsplit argument is not supported (use
// strings.SplitN with a literal sep for that pattern).
const helperReSplit = `func __gopy_re_split(pattern, s string) []string {
	r := regexp.MustCompile(pattern)
	out := r.Split(s, -1)
	if out == nil {
		return []string{}
	}
	return out
}`

// helperPatternType wraps a compiled regexp so re.compile(p).match(s)
// and friends share one re-usable underlying *regexp.Regexp. Method
// names match the (already-renamed) Match/Search/Findall/Sub forms.
const helperPatternType = `type __Pattern struct {
	r       *regexp.Regexp
	anchor  *regexp.Regexp
}

func (p *__Pattern) Match(s string) *__Match {
	return __gopy_match_build(p.anchor, s, false)
}

func (p *__Pattern) Search(s string) *__Match {
	return __gopy_match_build(p.r, s, false)
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

// helperHasherType bridges hashlib's hash objects. The algo string drives
// Hexdigest's dispatch so fixtures can compare hex strings across CPython
// and Go for any of the SHA / MD5 variants.
const helperHasherType = `type __Hasher struct {
	data []byte
	algo string
}

func (h *__Hasher) Hexdigest() string {
	switch h.algo {
	case "sha256":
		sum := sha256.Sum256(h.data)
		return hex.EncodeToString(sum[:])
	case "sha512":
		sum := sha512.Sum512(h.data)
		return hex.EncodeToString(sum[:])
	case "sha1":
		sum := sha1.Sum(h.data)
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

const helperHashlibSha512 = `func __gopy_hashlib_sha512(data string) *__Hasher {
	return &__Hasher{data: []byte(data), algo: "sha512"}
}`

const helperHashlibSha1 = `func __gopy_hashlib_sha1(data string) *__Hasher {
	return &__Hasher{data: []byte(data), algo: "sha1"}
}`

const helperHashlibMd5 = `func __gopy_hashlib_md5(data string) *__Hasher {
	return &__Hasher{data: []byte(data), algo: "md5"}
}`

const helperSecretsTokenHex = `func __gopy_secrets_token_hex(args ...int64) string {
	n := int64(32)
	if len(args) > 0 {
		n = args[0]
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}`

const helperSecretsTokenUrl = `func __gopy_secrets_token_urlsafe(args ...int64) string {
	n := int64(32)
	if len(args) > 0 {
		n = args[0]
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}`

const helperSecretsTokenBytes = `func __gopy_secrets_token_bytes(args ...int64) string {
	n := int64(32)
	if len(args) > 0 {
		n = args[0]
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return string(b)
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

const helperB64URLEncode = `func __gopy_b64urlencode(s string) string {
	return base64.URLEncoding.EncodeToString([]byte(s))
}`

const helperB64URLDecode = `func __gopy_b64urldecode(s string) string {
	out, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return string(out)
}`

const helperB32Encode = `func __gopy_b32encode(s string) string {
	return base32.StdEncoding.EncodeToString([]byte(s))
}`

const helperB32Decode = `func __gopy_b32decode(s string) string {
	out, err := base32.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return string(out)
}`

// helperB16Encode mirrors CPython's base64.b16encode (uppercase hex).
const helperB16Encode = `func __gopy_b16encode(s string) string {
	return strings.ToUpper(hex.EncodeToString([]byte(s)))
}`

const helperB16Decode = `func __gopy_b16decode(s string) string {
	out, err := hex.DecodeString(s)
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

// helperURLQuotePlus / UnquotePlus mirror Python's quote_plus / unquote_plus
// — same as quote / unquote except space ↔ `+`.
const helperURLQuotePlus = `func __gopy_url_quote_plus(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' {
			b.WriteByte('+')
			continue
		}
		safe := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '_' || c == '.' || c == '-' || c == '~'
		if safe {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}`

const helperURLUnquotePlus = `func __gopy_url_unquote_plus(s string) string {
	v, err := url.PathUnescape(strings.ReplaceAll(s, "+", " "))
	if err != nil {
		panic(err)
	}
	return v
}`

// helperURLParseQs returns map[string][]string from a query-string,
// mirroring urllib.parse.parse_qs. parse_qsl returns the same data as
// []any pairs.
const helperURLParseQs = `func __gopy_url_parse_qs(s string) map[string][]string {
	v, err := url.ParseQuery(s)
	if err != nil {
		panic(err)
	}
	out := map[string][]string{}
	for k, vs := range v {
		out[k] = vs
	}
	return out
}`

const helperURLParseQsl = `func __gopy_url_parse_qsl(s string) []any {
	v, err := url.ParseQuery(s)
	if err != nil {
		panic(err)
	}
	out := []any{}
	for k, vs := range v {
		for _, vv := range vs {
			out = append(out, []any{k, vv})
		}
	}
	return out
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

// helperMathTrunc returns the integer part (toward zero) as int64, matching
// Python's math.trunc.
const helperMathTrunc = `func __gopy_math_trunc(x float64) int64 { return int64(math.Trunc(x)) }`

// helperMathIsInf in CPython is a single predicate (sign-agnostic).
// Go's math.IsInf takes a sign; pass 0 to accept ±∞.
const helperMathIsInf = `func __gopy_math_isinf(x float64) bool { return math.IsInf(x, 0) }`

// helperMathIsFinite mirrors Python's math.isfinite: not NaN and not ±∞.
const helperMathIsFinite = `func __gopy_math_isfinite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }`

// helperMathGcd mirrors Python's math.gcd for two int64 args (Python 3.9+
// accepts a variadic form; gopy keeps the 2-arg shape for simplicity).
const helperMathGcd = `func __gopy_math_gcd(a, b int64) int64 {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	return a
}`

const helperMathDegrees = `func __gopy_math_degrees(r float64) float64 { return r * 180 / math.Pi }`
const helperMathRadians = `func __gopy_math_radians(d float64) float64 { return d * math.Pi / 180 }`

const helperMathFactorial = `func __gopy_math_factorial(n int64) int64 {
	if n < 0 {
		panic(NewException("ValueError: factorial() not defined for negative values"))
	}
	out := int64(1)
	for i := int64(2); i <= n; i++ {
		out *= i
	}
	return out
}`

const helperMathComb = `func __gopy_math_comb(n, k int64) int64 {
	if k < 0 || n < 0 {
		panic(NewException("ValueError: comb() requires non-negative inputs"))
	}
	if k > n {
		return 0
	}
	if k > n-k {
		k = n - k
	}
	out := int64(1)
	for i := int64(0); i < k; i++ {
		out = out * (n - i) / (i + 1)
	}
	return out
}`

// helperMathDist computes the Euclidean distance between two same-length
// numeric coordinate slices (Python 3.8+). Panics on length mismatch.
const helperMathDist = `func __gopy_math_dist(a, b []float64) float64 {
	if len(a) != len(b) {
		panic(NewException("ValueError: dist() coordinates differ in length"))
	}
	var s float64
	for i := range a {
		d := a[i] - b[i]
		s += d * d
	}
	return math.Sqrt(s)
}`

// helperMathProd multiplies every int64 in the slice; returns 1 on empty
// input (matches CPython's math.prod default start).
const helperMathProd = `func __gopy_math_prod(xs []int64) int64 {
	out := int64(1)
	for _, v := range xs {
		out *= v
	}
	return out
}`

const helperMathPerm = `func __gopy_math_perm(args ...int64) int64 {
	if len(args) == 0 {
		panic(NewException("TypeError: perm() requires at least one argument"))
	}
	n := args[0]
	k := n
	if len(args) > 1 {
		k = args[1]
	}
	if k < 0 || n < 0 {
		panic(NewException("ValueError: perm() requires non-negative inputs"))
	}
	if k > n {
		return 0
	}
	out := int64(1)
	for i := int64(0); i < k; i++ {
		out *= n - i
	}
	return out
}`

// helperRandomFloat / helperRandint / helperRandomSeed bridge Python's
// random module to Go's math/rand. We use the package-level rand source
// so callers can seed deterministically.
const helperRandomFloat = `func __gopy_random() float64 { return rand.Float64() }`

const helperRandint = `func __gopy_randint(a, b int64) int64 {
	// Python's random.randint is inclusive on both ends.
	return a + rand.Int63n(b-a+1)
}`

const helperRandomSeed = `func __gopy_random_seed(s int64) { rand.Seed(s) }`

const helperRandomUniform = `func __gopy_random_uniform(a, b float64) float64 {
	return a + rand.Float64()*(b-a)
}`

// helperStatsMean mirrors statistics.mean / statistics.fmean: arithmetic
// mean of a non-empty slice, returned as float64.
const helperStatsMean = `func __gopy_stats_mean(xs []float64) float64 {
	if len(xs) == 0 {
		panic(NewException("StatisticsError: mean requires at least one data point"))
	}
	var sum float64
	for _, v := range xs {
		sum += v
	}
	return sum / float64(len(xs))
}`

const helperStatsMedian = `func __gopy_stats_median(xs []float64) float64 {
	if len(xs) == 0 {
		panic(NewException("StatisticsError: median requires at least one data point"))
	}
	cp := make([]float64, len(xs))
	copy(cp, xs)
	sort.Float64s(cp)
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2]
	}
	return (cp[n/2-1] + cp[n/2]) / 2
}`

// helperStatsMode returns the first-encountered most-frequent value.
// Python's statistics.mode raises on multi-modal datasets in older 3.x;
// 3.8+ returns the first mode. Match the 3.8+ behavior.
const helperStatsMode = `func __gopy_stats_mode(xs []int64) int64 {
	if len(xs) == 0 {
		panic(NewException("StatisticsError: mode requires at least one data point"))
	}
	counts := map[int64]int{}
	order := []int64{}
	for _, v := range xs {
		if _, ok := counts[v]; !ok {
			order = append(order, v)
		}
		counts[v]++
	}
	best := order[0]
	bestN := counts[best]
	for _, v := range order[1:] {
		if counts[v] > bestN {
			best = v
			bestN = counts[v]
		}
	}
	return best
}`

const helperStatsVariance = `func __gopy_stats_variance(xs []float64) float64 {
	if len(xs) < 2 {
		panic(NewException("StatisticsError: variance requires at least two data points"))
	}
	var sum float64
	for _, v := range xs {
		sum += v
	}
	mean := sum / float64(len(xs))
	var ss float64
	for _, v := range xs {
		d := v - mean
		ss += d * d
	}
	return ss / float64(len(xs)-1)
}`

const helperStatsStdev = `func __gopy_stats_stdev(xs []float64) float64 {
	return math.Sqrt(__gopy_stats_variance(xs))
}`

const helperStatsMedianLow = `func __gopy_stats_median_low(xs []float64) float64 {
	if len(xs) == 0 {
		panic(NewException("StatisticsError: median_low requires at least one data point"))
	}
	cp := make([]float64, len(xs))
	copy(cp, xs)
	sort.Float64s(cp)
	n := len(cp)
	if n%2 == 1 {
		return cp[n/2]
	}
	return cp[n/2-1]
}`

const helperStatsMedianHigh = `func __gopy_stats_median_high(xs []float64) float64 {
	if len(xs) == 0 {
		panic(NewException("StatisticsError: median_high requires at least one data point"))
	}
	cp := make([]float64, len(xs))
	copy(cp, xs)
	sort.Float64s(cp)
	return cp[len(cp)/2]
}`

const helperStatsHarmonic = `func __gopy_stats_harmonic(xs []float64) float64 {
	if len(xs) == 0 {
		panic(NewException("StatisticsError: harmonic_mean requires at least one data point"))
	}
	var s float64
	for _, v := range xs {
		if v <= 0 {
			panic(NewException("StatisticsError: harmonic_mean requires positive values"))
		}
		s += 1.0 / v
	}
	return float64(len(xs)) / s
}`

const helperStatsPvariance = `func __gopy_stats_pvariance(xs []float64) float64 {
	if len(xs) == 0 {
		panic(NewException("StatisticsError: pvariance requires at least one data point"))
	}
	var sum float64
	for _, v := range xs {
		sum += v
	}
	mean := sum / float64(len(xs))
	var ss float64
	for _, v := range xs {
		d := v - mean
		ss += d * d
	}
	return ss / float64(len(xs))
}`

const helperStatsPstdev = `func __gopy_stats_pstdev(xs []float64) float64 {
	if len(xs) == 0 {
		panic(NewException("StatisticsError: pstdev requires at least one data point"))
	}
	var sum float64
	for _, v := range xs {
		sum += v
	}
	mean := sum / float64(len(xs))
	var ss float64
	for _, v := range xs {
		d := v - mean
		ss += d * d
	}
	return math.Sqrt(ss / float64(len(xs)))
}`

// helperUuid4 emits a 16-byte random UUID following RFC 4122 v4 layout
// (version 4, variant 10). Returned as a lowercase hyphenated hex string
// matching Python's str(uuid.uuid4()).
const helperUuid4 = `func __gopy_uuid4() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		b[0], b[1], b[2], b[3], b[4], b[5], b[6], b[7],
		b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15])
}`

// helperTextwrapDedent strips the longest common leading whitespace from
// every non-empty line. Mirrors textwrap.dedent semantics.
const helperTextwrapDedent = `func __gopy_textwrap_dedent(s string) string {
	lines := strings.Split(s, "\n")
	prefix := ""
	first := true
	for _, line := range lines {
		stripped := strings.TrimLeft(line, " \t")
		if stripped == "" {
			continue
		}
		lead := line[:len(line)-len(stripped)]
		if first {
			prefix = lead
			first = false
			continue
		}
		i := 0
		for i < len(prefix) && i < len(lead) && prefix[i] == lead[i] {
			i++
		}
		prefix = prefix[:i]
		if prefix == "" {
			break
		}
	}
	if prefix == "" {
		return s
	}
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if strings.HasPrefix(line, prefix) {
			b.WriteString(line[len(prefix):])
		} else {
			b.WriteString(line)
		}
	}
	return b.String()
}`

const helperTextwrapIndent = `func __gopy_textwrap_indent(s, prefix string) string {
	var b strings.Builder
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if strings.TrimSpace(line) != "" {
			b.WriteString(prefix)
		}
		b.WriteString(line)
	}
	return b.String()
}`

// helperTextwrapFill wraps text to width by breaking on spaces. CPython's
// textwrap is configurable; this shim covers the simple width-only form.
// helperStringTemplateType is the runtime Template struct used by
// string.Template(...). Supports $name and ${name} placeholders.
const helperStringTemplateType = `type __Template struct{ tmpl string }

func (t *__Template) substExpand(mapping any, safe bool) string {
	get := func(k string) (any, bool) {
		switch m := mapping.(type) {
		case map[string]any:
			v, ok := m[k]
			return v, ok
		case map[string]string:
			v, ok := m[k]
			return v, ok
		case map[string]int64:
			v, ok := m[k]
			return v, ok
		case map[string]float64:
			v, ok := m[k]
			return v, ok
		}
		return nil, false
	}
	var b strings.Builder
	s := t.tmpl
	i := 0
	for i < len(s) {
		if s[i] != '$' {
			b.WriteByte(s[i])
			i++
			continue
		}
		if i+1 >= len(s) {
			b.WriteByte('$')
			i++
			continue
		}
		if s[i+1] == '$' {
			b.WriteByte('$')
			i += 2
			continue
		}
		start := i + 1
		braced := false
		if s[start] == '{' {
			braced = true
			start++
		}
		j := start
		for j < len(s) {
			c := s[j]
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
				break
			}
			j++
		}
		name := s[start:j]
		if name == "" {
			b.WriteByte('$')
			i++
			continue
		}
		end := j
		if braced {
			if j >= len(s) || s[j] != '}' {
				if safe {
					b.WriteString(s[i:j])
					i = j
					continue
				}
				panic(NewException("ValueError: unclosed ${...} in template"))
			}
			end = j + 1
		}
		if v, ok := get(name); ok {
			b.WriteString(fmt.Sprint(v))
		} else if safe {
			b.WriteString(s[i:end])
		} else {
			panic(NewException("KeyError: " + name))
		}
		i = end
	}
	return b.String()
}

func (t *__Template) Substitute(mapping any) string {
	return t.substExpand(mapping, false)
}

func (t *__Template) SafeSubstitute(mapping any) string {
	return t.substExpand(mapping, true)
}`

const helperStringTemplateNew = `func __gopy_string_template_new(s string) *__Template {
	return &__Template{tmpl: s}
}`

// helperStringCapwords mirrors string.capwords: split on whitespace,
// title-case each word, join with single space.
const helperStringCapwords = `func __gopy_string_capwords(args ...string) string {
	if len(args) == 0 {
		return ""
	}
	s := args[0]
	sep := " "
	parts := strings.Fields(s)
	if len(args) > 1 {
		sep = args[1]
		parts = strings.Split(s, sep)
	}
	for i, w := range parts {
		if w == "" {
			continue
		}
		first := w[0]
		if first >= 'a' && first <= 'z' {
			first -= 32
		}
		rest := strings.ToLower(w[1:])
		parts[i] = string(first) + rest
	}
	return strings.Join(parts, sep)
}`

const helperTextwrapFill = `func __gopy_textwrap_fill(s string, width int64) string {
	w := int(width)
	if w <= 0 {
		return s
	}
	words := strings.Fields(s)
	var b strings.Builder
	col := 0
	for i, word := range words {
		wl := len(word)
		if i > 0 {
			if col+1+wl > w {
				b.WriteByte('\n')
				col = 0
			} else {
				b.WriteByte(' ')
				col++
			}
		}
		b.WriteString(word)
		col += wl
	}
	return b.String()
}`

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

func (p *__Path) String() string { return p.p }

func (p *__Path) Glob(pattern string) []*__Path {
	full := p.p
	if len(full) > 0 && full[len(full)-1] != '/' {
		full += "/"
	}
	matches, err := filepath.Glob(full + pattern)
	if err != nil {
		panic(err)
	}
	out := make([]*__Path, 0, len(matches))
	for _, m := range matches {
		out = append(out, &__Path{p: m})
	}
	return out
}

func (p *__Path) Iterdir() []*__Path {
	entries, err := os.ReadDir(p.p)
	if err != nil {
		panic(err)
	}
	out := make([]*__Path, 0, len(entries))
	for _, e := range entries {
		child := p.p
		if len(child) > 0 && child[len(child)-1] != '/' {
			child += "/"
		}
		child += e.Name()
		out = append(out, &__Path{p: child})
	}
	return out
}

func (p *__Path) Mkdir(args ...bool) {
	parents, exist_ok := false, false
	if len(args) > 0 {
		parents = args[0]
	}
	if len(args) > 1 {
		exist_ok = args[1]
	}
	var err error
	if parents {
		err = os.MkdirAll(p.p, 0o755)
	} else {
		err = os.Mkdir(p.p, 0o755)
	}
	if err != nil && !exist_ok {
		panic(err)
	}
}

func (p *__Path) Unlink() {
	if err := os.Remove(p.p); err != nil {
		panic(err)
	}
}

func (p *__Path) Suffix() string {
	for i := len(p.p) - 1; i >= 0; i-- {
		if p.p[i] == '/' {
			return ""
		}
		if p.p[i] == '.' {
			return p.p[i:]
		}
	}
	return ""
}

func (p *__Path) Stem() string {
	name := p.Name()
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			return name[:i]
		}
	}
	return name
}`

const helperPathNew = `func __gopy_path_new(s string) *__Path { return &__Path{p: s} }`

const helperOsGetcwd = `func __gopy_os_getcwd() string {
	d, err := os.Getwd()
	if err != nil {
		return ""
	}
	return d
}`

const helperOsListdir = `func __gopy_os_listdir(p string) []string {
	entries, err := os.ReadDir(p)
	if err != nil {
		panic(err)
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}`

const helperOsMakedirs = `func __gopy_os_makedirs(p string, args ...bool) {
	exist_ok := false
	if len(args) > 0 {
		exist_ok = args[0]
	}
	if err := os.MkdirAll(p, 0o755); err != nil {
		if !exist_ok {
			panic(err)
		}
	}
}`

// os.path.join: Python's join treats absolute parts as resets (later
// absolute path wins). filepath.Join does the same on Unix; on Windows
// the semantics differ for drive letters but we target Unix.
const helperPathJoin = `func __gopy_path_join(parts ...string) string {
	return filepath.Join(parts...)
}`

const helperPathExists = `func __gopy_path_exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}`

const helperPathIsfile = `func __gopy_path_isfile(p string) bool {
	i, err := os.Stat(p)
	return err == nil && !i.IsDir()
}`

const helperPathIsdir = `func __gopy_path_isdir(p string) bool {
	i, err := os.Stat(p)
	return err == nil && i.IsDir()
}`

const helperPathSplitext = `func __gopy_path_splitext(p string) []string {
	ext := filepath.Ext(p)
	base := p
	if ext != "" {
		base = p[:len(p)-len(ext)]
	}
	return []string{base, ext}
}`

const helperPathAbspath = `func __gopy_path_abspath(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return a
}`

const helperOsRemove = `func __gopy_os_remove(p string) {
	if err := os.Remove(p); err != nil {
		panic(err)
	}
}`

const helperOsRename = `func __gopy_os_rename(src, dst string) {
	if err := os.Rename(src, dst); err != nil {
		panic(err)
	}
}`

const helperOsMkdir = `func __gopy_os_mkdir(p string) {
	if err := os.Mkdir(p, 0o755); err != nil {
		panic(err)
	}
}`

const helperOsRmdir = `func __gopy_os_rmdir(p string) {
	if err := os.Remove(p); err != nil {
		panic(err)
	}
}`

// helperPathSplit mirrors Python's os.path.split: returns [head, tail].
// CPython splits on the last separator; filepath.Split keeps the trailing
// slash on head, which we strip to match the Python output.
const helperPathSplit = `func __gopy_path_split(p string) []string {
	d, f := filepath.Split(p)
	if len(d) > 1 && d[len(d)-1] == filepath.Separator {
		d = d[:len(d)-1]
	}
	return []string{d, f}
}`

const helperPathRelpath = `func __gopy_path_relpath(target string, base ...string) string {
	b := "."
	if len(base) > 0 {
		b = base[0]
	}
	r, err := filepath.Rel(b, target)
	if err != nil {
		return target
	}
	return r
}`

const helperPathGetsize = `func __gopy_path_getsize(p string) int64 {
	i, err := os.Stat(p)
	if err != nil {
		panic(err)
	}
	return i.Size()
}`

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

// helperTimedeltaNew accepts the full Python parameter order:
// (days, seconds, microseconds, milliseconds, minutes, hours, weeks).
// All are float64 so fractional days / hours work like CPython.
const helperTimedeltaNew = `func __gopy_timedelta_new(days, seconds, microseconds, milliseconds, minutes, hours, weeks float64) *__Timedelta {
	total := days * float64(24*time.Hour)
	total += seconds * float64(time.Second)
	total += microseconds * float64(time.Microsecond)
	total += milliseconds * float64(time.Millisecond)
	total += minutes * float64(time.Minute)
	total += hours * float64(time.Hour)
	total += weeks * 7 * float64(24*time.Hour)
	return &__Timedelta{d: time.Duration(total)}
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
}

func (d *__Datetime) Strftime(layout string) string {
	return __gopy_datetime_strftime(d.t, layout)
}`

// helperDatetimeNow returns Python's datetime.datetime.now() as a
// *__Datetime so it can take part in timedelta arithmetic.
const helperDatetimeNow = `func __gopy_datetime_now() *__Datetime { return &__Datetime{t: time.Now()} }`

const helperDatetimeUtcnow = `func __gopy_datetime_utcnow() *__Datetime { return &__Datetime{t: time.Now().UTC()} }`

// helperDatetimeFromTs builds *__Datetime from a Unix timestamp (seconds
// since epoch, may be fractional). Mirrors datetime.fromtimestamp's local
// timezone interpretation.
const helperDatetimeFromTs = `func __gopy_datetime_fromts(ts float64) *__Datetime {
	sec := int64(ts)
	nsec := int64((ts - float64(sec)) * 1e9)
	return &__Datetime{t: time.Unix(sec, nsec)}
}`

// helperDatetimeFromIso parses an ISO-8601 timestamp, mirroring
// datetime.fromisoformat. Accepts the common YYYY-MM-DD,
// YYYY-MM-DDTHH:MM:SS, and the same forms with fractional seconds.
const helperDatetimeFromIso = `func __gopy_datetime_fromiso(s string) *__Datetime {
	layouts := []string{
		"2006-01-02T15:04:05.000000",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.000000",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return &__Datetime{t: t}
		}
	}
	panic(NewException("ValueError: Invalid isoformat string: " + s))
}`

// helperPyTimeFormat translates a Python strftime/strptime layout to the
// equivalent Go reference-time layout. Subset covers the codes most
// fixtures lean on: Y/m/d/H/M/S/y/B/b/A/a/p/j.
const helperPyTimeFormat = `func __gopy_py_time_format(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] != '%' || i+1 >= len(s) {
			b.WriteByte(s[i])
			i++
			continue
		}
		c := s[i+1]
		switch c {
		case 'Y':
			b.WriteString("2006")
		case 'y':
			b.WriteString("06")
		case 'm':
			b.WriteString("01")
		case 'd':
			b.WriteString("02")
		case 'H':
			b.WriteString("15")
		case 'I':
			b.WriteString("03")
		case 'M':
			b.WriteString("04")
		case 'S':
			b.WriteString("05")
		case 'B':
			b.WriteString("January")
		case 'b':
			b.WriteString("Jan")
		case 'A':
			b.WriteString("Monday")
		case 'a':
			b.WriteString("Mon")
		case 'p':
			b.WriteString("PM")
		case 'j':
			b.WriteString("002")
		case 'z':
			b.WriteString("-0700")
		case '%':
			b.WriteByte('%')
		default:
			b.WriteByte('%')
			b.WriteByte(c)
		}
		i += 2
	}
	return b.String()
}`

const helperDatetimeStrftime = `func __gopy_datetime_strftime(t time.Time, layout string) string {
	return t.Format(__gopy_py_time_format(layout))
}`

const helperDatetimeStrptime = `func __gopy_datetime_strptime(s, layout string) *__Datetime {
	t, err := time.Parse(__gopy_py_time_format(layout), s)
	if err != nil {
		panic(err)
	}
	return &__Datetime{t: t}
}`

// helperDateType mirrors Python's datetime.date — year/month/day and an
// isoformat that prints YYYY-MM-DD.
const helperDateType = `type __Date struct {
	Y int64
	M int64
	D int64
}

func (d *__Date) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Y, d.M, d.D)
}

func (d *__Date) Isoformat() string { return d.String() }

func (d *__Date) Year() int64  { return d.Y }
func (d *__Date) Month() int64 { return d.M }
func (d *__Date) Day() int64   { return d.D }

func (d *__Date) Strftime(layout string) string {
	t := time.Date(int(d.Y), time.Month(int(d.M)), int(d.D), 0, 0, 0, 0, time.UTC)
	return __gopy_datetime_strftime(t, layout)
}`

const helperDateNew = `func __gopy_date_new(y, m, d int64) *__Date {
	return &__Date{Y: y, M: m, D: d}
}`

// helperTimeType mirrors Python's datetime.time — hour/minute/second
// (microseconds dropped) and isoformat printing HH:MM:SS.
const helperTimeType = `type __Time struct {
	H int64
	M int64
	S int64
}

func (t *__Time) String() string {
	return fmt.Sprintf("%02d:%02d:%02d", t.H, t.M, t.S)
}

func (t *__Time) Isoformat() string { return t.String() }

func (t *__Time) Hour() int64   { return t.H }
func (t *__Time) Minute() int64 { return t.M }
func (t *__Time) Second() int64 { return t.S }`

const helperTimeNew = `func __gopy_time_new(args ...int64) *__Time {
	t := &__Time{}
	if len(args) > 0 { t.H = args[0] }
	if len(args) > 1 { t.M = args[1] }
	if len(args) > 2 { t.S = args[2] }
	return t
}`

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
