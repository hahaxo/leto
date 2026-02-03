package leto

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
)

func GeneratePage(templateFile string, output string, data any) error {
	name := filepath.Base(templateFile)

	tpl := template.Must(template.New("page").ParseFiles(templateFile))

	var buf bytes.Buffer

	if err := tpl.ExecuteTemplate(&buf, name, data); err != nil {
		return err
	}

	if err := os.MkdirAll("dist", 0755); err != nil {
		return err
	}

	if err := os.WriteFile("dist/"+output, buf.Bytes(), 0644); err != nil {
		return err
	}

	return nil
}
