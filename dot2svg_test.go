package dot2svg_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zkrebbekx/dot2svg"

	. "github.com/smartystreets/goconvey/convey"
)

func TestToMermaidBasics(t *testing.T) {
	Convey("Given a directed graph with a plain edge", t, func() {
		src := []byte(`digraph { A -> B }`)

		Convey("When converted to Mermaid", func() {
			mmd, err := dot2svg.ToMermaid(src)

			Convey("Then it uses a directed arrow and rectangle nodes", func() {
				So(err, ShouldBeNil)
				So(mmd, ShouldContainSubstring, "n1[A]")
				So(mmd, ShouldContainSubstring, "n2[B]")
				So(mmd, ShouldContainSubstring, "n1 --> n2")
			})
		})
	})

	Convey("Given an undirected graph", t, func() {
		src := []byte(`graph { A -- B }`)

		Convey("When converted to Mermaid", func() {
			mmd, err := dot2svg.ToMermaid(src)

			Convey("Then it uses an undirected (no-arrowhead) connector", func() {
				So(err, ShouldBeNil)
				So(mmd, ShouldContainSubstring, "n1 --- n2")
			})
		})
	})

	Convey("Given a chain of edges", t, func() {
		src := []byte(`digraph { A -> B -> C }`)

		Convey("When converted to Mermaid", func() {
			mmd, err := dot2svg.ToMermaid(src)

			Convey("Then each consecutive pair becomes its own edge", func() {
				So(err, ShouldBeNil)
				So(mmd, ShouldContainSubstring, "n1 --> n2")
				So(mmd, ShouldContainSubstring, "n2 --> n3")
			})
		})
	})

	Convey("Given an edge with a label attribute", t, func() {
		src := []byte(`digraph { A -> B [label="healthy"] }`)

		Convey("When converted to Mermaid", func() {
			mmd, err := dot2svg.ToMermaid(src)

			Convey("Then the edge carries a pipe label", func() {
				So(err, ShouldBeNil)
				So(mmd, ShouldContainSubstring, "n1 -->|healthy| n2")
			})
		})
	})

	Convey("Given nodes with Graphviz shapes", t, func() {
		src := []byte(`digraph {
			A [shape=box]
			B [shape=ellipse]
			C [shape=circle]
			D [shape=diamond]
		}`)

		Convey("When converted to Mermaid", func() {
			mmd, err := dot2svg.ToMermaid(src)

			Convey("Then each maps to its closest Mermaid shape", func() {
				So(err, ShouldBeNil)
				So(mmd, ShouldContainSubstring, "n1[A]")
				So(mmd, ShouldContainSubstring, "n2(B)")
				So(mmd, ShouldContainSubstring, "n3((C))")
				So(mmd, ShouldContainSubstring, "n4{D}")
			})
		})
	})

	Convey("Given default node and edge attribute statements", t, func() {
		src := []byte(`digraph {
			node [shape=ellipse]
			edge [label="default"]
			A -> B
			C [shape=box]
		}`)

		Convey("When converted to Mermaid", func() {
			mmd, err := dot2svg.ToMermaid(src)

			Convey("Then later statements inherit the defaults, own attrs override", func() {
				So(err, ShouldBeNil)
				So(mmd, ShouldContainSubstring, "n1(A)")
				So(mmd, ShouldContainSubstring, "n2(B)")
				So(mmd, ShouldContainSubstring, "n3[C]")
				So(mmd, ShouldContainSubstring, "n1 -->|default| n2")
			})
		})
	})

	Convey("Given a rankdir attribute", t, func() {
		src := []byte(`digraph { rankdir="LR"; A -> B }`)

		Convey("When converted to Mermaid", func() {
			mmd, err := dot2svg.ToMermaid(src)

			Convey("Then the Mermaid direction matches", func() {
				So(err, ShouldBeNil)
				So(mmd, ShouldStartWith, "graph LR\n")
			})
		})
	})

	Convey("Given no rankdir attribute", t, func() {
		src := []byte(`digraph { A -> B }`)

		Convey("When converted to Mermaid", func() {
			mmd, err := dot2svg.ToMermaid(src)

			Convey("Then it defaults to top-to-bottom, same as Graphviz", func() {
				So(err, ShouldBeNil)
				So(mmd, ShouldStartWith, "graph TB\n")
			})
		})
	})

	Convey("Given a subgraph containing nodes", t, func() {
		src := []byte(`digraph {
			subgraph cluster_0 {
				label = "Group"
				A
				B
			}
			C
			A -> C
		}`)

		Convey("When converted to Mermaid", func() {
			mmd, err := dot2svg.ToMermaid(src)

			Convey("Then the subgraph's nodes are grouped, others stay top-level", func() {
				So(err, ShouldBeNil)
				So(mmd, ShouldContainSubstring, "subgraph clus_cluster_0 [Group]")
				So(mmd, ShouldContainSubstring, "n3[C]") // C declared outside, top-level
			})
		})
	})

	Convey("Given comments in several styles", t, func() {
		src := []byte(`digraph {
			// line comment
			# also a line comment
			/* block
			   comment */
			A -> B
		}`)

		Convey("When converted to Mermaid", func() {
			mmd, err := dot2svg.ToMermaid(src)

			Convey("Then they are skipped without affecting the graph", func() {
				So(err, ShouldBeNil)
				So(mmd, ShouldContainSubstring, "n1 --> n2")
			})
		})
	})

	Convey("Given a label using Graphviz line-break escapes", t, func() {
		src := []byte(`digraph { A [label="line1\nline2\lline3"] }`)

		Convey("When converted to Mermaid", func() {
			mmd, err := dot2svg.ToMermaid(src)

			Convey("Then the escapes collapse to spaces", func() {
				So(err, ShouldBeNil)
				So(mmd, ShouldContainSubstring, "n1[line1 line2 line3]")
			})
		})
	})

	Convey("Given a label containing Mermaid-structural characters", t, func() {
		src := []byte(`digraph { A [label="[root] thing (expand)"] }`)

		Convey("When converted to Mermaid", func() {
			mmd, err := dot2svg.ToMermaid(src)

			Convey("Then brackets and parens are stripped so the shape delimiter isn't confused", func() {
				So(err, ShouldBeNil)
				So(mmd, ShouldContainSubstring, "n1[root thing expand]")
			})
		})
	})

	Convey("Given quoted node IDs containing spaces", t, func() {
		src := []byte(`digraph { "my node" -> "other node" }`)

		Convey("When converted to Mermaid", func() {
			mmd, err := dot2svg.ToMermaid(src)

			Convey("Then the raw text becomes the label, with a generated safe ID", func() {
				So(err, ShouldBeNil)
				So(mmd, ShouldContainSubstring, "n1[my node]")
				So(mmd, ShouldContainSubstring, "n2[other node]")
			})
		})
	})

	Convey("Given malformed DOT", t, func() {
		src := []byte(`not a graph at all`)

		Convey("When converted to Mermaid", func() {
			_, err := dot2svg.ToMermaid(src)

			Convey("Then it returns an error", func() {
				So(err, ShouldNotBeNil)
			})
		})
	})
}

