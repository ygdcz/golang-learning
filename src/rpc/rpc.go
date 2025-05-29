package rpc

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"

	pb "github.com/ygdcz/golang-learning/src/rpc/proto" // 使用 pb 作为别名
)

type HelloService struct{}

func (p *HelloService) Hello(request *pb.String, reply *string) error {
	*reply = "Hello, " + request.GetValue()
	return nil
}

func startServer() {
	rpc.RegisterName("HelloService", new(HelloService))

	listener, err := net.Listen("tcp", ":1234")
	if err != nil {
		log.Fatal("Listen TCP error:", err)
	}
	defer listener.Close()

	log.Println("Server is running on port 1234...")
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}
		go rpc.ServeCodec(jsonrpc.NewServerCodec(conn))
	}
}

func startClient() {
	time.Sleep(1 * time.Second) // 等待服务端启动

	conn, err := net.Dial("tcp", "localhost:1234")
	if err != nil {
		log.Fatal("Dialing error:", err)
	}
	defer conn.Close()

	client := rpc.NewClientWithCodec(jsonrpc.NewClientCodec(conn))

	var reply string
	err = client.Call("HelloService.Hello", "World", &reply)
	if err != nil {
		log.Fatal("RPC call error:", err)
	}

	fmt.Println(reply) // 输出: Hello, World
}

func RPC() {
	// 启动服务端
	// go startServer()

	// 启动客户端
	// startClient()

	// 注册服务
	rpc.RegisterName("HelloService", new(HelloService))

	// 设置 JSON-RPC 的 HTTP 处理器
	http.HandleFunc("/jsonrpc", func(w http.ResponseWriter, r *http.Request) {
		var conn io.ReadWriteCloser = struct {
			io.Writer
			io.ReadCloser
		}{
			ReadCloser: r.Body,
			Writer:     w,
		}

		rpc.ServeRequest(jsonrpc.NewServerCodec(conn))
	})

	// 启动 HTTP 服务
	log.Println("HTTP server is running on port 1234...")
	http.ListenAndServe(":1234", nil)
}
