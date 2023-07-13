package main

import (
	"testing"

	"azul3d.org/engine/binpack"
	"golang.org/x/exp/slices"
)

type test struct {
	id         int
	x, y, w, h int
}

func (t test) size() int {
	if t.w > t.h {
		return t.w
	}
	return t.h
}

type tests struct {
	w       int
	h       int
	entries []*test
}

func (t tests) Len() int {
	return len(t.entries)
}

func (t tests) Size(n int) (w, h int) {
	entry := t.entries[n]
	w = entry.w
	h = entry.h
	return
}

func (t tests) Place(n, x, y int) {
	entry := t.entries[n]
	entry.x = x
	entry.y = y
}

func (t *tests) add(v *test) {
	t.entries = append(t.entries, v)
}

func (t *tests) sort() {
	slices.SortFunc(t.entries, func(a, b *test) bool {
		return a.size() > b.size()
	})
}

func TestBinpack(t *testing.T) {
	tests := tests{
		w: 512,
		h: 512,
	}
	tests.add(&test{id: 1, w: 384, h: 16})
	tests.add(&test{id: 2, w: 196, h: 32})
	tests.add(&test{id: 3, w: 64, h: 96})
	tests.add(&test{id: 4, w: 256, h: 256})
	tests.sort()

	binpack.Pack(tests)

	for _, entry := range tests.entries {
		t.Log(entry)
	}
}
