package main

import (
	"io"
	"log"
	"net"
	"fmt"
	"strings"
	"strconv"
	"os"
	"secure"
	"protocol"
)

func main() {
	var mode = "client"
	var port = "9000"

	if os.Args != nil && len(os.Args) >= 2 && os.Args[1] == "server" {
		mode = "server"
		port = "9090"
	}

	log.Printf("start in %s mode", mode)

	listener, err := net.Listen("tcp", "localhost:"+port)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
			continue
		}

		if mode == "client" {
			go handleConnInClientMode(conn)
		} else {
			go handleConnInServerMode(conn)
		}
	}
}

func doHandShake(c io.ReadWriter) error {
	buffer := make([]byte, 256)

	var count, err = c.Read(buffer)
	if err != nil {
		log.Print(err)
		return err
	}

	clientHello, err := proto.DecodeClientHello(buffer[:count])
	if err != nil {
		log.Print(err)
		return err
	}
	log.Print(clientHello)

	c.Write([]byte{0x05, 0x00}) // ServerHello{version:0x05, method: 0x00} // no authentication required

	return nil
}

func handleConnInServerMode(c net.Conn) {
	tunnel := secure.NewSecureTunnel(c)
	err := doHandShake(tunnel)
	if err != nil {
		log.Print("handshake failed")
		return
	}

	buffer := make([]byte, 256)
	count, err := tunnel.Read(buffer)
	if err != nil {
		log.Print(err)
		return
	}
	clientCommand, err := proto.DecodeClientCommand(buffer[:count])
	if err != nil {
		log.Print(err)
		return
	}

	if clientCommand.Command != 1 { //1: connect 2:bind 3. udp associate
		log.Print("unsupported command!")
		return
	}
	log.Print(fmt.Sprintf("%+v", clientCommand))

	target := fmt.Sprintf("%s:%d", clientCommand.DestAddr, clientCommand.Port)

	server, err := net.Dial("tcp", target)
	if err != nil {
		tunnel.Write([]byte{
			0x05,                   //version: 5
			0x04,                   //reply: 4, host cannot reach
			0x00,                   //reserved
			0x01,                   //addressType: ipv4
			0x00, 0x00, 0x00, 0x00, //ip address
			0x00, 0x00,             // port in network order
		})

		log.Print(err)
		return
	}

	var resp = []byte{0x05, 0x00, 0x00, 0x01} //version, reply, reserved, server_address_type
	resp = append(resp, tcpAddrToByteArray(server.LocalAddr())...)
	port := server.LocalAddr().(*net.TCPAddr).Port
	resp = append(resp, byte(port>>8), byte(port&0xFF))
	tunnel.Write(resp)

	go io.Copy(tunnel, server)
	io.Copy(server, tunnel)

}

func handleConnInClientMode(c net.Conn) {
	defer c.Close()
	server, err := net.Dial("tcp", "127.0.0.1:9090")
	if err != nil {
		log.Print("connect to server failed")
		return
	}

	tunnel := secure.NewSecureTunnel(server)
	go io.Copy(tunnel, c)
	io.Copy(c, tunnel)
}

func tcpAddrToByteArray(addr net.Addr) [] byte {
	var ip = strings.Split(addr.(*net.TCPAddr).IP.String(), ".")

	b1, _ := strconv.Atoi(ip[0])
	b2, _ := strconv.Atoi(ip[1])
	b3, _ := strconv.Atoi(ip[2])
	b4, _ := strconv.Atoi(ip[3])

	return []byte{byte(b1), byte(b2), byte(b3), byte(b4)}
}
