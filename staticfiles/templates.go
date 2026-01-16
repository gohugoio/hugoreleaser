package staticfiles

import (
	_ "embed"
	"text/template"

	"github.com/gohugoio/hugoreleaser/internal/common/templ"
)

var (
	//go:embed templates/release-notes.gotmpl
	releaseNotesTemplContent []byte

	//go:embed templates/homebrew-cask.rb.gotmpl
	homebrewCaskTemplContent []byte

	// ReleaseNotesTemplate is the template for the release notes.
	ReleaseNotesTemplate *template.Template

	// HomebrewCaskTemplate is the template for the Homebrew cask file.
	HomebrewCaskTemplate *template.Template
)

func init() {
	ReleaseNotesTemplate = template.Must(template.New("release-notes").Funcs(templ.BuiltInFuncs).Parse(string(releaseNotesTemplContent)))
	HomebrewCaskTemplate = template.Must(template.New("homebrew-cask").Funcs(templ.BuiltInFuncs).Parse(string(homebrewCaskTemplContent)))
}
