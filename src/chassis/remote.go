package chassis

import (
	"fmt"
	"github.com/Matrix0xCC/gotunnel/protocol"
	"github.com/Matrix0xCC/gotunnel/secure"
	"io"
	"log"
	"net"
)

type RemoteSocksServer struct {
	BaseServer
}

func (remoteSocksServer *RemoteSocksServer) PrepareForward(connection io.ReadWriteCloser) (s, p io.ReadWriteCloser) {
	tunnel, err := secure.NewServerTunnel(connection)
	if err != nil {
		log.Print(err)
		return nil, nil
	}

	//we just skipped the ClientHello between LocalSocksServer and RemoteSocksServer to speed up
	targetServer, err := remoteSocksServer.handleClientCommand(tunnel)
	if err != nil {
		log.Print(err)
		return nil, nil
	}
	return tunnel, targetServer
}

func (remoteSocksServer *RemoteSocksServer) handleClientCommand(client io.ReadWriteCloser) (net.Conn, error) {
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
		_, _ = client.Write([]byte{0x05, 0x04, 0x00, 0x01,
			//ip address ,          port in network order
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		return nil, fmt.Errorf("failed to connect %s, caused by %s", target, err)
	}

	var resp = []byte{0x05, 0x00, 0x00, 0x01} //version, reply, reserved, server_address_type

	ipByte := server.LocalAddr().(*net.TCPAddr).IP.To4()
	port := server.LocalAddr().(*net.TCPAddr).Port
	resp = append(resp, ipByte...)
	resp = append(resp, byte(port>>8), byte(port&0xFF))

	if _, err = client.Write(resp); err != nil {
		return nil, err
	}

	return server, nil
}

func NewRemoteServer(config *Config) *RemoteSocksServer {
	server := &RemoteSocksServer{BaseServer{Config: config}}
	server.Server = server
	if config.Secure {
		server.Listener = &TlsTunnelListener{}
	} else {
		server.Listener = &PlainListener{}
	}

	return server
}
