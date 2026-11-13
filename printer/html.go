package printer

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"regexp"

	"github.com/mibk/dupl/syntax"
)

type htmlprinter struct {
	iota int
	w    io.Writer
	ReadFile
}

func NewHTML(w io.Writer, fread ReadFile) Printer {
	return &htmlprinter{w: w, ReadFile: fread}
}

func (p *htmlprinter) PrintHeader() error {
	_, err := fmt.Fprint(p.w, `<!DOCTYPE html>
<meta charset="utf-8"/>
<title>Duplicates</title>
<style>
	pre {
		background-color: #FFD;
		border: 1px solid #E2E2E2;
		padding: 1ex;
	}
</style>
`)
	return err
}

func (p *htmlprinter) PrintClones(dups [][]*syntax.Node) error {
	p.iota++
	fmt.Fprintf(p.w, "<h1>#%d found %d clones</h1>\n", p.iota, len(dups))

	clones, err := clonesInfo(p.ReadFile, dups, true)
	if err != nil {
		return err
	}
	for _, cl := range clones {
		fmt.Fprintf(p.w, "<h2>%s:%d</h2>\n<pre>%s</pre>\n", cl.filename, cl.lineStart,
			html.EscapeString(string(cl.fragment)))
	}
	return nil
}

func (*htmlprinter) PrintFooter() error { return nil }

// fragment returns the source code between from and to, de-indented, with
// the text preceding it on the first line replaced by whitespace so that
// its alignment is preserved.
func fragment(file []byte, from, to int) []byte {
	start := findLineBeg(file, from)
	content := append(toWhitespace(file[start:from]), file[from:to]...)
	return deindent(content)
}

func findLineBeg(file []byte, index int) int {
	for i := index; i >= 0; i-- {
		if file[i] == '\n' {
			return i + 1
		}
	}
	return 0
}

func toWhitespace(str []byte) []byte {
	var out []byte
	for _, c := range bytes.Runes(str) {
		if c == '\t' {
			out = append(out, '\t')
		} else {
			out = append(out, ' ')
		}
	}
	return out
}

func deindent(block []byte) []byte {
	const maxVal = 99
	min := maxVal
	re := regexp.MustCompile(`(^|\n)(\t*)\S`)
	for _, line := range re.FindAllSubmatch(block, -1) {
		indent := line[2]
		if len(indent) < min {
			min = len(indent)
		}
	}
	if min == 0 || min == maxVal {
		return block
	}
	block = block[min:]
Loop:
	for i := 0; i < len(block); i++ {
		if block[i] == '\n' && i != len(block)-1 {
			for j := 0; j < min; j++ {
				if block[i+j+1] != '\t' {
					continue Loop
				}
			}
			block = append(block[:i+1], block[i+1+min:]...)
		}
	}
	return block
}
