// Command dot2svg renders a Graphviz DOT graph to SVG (or Mermaid source,
// or PNG) without depending on the native Graphviz binary.
//
// Usage:
//
//	dot2svg [flags] [input ...]
//
// With no input file (or "-"), source is read from stdin and output is
// written to stdout (or -o). With multiple input files, each FILE.dot is
// rendered to FILE.svg (or .mmd/.png) and -o is not allowed.
//
//	dot2svg graph.dot > graph.svg
//	terraform graph | dot2svg > graph.svg
//	go tool pprof -dot ./bin cpu.prof | dot2svg -o profile.svg
//	dot2svg -format mmd -o graph.mmd graph.dot
//	dot2svg a.dot b.dot   # writes a.svg, b.svg
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/zkrebbekx/dot2svg"
	mermaid "github.com/zkrebbekx/go-mermaid"
	"github.com/zkrebbekx/go-mermaid/raster"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "dot2svg:", err)
		os.Exit(1)
	}
}

func run() error {
	format := flag.String("format", "svg", "output format: svg, mmd, png")
	out := flag.String("o", "", "output file (single-input mode only; default stdout)")
	theme := flag.String("theme", "default", "color theme: default, dark, neutral, forest, base")
	scale := flag.Float64("scale", 1, "PNG scale factor")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		fmt.Println("dot2svg", version)
		return nil
	}

	opts := []mermaid.Option{mermaid.WithTheme(mermaid.Theme(*theme))}

	args := flag.Args()
	if len(args) > 1 {
		if *out != "" {
			return fmt.Errorf("-o cannot be used with multiple input files")
		}
		return renderBatch(args, *format, opts, *scale)
	}

	src, err := readInput(firstArg(args))
	if err != nil {
		return err
	}
	data, err := renderBytes(src, *format, opts, *scale)
	if err != nil {
		return err
	}
	if *out == "" || *out == "-" {
		_, err = os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(*out, data, 0o644)
}

// renderBytes produces output in the requested format for one DOT document.
func renderBytes(src []byte, format string, opts []mermaid.Option, scale float64) ([]byte, error) {
	switch format {
	case "mmd":
		mmd, err := dot2svg.ToMermaid(src)
		return []byte(mmd), err
	case "png":
		mmd, err := dot2svg.ToMermaid(src)
		if err != nil {
			return nil, err
		}
		return raster.PNG(mmd, scale, opts...)
	case "svg":
		return dot2svg.Render(src, opts...)
	default:
		return nil, fmt.Errorf("unknown -format %q (want svg, mmd, or png)", format)
	}
}

// renderBatch renders each input file to a sibling output file, with the
// extension matching format.
func renderBatch(files []string, format string, opts []mermaid.Option, scale float64) error {
	ext, err := extFor(format)
	if err != nil {
		return err
	}
	for _, f := range files {
		src, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		data, err := renderBytes(src, format, opts, scale)
		if err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
		dst := strings.TrimSuffix(f, filepath.Ext(f)) + ext
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "wrote", dst)
	}
	return nil
}

func extFor(format string) (string, error) {
	switch format {
	case "svg":
		return ".svg", nil
	case "mmd":
		return ".mmd", nil
	case "png":
		return ".png", nil
	default:
		return "", fmt.Errorf("unknown -format %q (want svg, mmd, or png)", format)
	}
}

func readInput(path string) ([]byte, error) {
	if path == "" || path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func usage() {
	fmt.Fprintf(os.Stderr, `dot2svg %s - render Graphviz DOT to SVG (pure Go, no graphviz binary)

Usage:
  dot2svg [flags] [input ...]

Flags:
`, version)
	flag.PrintDefaults()
}
