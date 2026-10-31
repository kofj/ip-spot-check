package model

import (
	"errors"
	"testing"
)

func TestObservationFailed(t *testing.T) {
	if (Observation{}).Failed() {
		t.Error("零值不应视为失败")
	}
	if !(Observation{Err: errors.New("x")}).Failed() {
		t.Error("带错误应视为失败")
	}
}
