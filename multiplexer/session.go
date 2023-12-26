package multiplexer

import (
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
)

type WrapConn struct {
	id   uint32
	real net.Conn
}

type Session struct {
	w           io.Writer
	r           io.Reader
	l           *sync.Mutex
	conns       map[uint32]*WrapConn
	readChan    chan Frame
	writeChan   chan Frame
	listenPort  string
	forwardPort string
}

func NewServer(listen string, target string) (*Session, error) {
	conn, err := net.Dial("tcp", target)
	if err != nil {
		log.Printf("server dial %s failed %s", target, err)
		return nil, err
	}
	s := &Session{
		l:          &sync.Mutex{},
		w:          conn,
		r:          conn,
		conns:      make(map[uint32]*WrapConn),
		readChan:   make(chan Frame, 200),
		writeChan:  make(chan Frame, 200),
		listenPort: listen,
	}
	return s, nil
}

func (s *Session) Read() {
	for {
		data := make([]byte, SizeIndex+DataLength)
		d, err := s.r.Read(data)
		if err != nil {
			log.Printf("session read failed %s", err)
			return
		}
		if d != SizeIndex+DataLength {
			log.Printf("read data length %d not equal %d", d, SizeIndex+DataLength)
		}
		f := NewFrame(data)
		if f == nil {
			log.Printf("new frame empty")
			continue
		}
		s.writeChan <- *f
	}
}

func (s *Session) Run() {
	for {
		select {
		case f := <-s.readChan:
			s.w.Write(f.bytes())
		case f := <-s.writeChan:
			if _, ok := s.conns[f.ID]; !ok {
				if s.forwardPort == "" {
					log.Printf("%d connection is empty", f.ID)
					continue
				}
				n, err := net.Dial("tcp", s.forwardPort)
				if err != nil {
					log.Printf("dial %s failed %s", s.forwardPort, err)
					continue
				}
				s.addConn(WrapConn{id: f.ID, real: n})
			}
			conn := s.conns[f.ID]
			conn.real.Write(f.Data)
		}
	}
}

func (s *Session) addConn(w WrapConn) {
	if w.id == 0 {
		for {
			id := rand.Uint32()
			if _, ok := s.conns[id]; ok {
				continue
			}
			w.id = id
			break
		}
	}
	s.l.Lock()
	s.conns[w.id] = &w
	s.l.Unlock()
	go s.readConn(w)
}

func (s *Session) Server() {
	l, err := net.Listen("tcp", s.listenPort)
	if err != nil {
		log.Printf("listen %s failed %s", s.listenPort, err)
		return
	}
	go s.Run()
	go s.Read()
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("acccept connection failed %s", err)
			continue
		}
		wrapConn := WrapConn{
			real: conn,
		}
		s.addConn(wrapConn)
	}
}

func NewClient(listen string, forward string) (*Session, error) {
	s := &Session{
		l:           &sync.Mutex{},
		conns:       make(map[uint32]*WrapConn),
		readChan:    make(chan Frame, 200),
		writeChan:   make(chan Frame, 200),
		listenPort:  listen,
		forwardPort: forward,
	}
	return s, nil
}

func (s *Session) Client() {
	l, err := net.Listen("tcp", s.listenPort)
	if err != nil {
		log.Printf("listen %s failed %s", s.listenPort, err)
		return
	}
	conn, err := l.Accept()
	if err != nil {
		log.Printf("acccept connection failed %s", err)
		return
	}
	s.w = conn
	s.r = conn
	go s.Run()
	s.Read()
}

var firstID uint32

func (s *Session) readConn(w WrapConn) error {
	for {
		data := make([]byte, DataLength)
		n, err := w.real.Read(data)
		if err != nil {
			log.Printf("session read failed %s", err)
			return err
		}
		// if n != DataLength {
		// log.Printf("session read %d not equal %d", n, DataLength)
		// }
		if firstID == 0 {
			firstID = w.id
		}
		if firstID != w.id {
			log.Printf("%d read data", w.id)
		}
		s.readChan <- Frame{
			ID:   w.id,
			Data: data[0:n],
			Size: uint32(n),
		}
		if firstID != w.id {
			log.Printf("%d send data", w.id)
		}
	}
}
