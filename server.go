package httpstream

import (
	"errors"
	"io"
	"net/http"
	"sync"

	"github.com/go-contrib/uuid"
)

type Server struct {
	c        map[string]chan io.ReadWriteCloser
	mx       *sync.Mutex
	acceptCh chan pipeReq
	openCh   chan pipeReq
}

func NewServer() *Server {
	return &Server{
		c:        make(map[string]chan io.ReadWriteCloser, 100),
		mx:       new(sync.Mutex),
		acceptCh: make(chan pipeReq, 100),
		openCh:   make(chan pipeReq, 100),
	}
}

func (s *Server) newReq(ch chan pipeReq) (io.ReadWriteCloser, error) {
	id := uuid.NewV4().String()
	res := make(chan io.ReadWriteCloser)

	ch <- pipeReq{id, res}

	// writer will always be first, since reader isn't added to the map until after
	w := <-res
	r := <-res
	if w == nil {
		return nil, errors.New("failed to aquire write channel")
	}
	if r == nil {
		w.Close()
		s.mx.Lock()
		delete(s.c, id)
		s.mx.Unlock()
		return nil, errors.New("failed to aquire read channel")
	}
	return &pipe{r, w}, nil
}

func setHeaders(h http.Header, id string) {
	h.Set("Stream-ID", id)
	h.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	h.Set("Pragma", "no-cache")
	h.Set("Expires", "0")
}

func (s *Server) Accept() (io.ReadWriteCloser, error) {
	return s.newReq(s.acceptCh)
}
func (s *Server) Open() (io.ReadWriteCloser, error) {
	return s.newReq(s.openCh)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")
	var p pipeReq
	switch {
	case req.Method == "GET":
		p = <-s.openCh
	case req.Method == "POST" && id == "":
		p = <-s.acceptCh
	case req.Method == "POST" && id != "":
		s.mx.Lock()
		ch := s.c[id]
		delete(s.c, id)
		s.mx.Unlock()

		if ch == nil {
			http.NotFound(w, req)
			return
		}

		c, buf, err := w.(http.Hijacker).Hijack()
		if err != nil {
			close(ch)
			return
		}
		ch <- &bufConn{c, buf}

		return
	default:
		http.Error(w, "only GET and POST are allowed", 405)
		return
	}
	if p.res == nil {
		http.NotFound(w, req)
		return
	}

	setHeaders(w.Header(), p.id)
	w.WriteHeader(200)

	c, buf, err := w.(http.Hijacker).Hijack()
	if err != nil {
		close(p.res)
		return
	}
	defer c.Close()
	err = buf.Flush()
	if err != nil {
		close(p.res)
		return
	}

	s.mx.Lock()
	s.c[p.id] = p.res
	s.mx.Unlock()

	p.res <- c
}
