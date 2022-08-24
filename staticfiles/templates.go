package staticfiles

import (
	_ "embed"
	"text/template"

	"github.com/gohugoio/hugoreleaser/internal/common/templ"
)

var (
	//go:embed templates/release-notes.gotmpl
	releaseNotesTemplContent []byte

	// ReleaseNotesTemplate is the template for the release notes.
	ReleaseNotesTemplate *template.Template
)

func init() {
	ReleaseNotesTemplate = template.Must(template.New("release-notes").Parse(string(releaseNotesTemplContent))).Funcs(templ.BuiltInFuncs)
}
