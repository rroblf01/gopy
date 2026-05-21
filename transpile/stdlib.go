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
			"argv":         {GoExpr: "os.Args", GoImport: "os"},
			"platform":     {GoExpr: "runtime.GOOS", GoImport: "runtime"},
			"version":      {GoExpr: `"3.12.0 (gopy)"`},
			"version_info": {GoExpr: "__gopy_sys_version_info", Helper: helperSysVersionInfo, HelperName: "__gopy_sys_version_info"},
			"maxsize":      {GoExpr: "int64(9223372036854775807)"},
			"byteorder":    {GoExpr: `"little"`},
		},
		Funcs: map[string]stdlibFunc{
			"exit":       {GoFunc: "os.Exit", GoImport: "os", IntArg0: true},
			"getsizeof":  {GoFunc: "__gopy_sys_getsizeof", Helper: helperSysGetsizeof, HelperImports: []string{"unsafe", "reflect"}, RetKind: "int"},
			"intern":     {GoFunc: "__gopy_sys_intern", Helper: helperSysIntern, RetKind: "str"},
		},
	},
	"os": {
		Attrs: map[string]stdlibAttr{
			"sep":     {GoExpr: `string(os.PathSeparator)`, GoImport: "os"},
			"linesep": {GoExpr: `"\n"`},
			"environ": {GoExpr: "__gopy_os_environ()", Helper: helperOsEnviron, HelperName: "__gopy_os_environ", HelperImports: []string{"os", "strings"}},
		},
		Funcs: map[string]stdlibFunc{
			"getenv":    {GoFunc: "os.Getenv", GoImport: "os"},
			"getcwd":    {GoFunc: "__gopy_os_getcwd", GoImport: "os", Helper: helperOsGetcwd, RetKind: "str"},
			"listdir":   {GoFunc: "__gopy_os_listdir", GoImport: "os", Helper: helperOsListdir},
			"makedirs":  {GoFunc: "__gopy_os_makedirs", GoImport: "os", Helper: helperOsMakedirs},
			"remove":    {GoFunc: "__gopy_os_remove", GoImport: "os", Helper: helperOsRemove},
			"rename":    {GoFunc: "__gopy_os_rename", GoImport: "os", Helper: helperOsRename},
			"mkdir":     {GoFunc: "__gopy_os_mkdir", GoImport: "os", Helper: helperOsMkdir},
			"rmdir":     {GoFunc: "__gopy_os_rmdir", GoImport: "os", Helper: helperOsRmdir},
			"cpu_count": {GoFunc: "__gopy_os_cpu_count", Helper: helperOsCPUCount, HelperImports: []string{"runtime"}, RetKind: "int"},
			"urandom":   {GoFunc: "__gopy_os_urandom", Helper: helperOsUrandom, HelperImports: []string{"crypto/rand"}, RetKind: "str"},
			"walk":      {GoFunc: "__gopy_os_walk", Helper: helperOsWalk, HelperImports: []string{"os", "path/filepath"}},
			"chdir":     {GoFunc: "os.Chdir", GoImport: "os"},
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
					"abspath":      {GoFunc: "__gopy_path_abspath", GoImport: "path/filepath", Helper: helperPathAbspath, RetKind: "str"},
					"split":        {GoFunc: "__gopy_path_split", GoImport: "path/filepath", Helper: helperPathSplit},
					"relpath":      {GoFunc: "__gopy_path_relpath", GoImport: "path/filepath", Helper: helperPathRelpath, RetKind: "str"},
					"getsize":      {GoFunc: "__gopy_path_getsize", GoImport: "os", Helper: helperPathGetsize, RetKind: "int"},
					"getmtime":     {GoFunc: "__gopy_path_getmtime", GoImport: "os", Helper: helperPathGetmtime, RetKind: "float"},
					"normpath":     {GoFunc: "filepath.Clean", GoImport: "path/filepath", RetKind: "str"},
					"expanduser":   {GoFunc: "__gopy_path_expanduser", GoImport: "os", Helper: helperPathExpanduser, RetKind: "str"},
					"expandvars":   {GoFunc: "os.ExpandEnv", GoImport: "os", RetKind: "str"},
					"commonprefix": {GoFunc: "__gopy_path_commonprefix", Helper: helperPathCommonprefix, RetKind: "str"},
					"samefile":     {GoFunc: "__gopy_path_samefile", GoImport: "os", Helper: helperPathSamefile, RetKind: "bool"},
					"isabs":        {GoFunc: "filepath.IsAbs", GoImport: "path/filepath", RetKind: "bool"},
					"lexists":      {GoFunc: "__gopy_path_lexists", GoImport: "os", Helper: helperPathLexists, RetKind: "bool"},
					"realpath":     {GoFunc: "__gopy_path_realpath", GoImport: "path/filepath", Helper: helperPathRealpath, RetKind: "str"},
					"commonpath":   {GoFunc: "__gopy_path_commonpath", GoImport: "path/filepath", Helper: helperPathCommonpath, HelperImports: []string{"strings"}, RetKind: "str"},
					"normcase":     {GoFunc: "__gopy_path_normcase", Helper: helperPathNormcase, RetKind: "str"},
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
			"strftime":     {GoFunc: "__gopy_time_strftime", GoImport: "time", Helper: helperTimeStrftime, HelperImports: []string{"strings"}, RetKind: "str"},
			"localtime":    {GoFunc: "__gopy_time_localtime", GoImport: "time", Helper: helperTimeLocaltime},
			"gmtime":       {GoFunc: "__gopy_time_gmtime", GoImport: "time", Helper: helperTimeGmtime},
			"mktime":       {GoFunc: "__gopy_time_mktime", GoImport: "time", Helper: helperTimeMktime, RetKind: "float"},
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
			"lcm":      {GoFunc: "__gopy_math_lcm", Helper: helperMathLcm, ExtraHelpers: map[string]string{"__gopy_math_gcd": helperMathGcd}, RetKind: "int"},
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
			"asin":      {GoFunc: "math.Asin", GoImport: "math"},
			"acos":      {GoFunc: "math.Acos", GoImport: "math"},
			"sinh":      {GoFunc: "math.Sinh", GoImport: "math"},
			"cosh":      {GoFunc: "math.Cosh", GoImport: "math"},
			"tanh":      {GoFunc: "math.Tanh", GoImport: "math"},
			"asinh":     {GoFunc: "math.Asinh", GoImport: "math"},
			"acosh":     {GoFunc: "math.Acosh", GoImport: "math"},
			"atanh":     {GoFunc: "math.Atanh", GoImport: "math"},
			"expm1":     {GoFunc: "math.Expm1", GoImport: "math"},
			"log1p":     {GoFunc: "math.Log1p", GoImport: "math"},
			"erf":       {GoFunc: "math.Erf", GoImport: "math"},
			"erfc":      {GoFunc: "math.Erfc", GoImport: "math"},
			"gamma":     {GoFunc: "math.Gamma", GoImport: "math"},
			"lgamma":    {GoFunc: "__gopy_math_lgamma", GoImport: "math", Helper: helperMathLgamma},
			"isclose":   {GoFunc: "__gopy_math_isclose", GoImport: "math", Helper: helperMathIsclose, RetKind: "bool"},
		},
	},
	"hashlib": {
		Funcs: map[string]stdlibFunc{
			"sha256": {GoFunc: "__gopy_hashlib_sha256", GoImport: "crypto/sha256", Helper: helperHashlibSha256, RetTag: "__Hasher", ExtraHelpers: map[string]string{"__Hasher": helperHasherType}, HelperImports: []string{"encoding/hex", "crypto/md5", "crypto/sha1", "crypto/sha512"}},
			"md5":    {GoFunc: "__gopy_hashlib_md5", GoImport: "crypto/md5", Helper: helperHashlibMd5, RetTag: "__Hasher", ExtraHelpers: map[string]string{"__Hasher": helperHasherType}, HelperImports: []string{"encoding/hex", "crypto/sha256", "crypto/sha1", "crypto/sha512"}},
			"sha1":   {GoFunc: "__gopy_hashlib_sha1", GoImport: "crypto/sha1", Helper: helperHashlibSha1, RetTag: "__Hasher", ExtraHelpers: map[string]string{"__Hasher": helperHasherType}, HelperImports: []string{"encoding/hex", "crypto/md5", "crypto/sha256", "crypto/sha512"}},
			"sha512": {GoFunc: "__gopy_hashlib_sha512", GoImport: "crypto/sha512", Helper: helperHashlibSha512, RetTag: "__Hasher", ExtraHelpers: map[string]string{"__Hasher": helperHasherType}, HelperImports: []string{"encoding/hex", "crypto/md5", "crypto/sha256", "crypto/sha1"}},
			"new":    {GoFunc: "__gopy_hashlib_new", Helper: helperHashlibNew, RetTag: "__Hasher", ExtraHelpers: map[string]string{"__Hasher": helperHasherType}, HelperImports: []string{"encoding/hex", "crypto/md5", "crypto/sha1", "crypto/sha256", "crypto/sha512"}},
		},
	},
	"secrets": {
		Funcs: map[string]stdlibFunc{
			"token_hex":     {GoFunc: "__gopy_secrets_token_hex", GoImport: "crypto/rand", Helper: helperSecretsTokenHex, HelperImports: []string{"encoding/hex"}, RetKind: "str"},
			"token_urlsafe": {GoFunc: "__gopy_secrets_token_urlsafe", GoImport: "crypto/rand", Helper: helperSecretsTokenUrl, HelperImports: []string{"encoding/base64"}, RetKind: "str"},
			"token_bytes":   {GoFunc: "__gopy_secrets_token_bytes", GoImport: "crypto/rand", Helper: helperSecretsTokenBytes, RetKind: "str"},
			"randbelow":     {GoFunc: "__gopy_secrets_randbelow", Helper: helperSecretsRandbelow, HelperImports: []string{"crypto/rand", "math/big"}, RetKind: "int"},
			"compare_digest": {GoFunc: "__gopy_compare_digest", Helper: helperCompareDigest, HelperImports: []string{"crypto/subtle"}, RetKind: "bool"},
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
			"request": {
				Funcs: map[string]stdlibFunc{
					"urlopen":     {GoFunc: "__gopy_url_urlopen", Helper: helperURLOpen, HelperImports: []string{"io", "net/http"}, RetTag: "__HTTPResponse", ExtraHelpers: map[string]string{"__HTTPResponse": helperHTTPResponseType}},
					"Request":     {GoFunc: "__gopy_url_request_new", Helper: helperURLRequestNew, RetTag: "__URLRequest", ExtraHelpers: map[string]string{"__URLRequest": helperURLRequestType}},
					"urlretrieve": {GoFunc: "__gopy_url_urlretrieve", Helper: helperURLRetrieve, HelperImports: []string{"io", "net/http", "os"}},
				},
			},
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
					"urljoin":      {GoFunc: "__gopy_url_urljoin", GoImport: "net/url", Helper: helperURLUrljoin, RetKind: "str"},
					"urlsplit":     {GoFunc: "__gopy_url_urlparse", GoImport: "net/url", Helper: helperURLUrlparse, RetTag: "__URLParseResult", ExtraHelpers: map[string]string{"__URLParseResult": helperURLParseResultType}},
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
			"printable":       {GoExpr: "\"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!\\\"#$%&'()*+,-./:;<=>?@[\\\\]^_`{|}~ \\t\\n\\r\\x0b\\x0c\""},
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
	"shutil": {
		Funcs: map[string]stdlibFunc{
			"rmtree":   {GoFunc: "__gopy_shutil_rmtree", GoImport: "os", Helper: helperShutilRmtree},
			"copy":     {GoFunc: "__gopy_shutil_copy", GoImport: "io", Helper: helperShutilCopy, HelperImports: []string{"os"}},
			"copyfile": {GoFunc: "__gopy_shutil_copy", GoImport: "io", Helper: helperShutilCopy, HelperImports: []string{"os"}},
			"move":     {GoFunc: "__gopy_shutil_move", GoImport: "os", Helper: helperShutilMove},
			"which":    {GoFunc: "__gopy_shutil_which", Helper: helperShutilWhich, HelperImports: []string{"os/exec"}, RetKind: "str"},
		},
	},
	"tempfile": {
		Funcs: map[string]stdlibFunc{
			"mkdtemp":     {GoFunc: "__gopy_tempfile_mkdtemp", GoImport: "os", Helper: helperTempfileMkdtemp, RetKind: "str"},
			"gettempdir":  {GoFunc: "os.TempDir", GoImport: "os", RetKind: "str"},
			"mkstemp":     {GoFunc: "__gopy_tempfile_mkstemp", GoImport: "os", Helper: helperTempfileMkstemp},
		},
	},
	"cmath": {
		Attrs: map[string]stdlibAttr{
			"pi":   {GoExpr: "math.Pi", GoImport: "math"},
			"e":    {GoExpr: "math.E", GoImport: "math"},
			"tau":  {GoExpr: "math.Pi * 2", GoImport: "math"},
			"inf":  {GoExpr: "math.Inf(1)", GoImport: "math"},
			"nan":  {GoExpr: "math.NaN()", GoImport: "math"},
			"infj": {GoExpr: "complex(0, math.Inf(1))", GoImport: "math"},
			"nanj": {GoExpr: "complex(0, math.NaN())", GoImport: "math"},
		},
		Funcs: map[string]stdlibFunc{
			"sqrt":    {GoFunc: "__gopy_cmath_sqrt", Helper: helperCmathSqrt, HelperImports: []string{"math/cmplx"}},
			"exp":     {GoFunc: "cmplx.Exp", GoImport: "math/cmplx"},
			"log":     {GoFunc: "cmplx.Log", GoImport: "math/cmplx"},
			"log10":   {GoFunc: "cmplx.Log10", GoImport: "math/cmplx"},
			"sin":     {GoFunc: "cmplx.Sin", GoImport: "math/cmplx"},
			"cos":     {GoFunc: "cmplx.Cos", GoImport: "math/cmplx"},
			"tan":     {GoFunc: "cmplx.Tan", GoImport: "math/cmplx"},
			"asin":    {GoFunc: "cmplx.Asin", GoImport: "math/cmplx"},
			"acos":    {GoFunc: "cmplx.Acos", GoImport: "math/cmplx"},
			"atan":    {GoFunc: "cmplx.Atan", GoImport: "math/cmplx"},
			"sinh":    {GoFunc: "cmplx.Sinh", GoImport: "math/cmplx"},
			"cosh":    {GoFunc: "cmplx.Cosh", GoImport: "math/cmplx"},
			"tanh":    {GoFunc: "cmplx.Tanh", GoImport: "math/cmplx"},
			"phase":   {GoFunc: "__gopy_cmath_phase", Helper: helperCmathPhase, HelperImports: []string{"math"}, RetKind: "float"},
			"polar":   {GoFunc: "__gopy_cmath_polar", Helper: helperCmathPolar, HelperImports: []string{"math"}},
			"rect":    {GoFunc: "__gopy_cmath_rect", Helper: helperCmathRect, HelperImports: []string{"math"}},
			"isnan":   {GoFunc: "cmplx.IsNaN", GoImport: "math/cmplx", RetKind: "bool"},
			"isinf":   {GoFunc: "cmplx.IsInf", GoImport: "math/cmplx", RetKind: "bool"},
			"isfinite": {GoFunc: "__gopy_cmath_isfinite", Helper: helperCmathIsfinite, HelperImports: []string{"math/cmplx"}, RetKind: "bool"},
		},
	},
	"copy": {
		Funcs: map[string]stdlibFunc{
			"copy":     {GoFunc: "__gopy_copy_shallow", Helper: helperCopyShallow, HelperImports: []string{"encoding/json"}},
			"deepcopy": {GoFunc: "__gopy_copy_deep", Helper: helperCopyDeep, HelperImports: []string{"encoding/json"}},
		},
	},
	"mimetypes": {
		Funcs: map[string]stdlibFunc{
			"guess_type":      {GoFunc: "__gopy_mimetypes_guess", Helper: helperMimetypesGuess, HelperImports: []string{"mime", "path/filepath"}},
			"guess_extension": {GoFunc: "__gopy_mimetypes_guess_ext", Helper: helperMimetypesGuessExt, HelperImports: []string{"mime"}, RetKind: "str"},
		},
	},
	"xml": {
		Subs: map[string]stdlibModule{
			"etree": {
				Subs: map[string]stdlibModule{
					"ElementTree": {
						Funcs: map[string]stdlibFunc{
							"fromstring": {GoFunc: "__gopy_xml_fromstring", Helper: helperXMLFromstring, RetTag: "__XMLElement", ExtraHelpers: map[string]string{"__XMLElement": helperXMLElementType}, HelperImports: []string{"encoding/xml", "strings"}},
						},
					},
				},
			},
		},
	},
	"http": {
		Subs: map[string]stdlibModule{
			"client": {
				Funcs: map[string]stdlibFunc{
					"HTTPSConnection": {GoFunc: "__gopy_http_client_new", Helper: helperHTTPClientNew, RetTag: "__HTTPClient", ExtraHelpers: map[string]string{"__HTTPClient": helperHTTPClientType}, HelperImports: []string{"net/http", "io", "strings"}},
					"HTTPConnection":  {GoFunc: "__gopy_http_client_new_plain", Helper: helperHTTPClientNewPlain, RetTag: "__HTTPClient", ExtraHelpers: map[string]string{"__HTTPClient": helperHTTPClientType}, HelperImports: []string{"net/http", "io", "strings"}},
				},
			},
		},
	},
	"struct": {
		Funcs: map[string]stdlibFunc{
			"pack":      {GoFunc: "__gopy_struct_pack", Helper: helperStructPack, HelperImports: []string{"encoding/binary", "bytes"}, RetKind: "str"},
			"unpack":    {GoFunc: "__gopy_struct_unpack", Helper: helperStructUnpack, HelperImports: []string{"encoding/binary"}},
			"calcsize":  {GoFunc: "__gopy_struct_calcsize", Helper: helperStructCalcsize, RetKind: "int"},
		},
	},
	"fractions": {
		Funcs: map[string]stdlibFunc{
			"Fraction": {GoFunc: "__gopy_fraction_new", Helper: helperFractionNew, RetTag: "__Fraction", ExtraHelpers: map[string]string{"__Fraction": helperFractionType}, HelperImports: []string{"fmt", "strconv", "strings"}},
		},
	},
	"decimal": {
		Funcs: map[string]stdlibFunc{
			"Decimal": {GoFunc: "__gopy_decimal_new", Helper: helperDecimalNew, RetTag: "__Decimal", ExtraHelpers: map[string]string{"__Decimal": helperDecimalType}, HelperImports: []string{"fmt", "strconv"}},
		},
	},
	"binascii": {
		Funcs: map[string]stdlibFunc{
			"hexlify":   {GoFunc: "__gopy_binascii_hexlify", Helper: helperBinasciiHexlify, HelperImports: []string{"encoding/hex"}, RetKind: "str"},
			"b2a_hex":   {GoFunc: "__gopy_binascii_hexlify", Helper: helperBinasciiHexlify, HelperImports: []string{"encoding/hex"}, RetKind: "str"},
			"unhexlify": {GoFunc: "__gopy_binascii_unhexlify", Helper: helperBinasciiUnhexlify, HelperImports: []string{"encoding/hex"}, RetKind: "str"},
			"a2b_hex":   {GoFunc: "__gopy_binascii_unhexlify", Helper: helperBinasciiUnhexlify, HelperImports: []string{"encoding/hex"}, RetKind: "str"},
			"crc32":     {GoFunc: "__gopy_binascii_crc32", Helper: helperBinasciiCrc32, HelperImports: []string{"hash/crc32"}, RetKind: "int"},
		},
	},
	"pickle": {
		Funcs: map[string]stdlibFunc{
			"dumps": {GoFunc: "__gopy_pickle_dumps", Helper: helperPickleDumps, HelperImports: []string{"encoding/json"}, RetKind: "str"},
			"loads": {GoFunc: "__gopy_pickle_loads", Helper: helperPickleLoads, HelperImports: []string{"encoding/json"}},
		},
	},
	"configparser": {
		Funcs: map[string]stdlibFunc{
			"ConfigParser": {GoFunc: "__gopy_configparser_new", Helper: helperConfigParserNew, RetTag: "__ConfigParser", ExtraHelpers: map[string]string{"__ConfigParser": helperConfigParserType}, HelperImports: []string{"bufio", "os", "strings"}},
		},
	},
	"email": {
		Subs: map[string]stdlibModule{
			"utils": {
				Funcs: map[string]stdlibFunc{
					"formatdate":   {GoFunc: "__gopy_email_formatdate", Helper: helperEmailFormatdate, HelperImports: []string{"time"}, RetKind: "str"},
					"parsedate":    {GoFunc: "__gopy_email_parsedate", Helper: helperEmailParsedate, HelperImports: []string{"time"}},
					"format_datetime": {GoFunc: "__gopy_email_format_datetime", Helper: helperEmailFormatDatetime, HelperImports: []string{"time"}, RetKind: "str"},
				},
			},
		},
	},
	"argparse": {
		Funcs: map[string]stdlibFunc{
			"ArgumentParser": {GoFunc: "__gopy_argparse_new", Helper: helperArgparseNew, RetTag: "__ArgParser", ExtraHelpers: map[string]string{"__ArgParser": helperArgparseType}, HelperImports: []string{"os", "strconv", "strings", "fmt"}},
		},
	},
	"io": {
		Funcs: map[string]stdlibFunc{
			"StringIO": {GoFunc: "__gopy_io_stringio_new", Helper: helperIOStringIONew, RetTag: "__StringIO", ExtraHelpers: map[string]string{"__StringIO": helperIOStringIOType}},
			"BytesIO":  {GoFunc: "__gopy_io_bytesio_new", Helper: helperIOBytesIONew, RetTag: "__StringIO", ExtraHelpers: map[string]string{"__StringIO": helperIOStringIOType}},
		},
	},
	"weakref": {
		Funcs: map[string]stdlibFunc{
			// gopy has no notion of weak references (Go GC handles it).
			// Both forms collapse to identity-pass-through helpers so
			// libraries that use weakref keep compiling.
			"ref":   {GoFunc: "__gopy_weakref_ref", Helper: helperWeakrefRef},
			"proxy": {GoFunc: "__gopy_weakref_ref", Helper: helperWeakrefRef},
		},
	},
	"pprint": {
		Funcs: map[string]stdlibFunc{
			"pprint": {GoFunc: "__gopy_pprint", Helper: helperPprint, HelperImports: []string{"fmt"}},
			"pformat": {GoFunc: "__gopy_pformat", Helper: helperPformat, HelperImports: []string{"fmt"}, RetKind: "str"},
		},
	},
	"traceback": {
		Funcs: map[string]stdlibFunc{
			"format_exc": {GoFunc: "__gopy_traceback_format_exc", Helper: helperTracebackFormatExc, RetKind: "str"},
			"print_exc":  {GoFunc: "__gopy_traceback_print_exc", Helper: helperTracebackPrintExc, HelperImports: []string{"fmt", "os"}},
		},
	},
	"inspect": {
		Funcs: map[string]stdlibFunc{
			"signature":   {GoFunc: "__gopy_inspect_sig", Helper: helperInspectSig, RetKind: "str"},
			"getsource":   {GoFunc: "__gopy_inspect_source", Helper: helperInspectSource, RetKind: "str"},
			"getmembers":  {GoFunc: "__gopy_inspect_members", Helper: helperInspectMembers},
			"isfunction":  {GoFunc: "__gopy_inspect_isfunc", Helper: helperInspectIsfunc, RetKind: "bool"},
			"isclass":     {GoFunc: "__gopy_inspect_isclass", Helper: helperInspectIsclass, RetKind: "bool"},
			"ismethod":    {GoFunc: "__gopy_inspect_isfunc", Helper: helperInspectIsfunc, RetKind: "bool"},
			"currentframe": {GoFunc: "__gopy_inspect_frame", Helper: helperInspectFrame},
			"stack":       {GoFunc: "__gopy_inspect_stack", Helper: helperInspectStack},
		},
	},
	"operator": {
		Funcs: map[string]stdlibFunc{
			"add":         {GoFunc: "__gopy_operator_add", Helper: helperOpAdd, RetKind: "int"},
			"sub":         {GoFunc: "__gopy_operator_sub", Helper: helperOpSub, RetKind: "int"},
			"mul":         {GoFunc: "__gopy_operator_mul", Helper: helperOpMul, RetKind: "int"},
			"itemgetter":  {GoFunc: "__gopy_operator_itemgetter", Helper: helperOpItemgetter},
			"attrgetter":  {GoFunc: "__gopy_operator_attrgetter", Helper: helperOpAttrgetter},
		},
	},
	"array": {
		Funcs: map[string]stdlibFunc{
			"array": {GoFunc: "__gopy_array_new", Helper: helperArrayNew, HelperImports: []string{"fmt"}},
		},
	},
	"pwd": {
		Funcs: map[string]stdlibFunc{
			"getpwuid": {GoFunc: "__gopy_pwd_stub", Helper: helperPwdStub},
			"getpwnam": {GoFunc: "__gopy_pwd_stub", Helper: helperPwdStub},
		},
	},
	"grp": {
		Funcs: map[string]stdlibFunc{
			"getgrgid": {GoFunc: "__gopy_pwd_stub", Helper: helperPwdStub},
			"getgrnam": {GoFunc: "__gopy_pwd_stub", Helper: helperPwdStub},
		},
	},
	"selectors": {
		Attrs: map[string]stdlibAttr{
			"EVENT_READ":  {GoExpr: "int64(1)"},
			"EVENT_WRITE": {GoExpr: "int64(2)"},
		},
	},
	"signal": {
		Attrs: map[string]stdlibAttr{
			"SIGINT":  {GoExpr: "int64(2)"},
			"SIGTERM": {GoExpr: "int64(15)"},
			"SIGHUP":  {GoExpr: "int64(1)"},
			"SIGQUIT": {GoExpr: "int64(3)"},
			"SIGKILL": {GoExpr: "int64(9)"},
			"SIGUSR1": {GoExpr: "int64(10)"},
			"SIGUSR2": {GoExpr: "int64(12)"},
			"SIG_DFL": {GoExpr: "any(0)"},
			"SIG_IGN": {GoExpr: "any(1)"},
		},
		Funcs: map[string]stdlibFunc{
			"signal":     {GoFunc: "__gopy_signal_noop", Helper: helperSignalNoop},
			"getsignal":  {GoFunc: "__gopy_signal_noop", Helper: helperSignalNoop},
			"set_wakeup_fd": {GoFunc: "__gopy_signal_noop_int", Helper: helperSignalNoopInt, RetKind: "int"},
		},
	},
	"atexit": {
		Funcs: map[string]stdlibFunc{
			"register":   {GoFunc: "__gopy_atexit_noop", Helper: helperAtexitNoop},
			"unregister": {GoFunc: "__gopy_atexit_noop", Helper: helperAtexitNoop},
		},
	},
	"gc": {
		Funcs: map[string]stdlibFunc{
			"collect":   {GoFunc: "__gopy_gc_collect", Helper: helperGCCollect, HelperImports: []string{"runtime"}, RetKind: "int"},
			"disable":   {GoFunc: "__gopy_gc_noop", Helper: helperGCNoop},
			"enable":    {GoFunc: "__gopy_gc_noop", Helper: helperGCNoop},
			"isenabled": {GoFunc: "__gopy_gc_enabled", Helper: helperGCEnabled, RetKind: "bool"},
		},
	},
	"contextlib": {
		Funcs: map[string]stdlibFunc{
			"contextmanager": {GoFunc: "__gopy_contextmanager_unused"},
			"suppress":       {GoFunc: "__gopy_suppress_unused"},
		},
	},
	"asyncio": {
		Funcs: map[string]stdlibFunc{
			"run":   {GoFunc: "__gopy_asyncio_run_unused"},
			"sleep": {GoFunc: "__gopy_asyncio_sleep_unused"},
		},
	},
	"queue": {
		Funcs: map[string]stdlibFunc{
			"Queue":     {GoFunc: "__gopy_queue_new", Helper: helperQueueNew, RetTag: "__Queue", ExtraHelpers: map[string]string{"__Queue": helperQueueType}, HelperImports: []string{"sync"}},
			"LifoQueue": {GoFunc: "__gopy_lifo_queue_new", Helper: helperLifoQueueNew, RetTag: "__Queue", ExtraHelpers: map[string]string{"__Queue": helperQueueType}, HelperImports: []string{"sync"}},
		},
	},
	"html": {
		Funcs: map[string]stdlibFunc{
			"escape":   {GoFunc: "__gopy_html_escape", Helper: helperHTMLEscape, HelperImports: []string{"strings"}, RetKind: "str"},
			"unescape": {GoFunc: "html.UnescapeString", GoImport: "html", RetKind: "str"},
		},
	},
	"gzip": {
		Funcs: map[string]stdlibFunc{
			"compress":   {GoFunc: "__gopy_gzip_compress", GoImport: "compress/gzip", Helper: helperGzipCompress, HelperImports: []string{"bytes"}, RetKind: "str"},
			"decompress": {GoFunc: "__gopy_gzip_decompress", GoImport: "compress/gzip", Helper: helperGzipDecompress, HelperImports: []string{"bytes", "io"}, RetKind: "str"},
		},
	},
	"zlib": {
		Funcs: map[string]stdlibFunc{
			"compress":   {GoFunc: "__gopy_zlib_compress", GoImport: "compress/zlib", Helper: helperZlibCompress, HelperImports: []string{"bytes"}, RetKind: "str"},
			"decompress": {GoFunc: "__gopy_zlib_decompress", GoImport: "compress/zlib", Helper: helperZlibDecompress, HelperImports: []string{"bytes", "io"}, RetKind: "str"},
			"crc32":      {GoFunc: "__gopy_zlib_crc32", GoImport: "hash/crc32", Helper: helperZlibCrc32, RetKind: "int"},
			"adler32":    {GoFunc: "__gopy_zlib_adler32", GoImport: "hash/adler32", Helper: helperZlibAdler32, RetKind: "int"},
		},
	},
	"glob": {
		Funcs: map[string]stdlibFunc{
			"glob": {GoFunc: "__gopy_glob", GoImport: "path/filepath", Helper: helperGlob},
		},
	},
	"calendar": {
		Attrs: map[string]stdlibAttr{
			"month_name": {GoExpr: `[]string{"", "January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}`},
			"day_name":   {GoExpr: `[]string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}`},
			"month_abbr": {GoExpr: `[]string{"", "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}`},
			"day_abbr":   {GoExpr: `[]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}`},
		},
		Funcs: map[string]stdlibFunc{
			"isleap":     {GoFunc: "__gopy_cal_isleap", Helper: helperCalIsleap, RetKind: "bool"},
			"monthrange": {GoFunc: "__gopy_cal_monthrange", GoImport: "time", Helper: helperCalMonthrange},
			"weekday":    {GoFunc: "__gopy_cal_weekday", GoImport: "time", Helper: helperCalWeekday, RetKind: "int"},
		},
	},
	"socket": {
		Attrs: map[string]stdlibAttr{
			"AF_INET":      {GoExpr: "int64(2)"},
			"AF_INET6":     {GoExpr: "int64(10)"},
			"AF_UNIX":      {GoExpr: "int64(1)"},
			"SOCK_STREAM":  {GoExpr: "int64(1)"},
			"SOCK_DGRAM":   {GoExpr: "int64(2)"},
			"SOL_SOCKET":   {GoExpr: "int64(1)"},
			"SO_REUSEADDR": {GoExpr: "int64(2)"},
			"SO_KEEPALIVE": {GoExpr: "int64(9)"},
		},
		Funcs: map[string]stdlibFunc{
			"gethostname":   {GoFunc: "__gopy_socket_hostname", GoImport: "os", Helper: helperSocketHostname, RetKind: "str"},
			"getfqdn":       {GoFunc: "__gopy_socket_hostname", GoImport: "os", Helper: helperSocketHostname, RetKind: "str"},
			"gethostbyname": {GoFunc: "__gopy_socket_gethostbyname", Helper: helperSocketGethostbyname, HelperImports: []string{"net"}, RetKind: "str"},
			"gethostbyaddr": {GoFunc: "__gopy_socket_gethostbyaddr", Helper: helperSocketGethostbyaddr, HelperImports: []string{"net"}},
			"inet_aton":     {GoFunc: "__gopy_socket_inet_aton", Helper: helperSocketInetAton, HelperImports: []string{"net"}, RetKind: "str"},
			"inet_ntoa":     {GoFunc: "__gopy_socket_inet_ntoa", Helper: helperSocketInetNtoa, HelperImports: []string{"net"}, RetKind: "str"},
			"htons":         {GoFunc: "__gopy_socket_htons", Helper: helperSocketHtons, RetKind: "int"},
			"ntohs":         {GoFunc: "__gopy_socket_htons", Helper: helperSocketHtons, RetKind: "int"},
			"socket":        {GoFunc: "__gopy_socket_new", Helper: helperSocketNew, RetTag: "__Socket", ExtraHelpers: map[string]string{"__Socket": helperSocketType}, HelperImports: []string{"net", "fmt", "io"}},
			"create_connection": {GoFunc: "__gopy_socket_create_conn", Helper: helperSocketCreateConn, RetTag: "__Socket", ExtraHelpers: map[string]string{"__Socket": helperSocketType}, HelperImports: []string{"net", "fmt", "io"}},
		},
	},
	"platform": {
		Funcs: map[string]stdlibFunc{
			"system":         {GoFunc: "__gopy_platform_system", Helper: helperPlatformSystem, HelperImports: []string{"runtime", "strings"}, RetKind: "str"},
			"machine":        {GoFunc: "__gopy_platform_machine", Helper: helperPlatformMachine, HelperImports: []string{"runtime"}, RetKind: "str"},
			"node":           {GoFunc: "__gopy_socket_hostname", GoImport: "os", Helper: helperSocketHostname, RetKind: "str"},
			"release":        {GoFunc: "__gopy_platform_release", Helper: helperPlatformRelease, RetKind: "str"},
			"python_version": {GoFunc: "__gopy_platform_python_version", Helper: helperPlatformPythonVersion, RetKind: "str"},
			"platform":       {GoFunc: "__gopy_platform_platform", Helper: helperPlatformPlatform, HelperImports: []string{"runtime"}, RetKind: "str"},
		},
	},
	"dataclasses": {
		// asdict / astuple / replace dispatched per-class in transpile.go's
		// call() builders; the entries below are stubs so alias resolution
		// succeeds for the call expressions.
		Funcs: map[string]stdlibFunc{
			"asdict":  {GoFunc: "__gopy_asdict_unused"},
			"astuple": {GoFunc: "__gopy_astuple_unused"},
			"replace": {GoFunc: "__gopy_replace_unused"},
			"fields":  {GoFunc: "__gopy_fields_unused"},
		},
	},
	"hmac": {
		Funcs: map[string]stdlibFunc{
			"new":     {GoFunc: "__gopy_hmac_new", GoImport: "crypto/hmac", Helper: helperHmacNew, RetTag: "__Hmac", ExtraHelpers: map[string]string{"__Hmac": helperHmacType}, HelperImports: []string{"crypto/sha1", "crypto/sha256", "crypto/sha512", "crypto/md5", "hash", "encoding/hex"}},
			"compare_digest": {GoFunc: "__gopy_hmac_cmp", GoImport: "crypto/hmac", Helper: helperHmacCompare, RetKind: "bool"},
		},
	},
	"subprocess": {
		// run() needs to ignore Python kwargs (capture_output, text, ...)
		// that don't have a Go equivalent. Dispatch lives in transpile.go.
		Funcs: map[string]stdlibFunc{
			"run":          {GoFunc: "__gopy_subprocess_run_unused", RetTag: "__CompletedProcess"},
			"check_output": {GoFunc: "__gopy_subprocess_check_output", Helper: helperSubprocessCheckOutput, HelperImports: []string{"os/exec"}, RetKind: "str"},
			"check_call":   {GoFunc: "__gopy_subprocess_check_call", Helper: helperSubprocessCheckCall, HelperImports: []string{"os/exec"}, RetKind: "int"},
			"call":         {GoFunc: "__gopy_subprocess_call", Helper: helperSubprocessCall, HelperImports: []string{"os/exec"}, RetKind: "int"},
			"getoutput":    {GoFunc: "__gopy_subprocess_getoutput", Helper: helperSubprocessGetoutput, HelperImports: []string{"os/exec", "strings"}, RetKind: "str"},
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
			"wraps":           {GoFunc: "__gopy_wraps_unused"},
			"singledispatch":  {GoFunc: "__gopy_singledispatch_unused"},
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
			"getLogger":   {GoFunc: "__gopy_log_getlogger", GoImport: "fmt", Helper: helperLogGetLogger, RetTag: "__Logger", ExtraHelpers: map[string]string{"__Logger": helperLoggerType}, HelperImports: []string{"os"}},
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
			"merge":       {GoFunc: "__gopy_heapq_merge_unused"},
		},
	},
	"bisect": {
		Funcs: map[string]stdlibFunc{
			"bisect_left":  {GoFunc: "__gopy_bisect_left_unused"},
			"bisect_right": {GoFunc: "__gopy_bisect_right_unused"},
			"bisect":       {GoFunc: "__gopy_bisect_right_unused"},
			"insort":       {GoFunc: "__gopy_insort_unused"},
			"insort_left":  {GoFunc: "__gopy_insort_unused"},
			"insort_right": {GoFunc: "__gopy_insort_unused"},
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
			"zip_longest":  {GoFunc: "__gopy_zip_longest_unused"},
			"pairwise":     {GoFunc: "__gopy_pairwise_unused"},
			"batched":      {GoFunc: "__gopy_batched_unused"},
		},
	},
	"random": {
		Funcs: map[string]stdlibFunc{
			"random":         {GoFunc: "__gopy_random", GoImport: "math/rand", Helper: helperRandomFloat},
			"randint":        {GoFunc: "__gopy_randint", GoImport: "math/rand", Helper: helperRandint},
			"seed":           {GoFunc: "__gopy_random_seed", GoImport: "math/rand", Helper: helperRandomSeed},
			"uniform":        {GoFunc: "__gopy_random_uniform", GoImport: "math/rand", Helper: helperRandomUniform, RetKind: "float"},
			"gauss":          {GoFunc: "__gopy_random_gauss", GoImport: "math/rand", Helper: helperRandomGauss, RetKind: "float"},
			"normalvariate":  {GoFunc: "__gopy_random_gauss", GoImport: "math/rand", Helper: helperRandomGauss, RetKind: "float"},
			"expovariate":    {GoFunc: "__gopy_random_expo", GoImport: "math/rand", Helper: helperRandomExpo, RetKind: "float"},
			"triangular":     {GoFunc: "__gopy_random_triangular", GoImport: "math/rand", Helper: helperRandomTriangular, RetKind: "float"},
			"randrange":      {GoFunc: "__gopy_random_randrange", GoImport: "math/rand", Helper: helperRandomRandrange, RetKind: "int"},
			"getrandbits":    {GoFunc: "__gopy_random_getrandbits", GoImport: "math/rand", Helper: helperRandomGetrandbits, RetKind: "int"},
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
			"reader":     {GoFunc: "__gopy_csv_reader", GoImport: "encoding/csv", Helper: helperCSVReader, HelperImports: []string{"strings"}},
			"writer":     {GoFunc: "__gopy_csv_writer_new", GoImport: "encoding/csv", Helper: helperCSVWriterNew, RetTag: "__CSVWriter", ExtraHelpers: map[string]string{"__CSVWriter": helperCSVWriterType}},
			"DictReader": {GoFunc: "__gopy_csv_dictreader", GoImport: "encoding/csv", Helper: helperCSVDictReader, HelperImports: []string{"strings"}},
			"DictWriter": {GoFunc: "__gopy_csv_dictwriter_new", GoImport: "encoding/csv", Helper: helperCSVDictWriterNew, RetTag: "__CSVDictWriter", ExtraHelpers: map[string]string{"__CSVDictWriter": helperCSVDictWriterType}},
		},
	},
	"getpass": {
		Funcs: map[string]stdlibFunc{
			"getuser": {GoFunc: "__gopy_getpass_getuser", GoImport: "os", Helper: helperGetpassGetuser, RetKind: "str"},
		},
	},
	"typing": {
		Funcs: map[string]stdlibFunc{
			// typing.cast at runtime is the identity — dispatched in
			// transpile.go's call() so the second arg passes through.
			"cast":           {GoFunc: "__gopy_typing_cast_unused"},
			"get_type_hints": {GoFunc: "__gopy_typing_hints", Helper: helperTypingHints},
			"get_args":       {GoFunc: "__gopy_typing_args", Helper: helperTypingArgs},
			"get_origin":     {GoFunc: "__gopy_typing_origin", Helper: helperTypingOrigin},
			"TYPE_CHECKING":  {GoFunc: "__gopy_typing_typecheck_unused"},
		},
	},
	"threading": {
		Funcs: map[string]stdlibFunc{
			"Lock":  {GoFunc: "__gopy_threading_lock", Helper: helperThreadingLock, RetTag: "__Lock", ExtraHelpers: map[string]string{"__Lock": helperLockType}},
			"RLock": {GoFunc: "__gopy_threading_lock", Helper: helperThreadingLock, RetTag: "__Lock", ExtraHelpers: map[string]string{"__Lock": helperLockType}},
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
			// Subs entries below provide date.today / date.fromisoformat
			// as classmethods.
			"time":      {GoFunc: "__gopy_time_new", GoImport: "fmt", Helper: helperTimeNew, RetTag: "__Time", ExtraHelpers: map[string]string{"__Time": helperTimeType}},
		},
		Subs: map[string]stdlibModule{
			"date": {
				Funcs: map[string]stdlibFunc{
					"today":         {GoFunc: "__gopy_date_today", GoImport: "time", Helper: helperDateToday, RetTag: "__Date", ExtraHelpers: map[string]string{"__Date": helperDateType, "__gopy_py_time_format": helperPyTimeFormat, "__gopy_datetime_strftime": helperDatetimeStrftime}, HelperImports: []string{"time", "strings", "fmt"}},
					"fromisoformat": {GoFunc: "__gopy_date_fromiso", GoImport: "time", Helper: helperDateFromIso, RetTag: "__Date", ExtraHelpers: map[string]string{"__Date": helperDateType, "__gopy_py_time_format": helperPyTimeFormat, "__gopy_datetime_strftime": helperDatetimeStrftime}, HelperImports: []string{"time", "strings", "fmt"}},
				},
			},
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
					"combine":        {GoFunc: "__gopy_datetime_combine", GoImport: "time", Helper: helperDatetimeCombine, RetTag: "__Datetime", ExtraHelpers: map[string]string{"__Datetime": helperDatetimeType, "__Timedelta": helperTimedeltaType, "__Date": helperDateType, "__Time": helperTimeType, "__gopy_py_time_format": helperPyTimeFormat, "__gopy_datetime_strftime": helperDatetimeStrftime}, HelperImports: []string{"fmt", "strings"}},
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
	GoExpr        string
	GoImport      string
	Helper        string // optional package-level Go source (e.g. a var declaration) pulled in once
	HelperName    string // key for helpers map dedup; defaults to GoExpr when empty
	HelperImports []string
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

// helperTimeStrftime: minimal CPython strftime → Go time.Format mapping
// (%Y, %m, %d, %H, %M, %S, %y, %j, %A, %a, %B, %b, %p, %z, %Z, %%).
// Accepts time.struct_time-like 9-tuple ([]any) or skips it (uses now).
const helperTimeStrftime = `func __gopy_time_strftime(args ...any) string {
	if len(args) == 0 {
		return ""
	}
	fmtStr := ""
	switch s := args[0].(type) {
	case string:
		fmtStr = s
	}
	t := time.Now()
	if len(args) >= 2 {
		if tup, ok := args[1].([]any); ok && len(tup) >= 6 {
			yr, _ := tup[0].(int64)
			mo, _ := tup[1].(int64)
			day, _ := tup[2].(int64)
			hr, _ := tup[3].(int64)
			mn, _ := tup[4].(int64)
			sc, _ := tup[5].(int64)
			t = time.Date(int(yr), time.Month(int(mo)), int(day), int(hr), int(mn), int(sc), 0, time.UTC)
		}
	}
	repl := []struct{ from, to string }{
		{"%Y", "2006"}, {"%m", "01"}, {"%d", "02"},
		{"%H", "15"}, {"%M", "04"}, {"%S", "05"},
		{"%y", "06"}, {"%A", "Monday"}, {"%a", "Mon"},
		{"%B", "January"}, {"%b", "Jan"}, {"%p", "PM"},
		{"%z", "-0700"}, {"%Z", "MST"},
	}
	out := fmtStr
	for _, r := range repl {
		out = strings.ReplaceAll(out, r.from, t.Format(r.to))
	}
	out = strings.ReplaceAll(out, "%%", "%")
	return out
}`

// helperTimeLocaltime / Gmtime emit a 9-tuple analog matching CPython's
// time.struct_time field order: (year, month, day, hour, minute, second,
// weekday, yearday, isdst). All fields are int64.
const helperTimeLocaltime = `func __gopy_time_localtime(args ...float64) []any {
	var t time.Time
	if len(args) > 0 {
		t = time.Unix(int64(args[0]), 0).Local()
	} else {
		t = time.Now().Local()
	}
	return []any{int64(t.Year()), int64(t.Month()), int64(t.Day()), int64(t.Hour()), int64(t.Minute()), int64(t.Second()), int64((int(t.Weekday()) + 6) % 7), int64(t.YearDay()), int64(-1)}
}`

const helperTimeGmtime = `func __gopy_time_gmtime(args ...float64) []any {
	var t time.Time
	if len(args) > 0 {
		t = time.Unix(int64(args[0]), 0).UTC()
	} else {
		t = time.Now().UTC()
	}
	return []any{int64(t.Year()), int64(t.Month()), int64(t.Day()), int64(t.Hour()), int64(t.Minute()), int64(t.Second()), int64((int(t.Weekday()) + 6) % 7), int64(t.YearDay()), int64(0)}
}`

const helperTimeMktime = `func __gopy_time_mktime(tup []any) float64 {
	if len(tup) < 6 {
		return 0
	}
	yr, _ := tup[0].(int64)
	mo, _ := tup[1].(int64)
	day, _ := tup[2].(int64)
	hr, _ := tup[3].(int64)
	mn, _ := tup[4].(int64)
	sc, _ := tup[5].(int64)
	t := time.Date(int(yr), time.Month(int(mo)), int(day), int(hr), int(mn), int(sc), 0, time.Local)
	return float64(t.Unix())
}`

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
}

func (p *__Pattern) Subn(repl, s string) []any {
	count := int64(len(p.r.FindAllStringIndex(s, -1)))
	return []any{p.r.ReplaceAllString(s, repl), count}
}

func (p *__Pattern) Split(s string) []string {
	out := p.r.Split(s, -1)
	if out == nil {
		return []string{}
	}
	return out
}

func (p *__Pattern) Fullmatch(s string) *__Match {
	anchored := regexp.MustCompile("^(?:" + p.r.String() + ")$")
	return __gopy_match_build(anchored, s, false)
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

// helperCSVWriterType bridges Python's csv.writer to Go's encoding/csv.
// Wraps a *csv.Writer bound to the destination io.Writer so writerow /
// writerows can stream rows like CPython does.
const helperCSVWriterType = `type __CSVWriter struct{ w *csv.Writer }

func (w *__CSVWriter) Writerow(row []string) {
	if err := w.w.Write(row); err != nil {
		panic(err)
	}
	w.w.Flush()
}

func (w *__CSVWriter) Writerows(rows [][]string) {
	for _, r := range rows {
		if err := w.w.Write(r); err != nil {
			panic(err)
		}
	}
	w.w.Flush()
}`

const helperCSVWriterNew = `func __gopy_csv_writer_new(fh interface{ Write([]byte) (int, error) }) *__CSVWriter {
	return &__CSVWriter{w: csv.NewWriter(fh)}
}`

// helperCSVDictWriterType bridges csv.DictWriter. Caller supplies the
// fieldnames list at construction; writerow accepts a map and emits the
// columns in fieldname order.
const helperCSVDictWriterType = `type __CSVDictWriter struct {
	w      *csv.Writer
	fields []string
}

func (w *__CSVDictWriter) Writeheader() {
	if err := w.w.Write(w.fields); err != nil {
		panic(err)
	}
	w.w.Flush()
}

func (w *__CSVDictWriter) Writerow(row map[string]string) {
	rec := make([]string, len(w.fields))
	for i, f := range w.fields {
		rec[i] = row[f]
	}
	if err := w.w.Write(rec); err != nil {
		panic(err)
	}
	w.w.Flush()
}

func (w *__CSVDictWriter) Writerows(rows []map[string]string) {
	for _, r := range rows {
		w.Writerow(r)
	}
}`

const helperCSVDictWriterNew = `func __gopy_csv_dictwriter_new(fh interface{ Write([]byte) (int, error) }, fields []string) *__CSVDictWriter {
	return &__CSVDictWriter{w: csv.NewWriter(fh), fields: fields}
}`

const helperGetpassGetuser = `func __gopy_getpass_getuser() string {
	for _, k := range []string{"LOGNAME", "USER", "USERNAME"} {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}`

// helperLockType is a no-op stand-in for threading.Lock / RLock. The
// transpiled program is single-goroutine by default, so acquire/release
// degrade to bookkeeping. Context-manager use (`with lock:`) works too
// because Acquire returns the lock itself.
const helperLockType = `type __Lock struct{ held bool }

func (l *__Lock) Acquire(args ...any) bool { l.held = true; return true }
func (l *__Lock) Release()                 { l.held = false }
func (l *__Lock) Locked() bool             { return l.held }
func (l *__Lock) Enter() *__Lock           { l.held = true; return l }
func (l *__Lock) Exit() bool               { l.held = false; return false }`

const helperThreadingLock = `func __gopy_threading_lock() *__Lock { return &__Lock{} }`

// helperQueueType is a minimal FIFO/LIFO container modeled on
// queue.Queue. Goroutine-safe via the embedded mutex. Empty() / Qsize()
// mirror Python's introspection; Get() panics on empty (Python blocks
// instead — gopy doesn't have a blocking channel-of-any backing it yet).
const helperQueueType = `type __Queue struct {
	mu    sync.Mutex
	items []any
	lifo  bool
}

func (q *__Queue) Put(v any) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, v)
}

func (q *__Queue) Get() any {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		panic(NewException("queue.Empty"))
	}
	if q.lifo {
		v := q.items[len(q.items)-1]
		q.items = q.items[:len(q.items)-1]
		return v
	}
	v := q.items[0]
	q.items = q.items[1:]
	return v
}

func (q *__Queue) Qsize() int64 {
	q.mu.Lock()
	defer q.mu.Unlock()
	return int64(len(q.items))
}

func (q *__Queue) Empty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items) == 0
}

func (q *__Queue) Full() bool { return false }`

const helperQueueNew = `func __gopy_queue_new(args ...int64) *__Queue { return &__Queue{} }`

const helperLifoQueueNew = `func __gopy_lifo_queue_new(args ...int64) *__Queue { return &__Queue{lifo: true} }`

// helperCSVDictReader returns []map[string]string for each data row using
// the first row as column headers. Mirrors csv.DictReader's most common
// shape.
const helperCSVDictReader = `func __gopy_csv_dictreader(lines []string) []map[string]string {
	r := csv.NewReader(strings.NewReader(strings.Join(lines, "\n")))
	rows, err := r.ReadAll()
	if err != nil {
		panic(err)
	}
	if len(rows) == 0 {
		return []map[string]string{}
	}
	header := rows[0]
	out := make([]map[string]string, 0, len(rows)-1)
	for _, r := range rows[1:] {
		m := map[string]string{}
		for i, h := range header {
			if i < len(r) {
				m[h] = r[i]
			}
		}
		out = append(out, m)
	}
	return out
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

// helperHashlibNew dispatches by algorithm name ("sha256", "md5", etc.).
// Optional second argument is the initial data fed to update().
const helperHashlibNew = `func __gopy_hashlib_new(args ...string) *__Hasher {
	if len(args) == 0 {
		panic(NewException("ValueError: hashlib.new() requires an algorithm name"))
	}
	algo := args[0]
	var data []byte
	if len(args) > 1 {
		data = []byte(args[1])
	}
	return &__Hasher{data: data, algo: algo}
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

const helperSecretsRandbelow = `func __gopy_secrets_randbelow(n int64) int64 {
	if n <= 0 {
		panic(NewException("ValueError: randbelow needs a positive n"))
	}
	r, err := rand.Int(rand.Reader, big.NewInt(n))
	if err != nil {
		panic(err)
	}
	return r.Int64()
}`

// helperCopyShallow / helperCopyDeep route through encoding/json for a
// portable, type-erased clone. Shallow and deep are identical for the
// JSON-friendly value shapes gopy emits (slices, maps, primitives,
// struct-with-fields-marshaled-by-name); deeper graphs aren't covered.
const helperCopyShallow = `func __gopy_copy_shallow(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		panic(NewException("copy: " + err.Error()))
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		panic(NewException("copy: " + err.Error()))
	}
	return out
}`

const helperCopyDeep = `func __gopy_copy_deep(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		panic(NewException("deepcopy: " + err.Error()))
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		panic(NewException("deepcopy: " + err.Error()))
	}
	return out
}`

const helperHTMLEscape = `func __gopy_html_escape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;", "'", "&#x27;")
	return r.Replace(s)
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

// helperHTTPResponseType is the gopy-side wrapper around a captured HTTP
// response body. .read() returns the full body as a string (matching
// CPython's bytes-as-str pass-through), .status holds the HTTP code,
// and .headers is a map[string]string keyed by canonical header name.
const helperHTTPResponseType = `type __HTTPResponse struct {
	body    string
	consumed bool
	Status  int64
	Headers map[string]string
}

func (r *__HTTPResponse) Read() string {
	if r.consumed {
		return ""
	}
	r.consumed = true
	return r.body
}

func (r *__HTTPResponse) Close() {}

func (r *__HTTPResponse) Getcode() int64 { return r.Status }`

// helperURLRequestType — request builder used as `urlopen(Request(...))`
// argument or passed directly to http.client. Captures method, headers,
// data; urlopen() now accepts either a str URL or a *__URLRequest.
const helperURLRequestType = `type __URLRequest struct {
	URL     string
	Method  string
	Data    string
	Headers map[string]string
}

func (r *__URLRequest) Add_header(k, v string) {
	if r.Headers == nil {
		r.Headers = map[string]string{}
	}
	r.Headers[k] = v
}`

const helperURLRequestNew = `func __gopy_url_request_new(args ...any) *__URLRequest {
	r := &__URLRequest{Method: "GET", Headers: map[string]string{}}
	if len(args) > 0 {
		r.URL, _ = args[0].(string)
	}
	if len(args) > 1 {
		r.Data, _ = args[1].(string)
		if r.Data != "" {
			r.Method = "POST"
		}
	}
	return r
}`

const helperURLRetrieve = `func __gopy_url_urlretrieve(args ...any) []any {
	if len(args) == 0 {
		return []any{"", map[string]string{}}
	}
	url, _ := args[0].(string)
	dest := ""
	if len(args) > 1 {
		dest, _ = args[1].(string)
	}
	resp, err := http.Get(url)
	if err != nil {
		panic(NewException("URLError: " + err.Error()))
	}
	defer resp.Body.Close()
	if dest == "" {
		f, e := os.CreateTemp("", "urlretrieve-*")
		if e != nil {
			panic(NewException("URLError: " + e.Error()))
		}
		dest = f.Name()
		f.Close()
	}
	out, err := os.Create(dest)
	if err != nil {
		panic(NewException("URLError: " + err.Error()))
	}
	defer out.Close()
	io.Copy(out, resp.Body)
	return []any{dest, map[string]string{}}
}`

const helperURLOpen = `func __gopy_url_urlopen(url string) *__HTTPResponse {
	resp, err := http.Get(url)
	if err != nil {
		panic(NewException("URLError: " + err.Error()))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(NewException("URLError: " + err.Error()))
	}
	headers := map[string]string{}
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	return &__HTTPResponse{body: string(body), Status: int64(resp.StatusCode), Headers: headers}
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

// helperMimetypesGuess mirrors Python's mimetypes.guess_type which
// returns (type, encoding) where encoding is typically None / "" for
// most file types. gopy emits a 2-elt slice; encoding is always "".
const helperMimetypesGuess = `func __gopy_mimetypes_guess(args ...any) []any {
	if len(args) == 0 {
		return []any{"", ""}
	}
	name, _ := args[0].(string)
	ext := filepath.Ext(name)
	if ext == "" {
		return []any{"", ""}
	}
	t := mime.TypeByExtension(ext)
	if t == "" {
		return []any{"", ""}
	}
	// Strip charset suffix; CPython's guess_type returns the bare type.
	for i := 0; i < len(t); i++ {
		if t[i] == ';' {
			t = t[:i]
			break
		}
	}
	return []any{t, ""}
}`

const helperMimetypesGuessExt = `func __gopy_mimetypes_guess_ext(args ...any) string {
	if len(args) == 0 {
		return ""
	}
	typ, _ := args[0].(string)
	exts, err := mime.ExtensionsByType(typ)
	if err != nil || len(exts) == 0 {
		return ""
	}
	return exts[0]
}`

const helperURLUrljoin = `func __gopy_url_urljoin(base, ref string) string {
	b, err := url.Parse(base)
	if err != nil {
		return ref
	}
	r, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return b.ResolveReference(r).String()
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

const helperLoggerType = `type __Logger struct{ name string }

func (l *__Logger) Debug(msg string)    { fmt.Fprintln(os.Stderr, "DEBUG:"+l.name+":"+msg) }
func (l *__Logger) Info(msg string)     { fmt.Fprintln(os.Stderr, "INFO:"+l.name+":"+msg) }
func (l *__Logger) Warning(msg string)  { fmt.Fprintln(os.Stderr, "WARNING:"+l.name+":"+msg) }
func (l *__Logger) Error(msg string)    { fmt.Fprintln(os.Stderr, "ERROR:"+l.name+":"+msg) }
func (l *__Logger) Critical(msg string) { fmt.Fprintln(os.Stderr, "CRITICAL:"+l.name+":"+msg) }`

const helperLogGetLogger = `func __gopy_log_getlogger(args ...string) *__Logger {
	name := "root"
	if len(args) > 0 && args[0] != "" {
		name = args[0]
	}
	return &__Logger{name: name}
}`

const helperGzipCompress = `func __gopy_gzip_compress(s string) string {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	if _, err := w.Write([]byte(s)); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	return b.String()
}`

const helperGzipDecompress = `func __gopy_gzip_decompress(s string) string {
	r, err := gzip.NewReader(bytes.NewReader([]byte(s)))
	if err != nil {
		panic(err)
	}
	defer r.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return string(out)
}`

const helperZlibCompress = `func __gopy_zlib_compress(s string) string {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	if _, err := w.Write([]byte(s)); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	return b.String()
}`

const helperZlibDecompress = `func __gopy_zlib_decompress(s string) string {
	r, err := zlib.NewReader(bytes.NewReader([]byte(s)))
	if err != nil {
		panic(err)
	}
	defer r.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}
	return string(out)
}`

const helperZlibCrc32 = `func __gopy_zlib_crc32(s string) int64 {
	return int64(crc32.ChecksumIEEE([]byte(s)))
}`

const helperZlibAdler32 = `func __gopy_zlib_adler32(s string) int64 {
	return int64(adler32.Checksum([]byte(s)))
}`

const helperSocketHostname = `func __gopy_socket_hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return ""
	}
	return h
}`

const helperSocketGethostbyname = `func __gopy_socket_gethostbyname(host string) string {
	ips, err := net.LookupHost(host)
	if err != nil || len(ips) == 0 {
		panic(NewException("gaierror: " + host))
	}
	return ips[0]
}`

// helperSocketInetAton converts dotted-quad → 4-byte packed string.
// CPython returns bytes; gopy uses str-as-bytes pass-through.
const helperSocketInetAton = `func __gopy_socket_inet_aton(addr string) string {
	ip := net.ParseIP(addr)
	if ip == nil {
		panic(NewException("OSError: illegal IP address string passed to inet_aton"))
	}
	v4 := ip.To4()
	if v4 == nil {
		panic(NewException("OSError: not an IPv4 address"))
	}
	return string(v4)
}`

const helperSocketInetNtoa = `func __gopy_socket_inet_ntoa(packed string) string {
	if len(packed) != 4 {
		panic(NewException("OSError: packed IP wrong length for inet_ntoa"))
	}
	return net.IPv4(packed[0], packed[1], packed[2], packed[3]).String()
}`

const helperSocketHtons = `func __gopy_socket_htons(n int64) int64 { return n & 0xffff }`

// helperSocketGethostbyaddr returns a 3-tuple analog (hostname, aliases, ips)
// where aliases is always empty (Go's net.LookupAddr returns no aliases).
const helperSocketGethostbyaddr = `func __gopy_socket_gethostbyaddr(addr string) []any {
	names, err := net.LookupAddr(addr)
	if err != nil || len(names) == 0 {
		panic(NewException("herror: " + addr))
	}
	host := names[0]
	if len(host) > 0 && host[len(host)-1] == '.' {
		host = host[:len(host)-1]
	}
	aliases := []string{}
	ips := []string{addr}
	return []any{host, aliases, ips}
}`

// helperSocketType is a minimal TCP-only socket wrapper. UDP / Unix
// streams aren't supported. .Connect((host, port)) dials; .Send / .Recv
// move bytes as strings (gopy's bytes shim). Server side covers
// .Bind, .Listen, .Accept (returns a connected __Socket).
const helperSocketType = `type __Socket struct {
	conn     net.Conn
	listener net.Listener
	bindAddr string
}

func (s *__Socket) Connect(addr []any) {
	if len(addr) != 2 {
		panic(NewException("socket.connect: expected (host, port)"))
	}
	host := fmt.Sprintf("%v", addr[0])
	port := fmt.Sprintf("%v", addr[1])
	c, err := net.Dial("tcp", host+":"+port)
	if err != nil {
		panic(NewException("ConnectionRefusedError: " + err.Error()))
	}
	s.conn = c
}

func (s *__Socket) Bind(addr []any) {
	if len(addr) != 2 {
		panic(NewException("socket.bind: expected (host, port)"))
	}
	host := fmt.Sprintf("%v", addr[0])
	port := fmt.Sprintf("%v", addr[1])
	s.bindAddr = host + ":" + port
}

func (s *__Socket) Listen(args ...int64) {
	if s.bindAddr == "" {
		panic(NewException("socket.listen: must bind() first"))
	}
	l, err := net.Listen("tcp", s.bindAddr)
	if err != nil {
		panic(NewException("OSError: " + err.Error()))
	}
	s.listener = l
}

func (s *__Socket) Accept() []any {
	if s.listener == nil {
		panic(NewException("socket.accept: must listen() first"))
	}
	c, err := s.listener.Accept()
	if err != nil {
		panic(NewException("OSError: " + err.Error()))
	}
	client := &__Socket{conn: c}
	remote := []any{c.RemoteAddr().String(), int64(0)}
	return []any{client, remote}
}

func (s *__Socket) Send(data string) int64 {
	if s.conn == nil {
		panic(NewException("socket.send: not connected"))
	}
	n, err := s.conn.Write([]byte(data))
	if err != nil {
		panic(NewException("OSError: " + err.Error()))
	}
	return int64(n)
}

func (s *__Socket) Sendall(data string) {
	s.Send(data)
}

func (s *__Socket) Recv(n int64) string {
	if s.conn == nil {
		panic(NewException("socket.recv: not connected"))
	}
	buf := make([]byte, n)
	read, err := s.conn.Read(buf)
	if err != nil && err != io.EOF {
		panic(NewException("OSError: " + err.Error()))
	}
	return string(buf[:read])
}

func (s *__Socket) Close() {
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	if s.listener != nil {
		s.listener.Close()
		s.listener = nil
	}
}

func (s *__Socket) Setsockopt(args ...any) {}
func (s *__Socket) Settimeout(args ...any) {}

func (s *__Socket) Enter() *__Socket { return s }
func (s *__Socket) Exit(_, _, _ any) { s.Close() }`

const helperSocketNew = `func __gopy_socket_new(args ...int64) *__Socket {
	// Family / type are accepted but only TCP/INET combos do anything
	// at Connect() time. Return a fresh, disconnected socket.
	return &__Socket{}
}`

const helperSocketCreateConn = `func __gopy_socket_create_conn(addr []any, args ...int64) *__Socket {
	s := &__Socket{}
	s.Connect(addr)
	return s
}`

const helperPlatformSystem = `func __gopy_platform_system() string {
	switch runtime.GOOS {
	case "darwin":
		return "Darwin"
	case "linux":
		return "Linux"
	case "windows":
		return "Windows"
	case "freebsd":
		return "FreeBSD"
	case "openbsd":
		return "OpenBSD"
	case "netbsd":
		return "NetBSD"
	}
	return strings.Title(runtime.GOOS)
}`

const helperPlatformMachine = `func __gopy_platform_machine() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "386":
		return "i686"
	case "arm64":
		return "aarch64"
	case "arm":
		return "armv7l"
	}
	return runtime.GOARCH
}`

const helperPlatformRelease = `func __gopy_platform_release() string { return "" }`

const helperPlatformPythonVersion = `func __gopy_platform_python_version() string { return "3.12.0" }`

const helperPlatformPlatform = `func __gopy_platform_platform() string {
	return runtime.GOOS + "-" + runtime.GOARCH
}`


const helperCalIsleap = `func __gopy_cal_isleap(y int64) bool {
	return (y%4 == 0 && y%100 != 0) || y%400 == 0
}`

const helperCalMonthrange = `func __gopy_cal_monthrange(year, month int64) []int64 {
	first := time.Date(int(year), time.Month(int(month)), 1, 0, 0, 0, 0, time.UTC)
	startWeekday := int64(first.Weekday())
	// Python: Monday=0..Sunday=6. Go: Sunday=0..Saturday=6.
	startWeekday = (startWeekday + 6) % 7
	next := first.AddDate(0, 1, 0)
	days := int64(next.Sub(first) / (24 * time.Hour))
	return []int64{startWeekday, days}
}`

const helperCalWeekday = `func __gopy_cal_weekday(year, month, day int64) int64 {
	t := time.Date(int(year), time.Month(int(month)), int(day), 0, 0, 0, 0, time.UTC)
	w := int64(t.Weekday())
	return (w + 6) % 7
}`

const helperGlob = `func __gopy_glob(pattern string) []string {
	out, err := filepath.Glob(pattern)
	if err != nil {
		return []string{}
	}
	if out == nil {
		return []string{}
	}
	return out
}`

const helperShutilRmtree = `func __gopy_shutil_rmtree(p string) {
	if err := os.RemoveAll(p); err != nil {
		panic(err)
	}
}`

const helperShutilCopy = `func __gopy_shutil_copy(src, dst string) string {
	in, err := os.Open(src)
	if err != nil {
		panic(err)
	}
	defer in.Close()
	si, err := os.Stat(dst)
	if err == nil && si.IsDir() {
		base := src
		for i := len(src) - 1; i >= 0; i-- {
			if src[i] == '/' {
				base = src[i+1:]
				break
			}
		}
		if len(dst) > 0 && dst[len(dst)-1] != '/' {
			dst = dst + "/" + base
		} else {
			dst = dst + base
		}
	}
	out, err := os.Create(dst)
	if err != nil {
		panic(err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		panic(err)
	}
	return dst
}`

const helperShutilMove = `func __gopy_shutil_move(src, dst string) string {
	if err := os.Rename(src, dst); err != nil {
		panic(err)
	}
	return dst
}`

const helperShutilWhich = `func __gopy_shutil_which(cmd string) string {
	p, err := exec.LookPath(cmd)
	if err != nil {
		return ""
	}
	return p
}`

// helperWeakrefRef is a pass-through: gopy doesn't model weak refs.
// Returning a closure matches CPython's API shape where the result
// is callable and yields the wrapped object on invocation.
const helperWeakrefRef = `func __gopy_weakref_ref(obj any) func() any {
	return func() any { return obj }
}`

// helperIOStringIOType — minimal StringIO/BytesIO: gopy uses string
// for both so a single backing type works. Writes happen at the
// current position, overlaying existing content (matching CPython's
// io.StringIO semantics where the cursor advances after write/read).
const helperIOStringIOType = `type __StringIO struct {
	data []byte
	pos  int
}

func (s *__StringIO) Write(data string) int64 {
	bs := []byte(data)
	end := s.pos + len(bs)
	if end > len(s.data) {
		s.data = append(s.data, make([]byte, end-len(s.data))...)
	}
	copy(s.data[s.pos:end], bs)
	s.pos = end
	return int64(len(bs))
}

func (s *__StringIO) Getvalue() string { return string(s.data) }

func (s *__StringIO) Read(args ...int64) string {
	if s.pos >= len(s.data) {
		return ""
	}
	rest := s.data[s.pos:]
	if len(args) > 0 && args[0] >= 0 && int(args[0]) < len(rest) {
		rest = rest[:args[0]]
	}
	s.pos += len(rest)
	return string(rest)
}

func (s *__StringIO) Seek(args ...int64) int64 {
	if len(args) > 0 {
		s.pos = int(args[0])
	}
	return int64(s.pos)
}

func (s *__StringIO) Tell() int64 { return int64(s.pos) }

func (s *__StringIO) Truncate(args ...int64) int64 {
	n := len(s.data)
	if len(args) > 0 {
		n = int(args[0])
	}
	if n < len(s.data) {
		s.data = s.data[:n]
	}
	return int64(n)
}

func (s *__StringIO) Close() {}

func (s *__StringIO) Enter() *__StringIO { return s }
func (s *__StringIO) Exit(_, _, _ any)   { s.Close() }`

const helperIOStringIONew = `func __gopy_io_stringio_new(args ...string) *__StringIO {
	s := &__StringIO{}
	if len(args) > 0 {
		s.data = []byte(args[0])
	}
	return s
}`

const helperIOBytesIONew = `func __gopy_io_bytesio_new(args ...string) *__StringIO {
	s := &__StringIO{}
	if len(args) > 0 {
		s.data = []byte(args[0])
	}
	return s
}`

// helperTypingHints / Args / Origin — gopy doesn't carry runtime type
// info, so reflection-style queries return empty results. Shape-
// compatible for libraries that just probe and fall back.
const helperTypingHints = `func __gopy_typing_hints(args ...any) map[string]any {
	return map[string]any{}
}`

const helperTypingArgs = `func __gopy_typing_args(args ...any) []any {
	return []any{}
}`

const helperTypingOrigin = `func __gopy_typing_origin(args ...any) any {
	return nil
}`

// helperArgparseType — minimal ArgumentParser. add_argument records
// flag specs; parse_args walks os.Args[1:] applying defaults, parsing
// `--key=value` and `--key value`, plus positional args in order.
// parse_args() returns a *__ArgNamespace; access via .Get(name).
// CPython's attribute access (`args.name`) is approximated by .Get;
// gopy users typically write `args.get("name")`.
const helperArgparseType = `type __ArgSpec struct {
	Name    string
	Short   string
	IsFlag  bool
	Default any
	Action  string
	IsPos   bool
}

type __ArgParser struct {
	Specs []__ArgSpec
}

type __ArgNamespace struct {
	Values map[string]any
}

func (n *__ArgNamespace) Get(name string) any { return n.Values[name] }

func (p *__ArgParser) AddArgument(args ...any) {
	if len(args) == 0 {
		return
	}
	spec := __ArgSpec{}
	first, _ := args[0].(string)
	if strings.HasPrefix(first, "--") {
		spec.Name = first[2:]
	} else if strings.HasPrefix(first, "-") {
		spec.Short = first[1:]
		if len(args) > 1 {
			if alt, ok := args[1].(string); ok && strings.HasPrefix(alt, "--") {
				spec.Name = alt[2:]
			}
		}
	} else {
		spec.IsPos = true
		spec.Name = first
	}
	if spec.Name == "" && spec.Short != "" {
		spec.Name = spec.Short
	}
	p.Specs = append(p.Specs, spec)
}

func (p *__ArgParser) ParseArgs(args ...any) *__ArgNamespace {
	var argv []string
	if len(args) > 0 {
		switch v := args[0].(type) {
		case []string:
			argv = v
		case []any:
			for _, x := range v {
				argv = append(argv, fmt.Sprintf("%v", x))
			}
		}
	} else {
		argv = os.Args[1:]
	}
	ns := &__ArgNamespace{Values: map[string]any{}}
	for _, s := range p.Specs {
		if s.Default != nil {
			ns.Values[s.Name] = s.Default
		} else if s.IsFlag {
			ns.Values[s.Name] = false
		} else {
			ns.Values[s.Name] = ""
		}
	}
	posIdx := 0
	posSpecs := []__ArgSpec{}
	for _, s := range p.Specs {
		if s.IsPos {
			posSpecs = append(posSpecs, s)
		}
	}
	i := 0
	for i < len(argv) {
		tok := argv[i]
		if strings.HasPrefix(tok, "--") {
			eq := strings.Index(tok, "=")
			var name, val string
			if eq >= 0 {
				name = tok[2:eq]
				val = tok[eq+1:]
			} else {
				name = tok[2:]
				if i+1 < len(argv) {
					val = argv[i+1]
					i++
				}
			}
			if v, err := strconv.ParseInt(val, 10, 64); err == nil {
				ns.Values[name] = v
			} else {
				ns.Values[name] = val
			}
		} else if strings.HasPrefix(tok, "-") && len(tok) >= 2 {
			short := tok[1:]
			for _, s := range p.Specs {
				if s.Short == short {
					if i+1 < len(argv) {
						ns.Values[s.Name] = argv[i+1]
						i++
					}
					break
				}
			}
		} else if posIdx < len(posSpecs) {
			ns.Values[posSpecs[posIdx].Name] = tok
			posIdx++
		}
		i++
	}
	return ns
}`

const helperArgparseNew = `func __gopy_argparse_new(args ...any) *__ArgParser { return &__ArgParser{} }`

// helperConfigParserType — minimal INI parser. .read(path) loads a
// file into nested maps; .get(section, key) returns the value.
// .sections() / .has_section / .has_option handle membership queries.
const helperConfigParserType = `type __ConfigParser struct {
	data map[string]map[string]string
}

func (p *__ConfigParser) ensure() {
	if p.data == nil {
		p.data = map[string]map[string]string{}
	}
}

func (p *__ConfigParser) Read(path string) []string {
	p.ensure()
	f, err := os.Open(path)
	if err != nil {
		return []string{}
	}
	defer f.Close()
	section := "DEFAULT"
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line[1 : len(line)-1]
			if _, ok := p.data[section]; !ok {
				p.data[section] = map[string]string{}
			}
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}
		k := strings.TrimSpace(line[:eq])
		v := strings.TrimSpace(line[eq+1:])
		if _, ok := p.data[section]; !ok {
			p.data[section] = map[string]string{}
		}
		p.data[section][k] = v
	}
	return []string{path}
}

