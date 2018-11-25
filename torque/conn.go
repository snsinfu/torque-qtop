package torque

import (
	"bufio"
	"net"

	"github.com/snsinfu/torque-qtop/dis"
)

// A Conn represents a connection to a PBS server.
type Conn struct {
	conn *net.TCPConn
	r    *bufio.Reader
	w    *bufio.Writer
	user  string
}

// User returns the name of the authorized user for the connection.
func (c *Conn) User() string {
	return c.user
}

// ReadInt reads an integer from the connection.
func (c *Conn) ReadInt() (int64, error) {
	return dis.ReadInt(c.r)
}

// ReadString reads a string from the connection.
func (c *Conn) ReadString() (string, error) {
	return dis.ReadString(c.r)
}

// WriteInt writes an integer to the connection.
func (c *Conn) WriteInt(n int64) error {
	_, err := c.w.WriteString(dis.EncodeInt(n))
	return err
}

// WriteString writes a string to the connection.
func (c *Conn) WriteString(s string) error {
	_, err := c.w.WriteString(dis.EncodeString(s))
	return err
}

// Flush sends any buffered data to the server.
func (c *Conn) Flush() error {
	return c.w.Flush()
}

// Close closes the connection without flushing any buffered data.
func (c *Conn) Close() error {
	return c.conn.Close()
}
