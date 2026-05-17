package parser

import (
	"os"
	"path/filepath"
)

// LocatePython picks the Python interpreter to run. Resolution order:
//  1. If GOPY_PYTHON env var is set and non-empty, use it.
//  2. If a virtualenv exists alongside the source (./.venv/, ./venv/, or
//     the same in the source file's parent dir), use its bin/python3.
//  3. Otherwise fall back to "python3" on PATH.
//
// The search root is hint — typically the source file or its containing
// directory. Empty hint disables step 2.
func LocatePython(hint string) string {
	if v := os.Getenv("GOPY_PYTHON"); v != "" {
		return v
	}
	if hint != "" {
		if p := findVenvPython(hint); p != "" {
			return p
		}
	}
	return "python3"
}

// findVenvPython walks up from start looking for a .venv/ or venv/ directory
// that contains bin/python3. Returns the absolute path or "".
func findVenvPython(start string) string {
	dir := start
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	for i := 0; i < 6; i++ { // bounded climb so we don't walk to /
		for _, name := range []string{".venv", "venv", ".env"} {
			cand := filepath.Join(dir, name, "bin", "python3")
			if _, err := os.Stat(cand); err == nil {
				abs, _ := filepath.Abs(cand)
				return abs
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
