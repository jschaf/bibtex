package bibtex

// newAuthor creates a new author using the number of strings to infer
// the name structure as follows:
//
//     1 strings: Last
//     2 strings: First, Last
//     3 strings: First, Prefix, Last
//     4 strings: First, Prefix, Last, Suffix
func newAuthor(names ...string) Author {
	switch len(names) {
	case 0:
		panic("need at least 1 name")
	case 1:
		return Author{
			Last: names[0],
		}
	case 2:
		return Author{
			First: names[0],
			Last:  names[1],
		}
	case 3:
		return Author{
			First:  names[0],
			Prefix: names[1],
			Last:   names[2],
		}
	case 4:
		return Author{
			First:  names[0],
			Prefix: names[1],
			Last:   names[2],
			Suffix: names[3],
		}
	default:
		panic("too many names")
	}
}
