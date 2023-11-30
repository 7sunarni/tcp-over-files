package main

import (
	"errors"
	"log"
	"net"
	"strings"
	"time"
)

type WrapConn struct {
	Real net.Conn
}

func (c *WrapConn) Read(b []byte) (n int, err error) {
	n, err = c.Real.Read(b)
	if err != nil {
		if strings.Contains(err.Error(), "EOF") {
			return n, errors.New("CONNECTION CLOSED")
		}
		log.Printf("tcp connection read failed %s", err.Error())
	}
	return n, err
}
func (c *WrapConn) Write(b []byte) (n int, err error) {
	n, err = c.Real.Write(b)
	if err != nil {
		log.Printf("tcp connection write failed %s", err.Error())
	}
	return n, err
}
func (c *WrapConn) Close() error {
	return c.Real.Close()
}
func (c *WrapConn) LocalAddr() net.Addr {
	return c.Real.LocalAddr()
}
func (c *WrapConn) RemoteAddr() net.Addr {
	return c.Real.RemoteAddr()
}
func (c *WrapConn) SetDeadline(t time.Time) error {
	return c.Real.SetDeadline(t)
}
func (c *WrapConn) SetReadDeadline(t time.Time) error {
	return c.Real.SetReadDeadline(t)
}
func (c *WrapConn) SetWriteDeadline(t time.Time) error {
	return c.Real.SetWriteDeadline(t)
}
