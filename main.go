package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"io"
	"net"
	"strconv"
)

func handleConnection(conn net.Conn) {
	defer func() {
		err := conn.Close()
		if err != nil {
			fmt.Println(err.Error())
		}
	}()

	var b [1024]byte

	// 授权认证阶段
	n, err := conn.Read(b[:])
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if b[0] != 0x05 { // 只接受 Socks5
		return
	}

	_, err = conn.Write([]byte{0x05, 0x00}) // 无需认证
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// 建立连接阶段
	n, err = conn.Read(b[:])
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	var host string // DST.ADDR 目标服务器地址
	var port string // DST.PORT 目标服务器端口，最后两字节

	switch b[3] { // ATYP 目标服务器地址类型
	case 0x01: // IPv4
		host = net.IPv4(b[4], b[5], b[6], b[7]).String()
	case 0x03: // 域名
		host = string(b[5 : n-2])
	case 0x04: // IPv6
		host = net.IP{b[4], b[5], b[6], b[7], b[8], b[9], b[10], b[11],
			b[12], b[13], b[14], b[15], b[16], b[17], b[18], b[19]}.String()
	}
	port = strconv.Itoa(int(b[n-2])<<8 | int(b[n-1]))            // 位运算计算最后两字节的端口号
	server, err := net.Dial("tcp", net.JoinHostPort(host, port)) // 建立到目标服务器的连接
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer func() {
		err = server.Close()
		if err != nil {
			fmt.Println(err.Error())
		}
	}()

	_, err = conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	/*
		0x05 socks版本5
		0x00 状态：成功
		0x00 RSV
		0x01 0x00 0x00 0x00 0x00 0x00 0x00 目标服务器地址类型IPv4，IP为0.0.0.0，端口为0（这样方便且不影响使用）
	*/

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// 数据转发阶段，tcp直接转发
	go func() { // 从客户端到目标服务器
		_, err = io.Copy(server, conn)
		if err != nil {
			fmt.Println(err.Error())
		}
	}()
	_, err = io.Copy(conn, server) // 从目标服务器到客户端
	if err != nil {
		fmt.Println(err.Error())
	}
}

func main() {
	// 读取配置文件
	var config struct {
		Listen string
	}
	if _, err := toml.DecodeFile("./config.toml", &config); err != nil {
		fmt.Println(err.Error())
		panic(err)
	}

	// 开始监听
	server, err := net.Listen("tcp", ":1080")

	if err == nil {
		fmt.Printf("Server running at %s\n", config.Listen)
	} else {
		fmt.Println(err.Error())
		panic(err)
	}

	defer func() {
		err = server.Close()
		if err != nil {
			fmt.Println(err.Error())
			panic(err)
		}
	}()

	for {
		conn, err := server.Accept()
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		go handleConnection(conn)
	}
}
