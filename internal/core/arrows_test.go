package core

import (
	"math"
	"strings"
	"testing"
)

func TestPaintedShapeAttachments(t *testing.T) {
	for _, shape := range []string{"rect", "table", "entity", "participant", "ellipse", "attribute", "keyAttribute", "diamond", "terminal", "database", "activation", "junction"} {
		n := Node{X: 100, Y: 100, Width: 220, Height: 74, Shape: shape}
		if shape == "activation" {
			n.Width = 12
		}
		if shape == "junction" {
			n.Width = 8
			n.Height = 8
		}
		for _, side := range []string{"left", "right", "top", "bottom"} {
			for _, ratio := range []float64{.1, .5, .9} {
				p := shapePort(n, side, ratio)
				q := shapeBoundary(n, p)
				if math.Hypot(p.X-q.X, p.Y-q.Y) > .01 {
					t.Errorf("%s %s %.1f: port %+v not on painted boundary %+v", shape, side, ratio, p, q)
				}
			}
		}
	}
}

func TestClassDependenciesHaveSeparateHeaderPortsAndLabels(t *testing.T) {
	d := Templates()[5]
	s := Generate(d)
	a, b := s.Edges[0], s.Edges[1]
	if a.Points[0] == b.Points[0] {
		t.Fatal("independent dependencies share source port")
	}
	for _, e := range s.Edges {
		for _, pair := range []struct {
			id string
			p  Point
		}{{e.Source, e.Points[0]}, {e.Target, e.Points[len(e.Points)-1]}} {
			n := sceneNode(s, pair.id)
			if pair.p.Y < n.Y || pair.p.Y > n.Y+44 {
				t.Fatal("dependency attaches to attribute/method compartment", e.ID, pair)
			}
		}
		if e.Arrow != "open" || !e.Dashed {
			t.Fatal("dependency UML style lost", e)
		}
		if e.LabelPosition == nil {
			t.Fatal("missing explicit label position")
		}
	}
	// In particular the two vertical elbows must not overlap as in the screenshot.
	for i := 1; i < len(a.Points); i++ {
		for j := 1; j < len(b.Points); j++ {
			p, q, r, z := a.Points[i-1], a.Points[i], b.Points[j-1], b.Points[j]
			if p.X == q.X && r.X == z.X && p.X == r.X && math.Max(math.Min(p.Y, q.Y), math.Min(r.Y, z.Y)) < math.Min(math.Max(p.Y, q.Y), math.Max(r.Y, z.Y)) {
				t.Fatal("dependencies share vertical trunk")
			}
		}
	}
}

