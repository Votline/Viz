package audio

import (
	"sync"
)

type audioQueue struct {
	mu sync.Mutex
	queue [][]byte
}
func newAQ() *audioQueue {
	return &audioQueue{queue: make([][]byte, 0)}
}

func (aq *audioQueue) Push(chunk []byte) {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	aq.queue = append(aq.queue, chunk)
}

func (aq *audioQueue) pop() []byte {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	if len(aq.queue) == 0 {
		return nil
	}
	chunk := aq.queue[0]
	aq.queue = aq.queue[1:]
	return chunk
}
