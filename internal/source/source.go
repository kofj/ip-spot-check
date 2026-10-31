package source

import (
	"context"
	"net/netip"

	"github.com/kofj/ip-spot-check/internal/model"
)

var ErrQuotaExceeded error = quotaExceededError{}

type quotaExceededError struct{}

func (quotaExceededError) Error() string { return "额度耗尽" }

type Source interface {
	Name() string
	Query(ctx context.Context, ip netip.Addr) model.Observation
}
