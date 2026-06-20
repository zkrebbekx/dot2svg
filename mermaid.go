package dot2svg

import (
	"fmt"
	"strings"
)

// toMermaid renders g as Mermaid flowchart source. Nodes inside a DOT
// subgraph/cluster are grouped into a Mermaid subgraph; nodes outside any
// cluster are emitted at the top level alongside those blocks.
func toMermaid(g *Graph) string {
	var b strings.Builder
	fmt.Fprintf(&b, "graph %s\n", g.RankDir)

	ids := nodeIDs(g.Nodes)
	connector := "---"
	if g.Directed {
		connector = "-->"
	}

	for _, n := range g.Nodes {
		if n.Cluster == "" {
			writeNode(&b, "  ", ids[n.Key], n)
		}
	}
	for _, c := range g.Clusters {
		fmt.Fprintf(&b, "  subgraph clus_%s [%s]\n", sanitizeID(c.ID), mermaidSafe(c.Title))
		for _, n := range g.Nodes {
			if n.Cluster == c.ID {
				writeNode(&b, "    ", ids[n.Key], n)
			}
		}
		b.WriteString("  end\n")
	}

	for _, e := range g.Edges {
		from, ok := ids[e.From]
		if !ok {
			from = sanitizeID(e.From)
		}
		to, ok := ids[e.To]
		if !ok {
			to = sanitizeID(e.To)
		}
		if e.Label != "" {
			fmt.Fprintf(&b, "  %s %s|%s| %s\n", from, connector, mermaidSafe(e.Label), to)
		} else {
			fmt.Fprintf(&b, "  %s %s %s\n", from, connector, to)
		}
	}

	return b.String()
}

func writeNode(b *strings.Builder, indent, id string, n Node) {
	label := mermaidSafe(n.Label)
	switch n.Shape {
	case "ellipse", "oval", "note":
		fmt.Fprintf(b, "%s%s(%s)\n", indent, id, label)
	case "circle", "doublecircle":
		fmt.Fprintf(b, "%s%s((%s))\n", indent, id, label)
	case "diamond":
		fmt.Fprintf(b, "%s%s{%s}\n", indent, id, label)
	default:
		fmt.Fprintf(b, "%s%s[%s]\n", indent, id, label)
	}
}

// nodeIDs assigns each node a short, sequential Mermaid-safe identifier.
// DOT keys are often long and full of punctuation (e.g. a quoted
// Terraform resource address) — sequential IDs sidestep sanitizing all of
// that into the diagram source, since the original text survives as the
// node's label regardless.
func nodeIDs(nodes []Node) map[string]string {
	ids := make(map[string]string, len(nodes))
	for i, n := range nodes {
		ids[n.Key] = fmt.Sprintf("n%d", i+1)
	}
	return ids
}

// sanitizeID maps an arbitrary DOT identifier (used for cluster IDs, and
// as a fallback for an edge endpoint not seen as a node statement) to a
// Mermaid-safe identifier: letters, digits, and underscores only.
func sanitizeID(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	id := b.String()
	if id == "" || (id[0] >= '0' && id[0] <= '9') {
		id = "n_" + id
	}
	return id
}

// mermaidSafe makes label text safe to embed in Mermaid shape/edge-label
// delimiters. go-mermaid's lexer scans for a shape's closing delimiter (or
// a pipe label's closing '|') as a plain substring search, with no escape
// syntax — a stray ']', ')', '}', '|', or '"' in the text would terminate
// the shape or label early. DOT labels (especially from tools like
// `terraform graph` or `go tool pprof -dot`) routinely contain brackets and
// parens, so those characters are stripped rather than passed through.
func mermaidSafe(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '[', ']', '(', ')', '{', '}', '|', '"', '`':
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
