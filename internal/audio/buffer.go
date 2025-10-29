package audio

import (
	"sync"

	"go.uber.org/zap"
)

type audioBuffer struct {
	vol  float32
	pcm  []int16
	wPos int
	rPos int
	mu sync.Mutex
	log *zap.Logger
	recording bool
}
func newAB(vol float32, log *zap.Logger) *audioBuffer {
	return &audioBuffer{
		vol: vol,
		log: log,
	}
}

func (b *audioBuffer) write(samples []float32) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.recording || b.pcm == nil {
		b.log.Warn("Buffer is nil or already recording")
		return
	}

	for _, sample := range samples {
		if b.wPos >= len(b.pcm) {
			b.log.Warn("Buffer overflow",
				zap.Int("wPos: ", b.wPos),
				zap.Int("bufferSize: ", len(b.pcm)))
			return
		}

		if sample < -1 {sample = -1
		} else if sample > 1 {sample = 1}

		b.pcm[b.wPos] = int16(sample * 32767)
		b.wPos++
	}
}

func (b *audioBuffer) data() []int16 {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.wPos == 0 {
		return nil
	}

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

	b.recording = true
	b.pcm = make([]int16, newSize)
	b.wPos = 0
	b.rPos = 0
}

func (b *audioBuffer) read(out []float32) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.log.Debug("Reading from buffer",
		zap.Int("outSize: ", len(out)),
		zap.Int("rPos: ", b.rPos),
		zap.Int("pcmLen: ", len(b.pcm)))

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

	played :=  b.rPos >= len(b.pcm)
	if played {
		b.log.Debug("Buffer fully played",
			zap.Int("rPos: ", b.rPos),
			zap.Int("pcmLen: ", len(b.pcm)))
	}
	return played
}

func (b *audioBuffer) resetPlay() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.rPos = 0
}

func (b *audioBuffer) stopRecording() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.recording = false
}

func (b *audioBuffer) cleanup() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.pcm = nil
	b.wPos = 0
	b.rPos = 0
	b.recording = false
}
