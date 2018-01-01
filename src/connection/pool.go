package connection

import (
	"io"
	"log"
	"time"
)

type TunnelFactory struct {
	NewTunnel func() (io.ReadWriteCloser, error)
}

type TunnelManager struct {
	coreSize int
	pool     chan io.ReadWriteCloser
	factory  TunnelFactory
}

func NewTunnelManager(factory TunnelFactory) *TunnelManager {
	manager := new(TunnelManager)
	manager.coreSize = 20
	manager.pool = make(chan io.ReadWriteCloser, manager.coreSize)
	manager.factory = factory

	go manager.createTunnels()

	return manager
}

func (manager *TunnelManager) createTunnels() {
	for {
		tunnel, err := manager.factory.NewTunnel()
		if err != nil {
			log.Print("create tunnel failed, caused by: ", err)
			time.Sleep(3 * time.Second)
			continue
		}
		manager.pool <- tunnel
	}
}

func (manager *TunnelManager) Borrow() (io.ReadWriteCloser, error) {
	return <-manager.pool, nil
}

func (manager *TunnelManager) Return(tunnel io.ReadWriteCloser) error {
	manager.pool <- tunnel
	return nil
}

