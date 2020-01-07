package goetty

import (
	"testing"
)

func TestLengthBasedEncoder(t *testing.T) {
	e := NewIntLengthFieldBasedEncoder(NewRawEncoder())
	buf := NewByteBuf(32)
	err := e.Encode([]byte("hello"), buf)
	if err != nil {
		t.Errorf("TestLengthBasedEncoder failed with %+v", err)
		return
	}

	if 9 != buf.Readable() {
		t.Errorf("TestLengthBasedEncoder failed with unexpect size %d",
			buf.Readable())
		return
	}

	n, _ := buf.ReadInt()
	if 5 != n {
		t.Errorf("TestLengthBasedEncoder failed with unexpect length size %d",
			n)
		return
	}

	_, v, _ := buf.ReadAll()
	if string(v) != "hello" {
		t.Errorf("TestLengthBasedEncoder failed with unexpect value %s",
			string(v))
		return
	}
}
