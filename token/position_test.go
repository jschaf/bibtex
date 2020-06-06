package token

import (
	"fmt"
	"testing"
)

func checkPos(t *testing.T, msg string, got, want Position) {
	if got.Filename != want.Filename {
		t.Errorf("%s: got filename = %q; want %q", msg, got.Filename, want.Filename)
	}
	if got.Offset != want.Offset {
		t.Errorf("%s: got offset = %d; want %d", msg, got.Offset, want.Offset)
	}
	if got.Line != want.Line {
		t.Errorf("%s: got line = %d; want %d", msg, got.Line, want.Line)
	}
	if got.Column != want.Column {
		t.Errorf("%s: got column = %d; want %d", msg, got.Column, want.Column)
	}
}

func TestNoPos(t *testing.T) {
	if NoPos.IsValid() {
		t.Errorf("NoPos should not be valid")
	}
	var fset *FileSet
	checkPos(t, "nil NoPos", fset.Position(NoPos), Position{})
	fset = NewFileSet()
	checkPos(t, "fset NoPos", fset.Position(NoPos), Position{})
}

var tests = []struct {
	filename string
	source   []byte // may be nil
	size     int
	lines    []int
}{
	{"a", []byte{}, 0, []int{}},
	{"b", []byte("01234"), 5, []int{0}},
	{"c", []byte("\n\n\n\n\n\n\n\n\n"), 9, []int{0, 1, 2, 3, 4, 5, 6, 7, 8}},
	{"d", nil, 100, []int{0, 5, 10, 20, 30, 70, 71, 72, 80, 85, 90, 99}},
	{"e", nil, 777, []int{0, 80, 100, 120, 130, 180, 267, 455, 500, 567, 620}},
	{"f", []byte("package p\n\nimport \"fmt\""), 23, []int{0, 10, 11}},
	{"g", []byte("package p\n\nimport \"fmt\"\n"), 24, []int{0, 10, 11}},
	{"h", []byte("package p\n\nimport \"fmt\"\n "), 25, []int{0, 10, 11, 24}},
}

func linecol(lines []int, offs int) (int, int) {
	prevLineOffs := 0
	for line, lineOffs := range lines {
		if offs < lineOffs {
			return line, offs - prevLineOffs + 1
		}
		prevLineOffs = lineOffs
	}
	return len(lines), offs - prevLineOffs + 1
}

func verifyPositions(t *testing.T, fset *FileSet, f *File, lines []int) {
	for offs := 0; offs < f.Size(); offs++ {
		p := f.Pos(offs)
		offs2 := f.Offset(p)
		if offs2 != offs {
			t.Errorf("%s, Offset: got offset %d; want %d", f.Name(), offs2, offs)
		}
		line, col := linecol(lines, offs)
		msg := fmt.Sprintf("%s (offs = %d, p = %d)", f.Name(), offs, p)
		checkPos(t, msg, f.Position(f.Pos(offs)), Position{f.Name(), offs, line, col})
		checkPos(t, msg, fset.Position(p), Position{f.Name(), offs, line, col})
	}
}

func TestPositions(t *testing.T) {
	const delta = 7 // a non-zero base offset increment
	fset := NewFileSet()
	for _, test := range tests {
		// verify consistency of test case
		if test.source != nil && len(test.source) != test.size {
			t.Errorf("%s: inconsistent test case: got file size %d; want %d", test.filename, len(test.source), test.size)
		}

		// add file and verify name and size
		f := fset.AddFile(test.filename, fset.Base()+delta, test.size)
		if f.Name() != test.filename {
			t.Errorf("got filename %q; want %q", f.Name(), test.filename)
		}
		if f.Size() != test.size {
			t.Errorf("%s: got file size %d; want %d", f.Name(), f.Size(), test.size)
		}
		if fset.File(f.Pos(0)) != f {
			t.Errorf("%s: f.Pos(0) was not found in f", f.Name())
		}

		// add lines individually and verify all positions
		for i, offset := range test.lines {
			f.AddLine(offset)
			if f.LineCount() != i+1 {
				t.Errorf("%s, AddLine: got line count %d; want %d", f.Name(), f.LineCount(), i+1)
			}
			// adding the same offset again should be ignored
			f.AddLine(offset)
			if f.LineCount() != i+1 {
				t.Errorf("%s, AddLine: got unchanged line count %d; want %d", f.Name(), f.LineCount(), i+1)
			}
			verifyPositions(t, fset, f, test.lines[0:i+1])
		}
	}
}

func TestFiles(t *testing.T) {
	fset := NewFileSet()
	for i, test := range tests {
		base := fset.Base()
		if i%2 == 1 {
			// Setting a negative base is equivalent to
			// fset.Base(), so test some of each.
			base = -1
		}
		fset.AddFile(test.filename, base, test.size)
		j := 0
		fset.Iterate(func(f *File) bool {
			if f.Name() != tests[j].filename {
				t.Errorf("got filename = %s; want %s", f.Name(), tests[j].filename)
			}
			j++
			return true
		})
		if j != i+1 {
			t.Errorf("got %d files; want %d", j, i+1)
		}
	}
}

// FileSet.File should return nil if Pos is past the end of the FileSet.
func TestFileSetPastEnd(t *testing.T) {
	fset := NewFileSet()
	for _, test := range tests {
		fset.AddFile(test.filename, fset.Base(), test.size)
	}
	if f := fset.File(Pos(fset.Base())); f != nil {
		t.Errorf("got %v, want nil", f)
	}
}

func TestFileSetCacheUnlikely(t *testing.T) {
	fset := NewFileSet()
	offsets := make(map[string]int)
	for _, test := range tests {
		offsets[test.filename] = fset.Base()
		fset.AddFile(test.filename, fset.Base(), test.size)
	}
	for file, pos := range offsets {
		f := fset.File(Pos(pos))
		if f.Name() != file {
			t.Errorf("got %q at position %d, want %q", f.Name(), pos, file)
		}
	}
}
