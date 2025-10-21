package audio

import (
	"sync"

	"github.com/jj11hh/opus"
	"go.uber.org/zap"
)

type audioBuffer struct {
	vol  float32
	pcm  []int16
	wPos int
	rPos int
	mu sync.Mutex
	log *zap.Logger
	encoder *opus.Encoder
	decoder *opus.Decoder
}
func newAB(vol float32, btr, smpR, ch int, log *zap.Logger) (*audioBuffer, error) {
	enc, err := opus.NewEncoder(smpR, ch, opus.AppVoIP)
	if err != nil {
		log.Error("Create OPUS encoder error: ", zap.Error(err))
		return nil, err
	}
	enc.SetBitrate(btr)
	
	dec, err := opus.NewDecoder(smpR, ch)
	if err != nil {
		log.Error("Create OPUS decoder error: ", zap.Error(err))
		return nil, err
	}

	return &audioBuffer{
		vol: vol,
		log: log,
		encoder: enc,
		decoder: dec,
	}, nil
}

func (b *audioBuffer) write(samples []float32) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, sample := range samples {
		if b.wPos < len(b.pcm) {
			b.pcm[b.wPos] = int16(sample * 32767)
			b.wPos++
		}
	}
}

func (b *audioBuffer) data() []int16 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.pcm[:b.wPos]
}

func (b *audioBuffer) setPCM(pcm []int16) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pcm = pcm
}
func (b *audioBuffer) resetPCM(newSize int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.pcm = make([]int16, newSize)
	b.wPos = 0
	b.rPos = 0
}

func (b *audioBuffer) read(out []float32) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i := range out {
		if b.rPos < len(b.pcm) {
			out[i] = float32(b.pcm[b.rPos]) / 32767 * b.vol
			b.rPos++
		} else {
			out[i] = 0
		}
	}
}

func (b *audioBuffer) recorded() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.wPos >= len(b.pcm)
}

func (b *audioBuffer) played() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.rPos >= len(b.pcm)
}

func (b *audioBuffer) resetPlay() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.rPos = 0
}

func (b *audioBuffer) encodeOPUS(sampleRate, channels int, pcm []int16) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	frameSize := sampleRate / 50
	frameBytes := make([]byte, 4000)
	out := make([]byte, 0, len(pcm))

	for i := 0; i+frameSize*channels <= len(pcm); i += frameSize * channels {
		frame := pcm[i : i+frameSize*channels]

		n, err := b.encoder.Encode(frame, frameBytes)
		if err != nil {
			b.log.Error("Encode PCM to OPUS error: ", zap.Error(err))
			return nil, err
		}

		out = append(out, byte(n>>8), byte(n&0xFF))
		out = append(out, frameBytes[:n]...)
	}

	return out, nil
}

func (b *audioBuffer) decodeOPUS(bufferSize, sampleRate, channels int, opusBuffer []byte) ([]int16, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	pcm := make([]int16, 0, len(opusBuffer)*4)
	pos := 0

	for pos+2 <= len(opusBuffer) {
		n := int(opusBuffer[pos])<<8 | int(opusBuffer[pos+1])
		pos += 2
		if pos+n > len(opusBuffer) {
			b.log.Warn("Incomplete OPUS packet")
			break
		}

		frame := opusBuffer[pos : pos+n]
		pos += n

		samples := make([]int16, sampleRate/50*channels)
		decoded, err := b.decoder.Decode(frame, samples)
		if err != nil {
			b.log.Error("Decode OPUS to PCM error: ", zap.Error(err))
			return nil, err
		}

		pcm = append(pcm, samples[:decoded]...)
	}

	return pcm, nil
}
