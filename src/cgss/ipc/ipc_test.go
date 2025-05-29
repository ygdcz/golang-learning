package ipc

import "testing"

type EchoServer struct {
}

func (server *EchoServer) Handle(method, params string) *Response {
	return &Response{"200", "hello from echo server"}
}

func (server *EchoServer) Name() string {
	return "EchoServer"
}

func TestIpc(t *testing.T) {
	server := NewIpcServer(&EchoServer{})
	client1 := NewIpcClient(server)
	client2 := NewIpcClient(server)
	resp1, err1 := client1.Call("From Client1", "connect request from client1")
	resp2, err2 := client2.Call("From Client2", "connect request from client2")
	if resp1.Body != "hello from echo server" || resp2.Body != "hello from echo server" || err1 != nil || err2 != nil {
		t.Error("IpcClient.Call failed. resp1:", resp1.Body, "resp2:", resp2.Body)
	}

	client1.Close()
	client2.Close()
}