func TestBoxRelationsReverseSelfIncomingAndBlocker(t *testing.T) {
	for _, direction := range []string{"LR", "TB"} {
		for _, positions := range []map[string]Point{
			{"a": {100, 200}, "b": {620, 90}, "c": {620, 450}, "block": {390, 230}},
			{"a": {820, 250}, "b": {80, 80}, "c": {80, 500}, "block": {430, 230}},
			{"a": {200, 80}, "b": {200, 600}, "c": {650, 600}, "block": {240, 340}},
		} {
			d := Diagram{Type: "class", Model: Model{Classes: []Item{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"}, {ID: "block", Name: "Blocker"}}, Relations: []Relation{{ID: "ab", Source: "a", Target: "b", Kind: "inheritance"}, {ID: "cb", Source: "c", Target: "b", Kind: "dependency"}, {ID: "aa", Source: "a", Target: "a", Kind: "association"}}}, View: View{Direction: direction, Positions: positions}}
			s := Generate(d)
			for _, e := range s.Edges {
				if e.ID == "ab" && (e.Arrow != "triangle" || e.Dashed) {
					t.Fatal("inheritance marker")
				}
				if e.ID == "aa" && e.Arrow != "none" {
					t.Fatal("association marker")
				}
				for _, pair := range []struct {
					id string
					p  Point
				}{{e.Source, e.Points[0]}, {e.Target, e.Points[len(e.Points)-1]}} {
					n := sceneNode(s, pair.id)
					q := shapeBoundary(n, pair.p)
					if math.Hypot(q.X-pair.p.X, q.Y-pair.p.Y) > .01 {
						t.Fatal("detached endpoint", e.ID, pair)
					}
				}
				for j := 1; j < len(e.Points); j++ {
					p, q := e.Points[j-1], e.Points[j]
					if p.X != q.X && p.Y != q.Y {
						t.Fatal("nonorthogonal relation", e.ID)
					}
					for _, n := range s.Nodes {
						if n.ID == e.Source && j == 1 || n.ID == e.Target && j == len(e.Points)-1 {
							continue
						}
						box := routeBox{left: n.X, right: n.X + n.Width, top: n.Y, bottom: n.Y + n.Height}
						if box.crosses(p, q) {
							t.Fatal("relation crosses node", e.ID, n.ID, p, q, positions, e.Points)
						}
					}
				}
				last, prior := e.Points[len(e.Points)-1], e.Points[len(e.Points)-2]
				if math.Hypot(last.X-prior.X, last.Y-prior.Y) < 14 {
					t.Fatal("arrowhead has insufficient final segment", e.ID)
				}
			}
		}
	}
}

func TestAllEightTypesLabelsAndArrowExportSemantics(t *testing.T) {
	for _, d := range Templates() {
		s := Generate(d)
		svg, drawio := SVG(s), DrawIO(s)
		labels := []routeBox{}
		for _, e := range s.Edges {
			if len(e.Points) < 2 {
				t.Fatal("empty edge", d.Type, e.ID)
			}
			if e.Arrow != "none" && !strings.Contains(svg, `marker-end="url(#`+e.Arrow+`)"`) {
				t.Fatal("missing SVG marker", d.Type, e.ID)
			}
			if e.Arrow == "triangle" && !strings.Contains(drawio, "endArrow=block;endFill=0;") {
				t.Fatal("drawio inheritance marker")
			}
			for _, v := range []struct {
				text string
				p    *Point
			}{{e.Label, e.LabelPosition}, {e.SourceLabel, e.SourceLabelPosition}, {e.TargetLabel, e.TargetLabelPosition}} {
				if v.text == "" {
					continue
				}
				if v.p == nil {
					t.Fatal("unplaced label", d.Type, e.ID)
				}
				box := routeBox{left: v.p.X - labelWidth(v.text)/2, right: v.p.X + labelWidth(v.text)/2, top: v.p.Y - 10, bottom: v.p.Y + 10}
				for _, n := range s.Nodes {
					if n.Shape == "boundary" {
						continue
					}
					if box.left < n.X+n.Width && box.right > n.X && box.top < n.Y+n.Height && box.bottom > n.Y {
						t.Errorf("%s %s label %s overlaps node %s", d.Type, e.ID, v.text, n.ID)
					}
				}
				for _, other := range labels {
					if box.left < other.right && box.right > other.left && box.top < other.bottom && box.bottom > other.top {
						t.Errorf("%s %s label overlap", d.Type, e.ID)
					}
				}
				labels = append(labels, box)
			}
		}
	}
}

func TestSequenceNestedSelfCallAndReturnPaintedActivationSides(t *testing.T) {
	d := Diagram{Type: "sequence", Model: Model{Participants: []Item{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}, Messages: []Message{{ID: "call", Source: "a", Target: "b", Label: "call"}, {ID: "self", Source: "b", Target: "b", Label: "nested"}, {ID: "self-return", Source: "b", Target: "b", Label: "done", Kind: "return"}, {ID: "return", Source: "b", Target: "a", Label: "done", Kind: "return"}}}, View: View{Positions: map[string]Point{}}}
	s := Generate(d)
	outer, inner := sceneNode(s, "$activation:call"), sceneNode(s, "$activation:self")
	if inner.X != outer.X+6 {
		t.Fatal("nested activation not offset")
	}
	if outer.Y+outer.Height != 412 || inner.Y+inner.Height != 334 {
		t.Fatal("nested returns matched wrong activation", outer, inner)
	}
	for _, e := range s.Edges {
		if e.ID == "call" {
			end := e.Points[len(e.Points)-1]
			if end.X != outer.X+1 {
				t.Fatal("call gap")
			}
		}
		if e.ID == "self" {
			end := e.Points[len(e.Points)-1]
			if end.X != inner.X+inner.Width-1 || end.Y != inner.Y {
				t.Fatal("self call misses activation", end, inner)
			}
		}
	}
}
