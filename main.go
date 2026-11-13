package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mibk/dupl/job"
	"github.com/mibk/dupl/printer"
	"github.com/mibk/dupl/suffixtree"
	"github.com/mibk/dupl/syntax"
)

const defaultThreshold = 15

var (
	paths     = []string{"."}
	vendor    = flag.Bool("vendor", false, "")
	verbose   = flag.Bool("verbose", false, "")
	threshold = flag.Int("threshold", defaultThreshold, "")
	files     = flag.Bool("files", false, "")

	html     = flag.Bool("html", false, "")
	plumbing = flag.Bool("plumbing", false, "")
)

const (
	vendorDirPrefix = "vendor" + string(filepath.Separator)
	vendorDirInPath = string(filepath.Separator) + vendorDirPrefix
)

func init() {
	flag.BoolVar(verbose, "v", false, "alias for -verbose")
	flag.IntVar(threshold, "t", defaultThreshold, "alias for -threshold")
}

func main() {
	flag.Usage = usage
	flag.Parse()
	if *html && *plumbing {
		log.Fatal("you can have either plumbing or HTML output")
	}
	if flag.NArg() > 0 {
		paths = flag.Args()
	}

	if *verbose {
		log.Println("Building suffix tree")
	}
	t, data := job.BuildTree(job.Parse(filesFeed()))

	if *verbose {
		log.Println("Searching for clones")
	}
	if err := printDupls(newPrinter(), findDupls(t, data, *threshold)); err != nil {
		log.Fatal(err)
	}
}

// findDupls searches the suffix tree for duplicate token sequences of at
// least threshold length and streams those that form complete syntax units.
func findDupls(t *suffixtree.STree, data []*syntax.Node, threshold int) <-chan syntax.Match {
	duplChan := make(chan syntax.Match)
	go func() {
		defer close(duplChan)
		for m := range t.FindDuplOver(threshold) {
			match := syntax.FindSyntaxUnits(data, m, threshold)
			if len(match.Frags) > 0 {
				duplChan <- match
			}
		}
	}()
	return duplChan
}

// newPrinter returns the printer selected by the output flags.
func newPrinter() printer.Printer {
	switch {
	case *html:
		return printer.NewHTML(os.Stdout, ioutil.ReadFile)
	case *plumbing:
		return printer.NewPlumbing(os.Stdout, ioutil.ReadFile)
	default:
		return printer.NewText(os.Stdout, ioutil.ReadFile)
	}
}

func filesFeed() <-chan string {
	if *files {
		fchan := make(chan string)
		go func() {
			s := bufio.NewScanner(os.Stdin)
			for s.Scan() {
				f := s.Text()
				fchan <- strings.TrimPrefix(f, "./")
			}
			close(fchan)
		}()
		return fchan
	}
	return crawlPaths(paths)
}

func crawlPaths(paths []string) <-chan string {
	fchan := make(chan string)
	go func() {
		for _, path := range paths {
			info, err := os.Lstat(path)
			if err != nil {
				log.Fatal(err)
			}
			if !info.IsDir() {
				fchan <- path
				continue
			}
			err = filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
				if !*vendor && (strings.HasPrefix(path, vendorDirPrefix) ||
					strings.Contains(path, vendorDirInPath)) {
					return nil
				}
				if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
					fchan <- path
				}
				return nil
			})
			if err != nil {
				log.Fatal(err)
			}
		}
		close(fchan)
	}()
	return fchan
}

func printDupls(p printer.Printer, duplChan <-chan syntax.Match) error {
	groups := make(map[string][][]*syntax.Node)
	for dupl := range duplChan {
		groups[dupl.Hash] = append(groups[dupl.Hash], dupl.Frags...)
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if err := p.PrintHeader(); err != nil {
		return err
	}
	for _, k := range keys {
		uniq := unique(groups[k])
		if len(uniq) > 1 {
			if err := p.PrintClones(uniq); err != nil {
				return err
			}
		}
	}
	return p.PrintFooter()
}

// unique drops fragments that start at the same position of the same file.
func unique(group [][]*syntax.Node) [][]*syntax.Node {
	type location struct {
		filename string
		pos      int
	}
	seen := make(map[location]bool)

	var newGroup [][]*syntax.Node
	for _, seq := range group {
		loc := location{seq[0].Filename, seq[0].Pos}
		if !seen[loc] {
			seen[loc] = true
			newGroup = append(newGroup, seq)
		}
	}
	return newGroup
}

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: dupl [flags] [paths]

Paths:
  If the given path is a file, dupl will use it regardless of
  the file extension. If it is a directory, it will recursively
  search for *.go files in that directory.

  If no path is given, dupl will recursively search for *.go
  files in the current directory.

Flags:
  -files
    	read file names from stdin one at each line
  -html
    	output the results as HTML, including duplicate code fragments
  -plumbing
    	plumbing (easy-to-parse) output for consumption by scripts or tools
  -t, -threshold size
    	minimum token sequence size as a clone (default 15)
  -vendor
    	check files in vendor directory
  -v, -verbose
    	explain what is being done

Examples:
  dupl -t 100
    	Search clones in the current directory of size at least
    	100 tokens.
  dupl $(find app/ -name '*_test.go')
    	Search for clones in tests in the app directory.
  find app/ -name '*_test.go' |dupl -files
    	The same as above.`)
	os.Exit(2)
}