func TestRenderProducesSVG(t *testing.T) {
	Convey("Given a simple DOT graph", t, func() {
		src := []byte(`digraph { A -> B -> C }`)

		Convey("When rendered", func() {
			svg, err := dot2svg.Render(src)

			Convey("Then it produces valid SVG output", func() {
				So(err, ShouldBeNil)
				So(string(svg), ShouldContainSubstring, "<svg")
			})
		})
	})
}

// Real-world-shaped fixtures: a hand-written Makefile-dependency-style
// graph (the shape make2graph/remake -x emit), a synthetic-but-structurally
// faithful `terraform graph -type=plan` output (quoted "[root] ... (expand)"
// IDs, a "root" subgraph, box/diamond/note shapes — modeled on real
// terraform graph output, with placeholder resource names), and a
// genuinely generated `go tool pprof -dot` profile of a real CPU profile.
// dot2svg must not choke on what these tools actually produce.
var realFixtures = []struct {
	file     string
	contains []string
}{
	{"makefile_deps.dot", []string{"all", "build", "test", "vendor", "clean"}},
	{"terraform_plan.dot", []string{"kube_config", "cloud_token"}},
	{"pprof_fib.dot", []string{"fib", "runtime"}},
}

func TestRealFixtures(t *testing.T) {
	for _, fx := range realFixtures {
		fx := fx
		Convey("Given the real-world DOT file "+fx.file, t, func() {
			src, err := os.ReadFile(filepath.Join("testdata", fx.file))
			So(err, ShouldBeNil)

			Convey("When rendered to SVG", func() {
				svg, err := dot2svg.Render(src)

				Convey("Then it renders without error", func() {
					So(err, ShouldBeNil)
					So(string(svg), ShouldContainSubstring, "<svg")
				})
			})

			Convey("When converted to Mermaid", func() {
				mmd, err := dot2svg.ToMermaid(src)

				Convey("Then expected content made it through", func() {
					So(err, ShouldBeNil)
					for _, want := range fx.contains {
						So(mmd, ShouldContainSubstring, want)
					}
				})
			})
		})
	}
}
