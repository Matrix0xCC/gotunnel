package connection

import (
	"io"
	"log"
	"time"
)

type TunnelFactory func() (io.ReadWriteCloser, error)

type TunnelPool struct {
	pool              chan io.ReadWriteCloser
	coreSize          int
	connectionFactory TunnelFactory
}

func NewTunnelPool(factory TunnelFactory) *TunnelPool {
	pool := new(TunnelPool)
	pool.coreSize = 20
	pool.pool = make(chan io.ReadWriteCloser, pool.coreSize)
	pool.connectionFactory = factory

	go pool.createTunnels()

	return pool
}

func (manager *TunnelPool) createTunnels() {
	for {
		tunnel, err := manager.connectionFactory()
		if err != nil {
			log.Print("create tunnel failed, caused by: ", err)
			time.Sleep(3 * time.Second)
			continue
		}
		manager.pool <- tunnel
	}
}

func (manager *TunnelPool) Borrow() (io.ReadWriteCloser, error) {
	return <-manager.pool, nil
}
