package domain

// FileEntry represents one item in an HTTP directory listing.
type FileEntry struct {
	Name  string // display name
	URL   string // absolute URL to this entry
	IsDir bool   // true if this entry is a directory
}
