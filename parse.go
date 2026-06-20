package dot2svg

import (
	"fmt"
	"strings"
)

// parseDOT parses a practical subset of the DOT language: digraph/graph,
// node and edge statements (including A -> B -> C chains), attribute
// lists (including default `node [...]`/`edge [...]` statements), and
// subgraphs/clusters. Ports, HTML-like labels, and `strict` multi-edge
// dedup are accepted syntactically but not given special meaning.
func parseDOT(src string) (*Graph, error) {
	toks, err := newLexer(src).tokens()
	if err != nil {
		return nil, err
	}
	p := &parser{
		toks:         toks,
		g:            &Graph{RankDir: "TB"},
		nodeIndex:    map[string]int{},
		clusterIndex: map[string]int{},
	}
	if err := p.parseGraph(); err != nil {
		return nil, err
	}
	p.dropEmptyClusters()
	return p.g, nil
}

type attrs map[string]string

func mergeAttrs(base, override attrs) attrs {
	merged := make(attrs, len(base)+len(override))
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range override {
		merged[k] = v
	}
	return merged
}

// scope tracks the default node/edge attributes and enclosing cluster in
// effect while parsing one { ... } block. Subgraphs inherit a copy of
// their parent's defaults.
type scope struct {
	cluster      string
	nodeDefaults attrs
	edgeDefaults attrs
}

func (s scope) clone() scope {
	return scope{
		cluster:      s.cluster,
		nodeDefaults: mergeAttrs(s.nodeDefaults, nil),
		edgeDefaults: mergeAttrs(s.edgeDefaults, nil),
	}
}

type parser struct {
	toks           []token
	pos            int
	g              *Graph
	nodeIndex      map[string]int
	clusterIndex   map[string]int
	clusterCounter int
}

func (p *parser) cur() token { return p.toks[p.pos] }

func (p *parser) advance() token {
	t := p.toks[p.pos]
	if p.pos < len(p.toks)-1 {
		p.pos++
	}
	return t
}

func (p *parser) isKeyword(word string) bool {
	t := p.cur()
	return t.kind == tokIdent && strings.EqualFold(t.val, word)
}

func (p *parser) expect(kind tokenKind, what string) (token, error) {
	if p.cur().kind != kind {
		return token{}, fmt.Errorf("dot2svg: line %d: expected %s, got %q", p.cur().line, what, p.cur().val)
	}
	return p.advance(), nil
}

func (p *parser) parseGraph() error {
	if p.isKeyword("strict") {
		p.advance()
	}
	switch {
	case p.isKeyword("digraph"):
		p.g.Directed = true
		p.advance()
	case p.isKeyword("graph"):
		p.g.Directed = false
		p.advance()
	default:
		return fmt.Errorf("dot2svg: line %d: expected 'graph' or 'digraph', got %q", p.cur().line, p.cur().val)
	}
	if p.cur().kind == tokIdent || p.cur().kind == tokString {
		p.advance() // graph name, unused
	}
	if _, err := p.expect(tokLBrace, "'{'"); err != nil {
		return err
	}
	if err := p.parseStmtList(scope{}); err != nil {
		return err
	}
	_, err := p.expect(tokRBrace, "'}'")
	return err
}

func (p *parser) parseStmtList(s scope) error {
	for p.cur().kind != tokRBrace && p.cur().kind != tokEOF {
		if err := p.parseStmt(&s); err != nil {
			return err
		}
		if p.cur().kind == tokSemi {
			p.advance()
		}
	}
	return nil
}

func (p *parser) parseStmt(s *scope) error {
	if p.isKeyword("subgraph") || p.cur().kind == tokLBrace {
		return p.parseSubgraph(s)
	}
	if p.isKeyword("node") && p.toks[p.next1()].kind == tokLBracket {
		p.advance()
		a, err := p.parseAttrList()
		if err != nil {
			return err
		}
		s.nodeDefaults = mergeAttrs(s.nodeDefaults, a)
		return nil
	}
	if p.isKeyword("edge") && p.toks[p.next1()].kind == tokLBracket {
		p.advance()
		a, err := p.parseAttrList()
		if err != nil {
			return err
		}
		s.edgeDefaults = mergeAttrs(s.edgeDefaults, a)
		return nil
	}
	if p.isKeyword("graph") && p.toks[p.next1()].kind == tokLBracket {
		p.advance()
		a, err := p.parseAttrList()
		if err != nil {
			return err
		}
		p.applyGraphAttrs(s, a)
		return nil
	}

	if p.cur().kind != tokIdent && p.cur().kind != tokString {
		return fmt.Errorf("dot2svg: line %d: unexpected token %q", p.cur().line, p.cur().val)
	}
	id1 := p.advance().val
	p.skipPort()

	if p.cur().kind == tokEquals {
		p.advance()
		val, err := p.readID()
		if err != nil {
			return err
		}
		p.applyGraphAttrs(s, attrs{strings.ToLower(id1): val})
		return nil
	}

	ids := []string{id1}
	for p.cur().kind == tokArrow || p.cur().kind == tokEdgeOp {
		p.advance()
		next, err := p.readID()
		if err != nil {
			return err
		}
		p.skipPort()
		ids = append(ids, next)
	}

	var own attrs
	if p.cur().kind == tokLBracket {
		a, err := p.parseAttrList()
		if err != nil {
			return err
		}
		own = a
	}

	if len(ids) == 1 {
		p.registerNode(s, ids[0], own)
		return nil
	}
	for _, id := range ids {
		p.registerNode(s, id, nil)
	}
	for i := 0; i < len(ids)-1; i++ {
		p.addEdge(s, ids[i], ids[i+1], own)
	}
	return nil
}

