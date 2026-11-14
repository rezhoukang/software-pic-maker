package core

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

func architectureFixture(direction string) Diagram {
	return Diagram{SchemaVersion: "1.0", ID: "architecture-regression", Type: "architecture", Title: "共享组件架构", View: View{Theme: "mono", Direction: direction, Positions: map[string]Point{}}, Model: Model{
		Nodes: []Item{{ID: "inA", Name: "输入端 A"}, {ID: "inB", Name: "输入端 B"}, {ID: "serviceA", Name: "服务 A"}, {ID: "serviceB", Name: "服务 B"}, {ID: "sharedA", Name: "共享组件 A"}, {ID: "sharedB", Name: "共享组件 B"}, {ID: "sharedC", Name: "共享组件 C"}, {ID: "local", Name: "本地渲染器"}, {ID: "output", Name: "输出文件"}},
		Relations: []Relation{
			{ID: "r1", Source: "inA", Target: "serviceA", Label: "界面绑定"}, {ID: "r2", Source: "inB", Target: "serviceB", Label: "协议请求"},
			{ID: "r3", Source: "serviceA", Target: "sharedA"}, {ID: "r4", Source: "serviceB", Target: "sharedA"},
			{ID: "r5", Source: "serviceA", Target: "sharedB"}, {ID: "r6", Source: "serviceB", Target: "sharedB"},
			{ID: "r7", Source: "serviceA", Target: "sharedC"}, {ID: "r8", Source: "serviceB", Target: "sharedC"},
			{ID: "r9", Source: "serviceB", Target: "local", Label: "图像渲染"}, {ID: "r10", Source: "sharedB", Target: "output", Label: "设计数据"}, {ID: "r11", Source: "sharedC", Target: "output", Label: "导出文件"},
		},
	}}
}

func TestArchitectureContinuationStaysStraightBesideWideSibling(t *testing.T) {
	for _, direction := range []string{"TB", "LR"} {
		for _, wide := range []bool{false, true} {
			d := Templates()[6]
			d.View.Direction = direction
			if wide {
				d.Model.Nodes[2].Name = strings.Repeat("1", 60) + strings.Repeat("2", 30)
			}
			before := raw(t, d)
			s := Generate(d)
			assertArchitectureGeometry(t, d, s)
			for _, id := range []string{"a1", "a3", "a4"} {
				found := false
				for _, e := range s.Edges {
					if e.ID == id {
						found = true
						if len(e.Points) != 2 {
							t.Fatal("continuation chain has an unnecessary elbow", direction, wide, id, e.Points)
						}
						for _, p := range e.Points {
							anchor := center(sceneNode(s, "client"))
							difference := math.Abs(p.X - anchor.X)
							if direction == "LR" {
								difference = math.Abs(p.Y - anchor.Y)
							}
							if difference > .001 {
								t.Fatal("continuation chain left its common axis", direction, wide, id, e.Points)
							}
						}
					}
				}
				if !found {
					t.Fatal("missing relation", id)
				}
			}
			if !reflect.DeepEqual(before, raw(t, d)) || !reflect.DeepEqual(s, Generate(d)) {
				t.Fatal("layout mutated input or is nondeterministic")
			}
		}
	}
}

