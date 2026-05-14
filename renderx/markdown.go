package renderx

import (
	"bytes"
	"html/template"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
)

const markdownHighlightStyle = "github"

var markdownChromaOptions = []chromahtml.Option{
	chromahtml.WithClasses(true),
}

var markdownRenderer = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		highlighting.NewHighlighting(
			highlighting.WithStyle(markdownHighlightStyle),
			highlighting.WithFormatOptions(
				markdownChromaOptions...,
			),
		),
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		goldmarkhtml.WithHardWraps(),
	),
)

// Markdown renders Markdown source into HTML safe for html/template.
func Markdown(source string) (template.HTML, error) {
	var buf bytes.Buffer
	if err := markdownRenderer.Convert([]byte(source), &buf); err != nil {
		return "", err
	}

	return template.HTML(buf.String()), nil
}

// MarkdownCSS renders the CSS rules required by Markdown code highlighting.
func MarkdownCSS() (template.CSS, error) {
	var buf bytes.Buffer
	formatter := chromahtml.New(markdownChromaOptions...)

	style := styles.Get(markdownHighlightStyle)
	if style == nil {
		style = styles.Fallback
	}

	if err := formatter.WriteCSS(&buf, style); err != nil {
		return "", err
	}

	return template.CSS(buf.String()), nil
}
