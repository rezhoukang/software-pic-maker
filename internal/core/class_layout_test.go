package core

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

func classBranchFixture() Diagram {
	items := []Item{}
	for i, count := range []int{7, 9, 3, 6, 3, 7, 5} {
		item := Item{ID: fmt.Sprintf("n%d", i), Name: fmt.Sprintf("Class%d", i)}
		for j := 0; j < count; j++ {
			item.Fields = append(item.Fields, Field{Name: fmt.Sprintf("field%d", j), Type: "string"})
		}
		items = append(items, item)
	}
	d := Diagram{Type: "class", View: View{Direction: "LR", Positions: map[string]Point{}}, Model: Model{Classes: items}}
	d.Model.Classes[0].Methods = []string{"Items(): []Item"}
	for i, pair := range [][2]string{{"n0", "n1"}, {"n0", "n2"}, {"n1", "n3"}, {"n3", "n4"}, {"n1", "n5"}, {"n1", "n6"}} {
		d.Model.Relations = append(d.Model.Relations, Relation{ID: fmt.Sprintf("r%d", i), Source: pair[0], Target: pair[1], Kind: "association", Label: "字段引用", SourceCardinality: "1", TargetCardinality: "0..N"})
	}
	return d
}

func assertClassSidePorts(t *testing.T, s Scene) {
	t.Helper()
	for _, e := range s.Edges {
		for _, end := range []struct {
			id string
			p  Point
		}{{e.Source, e.Points[0]}, {e.Target, e.Points[len(e.Points)-1]}} {
			n := sceneNode(s, end.id)
			if math.Abs(end.p.X-(n.X+1)) > .01 && math.Abs(end.p.X-(n.X+n.Width-1)) > .01 {
				t.Fatal("class connector attached to top/bottom", e.ID, end)
			}
		}
	}
}

func TestClassAutomaticLayoutBalancesBranchesAndAvoidsCrossings(t *testing.T) {
	d := classBranchFixture()
	before := raw(t, d)
	s := Generate(d)
	if !reflect.DeepEqual(before, raw(t, d)) || !reflect.DeepEqual(s, Generate(d)) {
		t.Fatal("class layout mutated input or is nondeterministic")
	}
	if len(s.Nodes) != 7 || len(s.Edges) != 6 || sceneNode(s, "n6").X >= sceneNode(s, "n1").X {
		t.Fatal("leaf branches were all pushed to one side")
	}
	assertClassSidePorts(t, s)
	for i, a := range s.Edges {
		if len(a.Points) > 4 {
			t.Fatal("unnecessary bends", a.ID, a.Points)
		}
		for j := 1; j < len(a.Points); j++ {
			p, q := a.Points[j-1], a.Points[j]
			if math.Abs(p.X-q.X) > .001 && math.Abs(p.Y-q.Y) > .001 {
				t.Fatal("diagonal class connector")
			}
			for _, n := range s.Nodes {
				if n.ID == a.Source && j == 1 || n.ID == a.Target && j == len(a.Points)-1 {
					continue
				}
				if (routeBox{left: n.X, right: n.X + n.Width, top: n.Y, bottom: n.Y + n.Height}).crosses(p, q) {
					t.Fatal("connector crosses class", a.ID, n.ID)
				}
			}
			for _, b := range s.Edges[i+1:] {
				for k := 1; k < len(b.Points); k++ {
					if architectureIntersects(p, q, b.Points[k-1], b.Points[k]) || architectureOverlap(p, q, b.Points[k-1], b.Points[k]) > .01 {
						t.Fatal("class connectors cross or overlap", a.ID, b.ID, a.Points, b.Points)
					}
				}
			}
		}
	}
	assertXML(t, SVG(s))
	assertXML(t, DrawIO(s))
}

func TestClassSavedPositionsAndSelfRelationsKeepSidePorts(t *testing.T) {
	for _, direction := range []string{"LR", "TB"} {
		d := classBranchFixture()
		d.View.Direction = direction
		d.View.Positions = map[string]Point{"n0": {100, 100}, "n1": {600, 100}, "n2": {100, 600}, "n3": {600, 600}, "n4": {1100, 600}, "n5": {1100, 100}, "n6": {600, 1100}}
		d.Model.Relations = append(d.Model.Relations, Relation{ID: "self", Source: "n0", Target: "n0"}, Relation{ID: "reverse", Source: "n5", Target: "n0", Kind: "dependency"})
		s := Generate(d)
		assertClassSidePorts(t, s)
		for id, p := range d.View.Positions {
			n := sceneNode(s, id)
			if n.X != p.X || n.Y != p.Y {
				t.Fatal("saved class position was moved", id)
			}
		}
	}
}

func TestClassTallDescendantsStayWithinTheirAllocatedBranches(t *testing.T) {
	d := classBranchFixture()
	d.Model.Classes[0].Fields = nil
	d.Model.Classes[0].Methods = nil
	for _, id := range []int{2, 4} {
		for i := 0; i < 18; i++ {
			d.Model.Classes[id].Fields = append(d.Model.Classes[id].Fields, Field{Name: fmt.Sprintf("extra%d", i), Type: "string"})
		}
	}
	s := Generate(d)
	assertClassSidePorts(t, s)
	for i, a := range s.Nodes {
		for _, b := range s.Nodes[i+1:] {
			if a.X < b.X+b.Width && a.X+a.Width > b.X && a.Y < b.Y+b.Height && a.Y+a.Height > b.Y {
				t.Fatal("class branches overlap", a.ID, b.ID)
			}
		}
	}
}

func TestClassOverlappingHeadersConnectWithoutTinyElbows(t *testing.T) {
	d := Templates()[5]
	before := raw(t, d)
	s := Generate(d)
	assertClassSidePorts(t, s)
	for _, e := range s.Edges {
		if len(e.Points) != 2 || math.Abs(e.Points[0].Y-e.Points[1].Y) > .001 {
			t.Fatal("overlapping headers were connected with an unnecessary elbow", e.ID, e.Points)
		}
		for _, end := range []struct {
			id string
			p  Point
		}{{e.Source, e.Points[0]}, {e.Target, e.Points[1]}} {
			n := sceneNode(s, end.id)
			if end.p.Y < n.Y+8 || end.p.Y > n.Y+40 {
				t.Fatal("straightening escaped the header", e.ID)
			}
		}
	}
	if !reflect.DeepEqual(before, raw(t, d)) {
		t.Fatal("straightening changed design data")
	}
}
