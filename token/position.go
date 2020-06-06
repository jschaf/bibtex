package token

import (
	"fmt"
	"sort"
	"sync"
)

type Position struct {
	Filename string // filename, if any
	Offset   int    // offset, starting at 0
	Line     int    // line number, starting at 1
	Column   int    // column number, starting at 1 (byte count)
}

// IsValid reports whether the position is valid.
func (pos *Position) IsValid() bool { return pos.Line > 0 }

// Pos is a compact encoding of a source position within a file. It can be
// converted into a Position for a more convenient, but much larger,
// representation.
//
// The Pos value for a file is in the range [0, size].
//
// To create the Pos value for a specific source offset (measured in bytes),
// call File.Pos(offset) for a file. Given a Pos value p for a specific file f,
// the corresponding position value is given by calling f.Position(p).
//
// Pos values can be compared directly with the usual comparison operators:
// For two Pos values p and q, comparing p and q is equivalent to comparing the
// respective source file offsets.
type Pos int

// The zero value for Pos is NoPos; there is no file and line information
// associated with it, and NoPos.IsValid() is false. NoPos is always smaller
// than any other Pos value. The corresponding Position value for NoPos is the
// zero value for Position.
const NoPos Pos = 0

// IsValid reports whether the position is valid.
func (p Pos) IsValid() bool {
	return p != NoPos
}

// String returns a string in one of several forms:
//
//	file:line:column    valid position with file name
//	file:line           valid position with file name but no column (column == 0)
//	line:column         valid position without file name
//	line                valid position without file name and no column (column == 0)
//	file                invalid position with file name
//	-                   invalid position without file name
func (pos Position) String() string {
	s := pos.Filename
	if pos.IsValid() {
		if s != "" {
			s += ":"
		}
		s += fmt.Sprintf("%d", pos.Line)
		if pos.Column != 0 {
			s += fmt.Sprintf(":%d", pos.Column)
		}
	}
	if s == "" {
		s = "-"
	}
	return s
}

// ----------------------------------------------------------------------------
// A File is a handle for a file belonging to a FileSet.
// A File has a name, size, and line offset table.
type File struct {
	set  *FileSet
	name string // file name as provided
	base int    // Pos value range for this file is [base...base+size]
	size int    // file size

	// lines are protected by a mutex
	mutex sync.Mutex
	// lines contains the offset of the first character for each line. lines[0] is
	// always 0.
	lines []int
}

func NewFile(filename string, size int) *File {
	return &File{
		name:  filename,
		size:  size,
		mutex: sync.Mutex{},
		lines: []int{0},
	}
}

// Name returns the file name of file f as registered with AddFile.
func (f *File) Name() string {
	return f.name
}

// Base returns the base offset of file f as registered with AddFile.
func (f *File) Base() int {
	return f.base
}

// Size returns the size of file f as registered with AddFile.
func (f *File) Size() int {
	return f.size
}

// LineCount returns the number of lines in file f.
func (f *File) LineCount() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return len(f.lines)
}

// AddLine adds the line offset for a new line.
// The line offset must be larger than the previous offset for the previous
// line and smaller than the file size; otherwise the line offset is ignored.
func (f *File) AddLine(offset int) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if i := len(f.lines); (i == 0 || f.lines[i-1] < offset) && offset < f.size {
		f.lines = append(f.lines, offset)
	}
}

// Pos returns the Pos value for the given file offset. The offset
// must be <= f.Size().
// f.Pos(f.Offset(p)) == p.
func (f *File) Pos(offset int) Pos {
	if offset > f.size {
		panic("illegal file offset")
	}
	return Pos(f.base + offset)
}

// Offset returns the offset for the given file position p.
// p must be a valid Pos in the file.
// f.Offset(f.Pos(offset)) == offset.
func (f *File) Offset(p Pos) int {
	if int(p) < f.base || int(p) > f.base+f.size {
		panic("illegal Pos value")
	}
	return int(p) - f.base
}

// Line returns the line number for a given file position p.
// p must be a valid Pos value or NoPos.
func (f *File) Line(p Pos) int {
	return f.Position(p).Line
}

