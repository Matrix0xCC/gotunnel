package chassis

import (
	"crypto/tls"
	"flag"
	"log"
	"io"
	"secure"
	conn "connection"
)

type Config struct {
	Mode    string
	Listen  string
	Connect string
	Tls     tls.Config
	Secure  bool
}

type Proxy struct {
	Config   *Config
	Pool     *conn.TunnelManager
	Listener conn.Listener
}

func (proxy *Proxy) Listen() (conn.Listener, error) {
	err := proxy.Listener.Listen("tcp", proxy.Config.Listen, proxy.Config.Tls)
	return proxy.Listener, err
}

func InitProxy() *Proxy {
	var config = new(Config)

	flag.StringVar(&config.Mode, "m", "client",
		"running Mode. client vs server. default to client")

	flag.StringVar(&config.Listen, "l", "127.0.0.1:9000",
		"Listen address. used to specified Listen address in client or server Mode")

	flag.StringVar(&config.Connect, "c", "127.0.0.1:9090",
		"Connect address. only used in client Mode")

	flag.BoolVar(&config.Secure, "s", false,
		"use tls between client and server")

	flag.Parse()

	if config.Mode != "client" && config.Mode != "server" {
		log.Fatalf("invalid Mode %s. only client and server Mode supported.", config.Mode)
	}

	var proxy = Proxy{Config: config}

	if config.Mode == "client" {
		proxy.Listener = &conn.BaseListener{}
	}

	if config.Secure {
		if config.Mode == "client" {
			factory := func() (io.ReadWriteCloser, error) {
				return tls.Dial("tcp", config.Connect, &proxy.Config.Tls)
			}
			proxy.Pool = conn.NewTunnelManager(conn.TunnelFactory{NewTunnel: factory})
		} else {
			proxy.Listener = &conn.TlsTunnelListener{}
		}
	} else {
		if config.Mode == "client" {
			proxy.Pool = conn.NewTunnelManager(conn.TunnelFactory{NewTunnel: secure.NewTunnelFactory(config.Connect)})
		} else {
			proxy.Listener = &secure.SimpleTunnelListener{}
		}
	}

	return &proxy
}
