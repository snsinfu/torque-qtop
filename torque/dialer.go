package torque

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/user"

	"github.com/snsinfu/torque-qtop/dis"
	"github.com/snsinfu/torque-qtop/pipeenc"
)

const (
	trqAuthConnection  = 1
	trqGetActiveServer = 2
	authTypeIFF        = 1
	authBufferSize     = 1024
)

// A Dialer contains options for connecting to PBS server.
type Dialer struct {
	AuthAddr string
}

// DefaultDialer
var DefaultDialer = Dialer{
	AuthAddr: "/tmp/trqauthd-unix",
}

// GetActiveServer returns the address of the active PBS server on the system.
func (d *Dialer) GetActiveServer() (string, error) {
	auth, err := net.Dial("unix", d.AuthAddr)
	if err != nil {
		return "", err
	}
	defer auth.Close()

	// Request: GetActiveServer
	enc := pipeenc.NewEncoder()
	enc.PutInt(trqGetActiveServer)

	if _, err := auth.Write([]byte(enc.String())); err != nil {
		return "", err
	}

	// Response: (error, host, port)
	buf := make([]byte, authBufferSize)

	n, err := auth.Read(buf)
	if err != nil {
		return "", err
	}

	dec := pipeenc.NewDecoder(string(buf[:n]))

	respCode, err := dec.GetInt()
	if err != nil {
		return "", err
	}

	if respCode != 0 {
		return "", fmt.Errorf("trqauthd error (%d)", respCode)
	}

	host, err := dec.GetString()
	if err != nil {
		return "", err
	}

	port, err := dec.GetInt()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%d", host, port), nil
}

// Dial connects to a PBS server.
func (d *Dialer) Dial(address string) (Conn, error) {
	addr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		return nil, err
	}

	if err := authorize(conn, d.AuthAddr); err != nil {
		conn.Close()
		return nil, err
	}

	me, err := user.Current()
	c := &pbsConn{
		conn: conn,
		r:    bufio.NewReader(conn),
		w:    bufio.NewWriter(conn),
		user: me.Username,
	}
	return c, nil
}

// authorize grants authorization for given TCP connection to PBS server.
func authorize(conn *net.TCPConn, authAddr string) error {
	auth, err := net.Dial("unix", authAddr)
	if err != nil {
		return err
	}
	defer auth.Close()

	me, err := user.Current()
	if err != nil {
		return err
	}

	username := me.Username
	pid := os.Getpid()
	port := conn.LocalAddr().(*net.TCPAddr).Port
	server := conn.RemoteAddr().(*net.TCPAddr)

	// Request: AuthConnection(host, port, auth_type, user, pid, client_port)
	enc := pipeenc.NewEncoder()
	enc.PutInt(trqAuthConnection)
	enc.PutString(server.IP.String())
	enc.PutInt(server.Port)
	enc.PutInt(authTypeIFF)
	enc.PutString(username)
	enc.PutInt(pid)
	enc.PutInt(port)

	if _, err := auth.Write([]byte(enc.String())); err != nil {
		return err
	}

	// Response: (error)
	buf := make([]byte, authBufferSize)

	n, err := auth.Read(buf)
	if err != nil {
		return err
	}

	dec := pipeenc.NewDecoder(string(buf[:n]))

	respCode, err := dec.GetInt()
	if err != nil {
		return err
	}

	if respCode != 0 {
		return fmt.Errorf("code %d", respCode)
	}

	return nil
}

// pbsConn is a real connection to a PBS server. It implements Conn interface.
type pbsConn struct {
	conn *net.TCPConn
	r    *bufio.Reader
	w    *bufio.Writer
	user  string
}

func (c *pbsConn) User() string {
	return c.user
}

func (c *pbsConn) ReadInt() (int64, error) {
	return dis.ReadInt(c.r)
}

func (c *pbsConn) ReadString() (string, error) {
	return dis.ReadString(c.r)
}

func (c *pbsConn) WriteInt(n int64) error {
	_, err := c.w.WriteString(dis.EncodeInt(n))
	return err
}

func (c *pbsConn) WriteString(s string) error {
	_, err := c.w.WriteString(dis.EncodeString(s))
	return err
}

func (c *pbsConn) Flush() error {
	return c.w.Flush()
}

func (c *pbsConn) Close() error {
	return c.conn.Close()
}

// Dial connects to the active PBS server on the system using DefaultDialer.
func Dial() (Conn, error) {
	address, err := DefaultDialer.GetActiveServer()
	if err != nil {
		return nil, err
	}
	return DefaultDialer.Dial(address)
}
