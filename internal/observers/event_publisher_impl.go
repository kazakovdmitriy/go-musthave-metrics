package observers

import (
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

func (p *EventPublisherImpl) Publish(event model.MetricProcessedEvent) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, observer := range p.observers {
		go observer.OnMetricProcessed(event)
	}

	return nil
}

func (p *EventPublisherImpl) Register(observer EventObserver) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.observers = append(p.observers, observer)
}
