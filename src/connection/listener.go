package connection

import (
	"crypto/tls"
	"io"
	"net"
)

type Listener interface {
	Listen(network string, addr string, config interface{}) (error)
	Accept() (io.ReadWriteCloser, error)
}

type BaseListener struct {
	delegate net.Listener
}

type TlsTunnelListener struct {
	BaseListener
}

func (listener *BaseListener) Listen(network string, addr string, config interface{}) (error) {
	var err error
	listener.delegate, err = net.Listen(network, addr)
	return err
}

func (listener *BaseListener) Accept() (io.ReadWriteCloser, error) {
	return listener.delegate.Accept()
}

func (listener *TlsTunnelListener) Listen(network string, addr string, config interface{}) (error) {
	var err error
	listener.delegate, err = tls.Listen(network, addr, config.(*tls.Config))
	return err
}

