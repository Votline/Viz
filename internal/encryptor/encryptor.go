// Package encryptor implements end-to-end encryption
// It uses NaCl Box algorithm
package encryptor

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"go.uber.org/zap"
	"golang.org/x/crypto/nacl/box"
)

type Encryptor struct {
	log           *zap.Logger
	recPub        *[32]byte
	PubKey        *[32]byte
	privKey       *[32]byte
	sharedKey     *[32]byte
	packetCounter uint64
}

func Setup(conn *net.UDPConn, isServer bool, log *zap.Logger) (*Encryptor, *net.UDPAddr, error) {
	enc, err := setupKeys(log)
	if err != nil {
		log.Error("Create encryptor error: ", zap.Error(err))
		return nil, nil, err
	}

	buf := make([]byte, 2048)
	var remoteAddr *net.UDPAddr

	if isServer {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Error("Failed to read client's pubkey: ", zap.Error(err))
			return nil, nil, err
		}
		remoteAddr = addr

		if err := enc.setupConnection(buf[:n]); err != nil {
			return nil, nil, err
		}

		if _, err := conn.WriteToUDP(enc.PubKey[:], remoteAddr); err != nil {
			log.Error("Failed send pubkey to client: ", zap.Error(err))
			return nil, nil, err
		}
	} else {
		if _, err := conn.Write(enc.PubKey[:]); err != nil {
			log.Error("Failed to send pubkey to server: ", zap.Error(err))
			return nil, nil, err
		}

		n, err := conn.Read(buf)
		if err != nil {
			log.Error("Failed to read server's pubkey: ", zap.Error(err))
			return nil, nil, err
		}

		if err := enc.setupConnection(buf[:n]); err != nil {
			return nil, nil, err
		}

		remoteAddr = conn.RemoteAddr().(*net.UDPAddr)
	}

	return enc, remoteAddr, nil
}

func setupKeys(log *zap.Logger) (*Encryptor, error) {
	pub, prv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		log.Error("Create E2EE keys error: ", zap.Error(err))
		return nil, err
	}

	return &Encryptor{
		log:           log,
		PubKey:        pub,
		privKey:       prv,
		packetCounter: 0,
	}, nil
}

func (enc *Encryptor) setupConnection(recPub []byte) error {
	const op = "encryptor.setupConnection"

	if len(recPub) != 32 {
		return fmt.Errorf("%s: invalid public key length: %d", op, len(recPub))
	}
	var othPub [32]byte
	copy(othPub[:], recPub)

	enc.recPub = &othPub
	enc.sharedKey = new([32]byte)
	enc.packetCounter = 0
	box.Precompute(enc.sharedKey, enc.recPub, enc.privKey)

	return nil
}

func (enc *Encryptor) Encrypt(msg []byte) []byte {
	var nonce [24]byte
	binary.BigEndian.PutUint64(nonce[:], enc.packetCounter)
	enc.packetCounter++

	sealed := box.SealAfterPrecomputation(nil, msg, &nonce, enc.sharedKey)

	result := make([]byte, 24+len(sealed))
	copy(result[:24], nonce[:])
	copy(result[24:], sealed)

	return result
}

func (enc *Encryptor) Decrypt(msg []byte) ([]byte, error) {
	const op = "encryptor.Decrypt"
	if len(msg) < 24 {
		return nil, fmt.Errorf("%s: invalid message length: %d", op, len(msg))
	}

	var nonce [24]byte
	copy(nonce[:], msg[:24])
	decrypted, ok := box.OpenAfterPrecomputation(nil, msg[24:], &nonce, enc.sharedKey)
	if !ok {
		enc.log.Error("Failed to decrypt message")
		return nil, errors.New("failed to decrypt message")
	}

	return decrypted, nil
}
