package chassis

import (
	"crypto/tls"
	"github.com/Matrix0xCC/gotunnel/connection"
	"github.com/Matrix0xCC/gotunnel/protocol"
	"github.com/Matrix0xCC/gotunnel/secure"
	"io"
	"log"
	"net"
)

type LocalSocksServer struct {
	BaseServer
	pool *connection.TunnelPool
}

func (local *LocalSocksServer) PrepareForward(browser io.ReadWriteCloser) (s, p io.ReadWriteCloser) {
	log.Print("handle client connection begin")
	if err := local.handleClientHello(browser); err != nil {
		log.Print("handshake failed: ", err)
		return nil, nil
	}

	tunnel, _ := local.pool.Borrow()
	return tunnel, browser
}

/**
 * ClientHello only happens between browser and LocalSocksServer
 */
func (local *LocalSocksServer) handleClientHello(c io.ReadWriter) error {
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
	_, err = c.Write([]byte{0x05, 0x00}) // ServerHello{version:0x05, method: 0x00}, means no authentication required
	return err
}

func NewLocalServer(config *Config) *LocalSocksServer {
	server := &LocalSocksServer{BaseServer{Listener: &PlainListener{}, Config: config}, nil}
	server.Server = server

	if config.Secure {
		tlsConnector := func() (io.ReadWriteCloser, error) {
			return tls.Dial("tcp", config.Connect, &config.Tls)
		}
		server.pool = connection.NewTunnelPool(tlsConnector)
	} else {
		plainConnector := func() (io.ReadWriteCloser, error) {
			server, err := net.Dial("tcp", config.Connect)
			if err != nil {
				return nil, err
			}
			return secure.NewClientTunnel(server)
		}

		server.pool = connection.NewTunnelPool(plainConnector)
	}

	return server
}
