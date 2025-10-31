package compressor

import (
	"sync"
	"errors"

	"go.uber.org/zap"
	"github.com/jj11hh/opus"
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

func NewCmpr(btr, smpR, ch int, log *zap.Logger) (*Compressor, error) {
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

		frameDurMs: 10,
	}, nil
}

func (c *Compressor) CompressVoice(pcm []int16) ([]byte, error) {
	if c.opusEn == nil {
		return nil, errors.New("Opus Encoder not initialized")
	}
	if len(pcm) == 0 {
		return nil, errors.New("empty PCM data")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	voice, err := c.encodeOpus(pcm)
	if err != nil {
		c.log.Error("Encode PCM to OPUS failed", zap.Error(err))
		return nil, err
	}

	return voice, nil
	//return c.zstdEn.EncodeAll(voice, nil), nil
}

func (c *Compressor) DecompressAudio(bufSize int, zstdAudio []byte) ([]int16, error) {
	defer func() {
		if r := recover(); r != nil {
			c.log.Error("recover: PANIC in DecompressAudio",
				zap.Any("recover", r))
		}
	}()
	if c.opusEn == nil {
		return nil, errors.New("Opus Encoder not initialized")
	}
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
	*/
	pcm, err := c.decodeOpus(zstdAudio)
	if err != nil {
		c.log.Error("Decode OPUS to PCM error", zap.Error(err))
		return nil, err
	}

	return pcm, nil
}

func (c *Compressor) encodeOpus(pcm []int16) ([]byte, error) {
	pcmCopy := make([]int16, len(pcm))
	copy(pcmCopy, pcm)
	frameSize := c.smpR * c.frameDurMs / 1000 * c.ch

	if frameSize%c.ch != 0 {
		frameSize = (frameSize / c.ch) * c.ch
	}
	if frameSize < 120 {
		frameSize = 120
	}
	if frameSize > 480 {
		frameSize = 480
	}

	frameBytes := make([]byte, 1500)
	out := make([]byte, 0, len(pcmCopy)/2)

	for i := 0; i+frameSize <= len(pcmCopy); i += frameSize {
		frame := pcmCopy[i : i+frameSize]

		n, err := c.opusEn.Encode(frame, frameBytes)
		if err != nil {
			c.log.Error("Encode PCM to OPUS error",
				zap.Error(err),
				zap.Int("frameSize", frameSize),
				zap.Int("frameIndex", i/frameSize))
			continue
		}

		if n > len(frameBytes) {
			c.log.Warn("Encoded frame too large", zap.Int("size", n))
			n = len(frameBytes)
		}

		out = append(out, byte(n>>8), byte(n&0xFF))
		out = append(out, frameBytes[:n]...)
	}

	return out, nil
}

func (c *Compressor) decodeOpus(opusBuffer []byte) ([]int16, error) {
	if len(opusBuffer) < 2 {
		return nil, errors.New("opus buffer too short")
	}

	pcm := make([]int16, 0, len(opusBuffer)*8)

	pos := 0
	packetCount := 0

	c.log.Debug("Starting opus decoding",
		zap.Int("inputBytes", len(opusBuffer)))

	tempSamples := make([]int16, 480*c.ch)

	for pos+2 <= len(opusBuffer) {
		n := int(opusBuffer[pos])<<8 | int(opusBuffer[pos+1])
		pos += 2

		if n <= 0 || n > len(opusBuffer)-pos {
			c.log.Warn("Invalid OPUS packet size",
				zap.Int("packetSize", n),
				zap.Int("remainingBytes", len(opusBuffer)-pos))
			break
		}

		if n > 4000 {
			c.log.Warn("Suspiciously large OPUS packet",
				zap.Int("size", n))
			pos += n
			continue
		}

		if pos+n > len(opusBuffer) {
			c.log.Warn("Incomplete OPUS packet",
				zap.Int("packetSize", n),
				zap.Int("remainingBytes", len(opusBuffer)-pos))
			break
		}

		frame := opusBuffer[pos : pos+n]
		pos += n

		if len(pcm) > 48000*5 {
			c.log.Warn("PCM buffer too large, truncating")
			break
		}

		decoded, err := c.opusDec.Decode(frame, tempSamples)
		if err != nil {
			c.log.Error("Decode OPUS to PCM error",
				zap.Error(err),
				zap.Int("packetSize", n),
				zap.Int("packetIndex", packetCount))
			continue
		}

		pcm = append(pcm, tempSamples[:decoded]...)
		packetCount++
	}

	c.log.Debug("Opus decoding completed",
		zap.Int("packets", packetCount),
		zap.Int("totalSamples", len(pcm)))

	return pcm, nil
}
