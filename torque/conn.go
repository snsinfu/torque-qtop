package torque

// Conn is a connection to a PBS server.
type Conn interface {
	// User returns the name of the authorized user for the connection.
	User() string

	// ReadInt reads an integer from the connection.
	ReadInt() (int64, error)

	// ReadString reads a string from the connection.
	ReadString() (string, error)

	// WriteInt writes an integer to the connection.
	WriteInt(n int64) error

	// WriteString writes a string to the connection.
	WriteString(s string) error

	// Flush sends any buffered data to the server.
	Flush() error

	// Close closes the connection without flushing any buffered data.
	Close() error
}
