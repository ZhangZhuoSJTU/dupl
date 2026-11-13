package suffixtree

import "sort"

type Match struct {
	Ps  []Pos
	Len Pos
}

// contextList groups suffix positions by their left context, i.e. by the
// value of the token immediately preceding each position.
type contextList map[int][]Pos

func (c contextList) getAll() []Pos {
	keys := make([]int, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var ps []Pos
	for _, k := range keys {
		ps = append(ps, c[k]...)
	}
	return ps
}

func (c contextList) append(c2 contextList) {
	for lc, ps := range c2 {
		c[lc] = append(c[lc], ps...)
	}
}

// FindDuplOver find pairs of maximal duplicities over a threshold
// length.
func (t *STree) FindDuplOver(threshold int) <-chan Match {
	auxTran := newTran(0, 0, t.root)
	ch := make(chan Match)
	go func() {
		walkTrans(auxTran, 0, threshold, ch)
		close(ch)
	}()
	return ch
}

func walkTrans(parent *tran, length, threshold int, ch chan<- Match) contextList {
	s := parent.state

	cl := make(contextList)

	if len(s.trans) == 0 {
		start := parent.end + 1 - Pos(length)
		ch := 0
		if start > 0 {
			ch = s.tree.data[start-1].Val()
		}
		cl[ch] = []Pos{start}
		return cl
	}

	for _, t := range s.trans {
		ln := length + t.len()
		cl2 := walkTrans(t, ln, threshold, ch)
		if ln >= threshold {
			cl.append(cl2)
		}
	}
	if length >= threshold && len(cl) > 1 {
		m := Match{cl.getAll(), Pos(length)}
		ch <- m
	}
	return cl
}
