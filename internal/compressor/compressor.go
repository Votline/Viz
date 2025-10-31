package compressor

import (
	"sync"
	"errors"

	"go.uber.org/zap"
	"github.com/hraban/opus"
	"github.com/klauspost/compress/zstd"
)

type Compressor struct {
	mu sync.Mutex
	log     *zap.Logger

	opusEn  *opus.Encoder
	opusDec *opus.Decoder
	zstdEn  *zstd.Encoder
	zstdDec *zstd.Decoder

	ch int
	btr int
	smpR int

	frameDurMs int
}

func NewCmpr(btr, smpR, ch, frameDurMs int, log *zap.Logger) (*Compressor, error) {
	opusEn, err := opus.NewEncoder(smpR, ch, opus.AppVoIP)
	if err != nil {
		log.Error("Create OPUS encoder error", zap.Error(err))
		return nil, err
	}
	opusEn.SetBitrate(btr)

	opusDec, err := opus.NewDecoder(smpR, ch)
	if err != nil {
		log.Error("Create OPUS decoder error", zap.Error(err))
		return nil, err
	}

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
		opusEn:     opusEn,
		opusDec:    opusDec,
		zstdEn:     zstdEn,
		zstdDec:    zstdDec,
		
		ch: ch,
		btr: btr,
		smpR: smpR,

		frameDurMs: frameDurMs,
	}, nil
}

func (c *Compressor) CompressVoice(pcm []int16) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if len(pcm) == 0 {
		return nil, errors.New("empty PCM data")
	}
	
	encoded, err := c.encode(pcm)
	if err != nil {
		c.log.Error("Failed to encode opus audio", zap.Error(err))
		return nil, err
	}
/*
	return c.zstdEn.EncodeAll(voice, nil), nil*/
	return encoded, nil
}

func (c *Compressor) DecompressAudio(bufSize int, zstdAudio []byte) ([]int16, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if len(zstdAudio) == 0 {
		return nil, errors.New("empty audio data")
	}
	
	/*	audio, err := c.zstdDec.DecodeAll(zstdAudio, nil)
		if err != nil {
			c.log.Error("Decode audio error: ", zap.Error(err))
			return nil, err
		}
	
	return pcm, nil*/

	decoded, err := c.decode(zstdAudio, bufSize)
	if err != nil {
		c.log.Error("Failed to decode opus audio", zap.Error(err))
		return nil, err
	}

	return decoded, nil
}

func (c *Compressor) encode(pcm []int16) ([]byte, error) {
	data := make([]byte, 1000)
	n, err := c.opusEn.Encode(pcm, data)
	if err != nil {
		return nil, err
	}
	return data[:n], err
}

func (c *Compressor) decode(opusData []byte, pcmSize int) ([]int16, error) {
	pcm := make([]int16, pcmSize)
	n, err := c.opusDec.Decode(opusData, pcm)
	if err != nil {
		return nil, err
	}
	return pcm[:n], nil
}