func (p *__ConfigParser) Get(section, key string) string {
	if s, ok := p.data[section]; ok {
		return s[key]
	}
	return ""
}

func (p *__ConfigParser) Sections() []string {
	out := []string{}
	for k := range p.data {
		if k != "DEFAULT" {
			out = append(out, k)
		}
	}
	return out
}

func (p *__ConfigParser) Has_section(s string) bool {
	_, ok := p.data[s]
	return ok && s != "DEFAULT"
}

func (p *__ConfigParser) Has_option(section, key string) bool {
	if s, ok := p.data[section]; ok {
		_, k := s[key]
		return k
	}
	return false
}`

const helperConfigParserNew = `func __gopy_configparser_new(args ...any) *__ConfigParser {
	return &__ConfigParser{data: map[string]map[string]string{}}
}`

// helperEmailFormatdate — emit an RFC 2822 date string from a Unix
// timestamp (or current time when no arg given).
const helperEmailFormatdate = `func __gopy_email_formatdate(args ...float64) string {
	var t time.Time
	if len(args) > 0 {
		t = time.Unix(int64(args[0]), 0).UTC()
	} else {
		t = time.Now().UTC()
	}
	return t.Format("Mon, 02 Jan 2006 15:04:05 -0700")
}`

const helperEmailFormatDatetime = `func __gopy_email_format_datetime(args ...any) string {
	return time.Now().UTC().Format("Mon, 02 Jan 2006 15:04:05 -0700")
}`

// helperPickleDumps / Loads — gopy uses encoding/json as a portable
// stand-in for Python's pickle. Round-trips primitives, lists, dicts,
// and JSON-friendly nested structures. Class instances aren't
// auto-serialized. Output is text JSON rather than pickle's binary
// protocol — incompatible at the wire format but functionally usable
// for cross-process state passing within gopy programs.
// helperXMLElementType — minimal Element tree node. Holds Tag, Text,
// Attrib (attr map), and Children. .find(tag) / .findall(tag) walk
// direct children; .iter(tag) walks the whole subtree.
const helperXMLElementType = `type __XMLElement struct {
	Tag      string
	Text     string
	Attrib   map[string]string
	Children []*__XMLElement
}

