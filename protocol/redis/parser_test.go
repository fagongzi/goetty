package redis

import (
	"errors"
	"strconv"
	"testing"

	"fmt"

	"github.com/fagongzi/goetty"
)

func TestParserCommandRespForStatus(t *testing.T) {
	status := "OK"

	buf := goetty.NewByteBuf(1024)
	WriteStatus([]byte("OK"), buf)

	checkStatusResp(buf, t, status)
}

func TestParserCommandRespForStatusNotComplete(t *testing.T) {
	status := "OK"

	buf := goetty.NewByteBuf(1024)
	buf.WriteByte('+')
	buf.Write([]byte(status))

	checkNotComplete(buf, t)

	buf.Write(Delims)

	checkStatusResp(buf, t, status)
}

func TestParserCommandRespForError(t *testing.T) {
	errInfo := "this is a error"

	buf := goetty.NewByteBuf(1024)
	WriteError([]byte(errInfo), buf)

	checkErrorResp(buf, t, fmt.Sprintf(" %s", errInfo))
}

func TestParserCommandRespForErrorNotComplete(t *testing.T) {
	errInfo := "this is a error"

	buf := goetty.NewByteBuf(1024)
	buf.WriteByte('-')
	buf.WriteByte(' ')
	buf.Write([]byte(errInfo))
	checkNotComplete(buf, t)

	buf.Write(Delims)
	checkErrorResp(buf, t, fmt.Sprintf(" %s", errInfo))
}

func TestParserCommandRespForInteger(t *testing.T) {
	var valueNumber int64
	valueNumber = 100
	value := fmt.Sprintf("%d", valueNumber)

	buf := goetty.NewByteBuf(1024)
	WriteInteger(valueNumber, buf)

	checkIntegerResp(buf, t, value)
}

func TestParserCommandRespForNotComplete(t *testing.T) {
	var valueNumber int64
	valueNumber = 100
	value := fmt.Sprintf("%d", valueNumber)

	buf := goetty.NewByteBuf(1024)
	buf.WriteByte(':')
	buf.Write([]byte(value))
	checkNotComplete(buf, t)

	buf.Write(Delims)
	checkIntegerResp(buf, t, value)
}

func TestParserCommandRespForBulk(t *testing.T) {
	buf := goetty.NewByteBuf(1024)
	WriteBulk(nil, buf)
	checkBulkNilResp(buf, t)

	data := "this is a bulk data"
	WriteBulk([]byte(data), buf)
	checkBulkResp(buf, t, data)
}

func TestParserCommandRespForBulkNotComplete(t *testing.T) {
	buf := goetty.NewByteBuf(1024)
	buf.WriteByte('$')
	buf.Write(NullBulk)
	checkNotComplete(buf, t)

	buf.Write(Delims)
	checkBulkNilResp(buf, t)

	data := "this is a bulk data"
	buf.WriteByte('$')
	buf.Write(goetty.StringToSlice(strconv.Itoa(len(data))))
	buf.Write(Delims)
	buf.Write([]byte(data))

	checkNotComplete(buf, t)

	buf.Write(Delims)
	checkBulkResp(buf, t, data)
}

func TestParserCommandRespForArray(t *testing.T) {
	buf := goetty.NewByteBuf(1024)
	status := "OK"
	errInfo := errors.New("this is a error")
	var integer int64
	integer = 100
	bulk := []byte("this is a bulk")

	lst := []interface{}{status, errInfo, integer, bulk}
	WriteArray(lst, buf)

	checkArrayResp(buf, t, len(lst))
}

func TestParserCommandRespForArrayNotComplete(t *testing.T) {
	buf := goetty.NewByteBuf(1024)
	status := "OK"
	errInfo := errors.New("this is a error")
	var integer int64
	integer = 100
	bulk := []byte("this is a bulk")

	lst := []interface{}{status, errInfo, integer, bulk}

	buf.WriteByte('*')
	buf.Write(goetty.StringToSlice(strconv.Itoa(len(lst))))
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
			WriteStatus(goetty.StringToSlice(v), buf)
		case error:
			WriteError(goetty.StringToSlice(v.Error()), buf)
		default:
			panic(fmt.Sprintf("invalid array type %T %v", lst[i], v))
		}

		if i < len(lst)-1 {
			checkNotComplete(buf, t)
		}
	}

	checkArrayResp(buf, t, len(lst))
}

func checkNotComplete(buf *goetty.ByteBuf, t *testing.T) {
	complete, _, err := readCommandResp(buf)
	if err != nil {
		t.Failed()
	}

	if complete {
		t.Failed()
	}
}

func checkErrorResp(buf *goetty.ByteBuf, t *testing.T, info string) {
	complete, value, err := readCommandResp(buf)
	if err != nil {
		t.Error(err)
	}

	if !complete {
		t.Error("not complete")
	}

	rsp, ok := value.(ErrResp)
	if !ok {
		t.Error("type mis match")
	}

	fmt.Printf("%s\n", []byte(rsp))
	if string(rsp) != info {
		t.Error("value mis match")
	}
}

func checkStatusResp(buf *goetty.ByteBuf, t *testing.T, info string) {
	complete, value, err := readCommandResp(buf)
	if err != nil {
		t.Error(err)
	}

	if !complete {
		t.Error("not complete")
	}

	rsp, ok := value.(StatusResp)
	if !ok {
		t.Error("type mis match")
	}

	if string(rsp) != info {
		t.Error("value mis match")
	}
}

func checkIntegerResp(buf *goetty.ByteBuf, t *testing.T, num string) {
	complete, value, err := readCommandResp(buf)
	if err != nil {
		t.Error(err)
	}

	if !complete {
		t.Error("not complete")
	}

	rsp, ok := value.(IntegerResp)
	if !ok {
		t.Error("type mis match")
	}

	if string(rsp) != num {
		t.Error("value mis match")
	}
}

func checkArrayResp(buf *goetty.ByteBuf, t *testing.T, num int) []interface{} {
	complete, value, err := readCommandResp(buf)
	if err != nil {
		t.Error(err)
	}

	if !complete {
		t.Error("not complete")
	}

	rsps, ok := value.([]interface{})
	if !ok {
		t.Error("type mis match")
	}

	if len(rsps) != num {
		t.Error("values mis match")
	}

	return rsps
}

func checkBulkNilResp(buf *goetty.ByteBuf, t *testing.T) {
	complete, value, err := readCommandResp(buf)
	if err != nil {
		t.Error(err)
	}

	if !complete {
		t.Error("not complete")
	}

	_, ok := value.(NullBulkResp)
	if !ok {
		t.Error("type mis match")
	}
}

func checkBulkResp(buf *goetty.ByteBuf, t *testing.T, data string) {
	complete, value, err := readCommandResp(buf)
	if err != nil {
		t.Error(err)
	}

	if !complete {
		t.Error("not complete")
	}

	rsp, ok := value.(BulkResp)
	if !ok {
		t.Error("type mis match")
	}

	if string(rsp) != data {
		t.Error("value mis match")
	}
}
