package item

// Mode represents different file types.
type Mode uint8

const (
	// ModeRegular represents a normal file
	ModeRegular = iota
	// ModeDir represents a directory
	ModeDir
)
