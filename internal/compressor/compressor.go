package compressor

import (
	"sync"
	"errors"

	"go.uber.org/zap"
	"github.com/klauspost/compress/zstd"
)

type Compressor struct {
	mu sync.Mutex
	log     *zap.Logger

	zstdEn  *zstd.Encoder
	zstdDec *zstd.Decoder

	ch int
	btr int
	smpR int

	frameDurMs int
}

func NewCmpr(btr, smpR, ch int, log *zap.Logger) (*Compressor, error) {
	zstdEn, err := zstd.NewWriter(nil)
	if err != nil {
		log.Error("Create compressor Writer error", zap.Error(err))
		return nil, err
	}

	zstdDec, err := zstd.NewReader(nil)
	if err != nil {
		log.Error("Create compressor Reader error", zap.Error(err))
		return nil, err
	}

	return &Compressor{
		log:        log,
		zstdEn:     zstdEn,
		zstdDec:    zstdDec,
		
		ch: ch,
		btr: btr,
		smpR: smpR,

		frameDurMs: 10,
	}, nil
}

func (c *Compressor) CompressVoice(pcm []int16) ([]int16, error) {
	if len(pcm) == 0 {
		return nil, errors.New("empty PCM data")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
/*
	return c.zstdEn.EncodeAll(voice, nil), nil*/
	return pcm, nil
}

func (c *Compressor) DecompressAudio(bufSize int, zstdAudio []int16) ([]int16, error) {
	if len(zstdAudio) == 0 {
		return nil, errors.New("empty audio data")
	}
	
	c.mu.Lock()
	defer c.mu.Unlock()

	/*	audio, err := c.zstdDec.DecodeAll(zstdAudio, nil)
		if err != nil {
			c.log.Error("Decode audio error: ", zap.Error(err))
			return nil, err
		}
	
	return pcm, nil*/
	return zstdAudio, nil

}