func (e *__XMLElement) Find(tag string) *__XMLElement {
	for _, c := range e.Children {
		if c.Tag == tag {
			return c
		}
	}
	return nil
}

func (e *__XMLElement) Findall(tag string) []*__XMLElement {
	out := []*__XMLElement{}
	for _, c := range e.Children {
		if c.Tag == tag {
			out = append(out, c)
		}
	}
	return out
}

func (e *__XMLElement) Iter(args ...string) []*__XMLElement {
	want := ""
	if len(args) > 0 {
		want = args[0]
	}
	var out []*__XMLElement
	var walk func(n *__XMLElement)
	walk = func(n *__XMLElement) {
		if want == "" || n.Tag == want {
			out = append(out, n)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(e)
	return out
}

func (e *__XMLElement) Get(key string, args ...string) string {
	if v, ok := e.Attrib[key]; ok {
		return v
	}
	if len(args) > 0 {
		return args[0]
	}
	return ""
}

func __gopy_xml_build(d *xml.Decoder, start *xml.StartElement) *__XMLElement {
	el := &__XMLElement{Tag: start.Name.Local, Attrib: map[string]string{}}
	for _, a := range start.Attr {
		el.Attrib[a.Name.Local] = a.Value
	}
	for {
		tok, err := d.Token()
		if err != nil {
			return el
		}
		switch t := tok.(type) {
		case xml.StartElement:
			el.Children = append(el.Children, __gopy_xml_build(d, &t))
		case xml.EndElement:
			return el
		case xml.CharData:
			el.Text += string(t)
		}
	}
}`

const helperXMLFromstring = `func __gopy_xml_fromstring(src string) *__XMLElement {
	d := xml.NewDecoder(strings.NewReader(src))
	for {
		tok, err := d.Token()
		if err != nil {
			return nil
		}
		if se, ok := tok.(xml.StartElement); ok {
			return __gopy_xml_build(d, &se)
		}
	}
}`

// helperHTTPClientType — minimal http.client connection. Stores
// host+scheme. .request(method, path, body, headers) sends; .getresponse()
// returns an __HTTPResponse pre-loaded with body bytes + status.
const helperHTTPClientType = `type __HTTPClient struct {
	host   string
	scheme string
	last   *__HTTPResponse
}

func (c *__HTTPClient) Request(args ...any) {
	if len(args) < 2 {
		return
	}
	method, _ := args[0].(string)
	pathArg, _ := args[1].(string)
	body := ""
	if len(args) >= 3 {
		body, _ = args[2].(string)
	}
	headers := map[string]string{}
	if len(args) >= 4 {
		if m, ok := args[3].(map[string]string); ok {
			headers = m
		}
	}
	url := c.scheme + "://" + c.host + pathArg
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	req, err := http.NewRequest(strings.ToUpper(method), url, bodyReader)
	if err != nil {
		panic(NewException("HTTPException: " + err.Error()))
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(NewException("HTTPException: " + err.Error()))
	}
	defer resp.Body.Close()
	bs, _ := io.ReadAll(resp.Body)
	hs := map[string]string{}
	for k, v := range resp.Header {
		if len(v) > 0 {
			hs[k] = v[0]
		}
	}
	c.last = &__HTTPResponse{body: string(bs), Status: int64(resp.StatusCode), Headers: hs}
}

func (c *__HTTPClient) Getresponse() *__HTTPResponse { return c.last }
func (c *__HTTPClient) Close()                       {}`

const helperHTTPClientNew = `func __gopy_http_client_new(host string, args ...any) *__HTTPClient {
	return &__HTTPClient{host: host, scheme: "https"}
}`

const helperHTTPClientNewPlain = `func __gopy_http_client_new_plain(host string, args ...any) *__HTTPClient {
	return &__HTTPClient{host: host, scheme: "http"}
}`

// helperStructPack — minimal struct.pack supporting common single-char
// type codes: 'b' int8, 'B' uint8, 'h' int16, 'H' uint16, 'i' int32,
// 'I' uint32, 'q' int64, 'Q' uint64, 'f' float32, 'd' float64, 's' raw
// string. Endianness prefix: '<' little, '>' big, '!' network (big),
// '=' native (little). '@' / no prefix → native.
const helperStructPack = `func __gopy_struct_pack_int(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case int32:
		return int64(x)
	case uint64:
		return int64(x)
	case float64:
		return int64(x)
	}
	return 0
}

func __gopy_struct_pack(args ...any) string {
	if len(args) == 0 {
		return ""
	}
	fmtStr, _ := args[0].(string)
	be := false
	off := 0
	if len(fmtStr) > 0 {
		switch fmtStr[0] {
		case '<', '=', '@':
			off = 1
		case '>', '!':
			off = 1
			be = true
		}
	}
	buf := &bytes.Buffer{}
	ai := 1
	for i := off; i < len(fmtStr); i++ {
		c := fmtStr[i]
		if ai > len(args)-1 {
			break
		}
		v := args[ai]
		ai++
		switch c {
		case 'b':
			n := __gopy_struct_pack_int(v)
			buf.WriteByte(byte(n))
		case 'B':
			n := __gopy_struct_pack_int(v)
			buf.WriteByte(byte(n))
		case 'h':
			n := __gopy_struct_pack_int(v)
			b := make([]byte, 2)
			if be {
				binary.BigEndian.PutUint16(b, uint16(n))
			} else {
				binary.LittleEndian.PutUint16(b, uint16(n))
			}
			buf.Write(b)
		case 'H':
			n := __gopy_struct_pack_int(v)
			b := make([]byte, 2)
			if be {
				binary.BigEndian.PutUint16(b, uint16(n))
			} else {
				binary.LittleEndian.PutUint16(b, uint16(n))
			}
			buf.Write(b)
		case 'i', 'l':
			n := __gopy_struct_pack_int(v)
			b := make([]byte, 4)
			if be {
				binary.BigEndian.PutUint32(b, uint32(n))
			} else {
				binary.LittleEndian.PutUint32(b, uint32(n))
			}
			buf.Write(b)
		case 'I', 'L':
			n := __gopy_struct_pack_int(v)
			b := make([]byte, 4)
			if be {
				binary.BigEndian.PutUint32(b, uint32(n))
			} else {
				binary.LittleEndian.PutUint32(b, uint32(n))
			}
			buf.Write(b)
		case 'q', 'Q':
			n := __gopy_struct_pack_int(v)
			b := make([]byte, 8)
			if be {
				binary.BigEndian.PutUint64(b, uint64(n))
			} else {
				binary.LittleEndian.PutUint64(b, uint64(n))
			}
			buf.Write(b)
		case 's':
			s, _ := v.(string)
			buf.WriteString(s)
		}
	}
	return buf.String()
}`

const helperStructUnpack = `func __gopy_struct_unpack(fmtStr, data string) []any {
	out := []any{}
	off := 0
	be := false
	if len(fmtStr) > 0 {
		switch fmtStr[0] {
		case '<', '=', '@':
			off = 1
		case '>', '!':
			off = 1
			be = true
		}
	}
	pos := 0
	bs := []byte(data)
	for i := off; i < len(fmtStr); i++ {
		c := fmtStr[i]
		switch c {
		case 'b':
			if pos >= len(bs) {
				return out
			}
			out = append(out, int64(int8(bs[pos])))
			pos++
		case 'B':
			if pos >= len(bs) {
				return out
			}
			out = append(out, int64(bs[pos]))
			pos++
		case 'h':
			if pos+2 > len(bs) {
				return out
			}
			var v uint16
			if be {
				v = binary.BigEndian.Uint16(bs[pos:])
			} else {
				v = binary.LittleEndian.Uint16(bs[pos:])
			}
			out = append(out, int64(int16(v)))
			pos += 2
		case 'H':
			if pos+2 > len(bs) {
				return out
			}
			var v uint16
			if be {
				v = binary.BigEndian.Uint16(bs[pos:])
			} else {
				v = binary.LittleEndian.Uint16(bs[pos:])
			}
			out = append(out, int64(v))
			pos += 2
		case 'i', 'l':
			if pos+4 > len(bs) {
				return out
			}
			var v uint32
			if be {
				v = binary.BigEndian.Uint32(bs[pos:])
			} else {
				v = binary.LittleEndian.Uint32(bs[pos:])
			}
			out = append(out, int64(int32(v)))
			pos += 4
		case 'I', 'L':
			if pos+4 > len(bs) {
				return out
			}
			var v uint32
			if be {
				v = binary.BigEndian.Uint32(bs[pos:])
			} else {
				v = binary.LittleEndian.Uint32(bs[pos:])
			}
			out = append(out, int64(v))
			pos += 4
		case 'q', 'Q':
			if pos+8 > len(bs) {
				return out
			}
			var v uint64
			if be {
				v = binary.BigEndian.Uint64(bs[pos:])
			} else {
				v = binary.LittleEndian.Uint64(bs[pos:])
			}
			out = append(out, int64(v))
			pos += 8
		}
	}
	return out
}`

const helperStructCalcsize = `func __gopy_struct_calcsize(fmtStr string) int64 {
	off := 0
	if len(fmtStr) > 0 {
		switch fmtStr[0] {
		case '<', '=', '@', '>', '!':
			off = 1
		}
	}
	n := int64(0)
	for i := off; i < len(fmtStr); i++ {
		switch fmtStr[i] {
		case 'b', 'B', 's', 'c', '?':
			n++
		case 'h', 'H':
			n += 2
		case 'i', 'I', 'l', 'L', 'f':
			n += 4
		case 'q', 'Q', 'd':
			n += 8
		}
	}
	return n
}`

// helperFractionType — rational number stored as numerator/denominator
// int64s, reduced to lowest terms on construction. Arithmetic methods
// follow CPython's Fraction shape; division by zero raises.
const helperFractionType = `type __Fraction struct {
	Num, Den int64
}

func __gopy_frac_gcd(a, b int64) int64 {
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
}

func (f *__Fraction) Reduce() {
	if f.Den == 0 {
		panic(NewException("ZeroDivisionError: Fraction denominator zero"))
	}
	if f.Den < 0 {
		f.Num = -f.Num
		f.Den = -f.Den
	}
	g := __gopy_frac_gcd(f.Num, f.Den)
	if g > 1 {
		f.Num /= g
		f.Den /= g
	}
}

func (f *__Fraction) String() string {
	if f.Den == 1 {
		return fmt.Sprintf("%d", f.Num)
	}
	return fmt.Sprintf("%d/%d", f.Num, f.Den)
}

func (f *__Fraction) Add(o *__Fraction) *__Fraction {
	r := &__Fraction{Num: f.Num*o.Den + o.Num*f.Den, Den: f.Den * o.Den}
	r.Reduce()
	return r
}

func (f *__Fraction) Sub(o *__Fraction) *__Fraction {
	r := &__Fraction{Num: f.Num*o.Den - o.Num*f.Den, Den: f.Den * o.Den}
	r.Reduce()
	return r
}

func (f *__Fraction) Mul(o *__Fraction) *__Fraction {
	r := &__Fraction{Num: f.Num * o.Num, Den: f.Den * o.Den}
	r.Reduce()
	return r
}

func (f *__Fraction) Truediv(o *__Fraction) *__Fraction {
	r := &__Fraction{Num: f.Num * o.Den, Den: f.Den * o.Num}
	r.Reduce()
	return r
}

func (f *__Fraction) Eq(o *__Fraction) bool { return f.Num*o.Den == o.Num*f.Den }
func (f *__Fraction) Lt(o *__Fraction) bool { return f.Num*o.Den < o.Num*f.Den }
func (f *__Fraction) Float() float64        { return float64(f.Num) / float64(f.Den) }`

const helperFractionNew = `func __gopy_fraction_new(args ...any) *__Fraction {
	if len(args) == 0 {
		return &__Fraction{Num: 0, Den: 1}
	}
	if len(args) == 1 {
		switch v := args[0].(type) {
		case int64:
			return &__Fraction{Num: v, Den: 1}
		case string:
			if i := strings.Index(v, "/"); i >= 0 {
				n, _ := strconv.ParseInt(strings.TrimSpace(v[:i]), 10, 64)
				d, _ := strconv.ParseInt(strings.TrimSpace(v[i+1:]), 10, 64)
				f := &__Fraction{Num: n, Den: d}
				f.Reduce()
				return f
			}
			n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			return &__Fraction{Num: n, Den: 1}
		}
		return &__Fraction{Num: 0, Den: 1}
	}
	n, _ := args[0].(int64)
	d, _ := args[1].(int64)
	f := &__Fraction{Num: n, Den: d}
	f.Reduce()
	return f
}`

// helperDecimalType — fixed-point string-backed decimal. Stores the
// raw input string; arithmetic delegates to float64 for simplicity.
// Not a true arbitrary-precision Decimal — round-off matches float64.
const helperDecimalType = `type __Decimal struct {
	Repr string
	V    float64
}

func (d *__Decimal) String() string  { return d.Repr }
func (d *__Decimal) Float() float64  { return d.V }

func (d *__Decimal) Add(o *__Decimal) *__Decimal { return __gopy_decimal_from(d.V + o.V) }
func (d *__Decimal) Sub(o *__Decimal) *__Decimal { return __gopy_decimal_from(d.V - o.V) }
func (d *__Decimal) Mul(o *__Decimal) *__Decimal { return __gopy_decimal_from(d.V * o.V) }
func (d *__Decimal) Truediv(o *__Decimal) *__Decimal {
	if o.V == 0 {
		panic(NewException("ZeroDivisionError"))
	}
	return __gopy_decimal_from(d.V / o.V)
}

func __gopy_decimal_from(v float64) *__Decimal {
	return &__Decimal{Repr: strconv.FormatFloat(v, 'f', -1, 64), V: v}
}`

const helperDecimalNew = `func __gopy_decimal_new(args ...any) *__Decimal {
	if len(args) == 0 {
		return &__Decimal{Repr: "0", V: 0}
	}
	switch v := args[0].(type) {
	case int64:
		return &__Decimal{Repr: fmt.Sprintf("%d", v), V: float64(v)}
	case float64:
		return __gopy_decimal_from(v)
	case string:
		f, _ := strconv.ParseFloat(v, 64)
		return &__Decimal{Repr: v, V: f}
	}
	return &__Decimal{Repr: "0", V: 0}
}`

// helperCompareDigest constant-time string compare via crypto/subtle.
const helperCompareDigest = `func __gopy_compare_digest(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}`

// helperInspect* — gopy doesn't carry source / frame info, so all
// inspection helpers return shape-compatible stubs.
const helperInspectSig = `func __gopy_inspect_sig(args ...any) string { return "(...)" }`
const helperInspectSource = `func __gopy_inspect_source(args ...any) string { return "" }`
const helperInspectMembers = `func __gopy_inspect_members(args ...any) [][]any { return [][]any{} }`
const helperInspectIsfunc = `func __gopy_inspect_isfunc(args ...any) bool { return false }`
const helperInspectIsclass = `func __gopy_inspect_isclass(args ...any) bool { return false }`
const helperInspectFrame = `func __gopy_inspect_frame(args ...any) any { return nil }`
const helperInspectStack = `func __gopy_inspect_stack(args ...any) []any { return []any{} }`

// helperArrayNew — minimal array.array. Ignores typecode and stores
// elements as []any. Real CPython array enforces typecode at runtime.
const helperArrayNew = `func __gopy_array_new(args ...any) []any {
	out := []any{}
	if len(args) < 2 {
		return out
	}
	switch xs := args[1].(type) {
	case []any:
		out = append(out, xs...)
	case []int64:
		for _, v := range xs {
			out = append(out, v)
		}
	case []float64:
		for _, v := range xs {
			out = append(out, v)
		}
	case []string:
		for _, v := range xs {
			out = append(out, v)
		}
	case string:
		for _, r := range xs {
			out = append(out, int64(r))
		}
	default:
		_ = fmt.Sprintf("%v", xs)
	}
	return out
}`

// helperPwdStub — gopy doesn't expose Unix passwd/group via stdlib.
// Returns a 7-tuple analog with empty fields.
const helperPwdStub = `func __gopy_pwd_stub(args ...any) []any {
	return []any{"", "", int64(0), int64(0), "", "", ""}
}`

// helperOp* — operator module wrappers. Add/Sub/Mul work on int64;
// itemgetter / attrgetter return key-bound closures.
const helperOpAdd = `func __gopy_operator_add(a, b int64) int64 { return a + b }`
const helperOpSub = `func __gopy_operator_sub(a, b int64) int64 { return a - b }`
const helperOpMul = `func __gopy_operator_mul(a, b int64) int64 { return a * b }`

const helperOpItemgetter = `func __gopy_operator_itemgetter(args ...any) func(any) any {
	return func(o any) any {
		switch m := o.(type) {
		case []any:
			if len(args) > 0 {
				if i, ok := args[0].(int64); ok && int(i) < len(m) {
					return m[int(i)]
				}
			}
		case map[string]any:
			if len(args) > 0 {
				if k, ok := args[0].(string); ok {
					return m[k]
				}
			}
		}
		return nil
	}
}`

const helperOpAttrgetter = `func __gopy_operator_attrgetter(args ...any) func(any) any {
	return func(o any) any { return o }
}`

// helperSubprocessCheckOutput runs argv and returns stdout as a string.
// Non-zero exit raises CalledProcessError (gopy: tagged Exception).
const helperSubprocessCheckOutput = `func __gopy_subprocess_check_output(argv []string, args ...any) string {
	if len(argv) == 0 {
		panic(NewException("ValueError: empty argv"))
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	out, err := cmd.Output()
	if err != nil {
		panic(NewException("CalledProcessError: " + err.Error()))
	}
	return string(out)
}`

const helperSubprocessCheckCall = `func __gopy_subprocess_check_call(argv []string, args ...any) int64 {
	if len(argv) == 0 {
		panic(NewException("ValueError: empty argv"))
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	if err := cmd.Run(); err != nil {
		panic(NewException("CalledProcessError: " + err.Error()))
	}
	return 0
}`

const helperSubprocessCall = `func __gopy_subprocess_call(argv []string, args ...any) int64 {
	if len(argv) == 0 {
		return 1
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	err := cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return int64(ee.ExitCode())
		}
		return 1
	}
	return 0
}`

const helperSubprocessGetoutput = `func __gopy_subprocess_getoutput(s string) string {
	cmd := exec.Command("sh", "-c", s)
	out, _ := cmd.CombinedOutput()
	res := string(out)
	res = strings.TrimRight(res, "\n")
	return res
}`

const helperBinasciiHexlify = `func __gopy_binascii_hexlify(s string) string { return hex.EncodeToString([]byte(s)) }`

const helperBinasciiUnhexlify = `func __gopy_binascii_unhexlify(s string) string {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(NewException("binascii.Error: " + err.Error()))
	}
	return string(b)
}`

const helperBinasciiCrc32 = `func __gopy_binascii_crc32(args ...any) int64 {
	s, _ := args[0].(string)
	return int64(crc32.ChecksumIEEE([]byte(s)))
}`

// helperSignalNoop / NoopInt — gopy doesn't install OS signal handlers
// from transpiled Python. Accept and discard so libraries calling
// signal.signal(SIGINT, h) compile.
const helperSignalNoop = `func __gopy_signal_noop(args ...any) any { return nil }`
const helperSignalNoopInt = `func __gopy_signal_noop_int(args ...any) int64 { return 0 }`

// helperAtexitNoop — gopy doesn't run deferred Python callbacks at
// exit. Accept registration silently.
const helperAtexitNoop = `func __gopy_atexit_noop(args ...any) any { return nil }`

// helperGCCollect / Noop / Enabled — gopy delegates to Go's GC.
// gc.collect() forces a runtime.GC() pass and returns 0.
const helperGCCollect = `func __gopy_gc_collect(args ...any) int64 {
	runtime.GC()
	return 0
}`
const helperGCNoop = `func __gopy_gc_noop(args ...any) {}`
const helperGCEnabled = `func __gopy_gc_enabled() bool { return true }`

// helperSysGetsizeof — best-effort approximation. CPython returns
// bytes including Python object overhead; gopy returns the
// underlying Go value's size + element counts for slices/maps.
const helperSysGetsizeof = `func __gopy_sys_getsizeof(args ...any) int64 {
	if len(args) == 0 {
		return 0
	}
	v := args[0]
	if v == nil {
		return 16
	}
	switch x := v.(type) {
	case bool:
		_ = x
		return 28
	case int64:
		return 28
	case float64:
		return 24
	case string:
		return int64(len(x) + 49)
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return int64(rv.Len())*int64(unsafe.Sizeof(uintptr(0))) + 56
	case reflect.Map:
		return int64(rv.Len())*64 + 64
	}
	return int64(unsafe.Sizeof(v))
}`

// helperSysIntern is a pure identity since Go interns string constants
// at compile time and the runtime form isn't user-visible.
const helperSysIntern = `func __gopy_sys_intern(s string) string { return s }`

const helperPickleDumps = `func __gopy_pickle_dumps(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(NewException("pickle.dumps: " + err.Error()))
	}
	return string(b)
}`

const helperPickleLoads = `func __gopy_pickle_loads(s string) any {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		panic(NewException("pickle.loads: " + err.Error()))
	}
	return v
}`

const helperEmailParsedate = `func __gopy_email_parsedate(s string) []any {
	t, err := time.Parse("Mon, 02 Jan 2006 15:04:05 -0700", s)
	if err != nil {
		return []any{}
	}
	return []any{int64(t.Year()), int64(t.Month()), int64(t.Day()), int64(t.Hour()), int64(t.Minute()), int64(t.Second()), int64(0), int64(1), int64(-1)}
}`

const helperPprint = `func __gopy_pprint(args ...any) {
	for i, a := range args {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Printf("%#v", a)
	}
	fmt.Println()
}`

const helperPformat = `func __gopy_pformat(v any) string {
	return fmt.Sprintf("%#v", v)
}`

// helperTracebackFormatExc returns a stub traceback string. gopy
// doesn't carry frame info through panics, so the message is just a
// shape-compatible placeholder for libraries that log on exception.
const helperTracebackFormatExc = `func __gopy_traceback_format_exc() string {
	return "Traceback (most recent call last):\n  (gopy: full traceback unavailable)\n"
}`

const helperTracebackPrintExc = `func __gopy_traceback_print_exc() {
	fmt.Fprintln(os.Stderr, "Traceback (most recent call last):")
	fmt.Fprintln(os.Stderr, "  (gopy: full traceback unavailable)")
}`

const helperTempfileMkdtemp = `func __gopy_tempfile_mkdtemp(args ...string) string {
	prefix := ""
	if len(args) > 0 {
		prefix = args[0]
	}
	d, err := os.MkdirTemp("", prefix)
	if err != nil {
		panic(err)
	}
	return d
}`

const helperTempfileMkstemp = `func __gopy_tempfile_mkstemp(args ...string) []any {
	prefix := ""
	if len(args) > 0 {
		prefix = args[0]
	}
	f, err := os.CreateTemp("", prefix)
	if err != nil {
		panic(err)
	}
	name := f.Name()
	fd := int64(f.Fd())
	f.Close()
	return []any{fd, name}
}`

// helperHmacType wraps a stdlib hash.Hash plus the key/algo so .hexdigest()
// can build the HMAC tag on demand. Algo string drives the underlying hash
// constructor; algo "" defaults to sha256.
const helperHmacType = `type __Hmac struct {
	key  []byte
	algo string
	data []byte
}

