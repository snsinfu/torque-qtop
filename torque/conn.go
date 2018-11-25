package torque

import (
	"net"
)

// A Conn represents a connection to a PBS server.
type Conn struct {
	conn *net.TCPConn
}

// Close closes the connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}
