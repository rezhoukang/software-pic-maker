package core

import (
	"math"
	"testing"
)

func TestTableHeadersAlignAndLabelsHaveSeparateSpace(t *testing.T) {
	d := Templates()[1]
	s := Generate(d)
	for _, n := range s.Nodes {
		if n.Y != 64 {
			t.Fatal("table headers should align despite different field counts", n)
		}
	}
	for _, e := range s.Edges {
		if len(e.Points) != 2 || math.Abs(e.Points[0].Y-e.Points[1].Y) > .01 {
			t.Fatal("aligned table headers should have a straight relation", e)
		}
		if e.LabelPosition == nil || e.LabelPosition.Y >= e.Points[0].Y {
			t.Fatal("field label should sit above the connector", e)
		}
		for _, p := range []*Point{e.SourceLabelPosition, e.TargetLabelPosition} {
			if p == nil || p.Y <= e.Points[0].Y {
				t.Fatal("cardinalities should sit below the connector", e)
			}
		}
	}
}
