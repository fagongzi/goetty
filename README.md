goetty
-------
Goetty is a framework to help you build socket application.

Example
--------
codec
```
package example

import (
    "github.com/fagongzi/goetty"
)

type StringDecoder struct {
}

func (decoder StringDecoder) Decode(in *ByteBuf) (bool, interface{}, error) {
    _, data, err := in.ReadMarkedBytes()

    if err != nil {
        return true, "", err
    }

    return true, string(data), nil
}

type StringEncoder struct {
}

func (self StringEncoder) Encode(data interface{}, out *ByteBuf) error {
    msg, _ := data.(string)
    bytes := []byte(msg)
    out.WriteInt(len(bytes))
    out.Write(bytes)
    return nil
}
```

server
```
package example

import (
    "fmt"
    "github.com/fagongzi/goetty"
)

type EchoServer struct {
    addr   string
    server *goetty.Server
}

func NewEchoServer(addr string) *EchoServer {
    return &EchoServer{
        addr:   addr,
        server: goetty.NewServer(addr, NewIntLengthFieldBasedDecoder(&StringDecoder{}), &StringEncoder{}, NewInt64IdGenerator()),
    }
}

func (self *EchoServer) Serve() error {
    return self.server.Serve(loopFn)
}

func (self *EchoServer) doConnection(session goetty.IOSession) {
    defer session.Close() // close the connection

    fmt.Printf("A new connection from <%s>", session.RemoteAddr())

    // start loop for read msg from this connection
    for {
        msg, err := session.Read() // if you want set a read deadline, you can use 'session.ReadTimeout(timeout)'
        if err != nil {
            return err
        }

        fmt.Printf("receive a msg<%s> from <%s>", msg, session.RemoteAddr())

        // echo msg back
        session.Write(msg)
    }
}
```

client
```
package example

import (
    "github.com/fagongzi/goetty"
)

type EchoClient struct {
    serverAddr string
    conn       *goetty.Connector
}

func NewEchoClient(serverAddr string) (*EchoClient, error) {
    cnf := &Conf{
        Addr: serverAddr,
        TimeoutConnectToServer: time.Second * 3,
    }

    c := &EchoClient{
        serverAddr: serverAddr,
        conn:       NewConnector(cnf, NewIntLengthFieldBasedDecoder(&StringDecoder{}), &StringEncoder{}),
    }

    // if you want to send heartbeat to server, you can set conf as below, otherwise not set

    // create a timewheel to calc timeout
    tw := NewHashedTimeWheel(time.Second, 60, 3)
    tw.Start()

    cnf.TimeoutWrite = time.Second * 3
    cnf.TimeWheel = tw
    cnf.WriteTimeoutFn = c.writeHeartbeat

    _, err := c.conn.Connect()

    return c, err
}

func (self *EchoClient) writeHeartbeat(serverAddr string, conn *goetty.Connector) {
    self.SendMsg("this is a heartbeat msg")
}

func (self *EchoClient) SendMsg(msg string) error {
    return self.conn.Write(msg)
}

func (self *EchoClient) ReadLoop() error {
    // start loop to read msg from server
    for {
        msg, err := connector.Read() // if you want set a read deadline, you can use 'connector.ReadTimeout(timeout)'
        if err != nil {
            fmt.Printf("read msg from server<%s> failure", self.serverAddr)
            return err
        }

        fmt.Printf("receive a msg<%s> from <%s>", msg, self.serverAddr)
    }

    return nil
}

```

