package audio

import (
	"sync"
)

type allQueue struct {
	AQ *queue
	pQ *queue
}
type queue struct {
	mu sync.Mutex
	buf []any
}
func newQueue() *allQueue {
	return &allQueue{
		AQ: &queue{buf: make([]any, 0)},
		pQ: &queue{buf: make([]any, 0)},
	}
}

func (aq *allQueue) Push(chunk any, q *queue) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.buf = append(q.buf, chunk)
}

func (aq *allQueue) pop(q *queue) any {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.buf) == 0 {
		return nil
	}
	chunk := q.buf[0]
	q.buf = q.buf[1:]
	return chunk
}

func (aq *allQueue) length(q *queue) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.buf)
}
