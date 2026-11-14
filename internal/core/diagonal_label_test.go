package core

import "testing"

func TestDiagonalRelationshipLabelsClearTheWholeTextBox(t *testing.T) {
	for _, points := range [][]Point{
		{{100, 100}, {500, 240}}, {{500, 240}, {100, 100}},
		{{100, 240}, {500, 100}}, {{500, 100}, {100, 240}},
		{{100, 100}, {170, 450}}, {{170, 450}, {100, 100}},
	} {
		s := Scene{Edges: []Edge{{Points: points, Label: "«extend»"}}}
		placeEdgeLabels(&s)
		p := s.Edges[0].LabelPosition
		// Reserve space around all glyphs, not just the text's center.
		box := routeBox{left: p.X - labelWidth("«extend»")/2 - 4, right: p.X + labelWidth("«extend»")/2 + 4, top: p.Y - 15, bottom: p.Y + 15}
		if box.crosses(points[0], points[1]) {
			t.Fatal("diagonal still crosses the label and its padding", points, p)
		}
	}
	s := Generate(Templates()[2])
	for _, e := range s.Edges {
		if e.Label != "«extend»" && e.Label != "«include»" {
			continue
		}
		p := e.LabelPosition
		box := routeBox{left: p.X - labelWidth(e.Label)/2, right: p.X + labelWidth(e.Label)/2, top: p.Y - 11, bottom: p.Y + 11}
		for _, other := range s.Edges {
			for i := 1; i < len(other.Points); i++ {
				if box.crosses(other.Points[i-1], other.Points[i]) {
					t.Fatal("use-case label crosses a relation", e.ID, other.ID, e.Points, p)
				}
			}
		}
	}
}
