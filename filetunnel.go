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
				log.Printf("FileTunnel copy %s:%s to %s failed: %s", t.conn.LocalAddr(), t.conn.RemoteAddr(), t.Output.Name(), err.Error())
				break
			}
		}
		closeC <- nil
	}()

	go func() {
		for {
			_, err := io.Copy(t.conn, t)
			if err != nil {
				log.Printf("FileTunnel copy %s to %s:%s failed: %s", t.Input.Name(), t.conn.LocalAddr(), t.conn.RemoteAddr(), err.Error())
				break
			}
		}
		closeC <- nil
	}()
	<-closeC
	t.Close()
}

func (t *FileTunnel) Close() {
	if err := t.conn.Close(); err != nil {
		log.Printf("FileTunnel %s close failed %s", t.conn.LocalAddr(), err)
	}
	if err := t.Input.Close(); err != nil {
		log.Printf("FileTunnel %s close failed %s", t.Input.Name(), err)
	}
	if err := t.Output.Close(); err != nil {
		log.Printf("FileTunnel %s close failed %s", t.Output.Name(), err)
	}
	if err := os.Truncate(t.Input.Name(), 0); err != nil {
		log.Printf("FileTunnel %s truncate failed %s", t.Output.Name(), err)
	}
}

func (t *FileTunnel) checkFileSize() bool {
	t.writeL.Lock()
	defer t.writeL.Unlock()
	stat, err := os.Stat(*output)
	if err != nil {
		log.Printf("FileTunnel check file size stat %s failed %s", *output, err)
		panic(err)
	}
	if stat.Size() < MaxFileSize {
		return true
	}
	if err := t.Output.Close(); err != nil {
		log.Printf("FileTunnel check file %s close failed %s", t.Output.Name(), err)
		panic(err)
	}
	log.Printf("FileTunnel file %s closed, waiting for empty", t.Output.Name())
	return t.waitFileEmpty()
}

func (t *FileTunnel) Write(p []byte) (int, error) {
	if !t.checkFileSize() {
		return 0, errors.New("no space left")
	}
	n, err := t.Output.Write(p)
	if err != nil {
		log.Printf("FileTunnel write %s failed %s", t.Output.Name(), err)
	}
	return n, err
}

func (t *FileTunnel) emptyFile() {
	stat, err := os.Stat(*input)
	if err != nil {
		log.Printf("FileTunnel empty file get %s stat failed %s", *input, err)
		panic(err)
	}
	if stat.Size() < MaxFileSize {
		return
	}

	if err := t.Input.Close(); err != nil {
		log.Printf("FileTunnel empty file close %s failed %s", t.Input.Name(), err)
		panic(err)
	}

	if err := os.Truncate(*input, 0); err != nil {
		log.Printf("FileTunnel empty file truncate %s failed %s", t.Input.Name(), err)
		panic(err)
	}
	i, err := os.OpenFile(*input, os.O_RDONLY, 0644)
	if err != nil {
		log.Printf("FileTunnel empty file reopen file %s failed %s", *input, err)
		panic(err)
	}
	log.Printf("FileTunnel already empty file %s", t.Input.Name())
	t.Input = i
}

func (t *FileTunnel) Read(p []byte) (int, error) {
	if !t.waitFileReady() {
		panic("FileTunnel input not ready")
	}
	t.readL.Lock()
	defer t.readL.Unlock()
	n, err := t.Input.Read(p)
	if err != nil && strings.Contains(err.Error(), "EOF") {
		t.emptyFile()
	}
	return n, err
}

func NewFileTunnel() *FileTunnel {
	i, err := os.OpenFile(*input, os.O_RDONLY, 0644)
	if err != nil {
		log.Printf("FileTunnel open file %s as input failed %s", *input, err)
		panic(err)
	}

	o, err := os.OpenFile(*output, os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("FileTunnel open file %s as output failed %s", *output, err)
		panic(err)
	}

	return &FileTunnel{
		Input:  i,
		Output: o,
		ready:  false,
		writeL: &sync.Mutex{},
		readL:  &sync.Mutex{},
	}
}

func (t *FileTunnel) waitFileReady() bool {
	if t.ready {
		return true
	}

	for {
		// can not use fsnotify.fsnotify
		// Windows watch Linux file will generate error?
		info, err := os.Stat(t.Input.Name())
		if err != nil {
			log.Printf("FileTunnel waiting file ready, get state %s failed %s", t.Input.Name(), err.Error())
			continue
		}
		if info.Size() <= 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		log.Printf("FileTunnel waiting file %s has data, starting use it as input", t.Input.Name())
		t.ready = true
		break
	}
	return true
}

func (t *FileTunnel) waitFileEmpty() bool {
	for {
		// can not use fsnotify.fsnotify
		// Windows watch Linux file will generate error?
		info, err := os.Stat(*output)
		if err != nil {
			log.Printf("FileTunnel waiting file empty, get state %s failed %s", *output, err.Error())
			continue
		}
		if info.Size() != 0 {
			// log.Printf("outputReady state size = %d", info.Size())
			// time.Sleep(10 * time.Millisecond)
			continue
		}
		log.Printf("FileTunnel file %s empty, starting reuse it as new output", *output)
		o, err := os.OpenFile(*output, os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("FileTunnel file %s failed %s", *output, err)
			panic(err)
		}
		t.Output = o
		return true
	}
}
