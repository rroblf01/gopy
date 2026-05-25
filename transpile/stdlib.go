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
			"argv":            {GoExpr: "os.Args", GoImport: "os"},
			"platform":        {GoExpr: "runtime.GOOS", GoImport: "runtime"},
			"version":         {GoExpr: `"3.12.0 (gopy)"`},
			"version_info":    {GoExpr: "__gopy_sys_version_info", Helper: helperSysVersionInfo, HelperName: "__gopy_sys_version_info"},
			"maxsize":         {GoExpr: "int64(9223372036854775807)"},
			"maxunicode":      {GoExpr: "int64(1114111)"},
			"byteorder":       {GoExpr: `"little"`},
			"executable":      {GoExpr: "__gopy_sys_executable()", Helper: helperSysExecutable, HelperName: "__gopy_sys_executable", HelperImports: []string{"os"}},
			"prefix":          {GoExpr: `"/usr"`},
			"exec_prefix":     {GoExpr: `"/usr"`},
			"base_prefix":     {GoExpr: `"/usr"`},
			"base_exec_prefix": {GoExpr: `"/usr"`},
			"path":            {GoExpr: `[]string{}`},
			"modules":         {GoExpr: `map[string]any{}`},
			"path_hooks":      {GoExpr: `[]any{}`},
			"path_importer_cache": {GoExpr: `map[string]any{}`},
			"meta_path":       {GoExpr: `[]any{}`},
			"builtin_module_names": {GoExpr: `[]string{"builtins", "sys", "_thread"}`},
			"hexversion":      {GoExpr: "int64(50923760)"},
			"api_version":     {GoExpr: "int64(1013)"},
			"copyright":       {GoExpr: `"gopy transpiler"`},
			"float_repr_style": {GoExpr: `"short"`},
			"int_info":        {GoExpr: `map[string]int64{"bits_per_digit": 30, "sizeof_digit": 4}`},
			"flags":           {GoExpr: `map[string]int64{"debug": 0, "inspect": 0, "interactive": 0, "optimize": 0, "verbose": 0}`},
			"dont_write_bytecode": {GoExpr: "false"},
			"warnoptions":     {GoExpr: `[]string{}`},
			"ps1":             {GoExpr: `">>> "`},
			"ps2":             {GoExpr: `"... "`},
		},
		Funcs: map[string]stdlibFunc{
			"exit":               {GoFunc: "os.Exit", GoImport: "os", IntArg0: true},
			"getsizeof":          {GoFunc: "__gopy_sys_getsizeof", Helper: helperSysGetsizeof, HelperImports: []string{"unsafe", "reflect"}, RetKind: "int"},
			"intern":             {GoFunc: "__gopy_sys_intern", Helper: helperSysIntern, RetKind: "str"},
			"exc_info":           {GoFunc: "__gopy_sys_exc_info", Helper: helperSysExcInfo},
			"setrecursionlimit":  {GoFunc: "__gopy_sys_setrecursion_unused"},
			"getrecursionlimit":  {GoFunc: "__gopy_sys_getrecursion", Helper: helperSysGetRecursion, RetKind: "int"},
			"getrefcount":        {GoFunc: "__gopy_sys_getrefcount", Helper: helperSysGetRefcount, RetKind: "int"},
			"getdefaultencoding": {GoFunc: "__gopy_sys_getenc", Helper: helperSysGetEnc, RetKind: "str"},
			"getfilesystemencoding": {GoFunc: "__gopy_sys_getfsenc", Helper: helperSysGetFsenc, RetKind: "str"},
			"set_int_max_str_digits": {GoFunc: "__gopy_sys_setintmax_unused"},
			"get_int_max_str_digits": {GoFunc: "__gopy_sys_getintmax", Helper: helperSysGetIntMaxDigits, RetKind: "int"},
			"settrace":           {GoFunc: "__gopy_sys_settrace_unused"},
			"setprofile":         {GoFunc: "__gopy_sys_setprofile_unused"},
			"gettrace":           {GoFunc: "__gopy_sys_gettrace_unused"},
			"getprofile":         {GoFunc: "__gopy_sys_getprofile_unused"},
			"audit":              {GoFunc: "__gopy_sys_audit_unused"},
			"breakpointhook":     {GoFunc: "__gopy_sys_breakpoint_unused"},
			"displayhook":        {GoFunc: "__gopy_sys_displayhook_unused"},
			"excepthook":         {GoFunc: "__gopy_sys_excepthook_unused"},
			"unraisablehook":     {GoFunc: "__gopy_sys_unraisable_unused"},
			"getswitchinterval":  {GoFunc: "__gopy_sys_getswitch", Helper: helperSysGetSwitch, RetKind: "float"},
			"setswitchinterval":  {GoFunc: "__gopy_sys_setswitch_unused"},
			"_getframe":          {GoFunc: "__gopy_sys_getframe_unused"},
			"getallocatedblocks": {GoFunc: "__gopy_sys_alloc", Helper: helperSysGetAllocated, RetKind: "int"},
			"is_finalizing":      {GoFunc: "__gopy_sys_isfin", Helper: helperSysIsFinalizing, RetKind: "bool"},
		},
	},
	"os": {
		Attrs: map[string]stdlibAttr{
			"sep":     {GoExpr: `string(os.PathSeparator)`, GoImport: "os"},
			"linesep": {GoExpr: `"\n"`},
			"pathsep": {GoExpr: `string(os.PathListSeparator)`, GoImport: "os"},
			"devnull": {GoExpr: `os.DevNull`, GoImport: "os"},
			"curdir":  {GoExpr: `"."`},
			"pardir":  {GoExpr: `".."`},
			"extsep":  {GoExpr: `"."`},
			"altsep":  {GoExpr: `""`},
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
			"scandir":   {GoFunc: "__gopy_os_scandir", Helper: helperOsScandir, ExtraHelpers: map[string]string{"__DirEntry": helperDirEntryType}, HelperImports: []string{"os"}},
			"chdir":     {GoFunc: "os.Chdir", GoImport: "os"},
			"getpid":    {GoFunc: "__gopy_os_getpid", GoImport: "os", Helper: helperOsGetpid, RetKind: "int"},
			"getppid":   {GoFunc: "__gopy_os_getppid", GoImport: "os", Helper: helperOsGetppid, RetKind: "int"},
			"getuid":    {GoFunc: "__gopy_os_getuid", GoImport: "os", Helper: helperOsGetuid, RetKind: "int"},
			"geteuid":   {GoFunc: "__gopy_os_geteuid", GoImport: "os", Helper: helperOsGeteuid, RetKind: "int"},
			"getgid":    {GoFunc: "__gopy_os_getgid", GoImport: "os", Helper: helperOsGetgid, RetKind: "int"},
			"getegid":   {GoFunc: "__gopy_os_getegid", GoImport: "os", Helper: helperOsGetegid, RetKind: "int"},
			"getlogin":  {GoFunc: "__gopy_os_getlogin", GoImport: "os", Helper: helperOsGetlogin, HelperImports: []string{"os/user"}, RetKind: "str"},
			"system":    {GoFunc: "__gopy_os_system", Helper: helperOsSystem, HelperImports: []string{"os/exec", "os"}, RetKind: "int"},
			"fspath":    {GoFunc: "__gopy_os_fspath", Helper: helperOsFspath, RetKind: "str"},
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
					"getatime":     {GoFunc: "__gopy_path_getatime", GoImport: "os", Helper: helperPathGetatime, ExtraHelpers: map[string]string{"__gopy_path_getmtime": helperPathGetmtime}, RetKind: "float"},
					"getctime":     {GoFunc: "__gopy_path_getctime", GoImport: "os", Helper: helperPathGetctime, ExtraHelpers: map[string]string{"__gopy_path_getmtime": helperPathGetmtime}, RetKind: "float"},
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
					"ismount":      {GoFunc: "__gopy_path_ismount", Helper: helperPathIsmount, HelperImports: []string{"os"}, RetKind: "bool"},
					"splitdrive":   {GoFunc: "__gopy_path_splitdrive", Helper: helperPathSplitdrive},
				},
			},
		},
	},
	"time": {
		Attrs: map[string]stdlibAttr{
			"CLOCK_REALTIME":  {GoExpr: "int64(0)"},
			"CLOCK_MONOTONIC": {GoExpr: "int64(1)"},
		},
		Funcs: map[string]stdlibFunc{
			"time":             {GoFunc: "__gopy_time_now_seconds", GoImport: "time", Helper: helperTimeNowSeconds, RetKind: "float"},
			"sleep":            {GoFunc: "__gopy_time_sleep", GoImport: "time", Helper: helperTimeSleep},
			"monotonic":        {GoFunc: "__gopy_time_monotonic", GoImport: "time", Helper: helperTimeMonotonic, RetKind: "float"},
			"perf_counter":     {GoFunc: "__gopy_time_monotonic", GoImport: "time", Helper: helperTimeMonotonic, RetKind: "float"},
			"perf_counter_ns":  {GoFunc: "__gopy_time_ns", GoImport: "time", Helper: helperTimeNs, RetKind: "int"},
			"monotonic_ns":     {GoFunc: "__gopy_time_ns", GoImport: "time", Helper: helperTimeNs, RetKind: "int"},
			"process_time":     {GoFunc: "__gopy_time_monotonic", GoImport: "time", Helper: helperTimeMonotonic, RetKind: "float"},
			"process_time_ns":  {GoFunc: "__gopy_time_ns", GoImport: "time", Helper: helperTimeNs, RetKind: "int"},
			"thread_time":      {GoFunc: "__gopy_time_monotonic", GoImport: "time", Helper: helperTimeMonotonic, RetKind: "float"},
			"thread_time_ns":   {GoFunc: "__gopy_time_ns", GoImport: "time", Helper: helperTimeNs, RetKind: "int"},
			"time_ns":          {GoFunc: "__gopy_time_ns", GoImport: "time", Helper: helperTimeNs, RetKind: "int"},
			"strftime":         {GoFunc: "__gopy_time_strftime", GoImport: "time", Helper: helperTimeStrftime, HelperImports: []string{"strings"}, RetKind: "str"},
			"localtime":        {GoFunc: "__gopy_time_localtime", GoImport: "time", Helper: helperTimeLocaltime},
			"gmtime":           {GoFunc: "__gopy_time_gmtime", GoImport: "time", Helper: helperTimeGmtime},
			"mktime":           {GoFunc: "__gopy_time_mktime", GoImport: "time", Helper: helperTimeMktime, RetKind: "float"},
			"asctime":          {GoFunc: "__gopy_time_asctime", GoImport: "time", Helper: helperTimeAsctime, RetKind: "str"},
			"ctime":            {GoFunc: "__gopy_time_ctime", GoImport: "time", Helper: helperTimeCtime, RetKind: "str"},
			"tzname":           {GoFunc: "__gopy_time_tzname", GoImport: "time", Helper: helperTimeTzname},
			"tzset":            {GoFunc: "__gopy_time_tzset_unused"},
			"clock_gettime":    {GoFunc: "__gopy_time_monotonic", GoImport: "time", Helper: helperTimeMonotonic, RetKind: "float"},
			"clock_gettime_ns": {GoFunc: "__gopy_time_ns", GoImport: "time", Helper: helperTimeNs, RetKind: "int"},
			"clock_settime":    {GoFunc: "__gopy_time_clock_settime_unused"},
			"clock_settime_ns": {GoFunc: "__gopy_time_clock_settime_unused"},
			"clock_getres":     {GoFunc: "__gopy_time_clockres", Helper: helperTimeClockres, RetKind: "float"},
			"get_clock_info":   {GoFunc: "__gopy_time_clockinfo_unused"},
		},
	},
	"json": {
		Funcs: map[string]stdlibFunc{
			"dumps":      {GoFunc: "__gopy_json_dumps", GoImport: "encoding/json", Helper: helperJSONDumps, HelperImports: []string{"strings"}},
			"loads":      {GoFunc: "__gopy_json_loads", GoImport: "encoding/json", Helper: helperJSONLoads},
			"load":       {GoFunc: "__gopy_json_load", GoImport: "encoding/json", Helper: helperJSONLoad, HelperImports: []string{"io"}},
			"dump":       {GoFunc: "__gopy_json_dump", GoImport: "encoding/json", Helper: helperJSONDump, HelperImports: []string{"strings"}},
			"JSONEncoder": {GoFunc: "__gopy_json_encoder_unused"},
			"JSONDecoder": {GoFunc: "__gopy_json_decoder_unused"},
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
			"fabs":      {GoFunc: "math.Abs", GoImport: "math", RetKind: "float"},
			"modf":      {GoFunc: "__gopy_math_modf", GoImport: "math", Helper: helperMathModf},
			"frexp":     {GoFunc: "__gopy_math_frexp", GoImport: "math", Helper: helperMathFrexp},
			"ldexp":     {GoFunc: "math.Ldexp", GoImport: "math", RetKind: "float"},
			"fsum":      {GoFunc: "__gopy_math_fsum", Helper: helperMathFsum, RetKind: "float"},
			"nextafter": {GoFunc: "math.Nextafter", GoImport: "math", RetKind: "float"},
			"ulp":       {GoFunc: "__gopy_math_ulp", GoImport: "math", Helper: helperMathUlp, RetKind: "float"},
		},
	},
	"hashlib": {
		Funcs: map[string]stdlibFunc{
			"sha256": {GoFunc: "__gopy_hashlib_sha256", GoImport: "crypto/sha256", Helper: helperHashlibSha256, RetTag: "__Hasher", ExtraHelpers: map[string]string{"__Hasher": helperHasherType}, HelperImports: []string{"encoding/hex", "crypto/md5", "crypto/sha1", "crypto/sha512"}},
			"md5":    {GoFunc: "__gopy_hashlib_md5", GoImport: "crypto/md5", Helper: helperHashlibMd5, RetTag: "__Hasher", ExtraHelpers: map[string]string{"__Hasher": helperHasherType}, HelperImports: []string{"encoding/hex", "crypto/sha256", "crypto/sha1", "crypto/sha512"}},
			"sha1":   {GoFunc: "__gopy_hashlib_sha1", GoImport: "crypto/sha1", Helper: helperHashlibSha1, RetTag: "__Hasher", ExtraHelpers: map[string]string{"__Hasher": helperHasherType}, HelperImports: []string{"encoding/hex", "crypto/md5", "crypto/sha256", "crypto/sha512"}},
			"sha512": {GoFunc: "__gopy_hashlib_sha512", GoImport: "crypto/sha512", Helper: helperHashlibSha512, RetTag: "__Hasher", ExtraHelpers: map[string]string{"__Hasher": helperHasherType}, HelperImports: []string{"encoding/hex", "crypto/md5", "crypto/sha256", "crypto/sha1"}},
			"sha224": {GoFunc: "__gopy_hashlib_sha224", GoImport: "crypto/sha256", Helper: helperHashlibSha224, RetTag: "__Hasher", ExtraHelpers: map[string]string{"__Hasher": helperHasherType}, HelperImports: []string{"encoding/hex", "crypto/md5", "crypto/sha1", "crypto/sha512"}},
			"sha384": {GoFunc: "__gopy_hashlib_sha384", GoImport: "crypto/sha512", Helper: helperHashlibSha384, RetTag: "__Hasher", ExtraHelpers: map[string]string{"__Hasher": helperHasherType}, HelperImports: []string{"encoding/hex", "crypto/md5", "crypto/sha1", "crypto/sha256"}},
			"new":    {GoFunc: "__gopy_hashlib_new", Helper: helperHashlibNew, RetTag: "__Hasher", ExtraHelpers: map[string]string{"__Hasher": helperHasherType}, HelperImports: []string{"encoding/hex", "crypto/md5", "crypto/sha1", "crypto/sha256", "crypto/sha512"}},
			"pbkdf2_hmac": {GoFunc: "__gopy_hashlib_pbkdf2", Helper: helperHashlibPbkdf2, HelperImports: []string{"crypto/hmac", "crypto/sha1", "crypto/sha256", "crypto/sha512", "crypto/md5", "hash", "encoding/binary", "encoding/hex"}, RetKind: "str"},
		},
	},
	"secrets": {
		Funcs: map[string]stdlibFunc{
			"token_hex":     {GoFunc: "__gopy_secrets_token_hex", GoImport: "crypto/rand", Helper: helperSecretsTokenHex, HelperImports: []string{"encoding/hex"}, RetKind: "str"},
			"token_urlsafe": {GoFunc: "__gopy_secrets_token_urlsafe", GoImport: "crypto/rand", Helper: helperSecretsTokenUrl, HelperImports: []string{"encoding/base64"}, RetKind: "str"},
			"token_bytes":   {GoFunc: "__gopy_secrets_token_bytes", GoImport: "crypto/rand", Helper: helperSecretsTokenBytes, RetKind: "str"},
			"randbelow":     {GoFunc: "__gopy_secrets_randbelow", Helper: helperSecretsRandbelow, HelperImports: []string{"crypto/rand", "math/big"}, RetKind: "int"},
			"choice":        {GoFunc: "__gopy_secrets_choice_unused"},
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
			"a85encode":         {GoFunc: "__gopy_a85encode", GoImport: "encoding/ascii85", Helper: helperA85Encode, RetKind: "str"},
			"a85decode":         {GoFunc: "__gopy_a85decode", GoImport: "encoding/ascii85", Helper: helperA85Decode, RetKind: "str"},
		},
	},
	"urllib": {
		Subs: map[string]stdlibModule{
			"request": {
				Funcs: map[string]stdlibFunc{
					"urlopen":     {GoFunc: "__gopy_url_urlopen", Helper: helperURLOpen, HelperImports: []string{"io", "net/http", "strings"}, RetTag: "__HTTPResponse", ExtraHelpers: map[string]string{"__HTTPResponse": helperHTTPResponseType}},
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
			"error": {
				Funcs: map[string]stdlibFunc{
					"URLError":             {GoFunc: "__gopy_urllib_error_unused"},
					"HTTPError":            {GoFunc: "__gopy_urllib_error_unused"},
					"ContentTooShortError": {GoFunc: "__gopy_urllib_error_unused"},
				},
			},
			"robotparser": {
				Funcs: map[string]stdlibFunc{
					"RobotFileParser": {GoFunc: "__gopy_urllib_robot_unused"},
				},
			},
			"response": {
				Funcs: map[string]stdlibFunc{
					"addinfourl": {GoFunc: "__gopy_urllib_resp_unused"},
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
			"Template":  {GoFunc: "__gopy_string_template_new", Helper: helperStringTemplateNew, RetTag: "__Template", ExtraHelpers: map[string]string{"__Template": helperStringTemplateType}, HelperImports: []string{"strings", "fmt"}},
			"capwords":  {GoFunc: "__gopy_string_capwords", Helper: helperStringCapwords, HelperImports: []string{"strings"}, RetKind: "str"},
			"Formatter": {GoFunc: "__gopy_string_formatter_unused"},
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
			"namedtuple":  {GoFunc: "__gopy_namedtuple_unused"},
			"ChainMap":    {GoFunc: "__gopy_chainmap_unused"},
			"UserDict":    {GoFunc: "__gopy_userdict_unused"},
			"UserList":    {GoFunc: "__gopy_userlist_unused"},
			"UserString":  {GoFunc: "__gopy_userstring_unused"},
		},
		Subs: map[string]stdlibModule{
			"abc": {
				Funcs: map[string]stdlibFunc{
					"Container":   {GoFunc: "__gopy_collections_abc_unused"},
					"Hashable":    {GoFunc: "__gopy_collections_abc_unused"},
					"Iterable":    {GoFunc: "__gopy_collections_abc_unused"},
					"Iterator":    {GoFunc: "__gopy_collections_abc_unused"},
					"Reversible":  {GoFunc: "__gopy_collections_abc_unused"},
					"Generator":   {GoFunc: "__gopy_collections_abc_unused"},
					"Sized":       {GoFunc: "__gopy_collections_abc_unused"},
					"Callable":    {GoFunc: "__gopy_collections_abc_unused"},
					"Collection":  {GoFunc: "__gopy_collections_abc_unused"},
					"Sequence":    {GoFunc: "__gopy_collections_abc_unused"},
					"MutableSequence": {GoFunc: "__gopy_collections_abc_unused"},
					"ByteString":  {GoFunc: "__gopy_collections_abc_unused"},
					"Set":         {GoFunc: "__gopy_collections_abc_unused"},
					"MutableSet":  {GoFunc: "__gopy_collections_abc_unused"},
					"Mapping":     {GoFunc: "__gopy_collections_abc_unused"},
					"MutableMapping": {GoFunc: "__gopy_collections_abc_unused"},
					"MappingView": {GoFunc: "__gopy_collections_abc_unused"},
					"ItemsView":   {GoFunc: "__gopy_collections_abc_unused"},
					"KeysView":    {GoFunc: "__gopy_collections_abc_unused"},
					"ValuesView":  {GoFunc: "__gopy_collections_abc_unused"},
					"Awaitable":   {GoFunc: "__gopy_collections_abc_unused"},
					"AsyncIterable": {GoFunc: "__gopy_collections_abc_unused"},
					"AsyncIterator": {GoFunc: "__gopy_collections_abc_unused"},
					"AsyncGenerator": {GoFunc: "__gopy_collections_abc_unused"},
					"Coroutine":   {GoFunc: "__gopy_collections_abc_unused"},
					"Buffer":      {GoFunc: "__gopy_collections_abc_unused"},
				},
			},
		},
	},
	"shutil": {
		Funcs: map[string]stdlibFunc{
			"rmtree":            {GoFunc: "__gopy_shutil_rmtree", GoImport: "os", Helper: helperShutilRmtree},
			"copy":              {GoFunc: "__gopy_shutil_copy", GoImport: "io", Helper: helperShutilCopy, HelperImports: []string{"os"}},
			"copyfile":          {GoFunc: "__gopy_shutil_copy", GoImport: "io", Helper: helperShutilCopy, HelperImports: []string{"os"}},
			"copytree":          {GoFunc: "__gopy_shutil_copytree", GoImport: "io", Helper: helperShutilCopytree, ExtraHelpers: map[string]string{"__gopy_shutil_copy": helperShutilCopy}, HelperImports: []string{"os"}, RetKind: "str"},
			"move":              {GoFunc: "__gopy_shutil_move", GoImport: "os", Helper: helperShutilMove},
			"which":             {GoFunc: "__gopy_shutil_which", Helper: helperShutilWhich, HelperImports: []string{"os/exec"}, RetKind: "str"},
			"disk_usage":        {GoFunc: "__gopy_shutil_diskusage", Helper: helperShutilDiskUsage},
			"get_terminal_size": {GoFunc: "__gopy_shutil_terminal_size", Helper: helperShutilTerminalSize},
		},
	},
	"tempfile": {
		Funcs: map[string]stdlibFunc{
			"mkdtemp":            {GoFunc: "__gopy_tempfile_mkdtemp", GoImport: "os", Helper: helperTempfileMkdtemp, RetKind: "str"},
			"gettempdir":         {GoFunc: "os.TempDir", GoImport: "os", RetKind: "str"},
			"mkstemp":            {GoFunc: "__gopy_tempfile_mkstemp", GoImport: "os", Helper: helperTempfileMkstemp},
			// Marker so `from tempfile import TemporaryDirectory` resolves
			// the alias; actual context-manager lowering lives in
			// emitTempDir, which intercepts the With statement before any
			// stdlib helper call would be emitted.
			"TemporaryDirectory":  {GoFunc: "__gopy_tempdir_unused"},
			"NamedTemporaryFile":  {GoFunc: "__gopy_namedtempfile_unused"},
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
		Attrs: map[string]stdlibAttr{
			"types_map":       {GoExpr: `map[string]string{".html": "text/html", ".htm": "text/html", ".css": "text/css", ".js": "application/javascript", ".json": "application/json", ".xml": "application/xml", ".txt": "text/plain", ".csv": "text/csv", ".pdf": "application/pdf", ".zip": "application/zip", ".gz": "application/gzip", ".tar": "application/x-tar", ".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif", ".svg": "image/svg+xml", ".mp3": "audio/mpeg", ".mp4": "video/mp4", ".wav": "audio/wav", ".py": "text/x-python", ".go": "text/x-go"}`},
			"common_types":    {GoExpr: `map[string]string{".jpg": "image/jpg"}`},
			"suffix_map":      {GoExpr: `map[string]string{".tgz": ".tar.gz", ".taz": ".tar.gz", ".tz": ".tar.gz", ".tbz2": ".tar.bz2", ".txz": ".tar.xz"}`},
			"encodings_map":   {GoExpr: `map[string]string{".gz": "gzip", ".Z": "compress", ".bz2": "bzip2", ".xz": "xz", ".br": "br"}`},
		},
		Funcs: map[string]stdlibFunc{
			"guess_type":      {GoFunc: "__gopy_mimetypes_guess", Helper: helperMimetypesGuess, HelperImports: []string{"mime", "path/filepath"}},
			"guess_extension": {GoFunc: "__gopy_mimetypes_guess_ext", Helper: helperMimetypesGuessExt, HelperImports: []string{"mime"}, RetKind: "str"},
			"init":            {GoFunc: "__gopy_mimetypes_init", Helper: helperMimetypesInit},
			"add_type":        {GoFunc: "__gopy_mimetypes_add", Helper: helperMimetypesAdd},
			"MimeTypes":       {GoFunc: "__gopy_mimetypes_class_unused"},
		},
	},
	"xml": {
		Subs: map[string]stdlibModule{
			"etree": {
				Subs: map[string]stdlibModule{
					"ElementTree": {
						Funcs: map[string]stdlibFunc{
							"fromstring": {GoFunc: "__gopy_xml_fromstring", Helper: helperXMLFromstring, RetTag: "__XMLElement", ExtraHelpers: map[string]string{"__XMLElement": helperXMLElementType, "__gopy_xml_serialize": helperXMLSerialize}, HelperImports: []string{"encoding/xml", "strings", "sort"}},
							"tostring":   {GoFunc: "__gopy_xml_tostring", Helper: helperXMLTostring, ExtraHelpers: map[string]string{"__XMLElement": helperXMLElementType, "__gopy_xml_serialize": helperXMLSerialize}, HelperImports: []string{"strings", "sort"}, RetKind: "str"},
							"Element":    {GoFunc: "__gopy_xml_element", Helper: helperXMLElement, RetTag: "__XMLElement", ExtraHelpers: map[string]string{"__XMLElement": helperXMLElementType, "__gopy_xml_serialize": helperXMLSerialize}, HelperImports: []string{"strings", "sort", "fmt"}},
							"SubElement": {GoFunc: "__gopy_xml_subelement", Helper: helperXMLSubElement, RetTag: "__XMLElement", ExtraHelpers: map[string]string{"__XMLElement": helperXMLElementType, "__gopy_xml_serialize": helperXMLSerialize, "__gopy_xml_element": helperXMLElement}, HelperImports: []string{"strings", "sort", "fmt"}},
							"parse":      {GoFunc: "__gopy_xml_parse", Helper: helperXMLParse, RetTag: "__XMLTree", ExtraHelpers: map[string]string{"__XMLTree": helperXMLTreeType, "__XMLElement": helperXMLElementType, "__gopy_xml_serialize": helperXMLSerialize, "__gopy_xml_fromstring": helperXMLFromstring}, HelperImports: []string{"encoding/xml", "os", "strings", "sort"}},
						},
					},
				},
			},
			"dom": {
				Subs: map[string]stdlibModule{
					"minidom": {
						Funcs: map[string]stdlibFunc{
							"Document":             {GoFunc: "__gopy_dom_doc_unused"},
							"DocumentType":         {GoFunc: "__gopy_dom_doc_unused"},
							"Element":              {GoFunc: "__gopy_dom_elem_unused"},
							"Attr":                 {GoFunc: "__gopy_dom_attr_unused"},
							"Text":                 {GoFunc: "__gopy_dom_text_unused"},
							"Comment":              {GoFunc: "__gopy_dom_comment_unused"},
							"Node":                 {GoFunc: "__gopy_dom_node_unused"},
							"NodeList":             {GoFunc: "__gopy_dom_nodelist_unused"},
							"parseString":          {GoFunc: "__gopy_dom_parse_str_unused"},
							"parse":                {GoFunc: "__gopy_dom_parse_unused"},
							"getDOMImplementation": {GoFunc: "__gopy_dom_impl_unused"},
						},
					},
				},
				Funcs: map[string]stdlibFunc{
					"Node":              {GoFunc: "__gopy_dom_node_unused"},
					"DOMException":      {GoFunc: "__gopy_dom_err_unused"},
					"DOMImplementation": {GoFunc: "__gopy_dom_impl_unused"},
				},
			},
			"sax": {
				Funcs: map[string]stdlibFunc{
					"make_parser":     {GoFunc: "__gopy_sax_parser_unused"},
					"parse":           {GoFunc: "__gopy_sax_parse_unused"},
					"parseString":     {GoFunc: "__gopy_sax_parse_unused"},
					"ContentHandler":  {GoFunc: "__gopy_sax_handler_unused"},
					"ErrorHandler":    {GoFunc: "__gopy_sax_handler_unused"},
					"SAXException":    {GoFunc: "__gopy_sax_err_unused"},
					"SAXParseException": {GoFunc: "__gopy_sax_err_unused"},
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
			"server": {
				Funcs: map[string]stdlibFunc{
					"HTTPServer":               {GoFunc: "__gopy_http_server_unused"},
					"ThreadingHTTPServer":      {GoFunc: "__gopy_http_server_unused"},
					"BaseHTTPRequestHandler":   {GoFunc: "__gopy_http_handler_unused"},
					"SimpleHTTPRequestHandler": {GoFunc: "__gopy_http_handler_unused"},
					"CGIHTTPRequestHandler":    {GoFunc: "__gopy_http_handler_unused"},
				},
			},
			"cookies": {
				Funcs: map[string]stdlibFunc{
					"SimpleCookie":   {GoFunc: "__gopy_http_cookie_unused"},
					"Morsel":         {GoFunc: "__gopy_http_cookie_unused"},
					"BaseCookie":     {GoFunc: "__gopy_http_cookie_unused"},
					"CookieError":    {GoFunc: "__gopy_http_cookie_unused"},
				},
			},
			"cookiejar": {
				Funcs: map[string]stdlibFunc{
					"CookieJar":        {GoFunc: "__gopy_http_cookiejar_unused"},
					"FileCookieJar":    {GoFunc: "__gopy_http_cookiejar_unused"},
					"MozillaCookieJar": {GoFunc: "__gopy_http_cookiejar_unused"},
					"LWPCookieJar":     {GoFunc: "__gopy_http_cookiejar_unused"},
					"Cookie":           {GoFunc: "__gopy_http_cookiejar_unused"},
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
		Attrs: map[string]stdlibAttr{
			"ROUND_UP":         {GoExpr: `"ROUND_UP"`},
			"ROUND_DOWN":       {GoExpr: `"ROUND_DOWN"`},
			"ROUND_CEILING":    {GoExpr: `"ROUND_CEILING"`},
			"ROUND_FLOOR":      {GoExpr: `"ROUND_FLOOR"`},
			"ROUND_HALF_UP":    {GoExpr: `"ROUND_HALF_UP"`},
			"ROUND_HALF_DOWN":  {GoExpr: `"ROUND_HALF_DOWN"`},
			"ROUND_HALF_EVEN":  {GoExpr: `"ROUND_HALF_EVEN"`},
			"ROUND_05UP":       {GoExpr: `"ROUND_05UP"`},
			"MAX_PREC":         {GoExpr: "int64(999999999999999999)"},
			"MAX_EMAX":         {GoExpr: "int64(999999999999999999)"},
			"MIN_EMIN":         {GoExpr: "int64(-999999999999999999)"},
		},
		Funcs: map[string]stdlibFunc{
			"Decimal":          {GoFunc: "__gopy_decimal_new", Helper: helperDecimalNew, RetTag: "__Decimal", ExtraHelpers: map[string]string{"__Decimal": helperDecimalType}, HelperImports: []string{"fmt", "strconv"}},
			"Context":          {GoFunc: "__gopy_decimal_ctx_unused"},
			"getcontext":       {GoFunc: "__gopy_decimal_getctx_unused"},
			"setcontext":       {GoFunc: "__gopy_decimal_setctx_unused"},
			"localcontext":     {GoFunc: "__gopy_decimal_localctx_unused"},
			"BasicContext":     {GoFunc: "__gopy_decimal_basicctx_unused"},
			"ExtendedContext":  {GoFunc: "__gopy_decimal_extctx_unused"},
			"DefaultContext":   {GoFunc: "__gopy_decimal_defctx_unused"},
			"InvalidOperation": {GoFunc: "__gopy_decimal_err_unused"},
			"DivisionByZero":   {GoFunc: "__gopy_decimal_err_unused"},
			"Overflow":         {GoFunc: "__gopy_decimal_err_unused"},
			"Underflow":        {GoFunc: "__gopy_decimal_err_unused"},
			"Inexact":          {GoFunc: "__gopy_decimal_err_unused"},
			"Rounded":          {GoFunc: "__gopy_decimal_err_unused"},
			"Subnormal":        {GoFunc: "__gopy_decimal_err_unused"},
			"Clamped":          {GoFunc: "__gopy_decimal_err_unused"},
			"FloatOperation":   {GoFunc: "__gopy_decimal_err_unused"},
			"DecimalException": {GoFunc: "__gopy_decimal_err_unused"},
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
		Attrs: map[string]stdlibAttr{
			"HIGHEST_PROTOCOL": {GoExpr: "int64(5)"},
			"DEFAULT_PROTOCOL": {GoExpr: "int64(5)"},
		},
		Funcs: map[string]stdlibFunc{
			"dumps":     {GoFunc: "__gopy_pickle_dumps", Helper: helperPickleDumps, HelperImports: []string{"encoding/json"}, RetKind: "str"},
			"loads":     {GoFunc: "__gopy_pickle_loads", Helper: helperPickleLoads, HelperImports: []string{"encoding/json"}},
			"Pickler":   {GoFunc: "__gopy_pickle_pickler_unused"},
			"Unpickler": {GoFunc: "__gopy_pickle_unpickler_unused"},
		},
	},
	"configparser": {
		Funcs: map[string]stdlibFunc{
			"ConfigParser": {GoFunc: "__gopy_configparser_new", Helper: helperConfigParserNew, RetTag: "__ConfigParser", ExtraHelpers: map[string]string{"__ConfigParser": helperConfigParserType}, HelperImports: []string{"bufio", "os", "strings", "strconv"}},
		},
	},
	"email": {
		Subs: map[string]stdlibModule{
			"utils": {
				Funcs: map[string]stdlibFunc{
					"formatdate":      {GoFunc: "__gopy_email_formatdate", Helper: helperEmailFormatdate, HelperImports: []string{"time"}, RetKind: "str"},
					"parsedate":       {GoFunc: "__gopy_email_parsedate", Helper: helperEmailParsedate, HelperImports: []string{"time"}},
					"format_datetime": {GoFunc: "__gopy_email_format_datetime", Helper: helperEmailFormatDatetime, HelperImports: []string{"time"}, RetKind: "str"},
					"parseaddr":       {GoFunc: "__gopy_email_parseaddr", Helper: helperEmailParseaddr, HelperImports: []string{"strings"}},
					"formataddr":      {GoFunc: "__gopy_email_formataddr", Helper: helperEmailFormataddr, HelperImports: []string{"fmt"}, RetKind: "str"},
					"make_msgid":      {GoFunc: "__gopy_email_make_msgid", Helper: helperEmailMakeMsgid, HelperImports: []string{"os", "fmt", "time"}, RetKind: "str"},
				},
			},
			"parser": {
				Funcs: map[string]stdlibFunc{
					"Parser":       {GoFunc: "__gopy_email_parser_unused"},
					"BytesParser":  {GoFunc: "__gopy_email_parser_unused"},
					"FeedParser":   {GoFunc: "__gopy_email_parser_unused"},
					"HeaderParser": {GoFunc: "__gopy_email_parser_unused"},
				},
			},
			"message": {
				Funcs: map[string]stdlibFunc{
					"EmailMessage": {GoFunc: "__gopy_email_message_unused"},
					"Message":      {GoFunc: "__gopy_email_message_unused"},
				},
			},
			"mime": {
				Subs: map[string]stdlibModule{
					"text": {
						Funcs: map[string]stdlibFunc{
							"MIMEText": {GoFunc: "__gopy_mime_text_unused"},
						},
					},
					"base": {
						Funcs: map[string]stdlibFunc{
							"MIMEBase": {GoFunc: "__gopy_mime_base_unused"},
						},
					},
					"multipart": {
						Funcs: map[string]stdlibFunc{
							"MIMEMultipart": {GoFunc: "__gopy_mime_multipart_unused"},
						},
					},
					"image": {
						Funcs: map[string]stdlibFunc{
							"MIMEImage": {GoFunc: "__gopy_mime_image_unused"},
						},
					},
					"audio": {
						Funcs: map[string]stdlibFunc{
							"MIMEAudio": {GoFunc: "__gopy_mime_audio_unused"},
						},
					},
					"application": {
						Funcs: map[string]stdlibFunc{
							"MIMEApplication": {GoFunc: "__gopy_mime_application_unused"},
						},
					},
				},
			},
			"policy": {
				Funcs: map[string]stdlibFunc{
					"default":     {GoFunc: "__gopy_email_policy_unused"},
					"strict":      {GoFunc: "__gopy_email_policy_unused"},
					"compat32":    {GoFunc: "__gopy_email_policy_unused"},
					"SMTP":        {GoFunc: "__gopy_email_policy_unused"},
					"HTTP":        {GoFunc: "__gopy_email_policy_unused"},
					"EmailPolicy": {GoFunc: "__gopy_email_policy_unused"},
				},
			},
			"charset": {
				Funcs: map[string]stdlibFunc{
					"Charset": {GoFunc: "__gopy_email_charset_unused"},
				},
			},
			"contentmanager": {
				Funcs: map[string]stdlibFunc{
					"ContentManager":   {GoFunc: "__gopy_email_cm_unused"},
					"raw_data_manager": {GoFunc: "__gopy_email_cm_unused"},
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
			"format_exc":        {GoFunc: "__gopy_traceback_format_exc", Helper: helperTracebackFormatExc, RetKind: "str"},
			"print_exc":         {GoFunc: "__gopy_traceback_print_exc", Helper: helperTracebackPrintExc, HelperImports: []string{"fmt", "os"}},
			"print_exception":   {GoFunc: "__gopy_traceback_print_exception", Helper: helperTracebackPrintException, HelperImports: []string{"fmt", "os"}},
			"format_exception":  {GoFunc: "__gopy_traceback_format_exception", Helper: helperTracebackFormatException, HelperImports: []string{"fmt"}},
			"format_exception_only": {GoFunc: "__gopy_traceback_format_exception_only", Helper: helperTracebackFormatExceptionOnly, HelperImports: []string{"fmt"}},
			"format_stack":      {GoFunc: "__gopy_traceback_format_stack", Helper: helperTracebackFormatStack},
			"print_stack":       {GoFunc: "__gopy_traceback_print_stack", Helper: helperTracebackPrintStack},
			"extract_stack":     {GoFunc: "__gopy_traceback_extract_stack", Helper: helperTracebackExtractStack},
			"extract_tb":        {GoFunc: "__gopy_traceback_extract_tb_unused"},
			"print_tb":          {GoFunc: "__gopy_traceback_print_tb_unused"},
			"format_tb":         {GoFunc: "__gopy_traceback_format_tb_unused"},
			"format_list":       {GoFunc: "__gopy_traceback_format_list_unused"},
			"clear_frames":      {GoFunc: "__gopy_traceback_clear_unused"},
			"walk_stack":        {GoFunc: "__gopy_traceback_walk_unused"},
			"walk_tb":           {GoFunc: "__gopy_traceback_walk_tb_unused"},
			"StackSummary":      {GoFunc: "__gopy_traceback_stack_summary_unused"},
			"FrameSummary":      {GoFunc: "__gopy_traceback_frame_summary_unused"},
			"TracebackException": {GoFunc: "__gopy_traceback_exception_unused"},
		},
	},
	"inspect": {
		Funcs: map[string]stdlibFunc{
			"signature":       {GoFunc: "__gopy_inspect_sig", Helper: helperInspectSig, RetKind: "str"},
			"getsource":       {GoFunc: "__gopy_inspect_source", Helper: helperInspectSource, RetKind: "str"},
			"getmembers":      {GoFunc: "__gopy_inspect_members", Helper: helperInspectMembers},
			"isfunction":      {GoFunc: "__gopy_inspect_isfunc", Helper: helperInspectIsfunc, RetKind: "bool"},
			"isclass":         {GoFunc: "__gopy_inspect_isclass", Helper: helperInspectIsclass, RetKind: "bool"},
			"ismethod":        {GoFunc: "__gopy_inspect_isfunc", Helper: helperInspectIsfunc, RetKind: "bool"},
			"isbuiltin":       {GoFunc: "__gopy_inspect_isfunc", Helper: helperInspectIsfunc, RetKind: "bool"},
			"iscoroutine":     {GoFunc: "__gopy_inspect_isfunc", Helper: helperInspectIsfunc, RetKind: "bool"},
			"iscoroutinefunction": {GoFunc: "__gopy_inspect_isfunc", Helper: helperInspectIsfunc, RetKind: "bool"},
			"isgenerator":     {GoFunc: "__gopy_inspect_isfunc", Helper: helperInspectIsfunc, RetKind: "bool"},
			"isgeneratorfunction": {GoFunc: "__gopy_inspect_isfunc", Helper: helperInspectIsfunc, RetKind: "bool"},
			"isawaitable":     {GoFunc: "__gopy_inspect_isfunc", Helper: helperInspectIsfunc, RetKind: "bool"},
			"getfullargspec":  {GoFunc: "__gopy_inspect_argspec", Helper: helperInspectArgspec, RetKind: "str"},
			"getmodule":       {GoFunc: "__gopy_inspect_getmodule", Helper: helperInspectGetmodule, RetKind: "str"},
			"getfile":         {GoFunc: "__gopy_inspect_getfile", Helper: helperInspectGetfile, RetKind: "str"},
			"currentframe":    {GoFunc: "__gopy_inspect_frame", Helper: helperInspectFrame},
			"stack":           {GoFunc: "__gopy_inspect_stack", Helper: helperInspectStack},
		},
	},
	"operator": {
		Funcs: map[string]stdlibFunc{
			"add":         {GoFunc: "__gopy_operator_add", Helper: helperOpAdd, RetKind: "int"},
			"sub":         {GoFunc: "__gopy_operator_sub", Helper: helperOpSub, RetKind: "int"},
			"mul":         {GoFunc: "__gopy_operator_mul", Helper: helperOpMul, RetKind: "int"},
			"truediv":     {GoFunc: "__gopy_operator_truediv", Helper: helperOpTruediv, RetKind: "float"},
			"floordiv":    {GoFunc: "__gopy_operator_floordiv", Helper: helperOpFloordiv, RetKind: "int"},
			"mod":         {GoFunc: "__gopy_operator_mod", Helper: helperOpMod, RetKind: "int"},
			"neg":         {GoFunc: "__gopy_operator_neg", Helper: helperOpNeg, RetKind: "int"},
			"pos":         {GoFunc: "__gopy_operator_pos", Helper: helperOpPos, RetKind: "int"},
			"abs":         {GoFunc: "__gopy_operator_abs", Helper: helperOpAbs, RetKind: "int"},
			"lt":          {GoFunc: "__gopy_operator_lt", Helper: helperOpLt, RetKind: "bool"},
			"le":          {GoFunc: "__gopy_operator_le", Helper: helperOpLe, RetKind: "bool"},
			"eq":          {GoFunc: "__gopy_operator_eq", Helper: helperOpEq, RetKind: "bool"},
			"ne":          {GoFunc: "__gopy_operator_ne", Helper: helperOpNe, RetKind: "bool"},
			"gt":          {GoFunc: "__gopy_operator_gt", Helper: helperOpGt, RetKind: "bool"},
			"ge":          {GoFunc: "__gopy_operator_ge", Helper: helperOpGe, RetKind: "bool"},
			"not_":        {GoFunc: "__gopy_operator_not", Helper: helperOpNot, RetKind: "bool"},
			"truth":       {GoFunc: "__gopy_operator_truth", Helper: helperOpTruth, RetKind: "bool"},
			"is_":         {GoFunc: "__gopy_operator_is", Helper: helperOpIs, RetKind: "bool"},
			"is_not":      {GoFunc: "__gopy_operator_isnot", Helper: helperOpIsnot, RetKind: "bool"},
			"itemgetter":   {GoFunc: "__gopy_operator_itemgetter", Helper: helperOpItemgetter},
			"attrgetter":   {GoFunc: "__gopy_operator_attrgetter", Helper: helperOpAttrgetter},
			"methodcaller": {GoFunc: "__gopy_operator_methodcaller", Helper: helperOpMethodcaller},
			"length_hint":  {GoFunc: "__gopy_operator_length_hint", Helper: helperOpLengthHint, RetKind: "int"},
			"index":        {GoFunc: "__gopy_operator_index", Helper: helperOpIndex, RetKind: "int"},
			"indexOf":      {GoFunc: "__gopy_operator_indexof_unused"},
			"countOf":      {GoFunc: "__gopy_operator_countof_unused"},
			"concat":       {GoFunc: "__gopy_operator_concat_unused"},
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
		Funcs: map[string]stdlibFunc{
			"BaseSelector":    {GoFunc: "__gopy_selectors_unused"},
			"DefaultSelector": {GoFunc: "__gopy_selectors_unused"},
			"SelectSelector":  {GoFunc: "__gopy_selectors_unused"},
			"PollSelector":    {GoFunc: "__gopy_selectors_unused"},
			"EpollSelector":   {GoFunc: "__gopy_selectors_unused"},
			"DevpollSelector": {GoFunc: "__gopy_selectors_unused"},
			"KqueueSelector":  {GoFunc: "__gopy_selectors_unused"},
			"SelectorKey":     {GoFunc: "__gopy_selectors_unused"},
		},
	},
	"errno": {
		Attrs: map[string]stdlibAttr{
			"EACCES":          {GoExpr: "int64(syscall.EACCES)", GoImport: "syscall"},
			"EBADF":           {GoExpr: "int64(syscall.EBADF)", GoImport: "syscall"},
			"EBUSY":           {GoExpr: "int64(syscall.EBUSY)", GoImport: "syscall"},
			"ECONNREFUSED":    {GoExpr: "int64(syscall.ECONNREFUSED)", GoImport: "syscall"},
			"EEXIST":          {GoExpr: "int64(syscall.EEXIST)", GoImport: "syscall"},
			"EINTR":           {GoExpr: "int64(syscall.EINTR)", GoImport: "syscall"},
			"EINVAL":          {GoExpr: "int64(syscall.EINVAL)", GoImport: "syscall"},
			"EIO":             {GoExpr: "int64(syscall.EIO)", GoImport: "syscall"},
			"EISDIR":          {GoExpr: "int64(syscall.EISDIR)", GoImport: "syscall"},
			"ENOENT":          {GoExpr: "int64(syscall.ENOENT)", GoImport: "syscall"},
			"ENOMEM":          {GoExpr: "int64(syscall.ENOMEM)", GoImport: "syscall"},
			"ENOSPC":          {GoExpr: "int64(syscall.ENOSPC)", GoImport: "syscall"},
			"ENOTDIR":         {GoExpr: "int64(syscall.ENOTDIR)", GoImport: "syscall"},
			"EPERM":           {GoExpr: "int64(syscall.EPERM)", GoImport: "syscall"},
			"EPIPE":           {GoExpr: "int64(syscall.EPIPE)", GoImport: "syscall"},
			"ETIMEDOUT":       {GoExpr: "int64(syscall.ETIMEDOUT)", GoImport: "syscall"},
			"EAGAIN":          {GoExpr: "int64(syscall.EAGAIN)", GoImport: "syscall"},
			"EWOULDBLOCK":     {GoExpr: "int64(syscall.EWOULDBLOCK)", GoImport: "syscall"},
			"ECHILD":          {GoExpr: "int64(syscall.ECHILD)", GoImport: "syscall"},
			"EFAULT":          {GoExpr: "int64(syscall.EFAULT)", GoImport: "syscall"},
			"ENFILE":          {GoExpr: "int64(syscall.ENFILE)", GoImport: "syscall"},
			"EMFILE":          {GoExpr: "int64(syscall.EMFILE)", GoImport: "syscall"},
			"ENOTTY":          {GoExpr: "int64(syscall.ENOTTY)", GoImport: "syscall"},
			"EXDEV":           {GoExpr: "int64(syscall.EXDEV)", GoImport: "syscall"},
			"EROFS":           {GoExpr: "int64(syscall.EROFS)", GoImport: "syscall"},
			"ESPIPE":          {GoExpr: "int64(syscall.ESPIPE)", GoImport: "syscall"},
			"ENXIO":           {GoExpr: "int64(syscall.ENXIO)", GoImport: "syscall"},
			"EDOM":            {GoExpr: "int64(syscall.EDOM)", GoImport: "syscall"},
			"ERANGE":          {GoExpr: "int64(syscall.ERANGE)", GoImport: "syscall"},
			"ECONNRESET":      {GoExpr: "int64(syscall.ECONNRESET)", GoImport: "syscall"},
			"ECONNABORTED":    {GoExpr: "int64(syscall.ECONNABORTED)", GoImport: "syscall"},
			"EADDRINUSE":      {GoExpr: "int64(syscall.EADDRINUSE)", GoImport: "syscall"},
			"EADDRNOTAVAIL":   {GoExpr: "int64(syscall.EADDRNOTAVAIL)", GoImport: "syscall"},
			"EHOSTUNREACH":    {GoExpr: "int64(syscall.EHOSTUNREACH)", GoImport: "syscall"},
			"EHOSTDOWN":       {GoExpr: "int64(syscall.EHOSTDOWN)", GoImport: "syscall"},
			"ENETUNREACH":     {GoExpr: "int64(syscall.ENETUNREACH)", GoImport: "syscall"},
			"ENETDOWN":        {GoExpr: "int64(syscall.ENETDOWN)", GoImport: "syscall"},
			"EINPROGRESS":     {GoExpr: "int64(syscall.EINPROGRESS)", GoImport: "syscall"},
			"EALREADY":        {GoExpr: "int64(syscall.EALREADY)", GoImport: "syscall"},
			"EISCONN":         {GoExpr: "int64(syscall.EISCONN)", GoImport: "syscall"},
			"ENOTCONN":        {GoExpr: "int64(syscall.ENOTCONN)", GoImport: "syscall"},
			"ENOTSOCK":        {GoExpr: "int64(syscall.ENOTSOCK)", GoImport: "syscall"},
			"EAFNOSUPPORT":    {GoExpr: "int64(syscall.EAFNOSUPPORT)", GoImport: "syscall"},
			"EPROTOTYPE":      {GoExpr: "int64(syscall.EPROTOTYPE)", GoImport: "syscall"},
			"EPROTONOSUPPORT": {GoExpr: "int64(syscall.EPROTONOSUPPORT)", GoImport: "syscall"},
			"EOPNOTSUPP":      {GoExpr: "int64(syscall.EOPNOTSUPP)", GoImport: "syscall"},
			"EPFNOSUPPORT":    {GoExpr: "int64(syscall.EPFNOSUPPORT)", GoImport: "syscall"},
			"ELOOP":           {GoExpr: "int64(syscall.ELOOP)", GoImport: "syscall"},
			"ENAMETOOLONG":    {GoExpr: "int64(syscall.ENAMETOOLONG)", GoImport: "syscall"},
			"ENOTEMPTY":       {GoExpr: "int64(syscall.ENOTEMPTY)", GoImport: "syscall"},
			"EDQUOT":          {GoExpr: "int64(syscall.EDQUOT)", GoImport: "syscall"},
			"EOVERFLOW":       {GoExpr: "int64(syscall.EOVERFLOW)", GoImport: "syscall"},
		},
	},
	"stat": {
		Attrs: map[string]stdlibAttr{
			"S_IFREG":  {GoExpr: "int64(0o100000)"},
			"S_IFDIR":  {GoExpr: "int64(0o040000)"},
			"S_IFLNK":  {GoExpr: "int64(0o120000)"},
			"S_IFCHR":  {GoExpr: "int64(0o020000)"},
			"S_IFBLK":  {GoExpr: "int64(0o060000)"},
			"S_IFIFO":  {GoExpr: "int64(0o010000)"},
			"S_IFSOCK": {GoExpr: "int64(0o140000)"},
			"S_IRUSR":  {GoExpr: "int64(0o400)"},
			"S_IWUSR":  {GoExpr: "int64(0o200)"},
			"S_IXUSR":  {GoExpr: "int64(0o100)"},
			"S_IRGRP":  {GoExpr: "int64(0o040)"},
			"S_IWGRP":  {GoExpr: "int64(0o020)"},
			"S_IXGRP":  {GoExpr: "int64(0o010)"},
			"S_IROTH":  {GoExpr: "int64(0o004)"},
			"S_IWOTH":  {GoExpr: "int64(0o002)"},
			"S_IXOTH":  {GoExpr: "int64(0o001)"},
			"S_ISUID":  {GoExpr: "int64(0o4000)"},
			"S_ISGID":  {GoExpr: "int64(0o2000)"},
			"S_ISVTX":  {GoExpr: "int64(0o1000)"},
			"S_IRWXU":  {GoExpr: "int64(0o700)"},
			"S_IRWXG":  {GoExpr: "int64(0o070)"},
			"S_IRWXO":  {GoExpr: "int64(0o007)"},
		},
		Funcs: map[string]stdlibFunc{
			"S_ISREG":  {GoFunc: "__gopy_stat_isreg", Helper: helperStatIsreg, RetKind: "bool"},
			"S_ISDIR":  {GoFunc: "__gopy_stat_isdir", Helper: helperStatIsdir, RetKind: "bool"},
			"S_ISLNK":  {GoFunc: "__gopy_stat_islnk", Helper: helperStatIslnk, RetKind: "bool"},
			"S_ISCHR":  {GoFunc: "__gopy_stat_ischr", Helper: helperStatIschr, RetKind: "bool"},
			"S_ISBLK":  {GoFunc: "__gopy_stat_isblk", Helper: helperStatIsblk, RetKind: "bool"},
			"S_ISFIFO": {GoFunc: "__gopy_stat_isfifo", Helper: helperStatIsfifo, RetKind: "bool"},
			"S_ISSOCK": {GoFunc: "__gopy_stat_issock", Helper: helperStatIssock, RetKind: "bool"},
			"S_IMODE":  {GoFunc: "__gopy_stat_imode", Helper: helperStatImode, RetKind: "int"},
			"S_IFMT":   {GoFunc: "__gopy_stat_ifmt", Helper: helperStatIfmt, RetKind: "int"},
			"filemode": {GoFunc: "__gopy_stat_filemode", Helper: helperStatFilemode, RetKind: "str"},
		},
	},
	"fnmatch": {
		Funcs: map[string]stdlibFunc{
			"translate":   {GoFunc: "__gopy_fnmatch_translate", Helper: helperFnmatchTranslate, HelperImports: []string{"strings"}, RetKind: "str"},
			"fnmatch":     {GoFunc: "__gopy_fnmatch", Helper: helperFnmatch, HelperImports: []string{"path/filepath"}, RetKind: "bool"},
			"fnmatchcase": {GoFunc: "__gopy_fnmatch", Helper: helperFnmatch, HelperImports: []string{"path/filepath"}, RetKind: "bool"},
			"filter":      {GoFunc: "__gopy_fnmatch_filter", Helper: helperFnmatchFilter, HelperImports: []string{"path/filepath"}},
		},
	},
	"linecache": {
		Funcs: map[string]stdlibFunc{
			"getline":  {GoFunc: "__gopy_linecache_getline", Helper: helperLinecacheGetline, HelperImports: []string{"bufio", "os"}, RetKind: "str"},
			"getlines": {GoFunc: "__gopy_linecache_getlines", Helper: helperLinecacheGetlines, HelperImports: []string{"bufio", "os"}},
			"clearcache": {GoFunc: "__gopy_warnings_noop", Helper: helperWarningsNoop},
			"checkcache": {GoFunc: "__gopy_warnings_noop", Helper: helperWarningsNoop},
		},
	},
	"getopt": {
		Funcs: map[string]stdlibFunc{
			"getopt":       {GoFunc: "__gopy_getopt", Helper: helperGetopt, HelperImports: []string{"strings"}},
			"gnu_getopt":   {GoFunc: "__gopy_getopt", Helper: helperGetopt, HelperImports: []string{"strings"}},
			"GetoptError":  {GoFunc: "__gopy_getopt_err_unused"},
		},
	},
	"timeit": {
		Funcs: map[string]stdlibFunc{
			"default_timer": {GoFunc: "__gopy_time_monotonic", GoImport: "time", Helper: helperTimeMonotonic, RetKind: "float"},
			"timeit":        {GoFunc: "__gopy_timeit_stub", Helper: helperTimeitStub, RetKind: "float"},
		},
	},
	"cProfile": {
		Funcs: map[string]stdlibFunc{
			"run":     {GoFunc: "__gopy_warnings_noop", Helper: helperWarningsNoop},
			"runctx":  {GoFunc: "__gopy_warnings_noop", Helper: helperWarningsNoop},
		},
	},
	"profile": {
		Funcs: map[string]stdlibFunc{
			"run":    {GoFunc: "__gopy_warnings_noop", Helper: helperWarningsNoop},
			"runctx": {GoFunc: "__gopy_warnings_noop", Helper: helperWarningsNoop},
		},
	},
	"pdb": {
		Funcs: map[string]stdlibFunc{
			"set_trace":  {GoFunc: "__gopy_warnings_noop", Helper: helperWarningsNoop},
			"post_mortem": {GoFunc: "__gopy_warnings_noop", Helper: helperWarningsNoop},
		},
	},
	"posixpath": {
		Funcs: map[string]stdlibFunc{
			"join":     {GoFunc: "__gopy_path_join", GoImport: "path/filepath", Helper: helperPathJoin, RetKind: "str"},
			"basename": {GoFunc: "filepath.Base", GoImport: "path/filepath", RetKind: "str"},
			"dirname":  {GoFunc: "filepath.Dir", GoImport: "path/filepath", RetKind: "str"},
			"split":    {GoFunc: "__gopy_path_split", GoImport: "path/filepath", Helper: helperPathSplit},
			"splitext": {GoFunc: "__gopy_path_splitext", GoImport: "path/filepath", Helper: helperPathSplitext},
		},
	},
	"warnings": {
		Funcs: map[string]stdlibFunc{
			"warn":             {GoFunc: "__gopy_warnings_warn", Helper: helperWarningsWarn, HelperImports: []string{"fmt", "os"}},
			"filterwarnings":   {GoFunc: "__gopy_warnings_noop", Helper: helperWarningsNoop},
			"simplefilter":     {GoFunc: "__gopy_warnings_noop", Helper: helperWarningsNoop},
			"resetwarnings":    {GoFunc: "__gopy_warnings_noop", Helper: helperWarningsNoop},
			"showwarning":      {GoFunc: "__gopy_warnings_warn", Helper: helperWarningsWarn, HelperImports: []string{"fmt", "os"}},
		},
	},
	"gettext": {
		Funcs: map[string]stdlibFunc{
			"gettext":  {GoFunc: "__gopy_gettext_identity", Helper: helperGettextIdentity, RetKind: "str"},
			"ngettext": {GoFunc: "__gopy_gettext_n", Helper: helperGettextN, RetKind: "str"},
			"install":  {GoFunc: "__gopy_warnings_noop", Helper: helperWarningsNoop},
		},
	},
	"locale": {
		Attrs: map[string]stdlibAttr{
			"LC_ALL":      {GoExpr: "int64(6)"},
			"LC_COLLATE":  {GoExpr: "int64(3)"},
			"LC_CTYPE":    {GoExpr: "int64(0)"},
			"LC_MONETARY": {GoExpr: "int64(4)"},
			"LC_NUMERIC":  {GoExpr: "int64(1)"},
			"LC_TIME":     {GoExpr: "int64(2)"},
		},
		Funcs: map[string]stdlibFunc{
			"setlocale":   {GoFunc: "__gopy_locale_setlocale", Helper: helperLocaleSetlocale, RetKind: "str"},
			"getlocale":   {GoFunc: "__gopy_locale_getlocale", Helper: helperLocaleGetlocale},
			"getdefaultlocale": {GoFunc: "__gopy_locale_getlocale", Helper: helperLocaleGetlocale},
		},
	},
	"colorsys": {
		Funcs: map[string]stdlibFunc{
			"rgb_to_hsv": {GoFunc: "__gopy_colorsys_rgb_hsv", Helper: helperColorsysRgbHsv, HelperImports: []string{"math"}},
			"hsv_to_rgb": {GoFunc: "__gopy_colorsys_hsv_rgb", Helper: helperColorsysHsvRgb, HelperImports: []string{"math"}},
			"rgb_to_yiq": {GoFunc: "__gopy_colorsys_rgb_yiq", Helper: helperColorsysRgbYiq},
			"yiq_to_rgb": {GoFunc: "__gopy_colorsys_yiq_rgb", Helper: helperColorsysYiqRgb},
		},
	},
	"keyword": {
		Attrs: map[string]stdlibAttr{
			"kwlist": {GoExpr: `[]string{"False","None","True","and","as","assert","async","await","break","class","continue","def","del","elif","else","except","finally","for","from","global","if","import","in","is","lambda","nonlocal","not","or","pass","raise","return","try","while","with","yield"}`},
			"softkwlist": {GoExpr: `[]string{"match","case","type","_"}`},
		},
		Funcs: map[string]stdlibFunc{
			"iskeyword":     {GoFunc: "__gopy_keyword_iskw", Helper: helperKeywordIskw, RetKind: "bool"},
			"issoftkeyword": {GoFunc: "__gopy_keyword_issoftkw", Helper: helperKeywordIssoftkw, RetKind: "bool"},
		},
	},
	"unicodedata": {
		Funcs: map[string]stdlibFunc{
			"category": {GoFunc: "__gopy_unicodedata_category", Helper: helperUnicodedataCategory, HelperImports: []string{"unicode"}, RetKind: "str"},
			"name":     {GoFunc: "__gopy_unicodedata_name", Helper: helperUnicodedataName, RetKind: "str"},
		},
	},
	"dis": {
		Funcs: map[string]stdlibFunc{
			"dis":              {GoFunc: "__gopy_dis_noop", Helper: helperDisNoop},
			"disassemble":      {GoFunc: "__gopy_dis_noop", Helper: helperDisNoop},
			"get_instructions": {GoFunc: "__gopy_dis_instr", Helper: helperDisInstr},
			"code_info":        {GoFunc: "__gopy_dis_codeinfo", Helper: helperDisCodeInfo, RetKind: "str"},
			"show_code":        {GoFunc: "__gopy_dis_noop", Helper: helperDisNoop},
			"Bytecode":         {GoFunc: "__gopy_dis_bytecode_unused"},
			"Instruction":      {GoFunc: "__gopy_dis_instruction_unused"},
		},
	},
	"tracemalloc": {
		Funcs: map[string]stdlibFunc{
			"start":           {GoFunc: "__gopy_warnings_noop", Helper: helperWarningsNoop},
			"stop":            {GoFunc: "__gopy_warnings_noop", Helper: helperWarningsNoop},
			"is_tracing":      {GoFunc: "__gopy_dis_isfalse", Helper: helperDisIsfalse, RetKind: "bool"},
			"get_traced_memory": {GoFunc: "__gopy_dis_traced_mem", Helper: helperDisTracedMem},
		},
	},
	"signal": {
		Attrs: map[string]stdlibAttr{
			"SIGINT":   {GoExpr: "int64(syscall.SIGINT)", GoImport: "syscall"},
			"SIGTERM":  {GoExpr: "int64(syscall.SIGTERM)", GoImport: "syscall"},
			"SIGHUP":   {GoExpr: "int64(syscall.SIGHUP)", GoImport: "syscall"},
			"SIGQUIT":  {GoExpr: "int64(syscall.SIGQUIT)", GoImport: "syscall"},
			"SIGKILL":  {GoExpr: "int64(syscall.SIGKILL)", GoImport: "syscall"},
			"SIGUSR1":  {GoExpr: "int64(syscall.SIGUSR1)", GoImport: "syscall"},
			"SIGUSR2":  {GoExpr: "int64(syscall.SIGUSR2)", GoImport: "syscall"},
			"SIGCHLD":  {GoExpr: "int64(syscall.SIGCHLD)", GoImport: "syscall"},
			"SIGSTOP":  {GoExpr: "int64(syscall.SIGSTOP)", GoImport: "syscall"},
			"SIGCONT":  {GoExpr: "int64(syscall.SIGCONT)", GoImport: "syscall"},
			"SIGPIPE":  {GoExpr: "int64(syscall.SIGPIPE)", GoImport: "syscall"},
			"SIGALRM":  {GoExpr: "int64(syscall.SIGALRM)", GoImport: "syscall"},
			"SIGSEGV":  {GoExpr: "int64(syscall.SIGSEGV)", GoImport: "syscall"},
			"SIGFPE":   {GoExpr: "int64(syscall.SIGFPE)", GoImport: "syscall"},
			"SIGBUS":   {GoExpr: "int64(syscall.SIGBUS)", GoImport: "syscall"},
			"SIGABRT":  {GoExpr: "int64(syscall.SIGABRT)", GoImport: "syscall"},
			"SIGILL":   {GoExpr: "int64(syscall.SIGILL)", GoImport: "syscall"},
			"SIGTRAP":  {GoExpr: "int64(syscall.SIGTRAP)", GoImport: "syscall"},
			"SIGTSTP":  {GoExpr: "int64(syscall.SIGTSTP)", GoImport: "syscall"},
			"SIGTTIN":  {GoExpr: "int64(syscall.SIGTTIN)", GoImport: "syscall"},
			"SIGTTOU":  {GoExpr: "int64(syscall.SIGTTOU)", GoImport: "syscall"},
			"SIGURG":   {GoExpr: "int64(syscall.SIGURG)", GoImport: "syscall"},
			"SIGXCPU":  {GoExpr: "int64(syscall.SIGXCPU)", GoImport: "syscall"},
			"SIGXFSZ":  {GoExpr: "int64(syscall.SIGXFSZ)", GoImport: "syscall"},
			"SIGVTALRM": {GoExpr: "int64(syscall.SIGVTALRM)", GoImport: "syscall"},
			"SIGPROF":  {GoExpr: "int64(syscall.SIGPROF)", GoImport: "syscall"},
			"SIGWINCH": {GoExpr: "int64(syscall.SIGWINCH)", GoImport: "syscall"},
			"SIGIO":    {GoExpr: "int64(syscall.SIGIO)", GoImport: "syscall"},
			"SIGSYS":   {GoExpr: "int64(syscall.SIGSYS)", GoImport: "syscall"},
			"NSIG":     {GoExpr: "int64(65)"},
			"SIG_DFL":  {GoExpr: "any(0)"},
			"SIG_IGN":  {GoExpr: "any(1)"},
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
			"run":             {GoFunc: "__gopy_asyncio_run_unused"},
			"sleep":           {GoFunc: "__gopy_asyncio_sleep_unused"},
			"gather":          {GoFunc: "__gopy_asyncio_gather_unused"},
			"create_task":     {GoFunc: "__gopy_asyncio_create_task_unused"},
			"wait":            {GoFunc: "__gopy_asyncio_wait_unused"},
			"wait_for":        {GoFunc: "__gopy_asyncio_wait_for_unused"},
			"shield":          {GoFunc: "__gopy_asyncio_shield_unused"},
			"as_completed":    {GoFunc: "__gopy_asyncio_as_completed_unused"},
			"ensure_future":   {GoFunc: "__gopy_asyncio_ensure_future_unused"},
			"current_task":    {GoFunc: "__gopy_asyncio_current_task_unused"},
			"all_tasks":       {GoFunc: "__gopy_asyncio_all_tasks_unused"},
			"get_event_loop":  {GoFunc: "__gopy_asyncio_get_loop_unused"},
			"new_event_loop":  {GoFunc: "__gopy_asyncio_new_loop_unused"},
			"set_event_loop":  {GoFunc: "__gopy_asyncio_set_loop_unused"},
			"get_running_loop": {GoFunc: "__gopy_asyncio_running_loop_unused"},
			"Lock":            {GoFunc: "__gopy_asyncio_lock_unused"},
			"Event":           {GoFunc: "__gopy_asyncio_event_unused"},
			"Condition":       {GoFunc: "__gopy_asyncio_cond_unused"},
			"Semaphore":       {GoFunc: "__gopy_asyncio_sem_unused"},
			"BoundedSemaphore": {GoFunc: "__gopy_asyncio_sem_unused"},
			"Queue":           {GoFunc: "__gopy_asyncio_queue_unused"},
			"LifoQueue":       {GoFunc: "__gopy_asyncio_queue_unused"},
			"PriorityQueue":   {GoFunc: "__gopy_asyncio_queue_unused"},
			"Task":            {GoFunc: "__gopy_asyncio_task_unused"},
			"Future":          {GoFunc: "__gopy_asyncio_future_unused"},
			"AbstractEventLoop": {GoFunc: "__gopy_asyncio_abstract_loop_unused"},
			"CancelledError":  {GoFunc: "__gopy_asyncio_cancel_err_unused"},
			"TimeoutError":    {GoFunc: "__gopy_asyncio_timeout_err_unused"},
			"InvalidStateError": {GoFunc: "__gopy_asyncio_state_err_unused"},
			"open_connection": {GoFunc: "__gopy_asyncio_open_conn_unused"},
			"start_server":    {GoFunc: "__gopy_asyncio_start_server_unused"},
			"iscoroutine":     {GoFunc: "__gopy_asyncio_iscoro_unused"},
			"iscoroutinefunction": {GoFunc: "__gopy_asyncio_iscoro_fn_unused"},
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
		Subs: map[string]stdlibModule{
			"entities": {
				Attrs: map[string]stdlibAttr{
					"html5":          {GoExpr: `map[string]string{"amp;": "&", "lt;": "<", "gt;": ">", "quot;": "\"", "apos;": "'", "nbsp;": " "}`},
					"name2codepoint": {GoExpr: `map[string]int64{"amp": 38, "lt": 60, "gt": 62, "quot": 34, "apos": 39, "nbsp": 160, "copy": 169, "reg": 174, "trade": 8482, "euro": 8364, "pound": 163, "yen": 165, "cent": 162}`},
				},
			},
			"parser": {
				Funcs: map[string]stdlibFunc{
					"HTMLParser": {GoFunc: "__gopy_html_parser_unused"},
				},
			},
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
			"glob":      {GoFunc: "__gopy_glob", GoImport: "path/filepath", Helper: helperGlob},
			"iglob":     {GoFunc: "__gopy_glob", GoImport: "path/filepath", Helper: helperGlob},
			"has_magic": {GoFunc: "__gopy_glob_has_magic", Helper: helperGlobHasMagic, HelperImports: []string{"strings"}, RetKind: "bool"},
			"escape":    {GoFunc: "__gopy_glob_escape", Helper: helperGlobEscape, HelperImports: []string{"strings"}, RetKind: "str"},
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
			"isleap":        {GoFunc: "__gopy_cal_isleap", Helper: helperCalIsleap, RetKind: "bool"},
			"monthrange":    {GoFunc: "__gopy_cal_monthrange", GoImport: "time", Helper: helperCalMonthrange},
			"weekday":       {GoFunc: "__gopy_cal_weekday", GoImport: "time", Helper: helperCalWeekday, RetKind: "int"},
			"monthcalendar": {GoFunc: "__gopy_cal_monthcal", GoImport: "time", Helper: helperCalMonthcalendar, ExtraHelpers: map[string]string{"__gopy_cal_monthrange": helperCalMonthrange}},
			"leapdays":      {GoFunc: "__gopy_cal_leapdays", Helper: helperCalLeapdays, ExtraHelpers: map[string]string{"__gopy_cal_isleap": helperCalIsleap}, RetKind: "int"},
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
			"IPPROTO_TCP":  {GoExpr: "int64(6)"},
			"IPPROTO_UDP":  {GoExpr: "int64(17)"},
			"IPPROTO_IP":   {GoExpr: "int64(0)"},
			"IPPROTO_ICMP": {GoExpr: "int64(1)"},
			"TCP_NODELAY":  {GoExpr: "int64(1)"},
			"SOCK_RAW":     {GoExpr: "int64(3)"},
			"SOMAXCONN":    {GoExpr: "int64(128)"},
			"AI_PASSIVE":   {GoExpr: "int64(1)"},
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
			"if_nameindex":     {GoFunc: "__gopy_socket_if_nameindex", Helper: helperSocketIfNameindex, HelperImports: []string{"net"}},
			"if_indextoname":   {GoFunc: "__gopy_socket_if_indextoname", Helper: helperSocketIfIndextoname, HelperImports: []string{"net"}, RetKind: "str"},
			"if_nametoindex":   {GoFunc: "__gopy_socket_if_nametoindex", Helper: helperSocketIfNametoindex, HelperImports: []string{"net"}, RetKind: "int"},
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
		Attrs: map[string]stdlibAttr{
			"MISSING":          {GoExpr: "any(nil)"},
			"KW_ONLY":          {GoExpr: "any(nil)"},
		},
		Funcs: map[string]stdlibFunc{
			"asdict":         {GoFunc: "__gopy_asdict_unused"},
			"astuple":        {GoFunc: "__gopy_astuple_unused"},
			"replace":        {GoFunc: "__gopy_replace_unused"},
			"fields":         {GoFunc: "__gopy_fields_unused"},
			"field":          {GoFunc: "__gopy_dc_field_unused"},
			"make_dataclass": {GoFunc: "__gopy_dc_make_unused"},
			"is_dataclass":   {GoFunc: "__gopy_dc_is_unused"},
			"InitVar":        {GoFunc: "__gopy_dc_initvar_unused"},
			"FrozenInstanceError": {GoFunc: "__gopy_dc_frozen_err_unused"},
		},
	},
	"hmac": {
		Funcs: map[string]stdlibFunc{
			"new":     {GoFunc: "__gopy_hmac_new", GoImport: "crypto/hmac", Helper: helperHmacNew, RetTag: "__Hmac", ExtraHelpers: map[string]string{"__Hmac": helperHmacType}, HelperImports: []string{"crypto/sha1", "crypto/sha256", "crypto/sha512", "crypto/md5", "hash", "encoding/hex"}},
			"compare_digest": {GoFunc: "__gopy_hmac_cmp", GoImport: "crypto/hmac", Helper: helperHmacCompare, RetKind: "bool"},
		},
	},
	"subprocess": {
		Attrs: map[string]stdlibAttr{
			"PIPE":    {GoExpr: "int64(-1)"},
			"STDOUT":  {GoExpr: "int64(-2)"},
			"DEVNULL": {GoExpr: "int64(-3)"},
		},
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
		Attrs: map[string]stdlibAttr{
			"WRAPPER_ASSIGNMENTS": {GoExpr: `[]string{"__module__", "__name__", "__qualname__", "__annotations__", "__doc__"}`},
			"WRAPPER_UPDATES":     {GoExpr: `[]string{"__dict__"}`},
		},
		Funcs: map[string]stdlibFunc{
			// reduce uses an inline lambda for the binary op; dispatch
			// lives in transpile.go's call() builder.
			"reduce":          {GoFunc: "__gopy_reduce_unused"},
			"partial":         {GoFunc: "__gopy_partial_unused"},
			"cache":           {GoFunc: "__gopy_cache_unused"},
			"cached_property": {GoFunc: "__gopy_cached_prop_unused"},
			"wraps":           {GoFunc: "__gopy_wraps_unused"},
			"singledispatch":  {GoFunc: "__gopy_singledispatch_unused"},
			"cmp_to_key":      {GoFunc: "__gopy_cmp_to_key_unused"},
			"total_ordering":  {GoFunc: "__gopy_total_ordering", Helper: helperFunctoolsTotalOrdering},
			"update_wrapper":  {GoFunc: "__gopy_update_wrapper", Helper: helperFunctoolsUpdateWrapper},
			"lru_cache":       {GoFunc: "__gopy_lru_cache_unused"},
			"reduce_unused":   {GoFunc: "__gopy_reduce_unused"},
		},
	},
	"logging": {
		Attrs: map[string]stdlibAttr{
			"DEBUG":    {GoExpr: "int64(10)"},
			"INFO":     {GoExpr: "int64(20)"},
			"WARNING":  {GoExpr: "int64(30)"},
			"WARN":     {GoExpr: "int64(30)"},
			"ERROR":    {GoExpr: "int64(40)"},
			"CRITICAL": {GoExpr: "int64(50)"},
			"FATAL":    {GoExpr: "int64(50)"},
			"NOTSET":   {GoExpr: "int64(0)"},
		},
		Funcs: map[string]stdlibFunc{
			"debug":         {GoFunc: "__gopy_log_debug", GoImport: "fmt", Helper: helperLogDebug, ExtraHelpers: map[string]string{"__gopy_log_state": helperLogState}, HelperImports: []string{"os"}},
			"info":          {GoFunc: "__gopy_log_info", GoImport: "fmt", Helper: helperLogInfo, ExtraHelpers: map[string]string{"__gopy_log_state": helperLogState}, HelperImports: []string{"os"}},
			"warning":       {GoFunc: "__gopy_log_warning", GoImport: "fmt", Helper: helperLogWarning, ExtraHelpers: map[string]string{"__gopy_log_state": helperLogState}, HelperImports: []string{"os"}},
			"error":         {GoFunc: "__gopy_log_error", GoImport: "fmt", Helper: helperLogError, ExtraHelpers: map[string]string{"__gopy_log_state": helperLogState}, HelperImports: []string{"os"}},
			"critical":      {GoFunc: "__gopy_log_critical", GoImport: "fmt", Helper: helperLogCritical, ExtraHelpers: map[string]string{"__gopy_log_state": helperLogState}, HelperImports: []string{"os"}},
			"basicConfig":   {GoFunc: "__gopy_log_basicConfig", Helper: helperLogBasicConfig, ExtraHelpers: map[string]string{"__gopy_log_state": helperLogState}},
			"getLogger":     {GoFunc: "__gopy_log_getlogger", GoImport: "fmt", Helper: helperLogGetLogger, RetTag: "__Logger", ExtraHelpers: map[string]string{"__Logger": helperLoggerType, "__gopy_log_state": helperLogState}, HelperImports: []string{"os"}},
			"Handler":       {GoFunc: "__gopy_logging_handler_unused"},
			"StreamHandler": {GoFunc: "__gopy_logging_handler_unused"},
			"FileHandler":   {GoFunc: "__gopy_logging_handler_unused"},
			"NullHandler":   {GoFunc: "__gopy_logging_handler_unused"},
			"Formatter":     {GoFunc: "__gopy_logging_handler_unused"},
			"Filter":        {GoFunc: "__gopy_logging_handler_unused"},
			"LogRecord":     {GoFunc: "__gopy_logging_handler_unused"},
			"Logger":        {GoFunc: "__gopy_logging_handler_unused"},
		},
		Subs: map[string]stdlibModule{
			"handlers": {
				Funcs: map[string]stdlibFunc{
					"RotatingFileHandler":   {GoFunc: "__gopy_logging_handler_unused"},
					"TimedRotatingFileHandler": {GoFunc: "__gopy_logging_handler_unused"},
					"WatchedFileHandler":    {GoFunc: "__gopy_logging_handler_unused"},
					"SysLogHandler":         {GoFunc: "__gopy_logging_handler_unused"},
					"SocketHandler":         {GoFunc: "__gopy_logging_handler_unused"},
					"DatagramHandler":       {GoFunc: "__gopy_logging_handler_unused"},
					"SMTPHandler":           {GoFunc: "__gopy_logging_handler_unused"},
					"MemoryHandler":         {GoFunc: "__gopy_logging_handler_unused"},
					"HTTPHandler":           {GoFunc: "__gopy_logging_handler_unused"},
					"QueueHandler":          {GoFunc: "__gopy_logging_handler_unused"},
					"QueueListener":         {GoFunc: "__gopy_logging_handler_unused"},
					"NTEventLogHandler":     {GoFunc: "__gopy_logging_handler_unused"},
					"BufferingHandler":      {GoFunc: "__gopy_logging_handler_unused"},
				},
			},
			"config": {
				Funcs: map[string]stdlibFunc{
					"fileConfig":   {GoFunc: "__gopy_logging_config_unused"},
					"dictConfig":   {GoFunc: "__gopy_logging_config_unused"},
					"listen":       {GoFunc: "__gopy_logging_config_unused"},
					"stopListening": {GoFunc: "__gopy_logging_config_unused"},
				},
			},
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
			"choices": {GoFunc: "__gopy_random_choices_unused"},
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
			"correlation":   {GoFunc: "__gopy_stats_correlation", Helper: helperStatsCorrelation, HelperImports: []string{"math"}, RetKind: "float"},
			"covariance":    {GoFunc: "__gopy_stats_covariance", Helper: helperStatsCovariance, RetKind: "float"},
			"geometric_mean":    {GoFunc: "__gopy_stats_geomean", Helper: helperStatsGeoMean, HelperImports: []string{"math"}, RetKind: "float"},
			"quantiles":         {GoFunc: "__gopy_stats_quantiles", Helper: helperStatsQuantiles, HelperImports: []string{"sort"}},
			"linear_regression": {GoFunc: "__gopy_stats_linreg", Helper: helperStatsLinreg},
		},
	},
	"uuid": {
		Attrs: map[string]stdlibAttr{
			"NAMESPACE_DNS":  {GoExpr: `"6ba7b810-9dad-11d1-80b4-00c04fd430c8"`},
			"NAMESPACE_URL":  {GoExpr: `"6ba7b811-9dad-11d1-80b4-00c04fd430c8"`},
			"NAMESPACE_OID":  {GoExpr: `"6ba7b812-9dad-11d1-80b4-00c04fd430c8"`},
			"NAMESPACE_X500": {GoExpr: `"6ba7b814-9dad-11d1-80b4-00c04fd430c8"`},
		},
		Funcs: map[string]stdlibFunc{
			"uuid4": {GoFunc: "__gopy_uuid4", GoImport: "crypto/rand", Helper: helperUuid4, RetKind: "str", HelperImports: []string{"fmt"}},
			"uuid1": {GoFunc: "__gopy_uuid1", GoImport: "crypto/rand", Helper: helperUuid1, RetKind: "str", HelperImports: []string{"fmt", "time"}},
			"uuid3": {GoFunc: "__gopy_uuid3", Helper: helperUuid3, HelperImports: []string{"crypto/md5", "fmt"}, RetKind: "str"},
			"uuid5": {GoFunc: "__gopy_uuid5", Helper: helperUuid5, HelperImports: []string{"crypto/sha1", "fmt"}, RetKind: "str"},
		},
	},
	"textwrap": {
		Funcs: map[string]stdlibFunc{
			"dedent":      {GoFunc: "__gopy_textwrap_dedent", Helper: helperTextwrapDedent, RetKind: "str", HelperImports: []string{"strings"}},
			"indent":      {GoFunc: "__gopy_textwrap_indent", Helper: helperTextwrapIndent, RetKind: "str", HelperImports: []string{"strings"}},
			"fill":        {GoFunc: "__gopy_textwrap_fill", Helper: helperTextwrapFill, RetKind: "str", HelperImports: []string{"strings"}},
			"wrap":        {GoFunc: "__gopy_textwrap_wrap", Helper: helperTextwrapWrap, HelperImports: []string{"strings"}},
			"shorten":     {GoFunc: "__gopy_textwrap_shorten", Helper: helperTextwrapShorten, HelperImports: []string{"strings"}, RetKind: "str"},
			"TextWrapper": {GoFunc: "__gopy_textwrap_class_unused"},
		},
	},
	"re": {
		Attrs: map[string]stdlibAttr{
			"IGNORECASE": {GoExpr: "int64(2)"},
			"MULTILINE":  {GoExpr: "int64(8)"},
			"DOTALL":     {GoExpr: "int64(16)"},
			"VERBOSE":    {GoExpr: "int64(64)"},
			"ASCII":      {GoExpr: "int64(256)"},
			"UNICODE":    {GoExpr: "int64(32)"},
			"I":          {GoExpr: "int64(2)"},
			"M":          {GoExpr: "int64(8)"},
			"S":          {GoExpr: "int64(16)"},
			"X":          {GoExpr: "int64(64)"},
		},
		Funcs: map[string]stdlibFunc{
			"findall":   {GoFunc: "__gopy_re_findall", GoImport: "regexp", Helper: helperReFindall, ExtraHelpers: map[string]string{"__gopy_re_flag_prefix": helperReFlagPrefix}},
			"search":    {GoFunc: "__gopy_re_search", GoImport: "regexp", Helper: helperReSearch, RetTag: "__Match", ExtraHelpers: map[string]string{"__Match": helperMatchType, "__gopy_match_build": helperMatchBuild, "__gopy_re_flag_prefix": helperReFlagPrefix}},
			"match":     {GoFunc: "__gopy_re_match", GoImport: "regexp", Helper: helperReMatch, RetTag: "__Match", ExtraHelpers: map[string]string{"__Match": helperMatchType, "__gopy_match_build": helperMatchBuild, "__gopy_re_flag_prefix": helperReFlagPrefix}},
			"fullmatch": {GoFunc: "__gopy_re_fullmatch", GoImport: "regexp", Helper: helperReFullmatch, RetTag: "__Match", ExtraHelpers: map[string]string{"__Match": helperMatchType, "__gopy_match_build": helperMatchBuild, "__gopy_re_flag_prefix": helperReFlagPrefix}},
			"sub":       {GoFunc: "__gopy_re_sub", GoImport: "regexp", Helper: helperReSub, ExtraHelpers: map[string]string{"__gopy_re_flag_prefix": helperReFlagPrefix}},
			"subn":      {GoFunc: "__gopy_re_subn", GoImport: "regexp", Helper: helperReSubn, ExtraHelpers: map[string]string{"__gopy_re_flag_prefix": helperReFlagPrefix}},
			"split":     {GoFunc: "__gopy_re_split", GoImport: "regexp", Helper: helperReSplit, ExtraHelpers: map[string]string{"__gopy_re_flag_prefix": helperReFlagPrefix}},
			"escape":    {GoFunc: "regexp.QuoteMeta", GoImport: "regexp", RetKind: "str"},
			"compile":   {GoFunc: "__gopy_re_compile", GoImport: "regexp", Helper: helperReCompile, RetTag: "__Pattern", ExtraHelpers: map[string]string{"__Pattern": helperPatternType, "__Match": helperMatchType, "__gopy_match_build": helperMatchBuild, "__gopy_re_flag_prefix": helperReFlagPrefix}},
		},
	},
	"csv": {
		Attrs: map[string]stdlibAttr{
			"QUOTE_ALL":           {GoExpr: "int64(1)"},
			"QUOTE_MINIMAL":       {GoExpr: "int64(0)"},
			"QUOTE_NONNUMERIC":    {GoExpr: "int64(2)"},
			"QUOTE_NONE":          {GoExpr: "int64(3)"},
			"QUOTE_STRINGS":       {GoExpr: "int64(4)"},
			"QUOTE_NOTNULL":       {GoExpr: "int64(5)"},
		},
		Funcs: map[string]stdlibFunc{
			"list_dialects":      {GoFunc: "__gopy_csv_list_dialects", Helper: helperCSVListDialects},
			"register_dialect":   {GoFunc: "__gopy_csv_register_dialect", Helper: helperCSVRegisterDialect},
			"unregister_dialect": {GoFunc: "__gopy_csv_unregister_dialect", Helper: helperCSVUnregisterDialect},
			"get_dialect":        {GoFunc: "__gopy_csv_get_dialect", Helper: helperCSVGetDialect, RetKind: "str"},
			"field_size_limit":   {GoFunc: "__gopy_csv_field_size_limit", Helper: helperCSVFieldSizeLimit, RetKind: "int"},
			"Dialect":            {GoFunc: "__gopy_csv_dialect_unused"},
			"Sniffer":            {GoFunc: "__gopy_csv_sniffer_unused"},
			"excel":              {GoFunc: "__gopy_csv_excel_unused"},
			"excel_tab":          {GoFunc: "__gopy_csv_excel_tab_unused"},
			"unix_dialect":       {GoFunc: "__gopy_csv_unix_unused"},
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
			"cast":              {GoFunc: "__gopy_typing_cast_unused"},
			"get_type_hints":    {GoFunc: "__gopy_typing_hints", Helper: helperTypingHints},
			"get_args":          {GoFunc: "__gopy_typing_args", Helper: helperTypingArgs},
			"get_origin":        {GoFunc: "__gopy_typing_origin", Helper: helperTypingOrigin},
			"NewType":           {GoFunc: "__gopy_typing_newtype", Helper: helperTypingNewtype},
			"TYPE_CHECKING":     {GoFunc: "__gopy_typing_typecheck_unused"},
			"runtime_checkable": {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"override":          {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"final":             {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"no_type_check":     {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"assert_type":       {GoFunc: "__gopy_typing_assert_type", Helper: helperTypingAssertType},
			"assert_never":      {GoFunc: "__gopy_typing_assert_never", Helper: helperTypingAssertNever},
			"reveal_type":       {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"TypeGuard":         {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"TypeIs":            {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"ParamSpec":         {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"Concatenate":       {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"NotRequired":       {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"Required":          {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"ReadOnly":          {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"dataclass_transform": {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"deprecated":        {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"TypeAliasType":     {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
		},
	},
	"threading": {
		Funcs: map[string]stdlibFunc{
			"Lock":          {GoFunc: "__gopy_threading_lock", Helper: helperThreadingLock, RetTag: "__Lock", ExtraHelpers: map[string]string{"__Lock": helperLockType}},
			"RLock":         {GoFunc: "__gopy_threading_lock", Helper: helperThreadingLock, RetTag: "__Lock", ExtraHelpers: map[string]string{"__Lock": helperLockType}},
			"Event":         {GoFunc: "__gopy_threading_event_unused"},
			"Condition":     {GoFunc: "__gopy_threading_cond_unused"},
			"Semaphore":     {GoFunc: "__gopy_threading_sem_unused"},
			"BoundedSemaphore": {GoFunc: "__gopy_threading_sem_unused"},
			"Barrier":       {GoFunc: "__gopy_threading_barrier_unused"},
			"Thread":        {GoFunc: "__gopy_threading_thread_unused"},
			"Timer":         {GoFunc: "__gopy_threading_timer_unused"},
			"local":         {GoFunc: "__gopy_threading_local_unused"},
			"current_thread": {GoFunc: "__gopy_threading_current_thread_unused"},
			"main_thread":   {GoFunc: "__gopy_threading_main_thread_unused"},
			"active_count":  {GoFunc: "__gopy_threading_active_count", Helper: helperThreadingActiveCount, RetKind: "int"},
			"enumerate":     {GoFunc: "__gopy_threading_enumerate_unused"},
			"get_ident":     {GoFunc: "__gopy_threading_get_ident", Helper: helperThreadingGetIdent, RetKind: "int"},
			"get_native_id": {GoFunc: "__gopy_threading_get_ident", Helper: helperThreadingGetIdent, RetKind: "int"},
			"settrace":      {GoFunc: "__gopy_threading_settrace_unused"},
			"setprofile":    {GoFunc: "__gopy_threading_setprofile_unused"},
			"stack_size":    {GoFunc: "__gopy_threading_stack_size_unused"},
			"excepthook":    {GoFunc: "__gopy_threading_excepthook_unused"},
		},
	},
	"pathlib": {
		Funcs: map[string]stdlibFunc{
			"Path":            {GoFunc: "__gopy_path_new", GoImport: "os", Helper: helperPathNew, RetTag: "__Path", ExtraHelpers: map[string]string{"__Path": helperPathType}, HelperImports: []string{"os", "path/filepath", "strings", "fmt"}},
			"PurePath":        {GoFunc: "__gopy_path_new", GoImport: "os", Helper: helperPathNew, RetTag: "__Path", ExtraHelpers: map[string]string{"__Path": helperPathType}, HelperImports: []string{"os", "path/filepath", "strings", "fmt"}},
			"PurePosixPath":   {GoFunc: "__gopy_path_new", GoImport: "os", Helper: helperPathNew, RetTag: "__Path", ExtraHelpers: map[string]string{"__Path": helperPathType}, HelperImports: []string{"os", "path/filepath", "strings", "fmt"}},
			"PureWindowsPath": {GoFunc: "__gopy_path_new", GoImport: "os", Helper: helperPathNew, RetTag: "__Path", ExtraHelpers: map[string]string{"__Path": helperPathType}, HelperImports: []string{"os", "path/filepath", "strings", "fmt"}},
			"PosixPath":       {GoFunc: "__gopy_path_new", GoImport: "os", Helper: helperPathNew, RetTag: "__Path", ExtraHelpers: map[string]string{"__Path": helperPathType}, HelperImports: []string{"os", "path/filepath", "strings", "fmt"}},
			"WindowsPath":     {GoFunc: "__gopy_path_new", GoImport: "os", Helper: helperPathNew, RetTag: "__Path", ExtraHelpers: map[string]string{"__Path": helperPathType}, HelperImports: []string{"os", "path/filepath", "strings", "fmt"}},
		},
	},
	"datetime": {
		Attrs: map[string]stdlibAttr{
			"MINYEAR": {GoExpr: "int64(1)"},
			"MAXYEAR": {GoExpr: "int64(9999)"},
		},
		Funcs: map[string]stdlibFunc{
			"timezone":  {GoFunc: "__gopy_datetime_timezone_unused"},
			"timedelta": {GoFunc: "__gopy_timedelta_new", GoImport: "time", Helper: helperTimedeltaNew, RetTag: "__Timedelta", ExtraHelpers: map[string]string{"__Timedelta": helperTimedeltaType}, HelperImports: []string{"fmt"}},
			"date":      {GoFunc: "__gopy_date_new", GoImport: "fmt", Helper: helperDateNew, RetTag: "__Date", ExtraHelpers: map[string]string{"__Date": helperDateType, "__gopy_py_time_format": helperPyTimeFormat, "__gopy_datetime_strftime": helperDatetimeStrftime}, HelperImports: []string{"time", "strings"}},
			// Subs entries below provide date.today / date.fromisoformat
			// as classmethods.
			"time":      {GoFunc: "__gopy_time_new", GoImport: "fmt", Helper: helperTimeNew, RetTag: "__Time", ExtraHelpers: map[string]string{"__Time": helperTimeType}},
		},
		Subs: map[string]stdlibModule{
			"timezone": {
				Attrs: map[string]stdlibAttr{
					"utc": {GoExpr: `"UTC"`},
				},
			},
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
	"shlex": {
		Funcs: map[string]stdlibFunc{
			"split": {GoFunc: "__gopy_shlex_split", Helper: helperShlexSplit, HelperImports: []string{"strings"}},
			"quote": {GoFunc: "__gopy_shlex_quote", Helper: helperShlexQuote, HelperImports: []string{"strings"}, RetKind: "str"},
			"join":  {GoFunc: "__gopy_shlex_join", Helper: helperShlexJoin, ExtraHelpers: map[string]string{"__gopy_shlex_quote": helperShlexQuote}, HelperImports: []string{"strings"}, RetKind: "str"},
		},
	},
	"difflib": {
		Funcs: map[string]stdlibFunc{
			"get_close_matches": {GoFunc: "__gopy_difflib_close", Helper: helperDifflibClose, HelperImports: []string{"sort", "strings"}},
			"unified_diff":      {GoFunc: "__gopy_difflib_unified", Helper: helperDifflibUnified, HelperImports: []string{"fmt"}},
			"ndiff":             {GoFunc: "__gopy_difflib_ndiff", Helper: helperDifflibNdiff},
		},
	},
	"filecmp": {
		Funcs: map[string]stdlibFunc{
			"cmp": {GoFunc: "__gopy_filecmp_cmp", Helper: helperFilecmpCmp, HelperImports: []string{"bytes", "os"}, RetKind: "bool"},
		},
	},
	"codecs": {
		Attrs: map[string]stdlibAttr{
			"BOM":         {GoExpr: `"\xef\xbb\xbf"`},
			"BOM_UTF8":    {GoExpr: `"\xef\xbb\xbf"`},
			"BOM_UTF16":   {GoExpr: `"\xff\xfe"`},
			"BOM_UTF16_LE": {GoExpr: `"\xff\xfe"`},
			"BOM_UTF16_BE": {GoExpr: `"\xfe\xff"`},
			"BOM_UTF32":   {GoExpr: `"\xff\xfe\x00\x00"`},
			"BOM_UTF32_LE": {GoExpr: `"\xff\xfe\x00\x00"`},
			"BOM_UTF32_BE": {GoExpr: `"\x00\x00\xfe\xff"`},
		},
		Funcs: map[string]stdlibFunc{
			"encode":      {GoFunc: "__gopy_codecs_encode", Helper: helperCodecsEncode, HelperImports: []string{"encoding/hex", "encoding/base64"}, RetKind: "str"},
			"decode":      {GoFunc: "__gopy_codecs_decode", Helper: helperCodecsDecode, HelperImports: []string{"encoding/hex", "encoding/base64"}, RetKind: "str"},
			"lookup":      {GoFunc: "__gopy_codecs_lookup", Helper: helperCodecsLookup, RetKind: "str"},
			"getencoder":  {GoFunc: "__gopy_codecs_getencoder_unused"},
			"getdecoder":  {GoFunc: "__gopy_codecs_getdecoder_unused"},
			"register":    {GoFunc: "__gopy_codecs_noop", Helper: helperCodecsNoop},
			"open":        {GoFunc: "__gopy_codecs_open_unused"},
		},
	},
	"ntpath": {
		Funcs: map[string]stdlibFunc{
			"join":     {GoFunc: "__gopy_path_join", GoImport: "path/filepath", Helper: helperPathJoin, RetKind: "str"},
			"basename": {GoFunc: "filepath.Base", GoImport: "path/filepath", RetKind: "str"},
			"dirname":  {GoFunc: "filepath.Dir", GoImport: "path/filepath", RetKind: "str"},
			"split":    {GoFunc: "__gopy_path_split", GoImport: "path/filepath", Helper: helperPathSplit},
			"splitext": {GoFunc: "__gopy_path_splitext", GoImport: "path/filepath", Helper: helperPathSplitext},
		},
	},
	"multiprocessing": {
		Funcs: map[string]stdlibFunc{
			"cpu_count":         {GoFunc: "runtime.NumCPU", GoImport: "runtime", RetKind: "int"},
			"current_process":   {GoFunc: "__gopy_mp_current_process_unused"},
			"active_children":   {GoFunc: "__gopy_mp_active_children_unused"},
			"freeze_support":    {GoFunc: "__gopy_mp_freeze_unused"},
			"set_start_method":  {GoFunc: "__gopy_mp_start_method_unused"},
			"get_start_method":  {GoFunc: "__gopy_mp_start_method_unused"},
			"get_context":       {GoFunc: "__gopy_mp_context_unused"},
			"Process":           {GoFunc: "__gopy_mp_process_unused"},
			"Pool":              {GoFunc: "__gopy_mp_pool_unused"},
			"Lock":              {GoFunc: "__gopy_mp_lock_unused"},
			"RLock":             {GoFunc: "__gopy_mp_lock_unused"},
			"Event":             {GoFunc: "__gopy_mp_event_unused"},
			"Condition":         {GoFunc: "__gopy_mp_cond_unused"},
			"Semaphore":         {GoFunc: "__gopy_mp_sem_unused"},
			"BoundedSemaphore":  {GoFunc: "__gopy_mp_sem_unused"},
			"Barrier":           {GoFunc: "__gopy_mp_barrier_unused"},
			"Queue":             {GoFunc: "__gopy_mp_queue_unused"},
			"JoinableQueue":     {GoFunc: "__gopy_mp_queue_unused"},
			"SimpleQueue":       {GoFunc: "__gopy_mp_queue_unused"},
			"Pipe":              {GoFunc: "__gopy_mp_pipe_unused"},
			"Manager":           {GoFunc: "__gopy_mp_manager_unused"},
			"Value":             {GoFunc: "__gopy_mp_value_unused"},
			"Array":             {GoFunc: "__gopy_mp_array_unused"},
		},
	},
	"token": {
		Attrs: map[string]stdlibAttr{
			"NAME":               {GoExpr: "int64(1)"},
			"NUMBER":             {GoExpr: "int64(2)"},
			"STRING":             {GoExpr: "int64(3)"},
			"NEWLINE":            {GoExpr: "int64(4)"},
			"INDENT":             {GoExpr: "int64(5)"},
			"DEDENT":             {GoExpr: "int64(6)"},
			"OP":                 {GoExpr: "int64(55)"},
			"COMMENT":            {GoExpr: "int64(60)"},
			"NL":                 {GoExpr: "int64(61)"},
			"ENCODING":           {GoExpr: "int64(62)"},
			"ENDMARKER":          {GoExpr: "int64(0)"},
			"FSTRING_START":      {GoExpr: "int64(63)"},
			"FSTRING_MIDDLE":     {GoExpr: "int64(64)"},
			"FSTRING_END":        {GoExpr: "int64(65)"},
			"LPAR":               {GoExpr: "int64(7)"},
			"RPAR":               {GoExpr: "int64(8)"},
			"LSQB":               {GoExpr: "int64(9)"},
			"RSQB":               {GoExpr: "int64(10)"},
			"COLON":              {GoExpr: "int64(11)"},
			"COMMA":              {GoExpr: "int64(12)"},
			"SEMI":               {GoExpr: "int64(13)"},
			"PLUS":               {GoExpr: "int64(14)"},
			"MINUS":              {GoExpr: "int64(15)"},
			"STAR":               {GoExpr: "int64(16)"},
			"SLASH":              {GoExpr: "int64(17)"},
			"exact_token_types":  {GoExpr: `map[string]int64{"(": 7, ")": 8, "[": 9, "]": 10, ":": 11, ",": 12, ";": 13, "+": 14, "-": 15, "*": 16, "/": 17}`},
		},
		Funcs: map[string]stdlibFunc{
			"ISTERMINAL":    {GoFunc: "__gopy_token_isterminal", Helper: helperTokenIsterminal, RetKind: "bool"},
			"ISNONTERMINAL": {GoFunc: "__gopy_token_isnonterminal", Helper: helperTokenIsnonterminal, RetKind: "bool"},
			"ISEOF":         {GoFunc: "__gopy_token_iseof", Helper: helperTokenIseof, RetKind: "bool"},
		},
	},
	"resource": {
		Attrs: map[string]stdlibAttr{
			"RLIMIT_CPU":    {GoExpr: "int64(syscall.RLIMIT_CPU)", GoImport: "syscall"},
			"RLIMIT_FSIZE":  {GoExpr: "int64(syscall.RLIMIT_FSIZE)", GoImport: "syscall"},
			"RLIMIT_DATA":   {GoExpr: "int64(syscall.RLIMIT_DATA)", GoImport: "syscall"},
			"RLIMIT_STACK":  {GoExpr: "int64(syscall.RLIMIT_STACK)", GoImport: "syscall"},
			"RLIMIT_CORE":   {GoExpr: "int64(syscall.RLIMIT_CORE)", GoImport: "syscall"},
			"RLIMIT_NOFILE": {GoExpr: "int64(syscall.RLIMIT_NOFILE)", GoImport: "syscall"},
			"RLIMIT_AS":     {GoExpr: "int64(syscall.RLIMIT_AS)", GoImport: "syscall"},
			"RLIM_INFINITY": {GoExpr: "int64(syscall.RLIM_INFINITY)", GoImport: "syscall"},
		},
		Funcs: map[string]stdlibFunc{
			"getrlimit": {GoFunc: "__gopy_resource_getrlimit", Helper: helperResourceGetrlimit},
			"setrlimit": {GoFunc: "__gopy_resource_setrlimit", Helper: helperResourceSetrlimit},
		},
	},
	"syslog": {
		Attrs: map[string]stdlibAttr{
			"LOG_EMERG":   {GoExpr: "int64(0)"},
			"LOG_ALERT":   {GoExpr: "int64(1)"},
			"LOG_CRIT":    {GoExpr: "int64(2)"},
			"LOG_ERR":     {GoExpr: "int64(3)"},
			"LOG_WARNING": {GoExpr: "int64(4)"},
			"LOG_NOTICE":  {GoExpr: "int64(5)"},
			"LOG_INFO":    {GoExpr: "int64(6)"},
			"LOG_DEBUG":   {GoExpr: "int64(7)"},
		},
		Funcs: map[string]stdlibFunc{
			"openlog":  {GoFunc: "__gopy_syslog_openlog", Helper: helperSyslogOpenlog},
			"syslog":   {GoFunc: "__gopy_syslog_syslog", Helper: helperSyslogSyslog},
			"closelog": {GoFunc: "__gopy_syslog_closelog", Helper: helperSyslogCloselog},
		},
	},
	"quopri": {
		Funcs: map[string]stdlibFunc{
			"encodestring": {GoFunc: "__gopy_quopri_encode", Helper: helperQuopriEncode, HelperImports: []string{"fmt", "strings"}, RetKind: "str"},
			"decodestring": {GoFunc: "__gopy_quopri_decode", Helper: helperQuopriDecode, HelperImports: []string{"strings", "strconv"}, RetKind: "str"},
		},
	},
	"graphlib": {
		Funcs: map[string]stdlibFunc{
			"TopologicalSorter": {GoFunc: "__gopy_graphlib_toposort_unused"},
		},
	},
	"sysconfig": {
		Funcs: map[string]stdlibFunc{
			"get_paths":      {GoFunc: "__gopy_sysconfig_get_paths", Helper: helperSysconfigGetPaths, HelperImports: []string{"os"}},
			"get_platform":   {GoFunc: "__gopy_sysconfig_platform", Helper: helperSysconfigPlatform, HelperImports: []string{"runtime"}, RetKind: "str"},
			"get_python_version": {GoFunc: "__gopy_sysconfig_pyversion", Helper: helperSysconfigPyVersion, RetKind: "str"},
			"get_config_var":     {GoFunc: "__gopy_sysconfig_config_var", Helper: helperSysconfigConfigVar, RetKind: "str"},
		},
	},
	"enum": {
		Funcs: map[string]stdlibFunc{
			"Enum":       {GoFunc: "__gopy_enum_unused"},
			"IntEnum":    {GoFunc: "__gopy_enum_unused"},
			"StrEnum":    {GoFunc: "__gopy_enum_unused"},
			"Flag":       {GoFunc: "__gopy_enum_unused"},
			"IntFlag":    {GoFunc: "__gopy_enum_unused"},
			"ReprEnum":   {GoFunc: "__gopy_enum_unused"},
			"auto":       {GoFunc: "__gopy_enum_auto", Helper: helperEnumAuto, RetKind: "int"},
			"unique":     {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"verify":     {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"member":     {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"nonmember":  {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
		},
	},
	"types": {
		Funcs: map[string]stdlibFunc{
			"SimpleNamespace":  {GoFunc: "__gopy_types_simplens", Helper: helperTypesSimpleNS},
			"MappingProxyType": {GoFunc: "__gopy_types_proxy_unused"},
			"ModuleType":       {GoFunc: "__gopy_types_module_unused"},
			"FunctionType":     {GoFunc: "__gopy_types_func_unused"},
			"MethodType":       {GoFunc: "__gopy_types_method_unused"},
			"GenericAlias":     {GoFunc: "__gopy_types_genalias_unused"},
			"UnionType":        {GoFunc: "__gopy_types_union_unused"},
			"new_class":        {GoFunc: "__gopy_types_newclass_unused"},
			"resolve_bases":    {GoFunc: "__gopy_types_resolvebases_unused"},
		},
	},
	"numbers": {
		Funcs: map[string]stdlibFunc{
			"Number":   {GoFunc: "__gopy_numbers_unused"},
			"Real":     {GoFunc: "__gopy_numbers_unused"},
			"Complex":  {GoFunc: "__gopy_numbers_unused"},
			"Integral": {GoFunc: "__gopy_numbers_unused"},
			"Rational": {GoFunc: "__gopy_numbers_unused"},
		},
	},
	"ipaddress": {
		Funcs: map[string]stdlibFunc{
			"ip_address":   {GoFunc: "__gopy_ipaddress_addr", Helper: helperIpaddressAddr, HelperImports: []string{"net"}, RetKind: "str"},
			"ip_network":   {GoFunc: "__gopy_ipaddress_net", Helper: helperIpaddressNet, HelperImports: []string{"net"}, RetKind: "str"},
			"ip_interface": {GoFunc: "__gopy_ipaddress_addr", Helper: helperIpaddressAddr, HelperImports: []string{"net"}, RetKind: "str"},
			"IPv4Address":  {GoFunc: "__gopy_ipaddress_addr", Helper: helperIpaddressAddr, HelperImports: []string{"net"}, RetKind: "str"},
			"IPv6Address":  {GoFunc: "__gopy_ipaddress_addr", Helper: helperIpaddressAddr, HelperImports: []string{"net"}, RetKind: "str"},
			"IPv4Network":  {GoFunc: "__gopy_ipaddress_net", Helper: helperIpaddressNet, HelperImports: []string{"net"}, RetKind: "str"},
			"IPv6Network":  {GoFunc: "__gopy_ipaddress_net", Helper: helperIpaddressNet, HelperImports: []string{"net"}, RetKind: "str"},
		},
	},
	"netrc": {
		Funcs: map[string]stdlibFunc{
			"netrc":      {GoFunc: "__gopy_netrc_new", Helper: helperNetrcNew, HelperImports: []string{"bufio", "os", "strings"}},
			"NetrcParseError": {GoFunc: "__gopy_netrc_err_unused"},
		},
	},
	"socketserver": {
		Funcs: map[string]stdlibFunc{
			"TCPServer":             {GoFunc: "__gopy_socketserver_unused"},
			"UDPServer":             {GoFunc: "__gopy_socketserver_unused"},
			"ThreadingTCPServer":    {GoFunc: "__gopy_socketserver_unused"},
			"ThreadingUDPServer":    {GoFunc: "__gopy_socketserver_unused"},
			"ForkingTCPServer":      {GoFunc: "__gopy_socketserver_unused"},
			"ForkingUDPServer":      {GoFunc: "__gopy_socketserver_unused"},
			"BaseRequestHandler":    {GoFunc: "__gopy_socketserver_unused"},
			"StreamRequestHandler":  {GoFunc: "__gopy_socketserver_unused"},
			"DatagramRequestHandler": {GoFunc: "__gopy_socketserver_unused"},
			"BaseServer":            {GoFunc: "__gopy_socketserver_unused"},
		},
	},
	"wsgiref": {
		Subs: map[string]stdlibModule{
			"util": {
				Funcs: map[string]stdlibFunc{
					"shift_path_info":         {GoFunc: "__gopy_wsgi_shift_unused"},
					"setup_testing_defaults":  {GoFunc: "__gopy_wsgi_setup_unused"},
					"request_uri":             {GoFunc: "__gopy_wsgi_uri_unused"},
					"application_uri":         {GoFunc: "__gopy_wsgi_uri_unused"},
					"guess_scheme":            {GoFunc: "__gopy_wsgi_scheme_unused"},
					"FileWrapper":             {GoFunc: "__gopy_wsgi_wrap_unused"},
					"is_hop_by_hop":           {GoFunc: "__gopy_wsgi_hop_unused"},
				},
			},
			"headers": {
				Funcs: map[string]stdlibFunc{
					"Headers": {GoFunc: "__gopy_wsgi_headers_unused"},
				},
			},
			"simple_server": {
				Funcs: map[string]stdlibFunc{
					"WSGIServer":          {GoFunc: "__gopy_wsgi_server_unused"},
					"WSGIRequestHandler":  {GoFunc: "__gopy_wsgi_handler_unused"},
					"make_server":         {GoFunc: "__gopy_wsgi_makeserver_unused"},
					"demo_app":            {GoFunc: "__gopy_wsgi_demo_unused"},
				},
			},
			"handlers": {
				Funcs: map[string]stdlibFunc{
					"BaseHandler":    {GoFunc: "__gopy_wsgi_basehandler_unused"},
					"SimpleHandler":  {GoFunc: "__gopy_wsgi_simplehandler_unused"},
					"CGIHandler":     {GoFunc: "__gopy_wsgi_cgi_unused"},
					"IISCGIHandler":  {GoFunc: "__gopy_wsgi_iiscgi_unused"},
				},
			},
		},
	},
	"concurrent": {
		Subs: map[string]stdlibModule{
			"futures": {
				Attrs: map[string]stdlibAttr{
					"FIRST_COMPLETED":   {GoExpr: `"FIRST_COMPLETED"`},
					"FIRST_EXCEPTION":   {GoExpr: `"FIRST_EXCEPTION"`},
					"ALL_COMPLETED":     {GoExpr: `"ALL_COMPLETED"`},
				},
				Funcs: map[string]stdlibFunc{
					"ThreadPoolExecutor":  {GoFunc: "__gopy_cf_tpool_unused"},
					"ProcessPoolExecutor": {GoFunc: "__gopy_cf_ppool_unused"},
					"Executor":            {GoFunc: "__gopy_cf_exec_unused"},
					"Future":              {GoFunc: "__gopy_cf_future_unused"},
					"wait":                {GoFunc: "__gopy_cf_wait_unused"},
					"as_completed":        {GoFunc: "__gopy_cf_as_completed_unused"},
					"CancelledError":      {GoFunc: "__gopy_cf_cancel_err_unused"},
					"TimeoutError":        {GoFunc: "__gopy_cf_timeout_err_unused"},
					"BrokenExecutor":      {GoFunc: "__gopy_cf_broken_err_unused"},
					"InvalidStateError":   {GoFunc: "__gopy_cf_state_err_unused"},
				},
			},
		},
	},
	"abc": {
		Funcs: map[string]stdlibFunc{
			"ABC":                  {GoFunc: "__gopy_abc_unused"},
			"ABCMeta":              {GoFunc: "__gopy_abc_unused"},
			"abstractmethod":       {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"abstractproperty":     {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"abstractclassmethod":  {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"abstractstaticmethod": {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
			"get_cache_token":      {GoFunc: "__gopy_abc_cache_token_unused"},
			"update_abstractmethods": {GoFunc: "__gopy_typing_passthrough", Helper: helperTypingPassthrough},
		},
	},
	"contextvars": {
		Funcs: map[string]stdlibFunc{
			"ContextVar":    {GoFunc: "__gopy_cv_unused"},
			"Context":       {GoFunc: "__gopy_cv_unused"},
			"copy_context":  {GoFunc: "__gopy_cv_copy_unused"},
			"Token":         {GoFunc: "__gopy_cv_token_unused"},
		},
	},
	"plistlib": {
		Attrs: map[string]stdlibAttr{
			"FMT_XML":    {GoExpr: "int64(1)"},
			"FMT_BINARY": {GoExpr: "int64(2)"},
		},
		Funcs: map[string]stdlibFunc{
			"load":          {GoFunc: "__gopy_plist_load_unused"},
			"loads":         {GoFunc: "__gopy_plist_loads_unused"},
			"dump":          {GoFunc: "__gopy_plist_dump_unused"},
			"dumps":         {GoFunc: "__gopy_plist_dumps_unused"},
			"InvalidFileException": {GoFunc: "__gopy_plist_err_unused"},
			"UID":           {GoFunc: "__gopy_plist_uid_unused"},
		},
	},
	"smtplib": {
		Attrs: map[string]stdlibAttr{
			"SMTP_PORT":     {GoExpr: "int64(25)"},
			"SMTP_SSL_PORT": {GoExpr: "int64(465)"},
			"LMTP_PORT":     {GoExpr: "int64(2003)"},
		},
		Funcs: map[string]stdlibFunc{
			"SMTP":               {GoFunc: "__gopy_smtp_unused"},
			"SMTP_SSL":           {GoFunc: "__gopy_smtp_unused"},
			"LMTP":               {GoFunc: "__gopy_smtp_unused"},
			"SMTPException":      {GoFunc: "__gopy_smtp_err_unused"},
			"SMTPServerDisconnected": {GoFunc: "__gopy_smtp_err_unused"},
			"SMTPResponseException": {GoFunc: "__gopy_smtp_err_unused"},
			"SMTPSenderRefused":  {GoFunc: "__gopy_smtp_err_unused"},
			"SMTPRecipientsRefused": {GoFunc: "__gopy_smtp_err_unused"},
			"SMTPDataError":      {GoFunc: "__gopy_smtp_err_unused"},
			"SMTPConnectError":   {GoFunc: "__gopy_smtp_err_unused"},
			"SMTPHeloError":      {GoFunc: "__gopy_smtp_err_unused"},
			"SMTPNotSupportedError": {GoFunc: "__gopy_smtp_err_unused"},
			"SMTPAuthenticationError": {GoFunc: "__gopy_smtp_err_unused"},
		},
	},
	"imaplib": {
		Funcs: map[string]stdlibFunc{
			"IMAP4":      {GoFunc: "__gopy_imap_unused"},
			"IMAP4_SSL":  {GoFunc: "__gopy_imap_unused"},
			"IMAP4_stream": {GoFunc: "__gopy_imap_unused"},
		},
	},
	"poplib": {
		Funcs: map[string]stdlibFunc{
			"POP3":     {GoFunc: "__gopy_pop_unused"},
			"POP3_SSL": {GoFunc: "__gopy_pop_unused"},
			"error_proto": {GoFunc: "__gopy_pop_err_unused"},
		},
	},
	"ftplib": {
		Attrs: map[string]stdlibAttr{
			"FTP_PORT":      {GoExpr: "int64(21)"},
			"MAXLINE":       {GoExpr: "int64(8192)"},
		},
		Funcs: map[string]stdlibFunc{
			"FTP":     {GoFunc: "__gopy_ftp_unused"},
			"FTP_TLS": {GoFunc: "__gopy_ftp_unused"},
			"all_errors": {GoFunc: "__gopy_ftp_err_unused"},
			"error_reply": {GoFunc: "__gopy_ftp_err_unused"},
			"error_temp":  {GoFunc: "__gopy_ftp_err_unused"},
			"error_perm":  {GoFunc: "__gopy_ftp_err_unused"},
			"error_proto": {GoFunc: "__gopy_ftp_err_unused"},
		},
	},
	"telnetlib": {
		Funcs: map[string]stdlibFunc{
			"Telnet": {GoFunc: "__gopy_telnet_unused"},
		},
	},
	"nntplib": {
		Funcs: map[string]stdlibFunc{
			"NNTP":     {GoFunc: "__gopy_nntp_unused"},
			"NNTP_SSL": {GoFunc: "__gopy_nntp_unused"},
			"NNTPError": {GoFunc: "__gopy_nntp_err_unused"},
		},
	},
	"xmlrpc": {
		Subs: map[string]stdlibModule{
			"client": {
				Funcs: map[string]stdlibFunc{
					"ServerProxy":      {GoFunc: "__gopy_xmlrpc_server_proxy_unused"},
					"MultiCall":        {GoFunc: "__gopy_xmlrpc_multicall_unused"},
					"Binary":           {GoFunc: "__gopy_xmlrpc_binary_unused"},
					"DateTime":         {GoFunc: "__gopy_xmlrpc_datetime_unused"},
					"Fault":            {GoFunc: "__gopy_xmlrpc_fault_unused"},
					"ProtocolError":    {GoFunc: "__gopy_xmlrpc_err_unused"},
					"ResponseError":    {GoFunc: "__gopy_xmlrpc_err_unused"},
					"Error":            {GoFunc: "__gopy_xmlrpc_err_unused"},
					"Transport":        {GoFunc: "__gopy_xmlrpc_transport_unused"},
					"SafeTransport":    {GoFunc: "__gopy_xmlrpc_transport_unused"},
					"loads":            {GoFunc: "__gopy_xmlrpc_loads_unused"},
					"dumps":            {GoFunc: "__gopy_xmlrpc_dumps_unused"},
				},
			},
			"server": {
				Funcs: map[string]stdlibFunc{
					"SimpleXMLRPCServer":  {GoFunc: "__gopy_xmlrpc_server_unused"},
					"DocXMLRPCServer":     {GoFunc: "__gopy_xmlrpc_server_unused"},
					"CGIXMLRPCRequestHandler": {GoFunc: "__gopy_xmlrpc_handler_unused"},
					"SimpleXMLRPCRequestHandler": {GoFunc: "__gopy_xmlrpc_handler_unused"},
				},
			},
		},
	},
	"ssl": {
		Attrs: map[string]stdlibAttr{
			"PROTOCOL_TLS":            {GoExpr: "int64(2)"},
			"PROTOCOL_TLS_CLIENT":     {GoExpr: "int64(16)"},
			"PROTOCOL_TLS_SERVER":     {GoExpr: "int64(17)"},
			"PROTOCOL_TLSv1":          {GoExpr: "int64(3)"},
			"PROTOCOL_TLSv1_1":        {GoExpr: "int64(4)"},
			"PROTOCOL_TLSv1_2":        {GoExpr: "int64(5)"},
			"CERT_NONE":               {GoExpr: "int64(0)"},
			"CERT_OPTIONAL":           {GoExpr: "int64(1)"},
			"CERT_REQUIRED":           {GoExpr: "int64(2)"},
			"VERIFY_DEFAULT":          {GoExpr: "int64(0)"},
			"VERIFY_CRL_CHECK_LEAF":   {GoExpr: "int64(4)"},
			"VERIFY_CRL_CHECK_CHAIN":  {GoExpr: "int64(12)"},
			"VERIFY_X509_STRICT":      {GoExpr: "int64(32)"},
			"VERIFY_X509_TRUSTED_FIRST": {GoExpr: "int64(32768)"},
			"OP_NO_SSLv2":             {GoExpr: "int64(16777216)"},
			"OP_NO_SSLv3":             {GoExpr: "int64(33554432)"},
			"OP_NO_TLSv1":             {GoExpr: "int64(67108864)"},
			"OP_NO_TLSv1_1":           {GoExpr: "int64(268435456)"},
			"HAS_SNI":                 {GoExpr: "true"},
			"HAS_ECDH":                {GoExpr: "true"},
			"HAS_ALPN":                {GoExpr: "true"},
			"HAS_NPN":                 {GoExpr: "true"},
		},
		Funcs: map[string]stdlibFunc{
			"create_default_context":  {GoFunc: "__gopy_ssl_ctx_unused"},
			"SSLContext":              {GoFunc: "__gopy_ssl_ctx_unused"},
			"wrap_socket":             {GoFunc: "__gopy_ssl_wrap_unused"},
			"get_default_verify_paths": {GoFunc: "__gopy_ssl_paths_unused"},
			"get_server_certificate":  {GoFunc: "__gopy_ssl_get_cert_unused"},
			"SSLError":                {GoFunc: "__gopy_ssl_err_unused"},
			"SSLZeroReturnError":      {GoFunc: "__gopy_ssl_err_unused"},
			"SSLWantReadError":        {GoFunc: "__gopy_ssl_err_unused"},
			"SSLWantWriteError":       {GoFunc: "__gopy_ssl_err_unused"},
			"SSLSyscallError":         {GoFunc: "__gopy_ssl_err_unused"},
			"SSLEOFError":             {GoFunc: "__gopy_ssl_err_unused"},
			"CertificateError":        {GoFunc: "__gopy_ssl_err_unused"},
			"SSLObject":               {GoFunc: "__gopy_ssl_obj_unused"},
			"SSLSocket":               {GoFunc: "__gopy_ssl_socket_unused"},
			"SSLSession":              {GoFunc: "__gopy_ssl_session_unused"},
			"Purpose":                 {GoFunc: "__gopy_ssl_purpose_unused"},
		},
	},
	"mmap": {
		Attrs: map[string]stdlibAttr{
			"ACCESS_READ":    {GoExpr: "int64(1)"},
			"ACCESS_WRITE":   {GoExpr: "int64(2)"},
			"ACCESS_COPY":    {GoExpr: "int64(3)"},
			"ACCESS_DEFAULT": {GoExpr: "int64(0)"},
			"MAP_SHARED":     {GoExpr: "int64(1)"},
			"MAP_PRIVATE":    {GoExpr: "int64(2)"},
			"MAP_ANONYMOUS":  {GoExpr: "int64(32)"},
			"MAP_ANON":       {GoExpr: "int64(32)"},
			"PROT_READ":      {GoExpr: "int64(1)"},
			"PROT_WRITE":     {GoExpr: "int64(2)"},
			"PROT_EXEC":      {GoExpr: "int64(4)"},
			"PAGESIZE":       {GoExpr: "int64(4096)"},
			"ALLOCATIONGRANULARITY": {GoExpr: "int64(4096)"},
		},
		Funcs: map[string]stdlibFunc{
			"mmap": {GoFunc: "__gopy_mmap_unused"},
		},
	},
	"ast": {
		Attrs: map[string]stdlibAttr{
			"PyCF_ONLY_AST":           {GoExpr: "int64(1024)"},
			"PyCF_ALLOW_TOP_LEVEL_AWAIT": {GoExpr: "int64(8192)"},
			"PyCF_TYPE_COMMENTS":      {GoExpr: "int64(4096)"},
		},
		Funcs: map[string]stdlibFunc{
			"parse":          {GoFunc: "__gopy_ast_parse_unused"},
			"dump":           {GoFunc: "__gopy_ast_dump_unused"},
			"walk":           {GoFunc: "__gopy_ast_walk_unused"},
			"unparse":        {GoFunc: "__gopy_ast_unparse_unused"},
			"literal_eval":   {GoFunc: "__gopy_ast_literal_eval_unused"},
			"iter_fields":    {GoFunc: "__gopy_ast_iter_fields_unused"},
			"iter_child_nodes": {GoFunc: "__gopy_ast_iter_child_unused"},
			"fix_missing_locations": {GoFunc: "__gopy_ast_fix_unused"},
			"increment_lineno": {GoFunc: "__gopy_ast_inc_lineno_unused"},
			"copy_location":  {GoFunc: "__gopy_ast_copy_loc_unused"},
			"get_docstring":  {GoFunc: "__gopy_ast_docstring_unused"},
			"NodeVisitor":    {GoFunc: "__gopy_ast_node_unused"},
			"NodeTransformer": {GoFunc: "__gopy_ast_node_unused"},
			"AST":            {GoFunc: "__gopy_ast_node_unused"},
			"Module":         {GoFunc: "__gopy_ast_node_unused"},
			"Expression":     {GoFunc: "__gopy_ast_node_unused"},
			"FunctionDef":    {GoFunc: "__gopy_ast_node_unused"},
			"AsyncFunctionDef": {GoFunc: "__gopy_ast_node_unused"},
			"ClassDef":       {GoFunc: "__gopy_ast_node_unused"},
			"Return":         {GoFunc: "__gopy_ast_node_unused"},
			"Assign":         {GoFunc: "__gopy_ast_node_unused"},
			"AugAssign":      {GoFunc: "__gopy_ast_node_unused"},
			"AnnAssign":      {GoFunc: "__gopy_ast_node_unused"},
			"If":             {GoFunc: "__gopy_ast_node_unused"},
			"For":            {GoFunc: "__gopy_ast_node_unused"},
			"While":          {GoFunc: "__gopy_ast_node_unused"},
			"With":           {GoFunc: "__gopy_ast_node_unused"},
			"Try":            {GoFunc: "__gopy_ast_node_unused"},
			"Match":          {GoFunc: "__gopy_ast_node_unused"},
			"Import":         {GoFunc: "__gopy_ast_node_unused"},
			"ImportFrom":     {GoFunc: "__gopy_ast_node_unused"},
			"Constant":       {GoFunc: "__gopy_ast_node_unused"},
			"Name":           {GoFunc: "__gopy_ast_node_unused"},
			"Attribute":      {GoFunc: "__gopy_ast_node_unused"},
			"Call":           {GoFunc: "__gopy_ast_node_unused"},
			"BinOp":          {GoFunc: "__gopy_ast_node_unused"},
			"UnaryOp":        {GoFunc: "__gopy_ast_node_unused"},
			"Compare":        {GoFunc: "__gopy_ast_node_unused"},
			"Add":            {GoFunc: "__gopy_ast_node_unused"},
			"Sub":            {GoFunc: "__gopy_ast_node_unused"},
			"Mult":           {GoFunc: "__gopy_ast_node_unused"},
			"Div":            {GoFunc: "__gopy_ast_node_unused"},
			"Mod":            {GoFunc: "__gopy_ast_node_unused"},
			"Lambda":         {GoFunc: "__gopy_ast_node_unused"},
			"List":           {GoFunc: "__gopy_ast_node_unused"},
			"Tuple":          {GoFunc: "__gopy_ast_node_unused"},
			"Dict":           {GoFunc: "__gopy_ast_node_unused"},
			"Set":            {GoFunc: "__gopy_ast_node_unused"},
			"Subscript":      {GoFunc: "__gopy_ast_node_unused"},
			"Slice":          {GoFunc: "__gopy_ast_node_unused"},
			"Starred":        {GoFunc: "__gopy_ast_node_unused"},
			"Load":           {GoFunc: "__gopy_ast_node_unused"},
			"Store":          {GoFunc: "__gopy_ast_node_unused"},
			"Del":            {GoFunc: "__gopy_ast_node_unused"},
		},
	},
	"bdb": {
		Funcs: map[string]stdlibFunc{
			"Bdb":          {GoFunc: "__gopy_bdb_unused"},
			"Breakpoint":   {GoFunc: "__gopy_bdb_unused"},
			"BdbQuit":      {GoFunc: "__gopy_bdb_unused"},
		},
	},
	"bz2": {
		Funcs: map[string]stdlibFunc{
			"compress":   {GoFunc: "__gopy_bz2_compress_unused"},
			"decompress": {GoFunc: "__gopy_bz2_decompress", Helper: helperBz2Decompress, HelperImports: []string{"compress/bzip2", "bytes", "io"}, RetKind: "str"},
			"open":       {GoFunc: "__gopy_bz2_open_unused"},
			"BZ2File":    {GoFunc: "__gopy_bz2_file_unused"},
			"BZ2Compressor": {GoFunc: "__gopy_bz2_comp_unused"},
			"BZ2Decompressor": {GoFunc: "__gopy_bz2_decomp_unused"},
		},
	},
	"cmd": {
		Funcs: map[string]stdlibFunc{
			"Cmd": {GoFunc: "__gopy_cmd_unused"},
		},
	},
	"code": {
		Funcs: map[string]stdlibFunc{
			"InteractiveInterpreter": {GoFunc: "__gopy_code_unused"},
			"InteractiveConsole":     {GoFunc: "__gopy_code_unused"},
			"interact":               {GoFunc: "__gopy_code_unused"},
			"compile_command":        {GoFunc: "__gopy_code_unused"},
		},
	},
	"codeop": {
		Funcs: map[string]stdlibFunc{
			"compile_command":   {GoFunc: "__gopy_codeop_unused"},
			"Compile":           {GoFunc: "__gopy_codeop_unused"},
			"CommandCompiler":   {GoFunc: "__gopy_codeop_unused"},
		},
	},
	"compileall": {
		Funcs: map[string]stdlibFunc{
			"compile_dir":  {GoFunc: "__gopy_compileall_unused"},
			"compile_file": {GoFunc: "__gopy_compileall_unused"},
			"compile_path": {GoFunc: "__gopy_compileall_unused"},
		},
	},
	"py_compile": {
		Funcs: map[string]stdlibFunc{
			"compile":      {GoFunc: "__gopy_pycompile_unused"},
			"PyCompileError": {GoFunc: "__gopy_pycompile_unused"},
		},
	},
	"copyreg": {
		Funcs: map[string]stdlibFunc{
			"pickle":            {GoFunc: "__gopy_copyreg_unused"},
			"constructor":       {GoFunc: "__gopy_copyreg_unused"},
			"add_extension":     {GoFunc: "__gopy_copyreg_unused"},
			"remove_extension":  {GoFunc: "__gopy_copyreg_unused"},
			"clear_extension_cache": {GoFunc: "__gopy_copyreg_unused"},
			"__newobj__":        {GoFunc: "__gopy_copyreg_unused"},
			"__newobj_ex__":     {GoFunc: "__gopy_copyreg_unused"},
		},
	},
	"pickletools": {
		Funcs: map[string]stdlibFunc{
			"dis":      {GoFunc: "__gopy_pickletools_unused"},
			"genops":   {GoFunc: "__gopy_pickletools_unused"},
			"optimize": {GoFunc: "__gopy_pickletools_unused"},
		},
	},
	"marshal": {
		Attrs: map[string]stdlibAttr{
			"version": {GoExpr: "int64(5)"},
		},
		Funcs: map[string]stdlibFunc{
			"dump":  {GoFunc: "__gopy_marshal_unused"},
			"dumps": {GoFunc: "__gopy_marshal_unused"},
			"load":  {GoFunc: "__gopy_marshal_unused"},
			"loads": {GoFunc: "__gopy_marshal_unused"},
		},
	},
	"dbm": {
		Funcs: map[string]stdlibFunc{
			"open":  {GoFunc: "__gopy_dbm_open_unused"},
			"whichdb": {GoFunc: "__gopy_dbm_whichdb_unused"},
			"error": {GoFunc: "__gopy_dbm_err_unused"},
		},
		Subs: map[string]stdlibModule{
			"gnu":  {Funcs: map[string]stdlibFunc{"open": {GoFunc: "__gopy_dbm_unused"}, "error": {GoFunc: "__gopy_dbm_unused"}}},
			"ndbm": {Funcs: map[string]stdlibFunc{"open": {GoFunc: "__gopy_dbm_unused"}, "error": {GoFunc: "__gopy_dbm_unused"}}},
			"dumb": {Funcs: map[string]stdlibFunc{"open": {GoFunc: "__gopy_dbm_unused"}, "error": {GoFunc: "__gopy_dbm_unused"}}},
			"sqlite3": {Funcs: map[string]stdlibFunc{"open": {GoFunc: "__gopy_dbm_unused"}}},
		},
	},
	"doctest": {
		Attrs: map[string]stdlibAttr{
			"DONT_ACCEPT_TRUE_FOR_1":      {GoExpr: "int64(1)"},
			"DONT_ACCEPT_BLANKLINE":       {GoExpr: "int64(2)"},
			"NORMALIZE_WHITESPACE":        {GoExpr: "int64(4)"},
			"ELLIPSIS":                    {GoExpr: "int64(8)"},
			"SKIP":                        {GoExpr: "int64(16)"},
			"IGNORE_EXCEPTION_DETAIL":     {GoExpr: "int64(32)"},
			"COMPARISON_FLAGS":            {GoExpr: "int64(63)"},
			"REPORT_UDIFF":                {GoExpr: "int64(64)"},
			"REPORT_CDIFF":                {GoExpr: "int64(128)"},
			"REPORT_NDIFF":                {GoExpr: "int64(256)"},
			"REPORT_ONLY_FIRST_FAILURE":   {GoExpr: "int64(512)"},
			"REPORTING_FLAGS":             {GoExpr: "int64(960)"},
		},
		Funcs: map[string]stdlibFunc{
			"testmod":             {GoFunc: "__gopy_doctest_unused"},
			"testfile":            {GoFunc: "__gopy_doctest_unused"},
			"run_docstring_examples": {GoFunc: "__gopy_doctest_unused"},
			"DocTestSuite":        {GoFunc: "__gopy_doctest_unused"},
			"DocFileSuite":        {GoFunc: "__gopy_doctest_unused"},
			"DocTestParser":       {GoFunc: "__gopy_doctest_unused"},
			"DocTestRunner":       {GoFunc: "__gopy_doctest_unused"},
			"DocTestFinder":       {GoFunc: "__gopy_doctest_unused"},
			"DocTest":             {GoFunc: "__gopy_doctest_unused"},
			"Example":             {GoFunc: "__gopy_doctest_unused"},
			"OutputChecker":       {GoFunc: "__gopy_doctest_unused"},
			"DebugRunner":         {GoFunc: "__gopy_doctest_unused"},
			"set_unittest_reportflags": {GoFunc: "__gopy_doctest_unused"},
		},
	},
	"faulthandler": {
		Funcs: map[string]stdlibFunc{
			"enable":         {GoFunc: "__gopy_faulthandler_unused"},
			"disable":        {GoFunc: "__gopy_faulthandler_unused"},
			"is_enabled":     {GoFunc: "__gopy_faulthandler_isenabled", Helper: helperFaultHandlerIsEnabled, RetKind: "bool"},
			"register":       {GoFunc: "__gopy_faulthandler_unused"},
			"unregister":     {GoFunc: "__gopy_faulthandler_unused"},
			"dump_traceback": {GoFunc: "__gopy_faulthandler_unused"},
			"dump_traceback_later": {GoFunc: "__gopy_faulthandler_unused"},
			"cancel_dump_traceback_later": {GoFunc: "__gopy_faulthandler_unused"},
		},
	},
	"fcntl": {
		Attrs: map[string]stdlibAttr{
			"F_DUPFD":   {GoExpr: "int64(0)"},
			"F_GETFD":   {GoExpr: "int64(1)"},
			"F_SETFD":   {GoExpr: "int64(2)"},
			"F_GETFL":   {GoExpr: "int64(3)"},
			"F_SETFL":   {GoExpr: "int64(4)"},
			"F_GETLK":   {GoExpr: "int64(5)"},
			"F_SETLK":   {GoExpr: "int64(6)"},
			"F_SETLKW":  {GoExpr: "int64(7)"},
			"FD_CLOEXEC": {GoExpr: "int64(1)"},
			"LOCK_SH":   {GoExpr: "int64(1)"},
			"LOCK_EX":   {GoExpr: "int64(2)"},
			"LOCK_NB":   {GoExpr: "int64(4)"},
			"LOCK_UN":   {GoExpr: "int64(8)"},
		},
		Funcs: map[string]stdlibFunc{
			"fcntl":  {GoFunc: "__gopy_fcntl_unused"},
			"ioctl":  {GoFunc: "__gopy_fcntl_unused"},
			"flock":  {GoFunc: "__gopy_fcntl_unused"},
			"lockf":  {GoFunc: "__gopy_fcntl_unused"},
		},
	},
	"fileinput": {
		Funcs: map[string]stdlibFunc{
			"input":      {GoFunc: "__gopy_fileinput_unused"},
			"FileInput":  {GoFunc: "__gopy_fileinput_unused"},
			"filename":   {GoFunc: "__gopy_fileinput_unused"},
			"lineno":     {GoFunc: "__gopy_fileinput_unused"},
			"isfirstline": {GoFunc: "__gopy_fileinput_unused"},
			"isstdin":    {GoFunc: "__gopy_fileinput_unused"},
			"close":      {GoFunc: "__gopy_fileinput_unused"},
			"nextfile":   {GoFunc: "__gopy_fileinput_unused"},
			"hook_compressed": {GoFunc: "__gopy_fileinput_unused"},
			"hook_encoded":    {GoFunc: "__gopy_fileinput_unused"},
		},
	},
	"importlib": {
		Funcs: map[string]stdlibFunc{
			"import_module":   {GoFunc: "__gopy_importlib_unused"},
			"reload":          {GoFunc: "__gopy_importlib_unused"},
			"invalidate_caches": {GoFunc: "__gopy_importlib_unused"},
			"find_loader":     {GoFunc: "__gopy_importlib_unused"},
		},
		Subs: map[string]stdlibModule{
			"util": {
				Funcs: map[string]stdlibFunc{
					"find_spec":       {GoFunc: "__gopy_importlib_unused"},
					"spec_from_file_location": {GoFunc: "__gopy_importlib_unused"},
					"module_from_spec": {GoFunc: "__gopy_importlib_unused"},
					"resolve_name":    {GoFunc: "__gopy_importlib_unused"},
					"LazyLoader":      {GoFunc: "__gopy_importlib_unused"},
				},
			},
			"abc": {
				Funcs: map[string]stdlibFunc{
					"Loader":       {GoFunc: "__gopy_importlib_unused"},
					"MetaPathFinder": {GoFunc: "__gopy_importlib_unused"},
					"PathEntryFinder": {GoFunc: "__gopy_importlib_unused"},
					"FileLoader":   {GoFunc: "__gopy_importlib_unused"},
					"SourceLoader": {GoFunc: "__gopy_importlib_unused"},
				},
			},
			"machinery": {
				Funcs: map[string]stdlibFunc{
					"ModuleSpec":      {GoFunc: "__gopy_importlib_unused"},
					"SourceFileLoader": {GoFunc: "__gopy_importlib_unused"},
					"SourcelessFileLoader": {GoFunc: "__gopy_importlib_unused"},
					"ExtensionFileLoader": {GoFunc: "__gopy_importlib_unused"},
					"FileFinder":      {GoFunc: "__gopy_importlib_unused"},
					"PathFinder":      {GoFunc: "__gopy_importlib_unused"},
				},
			},
			"resources": {
				Funcs: map[string]stdlibFunc{
					"files":         {GoFunc: "__gopy_importlib_unused"},
					"as_file":       {GoFunc: "__gopy_importlib_unused"},
					"open_binary":   {GoFunc: "__gopy_importlib_unused"},
					"open_text":     {GoFunc: "__gopy_importlib_unused"},
					"read_binary":   {GoFunc: "__gopy_importlib_unused"},
					"read_text":     {GoFunc: "__gopy_importlib_unused"},
					"path":          {GoFunc: "__gopy_importlib_unused"},
					"contents":      {GoFunc: "__gopy_importlib_unused"},
					"is_resource":   {GoFunc: "__gopy_importlib_unused"},
				},
			},
			"metadata": {
				Funcs: map[string]stdlibFunc{
					"version":      {GoFunc: "__gopy_importlib_unused"},
					"distribution": {GoFunc: "__gopy_importlib_unused"},
					"distributions": {GoFunc: "__gopy_importlib_unused"},
					"metadata":     {GoFunc: "__gopy_importlib_unused"},
					"entry_points": {GoFunc: "__gopy_importlib_unused"},
					"files":        {GoFunc: "__gopy_importlib_unused"},
					"requires":     {GoFunc: "__gopy_importlib_unused"},
					"packages_distributions": {GoFunc: "__gopy_importlib_unused"},
					"PackageNotFoundError": {GoFunc: "__gopy_importlib_unused"},
				},
			},
		},
	},
	"pkgutil": {
		Funcs: map[string]stdlibFunc{
			"iter_modules":     {GoFunc: "__gopy_pkgutil_unused"},
			"walk_packages":    {GoFunc: "__gopy_pkgutil_unused"},
			"get_data":         {GoFunc: "__gopy_pkgutil_unused"},
			"get_loader":       {GoFunc: "__gopy_pkgutil_unused"},
			"resolve_name":     {GoFunc: "__gopy_pkgutil_unused"},
			"extend_path":      {GoFunc: "__gopy_pkgutil_unused"},
			"ImpImporter":      {GoFunc: "__gopy_pkgutil_unused"},
			"ImpLoader":        {GoFunc: "__gopy_pkgutil_unused"},
			"ModuleInfo":       {GoFunc: "__gopy_pkgutil_unused"},
		},
	},
	"zipimport": {
		Funcs: map[string]stdlibFunc{
			"zipimporter":     {GoFunc: "__gopy_zipimport_unused"},
			"ZipImportError":  {GoFunc: "__gopy_zipimport_unused"},
		},
	},
	"lzma": {
		Attrs: map[string]stdlibAttr{
			"FORMAT_XZ":      {GoExpr: "int64(1)"},
			"FORMAT_ALONE":   {GoExpr: "int64(2)"},
			"FORMAT_RAW":     {GoExpr: "int64(3)"},
			"FORMAT_AUTO":    {GoExpr: "int64(0)"},
			"CHECK_NONE":     {GoExpr: "int64(0)"},
			"CHECK_CRC32":    {GoExpr: "int64(1)"},
			"CHECK_CRC64":    {GoExpr: "int64(4)"},
			"CHECK_SHA256":   {GoExpr: "int64(10)"},
			"PRESET_DEFAULT": {GoExpr: "int64(6)"},
			"PRESET_EXTREME": {GoExpr: "int64(2147483648)"},
		},
		Funcs: map[string]stdlibFunc{
			"compress":   {GoFunc: "__gopy_lzma_unused"},
			"decompress": {GoFunc: "__gopy_lzma_unused"},
			"open":       {GoFunc: "__gopy_lzma_unused"},
			"LZMAFile":   {GoFunc: "__gopy_lzma_unused"},
			"LZMACompressor": {GoFunc: "__gopy_lzma_unused"},
			"LZMADecompressor": {GoFunc: "__gopy_lzma_unused"},
			"LZMAError":  {GoFunc: "__gopy_lzma_unused"},
		},
	},
	"tarfile": {
		Attrs: map[string]stdlibAttr{
			"REGTYPE":  {GoExpr: `"0"`},
			"AREGTYPE": {GoExpr: `"\x00"`},
			"LNKTYPE":  {GoExpr: `"1"`},
			"SYMTYPE":  {GoExpr: `"2"`},
			"DIRTYPE":  {GoExpr: `"5"`},
			"FIFOTYPE": {GoExpr: `"6"`},
			"CHRTYPE":  {GoExpr: `"3"`},
			"BLKTYPE":  {GoExpr: `"4"`},
			"CONTTYPE": {GoExpr: `"7"`},
			"DEFAULT_FORMAT": {GoExpr: "int64(2)"},
			"USTAR_FORMAT":   {GoExpr: "int64(0)"},
			"GNU_FORMAT":     {GoExpr: "int64(1)"},
			"PAX_FORMAT":     {GoExpr: "int64(2)"},
		},
		Funcs: map[string]stdlibFunc{
			"open":      {GoFunc: "__gopy_tarfile_unused"},
			"TarFile":   {GoFunc: "__gopy_tarfile_unused"},
			"TarInfo":   {GoFunc: "__gopy_tarfile_unused"},
			"is_tarfile": {GoFunc: "__gopy_tarfile_unused"},
			"TarError":  {GoFunc: "__gopy_tarfile_unused"},
			"ReadError": {GoFunc: "__gopy_tarfile_unused"},
			"CompressionError": {GoFunc: "__gopy_tarfile_unused"},
			"StreamError": {GoFunc: "__gopy_tarfile_unused"},
			"ExtractError": {GoFunc: "__gopy_tarfile_unused"},
			"HeaderError": {GoFunc: "__gopy_tarfile_unused"},
		},
	},
	"zipfile": {
		Attrs: map[string]stdlibAttr{
			"ZIP_STORED":   {GoExpr: "int64(0)"},
			"ZIP_DEFLATED": {GoExpr: "int64(8)"},
			"ZIP_BZIP2":    {GoExpr: "int64(12)"},
			"ZIP_LZMA":     {GoExpr: "int64(14)"},
		},
		Funcs: map[string]stdlibFunc{
			"ZipFile":      {GoFunc: "__gopy_zipfile_unused"},
			"ZipInfo":      {GoFunc: "__gopy_zipfile_unused"},
			"is_zipfile":   {GoFunc: "__gopy_zipfile_unused"},
			"Path":         {GoFunc: "__gopy_zipfile_unused"},
			"PyZipFile":    {GoFunc: "__gopy_zipfile_unused"},
			"BadZipFile":   {GoFunc: "__gopy_zipfile_unused"},
			"BadZipfile":   {GoFunc: "__gopy_zipfile_unused"},
			"LargeZipFile": {GoFunc: "__gopy_zipfile_unused"},
		},
	},
	"wave": {
		Funcs: map[string]stdlibFunc{
			"open":      {GoFunc: "__gopy_wave_unused"},
			"Wave_read":  {GoFunc: "__gopy_wave_unused"},
			"Wave_write": {GoFunc: "__gopy_wave_unused"},
			"Error":      {GoFunc: "__gopy_wave_unused"},
		},
	},
	"mailbox": {
		Funcs: map[string]stdlibFunc{
			"Mailbox":  {GoFunc: "__gopy_mailbox_unused"},
			"Maildir":  {GoFunc: "__gopy_mailbox_unused"},
			"mbox":     {GoFunc: "__gopy_mailbox_unused"},
			"MH":       {GoFunc: "__gopy_mailbox_unused"},
			"Babyl":    {GoFunc: "__gopy_mailbox_unused"},
			"MMDF":     {GoFunc: "__gopy_mailbox_unused"},
			"Message":  {GoFunc: "__gopy_mailbox_unused"},
			"MaildirMessage": {GoFunc: "__gopy_mailbox_unused"},
			"mboxMessage":    {GoFunc: "__gopy_mailbox_unused"},
			"MHMessage":      {GoFunc: "__gopy_mailbox_unused"},
			"BabylMessage":   {GoFunc: "__gopy_mailbox_unused"},
			"MMDFMessage":    {GoFunc: "__gopy_mailbox_unused"},
			"Error":          {GoFunc: "__gopy_mailbox_unused"},
			"NoSuchMailboxError": {GoFunc: "__gopy_mailbox_unused"},
			"NotEmptyError":      {GoFunc: "__gopy_mailbox_unused"},
			"ExternalClashError": {GoFunc: "__gopy_mailbox_unused"},
			"FormatError":        {GoFunc: "__gopy_mailbox_unused"},
		},
	},
	"optparse": {
		Funcs: map[string]stdlibFunc{
			"OptionParser":   {GoFunc: "__gopy_optparse_unused"},
			"Option":         {GoFunc: "__gopy_optparse_unused"},
			"OptionGroup":    {GoFunc: "__gopy_optparse_unused"},
			"OptionContainer": {GoFunc: "__gopy_optparse_unused"},
			"Values":         {GoFunc: "__gopy_optparse_unused"},
			"OptionError":    {GoFunc: "__gopy_optparse_unused"},
			"OptionValueError": {GoFunc: "__gopy_optparse_unused"},
			"OptionConflictError": {GoFunc: "__gopy_optparse_unused"},
			"BadOptionError": {GoFunc: "__gopy_optparse_unused"},
			"AmbiguousOptionError": {GoFunc: "__gopy_optparse_unused"},
			"HelpFormatter":  {GoFunc: "__gopy_optparse_unused"},
			"IndentedHelpFormatter": {GoFunc: "__gopy_optparse_unused"},
			"TitledHelpFormatter":   {GoFunc: "__gopy_optparse_unused"},
		},
	},
	"pstats": {
		Funcs: map[string]stdlibFunc{
			"Stats":          {GoFunc: "__gopy_pstats_unused"},
			"SortKey":        {GoFunc: "__gopy_pstats_unused"},
			"StatsProfile":   {GoFunc: "__gopy_pstats_unused"},
			"FunctionProfile": {GoFunc: "__gopy_pstats_unused"},
		},
	},
	"reprlib": {
		Funcs: map[string]stdlibFunc{
			"Repr":        {GoFunc: "__gopy_reprlib_unused"},
			"repr":        {GoFunc: "__gopy_reprlib_repr_unused"},
			"recursive_repr": {GoFunc: "__gopy_reprlib_recursive_unused"},
		},
	},
	"sched": {
		Funcs: map[string]stdlibFunc{
			"scheduler": {GoFunc: "__gopy_sched_unused"},
		},
	},
	"select": {
		Attrs: map[string]stdlibAttr{
			"PIPE_BUF": {GoExpr: "int64(4096)"},
		},
		Funcs: map[string]stdlibFunc{
			"select":  {GoFunc: "__gopy_select_unused"},
			"poll":    {GoFunc: "__gopy_select_unused"},
			"epoll":   {GoFunc: "__gopy_select_unused"},
			"kqueue":  {GoFunc: "__gopy_select_unused"},
			"kevent":  {GoFunc: "__gopy_select_unused"},
			"devpoll": {GoFunc: "__gopy_select_unused"},
			"error":   {GoFunc: "__gopy_select_unused"},
		},
	},
	"shelve": {
		Funcs: map[string]stdlibFunc{
			"open":     {GoFunc: "__gopy_shelve_unused"},
			"Shelf":    {GoFunc: "__gopy_shelve_unused"},
			"BsdDbShelf": {GoFunc: "__gopy_shelve_unused"},
			"DbfilenameShelf": {GoFunc: "__gopy_shelve_unused"},
		},
	},
	"sqlite3": {
		Attrs: map[string]stdlibAttr{
			"version":         {GoExpr: `"3.0.0"`},
			"sqlite_version":  {GoExpr: `"3.42.0"`},
			"version_info":    {GoExpr: `[]int64{3, 0, 0}`},
			"sqlite_version_info": {GoExpr: `[]int64{3, 42, 0}`},
			"PARSE_DECLTYPES": {GoExpr: "int64(1)"},
			"PARSE_COLNAMES":  {GoExpr: "int64(2)"},
			"SQLITE_OK":       {GoExpr: "int64(0)"},
			"SQLITE_DENY":     {GoExpr: "int64(1)"},
			"SQLITE_IGNORE":   {GoExpr: "int64(2)"},
			"threadsafety":    {GoExpr: "int64(3)"},
			"paramstyle":      {GoExpr: `"qmark"`},
		},
		Funcs: map[string]stdlibFunc{
			"connect":     {GoFunc: "__gopy_sqlite3_unused"},
			"Connection":  {GoFunc: "__gopy_sqlite3_unused"},
			"Cursor":      {GoFunc: "__gopy_sqlite3_unused"},
			"Row":         {GoFunc: "__gopy_sqlite3_unused"},
			"Binary":      {GoFunc: "__gopy_sqlite3_unused"},
			"Date":        {GoFunc: "__gopy_sqlite3_unused"},
			"Time":        {GoFunc: "__gopy_sqlite3_unused"},
			"Timestamp":   {GoFunc: "__gopy_sqlite3_unused"},
			"DateFromTicks": {GoFunc: "__gopy_sqlite3_unused"},
			"TimeFromTicks": {GoFunc: "__gopy_sqlite3_unused"},
			"TimestampFromTicks": {GoFunc: "__gopy_sqlite3_unused"},
			"register_adapter":  {GoFunc: "__gopy_sqlite3_unused"},
			"register_converter": {GoFunc: "__gopy_sqlite3_unused"},
			"complete_statement": {GoFunc: "__gopy_sqlite3_unused"},
			"enable_callback_tracebacks": {GoFunc: "__gopy_sqlite3_unused"},
			"Warning":     {GoFunc: "__gopy_sqlite3_unused"},
			"Error":       {GoFunc: "__gopy_sqlite3_unused"},
			"InterfaceError": {GoFunc: "__gopy_sqlite3_unused"},
			"DatabaseError":  {GoFunc: "__gopy_sqlite3_unused"},
			"DataError":      {GoFunc: "__gopy_sqlite3_unused"},
			"OperationalError": {GoFunc: "__gopy_sqlite3_unused"},
			"IntegrityError":   {GoFunc: "__gopy_sqlite3_unused"},
			"InternalError":    {GoFunc: "__gopy_sqlite3_unused"},
			"ProgrammingError": {GoFunc: "__gopy_sqlite3_unused"},
			"NotSupportedError": {GoFunc: "__gopy_sqlite3_unused"},
		},
	},
	"stringprep": {
		Funcs: map[string]stdlibFunc{
			"in_table_a1":  {GoFunc: "__gopy_stringprep_unused"},
			"in_table_b1":  {GoFunc: "__gopy_stringprep_unused"},
			"map_table_b2": {GoFunc: "__gopy_stringprep_unused"},
			"map_table_b3": {GoFunc: "__gopy_stringprep_unused"},
			"in_table_c11": {GoFunc: "__gopy_stringprep_unused"},
			"in_table_c12": {GoFunc: "__gopy_stringprep_unused"},
			"in_table_c11_c12": {GoFunc: "__gopy_stringprep_unused"},
			"in_table_c21": {GoFunc: "__gopy_stringprep_unused"},
			"in_table_c22": {GoFunc: "__gopy_stringprep_unused"},
			"in_table_c21_c22": {GoFunc: "__gopy_stringprep_unused"},
			"in_table_c3":  {GoFunc: "__gopy_stringprep_unused"},
			"in_table_c4":  {GoFunc: "__gopy_stringprep_unused"},
			"in_table_c5":  {GoFunc: "__gopy_stringprep_unused"},
			"in_table_c6":  {GoFunc: "__gopy_stringprep_unused"},
			"in_table_c7":  {GoFunc: "__gopy_stringprep_unused"},
			"in_table_c8":  {GoFunc: "__gopy_stringprep_unused"},
			"in_table_c9":  {GoFunc: "__gopy_stringprep_unused"},
			"in_table_d1":  {GoFunc: "__gopy_stringprep_unused"},
			"in_table_d2":  {GoFunc: "__gopy_stringprep_unused"},
		},
	},
	"tokenize": {
		Funcs: map[string]stdlibFunc{
			"tokenize":   {GoFunc: "__gopy_tokenize_unused"},
			"untokenize": {GoFunc: "__gopy_tokenize_unused"},
			"detect_encoding": {GoFunc: "__gopy_tokenize_unused"},
			"open":       {GoFunc: "__gopy_tokenize_unused"},
			"generate_tokens": {GoFunc: "__gopy_tokenize_unused"},
			"TokenInfo":  {GoFunc: "__gopy_tokenize_unused"},
			"TokenizerError": {GoFunc: "__gopy_tokenize_unused"},
			"TokenError": {GoFunc: "__gopy_tokenize_unused"},
		},
	},
	"tomllib": {
		Funcs: map[string]stdlibFunc{
			"load":          {GoFunc: "__gopy_tomllib_unused"},
			"loads":         {GoFunc: "__gopy_tomllib_unused"},
			"TOMLDecodeError": {GoFunc: "__gopy_tomllib_unused"},
		},
	},
	"zipapp": {
		Funcs: map[string]stdlibFunc{
			"create_archive": {GoFunc: "__gopy_zipapp_unused"},
			"get_interpreter": {GoFunc: "__gopy_zipapp_unused"},
			"ZipAppError":    {GoFunc: "__gopy_zipapp_unused"},
		},
	},
	"zoneinfo": {
		Funcs: map[string]stdlibFunc{
			"ZoneInfo":     {GoFunc: "__gopy_zoneinfo_zone", Helper: helperZoneinfoZone, HelperImports: []string{"time"}, RetKind: "str"},
			"available_timezones": {GoFunc: "__gopy_zoneinfo_unused"},
			"reset_tzpath": {GoFunc: "__gopy_zoneinfo_unused"},
			"TZPATH":       {GoFunc: "__gopy_zoneinfo_unused"},
			"InvalidTZPathWarning": {GoFunc: "__gopy_zoneinfo_unused"},
			"ZoneInfoNotFoundError": {GoFunc: "__gopy_zoneinfo_unused"},
		},
	},
	"webbrowser": {
		Funcs: map[string]stdlibFunc{
			"open":        {GoFunc: "__gopy_webbrowser_open", Helper: helperWebbrowserOpen, HelperImports: []string{"os/exec", "runtime"}, RetKind: "bool"},
			"open_new":    {GoFunc: "__gopy_webbrowser_open", Helper: helperWebbrowserOpen, HelperImports: []string{"os/exec", "runtime"}, RetKind: "bool"},
			"open_new_tab": {GoFunc: "__gopy_webbrowser_open", Helper: helperWebbrowserOpen, HelperImports: []string{"os/exec", "runtime"}, RetKind: "bool"},
			"get":         {GoFunc: "__gopy_webbrowser_unused"},
			"register":    {GoFunc: "__gopy_webbrowser_unused"},
			"Error":       {GoFunc: "__gopy_webbrowser_unused"},
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
const helperReFindall = `func __gopy_re_findall(pattern, s string, flags ...int64) []string {
	r := regexp.MustCompile(__gopy_re_flag_prefix(flags...) + pattern)
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

// helperReFlagPrefix translates Python re flag bits into Go regexp's
// embedded flag syntax. IGNORECASE=2 → (?i), MULTILINE=8 → (?m),
// DOTALL=16 → (?s); other flags are silently ignored (Go regexp's
// syntax differs from Python's so e.g. VERBOSE doesn't translate).
const helperReFlagPrefix = `func __gopy_re_flag_prefix(flags ...int64) string {
	var bits int64
	for _, f := range flags {
		bits |= f
	}
	if bits == 0 {
		return ""
	}
	out := "(?"
	if bits&2 != 0 {
		out += "i"
	}
	if bits&8 != 0 {
		out += "m"
	}
	if bits&16 != 0 {
		out += "s"
	}
	if out == "(?" {
		return ""
	}
	return out + ")"
}`

// helperReSearch returns a *__Match on hit, nil on miss — mirroring
// Python's re.search semantics. Truthy / `is None` checks at call sites
// work because the codegen rewrites them to a nil comparison. Trailing
// variadic flag args are mapped to Go regexp's (?i)/(?m)/(?s) syntax.
const helperReSearch = `func __gopy_re_search(pattern, s string, flags ...int64) *__Match {
	r := regexp.MustCompile(__gopy_re_flag_prefix(flags...) + pattern)
	return __gopy_match_build(r, s, false)
}`

// helperReMatch anchors the pattern to the start of the string, matching
// Python's re.match semantics. Returns nil on miss.
const helperReMatch = `func __gopy_re_match(pattern, s string, flags ...int64) *__Match {
	r := regexp.MustCompile("^(?:" + __gopy_re_flag_prefix(flags...) + pattern + ")")
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

// helperReSub replaces every match of pattern with repl. Trailing
// int args are interpreted as (count, flags) per Python's
// `re.sub(pattern, repl, string, count=0, flags=0)` signature: a
// non-zero count caps the number of substitutions.
const helperReSub = `func __gopy_re_sub(pattern, repl, s string, args ...int64) string {
	var count, flagBits int64
	if len(args) >= 1 {
		count = args[0]
	}
	if len(args) >= 2 {
		flagBits = args[1]
	}
	r := regexp.MustCompile(__gopy_re_flag_prefix(flagBits) + pattern)
	if count <= 0 {
		return r.ReplaceAllString(s, repl)
	}
	remaining := count
	return r.ReplaceAllStringFunc(s, func(match string) string {
		if remaining <= 0 {
			return match
		}
		remaining--
		return r.ReplaceAllString(match, repl)
	})
}`

// helperReSubn returns []any{result_string, n_substitutions} so callers
// can unpack via positional indexing. Mirrors Python's re.subn tuple.
const helperReSubn = `func __gopy_re_subn(pattern, repl, s string, args ...int64) []any {
	var count, flagBits int64
	if len(args) >= 1 {
		count = args[0]
	}
	if len(args) >= 2 {
		flagBits = args[1]
	}
	r := regexp.MustCompile(__gopy_re_flag_prefix(flagBits) + pattern)
	n := int64(len(r.FindAllStringIndex(s, -1)))
	if count <= 0 || count >= n {
		return []any{r.ReplaceAllString(s, repl), n}
	}
	remaining := count
	out := r.ReplaceAllStringFunc(s, func(match string) string {
		if remaining <= 0 {
			return match
		}
		remaining--
		return r.ReplaceAllString(match, repl)
	})
	return []any{out, count}
}`

// helperReFullmatch anchors at both ends, mirroring re.fullmatch.
const helperReFullmatch = `func __gopy_re_fullmatch(pattern, s string, flags ...int64) *__Match {
	r := regexp.MustCompile("^(?:" + __gopy_re_flag_prefix(flags...) + pattern + ")$")
	return __gopy_match_build(r, s, false)
}`

// helperReSplit splits s on every occurrence of the pattern. Mirrors
// re.split's default form; the maxsplit argument is not supported (use
// strings.SplitN with a literal sep for that pattern).
const helperReSplit = `func __gopy_re_split(pattern, s string, flags ...int64) []string {
	r := regexp.MustCompile(__gopy_re_flag_prefix(flags...) + pattern)
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

const helperReCompile = `func __gopy_re_compile(pattern string, flags ...int64) *__Pattern {
	pref := __gopy_re_flag_prefix(flags...)
	return &__Pattern{
		r:      regexp.MustCompile(pref + pattern),
		anchor: regexp.MustCompile("^(?:" + pref + pattern + ")"),
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
	case "sha224":
		sum := sha256.Sum224(h.data)
		return hex.EncodeToString(sum[:])
	case "sha512":
		sum := sha512.Sum512(h.data)
		return hex.EncodeToString(sum[:])
	case "sha384":
		sum := sha512.Sum384(h.data)
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

const helperURLOpen = `func __gopy_url_urlopen(url string, opts ...any) *__HTTPResponse {
	var resp *http.Response
	var err error
	method := "GET"
	body := ""
	for _, o := range opts {
		switch v := o.(type) {
		case string:
			body = v
			method = "POST"
		case []byte:
			body = string(v)
			method = "POST"
		case map[string]any:
			if d, ok := v["data"]; ok && d != nil {
				if s, ok := d.(string); ok {
					body = s
					method = "POST"
				}
			}
			if m, ok := v["method"].(string); ok {
				method = strings.ToUpper(m)
			}
		}
	}
	if method != "GET" {
		req, rerr := http.NewRequest(method, url, strings.NewReader(body))
		if rerr != nil {
			panic(NewException("URLError: " + rerr.Error()))
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err = http.DefaultClient.Do(req)
	} else {
		resp, err = http.Get(url)
	}
	if err != nil {
		panic(NewException("URLError: " + err.Error()))
	}
	defer resp.Body.Close()
	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(NewException("URLError: " + err.Error()))
	}
	headers := map[string]string{}
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	return &__HTTPResponse{body: string(rb), Status: int64(resp.StatusCode), Headers: headers}
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
// same shape so fixtures comparing stderr can round-trip. The module-level
// __gopy_log_level threshold gates emission: calls below the threshold
// are dropped (matching CPython's root logger behavior, default WARNING).
const helperLogState = `var __gopy_log_level int64 = 30

func __gopy_log_should(level int64) bool { return level >= __gopy_log_level }`

const helperLogDebug = `func __gopy_log_debug(msg string) {
	if __gopy_log_should(10) {
		fmt.Fprintln(os.Stderr, "DEBUG:root:"+msg)
	}
}`
const helperLogInfo = `func __gopy_log_info(msg string) {
	if __gopy_log_should(20) {
		fmt.Fprintln(os.Stderr, "INFO:root:"+msg)
	}
}`
const helperLogWarning = `func __gopy_log_warning(msg string) {
	if __gopy_log_should(30) {
		fmt.Fprintln(os.Stderr, "WARNING:root:"+msg)
	}
}`
const helperLogError = `func __gopy_log_error(msg string) {
	if __gopy_log_should(40) {
		fmt.Fprintln(os.Stderr, "ERROR:root:"+msg)
	}
}`
const helperLogCritical = `func __gopy_log_critical(msg string) {
	if __gopy_log_should(50) {
		fmt.Fprintln(os.Stderr, "CRITICAL:root:"+msg)
	}
}`
const helperLogBasicConfig = `func __gopy_log_basicConfig(args ...int64) {
	if len(args) > 0 {
		__gopy_log_level = args[0]
	}
}`

const helperLoggerType = `type __Logger struct {
	name  string
	level int64
}

func (l *__Logger) shouldLog(level int64) bool {
	eff := l.level
	if eff == 0 {
		eff = __gopy_log_level
	}
	return level >= eff
}

func (l *__Logger) SetLevel(level int64) { l.level = level }

func (l *__Logger) GetEffectiveLevel() int64 {
	if l.level != 0 {
		return l.level
	}
	return __gopy_log_level
}

func (l *__Logger) IsEnabledFor(level int64) bool { return l.shouldLog(level) }

func (l *__Logger) Debug(msg string) {
	if l.shouldLog(10) {
		fmt.Fprintln(os.Stderr, "DEBUG:"+l.name+":"+msg)
	}
}
func (l *__Logger) Info(msg string) {
	if l.shouldLog(20) {
		fmt.Fprintln(os.Stderr, "INFO:"+l.name+":"+msg)
	}
}
func (l *__Logger) Warning(msg string) {
	if l.shouldLog(30) {
		fmt.Fprintln(os.Stderr, "WARNING:"+l.name+":"+msg)
	}
}
func (l *__Logger) Error(msg string) {
	if l.shouldLog(40) {
		fmt.Fprintln(os.Stderr, "ERROR:"+l.name+":"+msg)
	}
}
func (l *__Logger) Critical(msg string) {
	if l.shouldLog(50) {
		fmt.Fprintln(os.Stderr, "CRITICAL:"+l.name+":"+msg)
	}
}`

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

const helperGlob = `func __gopy_glob(pattern string, opts ...any) []string {
	recursive := false
	for _, o := range opts {
		if m, ok := o.(map[string]any); ok {
			if v, ok := m["recursive"].(bool); ok {
				recursive = v
			}
		}
	}
	if recursive && strings.Contains(pattern, "**") {
		// Split on "**" — descend through every subdirectory beneath the
		// prefix, then match the suffix against each candidate. Mirrors
		// CPython recursive glob for common dir/**/*.ext shapes.
		idx := strings.Index(pattern, "**")
		prefix := pattern[:idx]
		suffix := pattern[idx+2:]
		suffix = strings.TrimLeft(suffix, "/")
		root := strings.TrimRight(prefix, "/")
		if root == "" {
			root = "."
		}
		out := []string{}
		_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			candidate := p
			if suffix != "" {
				if d.IsDir() {
					return nil
				}
				ok, _ := filepath.Match(suffix, d.Name())
				if !ok {
					return nil
				}
			}
			out = append(out, candidate)
			return nil
		})
		return out
	}
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

const helperShutilCopytree = `func __gopy_shutil_copytree(src, dst string) string {
	si, err := os.Stat(src)
	if err != nil {
		panic(err)
	}
	if !si.IsDir() {
		panic(NewException("NotADirectoryError: " + src))
	}
	if err := os.MkdirAll(dst, si.Mode()); err != nil {
		panic(err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		panic(err)
	}
	for _, e := range entries {
		srcChild := src + "/" + e.Name()
		dstChild := dst + "/" + e.Name()
		if e.IsDir() {
			__gopy_shutil_copytree(srcChild, dstChild)
			continue
		}
		__gopy_shutil_copy(srcChild, dstChild)
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

// helperTypingNewtype — NewType returns a callable identity. In gopy
// the result is a closure that returns its argument unchanged.
const helperTypingNewtype = `func __gopy_typing_newtype(args ...any) func(any) any {
	return func(v any) any { return v }
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
	Type    string
}

type __ArgParser struct {
	Specs []__ArgSpec
}

type __ArgNamespace struct {
	Values map[string]any
}

func (n *__ArgNamespace) Get(name string) any { return n.Values[name] }

func __gopy_argparse_convert(t, raw string) any {
	switch t {
	case "int":
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return v
		}
		return int64(0)
	case "float":
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			return v
		}
		return float64(0)
	case "bool":
		if v, err := strconv.ParseBool(raw); err == nil {
			return v
		}
		return false
	}
	return raw
}

func (p *__ArgParser) AddArgument(args ...any) {
	var kwargs map[string]any
	if n := len(args); n > 0 {
		if kv, ok := args[n-1].(map[string]any); ok {
			kwargs = kv
			args = args[:n-1]
		}
	}
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
	if kwargs != nil {
		if t, ok := kwargs["type"].(string); ok {
			spec.Type = t
		}
		if d, ok := kwargs["default"]; ok {
			spec.Default = d
		}
		if a, ok := kwargs["action"].(string); ok {
			spec.Action = a
			if a == "store_true" || a == "store_false" {
				spec.IsFlag = true
			}
		}
		if dn, ok := kwargs["dest"].(string); ok && dn != "" {
			spec.Name = dn
		}
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
		switch {
		case s.Default != nil:
			ns.Values[s.Name] = s.Default
		case s.IsFlag:
			if s.Action == "store_false" {
				ns.Values[s.Name] = true
			} else {
				ns.Values[s.Name] = false
			}
		case s.Type == "int":
			ns.Values[s.Name] = int64(0)
		case s.Type == "float":
			ns.Values[s.Name] = float64(0)
		case s.Type == "bool":
			ns.Values[s.Name] = false
		default:
			ns.Values[s.Name] = ""
		}
	}
	specByName := map[string]__ArgSpec{}
	specByShort := map[string]__ArgSpec{}
	for _, s := range p.Specs {
		if s.Name != "" {
			specByName[s.Name] = s
		}
		if s.Short != "" {
			specByShort[s.Short] = s
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
			haveVal := false
			if eq >= 0 {
				name = tok[2:eq]
				val = tok[eq+1:]
				haveVal = true
			} else {
				name = tok[2:]
			}
			spec, known := specByName[name]
			if known && spec.IsFlag {
				if spec.Action == "store_false" {
					ns.Values[name] = false
				} else {
					ns.Values[name] = true
				}
				i++
				continue
			}
			if !haveVal {
				if i+1 < len(argv) {
					val = argv[i+1]
					i++
				}
			}
			if known && spec.Type != "" {
				ns.Values[name] = __gopy_argparse_convert(spec.Type, val)
			} else if v, err := strconv.ParseInt(val, 10, 64); err == nil {
				ns.Values[name] = v
			} else {
				ns.Values[name] = val
			}
		} else if strings.HasPrefix(tok, "-") && len(tok) >= 2 {
			short := tok[1:]
			spec, known := specByShort[short]
			if known && spec.IsFlag {
				if spec.Action == "store_false" {
					ns.Values[spec.Name] = false
				} else {
					ns.Values[spec.Name] = true
				}
				i++
				continue
			}
			if known {
				if i+1 < len(argv) {
					raw := argv[i+1]
					if spec.Type != "" {
						ns.Values[spec.Name] = __gopy_argparse_convert(spec.Type, raw)
					} else {
						ns.Values[spec.Name] = raw
					}
					i++
				}
			}
		} else if posIdx < len(posSpecs) {
			spec := posSpecs[posIdx]
			if spec.Type != "" {
				ns.Values[spec.Name] = __gopy_argparse_convert(spec.Type, tok)
			} else {
				ns.Values[spec.Name] = tok
			}
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

// interpolate resolves CPython-style %(key)s placeholders. Lookup
// walks the value's home section first, falls back to DEFAULT. Max
// 10 expansion passes guards against accidental cycles.
func (p *__ConfigParser) interpolate(section, raw string) string {
	cur := raw
	for pass := 0; pass < 10; pass++ {
		next := cur
		i := 0
		for i < len(next) {
			if next[i] != '%' || i+1 >= len(next) {
				i++
				continue
			}
			if next[i+1] == '(' {
				end := strings.Index(next[i+2:], ")s")
				if end < 0 {
					i++
					continue
				}
				key := next[i+2 : i+2+end]
				var val string
				if s, ok := p.data[section]; ok {
					if v, ok2 := s[key]; ok2 {
						val = v
					}
				}
				if val == "" {
					if s, ok := p.data["DEFAULT"]; ok {
						if v, ok2 := s[key]; ok2 {
							val = v
						}
					}
				}
				next = next[:i] + val + next[i+2+end+2:]
				continue
			}
			if next[i+1] == '%' {
				next = next[:i] + "%" + next[i+2:]
				i++
				continue
			}
			i++
		}
		if next == cur {
			return cur
		}
		cur = next
	}
	return cur
}

func (p *__ConfigParser) Get(section, key string) string {
	if s, ok := p.data[section]; ok {
		if v, ok2 := s[key]; ok2 {
			return p.interpolate(section, v)
		}
	}
	if s, ok := p.data["DEFAULT"]; ok {
		if v, ok2 := s[key]; ok2 {
			return p.interpolate(section, v)
		}
	}
	return ""
}

func (p *__ConfigParser) Getint(section, key string) int64 {
	v := p.Get(section, key)
	n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	return n
}

func (p *__ConfigParser) Getfloat(section, key string) float64 {
	v := p.Get(section, key)
	f, _ := strconv.ParseFloat(strings.TrimSpace(v), 64)
	return f
}

func (p *__ConfigParser) Getboolean(section, key string) bool {
	v := strings.ToLower(strings.TrimSpace(p.Get(section, key)))
	switch v {
	case "1", "yes", "true", "on":
		return true
	}
	return false
}

func (p *__ConfigParser) Options(section string) []string {
	out := []string{}
	if s, ok := p.data[section]; ok {
		for k := range s {
			out = append(out, k)
		}
	}
	if s, ok := p.data["DEFAULT"]; ok && section != "DEFAULT" {
		for k := range s {
			out = append(out, k)
		}
	}
	return out
}

func (p *__ConfigParser) Set(section, key, value string) {
	p.ensure()
	if _, ok := p.data[section]; !ok {
		p.data[section] = map[string]string{}
	}
	p.data[section][key] = value
}

func (p *__ConfigParser) Add_section(name string) {
	p.ensure()
	if _, ok := p.data[name]; !ok {
		p.data[name] = map[string]string{}
	}
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
	Tag         string
	Text        string
	Attrib      map[string]string
	attribOrder []string
	Children    []*__XMLElement
}

func (e *__XMLElement) attribSet(k, v string) {
	if e.Attrib == nil {
		e.Attrib = map[string]string{}
	}
	if _, ok := e.Attrib[k]; !ok {
		e.attribOrder = append(e.attribOrder, k)
	}
	e.Attrib[k] = v
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

func (e *__XMLElement) Set(key, value string) { e.attribSet(key, value) }

func (e *__XMLElement) Append(child *__XMLElement) {
	e.Children = append(e.Children, child)
}

func (e *__XMLElement) Remove(child *__XMLElement) {
	for i, c := range e.Children {
		if c == child {
			e.Children = append(e.Children[:i], e.Children[i+1:]...)
			return
		}
	}
}

func (e *__XMLElement) Insert(idx int64, child *__XMLElement) {
	if idx < 0 {
		idx = 0
	}
	if idx >= int64(len(e.Children)) {
		e.Children = append(e.Children, child)
		return
	}
	e.Children = append(e.Children, nil)
	copy(e.Children[idx+1:], e.Children[idx:])
	e.Children[idx] = child
}

func (e *__XMLElement) Keys() []string {
	out := []string{}
	for k := range e.Attrib {
		out = append(out, k)
	}
	return out
}

func (e *__XMLElement) Items() [][]string {
	out := [][]string{}
	for k, v := range e.Attrib {
		out = append(out, []string{k, v})
	}
	return out
}


func __gopy_xml_build(d *xml.Decoder, start *xml.StartElement) *__XMLElement {
	el := &__XMLElement{Tag: start.Name.Local, Attrib: map[string]string{}}
	for _, a := range start.Attr {
		el.attribSet(a.Name.Local, a.Value)
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

const helperXMLSerialize = `func __gopy_xml_serialize(e *__XMLElement, b *strings.Builder) {
	b.WriteByte('<')
	b.WriteString(e.Tag)
	if len(e.Attrib) > 0 {
		keys := e.attribOrder
		if len(keys) != len(e.Attrib) {
			keys = make([]string, 0, len(e.Attrib))
			for k := range e.Attrib {
				keys = append(keys, k)
			}
			sort.Strings(keys)
		}
		for _, k := range keys {
			b.WriteByte(' ')
			b.WriteString(k)
			b.WriteString("=\"")
			b.WriteString(__gopy_xml_escape(e.Attrib[k]))
			b.WriteByte('"')
		}
	}
	if len(e.Children) == 0 && e.Text == "" {
		b.WriteString(" />")
		return
	}
	b.WriteByte('>')
	if e.Text != "" {
		b.WriteString(__gopy_xml_escape(e.Text))
	}
	for _, c := range e.Children {
		__gopy_xml_serialize(c, b)
	}
	b.WriteString("</")
	b.WriteString(e.Tag)
	b.WriteByte('>')
}

func __gopy_xml_escape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")
	return r.Replace(s)
}`

const helperXMLTostring = `func __gopy_xml_tostring(e *__XMLElement, args ...string) string {
	if e == nil {
		return ""
	}
	var b strings.Builder
	__gopy_xml_serialize(e, &b)
	return b.String()
}`

const helperXMLElement = `func __gopy_xml_element(tag string, args ...any) *__XMLElement {
	el := &__XMLElement{Tag: tag, Attrib: map[string]string{}}
	for _, a := range args {
		if m, ok := a.(map[string]string); ok {
			for k, v := range m {
				el.attribSet(k, v)
			}
		}
		if m, ok := a.(map[string]any); ok {
			for k, v := range m {
				el.attribSet(k, fmt.Sprintf("%v", v))
			}
		}
	}
	return el
}`

const helperXMLSubElement = `func __gopy_xml_subelement(parent *__XMLElement, tag string, args ...any) *__XMLElement {
	child := __gopy_xml_element(tag, args...)
	parent.Children = append(parent.Children, child)
	return child
}`

const helperXMLTreeType = `type __XMLTree struct {
	root *__XMLElement
}

func (t *__XMLTree) Getroot() *__XMLElement { return t.root }

func (t *__XMLTree) Write(path string, args ...any) {
	if t.root == nil {
		return
	}
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	__gopy_xml_serialize(t.root, &b)
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		panic(err)
	}
}`

const helperXMLParse = `func __gopy_xml_parse(path string) *__XMLTree {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	root := __gopy_xml_fromstring(string(data))
	return &__XMLTree{root: root}
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
func (f *__Fraction) Ne(o *__Fraction) bool { return f.Num*o.Den != o.Num*f.Den }
func (f *__Fraction) Lt(o *__Fraction) bool { return f.Num*o.Den < o.Num*f.Den }
func (f *__Fraction) Le(o *__Fraction) bool { return f.Num*o.Den <= o.Num*f.Den }
func (f *__Fraction) Gt(o *__Fraction) bool { return f.Num*o.Den > o.Num*f.Den }
func (f *__Fraction) Ge(o *__Fraction) bool { return f.Num*o.Den >= o.Num*f.Den }
func (f *__Fraction) Float() float64        { return float64(f.Num) / float64(f.Den) }`

const helperFractionNew = `func __gopy_fraction_new(args ...any) *__Fraction {
	asInt := func(v any) (int64, bool) {
		switch x := v.(type) {
		case int64:
			return x, true
		case int:
			return int64(x), true
		case int32:
			return int64(x), true
		case float64:
			return int64(x), true
		}
		return 0, false
	}
	if len(args) == 0 {
		return &__Fraction{Num: 0, Den: 1}
	}
	if len(args) == 1 {
		if n, ok := asInt(args[0]); ok {
			return &__Fraction{Num: n, Den: 1}
		}
		if v, ok := args[0].(string); ok {
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
	n, _ := asInt(args[0])
	d, _ := asInt(args[1])
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

func (d *__Decimal) Eq(o *__Decimal) bool { return d.V == o.V }
func (d *__Decimal) Ne(o *__Decimal) bool { return d.V != o.V }
func (d *__Decimal) Lt(o *__Decimal) bool { return d.V < o.V }
func (d *__Decimal) Le(o *__Decimal) bool { return d.V <= o.V }
func (d *__Decimal) Gt(o *__Decimal) bool { return d.V > o.V }
func (d *__Decimal) Ge(o *__Decimal) bool { return d.V >= o.V }

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

// helperWarningsWarn writes message to stderr; matches CPython's
// default warning stream. filterwarnings / simplefilter accepted as
// no-ops since gopy doesn't apply filters globally.
const helperWarningsWarn = `func __gopy_warnings_warn(args ...any) {
	if len(args) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, "Warning:", fmt.Sprintf("%v", args[0]))
}`

const helperWarningsNoop = `func __gopy_warnings_noop(args ...any) {}`

// helperGettext — identity / second-arg pass-through. gopy doesn't
// load .mo translation catalogs; the message string is returned as-is.
const helperGettextIdentity = `func __gopy_gettext_identity(s string) string { return s }`

const helperGettextN = `func __gopy_gettext_n(args ...any) string {
	if len(args) < 3 {
		if len(args) > 0 {
			s, _ := args[0].(string)
			return s
		}
		return ""
	}
	var n int64
	switch v := args[2].(type) {
	case int64:
		n = v
	case int:
		n = int64(v)
	case float64:
		n = int64(v)
	}
	if n == 1 {
		s, _ := args[0].(string)
		return s
	}
	s, _ := args[1].(string)
	return s
}`

// helperLocale — gopy doesn't honor C locale settings. setlocale
// echoes the requested locale name; getlocale returns ("C", "UTF-8").
const helperLocaleSetlocale = `func __gopy_locale_setlocale(args ...any) string {
	if len(args) >= 2 {
		if s, ok := args[1].(string); ok {
			return s
		}
	}
	return "C"
}`

const helperLocaleGetlocale = `func __gopy_locale_getlocale(args ...any) []any {
	return []any{"C", "UTF-8"}
}`

// helperColorsys — RGB / HSV / YIQ conversions follow Python's
// reference implementation; inputs and outputs are floats in [0, 1].
const helperColorsysRgbHsv = `func __gopy_colorsys_rgb_hsv(r, g, b float64) []any {
	maxc := math.Max(r, math.Max(g, b))
	minc := math.Min(r, math.Min(g, b))
	v := maxc
	if minc == maxc {
		return []any{0.0, 0.0, v}
	}
	s := (maxc - minc) / maxc
	rc := (maxc - r) / (maxc - minc)
	gc := (maxc - g) / (maxc - minc)
	bc := (maxc - b) / (maxc - minc)
	var h float64
	switch {
	case r == maxc:
		h = bc - gc
	case g == maxc:
		h = 2.0 + rc - bc
	default:
		h = 4.0 + gc - rc
	}
	h = math.Mod(h/6.0+1.0, 1.0)
	return []any{h, s, v}
}`

const helperColorsysHsvRgb = `func __gopy_colorsys_hsv_rgb(h, s, v float64) []any {
	if s == 0 {
		return []any{v, v, v}
	}
	i := int(h * 6.0)
	f := h*6.0 - float64(i)
	p := v * (1.0 - s)
	q := v * (1.0 - s*f)
	t := v * (1.0 - s*(1.0-f))
	switch i % 6 {
	case 0:
		return []any{v, t, p}
	case 1:
		return []any{q, v, p}
	case 2:
		return []any{p, v, t}
	case 3:
		return []any{p, q, v}
	case 4:
		return []any{t, p, v}
	default:
		return []any{v, p, q}
	}
}`

const helperColorsysRgbYiq = `func __gopy_colorsys_rgb_yiq(r, g, b float64) []any {
	y := 0.30*r + 0.59*g + 0.11*b
	i := 0.74*(r-y) - 0.27*(b-y)
	q := 0.48*(r-y) + 0.41*(b-y)
	return []any{y, i, q}
}`

const helperColorsysYiqRgb = `func __gopy_colorsys_yiq_rgb(y, i, q float64) []any {
	r := y + 0.9468822170900693*i + 0.6235565819861433*q
	g := y - 0.27478764629897834*i - 0.6356910791873801*q
	b := y - 1.1085450346420322*i + 1.7090069284064666*q
	return []any{r, g, b}
}`

const helperKeywordIskw = `func __gopy_keyword_iskw(s string) bool {
	for _, k := range []string{"False","None","True","and","as","assert","async","await","break","class","continue","def","del","elif","else","except","finally","for","from","global","if","import","in","is","lambda","nonlocal","not","or","pass","raise","return","try","while","with","yield"} {
		if k == s {
			return true
		}
	}
	return false
}`

const helperKeywordIssoftkw = `func __gopy_keyword_issoftkw(s string) bool {
	return s == "match" || s == "case" || s == "type" || s == "_"
}`

const helperUnicodedataCategory = `func __gopy_unicodedata_category(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)[0]
	switch {
	case unicode.IsLower(r):
		return "Ll"
	case unicode.IsUpper(r):
		return "Lu"
	case unicode.IsTitle(r):
		return "Lt"
	case unicode.IsLetter(r):
		return "Lo"
	case unicode.IsDigit(r):
		return "Nd"
	case unicode.IsNumber(r):
		return "No"
	case unicode.IsSpace(r):
		return "Zs"
	case unicode.IsPunct(r):
		return "Po"
	case unicode.IsControl(r):
		return "Cc"
	case unicode.IsSymbol(r):
		return "So"
	}
	return "Cn"
}`

const helperUnicodedataName = `func __gopy_unicodedata_name(args ...any) string {
	if len(args) > 1 {
		if s, ok := args[1].(string); ok {
			return s
		}
	}
	return ""
}`

const helperDisNoop = `func __gopy_dis_noop(args ...any) {}`
const helperDisInstr = `func __gopy_dis_instr(args ...any) []any { return []any{} }`
const helperDisIsfalse = `func __gopy_dis_isfalse(args ...any) bool { return false }`
const helperDisTracedMem = `func __gopy_dis_traced_mem(args ...any) []any { return []any{int64(0), int64(0)} }`

// helperStat* — mode-bit predicates over int64 file-mode values.
const helperStatIsreg = `func __gopy_stat_isreg(mode int64) bool { return mode&0o170000 == 0o100000 }`
const helperStatIsdir = `func __gopy_stat_isdir(mode int64) bool { return mode&0o170000 == 0o040000 }`
const helperStatIslnk = `func __gopy_stat_islnk(mode int64) bool { return mode&0o170000 == 0o120000 }`
const helperStatIschr = `func __gopy_stat_ischr(mode int64) bool { return mode&0o170000 == 0o020000 }`
const helperStatIsblk = `func __gopy_stat_isblk(mode int64) bool { return mode&0o170000 == 0o060000 }`
const helperStatIsfifo = `func __gopy_stat_isfifo(mode int64) bool { return mode&0o170000 == 0o010000 }`
const helperStatIssock = `func __gopy_stat_issock(mode int64) bool { return mode&0o170000 == 0o140000 }`
const helperStatImode = `func __gopy_stat_imode(mode int64) int64 { return mode & 0o7777 }`
const helperStatIfmt = `func __gopy_stat_ifmt(mode int64) int64 { return mode & 0o170000 }`

// helperFnmatch — backed by filepath.Match (close to fnmatch's *, ?,
// [chars] subset). fnmatchcase = fnmatch (Go is case-sensitive).
const helperFnmatch = `func __gopy_fnmatch(name, pattern string) bool {
	ok, _ := filepath.Match(pattern, name)
	return ok
}`

const helperFnmatchFilter = `func __gopy_fnmatch_filter(names []string, pattern string) []string {
	out := []string{}
	for _, n := range names {
		if ok, _ := filepath.Match(pattern, n); ok {
			out = append(out, n)
		}
	}
	return out
}`

const helperLinecacheGetline = `func __gopy_linecache_getline(filename string, lineno int64) string {
	f, err := os.Open(filename)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	cur := int64(0)
	for sc.Scan() {
		cur++
		if cur == lineno {
			return sc.Text() + "\n"
		}
	}
	return ""
}`

const helperLinecacheGetlines = `func __gopy_linecache_getlines(filename string) []string {
	out := []string{}
	f, err := os.Open(filename)
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		out = append(out, sc.Text()+"\n")
	}
	return out
}`

// helperGetopt — minimal getopt with short options only. Returns
// [opts, args] where opts is [][]any of (flag, value) pairs.
const helperGetopt = `func __gopy_getopt(args ...any) []any {
	argv := []string{}
	if v, ok := args[0].([]string); ok {
		argv = v
	} else if v, ok := args[0].([]any); ok {
		for _, a := range v {
			if s, ok := a.(string); ok {
				argv = append(argv, s)
			}
		}
	}
	short, _ := args[1].(string)
	withVal := map[byte]bool{}
	for i := 0; i < len(short); i++ {
		c := short[i]
		take := i+1 < len(short) && short[i+1] == ':'
		withVal[c] = take
		if take {
			i++
		}
	}
	opts := [][]any{}
	rest := []string{}
	for i := 0; i < len(argv); i++ {
		a := argv[i]
		if !strings.HasPrefix(a, "-") || a == "-" {
			rest = append(rest, a)
			continue
		}
		if a == "--" {
			rest = append(rest, argv[i+1:]...)
			break
		}
		for j := 1; j < len(a); j++ {
			c := a[j]
			if withVal[c] {
				if j+1 < len(a) {
					opts = append(opts, []any{"-" + string(c), a[j+1:]})
					j = len(a)
				} else if i+1 < len(argv) {
					i++
					opts = append(opts, []any{"-" + string(c), argv[i]})
				}
			} else {
				opts = append(opts, []any{"-" + string(c), ""})
			}
		}
	}
	return []any{opts, rest}
}`

const helperTimeitStub = `func __gopy_timeit_stub(args ...any) float64 { return 0.0 }`

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

const helperNamedTempFileType = `type __NamedTempFile struct {
	name string
	f    *os.File
}

func (n *__NamedTempFile) Name() string  { return n.name }
func (n *__NamedTempFile) Write(s string) int64 {
	n.f.Write([]byte(s))
	return int64(len(s))
}
func (n *__NamedTempFile) Read(args ...int64) string {
	b, err := io.ReadAll(n.f)
	if err != nil {
		panic(err)
	}
	return string(b)
}
func (n *__NamedTempFile) Seek(off int64, args ...int64) int64 {
	whence := 0
	if len(args) > 0 {
		whence = int(args[0])
	}
	pos, _ := n.f.Seek(off, whence)
	return pos
}
func (n *__NamedTempFile) Close() {
	if n.f != nil {
		n.f.Close()
		n.f = nil
	}
}
func (n *__NamedTempFile) Flush() {
	if n.f != nil {
		n.f.Sync()
	}
}

func __gopy_named_tempfile_new(prefix, suffix string) (*__NamedTempFile, error) {
	pat := prefix + "*" + suffix
	f, err := os.CreateTemp("", pat)
	if err != nil {
		return nil, err
	}
	return &__NamedTempFile{name: f.Name(), f: f}, nil
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

const helperSubprocessRun = `func __gopy_subprocess_run(args []string, opts ...any) *__CompletedProcess {
	if len(args) == 0 {
		return &__CompletedProcess{Returncode: -1}
	}
	cmd := exec.Command(args[0], args[1:]...)
	for _, o := range opts {
		if kv, ok := o.(map[string]any); ok {
			if v, ok := kv["input"]; ok && v != nil {
				if s, ok := v.(string); ok {
					cmd.Stdin = strings.NewReader(s)
				}
			}
			if v, ok := kv["cwd"]; ok && v != nil {
				if s, ok := v.(string); ok {
					cmd.Dir = s
				}
			}
		}
	}
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
}

func (t *__Template) Get_identifiers() []string {
	seen := map[string]bool{}
	out := []string{}
	s := t.tmpl
	i := 0
	for i < len(s) {
		if s[i] != '$' || i+1 >= len(s) {
			i++
			continue
		}
		if s[i+1] == '$' {
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
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
		if braced && j < len(s) && s[j] == '}' {
			i = j + 1
		} else {
			i = j
		}
	}
	return out
}

func (t *__Template) Is_valid() bool {
	s := t.tmpl
	i := 0
	for i < len(s) {
		if s[i] != '$' {
			i++
			continue
		}
		if i+1 >= len(s) {
			return false
		}
		if s[i+1] == '$' {
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
		if start == j {
			return false
		}
		if braced {
			if j >= len(s) || s[j] != '}' {
				return false
			}
			j++
		}
		i = j
	}
	return true
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

// ReadText / WriteText accept optional trailing string args
// (encoding, errors) for CPython API parity; gopy uses UTF-8 strings
// natively so the encoding hint is informational only.
func (p *__Path) ReadText(args ...string) string {
	b, err := os.ReadFile(p.p)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (p *__Path) WriteText(s string, args ...string) int64 {
	if err := os.WriteFile(p.p, []byte(s), 0o644); err != nil {
		panic(err)
	}
	return int64(len(s))
}

// ReadBytes / WriteBytes mirror their text counterparts. gopy maps
// bytes to Go's string (no separate bytes type) so these end up
// identical to ReadText / WriteText at runtime; they exist as
// distinct methods so source-level p.read_bytes / p.write_bytes
// dispatches via the tagged method table without an unsupported
// error.
func (p *__Path) ReadBytes() string {
	b, err := os.ReadFile(p.p)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (p *__Path) WriteBytes(s string) {
	if err := os.WriteFile(p.p, []byte(s), 0o644); err != nil {
		panic(err)
	}
}

func (p *__Path) String() string { return p.p }

// Match mirrors pathlib.PurePath.match — fnmatch-style glob against the
// path's basename (right-anchored). gopy uses Go's filepath.Match which
// handles *, ?, [..] character classes the same way fnmatch does for
// the patterns most fixtures lean on. Multi-segment patterns ("a/*.py")
// match against the joined path; bare patterns match the basename.
func (p *__Path) Match(pattern string) bool {
	if strings.Contains(pattern, "/") {
		ok, _ := filepath.Match(pattern, p.p)
		return ok
	}
	ok, _ := filepath.Match(pattern, filepath.Base(p.p))
	return ok
}

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

func (p *__Path) Rglob(pattern string) []*__Path {
	out := []*__Path{}
	_ = filepath.WalkDir(p.p, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path == p.p {
			return nil
		}
		ok, mErr := filepath.Match(pattern, d.Name())
		if mErr == nil && ok {
			out = append(out, &__Path{p: path})
		}
		return nil
	})
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
}

func (p *__Path) Absolute() *__Path {
	abs, err := filepath.Abs(p.p)
	if err != nil {
		return p
	}
	return &__Path{p: abs}
}

func (p *__Path) Resolve() *__Path {
	abs, err := filepath.Abs(p.p)
	if err != nil {
		return p
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return &__Path{p: resolved}
	}
	return &__Path{p: abs}
}

func (p *__Path) Touch() {
	f, err := os.OpenFile(p.p, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		panic(err)
	}
	_ = f.Close()
}

func (p *__Path) Is_absolute() bool { return filepath.IsAbs(p.p) }

func (p *__Path) Is_symlink() bool {
	st, err := os.Lstat(p.p)
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeSymlink != 0
}

func (p *__Path) Samefile(other any) bool {
	var rhs string
	switch v := other.(type) {
	case string:
		rhs = v
	case *__Path:
		rhs = v.p
	default:
		rhs = fmt.Sprintf("%v", v)
	}
	a, e1 := os.Stat(p.p)
	b, e2 := os.Stat(rhs)
	if e1 != nil || e2 != nil {
		return false
	}
	return os.SameFile(a, b)
}

func (p *__Path) As_posix() string { return filepath.ToSlash(p.p) }

func (p *__Path) Expanduser() *__Path {
	if len(p.p) > 0 && p.p[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			return &__Path{p: home + p.p[1:]}
		}
	}
	return &__Path{p: p.p}
}

func (p *__Path) With_stem(stem string) *__Path {
	ext := filepath.Ext(p.p)
	dir := filepath.Dir(p.p)
	if dir == "." {
		return &__Path{p: stem + ext}
	}
	return &__Path{p: dir + "/" + stem + ext}
}

func (p *__Path) Parts() []string {
	clean := filepath.ToSlash(p.p)
	if clean == "" {
		return []string{}
	}
	if clean == "/" {
		return []string{"/"}
	}
	abs := strings.HasPrefix(clean, "/")
	parts := strings.Split(strings.Trim(clean, "/"), "/")
	out := []string{}
	if abs {
		out = append(out, "/")
	}
	for _, s := range parts {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func (p *__Path) Parents() []*__Path {
	out := []*__Path{}
	cur := p.p
	for {
		next := filepath.Dir(cur)
		if next == cur {
			return out
		}
		out = append(out, &__Path{p: next})
		cur = next
		if next == "/" || next == "." {
			return out
		}
	}
}

func (p *__Path) Is_relative_to(other any) bool {
	var rhs string
	switch v := other.(type) {
	case string:
		rhs = v
	case *__Path:
		rhs = v.p
	default:
		rhs = fmt.Sprintf("%v", v)
	}
	rel, err := filepath.Rel(rhs, p.p)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}

func (p *__Path) Symlink_to(target any) {
	var dst string
	switch v := target.(type) {
	case string:
		dst = v
	case *__Path:
		dst = v.p
	default:
		dst = fmt.Sprintf("%v", v)
	}
	if err := os.Symlink(dst, p.p); err != nil {
		panic(err)
	}
}

func (p *__Path) Hardlink_to(target any) {
	var dst string
	switch v := target.(type) {
	case string:
		dst = v
	case *__Path:
		dst = v.p
	default:
		dst = fmt.Sprintf("%v", v)
	}
	if err := os.Link(dst, p.p); err != nil {
		panic(err)
	}
}

func (p *__Path) Rename(target any) *__Path {
	var dst string
	switch v := target.(type) {
	case string:
		dst = v
	case *__Path:
		dst = v.p
	default:
		dst = fmt.Sprintf("%v", v)
	}
	if err := os.Rename(p.p, dst); err != nil {
		panic(err)
	}
	return &__Path{p: dst}
}

func (p *__Path) Replace(target any) *__Path { return p.Rename(target) }

func (p *__Path) Chmod(mode int64) {
	if err := os.Chmod(p.p, os.FileMode(mode)); err != nil {
		panic(err)
	}
}

func (p *__Path) Lchmod(mode int64) { p.Chmod(mode) }

func (p *__Path) Lstat() map[string]any {
	st, err := os.Lstat(p.p)
	if err != nil {
		panic(err)
	}
	return map[string]any{
		"st_size":  int64(st.Size()),
		"st_mode":  int64(st.Mode()),
		"st_mtime": float64(st.ModTime().Unix()),
		"st_isdir": st.IsDir(),
	}
}

func (p *__Path) Relative_to(other any) *__Path {
	var rhs string
	switch v := other.(type) {
	case string:
		rhs = v
	case *__Path:
		rhs = v.p
	default:
		rhs = fmt.Sprintf("%v", v)
	}
	rel, err := filepath.Rel(rhs, p.p)
	if err != nil {
		panic(NewException("ValueError: " + err.Error()))
	}
	return &__Path{p: rel}
}

func (p *__Path) With_suffix(suffix string) *__Path {
	stem := p.p
	for i := len(stem) - 1; i >= 0; i-- {
		if stem[i] == '/' {
			break
		}
		if stem[i] == '.' {
			return &__Path{p: stem[:i] + suffix}
		}
	}
	return &__Path{p: stem + suffix}
}

func (p *__Path) With_name(name string) *__Path {
	dir := p.Parent().p
	if dir == "" || dir == "." {
		return &__Path{p: name}
	}
	return &__Path{p: dir + "/" + name}
}`

const helperPathNew = `func __gopy_path_new(s string) *__Path { return &__Path{p: s} }`

const helperPathCwd = `func __gopy_path_cwd() *__Path {
	d, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return &__Path{p: d}
}`

const helperPathHome = `func __gopy_path_home() *__Path {
	d, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return &__Path{p: d}
}`

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

const helperDirEntryType = `type __DirEntry struct {
	name string
	path string
	d    os.DirEntry
}

func (e *__DirEntry) Name() string     { return e.name }
func (e *__DirEntry) Path() string     { return e.path }
func (e *__DirEntry) Is_file() bool    { return !e.d.IsDir() && e.d.Type()&os.ModeSymlink == 0 }
func (e *__DirEntry) Is_dir() bool     { return e.d.IsDir() }
func (e *__DirEntry) Is_symlink() bool { return e.d.Type()&os.ModeSymlink != 0 }`

const helperOsScandir = `func __gopy_os_scandir(p string) []*__DirEntry {
	entries, err := os.ReadDir(p)
	if err != nil {
		panic(err)
	}
	out := make([]*__DirEntry, 0, len(entries))
	for _, e := range entries {
		full := p
		if len(full) > 0 && full[len(full)-1] != '/' {
			full += "/"
		}
		full += e.Name()
		out = append(out, &__DirEntry{name: e.Name(), path: full, d: e})
	}
	return out
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

const helperPathGetmtime = `func __gopy_path_getmtime(p string) float64 {
	i, err := os.Stat(p)
	if err != nil {
		panic(err)
	}
	t := i.ModTime()
	return float64(t.Unix()) + float64(t.Nanosecond())/1e9
}`

// Go's os.FileInfo doesn't expose atime / ctime portably, so atime
// and ctime collapse to mtime here. Better than panicking; matches
// what CPython returns on filesystems that don't track distinct
// access/change times.
const helperPathGetatime = `func __gopy_path_getatime(p string) float64 {
	return __gopy_path_getmtime(p)
}`

const helperPathGetctime = `func __gopy_path_getctime(p string) float64 {
	return __gopy_path_getmtime(p)
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
}

func (t *__Timedelta) Lt(o *__Timedelta) bool { return t.d < o.d }
func (t *__Timedelta) Le(o *__Timedelta) bool { return t.d <= o.d }
func (t *__Timedelta) Gt(o *__Timedelta) bool { return t.d > o.d }
func (t *__Timedelta) Ge(o *__Timedelta) bool { return t.d >= o.d }
func (t *__Timedelta) Eq(o *__Timedelta) bool { return t.d == o.d }
func (t *__Timedelta) Ne(o *__Timedelta) bool { return t.d != o.d }`

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
// Trailing offsets ("Z", "+0500", "+05:00", "-0700") shift the parsed
// time into UTC; gopy's __Datetime is naive (no tz wrapper) so we
// normalize to UTC rather than carrying a zone.
const helperDatetimeFromIso = `func __gopy_datetime_fromiso(s string) *__Datetime {
	tz := ""
	body := s
	if n := len(body); n >= 1 && (body[n-1] == 'Z' || body[n-1] == 'z') {
		tz = "+0000"
		body = body[:n-1]
	} else if i := strings.LastIndexAny(body, "+-"); i > 8 {
		off := body[i:]
		if len(off) == 6 && off[3] == ':' {
			off = off[:3] + off[4:]
		}
		if len(off) == 5 {
			tz = off
			body = body[:i]
		} else if len(off) == 3 {
			tz = off + "00"
			body = body[:i]
		}
	}
	layouts := []string{
		"2006-01-02T15:04:05.000000",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.000000",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, l := range layouts {
		if tz != "" {
			// Parse with the offset only to validate the form; keep the
			// local clock components on the result so strftime matches
			// CPython's behavior on tz-aware datetimes (which prints the
			// local part, not UTC).
			if _, err := time.Parse(l+"-0700", body+tz); err == nil {
				if t2, err2 := time.Parse(l, body); err2 == nil {
					return &__Datetime{t: t2}
				}
			}
		}
		if t, err := time.Parse(l, body); err == nil {
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

const helperShlexSplit = `func __gopy_shlex_split(s string) []string {
	out := []string{}
	cur := strings.Builder{}
	quote := byte(0)
	inTok := false
	flush := func() { if inTok { out = append(out, cur.String()); cur.Reset(); inTok = false } }
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if c == quote { quote = 0; continue }
			cur.WriteByte(c); inTok = true; continue
		}
		switch c {
		case ' ', '\t', '\n', '\r':
			flush()
		case '\'', '"':
			quote = c; inTok = true
		case '\\':
			if i+1 < len(s) { i++; cur.WriteByte(s[i]); inTok = true }
		default:
			cur.WriteByte(c); inTok = true
		}
	}
	flush()
	return out
}`

const helperShlexQuote = `func __gopy_shlex_quote(s string) string {
	if s == "" { return "''" }
	safe := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '@' || c == '%' || c == '+' || c == '=' || c == ':' || c == ',' || c == '.' || c == '/' || c == '-' || c == '_') {
			safe = false; break
		}
	}
	if safe { return s }
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}`

const helperShlexJoin = `func __gopy_shlex_join(parts []string) string {
	out := make([]string, len(parts))
	for i, p := range parts { out[i] = __gopy_shlex_quote(p) }
	return strings.Join(out, " ")
}`

const helperDifflibClose = `func __gopy_difflib_close(word string, possibilities []string, args ...any) []string {
	n := 3
	cutoff := 0.6
	if len(args) > 0 {
		switch v := args[0].(type) {
		case int: n = v
		case int64: n = int(v)
		}
	}
	if len(args) > 1 {
		switch v := args[1].(type) {
		case float64: cutoff = v
		case float32: cutoff = float64(v)
		}
	}
	type scored struct{ s string; r float64 }
	scores := []scored{}
	for _, p := range possibilities {
		r := __gopy_difflib_ratio(word, p)
		if r >= cutoff { scores = append(scores, scored{p, r}) }
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].r > scores[j].r })
	if len(scores) > n { scores = scores[:n] }
	out := make([]string, len(scores))
	for i, s := range scores { out[i] = s.s }
	return out
}

func __gopy_difflib_ratio(a, b string) float64 {
	la, lb := len(a), len(b)
	if la == 0 && lb == 0 { return 1.0 }
	matches := 0
	used := make([]bool, lb)
	for i := 0; i < la; i++ {
		for j := 0; j < lb; j++ {
			if !used[j] && a[i] == b[j] { used[j] = true; matches++; break }
		}
	}
	_ = strings.Compare
	return 2.0 * float64(matches) / float64(la+lb)
}`

const helperDifflibUnified = `func __gopy_difflib_unified(a, b []string, args ...any) []string {
	fromfile, tofile := "", ""
	if len(args) > 0 { if s, ok := args[0].(string); ok { fromfile = s } }
	if len(args) > 1 { if s, ok := args[1].(string); ok { tofile = s } }
	out := []string{}
	out = append(out, fmt.Sprintf("--- %s", fromfile))
	out = append(out, fmt.Sprintf("+++ %s", tofile))
	out = append(out, fmt.Sprintf("@@ -1,%d +1,%d @@", len(a), len(b)))
	for _, line := range a { out = append(out, "-"+line) }
	for _, line := range b { out = append(out, "+"+line) }
	return out
}`

const helperDifflibNdiff = `func __gopy_difflib_ndiff(a, b []string) []string {
	out := []string{}
	for _, line := range a { out = append(out, "- "+line) }
	for _, line := range b { out = append(out, "+ "+line) }
	return out
}`

const helperFilecmpCmp = `func __gopy_filecmp_cmp(a, b string, shallow ...bool) bool {
	ai, err1 := os.Stat(a)
	bi, err2 := os.Stat(b)
	if err1 != nil || err2 != nil { return false }
	if ai.Size() != bi.Size() { return false }
	if len(shallow) > 0 && shallow[0] && ai.ModTime().Equal(bi.ModTime()) { return true }
	da, err := os.ReadFile(a); if err != nil { return false }
	db, err := os.ReadFile(b); if err != nil { return false }
	return bytes.Equal(da, db)
}`

const helperCodecsEncode = `func __gopy_codecs_encode(obj string, args ...string) string {
	enc := "utf-8"
	if len(args) > 0 { enc = args[0] }
	switch enc {
	case "hex", "hex_codec":
		return hex.EncodeToString([]byte(obj))
	case "base64", "base64_codec":
		return base64.StdEncoding.EncodeToString([]byte(obj))
	default:
		return obj
	}
}`

const helperCodecsDecode = `func __gopy_codecs_decode(obj string, args ...string) string {
	enc := "utf-8"
	if len(args) > 0 { enc = args[0] }
	switch enc {
	case "hex", "hex_codec":
		b, _ := hex.DecodeString(obj); return string(b)
	case "base64", "base64_codec":
		b, _ := base64.StdEncoding.DecodeString(obj); return string(b)
	default:
		return obj
	}
}`

const helperResourceGetrlimit = `func __gopy_resource_getrlimit(_ int64) []int64 { return []int64{-1, -1} }`

const helperResourceSetrlimit = `func __gopy_resource_setrlimit(_ int64, _ []int64) {}`

const helperSyslogOpenlog = `func __gopy_syslog_openlog(_ ...any) {}`

const helperSyslogSyslog = `func __gopy_syslog_syslog(_ ...any) {}`

const helperSyslogCloselog = `func __gopy_syslog_closelog() {}`

const helperQuopriEncode = `func __gopy_quopri_encode(s string) string {
	out := strings.Builder{}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '=' || c < 32 || c > 126 {
			out.WriteString(fmt.Sprintf("=%02X", c))
		} else {
			out.WriteByte(c)
		}
	}
	return out.String()
}`

const helperQuopriDecode = `func __gopy_quopri_decode(s string) string {
	out := strings.Builder{}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '=' && i+2 < len(s) {
			v, err := strconv.ParseInt(s[i+1:i+3], 16, 32)
			if err == nil { out.WriteByte(byte(v)); i += 2; continue }
		}
		out.WriteByte(c)
	}
	return out.String()
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

const helperMathModf = `func __gopy_math_modf(f float64) []float64 {
	i, frac := math.Modf(f)
	return []float64{frac, i}
}`

const helperMathFrexp = `func __gopy_math_frexp(f float64) []any {
	frac, exp := math.Frexp(f)
	return []any{frac, int64(exp)}
}`

const helperMathFsum = `func __gopy_math_fsum(xs []float64) float64 {
	sum := 0.0
	c := 0.0
	for _, x := range xs {
		y := x - c
		t := sum + y
		c = (t - sum) - y
		sum = t
	}
	return sum
}`

const helperMathUlp = `func __gopy_math_ulp(x float64) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) { return x }
	if x == 0 { return math.Nextafter(0, 1) }
	ax := math.Abs(x)
	return math.Nextafter(ax, math.Inf(1)) - ax
}`

const helperUuid1 = `func __gopy_uuid1() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil { panic(err) }
	now := uint64(time.Now().UnixNano()/100) + 0x01b21dd213814000
	b[0] = byte(now); b[1] = byte(now >> 8); b[2] = byte(now >> 16); b[3] = byte(now >> 24)
	b[4] = byte(now >> 32); b[5] = byte(now >> 40)
	b[6] = byte((now >> 48) & 0x0f) | 0x10
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}`

const helperUuid3 = `func __gopy_uuid3(ns, name string) string {
	nsBytes := __gopy_uuid_ns_bytes(ns)
	h := md5.New()
	h.Write(nsBytes)
	h.Write([]byte(name))
	b := h.Sum(nil)[:16]
	b[6] = (b[6] & 0x0f) | 0x30
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func __gopy_uuid_ns_bytes(s string) []byte {
	out := []byte{}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '-' { continue }
		var v byte
		switch {
		case c >= '0' && c <= '9': v = c - '0'
		case c >= 'a' && c <= 'f': v = c - 'a' + 10
		case c >= 'A' && c <= 'F': v = c - 'A' + 10
		}
		out = append(out, v)
	}
	res := make([]byte, len(out)/2)
	for i := 0; i < len(res); i++ {
		res[i] = (out[2*i] << 4) | out[2*i+1]
	}
	return res
}`

const helperUuid5 = `func __gopy_uuid5(ns, name string) string {
	nsBytes := __gopy_uuid_ns_bytes(ns)
	h := sha1.New()
	h.Write(nsBytes)
	h.Write([]byte(name))
	b := h.Sum(nil)[:16]
	b[6] = (b[6] & 0x0f) | 0x50
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}`

const helperHashlibSha224 = `func __gopy_hashlib_sha224(data string) *__Hasher {
	return &__Hasher{data: []byte(data), algo: "sha224"}
}`

const helperHashlibSha384 = `func __gopy_hashlib_sha384(data string) *__Hasher {
	return &__Hasher{data: []byte(data), algo: "sha384"}
}`

const helperTextwrapWrap = `func __gopy_textwrap_wrap(s string, width int64) []string {
	if width <= 0 { width = 70 }
	words := strings.Fields(s)
	if len(words) == 0 { return []string{} }
	lines := []string{}
	cur := words[0]
	for _, w := range words[1:] {
		if int64(len(cur)+1+len(w)) <= width {
			cur += " " + w
		} else {
			lines = append(lines, cur)
			cur = w
		}
	}
	lines = append(lines, cur)
	return lines
}`

const helperInspectArgspec = `func __gopy_inspect_argspec(_ any) string { return "FullArgSpec()" }`

const helperInspectGetmodule = `func __gopy_inspect_getmodule(_ any) string { return "main" }`

const helperInspectGetfile = `func __gopy_inspect_getfile(_ any) string { return "<gopy>" }`

const helperGlobHasMagic = `func __gopy_glob_has_magic(s string) bool {
	return strings.ContainsAny(s, "*?[")
}`

const helperGlobEscape = `func __gopy_glob_escape(s string) string {
	out := strings.Builder{}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '*' || c == '?' || c == '[' {
			out.WriteByte('[')
			out.WriteByte(c)
			out.WriteByte(']')
		} else {
			out.WriteByte(c)
		}
	}
	return out.String()
}`

const helperCalMonthcalendar = `func __gopy_cal_monthcal(year, month int64) [][]int64 {
	pair := __gopy_cal_monthrange(year, month)
	firstWeekday := pair[0]
	daysInMonth := pair[1]
	weeks := [][]int64{}
	week := make([]int64, 7)
	for i := int64(0); i < firstWeekday; i++ { week[i] = 0 }
	day := int64(1)
	wd := firstWeekday
	for day <= daysInMonth {
		week[wd] = day
		wd++
		if wd == 7 {
			weeks = append(weeks, week)
			week = make([]int64, 7)
			wd = 0
		}
		day++
	}
	if wd != 0 {
		weeks = append(weeks, week)
	}
	return weeks
}`

const helperCalLeapdays = `func __gopy_cal_leapdays(y1, y2 int64) int64 {
	count := int64(0)
	for y := y1; y < y2; y++ {
		if __gopy_cal_isleap(y) { count++ }
	}
	return count
}`

const helperOpTruediv = `func __gopy_operator_truediv(a, b any) float64 {
	af, _ := __gopy_to_float(a)
	bf, _ := __gopy_to_float(b)
	return af / bf
}
func __gopy_to_float(x any) (float64, bool) {
	switch v := x.(type) {
	case int: return float64(v), true
	case int64: return float64(v), true
	case float64: return v, true
	case float32: return float64(v), true
	}
	return 0, false
}`

const helperOpFloordiv = `func __gopy_operator_floordiv(a, b int64) int64 {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) { q-- }
	return q
}`

const helperOpMod = `func __gopy_operator_mod(a, b int64) int64 {
	r := a % b
	if r != 0 && ((r < 0) != (b < 0)) { r += b }
	return r
}`

const helperOpNeg = `func __gopy_operator_neg(a int64) int64 { return -a }`
const helperOpPos = `func __gopy_operator_pos(a int64) int64 { return a }`
const helperOpAbs = `func __gopy_operator_abs(a int64) int64 { if a < 0 { return -a }; return a }`
const helperOpLt = `func __gopy_operator_lt(a, b int64) bool { return a < b }`
const helperOpLe = `func __gopy_operator_le(a, b int64) bool { return a <= b }`
const helperOpEq = `func __gopy_operator_eq(a, b any) bool { return a == b }`
const helperOpNe = `func __gopy_operator_ne(a, b any) bool { return a != b }`
const helperOpGt = `func __gopy_operator_gt(a, b int64) bool { return a > b }`
const helperOpGe = `func __gopy_operator_ge(a, b int64) bool { return a >= b }`
const helperOpNot = `func __gopy_operator_not(a any) bool {
	switch v := a.(type) {
	case bool: return !v
	case int: return v == 0
	case int64: return v == 0
	case float64: return v == 0
	case string: return v == ""
	case nil: return true
	}
	return false
}`
const helperOpTruth = `func __gopy_operator_truth(a any) bool {
	switch v := a.(type) {
	case bool: return v
	case int: return v != 0
	case int64: return v != 0
	case float64: return v != 0
	case string: return v != ""
	case nil: return false
	}
	return true
}`
const helperOpIs = `func __gopy_operator_is(a, b any) bool { return a == b }`
const helperOpIsnot = `func __gopy_operator_isnot(a, b any) bool { return a != b }`

const helperA85Encode = `func __gopy_a85encode(s string) string {
	in := []byte(s)
	maxLen := ascii85.MaxEncodedLen(len(in))
	out := make([]byte, maxLen)
	n := ascii85.Encode(out, in)
	return string(out[:n])
}`

const helperA85Decode = `func __gopy_a85decode(s string) string {
	in := []byte(s)
	out := make([]byte, len(in))
	n, _, _ := ascii85.Decode(out, in, true)
	return string(out[:n])
}`

const helperTypingPassthrough = `func __gopy_typing_passthrough(args ...any) any {
	if len(args) > 0 { return args[0] }
	return nil
}`

const helperTypingAssertType = `func __gopy_typing_assert_type(val any, _ any) any { return val }`

const helperTypingAssertNever = `func __gopy_typing_assert_never(_ any) { panic("assert_never reached") }`

const helperStatFilemode = `func __gopy_stat_filemode(mode int64) string {
	t := byte('?')
	switch mode & 0o170000 {
	case 0o100000: t = '-'
	case 0o040000: t = 'd'
	case 0o120000: t = 'l'
	case 0o020000: t = 'c'
	case 0o060000: t = 'b'
	case 0o010000: t = 'p'
	case 0o140000: t = 's'
	}
	bits := []byte{t}
	rwx := func(r, w, x int64, sticky byte, setid bool) []byte {
		out := []byte{'-', '-', '-'}
		if mode&r != 0 { out[0] = 'r' }
		if mode&w != 0 { out[1] = 'w' }
		if mode&x != 0 {
			if setid { out[2] = 's' } else { out[2] = 'x' }
		} else if setid {
			out[2] = 'S'
		}
		_ = sticky
		return out
	}
	bits = append(bits, rwx(0o400, 0o200, 0o100, 0, mode&0o4000 != 0)...)
	bits = append(bits, rwx(0o040, 0o020, 0o010, 0, mode&0o2000 != 0)...)
	other := rwx(0o004, 0o002, 0o001, 0, false)
	if mode&0o1000 != 0 {
		if mode&0o001 != 0 { other[2] = 't' } else { other[2] = 'T' }
	}
	bits = append(bits, other...)
	return string(bits)
}`

const helperOsGetpid = `func __gopy_os_getpid() int64 { return int64(os.Getpid()) }`
const helperOsGetppid = `func __gopy_os_getppid() int64 { return int64(os.Getppid()) }`
const helperOsGetuid = `func __gopy_os_getuid() int64 { return int64(os.Getuid()) }`
const helperOsGeteuid = `func __gopy_os_geteuid() int64 { return int64(os.Geteuid()) }`
const helperOsGetgid = `func __gopy_os_getgid() int64 { return int64(os.Getgid()) }`
const helperOsGetegid = `func __gopy_os_getegid() int64 { return int64(os.Getegid()) }`

const helperOsGetlogin = `func __gopy_os_getlogin() string {
	u, err := user.Current()
	if err != nil { return "" }
	return u.Username
}`

const helperOsSystem = `func __gopy_os_system(cmd string) int64 {
	c := exec.Command("sh", "-c", cmd)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	if err := c.Run(); err != nil {
		if e, ok := err.(*exec.ExitError); ok { return int64(e.ExitCode()) }
		return int64(1)
	}
	return 0
}`

const helperOsFspath = `func __gopy_os_fspath(p any) string {
	if s, ok := p.(string); ok { return s }
	return ""
}`

const helperPathIsmount = `func __gopy_path_ismount(p string) bool {
	if p == "/" { return true }
	info, err := os.Lstat(p)
	if err != nil { return false }
	parent, err2 := os.Lstat(p + "/..")
	if err2 != nil { return false }
	si, ok1 := info.Sys().(interface{ Dev() uint64 })
	sp, ok2 := parent.Sys().(interface{ Dev() uint64 })
	if ok1 && ok2 { return si.Dev() != sp.Dev() }
	return false
}`

const helperPathSplitdrive = `func __gopy_path_splitdrive(p string) []string {
	return []string{"", p}
}`

const helperStatsCorrelation = `func __gopy_stats_correlation(xs, ys []float64) float64 {
	n := float64(len(xs))
	if n != float64(len(ys)) || n < 2 { return 0 }
	var mx, my float64
	for i := range xs { mx += xs[i]; my += ys[i] }
	mx /= n; my /= n
	var num, dx2, dy2 float64
	for i := range xs {
		dx := xs[i] - mx
		dy := ys[i] - my
		num += dx * dy
		dx2 += dx * dx
		dy2 += dy * dy
	}
	if dx2 == 0 || dy2 == 0 { return 0 }
	return num / math.Sqrt(dx2*dy2)
}`

const helperStatsCovariance = `func __gopy_stats_covariance(xs, ys []float64) float64 {
	n := float64(len(xs))
	if n != float64(len(ys)) || n < 2 { return 0 }
	var mx, my float64
	for i := range xs { mx += xs[i]; my += ys[i] }
	mx /= n; my /= n
	var num float64
	for i := range xs { num += (xs[i] - mx) * (ys[i] - my) }
	return num / (n - 1)
}`

const helperStatsGeoMean = `func __gopy_stats_geomean(xs []float64) float64 {
	if len(xs) == 0 { return 0 }
	sum := 0.0
	for _, x := range xs { sum += math.Log(x) }
	return math.Exp(sum / float64(len(xs)))
}`

const helperHashlibPbkdf2 = `func __gopy_hashlib_pbkdf2(algo, pwd, salt string, iters int64, dklen ...int64) string {
	var h func() hash.Hash
	hashLen := 0
	switch algo {
	case "sha1": h = sha1.New; hashLen = 20
	case "sha256": h = sha256.New; hashLen = 32
	case "sha512": h = sha512.New; hashLen = 64
	case "md5": h = md5.New; hashLen = 16
	default: h = sha256.New; hashLen = 32
	}
	dkl := hashLen
	if len(dklen) > 0 { dkl = int(dklen[0]) }
	pwdb := []byte(pwd)
	saltb := []byte(salt)
	blocks := (dkl + hashLen - 1) / hashLen
	out := make([]byte, 0, dkl)
	for i := 1; i <= blocks; i++ {
		ib := make([]byte, 4)
		binary.BigEndian.PutUint32(ib, uint32(i))
		mac := hmac.New(h, pwdb)
		mac.Write(saltb)
		mac.Write(ib)
		u := mac.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)
		for j := int64(1); j < iters; j++ {
			mac = hmac.New(h, pwdb)
			mac.Write(u)
			u = mac.Sum(nil)
			for k := range t { t[k] ^= u[k] }
		}
		out = append(out, t...)
	}
	return hex.EncodeToString(out[:dkl])
}`

const helperSocketIfNameindex = `func __gopy_socket_if_nameindex() [][]any {
	ifs, err := net.Interfaces()
	if err != nil { return [][]any{} }
	out := make([][]any, 0, len(ifs))
	for _, ifc := range ifs {
		out = append(out, []any{int64(ifc.Index), ifc.Name})
	}
	return out
}`

const helperSocketIfIndextoname = `func __gopy_socket_if_indextoname(idx int64) string {
	ifc, err := net.InterfaceByIndex(int(idx))
	if err != nil { return "" }
	return ifc.Name
}`

const helperSocketIfNametoindex = `func __gopy_socket_if_nametoindex(name string) int64 {
	ifc, err := net.InterfaceByName(name)
	if err != nil { return 0 }
	return int64(ifc.Index)
}`

const helperSysconfigGetPaths = `func __gopy_sysconfig_get_paths() map[string]string {
	prefix := "/usr"
	if p := os.Getenv("PREFIX"); p != "" { prefix = p }
	return map[string]string{
		"stdlib":      prefix + "/lib/python3",
		"platstdlib":  prefix + "/lib/python3",
		"purelib":     prefix + "/lib/python3/site-packages",
		"platlib":     prefix + "/lib/python3/site-packages",
		"include":     prefix + "/include/python3",
		"platinclude": prefix + "/include/python3",
		"scripts":     prefix + "/bin",
		"data":        prefix,
	}
}`

const helperSysconfigPlatform = `func __gopy_sysconfig_platform() string {
	return runtime.GOOS + "-" + runtime.GOARCH
}`

const helperSysconfigPyVersion = `func __gopy_sysconfig_pyversion() string { return "3.12" }`

const helperSysconfigConfigVar = `func __gopy_sysconfig_config_var(name string) string {
	switch name {
	case "EXT_SUFFIX": return ".so"
	case "SOABI": return "gopy"
	case "prefix": return "/usr"
	}
	return ""
}`

const helperShutilDiskUsage = `func __gopy_shutil_diskusage(_ string) []int64 {
	return []int64{int64(1<<40), int64(1<<39), int64(1<<39)}
}`

const helperShutilTerminalSize = `func __gopy_shutil_terminal_size(args ...any) []int64 {
	return []int64{80, 24}
}`

const helperMimetypesInit = `func __gopy_mimetypes_init(args ...any) {}`
const helperMimetypesAdd = `func __gopy_mimetypes_add(_ string, _ string) {}`

const helperFnmatchTranslate = `func __gopy_fnmatch_translate(pat string) string {
	out := strings.Builder{}
	out.WriteString("(?s:")
	for i := 0; i < len(pat); i++ {
		c := pat[i]
		switch c {
		case '*': out.WriteString(".*")
		case '?': out.WriteByte('.')
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '\\':
			out.WriteByte('\\'); out.WriteByte(c)
		case '[':
			j := i + 1
			if j < len(pat) && pat[j] == '!' { j++ }
			if j < len(pat) && pat[j] == ']' { j++ }
			for j < len(pat) && pat[j] != ']' { j++ }
			if j >= len(pat) {
				out.WriteString("\\[")
			} else {
				cls := pat[i+1:j]
				if len(cls) > 0 && cls[0] == '!' { cls = "^" + cls[1:] }
				out.WriteByte('['); out.WriteString(cls); out.WriteByte(']')
				i = j
			}
		default:
			out.WriteByte(c)
		}
	}
	out.WriteString(")\\z")
	return out.String()
}`

const helperFunctoolsTotalOrdering = `func __gopy_total_ordering(cls any) any { return cls }`
const helperFunctoolsUpdateWrapper = `func __gopy_update_wrapper(wrapper, _ any, _ ...any) any { return wrapper }`

const helperOpMethodcaller = `func __gopy_operator_methodcaller(name string, args ...any) any { _ = name; _ = args; return nil }`
const helperOpLengthHint = `func __gopy_operator_length_hint(obj any, def ...int64) int64 {
	switch v := obj.(type) {
	case string: return int64(len(v))
	case []any: return int64(len(v))
	case []int64: return int64(len(v))
	case []string: return int64(len(v))
	case map[string]any: return int64(len(v))
	}
	if len(def) > 0 { return def[0] }
	return 0
}`
const helperOpIndex = `func __gopy_operator_index(obj any) int64 {
	switch v := obj.(type) {
	case int: return int64(v)
	case int64: return v
	case bool:
		if v { return 1 }
		return 0
	}
	return 0
}`

const helperCSVListDialects = `func __gopy_csv_list_dialects() []string { return []string{"excel", "excel-tab", "unix"} }`
const helperCSVRegisterDialect = `func __gopy_csv_register_dialect(_ string, _ ...any) {}`
const helperCSVUnregisterDialect = `func __gopy_csv_unregister_dialect(_ string) {}`
const helperCSVGetDialect = `func __gopy_csv_get_dialect(name string) string { return name }`
const helperCSVFieldSizeLimit = `func __gopy_csv_field_size_limit(args ...int64) int64 {
	if len(args) > 0 { return args[0] }
	return int64(131072)
}`

const helperEnumAuto = `var __gopy_enum_auto_counter int64 = 0
func __gopy_enum_auto() int64 { __gopy_enum_auto_counter++; return __gopy_enum_auto_counter }`

const helperTypesSimpleNS = `func __gopy_types_simplens(kwargs ...any) map[string]any {
	out := map[string]any{}
	for i := 0; i+1 < len(kwargs); i += 2 {
		if k, ok := kwargs[i].(string); ok { out[k] = kwargs[i+1] }
	}
	return out
}`

const helperIpaddressAddr = `func __gopy_ipaddress_addr(s string) string {
	ip := net.ParseIP(s)
	if ip == nil { panic(NewException("ValueError: '" + s + "' does not appear to be an IPv4 or IPv6 address")) }
	return ip.String()
}`

const helperIpaddressNet = `func __gopy_ipaddress_net(s string) string {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		ip := net.ParseIP(s)
		if ip == nil { panic(NewException("ValueError: '" + s + "' does not appear to be an IPv4 or IPv6 network")) }
		if ip.To4() != nil { return s + "/32" }
		return s + "/128"
	}
	return n.String()
}`

const helperCodecsLookup = `func __gopy_codecs_lookup(name string) string { return name }`
const helperCodecsNoop = `func __gopy_codecs_noop(_ ...any) {}`

const helperStatsQuantiles = `func __gopy_stats_quantiles(xs []float64, args ...any) []float64 {
	n := 4
	if len(args) > 0 {
		switch v := args[0].(type) {
		case int: n = v
		case int64: n = int(v)
		}
	}
	if len(xs) < 2 || n < 2 { return []float64{} }
	s := make([]float64, len(xs))
	copy(s, xs)
	sort.Float64s(s)
	m := len(s)
	out := []float64{}
	for i := 1; i < n; i++ {
		j := float64(i*(m+1)) / float64(n)
		k := int(j)
		frac := j - float64(k)
		if k <= 0 { out = append(out, s[0]); continue }
		if k >= m { out = append(out, s[m-1]); continue }
		out = append(out, s[k-1]+frac*(s[k]-s[k-1]))
	}
	return out
}`

const helperStatsLinreg = `func __gopy_stats_linreg(xs, ys []float64) []float64 {
	n := float64(len(xs))
	if n != float64(len(ys)) || n < 2 { return []float64{0, 0} }
	var sx, sy, sxx, sxy float64
	for i := range xs {
		sx += xs[i]; sy += ys[i]
		sxx += xs[i]*xs[i]; sxy += xs[i]*ys[i]
	}
	denom := n*sxx - sx*sx
	if denom == 0 { return []float64{0, 0} }
	slope := (n*sxy - sx*sy) / denom
	intercept := (sy - slope*sx) / n
	return []float64{slope, intercept}
}`

const helperTokenIsterminal = `func __gopy_token_isterminal(t int64) bool { return t < 256 }`
const helperTokenIsnonterminal = `func __gopy_token_isnonterminal(t int64) bool { return t >= 256 }`
const helperTokenIseof = `func __gopy_token_iseof(t int64) bool { return t == 0 }`

const helperDisCodeInfo = `func __gopy_dis_codeinfo(_ any) string { return "Name:              <gopy>\nArgcount:          0\nKwonlyargcount:    0\nNumber of locals:  0" }`

const helperEmailParseaddr = `func __gopy_email_parseaddr(s string) []string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, "<"); i >= 0 {
		if j := strings.Index(s[i:], ">"); j >= 0 {
			name := strings.TrimSpace(s[:i])
			addr := s[i+1 : i+j]
			return []string{name, addr}
		}
	}
	return []string{"", s}
}`

const helperEmailFormataddr = `func __gopy_email_formataddr(pair []string) string {
	if len(pair) < 2 { return "" }
	name, addr := pair[0], pair[1]
	if name == "" { return addr }
	return fmt.Sprintf("%s <%s>", name, addr)
}`

const helperEmailMakeMsgid = `func __gopy_email_make_msgid(args ...string) string {
	host, _ := os.Hostname()
	if host == "" { host = "localhost" }
	return fmt.Sprintf("<%d.%d@%s>", time.Now().UnixNano(), os.Getpid(), host)
}`

const helperTimeAsctime = `func __gopy_time_asctime(args ...any) string {
	return time.Now().Format("Mon Jan _2 15:04:05 2006")
}`

const helperTimeCtime = `func __gopy_time_ctime(args ...float64) string {
	t := time.Now()
	if len(args) > 0 {
		secs := args[0]
		t = time.Unix(int64(secs), 0)
	}
	return t.Format("Mon Jan _2 15:04:05 2006")
}`

const helperTimeTzname = `func __gopy_time_tzname() []string {
	std, dst := time.Now().Zone(), time.Now().Zone()
	_ = dst
	return []string{std, ""}
}`

const helperTimeClockres = `func __gopy_time_clockres(args ...int64) float64 { return 1e-9 }`

const helperSysExecutable = `func __gopy_sys_executable() string {
	if exe, err := os.Executable(); err == nil { return exe }
	return os.Args[0]
}`

const helperSysExcInfo = `func __gopy_sys_exc_info() []any { return []any{nil, nil, nil} }`
const helperSysGetRecursion = `func __gopy_sys_getrecursion() int64 { return int64(1000) }`
const helperSysGetRefcount = `func __gopy_sys_getrefcount(_ any) int64 { return int64(1) }`
const helperSysGetEnc = `func __gopy_sys_getenc() string { return "utf-8" }`
const helperSysGetFsenc = `func __gopy_sys_getfsenc() string { return "utf-8" }`
const helperSysGetIntMaxDigits = `func __gopy_sys_getintmax() int64 { return int64(4300) }`
const helperSysGetSwitch = `func __gopy_sys_getswitch() float64 { return 0.005 }`
const helperSysGetAllocated = `func __gopy_sys_alloc() int64 { return int64(0) }`
const helperSysIsFinalizing = `func __gopy_sys_isfin() bool { return false }`

const helperBz2Decompress = `func __gopy_bz2_decompress(data string) string {
	r := bzip2.NewReader(bytes.NewReader([]byte(data)))
	out, err := io.ReadAll(r)
	if err != nil { return "" }
	return string(out)
}`

const helperFaultHandlerIsEnabled = `func __gopy_faulthandler_isenabled() bool { return false }`

const helperZoneinfoZone = `func __gopy_zoneinfo_zone(name string) string {
	if _, err := time.LoadLocation(name); err != nil {
		panic(NewException("ZoneInfoNotFoundError: " + name))
	}
	return name
}`

const helperWebbrowserOpen = `func __gopy_webbrowser_open(url string, args ...any) bool {
	var cmd string
	switch runtime.GOOS {
	case "darwin": cmd = "open"
	case "windows": cmd = "rundll32"
	default: cmd = "xdg-open"
	}
	c := exec.Command(cmd, url)
	if err := c.Start(); err != nil { return false }
	return true
}`

const helperThreadingActiveCount = `func __gopy_threading_active_count() int64 { return int64(1) }`
const helperThreadingGetIdent = `func __gopy_threading_get_ident() int64 { return int64(1) }`

const helperTracebackPrintException = `func __gopy_traceback_print_exception(args ...any) {
	for _, a := range args { fmt.Fprintf(os.Stderr, "%v\n", a) }
}`

const helperTracebackFormatException = `func __gopy_traceback_format_exception(args ...any) []string {
	out := make([]string, len(args))
	for i, a := range args { out[i] = fmt.Sprint(a) }
	return out
}`

// format_exception_only(exc) returns a single-element list with the
// "ClassName: message\n" rendering. gopy exceptions carry a "Prefix:
// msg" string; we split on the colon to keep CPython parity. Falls
// back to a generic Exception label when the receiver lacks the
// prefix.
const helperTracebackFormatExceptionOnly = `func __gopy_traceback_format_exception_only(args ...any) []string {
	if len(args) == 0 {
		return []string{}
	}
	last := args[len(args)-1]
	if e, ok := last.(*Exception); ok {
		return []string{e.Msg + "\n"}
	}
	return []string{fmt.Sprintf("Exception: %v\n", last)}
}`

const helperTracebackFormatStack = `func __gopy_traceback_format_stack() []string { return []string{} }`
const helperTracebackPrintStack = `func __gopy_traceback_print_stack() {}`
const helperTracebackExtractStack = `func __gopy_traceback_extract_stack() []any { return []any{} }`

const helperNetrcNew = `func __gopy_netrc_new(args ...string) map[string]any {
	path := ""
	if len(args) > 0 {
		path = args[0]
	} else if h, err := os.UserHomeDir(); err == nil {
		path = h + "/.netrc"
	}
	hosts := map[string][]string{}
	macros := map[string][]string{}
	f, err := os.Open(path)
	if err != nil {
		return map[string]any{"hosts": hosts, "macros": macros}
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	var curMachine string
	var login, password, account string
	flush := func() {
		if curMachine != "" {
			hosts[curMachine] = []string{login, account, password}
		}
		curMachine, login, password, account = "", "", "", ""
	}
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		tokens := strings.Fields(line)
		for i := 0; i+1 < len(tokens); i += 2 {
			k, v := tokens[i], tokens[i+1]
			switch k {
			case "machine":
				flush()
				curMachine = v
			case "default":
				flush()
				curMachine = "default"
			case "login":
				login = v
			case "password":
				password = v
			case "account":
				account = v
			}
		}
	}
	flush()
	return map[string]any{"hosts": hosts, "macros": macros}
}`

const helperTextwrapShorten = `func __gopy_textwrap_shorten(s string, width int64) string {
	words := strings.Fields(s)
	out := ""
	placeholder := " [...]"
	for i, w := range words {
		next := w
		if i > 0 { next = out + " " + w }
		if int64(len(next)) <= width {
			out = next
		} else {
			for int64(len(out)+len(placeholder)) > width && out != "" {
				idx := strings.LastIndex(out, " ")
				if idx < 0 { out = ""; break }
				out = out[:idx]
			}
			return out + placeholder
		}
	}
	return out
}`
