package httpapi

import (
	"html/template"

	"github.com/gin-gonic/gin/render"
)

type tmplRender struct {
	templates *template.Template
}

func (r tmplRender) Instance(name string, data any) render.Render {
	return render.HTML{
		Template: r.templates.Lookup(name),
		Name:     name,
		Data:     data,
	}
}

// 保证只在 HTTP 层出现 gin 依赖
var _ render.HTMLRender = tmplRender{}

func newRender(t *template.Template) render.HTMLRender { return tmplRender{templates: t} }
