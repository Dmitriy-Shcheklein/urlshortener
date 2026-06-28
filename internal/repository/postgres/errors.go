package postgres

// ConflictErrorMessage is the prefix for ConflictError error messages.
const ConflictErrorMessage = "Conflict while creating new link, originalURL: "

// NewConflictError creates a new ConflictError for a URL that was already shortened.
// original is the original URL that conflicted, and shorten is the existing short URL.
func NewConflictError(original []byte, shorten []byte) *ConflictError {
	return &ConflictError{
		Original: original,
		Shorten:  shorten,
	}
}

// ConflictError indicates that a URL shortening request failed because
// the original URL was already shortened. It contains both the original
// URL and the existing short URL.
type ConflictError struct {
	Original []byte
	Shorten  []byte
}

// Error returns a human-readable error message containing the original URL.
func (c *ConflictError) Error() string {
	return ConflictErrorMessage + string(c.Original)
}
