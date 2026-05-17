// Command gopy-bench compares wall-time and peak RSS between running a
// Python script under CPython and running the same logic after transpiling
// it to Go and compiling with `go build`.
//
// Usage:
//
//	gopy-bench [-n 5] [-warmup 1] path/to/script.py
//
// It transpiles the script with gopy, builds the binary, then runs both
// versions N times and prints a comparison table.
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"syscall"
	"time"

	"github.com/rroblf01/gopy/ir"
	"github.com/rroblf01/gopy/parser"
	"github.com/rroblf01/gopy/transpile"
)

func main() {
	n := flag.Int("n", 5, "iterations per implementation")
	warmup := flag.Int("warmup", 1, "warmup runs (not counted)")
	dumper := flag.String("dumper", "", "path to scripts/py_ast_dump.py (default: auto-locate)")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: gopy-bench [-n N] [-warmup K] script.py")
		os.Exit(2)
	}
	pyPath := flag.Arg(0)

	dumperPath := *dumper
	if dumperPath == "" {
		dumperPath = locateDumper()
		if dumperPath == "" {
			die("cannot locate py_ast_dump.py (pass -dumper)")
		}
	}

	binPath, cleanup := buildBinary(dumperPath, pyPath)
	defer cleanup()

	fmt.Printf("script:   %s\n", pyPath)
	fmt.Printf("iters:    %d (+%d warmup)\n", *n, *warmup)
	fmt.Println()

	pyRes := benchmark("python3", []string{pyPath}, *n, *warmup)
	goRes := benchmark(binPath, nil, *n, *warmup)

	printReport("CPython", pyRes)
	printReport("gopy Go", goRes)
	printDelta(pyRes, goRes)
}

// buildBinary transpiles pyPath to Go, builds it, and returns the binary path
// plus a cleanup func that removes the temp dir.
func buildBinary(dumperPath, pyPath string) (string, func()) {
	root, err := parser.ParseFile(dumperPath, pyPath)
	check(err)
	mod, err := ir.Lower(filepath.Base(pyPath), root)
	check(err)
	src, err := transpile.Module(mod, transpile.Options{PackageName: "main"})
	check(err)

	dir, err := os.MkdirTemp("", "gopy-bench-*")
	check(err)
	goFile := filepath.Join(dir, "main.go")
	check(os.WriteFile(goFile, src, 0o644))
	bin := filepath.Join(dir, "prog")
	build := exec.Command("go", "build", "-o", bin, goFile)
	out, err := build.CombinedOutput()
	if err != nil {
		os.RemoveAll(dir)
		die(fmt.Sprintf("go build: %v: %s", err, out))
	}
	return bin, func() { os.RemoveAll(dir) }
}

type sample struct {
	wall   time.Duration
	maxRSS int64 // kilobytes (Linux Rusage.Maxrss unit)
}

type result struct {
	samples []sample
}

func benchmark(cmdName string, args []string, n, warmup int) result {
	for i := 0; i < warmup; i++ {
		runOnce(cmdName, args) // discard
	}
	r := result{}
	for i := 0; i < n; i++ {
		r.samples = append(r.samples, runOnce(cmdName, args))
	}
	return r
}

func runOnce(cmdName string, args []string) sample {
	cmd := exec.Command(cmdName, args...)
	// Discard stdout/stderr — printing inside the timed region would skew
	// results and we already verified correctness in the test suite.
	cmd.Stdout = nil
	cmd.Stderr = nil
	start := time.Now()
	if err := cmd.Run(); err != nil {
		die(fmt.Sprintf("run %s: %v", cmdName, err))
	}
	wall := time.Since(start)
	var rss int64
	if ru, ok := cmd.ProcessState.SysUsage().(*syscall.Rusage); ok {
		rss = ru.Maxrss
		if runtime.GOOS == "darwin" {
			// macOS reports bytes; Linux reports KB. Normalize to KB.
			rss /= 1024
		}
	}
	return sample{wall: wall, maxRSS: rss}
}

func printReport(name string, r result) {
	walls := make([]float64, len(r.samples))
	rsses := make([]float64, len(r.samples))
	for i, s := range r.samples {
		walls[i] = float64(s.wall.Microseconds()) / 1000.0 // ms
		rsses[i] = float64(s.maxRSS) / 1024.0              // MB
	}
	wMin, wMed, wMax, wMean := stats(walls)
	_, rMed, _, rMean := stats(rsses)
	fmt.Printf("=== %s ===\n", name)
	fmt.Printf("  wall ms : min=%.2f  med=%.2f  max=%.2f  mean=%.2f\n", wMin, wMed, wMax, wMean)
	fmt.Printf("  RSS  MB : med=%.2f  mean=%.2f\n", rMed, rMean)
	fmt.Println()
}

func printDelta(py, gp result) {
	pyMean := meanWall(py)
	goMean := meanWall(gp)
	pyRss := meanRSS(py)
	goRss := meanRSS(gp)
	fmt.Println("=== Delta (CPython / gopy) ===")
	if goMean > 0 {
		fmt.Printf("  speedup  : %.2fx\n", pyMean/goMean)
	}
	if goRss > 0 {
		fmt.Printf("  RSS save : %.2fx (Python %.1f MB  vs  Go %.1f MB)\n",
			pyRss/goRss, pyRss, goRss)
	}
}

func meanWall(r result) float64 {
	var s float64
	for _, x := range r.samples {
		s += float64(x.wall.Microseconds()) / 1000.0
	}
	return s / float64(len(r.samples))
}

func meanRSS(r result) float64 {
	var s float64
	for _, x := range r.samples {
		s += float64(x.maxRSS) / 1024.0
	}
	return s / float64(len(r.samples))
}

func stats(xs []float64) (min, med, max, mean float64) {
	if len(xs) == 0 {
		return
	}
	sorted := make([]float64, len(xs))
	copy(sorted, xs)
	sort.Float64s(sorted)
	min = sorted[0]
	max = sorted[len(sorted)-1]
	med = sorted[len(sorted)/2]
	if len(sorted)%2 == 0 {
		med = (sorted[len(sorted)/2-1] + sorted[len(sorted)/2]) / 2
	}
	var s float64
	for _, x := range xs {
		s += x
	}
	mean = s / float64(len(xs))
	if math.IsNaN(mean) {
		mean = 0
	}
	return
}

func locateDumper() string {
	candidates := []string{}
	if _, file, _, ok := runtime.Caller(0); ok {
		candidates = append(candidates, filepath.Join(filepath.Dir(file), "..", "..", "scripts", "py_ast_dump.py"))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "scripts", "py_ast_dump.py"),
			filepath.Join(cwd, "..", "scripts", "py_ast_dump.py"),
		)
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}
	return ""
}

func check(err error) {
	if err != nil {
		die(err.Error())
	}
}

func die(msg string) {
	fmt.Fprintln(os.Stderr, "gopy-bench:", msg)
	os.Exit(1)
}
