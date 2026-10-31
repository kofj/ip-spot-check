package model

import (
	"net/netip"
	"time"
)

type Observation struct {
	Source  string
	Fields  map[string]any
	Fetched time.Time
	Err     error
}

func (o Observation) Failed() bool { return o.Err != nil }

type Report struct {
	Addr         netip.Addr
	Observations []Observation
	QueriedAt    time.Time
}
