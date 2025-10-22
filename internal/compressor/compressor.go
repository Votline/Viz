package compressor

import (
	"sync"
	
	"go.uber.org/zap"
	"github.com/klauspost/compress/zstd"
)

type Compressor struct {
	log *zap.Logger
	mu sync.Mutex
	encoder *zstd.Encoder
	decoder *zstd.Decoder
}
func NewCmpr(log *zap.Logger) (*Compressor, error) {
	en, err := zstd.NewWriter(nil)
	if err != nil {
		log.Error("Create compressor Writer error: ", zap.Error(err))
		return nil, err
	}

	dec, err := zstd.NewReader(nil)
	if err != nil {
		log.Error("Create compressor Reader error: ", zap.Error(err))
		return nil, err
	}
	return &Compressor{
		log: log,
		encoder: en,
		decoder: dec,
	}, nil
}

func (c *Compressor) CompressVoice(voice []byte) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.encoder.EncodeAll(voice, nil)
}

func (c *Compressor) DecompressAudio(audio []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	bytes, err := c.decoder.DecodeAll(audio, nil)
	if err != nil {
		c.log.Error("Decode audio error: ", zap.Error(err))
		return nil, err
	}

	return bytes, nil
}
