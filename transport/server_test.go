package transport

import (
	"errors"
	"testing"
	"time"

	"github.com/fagongzi/goetty"
	"github.com/fagongzi/goetty/codec/simple"
	"github.com/stretchr/testify/assert"
)

func TestServer(t *testing.T) {
	var rs *Session
	s := createTestServer(func(session *Session, req interface{}) error {
		rs = session
		msg := req.(string)
		if msg == "exit" {
			return errors.New(msg)
		}
		session.OnResp(msg)
		return nil
	})
	defer s.Stop()

	assert.NoError(t, s.Start(), "TestServer failed")

	conn := createConn(t)
	defer conn.Close()

	value := "hello"
	assert.NoError(t, conn.WriteAndFlush(value), "TestServer failed")

	rsp, err := conn.Read()
	assert.NoError(t, err, "TestServer failed")
	assert.Equal(t, value, rsp, "TestServer failed")

	_, ok := s.(*server).sessions.Load(rs.ID)
	assert.True(t, ok, "TestServer failed")

	assert.NoError(t, conn.WriteAndFlush("exit"), "TestServer failed")

	time.Sleep(time.Millisecond * 100)
	assert.True(t, rs.Closed(), "TestServer failed")

	_, ok = s.(*server).sessions.Load(rs.ID)
	assert.False(t, ok, "TestServer failed")

	assert.Error(t, conn.WriteAndFlush(value), "TestServer failed")
	assert.Error(t, rs.OnResp(value), "TestServer failed")
}

func createTestServer(handleFunc func(*Session, interface{}) error) Server {
	addr := "127.0.0.1:12345"
	decoder, encoder := simple.NewStringCodec()

	return New(addr, handleFunc, WithCodec(decoder, encoder))
}

func createConn(t *testing.T) goetty.IOSession {
	addr := "127.0.0.1:12345"
	decoder, encoder := simple.NewStringCodec()

	conn := goetty.NewConnector(addr,
		goetty.WithClientDecoder(decoder),
		goetty.WithClientEncoder(encoder))
	_, err := conn.Connect()
	assert.NoError(t, err, "createConn failed")
	return conn
}
