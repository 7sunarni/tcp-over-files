package multiplexer

import (
	"bytes"
	"net"
	"testing"
)

func TestSessionRead(t *testing.T) {
	data := make([]byte, 0)

	for i := 0; i < 4; i++ {
		{
			f := Frame{
				ID:     54321,
				Status: 0,
				Data:   []byte("foo"),
			}
			data = append(data, f.bytes()...)
		}

		{
			f := Frame{
				ID:     56789,
				Status: 0,
				Data:   []byte("bar"),
			}
			data = append(data, f.bytes()...)
		}
	}

	s := Session{
		conns: map[uint32]*WrapConn{},
		r:     bytes.NewBuffer(data),
	}

	{
		n, err := net.Dial("tcp", "0.0.0.0:54321")
		if err != nil {
			t.Fatalf("dial 54321 failed:%s", err)
		}
		s.conns[54321] = &WrapConn{
			id:   54321,
			real: n,
		}
	}
	{
		n, err := net.Dial("tcp", "0.0.0.0:56789")
		if err != nil {
			t.Fatalf("dial 56789 failed:%s", err)
		}
		s.conns[56789] = &WrapConn{
			id:   56789,
			real: n,
		}
	}
	s.Read()
}

func TestSessionClient(t *testing.T) {
	// listen 10000, forward to 20000
	s, err := NewClient("0.0.0.0:10000", "0.0.0.0:20000")
	if err != nil {
		t.Fatalf("listen failed %s", err)
		return
	}
	s.Client()
}

func TestSessionServer(t *testing.T) {
	s, err := NewServer("0.0.0.0:30000", "0.0.0.0:10000")
	if err != nil {

		t.Fatalf("listen failed %s", err)
		return
	}
	s.Server()
}

// curl 127.0.0.1:30000/test.file -o test.file.1
