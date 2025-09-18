package syncing

import (
	"container/heap"
	"iter"
	"sync"

	"github.com/evstack/ev-node/block/internal/common"
)

var _ heap.Interface = &PendingQueue{}

type PendingQueue struct {
	mu    sync.RWMutex
	queue []*common.DAHeightEvent
}

func NewPendingQueue() *PendingQueue {
	p := &PendingQueue{
		queue: make([]*common.DAHeightEvent, 0, 1024),
	}
	heap.Init(p)
	return p
}

func (p *PendingQueue) Len() int {
	return len(p.queue)
}

func (p *PendingQueue) Less(i, j int) bool {
	return p.queue[i].Header.Height() < p.queue[j].Header.Height()
}

func (p *PendingQueue) Swap(i, j int) {
	p.queue[i], p.queue[j] = p.queue[j], p.queue[i]
}

func (p *PendingQueue) Push(x any) {
	p.queue = append(p.queue, x.(*common.DAHeightEvent))
}

func (p *PendingQueue) Pop() any {
	n := len(p.queue)
	if n == 0 {
		return nil
	}
	item := p.queue[n-1]
	p.queue = p.queue[: n-1 : cap(p.queue)]
	return item
}

func (p *PendingQueue) AddPendingEvent(event *common.DAHeightEvent) {
	if event == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	heap.Push(p, event)
}

func (p *PendingQueue) Iter() iter.Seq[*common.DAHeightEvent] {
	return func(yield func(*common.DAHeightEvent) bool) {
		p.mu.RLock()
		if p.Len() == 0 {
			p.mu.RUnlock()
			return
		}
		p.mu.RUnlock()

		p.mu.Lock()
		item := heap.Pop(p).(*common.DAHeightEvent)
		p.mu.Unlock()

		if item == nil || !yield(item) {
			return
		}
	}
}