func assertArchitectureGeometry(t *testing.T, d Diagram, s Scene) {
	t.Helper()
	if len(s.Nodes) != len(d.Model.Nodes) || len(s.Edges) != len(d.Model.Relations) {
		t.Fatal("layout lost model nodes or relations")
	}
	for _, e := range s.Edges {
		if len(e.Points) < 2 || len(e.Points) > 4 {
			t.Fatal("missing route", e.ID)
		}
		for _, end := range []struct {
			id    string
			point Point
		}{{e.Source, e.Points[0]}, {e.Target, e.Points[len(e.Points)-1]}} {
			n := sceneNode(s, end.id)
			boundary := shapeBoundary(n, end.point)
			if math.Hypot(boundary.X-end.point.X, boundary.Y-end.point.Y) > .02 {
				t.Fatal("endpoint is detached from painted outline", e.ID, end)
			}
		}
		for j := 1; j < len(e.Points); j++ {
			a, b := e.Points[j-1], e.Points[j]
			if math.Abs(a.X-b.X) > .001 && math.Abs(a.Y-b.Y) > .001 {
				t.Fatal("nonorthogonal architecture route", e.ID)
			}
			for _, n := range s.Nodes {
				if n.ID == e.Source && j == 1 || n.ID == e.Target && j == len(e.Points)-1 {
					continue
				}
				if (routeBox{left: n.X, right: n.X + n.Width, top: n.Y, bottom: n.Y + n.Height}).crosses(a, b) {
					t.Fatal("route crosses component", e.ID, n.ID, e.Points)
				}
			}
		}
		last, prior := e.Points[len(e.Points)-1], e.Points[len(e.Points)-2]
		if math.Hypot(last.X-prior.X, last.Y-prior.Y) < 14 {
			t.Fatal("insufficient final arrow segment", e.ID)
		}
	}
	for i, a := range s.Edges {
		for _, b := range s.Edges[i+1:] {
			for j := 1; j < len(a.Points); j++ {
				for k := 1; k < len(b.Points); k++ {
					if architectureOverlap(a.Points[j-1], a.Points[j], b.Points[k-1], b.Points[k]) > .01 {
						t.Fatal("independent relations share a trunk", a.ID, b.ID, a.Points, b.Points)
					}
				}
			}
		}
	}
}

func TestArchitectureSharedComponentsHaveIndependentRoutes(t *testing.T) {
	for _, direction := range []string{"LR", "TB"} {
		t.Run(direction, func(t *testing.T) {
			d := architectureFixture(direction)
			before := raw(t, d)
			s := Generate(d)
			assertArchitectureGeometry(t, d, s)
			crossings := 0
			for i, edge := range s.Edges {
				source, target := sceneNode(s, edge.Source), sceneNode(s, edge.Target)
				start, end := edge.Points[0], edge.Points[len(edge.Points)-1]
				if direction == "LR" && (start.X < source.X+source.Width-2 || end.X > target.X+2) || direction == "TB" && (start.Y < source.Y+source.Height-2 || end.Y > target.Y+2) {
					t.Fatal("forward links mixed incoming/outgoing faces", edge.ID, edge.Points)
				}
				for _, other := range s.Edges[i+1:] {
					intersects := false
					for j := 1; j < len(edge.Points); j++ {
						for k := 1; k < len(other.Points); k++ {
							intersects = intersects || architectureIntersects(edge.Points[j-1], edge.Points[j], other.Points[k-1], other.Points[k])
						}
					}
					if intersects {
						crossings++
					}
				}
			}
			// The two left services both connect to three right components. Their
			// fixed left-to-right order requires three crossings; avoid adding more.
			if crossings > 3 {
				t.Fatal("unnecessary connector crossings", crossings)
			}
			if !reflect.DeepEqual(before, raw(t, d)) {
				t.Fatal("layout changed the model or saved positions")
			}
			if !reflect.DeepEqual(s, Generate(d)) {
				t.Fatal("layout is nondeterministic")
			}
			assertXML(t, SVG(s))
			assertXML(t, DrawIO(s))
		})
	}
}

