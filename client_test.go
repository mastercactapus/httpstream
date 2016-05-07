package httpstream

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClient_Open(t *testing.T) {
	reqCh := make(chan struct{})
	resCh := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "POST" && req.URL.Query().Get("id") == "" {
			w.Header().Set("Stream-ID", "specialid")
			w.WriteHeader(200)
			w.(http.Flusher).Flush()
			io.WriteString(w, "bar")
			<-reqCh
		} else if req.Method == "POST" {
			assert.Equal(t, "specialid", req.URL.Query().Get("id"))
			close(reqCh)
			buf := make([]byte, 3)
			io.ReadFull(req.Body, buf)
			assert.Equal(t, "foo", string(buf))
			close(resCh)
		} else {
			t.Error("unexpected method:", req.Method)
		}
	}))
	defer srv.Close()

	c := NewClient(nil, srv.URL)
	rw, err := c.Open()
	if err != nil {
		t.Fatal("open failed:", err)
	}
	defer rw.Close()
	assert.Nil(t, err)
	io.WriteString(rw, "foo")

	buf := make([]byte, 3)
	io.ReadFull(rw, buf)
	assert.Equal(t, "bar", string(buf))
	tc := time.NewTimer(time.Second)
	select {
	case <-resCh:
	case <-tc.C:
		t.Error("should have recieved 'foo' on server end")
	}
}

func TestClient_Accept(t *testing.T) {
	reqCh := make(chan struct{})
	resCh := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "GET" {
			w.Header().Set("Stream-ID", "specialid")
			w.WriteHeader(200)
			w.(http.Flusher).Flush()
			io.WriteString(w, "bar")
			<-reqCh
		} else if req.Method == "POST" {
			assert.Equal(t, "specialid", req.URL.Query().Get("id"))
			close(reqCh)
			buf := make([]byte, 3)
			io.ReadFull(req.Body, buf)
			assert.Equal(t, "foo", string(buf))
			close(resCh)
		} else {
			t.Error("unexpected method:", req.Method)
		}
	}))
	defer srv.Close()

	c := NewClient(nil, srv.URL)
	rw, err := c.Accept()
	if err != nil {
		t.Fatal("open failed:", err)
	}
	defer rw.Close()
	assert.Nil(t, err)
	io.WriteString(rw, "foo")

	buf := make([]byte, 3)
	io.ReadFull(rw, buf)
	assert.Equal(t, "bar", string(buf))
	tc := time.NewTimer(time.Second)
	select {
	case <-resCh:
	case <-tc.C:
		t.Error("should have recieved 'foo' on server end")
	}
}
