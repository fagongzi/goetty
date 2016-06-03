package goetty

import (
	"net"
	"sync"
)

type IOSession interface {
	Id() interface{}
	Hash() int
	Close() error
	Read() (interface{}, error)
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
	return self.svr.read(self.conn)
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
	defer self.Unlock()

	self.attrs[key] = value
}

func (self clientIOSession) GetAttr(key string) interface{} {
	self.Lock()
	defer self.Unlock()

	return self.attrs[key]
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