func TestArchitectureFeedbackAndLongLabelsKeepTheirRelations(t *testing.T) {
	d := architectureFixture("LR")
	d.Model.Relations[0].Label = "携带设计文件版本的本地界面调用"
	d.Model.Relations = append(d.Model.Relations, Relation{ID: "feedback", Source: "output", Target: "serviceA", Kind: "return", Label: "处理结果"}, Relation{ID: "self", Source: "serviceB", Target: "serviceB", Label: "内部检查"})
	s := Generate(d)
	assertArchitectureGeometry(t, d, s)
	if sceneNode(s, "output").X <= sceneNode(s, "serviceB").X {
		t.Fatal("feedback collapsed forward layers")
	}
	for _, e := range s.Edges {
		if e.Label == "" {
			continue
		}
		if e.LabelPosition == nil {
			t.Fatal("missing label position", e.ID)
		}
		box := routeBox{left: e.LabelPosition.X - labelWidth(e.Label)/2, right: e.LabelPosition.X + labelWidth(e.Label)/2, top: e.LabelPosition.Y - 10, bottom: e.LabelPosition.Y + 10}
		for _, n := range s.Nodes {
			if box.left < n.X+n.Width && box.right > n.X && box.top < n.Y+n.Height && box.bottom > n.Y {
				t.Fatal("label overlaps component", e.ID, n.ID)
			}
		}
	}
}

func TestArchitectureDenseFanOutDoesNotRequireManualCoordinates(t *testing.T) {
	d := architectureFixture("TB")
	for i := range 9 {
		id := fmt.Sprintf("additional%d", i)
		d.Model.Nodes = append(d.Model.Nodes, Item{ID: id, Name: fmt.Sprintf("新增共享组件 %d", i)})
		d.Model.Relations = append(d.Model.Relations, Relation{ID: id + "A", Source: "serviceA", Target: id}, Relation{ID: id + "B", Source: "serviceB", Target: id})
	}
	assertArchitectureGeometry(t, d, Generate(d))
}

func TestArchitectureUnannotatedCycleDoesNotCollapseItsLayers(t *testing.T) {
	d := architectureFixture("LR")
	d.Model.Relations = append(d.Model.Relations, Relation{ID: "cycle", Source: "output", Target: "serviceA", Label: "完成通知"})
	s := Generate(d)
	assertArchitectureGeometry(t, d, s)
	if sceneNode(s, "output").X <= sceneNode(s, "serviceA").X {
		t.Fatal("a normal cyclic relation collapsed the architecture layers")
	}
}

func TestArchitectureStraightConnectionsTakePrecedence(t *testing.T) {
	for _, target := range []Point{{600, 200}, {600, 220}, {100, 600}} {
		d := Diagram{Type: "architecture", Model: Model{Nodes: []Item{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}, Relations: []Relation{{ID: "ab", Source: "a", Target: "b"}}}, View: View{Direction: "LR", Positions: map[string]Point{"a": {100, 200}, "b": target}}}
		s := Generate(d)
		assertArchitectureGeometry(t, d, s)
		if len(s.Edges[0].Points) != 2 {
			t.Fatal("clear straight connection was given unnecessary elbows", s.Edges[0].Points)
		}
	}
}

func TestArchitectureDiagonalNeighborsUseRightAngles(t *testing.T) {
	d := Diagram{Type: "architecture", Model: Model{Nodes: []Item{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}, Relations: []Relation{{ID: "ab", Source: "a", Target: "b"}}}, View: View{Direction: "LR", Positions: map[string]Point{"a": {100, 200}, "b": {600, 400}}}}
	s := Generate(d)
	assertArchitectureGeometry(t, d, s)
	if len(s.Edges[0].Points) != 4 {
		t.Fatal("diagonal neighbors must use two right angles and separate input/output faces", s.Edges[0].Points)
	}
}

func TestArchitectureBlockedStraightRouteUsesTwoRightAngles(t *testing.T) {
	d := Diagram{Type: "architecture", Model: Model{Nodes: []Item{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "blocker", Name: "Blocker"}}, Relations: []Relation{{ID: "ab", Source: "a", Target: "b"}}}, View: View{Direction: "LR", Positions: map[string]Point{"a": {100, 200}, "b": {700, 200}, "blocker": {390, 200}}}}
	s := Generate(d)
	assertArchitectureGeometry(t, d, s)
	if len(s.Edges[0].Points) != 4 {
		t.Fatal("blocked straight route should use exactly two right angles", s.Edges[0].Points)
	}
}
