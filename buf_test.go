package goetty

import (
	"testing"
)

func TestNewWrap(t *testing.T) {
	value := []byte{1, 2, 3}
	buf := WrapBytes(value)
	if buf.Readable() != 3 {
		t.Errorf("TestWrap failed, expect readable is 3, but %+v", buf.Readable())
		return
	}

	v, err := buf.ReadByte()
	if err != nil {
		t.Errorf("TestWrap failed, expect no error, but %+v", err)
		return
	}
	if v != 1 {
		t.Errorf("TestWrap failed, expect 1, but %+v", v)
		return
	}

	v, err = buf.ReadByte()
	if err != nil {
		t.Errorf("TestWrap failed, expect no error, but %+v", err)
		return
	}
	if v != 2 {
		t.Errorf("TestWrap failed, expect 2, but %+v", v)
		return
	}

	v, err = buf.ReadByte()
	if err != nil {
		t.Errorf("TestWrap failed, expect no error, but %+v", err)
		return
	}
	if v != 3 {
		t.Errorf("TestWrap failed, expect 3, but %+v", v)
		return
	}
}

func TestWrap(t *testing.T) {
	buf := NewByteBuf(4)
	buf.Write([]byte{5, 6, 7})

	value := []byte{1, 2, 3}
	buf.Wrap(value)
	if buf.Readable() != 3 {
		t.Errorf("TestWrap failed, expect readable is 3, but %+v", buf.Readable())
		return
	}

	v, err := buf.ReadByte()
	if err != nil {
		t.Errorf("TestWrap failed, expect no error, but %+v", err)
		return
	}
	if v != 1 {
		t.Errorf("TestWrap failed, expect 1, but %+v", v)
		return
	}

	v, err = buf.ReadByte()
	if err != nil {
		t.Errorf("TestWrap failed, expect no error, but %+v", err)
		return
	}
	if v != 2 {
		t.Errorf("TestWrap failed, expect 2, but %+v", v)
		return
	}

	v, err = buf.ReadByte()
	if err != nil {
		t.Errorf("TestWrap failed, expect no error, but %+v", err)
		return
	}
	if v != 3 {
		t.Errorf("TestWrap failed, expect 3, but %+v", v)
		return
	}
}

func TestSlice(t *testing.T) {
	buf := NewByteBuf(32)
	buf.Write([]byte("hello"))
	s := buf.Slice(0, 5)
	if string(s.Data()) != "hello" {
		t.Errorf("TestSlice failed, expect hello, but %+v", string(s.Data()))
		return
	}
}

func TestWrittenDataAfterMark(t *testing.T) {
	buf := NewByteBuf(32)
	buf.MarkWrite()
	buf.Write([]byte("hello"))
	s := buf.WrittenDataAfterMark()
	if string(s.Data()) != "hello" {
		t.Errorf("TestWrittenDataAfterMark failed, expect hello, but %+v", string(s.Data()))
		return
	}
}

func TestExpansion(t *testing.T) {
	buf := NewByteBuf(256)
	data := make([]byte, 257, 257)
	buf.Write(data)
	EqualNow(t, cap(buf.buf), 512)
}