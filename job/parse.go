package job

import (
	"log"

	"github.com/mibk/dupl/syntax"
	"github.com/mibk/dupl/syntax/golang"
)

// Parse parses the Go files received on fchan and streams their serialized
// syntax trees. Files that fail to parse are logged and skipped.
func Parse(fchan <-chan string) <-chan []*syntax.Node {
	// parse AST
	achan := make(chan *syntax.Node)
	go func() {
		for file := range fchan {
			ast, err := golang.Parse(file)
			if err != nil {
				log.Println(err)
				continue
			}
			achan <- ast
		}
		close(achan)
	}()

	// serialize
	schan := make(chan []*syntax.Node)
	go func() {
		for ast := range achan {
			schan <- syntax.Serialize(ast)
		}
		close(schan)
	}()
	return schan
}
