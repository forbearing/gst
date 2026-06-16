package tunnel_test

import (
	"net"
	"testing"

	"github.com/forbearing/gst/bootstrap"
	"github.com/forbearing/gst/pkg/tunnel"
	"github.com/forbearing/gst/types/consts"
	"github.com/stretchr/testify/assert"
)

var (
	addr   = "0.0.0.0:12345"
	doneCh = make(chan struct{}, 1)

	Bye   = tunnel.NewCmd("bye", 1000)
	Hello = tunnel.NewCmd("hello", 1001)
)

type ByePaylod struct {
	Field1 string
	Field2 uint64
}
type HelloPaylod struct {
	Field3 string
	Field4 float64
}

var (
	byePayload1   = ByePaylod{Field1: "bye1", Field2: 123}
	byePayload2   = ByePaylod{Field1: "bye2", Field2: 456}
	helloPayload1 = HelloPaylod{Field3: "hello1", Field4: 3.14}
	helloPayload2 = HelloPaylod{Field3: "hello2", Field4: 3.14}
)

func TestSession(t *testing.T) {
	assert.NoError(t, bootstrap.Bootstrap())
	go server(t)
	client(t)
	<-doneCh
}

func server(t *testing.T) {
	t.Helper()
	l, err := net.Listen("tcp", addr)
	assert.NoError(t, err)
	defer l.Close()

	doneCh <- struct{}{}

	conn, err := l.Accept()
	assert.NoError(t, err)
	defer conn.Close()

	session, _ := tunnel.NewSession(conn, consts.Server)
	for {
		event, err := session.Read()
		assert.NoError(t, err)
		switch event.Cmd {
		case tunnel.Ping:
			t.Log("client ping")
			_ = session.Write(&tunnel.Event{Cmd: tunnel.Pong})
		case Hello:
			payload := new(HelloPaylod)
			assert.NoError(t, tunnel.DecodePayload(event.Payload, payload))
			assert.Equal(t, helloPayload1, *payload)
			t.Logf("client hello: %+v\n", *payload)
			_ = session.Write(&tunnel.Event{Cmd: Hello, Payload: helloPayload2})
		case Bye:
			payload := new(ByePaylod)
			assert.NoError(t, tunnel.DecodePayload(event.Payload, payload))
			assert.Equal(t, byePayload1, *payload)
			t.Logf("client bye: %+v\n", *payload)
			_ = session.Write(&tunnel.Event{Cmd: Bye, Payload: byePayload2})
		}
	}
}

func client(t *testing.T) {
	t.Helper()
	<-doneCh

	conn, err := net.Dial("tcp", addr)
	assert.NoError(t, err)
	defer conn.Close()

	session, _ := tunnel.NewSession(conn, consts.Client)
	_ = session.Write(&tunnel.Event{Cmd: tunnel.Ping})

	for {
		event, err := session.Read()
		assert.NoError(t, err)

		switch event.Cmd {
		case tunnel.Pong:
			t.Log("server pong")
			_ = session.Write(&tunnel.Event{Cmd: Hello, Payload: helloPayload1})
		case Hello:
			payload := new(HelloPaylod)
			assert.NoError(t, tunnel.DecodePayload(event.Payload, payload))
			assert.Equal(t, helloPayload2, *payload)
			t.Logf("server hello: %+v\n", *payload)
			_ = session.Write(&tunnel.Event{Cmd: Bye, Payload: byePayload1})
		case Bye:
			payload := new(ByePaylod)
			assert.NoError(t, tunnel.DecodePayload(event.Payload, payload))
			assert.Equal(t, byePayload2, *payload)
			t.Logf("server bye: %+v\n", *payload)
			doneCh <- struct{}{}
			return
		}
	}
}
