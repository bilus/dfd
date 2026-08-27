package theme_test

import (
	"strings"
	"testing"

	"github.com/bilus/dfd/theme"
)

func TestLookupDefaultKeepsOneUniformTextStyle(t *testing.T) {
	th, err := theme.Lookup("default", 13)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if th.Name != "default" {
		t.Errorf("name = %q", th.Name)
	}
	for _, r := range []theme.Role{theme.Title, theme.Label, theme.StoreName} {
		st := th.Style(r)
		if st.Size != 13 {
			t.Errorf("role %v size = %g, want the base size 13", r, st.Size)
		}
		if st.LineH() != 17 {
			t.Errorf("role %v line height = %d, want 17 (the historical constant)", r, st.LineH())
		}
		if st.Weight != 0 {
			t.Errorf("role %v weight = %d, want 0 (unstyled)", r, st.Weight)
		}
	}
	if th.LabelChip {
		t.Error("default must not draw label chips")
	}
	if th.AccentW != 0 {
		t.Error("default must not draw accent bars")
	}
	if th.BoxRadius != 0 {
		t.Error("default boxes must have square corners")
	}
}

func TestLookupPlexScalesRolesFromTheBaseSize(t *testing.T) {
	th, err := theme.Lookup("plex", 13)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got := th.Style(theme.Title).Size; got != 15 {
		t.Errorf("title size = %g, want base+2", got)
	}
	if got := th.Style(theme.Label).Size; got != 12 {
		t.Errorf("label size = %g, want base-1", got)
	}
	if got := th.Style(theme.Title).Weight; got != 600 {
		t.Errorf("title weight = %d, want 600", got)
	}
	if !strings.Contains(th.Style(theme.Label).Family, "Mono") {
		t.Errorf("label family = %q, want a monospace stack", th.Style(theme.Label).Family)
	}
	if !th.LabelChip {
		t.Error("plex must draw label chips")
	}
	if th.AccentW != 3 || th.BoxRadius != 4 {
		t.Errorf("accent %d radius %d, want 3 and 4", th.AccentW, th.BoxRadius)
	}
}

func TestPlexScalesWithBaseSize(t *testing.T) {
	th, err := theme.Lookup("plex", 20)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got := th.Style(theme.Title).Size; got != 22 {
		t.Errorf("title size = %g, want 22 for base 20", got)
	}
}

func TestEveryThemeSuppliesAFaceForEveryRole(t *testing.T) {
	for _, name := range theme.Names() {
		th, err := theme.Lookup(name, 13)
		if err != nil {
			t.Fatalf("Lookup(%q): %v", name, err)
		}
		for _, r := range []theme.Role{theme.Title, theme.Label, theme.StoreName} {
			if th.Style(r).Face == nil {
				t.Errorf("theme %q role %v has no face; layout cannot measure it", name, r)
			}
		}
	}
}

func TestLookupUnknown(t *testing.T) {
	if _, err := theme.Lookup("nope", 13); err == nil {
		t.Fatal("want error for unknown theme")
	}
}

func TestNames(t *testing.T) {
	got := strings.Join(theme.Names(), ",")
	if got != "default,plex" {
		t.Errorf("Names() = %q, want default,plex in that order", got)
	}
}
