package source

import (
	"errors"
	"testing"
)

func TestErrQuotaExceededIsComparable(t *testing.T) {
	var err error = ErrQuotaExceeded
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Error("应可用 errors.Is 判断")
	}
	if err.Error() != "额度耗尽" {
		t.Errorf("message = %q", err.Error())
	}
}
