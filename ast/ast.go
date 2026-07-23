// Package ast defines the validated in-memory form of a dfd document:
// a strictly linear sequence of process steps with attached datastores.
package ast

import "fmt"

// Diagram is a linear flow. Construct with New so invariants hold.
type Diagram struct{ Steps []Step }

// Step is one process box. In labels the flow arrow from the previous
// step ("" = unlabeled); it is meaningless on the first step and must be
// empty there.
type Step struct {
	Title  string
	In     string
	Stores []StoreLink
}

// StoreLink attaches a datastore to a step. At least one of Put (step
// sends to store) and Get (store returns to step) is non-nil.
type StoreLink struct {
	Name     string
	Put, Get *Arrow
}

type Arrow struct{ Label string }

func New(steps []Step) (*Diagram, error) {
	if len(steps) == 0 {
		return nil, fmt.Errorf("no processes found")
	}
	if steps[0].In != "" {
		return nil, fmt.Errorf("arrow has no source process")
	}
	for _, s := range steps {
		for _, l := range s.Stores {
			if l.Put == nil && l.Get == nil {
				return nil, fmt.Errorf("datastore %q has no arrows", l.Name)
			}
		}
	}
	return &Diagram{Steps: steps}, nil
}

func NewStoreLink(name string, put, get *Arrow) (StoreLink, error) {
	if put == nil && get == nil {
		return StoreLink{}, fmt.Errorf("datastore %q has no arrows; add > and/or < lines before it", name)
	}
	return StoreLink{Name: name, Put: put, Get: get}, nil
}