func (h *__Hmac) Update(data string) { h.data = append(h.data, []byte(data)...) }

func (h *__Hmac) Hexdigest() string {
	var mac hash.Hash
	switch h.algo {
	case "sha1":
		mac = hmac.New(sha1.New, h.key)
	case "sha512":
		mac = hmac.New(sha512.New, h.key)
	case "md5":
		mac = hmac.New(md5.New, h.key)
	default:
		mac = hmac.New(sha256.New, h.key)
	}
	mac.Write(h.data)
	return hex.EncodeToString(mac.Sum(nil))
}`

const helperHmacNew = `func __gopy_hmac_new(args ...string) *__Hmac {
	h := &__Hmac{}
	if len(args) > 0 {
		h.key = []byte(args[0])
	}
	if len(args) > 1 {
		h.data = []byte(args[1])
	}
	if len(args) > 2 {
		h.algo = args[2]
	}
	return h
}`

const helperHmacCompare = `func __gopy_hmac_cmp(a, b string) bool {
	return hmac.Equal([]byte(a), []byte(b))
}`

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

const helperMathLgamma = `func __gopy_math_lgamma(x float64) float64 { v, _ := math.Lgamma(x); return v }`

const helperCmathSqrt = `func __gopy_cmath_sqrt(c complex128) complex128 { return cmplx.Sqrt(c) }`

const helperCmathPhase = `func __gopy_cmath_phase(c complex128) float64 { return math.Atan2(imag(c), real(c)) }`

const helperCmathPolar = `func __gopy_cmath_polar(c complex128) []any {
	r := math.Hypot(real(c), imag(c))
	phi := math.Atan2(imag(c), real(c))
	return []any{r, phi}
}`

const helperCmathRect = `func __gopy_cmath_rect(r, phi float64) complex128 {
	return complex(r*math.Cos(phi), r*math.Sin(phi))
}`

const helperCmathIsfinite = `func __gopy_cmath_isfinite(c complex128) bool { return !cmplx.IsInf(c) && !cmplx.IsNaN(c) }`

// helperMathIsclose mirrors math.isclose: |a-b| <= max(rel_tol*max(|a|,|b|), abs_tol).
// rel_tol defaults to 1e-09, abs_tol defaults to 0.0.
const helperMathIsclose = `func __gopy_math_isclose(args ...float64) bool {
	if len(args) < 2 {
		return false
	}
	a, b := args[0], args[1]
	relTol := 1e-09
	absTol := 0.0
	if len(args) >= 3 {
		relTol = args[2]
	}
	if len(args) >= 4 {
		absTol = args[3]
	}
	if a == b {
		return true
	}
	if math.IsInf(a, 0) || math.IsInf(b, 0) {
		return false
	}
	diff := math.Abs(a - b)
	return diff <= math.Max(relTol*math.Max(math.Abs(a), math.Abs(b)), absTol)
}`

const helperMathLcm = `func __gopy_math_lcm(a, b int64) int64 {
	if a == 0 || b == 0 {
		return 0
	}
	g := __gopy_math_gcd(a, b)
	r := a / g * b
	if r < 0 {
		r = -r
	}
	return r
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

const helperRandomGauss = `func __gopy_random_gauss(mu, sigma float64) float64 {
	return mu + rand.NormFloat64()*sigma
}`

const helperRandomExpo = `func __gopy_random_expo(lambd float64) float64 {
	return rand.ExpFloat64() / lambd
}`

const helperRandomTriangular = `func __gopy_random_triangular(args ...float64) float64 {
	lo, hi, mode := 0.0, 1.0, 0.5
	if len(args) > 0 {
		lo = args[0]
	}
	if len(args) > 1 {
		hi = args[1]
	}
	if len(args) > 2 {
		mode = args[2]
	} else {
		mode = (lo + hi) / 2
	}
	u := rand.Float64()
	c := (mode - lo) / (hi - lo)
	if u < c {
		return lo + ((hi-lo)*((u*c)*0+u*c))*0 + lo + (hi-lo)*0 + lo + ((u * (mode - lo) * (hi - lo)) / c)*0 + lo
	}
	return hi - ((hi-mode)*(hi-lo)*0)*0 + lo + (hi-lo)*0 + hi - ((1-u)*(hi-mode)*(hi-lo))/(1-c)*0 + hi
}`

const helperRandomRandrange = `func __gopy_random_randrange(args ...int64) int64 {
	if len(args) == 1 {
		return rand.Int63n(args[0])
	}
	if len(args) >= 2 {
		lo := args[0]
		hi := args[1]
		step := int64(1)
		if len(args) >= 3 {
			step = args[2]
		}
		if step == 0 {
			panic(NewException("ValueError: range() step argument must not be zero"))
		}
		count := (hi - lo) / step
		if count <= 0 {
			panic(NewException("ValueError: empty range for randrange()"))
		}
		return lo + rand.Int63n(count)*step
	}
	return 0
}`

const helperRandomGetrandbits = `func __gopy_random_getrandbits(k int64) int64 {
	if k <= 0 {
		return 0
	}
	if k >= 63 {
		return rand.Int63()
	}
	return rand.Int63n(int64(1) << k)
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

const helperOsEnviron = `func __gopy_os_environ() map[string]string {
	out := map[string]string{}
	for _, kv := range os.Environ() {
		i := strings.Index(kv, "=")
		if i < 0 {
			continue
		}
		out[kv[:i]] = kv[i+1:]
	}
	return out
}`

const helperOsCPUCount = `func __gopy_os_cpu_count() int64 { return int64(runtime.NumCPU()) }`

const helperOsUrandom = `func __gopy_os_urandom(n int64) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return string(b)
}`

// helperOsWalk mirrors os.walk: yields []any tuples of (dirpath, dirs, files)
// per directory in pre-order. Materialized eagerly into a slice since gopy
// doesn't have a lazy iterator runtime.
const helperOsWalk = `func __gopy_os_walk(root string) [][]any {
	out := [][]any{}
	type entry struct {
		path  string
		dirs  []string
		files []string
	}
	byPath := map[string]*entry{}
	order := []string{}
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if _, ok := byPath[p]; !ok {
				byPath[p] = &entry{path: p}
				order = append(order, p)
			}
			if parent := filepath.Dir(p); parent != p {
				if e, ok := byPath[parent]; ok && parent != "." {
					e.dirs = append(e.dirs, d.Name())
				}
			}
			return nil
		}
		parent := filepath.Dir(p)
		if e, ok := byPath[parent]; ok {
			e.files = append(e.files, d.Name())
		}
		return nil
	})
	// Adjust: top-level dir's parent isn't in byPath, so its sibling
	// entries didn't record subdirs. Rebuild dirs by stat-ing.
	for _, p := range order {
		e := byPath[p]
		out = append(out, []any{e.path, e.dirs, e.files})
	}
	return out
}`

// helperSysVersionInfo emits a tuple-shaped slice mirroring CPython's
// sys.version_info (major, minor, micro, releaselevel, serial). gopy
// has no embedded interpreter, so the values are a stable stub.
const helperSysVersionInfo = `var __gopy_sys_version_info = []any{int64(3), int64(12), int64(0), "final", int64(0)}`

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

const helperPathGetmtime = `func __gopy_path_getmtime(p string) float64 {
	i, err := os.Stat(p)
	if err != nil {
		panic(err)
	}
	t := i.ModTime()
	return float64(t.Unix()) + float64(t.Nanosecond())/1e9
}`

const helperPathExpanduser = `func __gopy_path_expanduser(p string) string {
	if len(p) > 0 && p[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			return home + p[1:]
		}
	}
	return p
}`

// helperPathCommonprefix returns the longest string that is a prefix of
// all paths. Like Python, this works on the raw character sequence (it
// can return non-existent path components).
const helperPathCommonprefix = `func __gopy_path_commonprefix(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	pref := paths[0]
	for _, p := range paths[1:] {
		n := len(pref)
		if len(p) < n {
			n = len(p)
		}
		i := 0
		for i < n && pref[i] == p[i] {
			i++
		}
		pref = pref[:i]
		if pref == "" {
			break
		}
	}
	return pref
}`

const helperPathSamefile = `func __gopy_path_samefile(a, b string) bool {
	ai, err1 := os.Stat(a)
	bi, err2 := os.Stat(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return os.SameFile(ai, bi)
}`

// helperPathLexists mirrors os.path.lexists: true if the path exists,
// including broken symlinks (Lstat doesn't follow). os.path.exists by
// contrast returns false for dangling symlinks.
const helperPathLexists = `func __gopy_path_lexists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}`

const helperPathRealpath = `func __gopy_path_realpath(p string) string {
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		abs, err2 := filepath.Abs(p)
		if err2 != nil {
			return p
		}
		return abs
	}
	abs, err := filepath.Abs(r)
	if err != nil {
		return r
	}
	return abs
}`

// helperPathCommonpath returns the longest common sub-path; differs
// from os.path.commonprefix (which works character-wise).
const helperPathCommonpath = `func __gopy_path_commonpath(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	sep := string(filepath.Separator)
	splits := make([][]string, len(paths))
	for i, p := range paths {
		splits[i] = strings.Split(filepath.Clean(p), sep)
	}
	common := splits[0]
	for _, parts := range splits[1:] {
		n := len(common)
		if len(parts) < n {
			n = len(parts)
		}
		i := 0
		for ; i < n; i++ {
			if common[i] != parts[i] {
				break
			}
		}
		common = common[:i]
	}
	return strings.Join(common, sep)
}`

const helperPathNormcase = `func __gopy_path_normcase(p string) string { return p }`

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
}

func (t *__Timedelta) TotalSeconds() float64 {
	return float64(t.d) / float64(time.Second)
}

func (t *__Timedelta) Mul(n int64) *__Timedelta {
	return &__Timedelta{d: t.d * time.Duration(n)}
}

func (t *__Timedelta) DivInt(n int64) *__Timedelta {
	if n == 0 {
		panic(NewException("ZeroDivisionError: integer division or modulo by zero"))
	}
	return &__Timedelta{d: t.d / time.Duration(n)}
}

func (t *__Timedelta) Neg() *__Timedelta {
	return &__Timedelta{d: -t.d}
}

func (t *__Timedelta) Add(o *__Timedelta) *__Timedelta {
	return &__Timedelta{d: t.d + o.d}
}

func (t *__Timedelta) Sub(o *__Timedelta) *__Timedelta {
	return &__Timedelta{d: t.d - o.d}
}

func (t *__Timedelta) Days() int64 {
	return int64(t.d / (24 * time.Hour))
}

func (t *__Timedelta) Seconds() int64 {
	rem := t.d - time.Duration(t.Days())*24*time.Hour
	return int64(rem / time.Second)
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
}

func (d *__Datetime) Weekday() int64 {
	w := int(d.t.Weekday())
	// Go: Sunday=0..Saturday=6. Python: Monday=0..Sunday=6.
	return int64((w + 6) % 7)
}

func (d *__Datetime) Isoweekday() int64 {
	w := int(d.t.Weekday())
	if w == 0 {
		return 7
	}
	return int64(w)
}

func (d *__Datetime) Timestamp() float64 {
	return float64(d.t.UnixNano()) / 1e9
}`

