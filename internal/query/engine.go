package query

import (
	"context"
	"sync"
	"time"

	"net/netip"

	"github.com/kofj/ip-spot-check/internal/model"
	"github.com/kofj/ip-spot-check/internal/source"
)

type Engine struct {
	sources []source.Source
}

func New(sources ...source.Source) *Engine {
	return &Engine{sources: sources}
}

func (e *Engine) Run(ctx context.Context, ip netip.Addr) model.Report {
	results := make([]model.Observation, len(e.sources))
	var wg sync.WaitGroup

	for i, s := range e.sources {
		wg.Add(1)
		go func(i int, s source.Source) {
			defer wg.Done()
			results[i] = s.Query(ctx, ip)
		}(i, s)
	}
	wg.Wait()

	return model.Report{Addr: ip, Observations: results, QueriedAt: time.Now()}
}

func (e *Engine) Sources() []source.Source { return e.sources }
