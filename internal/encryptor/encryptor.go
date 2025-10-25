package encryptor

import (
	"io"
	"errors"
	"crypto/rand"

	"go.uber.org/zap"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/nacl/box"
)

type Encryptor struct {
	log *zap.Logger
	recPub *[32]byte
	PubKey *[32]byte
	privKey *[32]byte
	sharedKey *[32]byte
}

func Setup(log *zap.Logger, conn *websocket.Conn) (*Encryptor, error) {
	enc, err := setupKeys(log)
	if err != nil {
		log.Error("Create encryptor error: ", zap.Error(err))
		return nil, err
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, enc.PubKey[:]); err != nil {
		log.Error("Failed send pubkey to client: ", zap.Error(err))
		return nil, err
	}
	_, othKey, err := conn.ReadMessage()
	if err != nil {
		log.Error("Failed to read client's pubkey: ", zap.Error(err))
		return nil, err
	}
	if err := enc.setupConnection(othKey); err != nil {
		log.Error("Failed to setup connection: ", zap.Error(err))
		return nil, err
	}
	return enc, nil
}

func setupKeys(log *zap.Logger) (*Encryptor, error) {
	pub, prv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		log.Error("Create E2EE keys error: ", zap.Error(err))
		return nil, err
	}

	return &Encryptor{
		log: log,
		PubKey: pub,
		privKey: prv,
	}, nil
}

func (enc *Encryptor) setupConnection(recPub []byte) error {
	if len(recPub) != 32 {
		return errors.New("Invalid public key length")
	}
	var othPub [32]byte
	copy(othPub[:], recPub)

	enc.recPub = &othPub
	enc.sharedKey = new([32]byte)
	box.Precompute(enc.sharedKey, enc.recPub, enc.privKey)

	return nil
}

func (enc *Encryptor) Encrypt(msg []byte) ([]byte, error) {
	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		enc.log.Error("Failed to full nonce: ", zap.Error(err))
		return nil, err
	}
	return box.SealAfterPrecomputation(nonce[:], msg, &nonce, enc.sharedKey), nil
}

func (enc *Encryptor) Decrypt(msg []byte) ([]byte, error) {
	var nonce [24]byte
	copy(nonce[:], msg[:24])
	decrypted, ok := box.OpenAfterPrecomputation(nil, msg[24:], &nonce, enc.sharedKey)
	if !ok {
		enc.log.Error("Failed to decrypt message")
		return nil, errors.New("failed to decrypt message")
	}
	return decrypted, nil
}
