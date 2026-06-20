# dot2svg

[![CI](https://github.com/zkrebbekx/dot2svg/actions/workflows/ci.yml/badge.svg)](https://github.com/zkrebbekx/dot2svg/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/zkrebbekx/dot2svg.svg)](https://pkg.go.dev/github.com/zkrebbekx/dot2svg)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Render a Graphviz [DOT](https://graphviz.org/doc/info/lang.html) graph to
SVG — in **pure Go**, via [go-mermaid](https://github.com/zkrebbekx/go-mermaid).
No Graphviz binary required.

## Why

Plenty of tools emit DOT — `terraform graph`, `go tool pprof -dot`,
`make2graph`, `git log --graph`-adjacent tooling — but rendering it has
always meant shelling out to the native Graphviz `dot` binary. `dot2svg`
skips that: it's a single static binary, or a library call.

This is **not** a Graphviz replacement. It covers a practical subset —
`digraph`/`graph`, node and edge statements (including chains like
`A -> B -> C`), default `node [...]`/`edge [...]` attributes, `rankdir`,
and subgraphs/clusters — which is what the tools above actually emit.
Record shapes, HTML-like labels, and Graphviz's own layout algorithm are
out of scope; go-mermaid's Sugiyama layout lays the graph out instead, so
the result won't be pixel-identical to `dot`.

## Install

Library:

```bash
go get github.com/zkrebbekx/dot2svg
```

CLI:

```bash
go install github.com/zkrebbekx/dot2svg/cmd/dot2svg@latest
```

Homebrew:

```bash
brew install zkrebbekx/tap/dot2svg
```

Docker:

```bash
terraform graph | docker run -i --rm ghcr.io/zkrebbekx/dot2svg > graph.svg
```

Prebuilt binaries for Linux/macOS/Windows (amd64/arm64) are attached to each
[GitHub release](https://github.com/zkrebbekx/dot2svg/releases).

## Usage

```bash
dot2svg graph.dot > graph.svg
terraform graph | dot2svg > graph.svg
go tool pprof -dot ./mybinary cpu.prof | dot2svg -o profile.svg
dot2svg -format mmd -o graph.mmd graph.dot   # raw Mermaid source
dot2svg -format png -scale 2 -o graph.png graph.dot
dot2svg a.dot b.dot                          # batch: writes a.svg, b.svg
```

Library:

```go
package main

import (
	"os"

	"github.com/zkrebbekx/dot2svg"
)

func main() {
	src, _ := os.ReadFile("graph.dot")
	svg, err := dot2svg.Render(src)
	if err != nil {
		panic(err)
	}
	os.WriteFile("graph.svg", svg, 0o644)
}
```

`Render` accepts the same functional options as `go-mermaid.Render` (theme,
spacing, ...). `dot2svg.ToMermaid(src)` returns just the Mermaid flowchart
text, without rendering it.

## How it maps

| DOT | diagram |
| --- | --- |
| `digraph` / `graph` | directed (`-->`) or undirected (`---`) edges |
| each node statement | a node — shape follows the closest Mermaid equivalent (see below) |
| `rankdir` | the Mermaid flowchart direction (`TB`/`LR`/`RL`/`BT` — same vocabulary as DOT) |
| `label="..."` (edge) | a pipe-delimited Mermaid edge label |
| `subgraph` / `cluster` | a Mermaid subgraph, titled by the subgraph's own `label` if set |
| `node [...]` / `edge [...]` | default attributes for statements that follow in the same block |

Shape mapping: `box`/`rect`/unset → rectangle, `ellipse`/`oval`/`note` →
rounded, `circle`/`doublecircle` → double-circle, `diamond` → diamond.
Anything else falls back to a rectangle.

Node IDs in the diagram are short and sequential (`n1`, `n2`, ...) rather
than derived from the DOT identifier — DOT IDs from tools like Terraform
are often long, quoted resource addresses; the original text is kept as
the node's *label*, not mangled into an identifier. Labels are sanitized
of characters (`[`, `]`, `(`, `)`, `{`, `}`, `|`, `"`) that would otherwise
be misread as Mermaid shape or edge-label delimiters, and Graphviz's
`\n`/`\l`/`\r` line-break escapes collapse to spaces.

## Develop

```sh
make test
make lint
make build
```

## License

MIT
