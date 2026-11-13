package printer

import (
	"fmt"
	"io"
	"sort"

	"github.com/mibk/dupl/syntax"
)

type text struct {
	cnt int
	w   io.Writer
	ReadFile
}

func NewText(w io.Writer, fread ReadFile) Printer {
	return &text{w: w, ReadFile: fread}
}

func (p *text) PrintHeader() error { return nil }

func (p *text) PrintClones(dups [][]*syntax.Node) error {
	p.cnt++
	fmt.Fprintf(p.w, "found %d clones:\n", len(dups))
	clones, err := clonesInfo(p.ReadFile, dups, false)
	if err != nil {
		return err
	}
	for _, cl := range clones {
		fmt.Fprintf(p.w, "  %s:%d,%d\n", cl.filename, cl.lineStart, cl.lineEnd)
	}
	return nil
}

func (p *text) PrintFooter() error {
	_, err := fmt.Fprintf(p.w, "\nFound total %d clone groups.\n", p.cnt)
	return err
}

// clonesInfo resolves each duplicate fragment to its location in the
// source, sorted by file name and starting line. If withFragment is set,
// the de-indented source code of each fragment is included as well.
func clonesInfo(fread ReadFile, dups [][]*syntax.Node, withFragment bool) ([]clone, error) {
	clones := make([]clone, len(dups))
	for i, dup := range dups {
		if len(dup) == 0 {
			panic("zero length dup")
		}
		nstart, nend := dup[0], dup[len(dup)-1]

		file, err := fread(nstart.Filename)
		if err != nil {
			return nil, err
		}

		cl := clone{filename: nstart.Filename}
		cl.lineStart, cl.lineEnd = blockLines(file, nstart.Pos, nend.End)
		if withFragment {
			cl.fragment = fragment(file, nstart.Pos, nend.End)
		}
		clones[i] = cl
	}
	sort.Sort(byNameAndLine(clones))
	return clones, nil
}

func blockLines(file []byte, from, to int) (int, int) {
	line := 1
	lineStart, lineEnd := 0, 0
	for offset, b := range file {
		if b == '\n' {
			line++
		}
		if offset == from {
			lineStart = line
		}
		if offset == to-1 {
			lineEnd = line
			break
		}
	}
	return lineStart, lineEnd
}

type clone struct {
	filename  string
	lineStart int
	lineEnd   int
	fragment  []byte
}

type byNameAndLine []clone

func (c byNameAndLine) Len() int { return len(c) }

func (c byNameAndLine) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

func (c byNameAndLine) Less(i, j int) bool {
	if c[i].filename == c[j].filename {
		return c[i].lineStart < c[j].lineStart
	}
	return c[i].filename < c[j].filename
}
