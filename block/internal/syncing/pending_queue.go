package syncing

import (
	"container/heap"
	"iter"
	"sync"

	"github.com/evstack/ev-node/block/internal/common"
)

var _ heap.Interface = &queue{}

type queue []*common.DAHeightEvent

func (p queue) Len() int {
	return len(p)
}

func (p queue) Less(i, j int) bool {
	return p[i].Header.Height() < p[j].Header.Height()
}

func (p queue) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p *queue) Push(x any) {
	*p = append(*p, x.(*common.DAHeightEvent))
}

func (p *queue) Pop() any {
	n := len(*p)
	if n == 0 {
		return nil
	}
	item := (*p)[n-1]
	*p = (*p)[: n-1 : cap(*p)]
	return item
}

type PendingQueue struct {
	mu    sync.RWMutex
	queue queue
}

func NewPendingQueue() *PendingQueue {
	p := &PendingQueue{
		queue: make([]*common.DAHeightEvent, 0, 1024),
	}
	heap.Init(&p.queue)
	return p
}

func (p *PendingQueue) AddPendingEvent(event *common.DAHeightEvent) {
	if event == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	heap.Push(&p.queue, event)
}

func (p *PendingQueue) Iter() iter.Seq[*common.DAHeightEvent] {
	return func(yield func(*common.DAHeightEvent) bool) {
		item := p.Pop()
		if item == nil || !yield(item) {
			return
		}
	}
}

func (p *PendingQueue) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.queue.Len()
}

func (p *PendingQueue) Pop() *common.DAHeightEvent {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.queue.Len() == 0 {
		return nil
	}
	return heap.Pop(&p.queue).(*common.DAHeightEvent)
}

func (p *PendingQueue) Peek() *common.DAHeightEvent {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.queue.Len() == 0 {
		return nil
	}
	return p.queue[0]
}
