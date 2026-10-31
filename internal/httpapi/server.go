package httpapi

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/netip"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kofj/ip-spot-check/internal/query"
	"github.com/kofj/ip-spot-check/internal/store"
	webassets "github.com/kofj/ip-spot-check/web"
)

var hashedAsset = regexp.MustCompile(`\.[0-9a-f]{8,}\.`)

type Server struct {
	engine *query.Engine
	store  *store.Store
	tmpl   *template.Template
}

func New(engine *query.Engine, st *store.Store) (*Server, error) {
	tmpl, err := template.ParseFS(webassets.FS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("解析模板: %w", err)
	}
	return &Server{engine: engine, store: st, tmpl: tmpl}, nil
}

func (s *Server) Router() *gin.Engine {
	r := gin.New()
	r.SetHTMLTemplate(s.tmpl)
	r.Use(gin.Recovery(), gin.Logger())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	static, err := fs.Sub(webassets.FS, "static")
	if err != nil {
		panic(err)
	}
	r.GET("/static/*filepath", gin.WrapH(staticHandler(static)))
	r.HEAD("/static/*filepath", gin.WrapH(staticHandler(static)))

	r.GET("/", s.index)
	r.GET("/ip/:addr", s.report)
	r.GET("/:addr", s.plain)

	return r
}

func staticHandler(fsys fs.FS) http.Handler {
	fileServer := http.StripPrefix("/static/", http.FileServer(http.FS(fsys)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &cacheableWriter{ResponseWriter: w, cacheControl: cacheControlFor(r.URL.Path)}
		fileServer.ServeHTTP(sw, r)
	})
}

type cacheableWriter struct {
	http.ResponseWriter
	cacheControl string
	wroteHeader  bool
}

func (w *cacheableWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		if code == http.StatusOK {
			w.Header().Set("Cache-Control", w.cacheControl)
		}
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *cacheableWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

func (s *Server) index(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=300, s-maxage=3600")
	c.Render(http.StatusOK, newRender(s.tmpl).Instance("index.html", nil))
}

func (s *Server) report(c *gin.Context) {
	addr := c.Param("addr")
	if !looksLikeIP(addr) {
		c.String(http.StatusNotFound, "not an ip: %s", addr)
		return
	}
	c.Header("Cache-Control", "public, max-age=60, s-maxage=3600, stale-while-revalidate=86400")
	c.Render(http.StatusOK, newRender(s.tmpl).Instance("report.html", gin.H{
		"Addr": addr,
	}))
}

func (s *Server) plain(c *gin.Context) {
	addr := c.Param("addr")
	if !looksLikeIP(addr) {
		c.String(http.StatusNotFound, "not an ip: %s", addr)
		return
	}
	c.Header("Cache-Control", "public, max-age=300, s-maxage=3600, stale-while-revalidate=86400")
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, "ip: %s\nqueried_at: %s\n", addr, time.Now().UTC().Format(time.RFC3339))
}

// 只有 200 响应带缓存头；404 等错误响应不缓存，避免缺失的 hash 资源被长期记住。
func cacheControlFor(path string) string {
	if hashedAsset.MatchString(strings.TrimSuffix(path, "/")) {
		return "public, max-age=31536000, immutable"
	}
	return "no-cache"
}

func looksLikeIP(s string) bool {
	_, err := netip.ParseAddr(s)
	return err == nil
}
