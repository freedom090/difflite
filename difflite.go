package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

type LineType int

const (
	LineContext  LineType = iota
	LineAddition
	LineDeletion
)

type DiffLine struct {
	Type    LineType
	Content string
}

type Hunk struct {
	Header string
	Lines  []DiffLine
}

type FileDiff struct {
	Header  string
	OldName string
	NewName string
	Binary  bool
	Hunks   []Hunk
}

// ANSI truecolor backgrounds
var (
	bgContext = [3]uint8{28, 28, 30}
	bgAdd     = [3]uint8{22, 50, 22}
	bgDel     = [3]uint8{50, 22, 22}
)

const (
	headerColor = "\033[1;38;2;100;149;237m" // bold cornflower blue
	hunkColor   = "\033[38;2;100;149;237m"   // cornflower blue
	dimItalic   = "\033[2;3m"
	reset       = "\033[0m"
)

func ParseDiff(lines []string) []FileDiff {
	var files []FileDiff
	var curFile *FileDiff
	var curHunk *Hunk
	inHunk := false

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff --git"):
			if curFile != nil {
				if curHunk != nil {
					curFile.Hunks = append(curFile.Hunks, *curHunk)
					curHunk = nil
				}
				files = append(files, *curFile)
			}
			curFile = &FileDiff{Header: line}
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				curFile.OldName = parts[2]
				curFile.NewName = parts[3]
			}
			inHunk = false

		case strings.HasPrefix(line, "Binary files"):
			if curFile != nil {
				curFile.Binary = true
			}

		case strings.HasPrefix(line, "--- "):
			if curFile != nil {
				curFile.OldName = strings.TrimPrefix(line, "--- ")
			}

		case strings.HasPrefix(line, "+++ "):
			if curFile != nil {
				curFile.NewName = strings.TrimPrefix(line, "+++ ")
			}

		case strings.HasPrefix(line, "@@"):
			if curFile != nil {
				if curHunk != nil {
					curFile.Hunks = append(curFile.Hunks, *curHunk)
				}
				curHunk = &Hunk{Header: line}
				inHunk = true
			}

		case inHunk && curHunk != nil:
			if len(line) == 0 {
				continue
			}
			prefix := line[0]
			switch prefix {
			case ' ':
				curHunk.Lines = append(curHunk.Lines, DiffLine{Type: LineContext, Content: line[1:]})
			case '+':
				curHunk.Lines = append(curHunk.Lines, DiffLine{Type: LineAddition, Content: line[1:]})
			case '-':
				curHunk.Lines = append(curHunk.Lines, DiffLine{Type: LineDeletion, Content: line[1:]})
			}
		}
	}

	if curHunk != nil && curFile != nil {
		curFile.Hunks = append(curFile.Hunks, *curHunk)
	}
	if curFile != nil {
		files = append(files, *curFile)
	}

	return files
}

func detectLexer(filename string) chroma.Lexer {
	if filename == "/dev/null" {
		return lexers.Fallback
	}
	// Strip "a/" or "b/" prefix
	if len(filename) > 2 && filename[1] == '/' {
		filename = filename[2:]
	}
	l := lexers.Match(filename)
	if l == nil {
		l = lexers.Fallback
	}
	return l
}

func CollapseHunk(h Hunk, threshold int) Hunk {
	if len(h.Lines) <= threshold {
		return h
	}

	var out []DiffLine
	i := 0
	for i < len(h.Lines) {
		if h.Lines[i].Type != LineContext {
			out = append(out, h.Lines[i])
			i++
			continue
		}

		// Measure run of context lines
		start := i
		for i < len(h.Lines) && h.Lines[i].Type == LineContext {
			i++
		}
		runLen := i - start
		keep := 3

		if runLen <= threshold {
			for j := start; j < i; j++ {
				out = append(out, h.Lines[j])
			}
			continue
		}

		// Collapse: first 3, placeholder, last 3
		head := min(keep, runLen)
		tail := min(keep, runLen-head)
		hidden := runLen - head - tail

		for j := start; j < start+head; j++ {
			out = append(out, h.Lines[j])
		}

		placeholder := fmt.Sprintf("  %d unchanged lines hidden  ", hidden)
		out = append(out, DiffLine{Type: LineContext, Content: placeholder})

		for j := i - tail; j < i; j++ {
			out = append(out, h.Lines[j])
		}
	}

	return Hunk{Header: h.Header, Lines: out}
}

func renderLine(w io.Writer, dl DiffLine, tokens []chroma.Token, style *chroma.Style) {
	var bg [3]uint8
	var prefix string

	switch dl.Type {
	case LineAddition:
		bg = bgAdd
		prefix = "+"
	case LineDeletion:
		bg = bgDel
		prefix = "-"
	default:
		bg = bgContext
		prefix = " "
	}

	fmt.Fprintf(w, "\033[48;2;%d;%d;%dm%s", bg[0], bg[1], bg[2], prefix)

	if len(tokens) == 0 {
		fmt.Fprintf(w, "%s\n", reset)
		return
	}

	for _, tok := range tokens {
		entry := style.Get(tok.Type)
		if entry.IsZero() {
			fmt.Fprint(w, tok.Value)
			continue
		}
		r, g, b := entry.Colour.Red(), entry.Colour.Green(), entry.Colour.Blue()
		fmt.Fprintf(w, "\033[38;2;%d;%d;%dm%s\033[39m", r, g, b, tok.Value)
	}
	fmt.Fprintf(w, "%s\n", reset)
}

func RenderFileDiff(w io.Writer, fd FileDiff, lexer chroma.Lexer, style *chroma.Style, collapse bool) {
	if fd.Binary {
		name := fd.NewName
		if name == "" || name == "/dev/null" {
			name = fd.OldName
		}
		if len(name) > 2 && name[1] == '/' {
			name = name[2:]
		}
		fmt.Fprintf(w, "%sBinary file: %s%s\n", dimItalic, name, reset)
		return
	}

	fmt.Fprintf(w, "%s%s%s\n", headerColor, fd.Header, reset)

	for _, hunk := range fd.Hunks {
		fmt.Fprintf(w, "%s%s%s\n", hunkColor, hunk.Header, reset)

		if collapse {
			hunk = CollapseHunk(hunk, 6)
		}

		for _, dl := range hunk.Lines {
			iterator, err := lexer.Tokenise(nil, dl.Content)
			var tokens []chroma.Token
			if err == nil {
				for t := iterator(); t.Type != chroma.EOFType; t = iterator() {
					tokens = append(tokens, t)
				}
			}
			renderLine(w, dl, tokens, style)
		}
	}
}
