package core

import (
	"math"
	"reflect"
	"strings"
	"testing"
)

func sharedIncludeFixture() Diagram {
	return Diagram{Type: "usecase", View: View{Direction: "TB", Positions: map[string]Point{"a": {100, 100}, "b": {100, 250}, "c": {100, 400}, "shared": {500, 250}}}, Model: Model{UseCases: []Item{{ID: "a", Name: "操作甲"}, {ID: "b", Name: "操作乙"}, {ID: "c", Name: "操作丙"}, {ID: "shared", Name: "共同校验"}}, Relations: []Relation{{ID: "as", Source: "a", Target: "shared", Kind: "include"}, {ID: "bs", Source: "b", Target: "shared", Kind: "include"}, {ID: "cs", Source: "c", Target: "shared", Kind: "include"}}}}
}

func TestUseCaseSharedArrowsAndRotatedLabels(t *testing.T) {
	d := sharedIncludeFixture()
	before := raw(t, d)
	s := Generate(d)
	for id, p := range d.View.Positions {
		n := sceneNode(s, id)
		if n.X != p.X || n.Y != p.Y {
			t.Fatal("manual position changed", id)
		}
	}
	assertUseCaseEdgesAvoidNodes(t, s)
	if !reflect.DeepEqual(before, raw(t, d)) || !reflect.DeepEqual(s, Generate(d)) {
		t.Fatal("render mutated the model or is not deterministic")
	}
	for i, e := range s.Edges {
		if len(e.Points) != 2 || e.LabelPosition == nil {
			t.Fatal("missing straight line or label", e.ID)
		}
		a, b := e.Points[0], e.Points[1]
		for _, other := range s.Edges[i+1:] {
			if architectureOverlap(a, b, other.Points[0], other.Points[1]) > .01 {
				t.Fatal("shared include trunk")
			}
			if math.Hypot(b.X-other.Points[1].X, b.Y-other.Points[1].Y) < 18 {
				t.Fatal("arrowheads crowd the same landing point", e.ID, other.ID)
			}
		}
		angle := useCaseLabelAngle(a, b)
		if math.Abs(e.LabelAngle-angle) > .001 || math.Abs(e.LabelAngle) > 85 {
			t.Fatal("label does not follow its connector", e.ID)
		}
		length := math.Hypot(b.X-a.X, b.Y-a.Y)
		p := *e.LabelPosition
		distance := math.Abs((b.X-a.X)*(p.Y-a.Y)-(b.Y-a.Y)*(p.X-a.X)) / length
		if distance < 14 || distance > 24 {
			t.Fatal("label drifted away from its own line", e.ID, distance)
		}
		box := rotatedLabel{p, labelWidth(e.Label) + 6, 24, e.LabelAngle}
		for _, other := range s.Edges {
			if box.crosses(other.Points[0], other.Points[1]) {
				t.Fatal("rotated text hides a connector", e.ID, other.ID)
			}
		}
		if !strings.Contains(SVG(s), `rotate(`+num(angle)+` `) || !strings.Contains(DrawIO(s), `rotation=`+num(angle)+`;`) {
			t.Fatal("label rotation missing from export", e.ID)
		}
	}
	assertXML(t, SVG(s))
	assertXML(t, DrawIO(s))
}

func TestUseCaseLabelAnglesRemainReadableWithin85Degrees(t *testing.T) {
	for _, p := range []Point{{0, 200}, {0, -200}, {-200, 80}, {-200, -80}, {200, 80}, {200, -80}} {
		s := Scene{Type: "usecase", Edges: []Edge{{ID: "angle", Points: []Point{{300, 300}, {300 + p.X, 300 + p.Y}}, Label: "«extend»"}}}
		placeEdgeLabels(&s)
		if math.Abs(s.Edges[0].LabelAngle) > 85 || s.Edges[0].LabelPosition == nil {
			t.Fatal("unreadable label angle", p)
		}
		if p.X == 0 && math.Abs(s.Edges[0].LabelAngle) != 85 {
			t.Fatal("vertical label must clamp at 85 degrees")
		}
	}
}