// Position returns the Position value for the given file position p.
func (f *File) Position(p Pos) Position {
	if p == NoPos {
		return Position{}
	}
	if int(p) < f.base || int(p) > f.base+f.size {
		panic("illegal Pos value")
	}
	offset := int(p) - f.base
	i := sort.Search(
		len(f.lines), func(i int) bool { return f.lines[i] > offset }) - 1
	return Position{
		Filename: f.name,
		Offset:   offset,
		Line:     i + 1,
		Column:   offset - f.lines[i] + 1,
	}
}

// ----------------------------------------------------------------------------
// FileSet

// A FileSet represents a set of source files. Methods of file sets are
// synchronized; multiple goroutines my invoke them correctly.
type FileSet struct {
	mutex sync.RWMutex // protects the file set
	base  int          // base offset for the next file
	files []*File      // list of files in the order added to the set
	last  *File        // cache of last file looked up
}

func NewFileSet() *FileSet {
	return &FileSet{
		base: 1, // 0 == NoPos
	}
}

// Base returns the minimum base offset that must be provided to AddFile when
// adding the next file.
func (s *FileSet) Base() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.base
}

// AddFile adds a new file with a given filename, base offset, and file size to
// the file set s and returns the file. Multiple files may have the same name.
// The base offset must not be smaller than the FileSet's Base(), and size must
// not be negative. As a special case, if a negative base is provided, the
// current value of FileSet's Base() is used instead.
//
// Adding the file will set the file set's Base() value to base + size + 1 as
// the minimum base value for the next file. The following relationship exists
// between a Pos value p for a given file offset offs:
//
//  int(p) = base + offs
//
// with offs in the range [0, size] and thus p in the range [base, base+size].
// For convenience, File.Pos may be used to create file-specific position values
// from a file offset.
func (s *FileSet) AddFile(filename string, base, size int) *File {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if base < 0 {
		base = s.base
	}
	if base < s.base || size < 0 {
		panic("illegal base or size")
	}
	f := &File{set: s, name: filename, base: base, size: size, lines: []int{0}}
	base += size + 1 // +1 because EOF also has a position
	if base < 0 {
		panic("token.Pos offset overflow (>2G of source code in file set)")
	}
	s.base = base
	s.files = append(s.files, f)
	s.last = f
	return f
}

// Iterate calls f for the files in the file set in the order they were added
// until f returns true.
func (s *FileSet) Iterate(f func(*File) bool) {
	for i := 0; ; i++ {
		var file *File
		s.mutex.RLock()
		if i < len(s.files) {
			file = s.files[i]
		}
		s.mutex.RUnlock()
		if file == nil || !f(file) {
			break
		}
	}
}

func searchFiles(a []*File, x int) int {
	return sort.Search(len(a), func(i int) bool { return a[i].base > x }) - 1
}

func (s *FileSet) file(p Pos) *File {
	s.mutex.RLock()

	// common case: p is in last file
	if f := s.last; f != nil && f.base <= int(p) && int(p) <= f.base+f.size {
		s.mutex.RUnlock()
		return f
	}

	// p is not in last file, search all files
	if i := searchFiles(s.files, int(p)); i >= 0 {
		f := s.files[i]
		if int(p) <= f.base+f.size {
			s.mutex.RUnlock()
			s.mutex.Lock()
			s.last = f // race is ok - s.last is only a cache
			s.mutex.Unlock()
			return f
		}
	}
	s.mutex.RUnlock()
	return nil
}

// File returns the file that contains position p. If no such file is found
// (for instance for p == NoPos), the result is nil.
func (s *FileSet) File(p Pos) (f *File) {
	if p != NoPos {
		f = s.file(p)
	}
	return
}

// Position converts a Pos p in the fileset into a Position value.
func (s *FileSet) Position(p Pos) Position {
	if p == NoPos {
		return Position{}
	}
	if f := s.file(p); f != nil {
		return f.Position(p)
	} else {
		return Position{}
	}
}
