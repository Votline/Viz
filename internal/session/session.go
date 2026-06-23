// Package session manages websocket connection and audio stream
package session

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"Viz/internal/audio"
	"Viz/internal/batch"
	"Viz/internal/encryptor"
	"Viz/internal/ringbuffer"
)

const (
	bufferSize   = 1920 * 10
	standartSize = 256
	batchSize    = 8
)

type Session struct {
	conn    *websocket.Conn
	log     *zap.Logger
	recBuf  *ringbuffer.RingBuffer[byte]
	playBuf *ringbuffer.RingBuffer[byte]
	enc     *encryptor.Encryptor
	errCom  atomic.Value
	warnCom atomic.Value
}

// StartSession initialize audio client, encryptor, ringbuffers and websocket connection
// starts sending voice and appending voice to ringbuffers in goroutines
func StartSession(conn *websocket.Conn, log *zap.Logger) error {
	const op = "session.StartSession"

	acl, err := audio.NewAudioStream(log)
	if err != nil {
		return fmt.Errorf("%s: create audiostream: %w", op, err)
	}

	log.Debug("Audio stream created",
		zap.String("op", op))

	enc, err := encryptor.Setup(log, conn)
	if err != nil {
		return fmt.Errorf("%s: setup encryptor: %w", op, err)
	}

	log.Debug("Encryptor setup",
		zap.String("op", op))

	recBuf := ringbuffer.NewRB[byte](bufferSize)
	playBuf := ringbuffer.NewRB[byte](bufferSize)

	go acl.Record(recBuf)
	go acl.Play(playBuf)

	var errCom atomic.Value
	var warnCom atomic.Value
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sess := &Session{
		conn:    conn,
		log:     log,
		recBuf:  recBuf,
		playBuf: playBuf,
		errCom:  errCom,
		warnCom: warnCom,
		enc:     enc,
	}

	go func() {
		defer cancel()
		sess.sendVoice()
	}()

	log.Debug("Voice sending started",
		zap.String("op", op))

	go func() {
		defer cancel()
		sess.appendVoice()
	}()

	log.Debug("Voice appending started",
		zap.String("op", op))

	for {
		if msg := errCom.Load(); msg != nil {
			err := msg.(error)
			cancel()
			recBuf.Close()
			playBuf.Close()
			return fmt.Errorf("%s: error: %w", op, err)
		}
		if msg := warnCom.Load(); msg != nil {
			err := msg.(error)
			log.Warn("Warning: ",
				zap.String("op", op),
				zap.Error(err))
		}

		select {
		case <-ctx.Done():
			log.Debug("Session ended",
				zap.String("op", op))
			recBuf.Close()
			playBuf.Close()
			return nil
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// sendVoice reads voice chunk from ringbuffer and sends it to websocket
// uses batching to send voice chunks to websocket
func (s *Session) sendVoice() {
	const op = "session.sendVoice"

	var sizeBuf [4]byte
	batchBuffer := make([][]byte, 0, batchSize*2)
	voiceChunkFull := make([]byte, standartSize*2)
	for {
		if _, err := io.ReadFull(s.recBuf, sizeBuf[:]); err != nil {
			break // closed
		}

		size := binary.LittleEndian.Uint32(sizeBuf[:])
		voiceChunk := voiceChunkFull[:size]

		if _, err := io.ReadFull(s.recBuf, voiceChunk); err != nil {
			break // closed
		}

		s.log.Warn("Read voice chunk",
			zap.String("op", op),
			zap.Uint32("size", size))

		encChunk := s.enc.Encrypt(voiceChunk)
		batchBuffer = append(batchBuffer, encChunk)

		s.log.Debug("Encrypted voice chunk",
			zap.String("op", op),
			zap.Int("size", len(encChunk)))

		if len(batchBuffer) >= batchSize {
			packedBatch := batch.PackBatch(batchBuffer)
			if err := s.conn.WriteMessage(websocket.BinaryMessage, packedBatch); err != nil {
				s.errCom.Store(fmt.Errorf("%s: write message: %w", op, err))
				return
			}
			batchBuffer = batchBuffer[:0]
			s.log.Debug("Written batch",
				zap.String("op", op),
				zap.Int("size", len(packedBatch)))
		}
	}
}

// appendVoice applies voice chunk, decrypts it and appends it to ringbuffer
// uses unbatching to send voice chunks to ringbuffer
func (s *Session) appendVoice() {
	const op = "session.appendVoice"

	var sizeBuf [4]byte
	for {
		_, msg, err := s.conn.ReadMessage()
		if err != nil {
			s.errCom.Store(fmt.Errorf("%s: read message: %w", op, err))
			return
		}

		s.log.Debug("Read message",
			zap.String("op", op),
			zap.Int("size", len(msg)))

		frames, err := batch.UnpackBatch(msg)
		if err != nil {
			s.warnCom.Store(fmt.Errorf("%s: unpack batch: %w", op, err))
			continue
		}

		s.log.Debug("Unpacked batch",
			zap.String("op", op),
			zap.Int("size", len(frames)))

		for _, frame := range frames {
			decFrame, err := s.enc.Decrypt(frame)
			if err != nil {
				s.warnCom.Store(fmt.Errorf("%s: decrypt frame: %w", op, err))
			}

			s.log.Debug("Decrypted frame",
				zap.String("op", op),
				zap.Int("size", len(decFrame)))

			size := uint32(len(decFrame))
			binary.LittleEndian.PutUint32(sizeBuf[:], size)
			if n := s.playBuf.WriteSimple(sizeBuf[:]); n == -1 {
				break
			}

			n := s.playBuf.WriteSimple(decFrame)
			if n == 0 {
				s.warnCom.Store(fmt.Errorf("%s: write frame: %w", op, err))
			} else if n == -1 {
				break
			}
			s.log.Debug("Written frame",
				zap.String("op", op),
				zap.Int("size", n))
		}
	}
}
