package dot2svg

// Node is one node declared in the DOT source.
type Node struct {
	// Key is the node's identifier as written (unescaped, but otherwise
	// verbatim — may contain spaces or punctuation if it was quoted).
	Key string
	// Label is the display text: the node's `label` attribute if set,
	// otherwise Key.
	Label string
	// Shape is the Graphviz `shape` attribute, lowercased; empty if unset.
	Shape string
	// Cluster is the ID of the subgraph this node was declared in; empty
	// if the node is at the top level.
	Cluster string
}

// Edge is a directed or undirected relationship between two nodes.
type Edge struct {
	From, To string
	// Label is the edge's `label` attribute, if any.
	Label string
}

// Cluster is a subgraph that contains at least one node.
type Cluster struct {
	// ID is the subgraph's name as written (e.g. "cluster_0"), or a
	// generated "cluster1", "cluster2", ... for anonymous subgraphs.
	ID string
	// Title is the subgraph's `label` attribute if set, otherwise ID.
	Title string
}

// Graph is the graph extracted from a DOT source. Nodes, Edges, and
// Clusters are in first-seen order.
type Graph struct {
	// Directed is true for `digraph`, false for `graph`.
	Directed bool
	// RankDir is the graph's `rankdir` attribute (TB, LR, RL, BT),
	// defaulting to TB as Graphviz does.
	RankDir string

	Nodes    []Node
	Edges    []Edge
	Clusters []Cluster
}
