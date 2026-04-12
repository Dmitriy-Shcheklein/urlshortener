package postgres

const ConflictErrorMessage = "Conflict while creating new link, originalURL: "

func NewConflictError(original []byte, shorten []byte) *ConflictError {
	return &ConflictError{
		Original: original,
		Shorten:  shorten,
	}
}

type ConflictError struct {
	Original []byte
	Shorten  []byte
}

func (c *ConflictError) Error() string {
	return ConflictErrorMessage + string(c.Original)
}
