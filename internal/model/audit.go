package model

// AuditMsg represents an audit event message emitted when a user performs
// an auditable action (e.g., creating or following a shortened URL).
// Ts is a Unix timestamp, Action describes what happened,
// UserID is the authenticated user (nil for anonymous), and URL is the target.
type AuditMsg struct {
	Ts     int64   `json:"ts"`
	Action string  `json:"action"`
	UserID *string `json:"user_id,omitempty"`
	URL    string  `json:"url"`
}
