package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	srv, err := New(nil, nil)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return srv.Router()
}

func do(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHealthz(t *testing.T) {
	rec := do(t, newTestRouter(t), "/healthz")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ok") {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestIndexServed(t *testing.T) {
	rec := do(t, newTestRouter(t), "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ip-spot-check") {
		t.Error("首页应包含站名")
	}
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "s-maxage") {
		t.Errorf("Cache-Control = %q，应可被 CDN 缓存", cc)
	}
}

func TestReportPageNoTotalScore(t *testing.T) {
	rec := do(t, newTestRouter(t), "/ip/1.1.1.1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "不提供总分") {
		t.Error("报告页必须声明不提供总分")
	}
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "stale-while-revalidate") {
		t.Errorf("Cache-Control = %q", cc)
	}
}

func TestPlainTextResponse(t *testing.T) {
	rec := do(t, newTestRouter(t), "/8.8.8.8")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "8.8.8.8") {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestNonIPPathIs404(t *testing.T) {
	for _, p := range []string{"/favicon", "/favicon.ico", "/robots.txt", "/hello.world"} {
		if rec := do(t, newTestRouter(t), p); rec.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404", p, rec.Code)
		}
	}
}

func TestIPv6PlainText(t *testing.T) {
	rec := do(t, newTestRouter(t), "/2606:4700::1111")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "2606:4700::1111") {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestCacheControlForHashedAsset(t *testing.T) {
	for _, p := range []string{"/static/app.a3f9c2d1.css", "/static/app.0123456789abcdef.js"} {
		if cc := cacheControlFor(p); !strings.Contains(cc, "immutable") {
			t.Errorf("%s -> %q，应 immutable", p, cc)
		}
	}
}

func TestCacheControlForUnhashedAsset(t *testing.T) {
	for _, p := range []string{"/static/app.css", "/static/a3f9c2d1.css"} {
		if cc := cacheControlFor(p); !strings.Contains(cc, "no-cache") {
			t.Errorf("%s -> %q，应 no-cache", p, cc)
		}
	}
}

func TestUnhashedAssetIsNoCache(t *testing.T) {
	rec := do(t, newTestRouter(t), "/static/app.css")
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "no-cache") {
		t.Errorf("不带 hash 的资源 Cache-Control = %q", cc)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("静态资源应可访问，status = %d", rec.Code)
	}
}
