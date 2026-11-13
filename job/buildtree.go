package job

import (
	"github.com/mibk/dupl/suffixtree"
	"github.com/mibk/dupl/syntax"
)

// BuildTree consumes every node sequence from schan, feeding it into a new
// suffix tree, and returns the finished tree together with the concatenated
// node data. The tree is terminated with a sentinel node, so it is ready to
// be searched as soon as BuildTree returns.
func BuildTree(schan <-chan []*syntax.Node) (*suffixtree.STree, []*syntax.Node) {
	t := suffixtree.New()
	data := make([]*syntax.Node, 0, 100)
	for seq := range schan {
		data = append(data, seq...)
		for _, node := range seq {
			t.Update(node)
		}
	}
	t.Update(&syntax.Node{Type: -1})
	return t, data
}
