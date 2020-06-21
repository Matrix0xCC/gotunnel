package secure

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
	"log"
	"math/rand"
	"time"
)

const AesKeyLen = 128 / 8

type aesEncryptCtx struct {
	iv          []byte
	key         []byte
	encStream   cipher.Stream
	decStream   cipher.Stream
	negotiation []byte
}

/**
 * simple key negotiation using client generated key and iv.
 * better than hard coding but still not safe and not recommended.
 */
func encryptCtxFromRandom() *aesEncryptCtx {
	noisyByteLen := randomBytes(1)[0]
	negotiation := randomBytes(AesKeyLen*2 + 1 + int(noisyByteLen))
	negotiation[AesKeyLen*2] = noisyByteLen
	return newCtxFromBytes(negotiation, AesKeyLen*2+1+int(noisyByteLen))
}

func readEncryptCtx(connection io.ReadWriteCloser) (*aesEncryptCtx, error) {
	negotiation := make([]byte, 512)

	recvLen, err := connection.Read(negotiation)
	if err != nil {
		return nil, err
	}
	expectedLen := int(negotiation[AesKeyLen*2]) + AesKeyLen*2 + 1
	if recvLen < AesKeyLen*2+1 || recvLen != expectedLen { //32 <= len < 256 + 32
		return nil, fmt.Errorf("negotiation bytes may be segmented due to small MTU")
	}
	return newCtxFromBytes(negotiation, recvLen), nil
}

func newCtxFromBytes(buffer []byte, len int) *aesEncryptCtx {
	var encryptCtx = new(aesEncryptCtx)
	encryptCtx.key = buffer[0:AesKeyLen]
	encryptCtx.iv = buffer[AesKeyLen : AesKeyLen+AesKeyLen]
	encryptCtx.negotiation = buffer[0:len]

	encBlock, _ := aes.NewCipher([]byte(encryptCtx.key))
	encryptCtx.encStream = cipher.NewCTR(encBlock, encryptCtx.iv)
	decBlock, _ := aes.NewCipher([]byte(encryptCtx.key))
	encryptCtx.decStream = cipher.NewCTR(decBlock, encryptCtx.iv)

	log.Printf("iv: %+v, key: %v", encryptCtx.key, encryptCtx.iv)
	return encryptCtx
}

func randomBytes(len int) []byte {
	token := make([]byte, len)
	now := time.Now().UnixNano()
	random := rand.New(rand.NewSource(now).(rand.Source64))
	random.Read(token)
	return token
}

type Tunnel struct {
	conn          io.ReadWriteCloser
	encryptCtx    *aesEncryptCtx
	encryptWriter io.Writer
	decryptReader io.Reader
}

func newTunnel(encryptCtx *aesEncryptCtx, connection io.ReadWriteCloser) *Tunnel {
	var tunnel = new(Tunnel)
	tunnel.encryptWriter = cipher.StreamWriter{S: encryptCtx.encStream, W: connection}
	tunnel.decryptReader = cipher.StreamReader{S: encryptCtx.decStream, R: connection}
	tunnel.conn = connection //
	tunnel.encryptCtx = encryptCtx

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

func NewClientTunnel(connection io.ReadWriteCloser) (*Tunnel, error) {
	encryptCtx := encryptCtxFromRandom()

	if _, err := connection.Write(encryptCtx.negotiation); err != nil {
		_ = connection.Close()
		return nil, err
	}

	log.Printf("send negotiation to server %+v, %+v", encryptCtx.key, encryptCtx.iv)
	return newTunnel(encryptCtx, connection), nil
}

func NewServerTunnel(connection io.ReadWriteCloser) (*Tunnel, error) {
	encryptCtx, err := readEncryptCtx(connection)
	if err != nil {
		return nil, err
	}

	return newTunnel(encryptCtx, connection), nil
}
