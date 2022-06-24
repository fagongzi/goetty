package redis

import (
	"errors"
	"strconv"
	"testing"

	"fmt"

	"github.com/fagongzi/goetty/v2/buf"
	"github.com/fagongzi/util/hack"
	"github.com/stretchr/testify/assert"
)

func TestParserCommandReplyForStatus(t *testing.T) {
	status := "OK"

	buf := buf.NewByteBuf(1024)
	WriteStatus([]byte(status), buf)
	checkStatusReply(t, buf, status)
}

func TestParserCommandReplyForStatusNotComplete(t *testing.T) {
	status := "OK"

	buf := buf.NewByteBuf(1024)
	buf.WriteByte('+')
	buf.Write([]byte(status))

	checkNotComplete(t, buf)

	buf.Write(Delims)

	checkStatusReply(t, buf, status)
}

func TestParserCommandReplyForError(t *testing.T) {
	errInfo := "this is a error"

	buf := buf.NewByteBuf(1024)
	WriteError([]byte(errInfo), buf)

	checkErrorReply(t, buf, fmt.Sprintf(" %s", errInfo))
}

func TestParserCommandReplyForErrorNotComplete(t *testing.T) {
	errInfo := "this is a error"

	buf := buf.NewByteBuf(1024)
	buf.WriteByte('-')
	buf.WriteByte(' ')
	buf.Write([]byte(errInfo))
	checkNotComplete(t, buf)

	buf.Write(Delims)
	checkErrorReply(t, buf, fmt.Sprintf(" %s", errInfo))
}

func TestParserCommandReplyForInteger(t *testing.T) {
	var valueNumber int64
	valueNumber = 100
	value := fmt.Sprintf("%d", valueNumber)

	buf := buf.NewByteBuf(1024)
	WriteInteger(valueNumber, buf)

	checkIntegerReply(t, buf, value)
}

func TestParserCommandReplyForNotComplete(t *testing.T) {
	var valueNumber int64
	valueNumber = 100
	value := fmt.Sprintf("%d", valueNumber)

	buf := buf.NewByteBuf(1024)
	buf.WriteByte(':')
	buf.Write([]byte(value))
	checkNotComplete(t, buf)

	buf.Write(Delims)
	checkIntegerReply(t, buf, value)
}

func TestParserCommandReplyForBulk(t *testing.T) {
	buf := buf.NewByteBuf(1024)
	WriteBulk(nil, buf)
	checkBulkNilReply(t, buf)

	data := "this is a bulk data"
	WriteBulk([]byte(data), buf)
	checkBulkReply(t, buf, data)
}

func TestParserCommandReplyForBulkNotComplete(t *testing.T) {
	buf := buf.NewByteBuf(1024)
	buf.WriteByte('$')
	buf.Write(NullBulk)
	checkNotComplete(t, buf)

	buf.Write(Delims)
	checkBulkNilReply(t, buf)

	data := "this is a bulk data"
	buf.WriteByte('$')
	buf.Write(hack.StringToSlice(strconv.Itoa(len(data))))
	buf.Write(Delims)
	buf.Write([]byte(data))

	checkNotComplete(t, buf)

	buf.Write(Delims)
	checkBulkReply(t, buf, data)
}

func TestParserCommandReplyForArray(t *testing.T) {
	buf := buf.NewByteBuf(1024)
	status := "OK"
	errInfo := errors.New("this is a error")
	var integer int64
	integer = 100
	bulk := []byte("this is a bulk")

	lst := []interface{}{status, errInfo, integer, bulk}
	WriteArray(lst, buf)

	checkArrayReply(t, buf, len(lst))
}

func TestParserCommandReplyForArrayNotComplete(t *testing.T) {
	buf := buf.NewByteBuf(1024)
	status := "OK"
	errInfo := errors.New("this is a error")
	var integer int64
	integer = 100
	bulk := []byte("this is a bulk")

	lst := []interface{}{status, errInfo, integer, bulk}

	buf.WriteByte('*')
	buf.Write(hack.StringToSlice(strconv.Itoa(len(lst))))
	buf.Write(Delims)
	for i := 0; i < len(lst); i++ {
		switch v := lst[i].(type) {
		case []interface{}:
			WriteArray(v, buf)
		case [][]byte:
			WriteSliceArray(v, buf)
		case []byte:
			WriteBulk(v, buf)
		case nil:
			WriteBulk(nil, buf)
		case int64:
			WriteInteger(v, buf)
		case string:
			WriteStatus(hack.StringToSlice(v), buf)
		case error:
			WriteError(hack.StringToSlice(v.Error()), buf)
		default:
			panic(fmt.Sprintf("invalid array type %T %v", lst[i], v))
		}

		if i < len(lst)-1 {
			checkNotComplete(t, buf)
		}
	}

	checkArrayReply(t, buf, len(lst))
}

func checkNotComplete(t *testing.T, buf *buf.ByteBuf) {
	complete, _, err := readCommandReply(buf)
	assert.NoError(t, err)

	assert.False(t, complete)
}

func checkErrorReply(t *testing.T, buf *buf.ByteBuf, info string) {
	complete, value, err := readCommandReply(buf)
	assert.NoError(t, err)
	assert.True(t, complete)

	rsp, ok := value.(ErrResp)
	assert.True(t, ok)
	assert.Equal(t, string(rsp), info)
}

func checkStatusReply(t *testing.T, buf *buf.ByteBuf, info string) {
	complete, value, err := readCommandReply(buf)
	assert.NoError(t, err)
	assert.True(t, complete)

	rsp, ok := value.(StatusResp)
	assert.True(t, ok)
	assert.Equal(t, string(rsp), info)
}

func checkIntegerReply(t *testing.T, buf *buf.ByteBuf, num string) {
	complete, value, err := readCommandReply(buf)
	assert.NoError(t, err)
	assert.True(t, complete)

	rsp, ok := value.(IntegerResp)
	assert.True(t, ok)
	assert.Equal(t, string(rsp), num)
}

func checkArrayReply(t *testing.T, buf *buf.ByteBuf, num int) []interface{} {
	complete, value, err := readCommandReply(buf)
	assert.NoError(t, err)
	assert.True(t, complete)

	rsps, ok := value.([]interface{})
	assert.True(t, ok)
	assert.Equal(t, len(rsps), num)

	return rsps
}

func checkBulkNilReply(t *testing.T, buf *buf.ByteBuf) {
	complete, value, err := readCommandReply(buf)
	assert.NoError(t, err)
	assert.True(t, complete)

	_, ok := value.(NullBulkResp)
	assert.True(t, ok)
}

func checkBulkReply(t *testing.T, buf *buf.ByteBuf, data string) {
	complete, value, err := readCommandReply(buf)
	assert.NoError(t, err)
	assert.True(t, complete)

	rsp, ok := value.(BulkResp)
	assert.True(t, ok)
	assert.Equal(t, string(rsp), data)
}

func TestReadLine(t *testing.T) {
	buf := buf.NewByteBuf(1024)
	buf.Write([]byte("*3\r\n"))

	_, _, err := readLine(buf)
	if err != nil {
		t.Error("read line error")
	}
}
