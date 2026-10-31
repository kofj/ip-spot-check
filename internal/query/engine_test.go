package query

import (
	"context"
	"net/netip"
	"testing"

	"github.com/kofj/ip-spot-check/internal/model"
	"github.com/kofj/ip-spot-check/internal/source"
)

type fakeSource struct {
	name    string
	fields  map[string]any
	err     error
	queried *[]string
}

func (f fakeSource) Name() string { return f.name }

func (f fakeSource) Query(_ context.Context, ip netip.Addr) model.Observation {
	if f.queried != nil {
		*f.queried = append(*f.queried, f.name)
	}
	return model.Observation{Source: f.name, Fields: f.fields, Err: f.err}
}

func TestRunQueriesAllSources(t *testing.T) {
	ip := netip.MustParseAddr("1.1.1.1")
	var order []string

	e := New(
		fakeSource{name: "a", fields: map[string]any{"k": 1}, queried: &order},
		fakeSource{name: "b", fields: map[string]any{"k": 2}, queried: &order},
	)
	rep := e.Run(context.Background(), ip)

	if !rep.Addr.IsValid() || rep.Addr != ip {
		t.Errorf("addr = %v", rep.Addr)
	}
	if len(rep.Observations) != 2 {
		t.Fatalf("observations = %d", len(rep.Observations))
	}
	if rep.Observations[0].Source != "a" || rep.Observations[1].Source != "b" {
		t.Error("结果顺序应与源注册顺序一致")
	}
	if len(order) != 2 {
		t.Errorf("两个源都应被调用，实际 %d 次", len(order))
	}
}

func TestRunPreservesPartialFailure(t *testing.T) {
	e := New(
		fakeSource{name: "ok", fields: map[string]any{"v": 1}},
		fakeSource{name: "broken", err: source.ErrQuotaExceeded},
	)
	rep := e.Run(context.Background(), netip.MustParseAddr("8.8.8.8"))

	if len(rep.Observations) != 2 {
		t.Fatalf("源失败不应导致整体结果缺失")
	}
	if rep.Observations[0].Failed() {
		t.Errorf("第一个源应成功，实际错误: %v", rep.Observations[0].Err)
	}
	if !rep.Observations[1].Failed() {
		t.Fatal("第二个源应标记失败")
	}
	if rep.Observations[1].Err != source.ErrQuotaExceeded {
		t.Errorf("错误应原样透出，得到 %v", rep.Observations[1].Err)
	}
}
