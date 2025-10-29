package audio

import (
	"sync"
)

type allQueue struct {
	mu sync.Mutex
	AQ *queue
	pQ *queue
}
type queue struct {
	buf []any
}
func newQueue() *allQueue {
	return &allQueue{
		AQ: &queue{buf: make([]any, 0)},
		pQ: &queue{buf: make([]any, 0)},
	}
}

func (aq *allQueue) Push(chunk any, q *queue) {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	q.buf = append(q.buf, chunk)
}

func (aq *allQueue) pop(q *queue) any {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	if len(q.buf) == 0 {
		return nil
	}
	chunk := q.buf[0]
	q.buf = q.buf[1:]
	return chunk
}

func (aq *allQueue) length() int {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	return len(aq.pQ.buf)
}

func (aq *allQueue) clearQueue(q *queue) {
	aq.mu.Lock()
	defer aq.mu.Unlock()
	q.buf = make([]any, 0)
}
