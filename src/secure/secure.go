package secure

import (
	"crypto/aes"
	"crypto/cipher"
	"io"
	"net"
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
	conn          net.Conn
	encryptor     *AESEncryptor
	decryptor     *AESEncryptor
	encryptWriter io.Writer
	decryptReader io.Reader
}

func NewSecureTunnel(c net.Conn) *Tunnel {
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
