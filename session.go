package goetty

import (
	"net"
	"sync"
	"time"
)

type IOSession interface {
	Id() interface{}
	Hash() int
	Close() error
	Read() (interface{}, error)
	ReadTimeout(timeout time.Duration) (interface{}, error)
	Write(msg interface{}) error
	SetAttr(key string, value interface{})
	GetAttr(key string) interface{}
	RemoteAddr() string
}

type clientIOSession struct {
	sync.RWMutex
	id    interface{}
	conn  net.Conn
	svr   *Server
	attrs map[string]interface{}
}

func newClientIOSession(id interface{}, conn net.Conn, svr *Server) IOSession {
	return &clientIOSession{
		id:    id,
		conn:  conn,
		svr:   svr,
		attrs: make(map[string]interface{}),
	}
}

func (self clientIOSession) Read() (interface{}, error) {
	return self.svr.read(self.conn, 0)
}

func (self clientIOSession) ReadTimeout(timeout time.Duration) (interface{}, error) {
	return self.svr.read(self.conn, timeout)
}

func (self clientIOSession) Write(msg interface{}) error {
	return self.svr.write(msg, self.conn)
}

func (self clientIOSession) Close() error {
	return self.conn.Close()
}

func (self clientIOSession) Id() interface{} {
	return self.id
}

func (self clientIOSession) Hash() int {
	return getHash(self.id)
}

func (self clientIOSession) SetAttr(key string, value interface{}) {
	self.Lock()
	self.attrs[key] = value
	self.Unlock()
}

func (self clientIOSession) GetAttr(key string) interface{} {
	self.Lock()
	v := self.attrs[key]
	self.Unlock()
	return v
}

func (self clientIOSession) RemoteAddr() string {
	if nil != self.conn {
		return self.conn.RemoteAddr().String()
	}

	return ""
}

func getHash(id interface{}) int {
	if v, ok := id.(int64); ok {
		return int(v)
	} else if v, ok := id.(int); ok {
		return v
	} else if v, ok := id.(string); ok {
		return hashCode(v)
	}

	return 0
}
