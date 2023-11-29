package main

import (
	"errors"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const MaxFileSize = 1024 * 1024 * 100

type FileTunnel struct {
	Input  *os.File
	Output *os.File
	ready  bool
	conn   net.Conn
	writeL *sync.Mutex
	readL  *sync.Mutex
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

func (t *FileTunnel) canWrite() bool {
	t.writeL.Lock()
	defer t.writeL.Unlock()
	stat, err := os.Stat(*output)
	if err != nil {
		log.Printf("canWrite stat %s failed %s", *output, err)
		panic(err)
	}
	if stat.Size() < MaxFileSize {
		return true
	}
	if err := t.Output.Close(); err != nil {
		log.Printf("canWrite %s close failed %s", t.Output.Name(), err)
		panic(err)
	}
	log.Printf("canWrite wait %s empty", t.Output.Name())
	return t.outputReady()
}

func (t *FileTunnel) Write(p []byte) (int, error) {
	if !t.canWrite() {
		return 0, errors.New("no space left")
	}
	n, err := t.Output.Write(p)
	if err != nil {
		log.Printf("write %s failed %s", t.Output.Name(), err)
	}
	return n, err
}

func (t *FileTunnel) clearRead() {

	stat, err := os.Stat(*input)
	if err != nil {
		log.Printf("clearRead stat %s failed %s", *input, err)
		panic(err)
	}
	if stat.Size() < MaxFileSize {
		return
	}

	if err := t.Input.Close(); err != nil {
		log.Printf("clearRead close %s failed %s", t.Input.Name(), err)
		panic(err)
	}

	if err := os.Truncate(*input, 0); err != nil {
		log.Printf("clearRead truncate %s failed %s", t.Input.Name(), err)
		panic(err)
	}
	i, err := os.OpenFile(*input, os.O_RDONLY, 0644)
	if err != nil {
		log.Printf("clearRead open file %s failed %s", *input, err)
		panic(err)
	}
	log.Printf("%s ready clear", t.Input.Name())
	t.Input = i
}

func (t *FileTunnel) Read(p []byte) (int, error) {
	if !t.readyRead() {
		return 0, errors.New("no data writed")
	}
	t.readL.Lock()
	defer t.readL.Unlock()
	n, err := t.Input.Read(p)
	if err != nil && strings.Contains(err.Error(), "EOF") {
		t.clearRead()
	}
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
		writeL: &sync.Mutex{},
		readL:  &sync.Mutex{},
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

func (t *FileTunnel) outputReady() bool {
	for {
		// can not use fsnotify.fsnotify
		// Windows watch Linux file will generate error?
		info, err := os.Stat(*output)
		if err != nil {
			log.Printf("outputReady state %s failed %s", *output, err.Error())
			continue
		}
		if info.Size() != 0 {
			// log.Printf("outputReady state size = %d", info.Size())
			// time.Sleep(10 * time.Millisecond)
			continue
		}
		log.Printf("outputReady %s stat size = 0", t.Output.Name())
		o, err := os.OpenFile(*output, os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("outputReady open file %s failed %s", *output, err)
			panic(err)
		}
		t.Output = o
		return true
	}
}
