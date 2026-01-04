package observers

import (
	"context"
	"sync"

	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
)

type EventPublisherImpl struct {
	observers []EventObserver
	mu        sync.RWMutex
}

func NewEventPublisher() *EventPublisherImpl {
	return &EventPublisherImpl{
		observers: make([]EventObserver, 0),
	}
}

func (p *EventPublisherImpl) Publish(ctx context.Context, event model.MetricProcessedEvent) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, observer := range p.observers {
		go observer.OnMetricProcessed(ctx, event)
	}

	return nil
}

func (p *EventPublisherImpl) Register(observer EventObserver) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observers = append(p.observers, observer)
}

func (p *EventPublisherImpl) Unregister(observer EventObserver) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, observerIter := range p.observers {
		if observer == observerIter {
			p.observers = append(p.observers[:i], p.observers[i+1:]...)
		}
	}
}
