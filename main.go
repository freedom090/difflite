package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/alecthomas/chroma/v2/styles"
)

func main() {
	noCollapse := flag.Bool("no-collapse", false, "show all context lines without collapsing")
	flag.Parse()

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "difflite: error reading stdin: %v\n", err)
		os.Exit(1)
	}

	content := strings.TrimRight(string(input), "\n")
	if content == "" {
		return
	}

	lines := strings.Split(content, "\n")
	files := ParseDiff(lines)

	if len(files) == 0 {
		return
	}

	style := styles.Monokai
	for _, fd := range files {
		ext := fd.NewName
		if ext == "" || ext == "/dev/null" {
			ext = fd.OldName
		}
		lexer := detectLexer(ext)
		RenderFileDiff(os.Stdout, fd, lexer, style, !*noCollapse)
	}
}
