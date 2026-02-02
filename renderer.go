package leto

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

type Renderer struct {
	templates map[string]*template.Template
}

// New 只做一件事：
// 把「入口模板 = layout + page + partials」预先组合好
// 渲染时通过名字手动指定入口
func New(viewDir string) (*Renderer, error) {
	r := &Renderer{
		templates: make(map[string]*template.Template),
	}

	// 1. base layout（唯一入口）
	layout := filepath.Join(viewDir, "layout", "base.html")
	if _, err := os.Stat(layout); err != nil {
		return nil, fmt.Errorf("layout/base.html not found")
	}

	// 2. partial
	partials, err := filepath.Glob(filepath.Join(viewDir, "partial", "*.html"))
	if err != nil {
		return nil, err
	}

	// 3. pages
	pages, err := filepath.Glob(filepath.Join(viewDir, "page", "*.html"))
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := strings.TrimSuffix(filepath.Base(page), ".html")

		files := []string{layout}
		files = append(files, partials...)
		files = append(files, page)

		tpl, err := template.New("").ParseFiles(files...)
		if err != nil {
			return nil, err
		}

		r.templates[name] = tpl
	}

	return r, nil
}

func (r *Renderer) Render(name string, output string, data any) error {
	tpl, ok := r.templates[name]
	if !ok {
		return fmt.Errorf("template %q not found", name)
	}

	var buf bytes.Buffer
	if err := tpl.ExecuteTemplate(&buf, "base.html", data); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}

	return os.WriteFile(output, buf.Bytes(), 0644)
}
