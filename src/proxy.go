package main

import (
	"chassis"
	"fmt"
	"io"
	"log"
	"net"
	"protocol"
)

func main() {
	proxy := chassis.InitProxy()
	log.Printf("start using config: %+v", proxy.Config)

	listener, err := proxy.Listen()
	if err != nil {
		log.Fatal(err)
	}

	for {
		client, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
			continue
		}

		if proxy.Config.Mode == "client" {
			go handleConnInClientMode(client, proxy)
		} else {
			go handleConnInServerMode(client, proxy)
		}
	}
}

func handleHandShake(c io.ReadWriter) error {
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
	log.Printf("%+v", clientHello)

	c.Write([]byte{0x05, 0x00}) // ServerHello{version:0x05, method: 0x00}, means no authentication required

	return nil
}

func handleClientCommand(client io.ReadWriter) (net.Conn, error) {
	buffer := make([]byte, 1024)
	count, err := client.Read(buffer)
	if err != nil {
		return nil, err
	}

	log.Print(buffer[:count])

	clientCommand, err := proto.DecodeClientCommand(buffer[:count])
	if err != nil {
		return nil, err
	}

	if clientCommand.Command != 1 { //1: connect 2:bind 3. udp associate
		return nil, fmt.Errorf("unsupported command")
	}

	log.Printf("%+v", clientCommand)

	target := fmt.Sprintf("%s:%d", clientCommand.DestAddr, clientCommand.Port)
	server, err := net.Dial("tcp", target)
	if err != nil {
		//version: 5, reply: 4, host cannot reach, reserved: 0, addressType: ipv4
		client.Write([]byte{ 0x05, 0x04, 0x00, 0x01,
			//ip address ,          port in network order
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return nil, fmt.Errorf("failed to connect %s, caused by %s", target, err)
	}

	var resp = []byte{0x05, 0x00, 0x00, 0x01} //version, reply, reserved, server_address_type

	ipByte := server.LocalAddr().(*net.TCPAddr).IP.To4()
	port := server.LocalAddr().(*net.TCPAddr).Port
	resp = append(resp, ipByte...)
	resp = append(resp, byte(port>>8), byte(port&0xFF))

	client.Write(resp)

	return server, nil
}

func handleConnInServerMode(tunnel io.ReadWriteCloser, proxy *chassis.Proxy) {
	server, err := handleClientCommand(tunnel)
	if err != nil {
		log.Print(err)
		return
	}

	go forward(tunnel, server)
	forward(server, tunnel)
}

func handleConnInClientMode(browser io.ReadWriteCloser, proxy *chassis.Proxy) {
	log.Print("handle client connection begin")
	//handshake happens between browser and client
	handleHandShake(browser)

	tunnel, err := proxy.Pool.Borrow()
	if err != nil {
		log.Print("failed to create tunnel")
		return
	}

	go forward(tunnel, browser)
	forward(browser, tunnel)
}

func forward(to, from io.ReadWriteCloser) {
	io.Copy(to, from)
	defer from.Close()
	defer to.Close()
}
