package compressor

import (
	"runtime"

	"go.uber.org/zap"
	"github.com/jj11hh/opus"
	"github.com/klauspost/compress/zstd"
)

type Compressor struct {
	log *zap.Logger
	opusEn *opus.Encoder
	opusDec *opus.Decoder
	zstdEn *zstd.Encoder
	zstdDec *zstd.Decoder
}
func NewCmpr(btr, smpR, ch int, log *zap.Logger) (*Compressor, error) {
	opusEn, err := opus.NewEncoder(smpR, ch, opus.AppVoIP)
	if err != nil {
		log.Error("Create OPUS encoder error: ", zap.Error(err))
		return nil, err
	}
	opusEn.SetBitrate(btr)
	
	opusDec, err := opus.NewDecoder(smpR, ch)
	if err != nil {
		log.Error("Create OPUS decoder error: ", zap.Error(err))
		return nil, err
	}

	zstdEn, err := zstd.NewWriter(nil)
	if err != nil {
		log.Error("Create compressor Writer error: ", zap.Error(err))
		return nil, err
	}

	zstdDec, err := zstd.NewReader(nil)
	if err != nil {
		log.Error("Create compressor Reader error: ", zap.Error(err))
		return nil, err
	}

	return &Compressor{
		log: log,
		opusEn: opusEn,
		opusDec: opusDec,
		zstdEn: zstdEn,
		zstdDec: zstdDec,
	}, nil
}

func (c *Compressor) CompressVoice(smpR, ch int, pcm []int16) ([]byte, error) {
	voice, err := c.encodeOpus(smpR, ch, pcm)
	if err != nil {
		c.log.Error("Encode PCM to OPUS failed: ", zap.Error(err))
		return nil, err
	}

	return c.zstdEn.EncodeAll(voice, nil), nil
}

func (c *Compressor) DecompressAudio(bufSize, smpR, ch int, zstdAudio []byte) ([]int16, error) {
	audio, err := c.zstdDec.DecodeAll(zstdAudio, nil)
	if err != nil {
		c.log.Error("Decode audio error: ", zap.Error(err))
		return nil, err
	}
	
	pcm, err := c.decodeOpus(bufSize, smpR, ch, audio)
	if err != nil {
		c.log.Error("Decode OPUS to PCM error: ", zap.Error(err))
		return nil, err
	}

	return pcm, nil
}

func (c *Compressor) encodeOpus(sampleRate, channels int, pcm []int16) ([]byte, error) {
	frameDurMs := 20
	frameSize := sampleRate * frameDurMs / 1000 * channels

	frameBytes := make([]byte, 4000)
	out := make([]byte, 0, len(pcm))

	c.log.Debug("Starting opus encoding",
		zap.Int("totalSamples: ", len(pcm)),
		zap.Int("frameSize", frameSize),
		zap.Int("channels: ", channels))

	for i := 0; i+frameSize*channels <= len(pcm); i += frameSize * channels {
		frame := pcm[i : i+frameSize]

		n, err := c.opusEn.Encode(frame, frameBytes)
		if err != nil {
			c.log.Error("Encode PCM to OPUS error: ",
			zap.Error(err),
			zap.Int("frameSize: ", frameSize),
			zap.Int("frameIndex: ", i/frameSize))
			return nil, err
		}

		out = append(out, byte(n>>8), byte(n&0xFF))
		out = append(out, frameBytes[:n]...)

		c.log.Debug("Encoded frame",
			zap.Int("frameBytes: ", n),
			zap.Int("totalOut: ", len(out)))
	}

	runtime.GC()

	return out, nil
}

func (c *Compressor) decodeOpus(bufferSize, sampleRate, channels int, opusBuffer []byte) ([]int16, error) {
	pcm := make([]int16, 0, len(opusBuffer)*10)
	frameDurMs := 60
	pos := 0

	c.log.Debug("Starting opus decoding",
		zap.Int("inputBytes: ", len(opusBuffer)))

	packetCount := 0
	for pos+2 <= len(opusBuffer) {
		n := int(opusBuffer[pos])<<8 | int(opusBuffer[pos+1])
		pos += 2
		if pos+n > len(opusBuffer) {
			c.log.Warn("Incomplete OPUS packet",
				zap.Int("packetSize: ", n),
				zap.Int("remainingBytes: ", len(opusBuffer)-pos))
			break
		}

		frame := opusBuffer[pos : pos+n]
		pos += n

		maxFrameSize := sampleRate * frameDurMs / 1000 * channels
		samples := make([]int16, maxFrameSize)

		decoded, err := c.opusDec.Decode(frame, samples)
		if err != nil {
			c.log.Error("Decode OPUS to PCM error: ", zap.Error(err))
			return nil, err
		}

		pcm = append(pcm, samples[:decoded]...)
		packetCount++

		c.log.Debug("Decoded packet",
			zap.Int("decodedSamples: ", decoded),
			zap.Int("totalPCM: ", len(pcm)))
	}

	c.log.Debug("Opus decoding completed",
		zap.Int("packets: ", packetCount),
		zap.Int("totalSamples: ", len(pcm)))

	return pcm, nil
}
