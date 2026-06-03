# difflite

Syntax-highlighted `git diff` viewer for the terminal. Pipe it and forget it.

## Install

```bash
go install github.com/freedom090/difflite@latest
```

## Usage

```bash
git diff | difflite
git diff --cached | difflite
git diff main..feature | difflite
git show | difflite --no-collapse
```

`--no-collapse` shows every context line without collapsing long unchanged sections.
