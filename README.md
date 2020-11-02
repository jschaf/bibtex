# Bibtex parser in Go

https://pkg.go.dev/github.com/jschaf/bibtex

A parser for [Bibtex][bibtex-wiki] files, a reference formatting language, in Go. I needed a parser for my static site generator and the existing library https://github.com/caltechlibrary/bibtex was difficult to integrate into idiomatic Go.

- Uses a real, recursive descent parser based on the Golang parser to read Bibtex files into an AST.
- Handles parsing different author formats.
- Reasonably fast: parses a 30,000 line Bibtex file in 16 ms. 

```shell script
go get github.com/jschaf/bibtex
```

## Example: read a bibtex file into an AST

```go
func readBibtexFile() ([]bibtex.Entry, error) {
	f, err := os.Open("refs.bib")
	if err != nil {
		return nil, err
	}
	entries, err := bibtex.Read(f)
	if err != nil {
		return nil, err
	}
	return entries, nil
}
```

## Example: format a bibtex entry into HTML

Formats a Bibtex entry into HTML:

```bibtex
@article{chattopadhyay2019procella,
  title={Procella: Unifying serving and analytical data at YouTube},
  author={Chattopadhyay, Biswapesh and Dutta, Priyam and Liu, Weiran and Tinn, Ott and Mccormick, Andrew and Mokashi, Aniket and Harvey, Paul and Gonzalez, Hector and Lomax, David and Mittal, Sagar and others},
  journal={Proceedings of the VLDB Endowment},
  volume={12},
  number={12},
  pages={2022--2034},
  year={2019},
  publisher={VLDB Endowment}
}
```

> B. Chattopadhyay, P. Dutta, W. Liu, O. Tinn, A. Mccormick, A. Mokashi, P. Harvey, H. Gonzalez, D. Lomax, S. Mittal *et al*, "Procella: Unifying serving and analytical data at YouTube," in *Proceedings of the VLDB Endowment*, Vol. 12, 2019.

```go
// formatEntry returns an HTML string in a IEEE citation style.
func formatEntry(entry bibtex.Entry) string {
	w := strings.Builder{}
	w.WriteString("<div>")

	// Format all authors.
	authors := entry.Author
	for i, author := range authors {
		sp := strings.Split(author.First, " ")
		for _, s := range sp {
			if r, _ := utf8.DecodeRuneInString(s); r != utf8.RuneError {
				w.WriteRune(r)
				w.WriteString(". ")
			}
		}
		w.WriteString(author.Last)
		if i < len(authors)-2 {
			w.WriteString(", ")
		} else if i == len(authors)-2 {
			if authors[len(authors)-1].IsOthers() {
				w.WriteString(" <em>et al</em>")
				break

			} else {
				w.WriteString(" and ")
			}
		}
	}

	title := entry.Tags[bibtex.FieldTitle]
	title = trimBraces(title)
	w.WriteString(`, "`)
	w.WriteString(title)
	w.WriteString(`,"`)

	journal := entry.Tags[bibtex.FieldJournal]
	journal = trimBraces(journal)
	if journal != "" {
		w.WriteString(" in <em class=cite-journal>")
		w.WriteString(journal)
		w.WriteString("</em>")
	}

	vol := entry.Tags[bibtex.FieldVolume]
	vol = trimBraces(vol)
	if vol != "" {
		w.WriteString(", Vol. ")
		w.WriteString(vol)
	}

	year := entry.Tags[bibtex.FieldYear]
	year = trimBraces(year)
	if year != "" {
		w.WriteString(", ")
		w.WriteString(year)
	}

	w.WriteString(".")
	w.WriteString(`</div>`)
	return w.String()
}

func trimBraces(s string) string {
	return strings.TrimFunc(s, func(r rune) bool {
		return r == '{' || r == '}'
	})
}
```

[bibtex-wiki]: https://en.wikipedia.org/wiki/BibTeX

# Features

- [x] Parse authors.
- [ ] Resolve string abbreviations.
- [ ] Resolve [Crossref] references.

[Crossref]: https://tex.stackexchange.com/questions/401138/what-is-the-bibtex-crossref-field-used-for

