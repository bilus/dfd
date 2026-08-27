package ast_test

import (
	"testing"

	"github.com/bilus/dfd/ast"
)

func TestNewValidDiagram(t *testing.T) {
	d, err := ast.New([]ast.Step{{Title: "A"}, {Title: "B", In: "label"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(d.Steps) != 2 || d.Steps[1].In != "label" {
		t.Fatalf("unexpected diagram: %+v", d)
	}
}

func TestNewRejectsInvalid(t *testing.T) {
	cases := []struct {
		name  string
		steps []ast.Step
	}{
		{"empty", nil},
		{"first step with incoming label", []ast.Step{{Title: "A", In: "x"}}},
		{"store link without arrows", []ast.Step{{Title: "A", Stores: []ast.StoreLink{{Name: "S"}}}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ast.New(c.steps); err == nil {
				t.Fatal("want error, got nil")
			}
		})
	}
}

func TestNewStoreLink(t *testing.T) {
	if _, err := ast.NewStoreLink("S", nil, nil); err == nil {
		t.Fatal("want error for zero-arrow store link")
	}
	l, err := ast.NewStoreLink("S", &ast.Arrow{Label: "in"}, nil)
	if err != nil {
		t.Fatalf("NewStoreLink: %v", err)
	}
	if l.Put == nil || l.Put.Label != "in" || l.Get != nil {
		t.Fatalf("unexpected link: %+v", l)
	}
}

func TestStepKind(t *testing.T) {
	d, err := ast.New([]ast.Step{
		{Title: "Client", Kind: ast.Entity},
		{Title: "Handle", In: "req"},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if d.Steps[0].Kind != ast.Entity {
		t.Error("entity kind lost")
	}
	if d.Steps[1].Kind != ast.Process {
		t.Error("the zero kind must be Process, so existing callers stay right")
	}
}

func TestNewRejectsAStoreOnAnEntity(t *testing.T) {
	put := &ast.Arrow{Label: "x"}
	_, err := ast.New([]ast.Step{{
		Title:  "Client",
		Kind:   ast.Entity,
		Stores: []ast.StoreLink{{Name: "S", Put: put}},
	}})
	if err == nil {
		t.Fatal("want error: data cannot flow between an external entity and a datastore")
	}
}
