package httpconn

import (
	"bufio"
	"errors"
	"io"
	"net"
)

/* Routes:

POST /
	Create new connection, response has 'Stream-ID' of new ID, body is READ stream
GET /
	Wait for connection, response has 'Stream-ID' of new ID, body is READ stream
POST /?id=<id>
	Connect to WRITE stream of <id>
*/

var errNotFound = errors.New("no stream with specified ID")
var errClosed = errors.New("closed")

type bufConn struct {
	net.Conn
	*bufio.ReadWriter
}

func (b *bufConn) Read(p []byte) (int, error) {
	return b.ReadWriter.Read(p)
}

type pipe struct {
	io.ReadCloser
	io.WriteCloser
}

func (p *pipe) Close() error {
	p.ReadCloser.Close()
	p.WriteCloser.Close()
	return nil
}

type pipeReq struct {
	id  string
	res chan net.Conn
}
