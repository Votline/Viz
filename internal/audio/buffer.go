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
		b.log.Warn("Buffer is nil")
		return
	}

	for _, sample := range samples {
		if b.wPos >= len(b.pcm) {
			b.log.Warn("Buffer overflow",
				zap.Int("wPos: ", b.wPos),
				zap.Int("bufferSize: ", len(b.pcm)))
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
		return []int16{}
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
