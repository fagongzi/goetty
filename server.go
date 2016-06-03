package goetty

import (
	"github.com/CodisLabs/codis/pkg/utils/atomic2"
	"net"
	"sync"
	"time"
)

type IdGenerator interface {
	NewId() interface{}
}

type Int64IdGenerator struct {
	counter atomic2.Int64
}

func (self Int64IdGenerator) NewId() interface{} {
	return self.counter.Incr()
}

func NewInt64IdGenerator() IdGenerator {
	return &Int64IdGenerator{}
}

type UUIDV4IdGenerator struct {
}

func (self UUIDV4IdGenerator) NewId() interface{} {
	return NewV4UUID()
}

func NewUUIDV4IdGenerator() IdGenerator {
	return &UUIDV4IdGenerator{}
}

type sessionMap struct {
	sync.RWMutex
	sessions map[interface{}]IOSession
}

const DEFAULT_SESSION_SIZE = 64

type Server struct {
	addr     string
	listener *net.TCPListener

	sessionMaps map[int]*sessionMap

	in  sync.Pool
	out sync.Pool

	readBufSize, writeBufSize int

	decoder Decoder
	encoder Encoder

	generator IdGenerator

	stopOnce *sync.Once
	stopped  bool
}

func NewServer(addr string, decoder Decoder, encoder Encoder, generator IdGenerator) *Server {
	return NewServerSize(addr, decoder, encoder, BUF_READ_SIZE, BUF_WRITE_SIZE, generator)
}

func NewServerSize(addr string, decoder Decoder, encoder Encoder, readBufSize, writeBufSize int, generator IdGenerator) *Server {
	s := &Server{
		addr:        addr,
		sessionMaps: make(map[int]*sessionMap, DEFAULT_SESSION_SIZE),

		decoder:      decoder,
		encoder:      encoder,
		readBufSize:  readBufSize,
		writeBufSize: writeBufSize,

		generator: generator,

		stopOnce: &sync.Once{},
	}

	for i := 0; i < DEFAULT_SESSION_SIZE; i++ {
		s.sessionMaps[i] = &sessionMap{
			sessions: make(map[interface{}]IOSession),
		}
	}

	return s
}

func (self *Server) Stop() {
	self.stopOnce.Do(func() {
		self.stopped = true
		self.listener.Close()

		for _, sessions := range self.sessionMaps {
			for _, session := range sessions.sessions {
				session.Close()
			}
		}
	})
}

func (self *Server) Serve(loopFn func(IOSession) error) error {
	addr, err := net.ResolveTCPAddr("tcp", self.addr)

	if err != nil {
		return err
	}

	self.listener, err = net.ListenTCP("tcp", addr)

	if err != nil {
		return err
	}

	var tempDelay time.Duration
	for {
		conn, err := self.listener.AcceptTCP()

		if self.stopped {
			if nil != conn {
				conn.Close()
			}

			return nil
		}

		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		tempDelay = 0

		session := newClientIOSession(self.generator.NewId(), conn, self)
		self.addSession(session)

		go func() {
			defer self.deleteSession(session)
			loopFn(session)
		}()
	}
}

func (self *Server) closeSession(session IOSession) {
	self.deleteSession(session)
	session.Close()
}

func (self *Server) addSession(session IOSession) {
	m := self.sessionMaps[session.Hash()%DEFAULT_SESSION_SIZE]
	m.Lock()
	defer m.Unlock()

	m.sessions[session.Id()] = session
}

func (self *Server) deleteSession(session IOSession) {
	m := self.sessionMaps[session.Hash()%DEFAULT_SESSION_SIZE]
	m.Lock()
	defer m.Unlock()

	delete(m.sessions, session.Id())
}

func (self *Server) GetSession(id interface{}) IOSession {
	m := self.sessionMaps[getHash(id)%DEFAULT_SESSION_SIZE]
	m.RLock()
	defer m.RUnlock()

	return m.sessions[id]
}

func (self *Server) read(conn net.Conn) (interface{}, error) {
	buf, ok := self.in.Get().(*ByteBuf)

	if !ok {
		buf = NewByteBuf(self.readBufSize)
	}

	defer func() {
		buf.Clear()
		if !ok {
			self.in.Put(buf)
		}
	}()

	for {
		_, err := buf.ReadFrom(conn)

		if err != nil {
			return nil, err
		}

		complete, msg, err := self.decoder.Decode(buf)

		if nil != err {
			return nil, err
		}

		if complete {
			return msg, nil
		}
	}

	return nil, nil
}

func (self *Server) write(msg interface{}, conn net.Conn) error {
	buf, ok := self.out.Get().(*ByteBuf)

	if !ok {
		buf = NewByteBuf(self.writeBufSize)
	}

	defer func() {
		buf.Clear()
		if !ok {
			self.out.Put(buf)
		}
	}()

	err := self.encoder.Encode(msg, buf)

	if err != nil {
		return err
	}

	_, bytes, _ := buf.ReadAll()

	n, err := conn.Write(bytes)

	if err != nil {
		return err
	}

	if n != len(bytes) {
		return WriteErr
	}

	return nil
}