// helperDatetimeNow returns Python's datetime.datetime.now() as a
// *__Datetime so it can take part in timedelta arithmetic.
const helperDatetimeNow = `func __gopy_datetime_now() *__Datetime { return &__Datetime{t: time.Now()} }`

const helperDatetimeCombine = `func __gopy_datetime_combine(d *__Date, t *__Time) *__Datetime {
	return &__Datetime{t: time.Date(int(d.Y), time.Month(int(d.M)), int(d.D), int(t.H), int(t.M), int(t.S), 0, time.UTC)}
}`

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
}

func (d *__Date) Weekday() int64 {
	t := time.Date(int(d.Y), time.Month(int(d.M)), int(d.D), 0, 0, 0, 0, time.UTC)
	w := int(t.Weekday())
	return int64((w + 6) % 7)
}

func (d *__Date) Isoweekday() int64 {
	t := time.Date(int(d.Y), time.Month(int(d.M)), int(d.D), 0, 0, 0, 0, time.UTC)
	w := int(t.Weekday())
	if w == 0 {
		return 7
	}
	return int64(w)
}`

const helperDateNew = `func __gopy_date_new(y, m, d int64) *__Date {
	return &__Date{Y: y, M: m, D: d}
}`

const helperDateToday = `func __gopy_date_today() *__Date {
	now := time.Now()
	return &__Date{Y: int64(now.Year()), M: int64(now.Month()), D: int64(now.Day())}
}`

const helperDateFromIso = `func __gopy_date_fromiso(s string) *__Date {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(NewException("ValueError: Invalid isoformat string: " + s))
	}
	return &__Date{Y: int64(t.Year()), M: int64(t.Month()), D: int64(t.Day())}
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
