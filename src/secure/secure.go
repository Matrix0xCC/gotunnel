package secure

import (
	"crypto/aes"
	"crypto/cipher"
	"io"
	"net"
	"connection"
)

type AESEncryptor struct {
	key    string
	block  cipher.Block
	stream cipher.Stream
}

func newAesEncryptor() *AESEncryptor {
	var encryptor = new(AESEncryptor)
	encryptor.key = "1234567812345678"
	encryptor.block, _ = aes.NewCipher([]byte(encryptor.key))
	iv := []byte("1234567812345678")
	encryptor.stream = cipher.NewCTR(encryptor.block, iv)
	return encryptor
}

type Tunnel struct {
	conn          io.ReadWriteCloser
	encryptor     *AESEncryptor
	decryptor     *AESEncryptor
	encryptWriter io.Writer
	decryptReader io.Reader
}

func NewSecureTunnel(c io.ReadWriteCloser) *Tunnel {
	var tunnel = new(Tunnel)
	tunnel.conn = c //
	tunnel.encryptor = newAesEncryptor()
	tunnel.decryptor = newAesEncryptor()
	tunnel.encryptWriter = cipher.StreamWriter{S: tunnel.encryptor.stream, W: tunnel.conn}
	tunnel.decryptReader = cipher.StreamReader{S: tunnel.decryptor.stream, R: tunnel.conn}

	return tunnel
}

func (tunnel *Tunnel) Read(p []byte) (int, error) {
	return tunnel.decryptReader.Read(p)
}

func (tunnel *Tunnel) Write(p []byte) (int, error) {
	return tunnel.encryptWriter.Write(p)
}

func (tunnel *Tunnel) Close() error {
	return tunnel.conn.Close()
}

type SimpleTunnelListener struct {
	connection.BaseListener
}

func (listener *SimpleTunnelListener) Accept() (io.ReadWriteCloser, error) {
	conn, err := listener.BaseListener.Accept()
	return NewSecureTunnel(conn), err
}

func NewTunnelFactory(target string) func() (io.ReadWriteCloser, error) {
	return func() (io.ReadWriteCloser, error) {
		server, err := net.Dial("tcp", target)
		if err != nil {
			return nil, err
		}
		return NewSecureTunnel(server), nil
	}
}