// next1 returns the index of the token after the current one, clamped to
// the last token (EOF), for one-token lookahead.
func (p *parser) next1() int {
	if p.pos+1 < len(p.toks) {
		return p.pos + 1
	}
	return len(p.toks) - 1
}

func (p *parser) readID() (string, error) {
	if p.cur().kind != tokIdent && p.cur().kind != tokString {
		return "", fmt.Errorf("dot2svg: line %d: expected identifier, got %q", p.cur().line, p.cur().val)
	}
	return p.advance().val, nil
}

// skipPort tolerates a Graphviz "node:port" or "node:port:compass"
// suffix; ports have no meaning in our flat node model.
func (p *parser) skipPort() {
	for p.cur().kind == tokColon {
		p.advance()
		if p.cur().kind == tokIdent || p.cur().kind == tokString {
			p.advance()
		}
	}
}

func (p *parser) parseAttrList() (attrs, error) {
	result := attrs{}
	for p.cur().kind == tokLBracket {
		p.advance()
		for p.cur().kind != tokRBracket && p.cur().kind != tokEOF {
			key, err := p.readID()
			if err != nil {
				return nil, err
			}
			val := ""
			if p.cur().kind == tokEquals {
				p.advance()
				val, err = p.readID()
				if err != nil {
					return nil, err
				}
			}
			result[strings.ToLower(key)] = val
			if p.cur().kind == tokComma || p.cur().kind == tokSemi {
				p.advance()
			}
		}
		if _, err := p.expect(tokRBracket, "']'"); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (p *parser) parseSubgraph(s *scope) error {
	name := ""
	if p.isKeyword("subgraph") {
		p.advance()
		if p.cur().kind == tokIdent || p.cur().kind == tokString {
			name = p.advance().val
		}
	}
	id := name
	if id == "" {
		p.clusterCounter++
		id = fmt.Sprintf("cluster%d", p.clusterCounter)
	}
	if _, exists := p.clusterIndex[id]; !exists {
		p.clusterIndex[id] = len(p.g.Clusters)
		p.g.Clusters = append(p.g.Clusters, Cluster{ID: id, Title: id})
	}

	inner := s.clone()
	inner.cluster = id

	if _, err := p.expect(tokLBrace, "'{'"); err != nil {
		return err
	}
	if err := p.parseStmtList(inner); err != nil {
		return err
	}
	_, err := p.expect(tokRBrace, "'}'")
	return err
}

func (p *parser) registerNode(s *scope, key string, own attrs) {
	idx, exists := p.nodeIndex[key]
	if !exists {
		idx = len(p.g.Nodes)
		p.g.Nodes = append(p.g.Nodes, Node{Key: key, Label: key, Cluster: s.cluster})
		p.nodeIndex[key] = idx
	}
	merged := mergeAttrs(s.nodeDefaults, own)
	if label, ok := merged["label"]; ok {
		p.g.Nodes[idx].Label = cleanLabel(label)
	}
	if shape, ok := merged["shape"]; ok {
		p.g.Nodes[idx].Shape = strings.ToLower(shape)
	}
}

func (p *parser) addEdge(s *scope, from, to string, own attrs) {
	merged := mergeAttrs(s.edgeDefaults, own)
	p.g.Edges = append(p.g.Edges, Edge{From: from, To: to, Label: cleanLabel(merged["label"])})
}

func (p *parser) applyGraphAttrs(s *scope, a attrs) {
	if v, ok := a["rankdir"]; ok && s.cluster == "" {
		p.g.RankDir = normalizeRankDir(v)
	}
	if v, ok := a["label"]; ok && s.cluster != "" {
		if idx, ok := p.clusterIndex[s.cluster]; ok {
			p.g.Clusters[idx].Title = cleanLabel(v)
		}
	}
}

// dropEmptyClusters removes subgraphs that ended up with no member nodes
// (pure layout/scoping subgraphs carry no diagram information).
func (p *parser) dropEmptyClusters() {
	has := map[string]bool{}
	for _, n := range p.g.Nodes {
		if n.Cluster != "" {
			has[n.Cluster] = true
		}
	}
	kept := p.g.Clusters[:0]
	for _, c := range p.g.Clusters {
		if has[c.ID] {
			kept = append(kept, c)
		}
	}
	p.g.Clusters = kept
}

func normalizeRankDir(v string) string {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "LR":
		return "LR"
	case "RL":
		return "RL"
	case "BT":
		return "BT"
	default:
		return "TB"
	}
}

// cleanLabel turns a raw Graphviz label into single-line display text:
// \n, \l, \r (Graphviz's line-break escapes) become a space, then runs of
// whitespace collapse and the ends trim.
func cleanLabel(raw string) string {
	raw = strings.ReplaceAll(raw, `\n`, " ")
	raw = strings.ReplaceAll(raw, `\l`, " ")
	raw = strings.ReplaceAll(raw, `\r`, " ")
	raw = strings.ReplaceAll(raw, "\n", " ")
	fields := strings.Fields(raw)
	return strings.Join(fields, " ")
}
