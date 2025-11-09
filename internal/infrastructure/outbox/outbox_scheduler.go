package outbox

import (
	"context"
	"log"
	"time"
)

type Scheduler struct {
	dispatcher *Dispatcher
	interval   time.Duration
}

func NewScheduler(d *Dispatcher, intervalSec int) *Scheduler {
	return &Scheduler{
		dispatcher: d,
		interval:   time.Duration(intervalSec) * time.Second,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Printf("Outbox scheduler stopped")
				return
			case <-ticker.C:
				n, err := s.dispatcher.DispatchOnce(ctx)
				if err != nil {
					log.Printf("Outbox dispatch error: %v", err)
				} else if n > 0 {
					log.Printf("Outbox dispatch processed %d messages", n)
				}
			}
		}
	}()
}
