// Package dot2svg renders a practical subset of the Graphviz DOT language
// to SVG — in pure Go, via go-mermaid — for tools that emit DOT but don't
// want to depend on the native Graphviz binary: `terraform graph`,
// `go tool pprof -dot`, `make2graph`, and similar.
//
//	svg, err := dot2svg.Render(dotSource)
//
// DOT is parsed into a graph model, translated into Mermaid flowchart
// source, and handed to [mermaid.Render]. Supported DOT: digraph/graph,
// node and edge statements (including A -> B -> C chains), default
// `node [...]`/`edge [...]` attributes, `rankdir`, and subgraphs/clusters
// (rendered as Mermaid subgraphs). This is deliberately not full Graphviz
// fidelity — record shapes, HTML-like labels, and Graphviz's own layout
// algorithm are out of scope; go-mermaid's Sugiyama layout lays the graph
// out instead, so the result won't be pixel-identical to `dot`.
package dot2svg

import (
	"fmt"

	mermaid "github.com/zkrebbekx/go-mermaid"
)

// ToMermaid parses a DOT source document and returns the equivalent
// Mermaid flowchart source.
func ToMermaid(src []byte) (string, error) {
	g, err := parseDOT(string(src))
	if err != nil {
		return "", fmt.Errorf("dot2svg: %w", err)
	}
	return toMermaid(g), nil
}

// Render parses a DOT source document and renders it straight to SVG via
// go-mermaid. opts are passed through to [mermaid.Render] (theme, padding,
// spacing, ...).
func Render(src []byte, opts ...mermaid.Option) ([]byte, error) {
	mmd, err := ToMermaid(src)
	if err != nil {
		return nil, err
	}
	return mermaid.Render(mmd, opts...)
}
