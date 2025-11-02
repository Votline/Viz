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

	if b.pcm == nil {
		b.log.Warn("Buffer is nil or already recording")
		return
	}

	for _, sample := range samples {
		if sample < -1 {sample = -1
		} else if sample > 1 {sample = 1}

		b.pcm[b.wPos] = int16(sample * 32767)
		b.wPos = (b.wPos + 1) % len(b.pcm)

		if b.wPos == b.rPos {
			b.rPos = (b.rPos + 1) % len(b.pcm)
		}
	}
}

func (b *audioBuffer) read(out []float32) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.log.Debug("Reading from buffer",
		zap.Int("outSize: ", len(out)),
		zap.Int("rPos: ", b.rPos),
		zap.Int("pcmLen: ", len(b.pcm)))

	for i := range out {
		if b.rPos != b.wPos {
			out[i] = float32(b.pcm[b.rPos]) / 32767 * b.vol
			b.rPos = (b.rPos + 1) % len(b.pcm)
		} else {
			out[i] = 0
		}
	}
}

func (b *audioBuffer) appendPCM(newPCM []int16) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.pcm == nil {
		return
	}

	for _, sample := range newPCM {
		b.pcm[b.wPos] = sample
		b.wPos = (b.wPos + 1) % len(b.pcm)

		if b.wPos == b.rPos {
			b.rPos = (b.rPos + 1) % len(b.pcm)
		}
	}
}

func (b *audioBuffer) copyChunk(dest []int16) int {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.pcm == nil || b.rPos == b.wPos {
		return 0
	}

	copied := 0
	for copied < len(dest) {
		if b.rPos == b.wPos {
			break
		}

		dest[copied] = b.pcm[b.rPos]
		b.rPos = (b.rPos + 1) % len(b.pcm)
		copied++
	}

	return copied
}

func (b *audioBuffer) available() int {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.pcm == nil {
		return 0
	}

	if b.wPos >= b.rPos {
		return b.wPos - b.rPos
	}

	return len(b.pcm) - b.rPos + b.wPos
}

func (b *audioBuffer) data() []int16 {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.wPos == 0 {
		return nil
	}

	return b.pcm[:b.wPos]
}

func (b *audioBuffer) resetPCM(newSize int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.recording = true
	b.pcm = make([]int16, newSize)
	b.wPos = 0
	b.rPos = 0
}

func (b *audioBuffer) recorded() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.wPos >= len(b.pcm)
}

func (b *audioBuffer) getReadPos() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.rPos
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
