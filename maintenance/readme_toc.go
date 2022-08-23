package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

func main() {
	readmeFilename := "../README.md"
	readmeContent, err := os.ReadFile(readmeFilename)
	must(err)

	toc, err := createToc(string(readmeContent))
	must(err)

	fmt.Println(toc)
}

func createToc(s string) (string, error) {
	r := renderer.NewRenderer(renderer.WithNodeRenderers(util.Prioritized(&tocRenderer{}, 1000)))
	markdown := goldmark.New(
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRenderer(r),
	)

	var buff bytes.Buffer
	if err := markdown.Convert([]byte(s), &buff); err != nil {
		return "", err
	}

	return buff.String(), nil
}

type tocRenderer struct {
}

func (r *tocRenderer) renderText(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	parent := node.Parent()
	if !entering || parent.Kind() != ast.KindHeading {
		return ast.WalkContinue, nil
	}

	hn := parent.(*ast.Heading)

	idattr, ok := hn.AttributeString("id")
	if !ok {
		return ast.WalkSkipChildren, nil
	}

	id := string(idattr.([]byte))

	n := node.(*ast.Text)
	segment := n.Segment

	// Start at level 2.
	numIndentation := hn.Level - 2
	if numIndentation < 0 {
		return ast.WalkContinue, nil
	}

	fmt.Fprintf(w, "%s * [%s](#%s)\n", strings.Repeat(" ", numIndentation*4), string(segment.Value(source)), id)

	return ast.WalkContinue, nil
}

func (r *tocRenderer) renderNoop(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *tocRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// I'm sure there must simpler way of doing this ...
	reg.Register(ast.KindText, r.renderText)

	// Ignore everything else.
	reg.Register(ast.KindHeading, r.renderNoop)
	reg.Register(ast.KindDocument, r.renderNoop)
	reg.Register(ast.KindBlockquote, r.renderNoop)
	reg.Register(ast.KindCodeBlock, r.renderNoop)
	reg.Register(ast.KindFencedCodeBlock, r.renderNoop)
	reg.Register(ast.KindHTMLBlock, r.renderNoop)
	reg.Register(ast.KindList, r.renderNoop)
	reg.Register(ast.KindListItem, r.renderNoop)
	reg.Register(ast.KindParagraph, r.renderNoop)
	reg.Register(ast.KindTextBlock, r.renderNoop)
	reg.Register(ast.KindThematicBreak, r.renderNoop)
	reg.Register(ast.KindAutoLink, r.renderNoop)
	reg.Register(ast.KindCodeSpan, r.renderNoop)
	reg.Register(ast.KindEmphasis, r.renderNoop)
	reg.Register(ast.KindImage, r.renderNoop)
	reg.Register(ast.KindLink, r.renderNoop)
	reg.Register(ast.KindRawHTML, r.renderNoop)

	reg.Register(ast.KindString, r.renderNoop)

}

func (r *tocRenderer) AddOptions(...renderer.Option) {

}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
