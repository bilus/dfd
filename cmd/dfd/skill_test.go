package main_test

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/bilus/dfd/internal/cli"
	"github.com/bilus/dfd/layout"
	"github.com/bilus/dfd/parse"
	"github.com/bilus/dfd/render"
	"github.com/bilus/dfd/theme"
)

const skillPath = "../../skills/dfd/SKILL.md"

func skillText(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("read skill: %v", err)
	}
	return string(b)
}

// Every diagram the skill shows must actually render, in both themes,
// so the reference cannot drift from the tool it documents.
func TestSkillExamplesRender(t *testing.T) {
	blocks := regexp.MustCompile("(?s)```dfd\n(.*?)```").FindAllStringSubmatch(skillText(t), -1)
	if len(blocks) < 2 {
		t.Fatalf("found %d dfd examples in the skill; it should show at least two", len(blocks))
	}
	for i, b := range blocks {
		src := b[1]
		for _, name := range theme.Names() {
			th, err := theme.Lookup(name, 13)
			if err != nil {
				t.Fatalf("theme: %v", err)
			}
			d, err := parse.Parse(strings.NewReader(src), "SKILL.md")
			if err != nil {
				t.Fatalf("example %d does not parse: %v\n%s", i+1, err, src)
			}
			s, err := layout.Arrange(d, layout.Config{BoxW: 160, BoxH: 60, FontSize: 13, Theme: th, Number: true})
			if err != nil {
				t.Fatalf("example %d does not lay out in %q: %v", i+1, name, err)
			}
			var buf bytes.Buffer
			if err := render.SVG(s, th, &buf); err != nil {
				t.Fatalf("example %d does not render in %q: %v", i+1, name, err)
			}
		}
	}
}

// Every flag the skill names must exist, so a renamed flag breaks the
// build rather than misleading a reader.
func TestSkillFlagsExist(t *testing.T) {
	var usage bytes.Buffer
	cli.Run([]string{"--help"}, strings.NewReader(""), io.Discard, &usage)
	for _, m := range regexp.MustCompile(`--[a-z][a-z-]+`).FindAllString(skillText(t), -1) {
		name := "-" + strings.TrimPrefix(m, "--")
		if !strings.Contains(usage.String(), name+" ") && !strings.Contains(usage.String(), name+"\n") {
			t.Errorf("skill names %s, which dfd does not accept:\n%s", m, usage.String())
		}
	}
}
