package chassis

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"time"
)

type Config struct {
	Mode    string
	Listen  string
	Connect string
	Tls     tls.Config
	Secure  bool
}

type Listener interface {
	Listen(network string, addr string, config *Config) error
	Accept() (io.ReadWriteCloser, error)
}

type PlainListener struct {
	delegate net.Listener
}

type TlsTunnelListener struct {
	PlainListener
}

type Server interface {
	PrepareForward(conn io.ReadWriteCloser) (sec, plain io.ReadWriteCloser)
	MainLoop()
}

type BaseServer struct {
	Server
	Listener Listener
	Config   *Config
}

func (server *BaseServer) MainLoop() {
	err := server.Listener.Listen("tcp", server.Config.Listen, server.Config)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := server.Listener.Accept()
		if err != nil {
			log.Fatal(err)
			continue
		}

		ahead := func() {
			encryptedTun, plainTun := server.PrepareForward(conn)
			if encryptedTun == nil || plainTun == nil {
				log.Printf("create tunnel failed")
				_ = conn.Close()
				return
			}

			forward := func(to, from io.ReadWriteCloser) {
				io.Copy(to, from)
				defer from.Close()
				defer to.Close()
			}

			go forward(encryptedTun, plainTun)
			forward(plainTun, encryptedTun)
		}
		go ahead()
	}
}

func (listener *PlainListener) Listen(network string, addr string, config *Config) error {
	var err error
	listener.delegate, err = net.Listen(network, addr)
	return err
}

func (listener *PlainListener) Accept() (io.ReadWriteCloser, error) {
	conn, err := listener.delegate.Accept()
	setKeepAlive(conn.(*net.TCPConn))
	return conn, err
}

func (listener *TlsTunnelListener) Listen(network string, addr string, config *Config) error {
	var err error
	listener.delegate, err = tls.Listen(network, addr, &config.Tls)
	return err
}

func setKeepAlive(c *net.TCPConn) {
	_ = c.SetKeepAlive(true)
	_ = c.SetKeepAlivePeriod(30 * time.Second)
}
