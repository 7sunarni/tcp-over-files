package main

import (
	"errors"
	"io"
	"log"
	"net"
	"os"
	"time"
)

type FileTunnel struct {
	Input  *os.File
	Output *os.File
	ready  bool
	conn   net.Conn
}

func (t *FileTunnel) Tunnel() {
	closeC := make(chan interface{})
	go func() {
		for {
			_, err := io.Copy(t, t.conn)
			if err != nil {
				log.Printf("end write %s:%s to %s failed: %s", t.conn.LocalAddr(), t.conn.RemoteAddr(), t.Output.Name(), err.Error())
				break
			}
		}
		log.Printf("end write %s:%s to %s", t.conn.LocalAddr(), t.conn.RemoteAddr(), t.Output.Name())
		closeC <- nil
	}()

	go func() {
		for {
			_, err := io.Copy(t.conn, t)
			if err != nil {
				log.Printf("copy %s to %s:%s failed: %s", t.Input.Name(), t.conn.LocalAddr(), t.conn.RemoteAddr(), err.Error())
				break
			}
		}
		log.Printf("end write %s to %s:%s", t.Input.Name(), t.conn.LocalAddr(), t.conn.RemoteAddr())
		closeC <- nil
	}()
	<-closeC
	t.Close()
}

func (t *FileTunnel) Close() {
	if err := t.conn.Close(); err != nil {
		log.Printf("%s close failed %s", t.conn.LocalAddr(), err)
	}
	if err := t.Input.Close(); err != nil {
		log.Printf("%s close failed %s", t.Input.Name(), err)
	}
	if err := t.Output.Close(); err != nil {
		log.Printf("%s close failed %s", t.Output.Name(), err)
	}
	if err := os.Truncate(t.Input.Name(), 0); err != nil {
		log.Printf("%s truncate failed %s", t.Output.Name(), err)
	}
}

func (t *FileTunnel) Write(p []byte) (int, error) {
	n, err := t.Output.Write(p)
	return n, err
}

func (t *FileTunnel) Read(p []byte) (int, error) {
	if !t.readyRead() {
		return 0, errors.New("no data writed")
	}
	n, err := t.Input.Read(p)
	return n, err
}

func NewFileTunnel() *FileTunnel {
	i, err := os.OpenFile(*input, os.O_RDONLY, 0644)
	if err != nil {
		log.Printf("open file %s failed %s", *input, err)
		return nil
	}

	o, err := os.OpenFile(*output, os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("open file %s failed %s", *output, err)
		return nil
	}

	return &FileTunnel{
		Input:  i,
		Output: o,
		ready:  false,
	}
}

func (t *FileTunnel) readyRead() bool {
	if t.ready {
		return true
	}

	for {
		// can not use fsnotify.fsnotify
		// Windows watch Linux file will generate error?
		info, err := os.Stat(t.Input.Name())
		if err != nil {
			log.Printf("state %s failed %s", t.Input.Name(), err.Error())
			continue
		}
		if info.Size() > 0 {
			log.Println("stat size > 0")
			t.ready = true
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
}
