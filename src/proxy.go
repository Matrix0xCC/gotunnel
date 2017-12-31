package main

import (
	conn "connection"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"protocol"
	"secure"
)

func main() {

	config := initConfig()
	log.Printf("start using config: %+v", config)

	var listener, err = net.Listen("tcp", config.listen)
	if err != nil {
		log.Fatal(err)
	}

	pool := conn.NewTunnelManager(conn.TunnelFactory{tunnelFactory(config.connect)})
	proxy := &Proxy{config, pool}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
			continue
		}

		if config.mode == "client" {
			go handleConnInClientMode(conn, proxy)
		} else {
			go handleConnInServerMode(conn, proxy)
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
		client.Write([]byte{
			0x05,                   //version: 5
			0x04,                   //reply: 4, host cannot reach
			0x00,                   //reserved
			0x01,                   //addressType: ipv4
			0x00, 0x00, 0x00, 0x00, //ip address
			0x00, 0x00, // port in network order
		})
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

func handleConnInServerMode(client net.Conn, proxy *Proxy) {
	tunnel := secure.NewSecureTunnel(client)
	server, err := handleClientCommand(tunnel)
	if err != nil {
		log.Print(err)
		return
	}

	go forward(tunnel, server)
	forward(server, tunnel)
}

func handleConnInClientMode(browser net.Conn, proxy *Proxy) {
	log.Print("handle client connection begin")
	//handshake happens between browser and client
	handleHandShake(browser)

	tunnel, err := proxy.pool.Borrow()
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

type Config struct {
	mode    string
	listen  string
	connect string
}

type Proxy struct {
	config *Config
	pool   *conn.TunnelManager
}

func initConfig() *Config {
	var config = new(Config)

	flag.StringVar(&config.mode, "m", "client",
		"running mode. client vs server. default to client")

	flag.StringVar(&config.listen, "l", "127.0.0.1:9000",
		"listen address. used to specified listen address in client or server mode")

	flag.StringVar(&config.connect, "c", "127.0.0.1:9090",
		"connect address. only used in client mode")

	flag.Parse()

	if config.mode != "client" && config.mode != "server" {
		log.Fatalf("invalid mode %s. only client and server mode supported.", config.mode)
	}
	return config
}

func tunnelFactory(target string) func() (io.ReadWriteCloser, error) {
	return func() (io.ReadWriteCloser, error) {
		server, err := net.Dial("tcp", target)
		if err != nil {
			return nil, err
		}

		tunnel := secure.NewSecureTunnel(server)
		return tunnel, nil
	}
}
