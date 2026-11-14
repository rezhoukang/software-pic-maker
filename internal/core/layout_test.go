package core

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

func sceneNode(s Scene, id string) Node {
	for _, n := range s.Nodes {
		if n.ID == id {
			return n
		}
	}
	return Node{}
}

func TestFlowchartCentersAndStraightTrunk(t *testing.T) {
	for _, direction := range []string{"TB", "LR"} {
		d := Templates()[3]
		d.View.Direction = direction
		s := Generate(d)
		first := center(sceneNode(s, "start"))
		for _, id := range []string{"select", "check", "save", "finish"} {
			p := center(sceneNode(s, id))
			delta := p.X - first.X
			if direction == "LR" {
				delta = p.Y - first.Y
			}
			if math.Abs(delta) > 0.01 {
				t.Fatalf("%s trunk is not aligned: %s", direction, id)
			}
		}
		for _, e := range s.Edges {
			if e.ID == "f2" && len(e.Points) != 2 {
				t.Fatal("aligned trunk has a needless bend", e.Points)
			}
		}
	}
}

func TestFlowchartDecisionExitsAndSingleMergeArrow(t *testing.T) {
	for _, direction := range []string{"TB", "LR"} {
		d := Templates()[3]
		d.View.Direction = direction
		s := Generate(d)
		decision := sceneNode(s, "check")
		dc := center(decision)
		arrowsIntoFinish := 0
		for _, e := range s.Edges {
			if e.Target == "finish" && e.Arrow != "none" {
				arrowsIntoFinish++
			}
			if e.ID == "f2" {
				end := e.Points[len(e.Points)-1]
				if direction == "TB" && (end.X != dc.X || end.Y != decision.Y+2) {
					t.Fatal("arrow misses drawn top diamond vertex", end)
				}
				if direction == "LR" && (end.X != decision.X+2 || end.Y != dc.Y) {
					t.Fatal("arrow misses drawn left diamond vertex", end)
				}
			}
			if e.ID == "f3" || e.ID == "f4" {
				p := e.Points[0]
				if e.LabelPosition == nil {
					t.Fatal("decision label has no explicit position")
				}
				side := "bottom"
				if e.ID == "f4" {
					side = "right"
				}
				if direction == "LR" {
					side = "right"
					if e.ID == "f4" {
						side = "bottom"
					}
				}
				if p != shapePort(decision, side, .5) {
					t.Fatal("normal branch must follow the main axis and alternate branch the side", e.ID, p)
				}
			}
			if e.ID == "f5" || e.ID == "f6" {
				if e.Arrow != "none" || e.Target != "$junction:finish" {
					t.Fatal("incoming merge arrows overlap", e)
				}
			}
		}
		if arrowsIntoFinish != 1 {
			t.Fatal("finish must have exactly one arrow", arrowsIntoFinish)
		}
		junction := sceneNode(s, "$junction:finish")
		if junction.Shape != "junction" || junction.Movable {
			t.Fatal("missing automatic merge point")
		}
		if strings.Contains(NodeMarkup(junction), "circle") || !strings.Contains(DrawIO(s), "opacity=0") {
			t.Fatal("merge anchor must not draw a flowchart dot")
		}
		if !strings.Contains(DrawIO(s), `as="offset"`) {
			t.Fatal("branch label positions not exported")
		}
	}
}

func TestFlowchartShortMergeGapUsesDifferentTargetPorts(t *testing.T) {
	for _, direction := range []string{"TB", "LR"} {
		d := Templates()[3]
		d.View.Direction = direction
		base := Generate(d)
		finish := sceneNode(base, "finish")
		a, b := sceneNode(base, "save"), sceneNode(base, "reject")
		p := Point{finish.X, math.Max(a.Y+a.Height, b.Y+b.Height) + 16}
		if direction == "LR" {
			p = Point{math.Max(a.X+a.Width, b.X+b.Width) + 16, finish.Y}
		}
		d.View.Positions["finish"] = p
		s := Generate(d)
		ends := []Point{}
		for _, e := range s.Edges {
			if e.Target == "finish" && e.Arrow != "none" {
				ends = append(ends, e.Points[len(e.Points)-1])
			}
		}
		if len(ends) != 2 || math.Hypot(ends[0].X-ends[1].X, ends[0].Y-ends[1].Y) < 10 {
			t.Fatal("short-gap arrows share the same target port", direction, ends)
		}
	}
}

func TestUseCaseBoundaryIncludesTwelveActorsAndMovedCases(t *testing.T) {
	d := Templates()[2]
	for i := len(d.Model.Actors); i < 12; i++ {
		d.Model.Actors = append(d.Model.Actors, Item{ID: fmt.Sprintf("actor%d", i), Name: fmt.Sprintf("参与者%d", i)})
	}
	d.View.Positions["search"] = Point{1150, 1200}
	s := Generate(d)
	boundary := sceneNode(s, "$boundary")
	for _, n := range s.Nodes {
		if n.Shape == "actor" && (n.Y < boundary.Y || n.Y+n.Height > boundary.Y+boundary.Height) {
			t.Fatalf("actor %s extends past system boundary", n.ID)
		}
		if n.Shape == "ellipse" && (n.X < boundary.X || n.X+n.Width > boundary.X+boundary.Width || n.Y+n.Height > boundary.Y+boundary.Height) {
			t.Fatalf("use case %s outside boundary", n.ID)
		}
		if n.X+n.Width > s.Width || n.Y+n.Height > s.Height {
			t.Fatalf("node %s outside scene", n.ID)
		}
	}
}

func TestUseCaseStereotypesAndActorAnchors(t *testing.T) {
	d := Templates()[2]
	s := Generate(d)
	for _, id := range []string{"u6", "u7"} {
		found := false
		for _, e := range s.Edges {
			if e.ID == id {
				found = true
				if !e.Dashed || e.Arrow != "open" || !strings.HasPrefix(e.Label, "«") {
					t.Fatal("invalid UML stereotype", e)
				}
				if id == "u6" && (e.Source != "borrow" || e.Target != "eligibility") {
					t.Fatal("include direction reversed")
				}
				if id == "u7" && (e.Source != "renew" || e.Target != "borrow") {
					t.Fatal("extend direction reversed")
				}
			}
		}
		if !found {
			t.Fatalf("missing %s", id)
		}
	}
	for _, e := range s.Edges {
		if e.Source == "reader" {
			n := sceneNode(s, "reader")
			if math.Abs(e.Points[0].X-(n.X+n.Width/2+28)) > .01 {
				t.Fatal("association is detached from actor")
			}
		}
	}
	for _, output := range []string{SVG(s), DrawIO(s)} {
		if !strings.Contains(output, "«include»") || !strings.Contains(output, "«extend»") {
			t.Fatal("stereotype missing in export")
		}
	}
	if !strings.Contains(DrawIO(s), "exitPerimeter=0") {
		t.Fatal("draw.io endpoints are not preserved")
	}
}

func populatedUseCases() Diagram {
	d := Templates()[2]
	for i := len(d.Model.Actors); i < 12; i++ {
		d.Model.Actors = append(d.Model.Actors, Item{ID: fmt.Sprintf("actor%d", i+1), Name: fmt.Sprintf("参与者%d", i+1)})
	}
	for i := len(d.Model.UseCases); i < 19; i++ {
		d.Model.UseCases = append(d.Model.UseCases, Item{ID: fmt.Sprintf("case%d", i+1), Name: fmt.Sprintf("用例%d", i+1)})
	}
	return d
}
func assertUseCaseEdgesAvoidNodes(t *testing.T, s Scene) {
	t.Helper()
	for _, e := range s.Edges {
		for _, n := range s.Nodes {
			if n.Shape == "boundary" || n.ID == e.Source || n.ID == e.Target {
				continue
			}
			box := routeBox{left: n.X, right: n.X + n.Width, top: n.Y, bottom: n.Y + n.Height}
			for i := 1; i < len(e.Points); i++ {
				if box.crosses(e.Points[i-1], e.Points[i]) {
					t.Fatalf("edge %s crosses unrelated node %s", e.ID, n.ID)
				}
			}
		}
	}
}
func TestUseCaseNineteenCasesKeepRelationsLocalAndAvoidNodes(t *testing.T) {
	s := Generate(populatedUseCases())
	assertUseCaseEdgesAvoidNodes(t, s)
	for _, e := range s.Edges {
		if (e.ID == "u1" || e.ID == "u2" || e.ID == "u3" || e.ID == "u4" || e.ID == "u5") && len(e.Points) != 2 {
			t.Fatalf("simple actor fan routed with needless bends: %s", e.ID)
		}
		length := 0.0
		for i := 1; i < len(e.Points); i++ {
			length += math.Hypot(e.Points[i].X-e.Points[i-1].X, e.Points[i].Y-e.Points[i-1].Y)
		}
		if (e.ID == "u6" || e.ID == "u7") && length > 300 {
			t.Fatalf("related case separated by unrelated columns: %s %.0f", e.ID, length)
		}
		if (e.ID == "u4" || e.ID == "u5") && length > 300 {
			t.Fatalf("administrator too far from own cases: %s %.0f", e.ID, length)
		}
	}
	bottom := 0.0
	for _, id := range []string{"search", "borrow", "history", "manage", "review", "eligibility", "renew"} {
		n := sceneNode(s, id)
		bottom = math.Max(bottom, n.Y+n.Height)
	}
	for i := 8; i <= 19; i++ {
		n := sceneNode(s, fmt.Sprintf("case%d", i))
		if n.Y < bottom+40 {
			t.Fatal("unconnected case inserted into linked group", n.ID)
		}
	}
	for i, a := range s.Nodes {
		if a.Shape == "boundary" {
			continue
		}
		for _, b := range s.Nodes[i+1:] {
			if b.Shape == "boundary" {
				continue
			}
			if a.X < b.X+b.Width && a.X+a.Width > b.X && a.Y < b.Y+b.Height && a.Y+a.Height > b.Y {
				t.Fatalf("nodes overlap: %s / %s", a.ID, b.ID)
			}
		}
	}
}
func TestUseCaseRoutingAvoidsManuallyPlacedBlockers(t *testing.T) {
	d := Templates()[2]
	d.Model.UseCases = append(d.Model.UseCases, Item{ID: "blocker", Name: "中间的无关用例"})
	d.View.Positions["eligibility"] = Point{1200, 244}
	d.View.Positions["blocker"] = Point{650, 244}
	s := Generate(d)
	assertUseCaseEdgesAvoidNodes(t, s)
	for _, e := range s.Edges {
		if len(e.Points) != 2 {
			t.Fatal("use-case routing must move automatic nodes instead of bending", e.ID)
		}
	}
	if n := sceneNode(s, "blocker"); n.X != 650 || n.Y != 244 {
		t.Fatal("manual position was overwritten")
	}
}
func TestUseCaseDenseAssociationsAvoidEveryUnrelatedNode(t *testing.T) {
	d := populatedUseCases()
	for i := 19; i < 40; i++ {
		d.Model.UseCases = append(d.Model.UseCases, Item{ID: fmt.Sprintf("case%d", i+1), Name: fmt.Sprintf("用例%d", i+1)})
	}
	for i, n := range d.Model.UseCases {
		d.Model.Relations = append(d.Model.Relations, Relation{ID: fmt.Sprintf("extra%d", i), Source: d.Model.Actors[i%len(d.Model.Actors)].ID, Target: n.ID})
	}
	s := Generate(d)
	assertUseCaseEdgesAvoidNodes(t, s)
}

func TestERAttributeSpokesAvoidOtherAttributes(t *testing.T) {
	d := Templates()[0]
	d.Model.Entities = d.Model.Entities[:1]
	d.Model.Relations = nil
	for _, count := range []int{3, 7, 12, 30} {
		d.Model.Entities[0].Fields = nil
		for i := 0; i < count; i++ {
			d.Model.Entities[0].Fields = append(d.Model.Entities[0].Fields, Field{Name: fmt.Sprintf("业务属性_%d", i)})
		}
		s := Generate(d)
		for _, e := range s.Edges {
			for _, n := range s.Nodes {
				if !strings.Contains(n.Shape, "ttribute") || n.ID == e.Source {
					continue
				}
				for step := 1; step < 100; step++ {
					p := pointAt(e.Points, float64(step)/100)
					x := (p.X - n.X - n.Width/2) / (n.Width / 2)
					y := (p.Y - n.Y - n.Height/2) / (n.Height / 2)
					if x*x+y*y < 1 {
						t.Fatalf("%d attributes: %s crosses %s", count, e.ID, n.ID)
					}
				}
			}
		}
	}
}

func TestERThirtyAttributesUseShortSpokesInAllDirections(t *testing.T) {
	d := Templates()[0]
	d.Model.Entities[0].Fields = nil
	for i := 0; i < 30; i++ {
		d.Model.Entities[0].Fields = append(d.Model.Entities[0].Fields, Field{Name: fmt.Sprintf("属性%d", i+1)})
	}
	s := Generate(d)
	entity := sceneNode(s, "reader")
	c := center(entity)
	above, below, left, right := false, false, false, false
	for _, n := range s.Nodes {
		if !strings.HasPrefix(n.ID, "$attr:reader:") {
			continue
		}
		p := center(n)
		above = above || p.Y < c.Y
		below = below || p.Y > c.Y
		left = left || p.X < c.X
		right = right || p.X > c.X
	}
	if !above || !below || !left || !right {
		t.Fatal("attributes are not distributed around all four sides")
	}
	for _, e := range s.Edges {
		if !strings.HasPrefix(e.Source, "$attr:reader:") {
			continue
		}
		length := 0.0
		for i := 1; i < len(e.Points); i++ {
			length += math.Hypot(e.Points[i].X-e.Points[i-1].X, e.Points[i].Y-e.Points[i-1].Y)
		}
		if length > 460 {
			t.Fatalf("attribute spoke is unnecessarily long: %.0f", length)
		}
	}
	if s.Width > 2000 {
		t.Fatalf("30 attributes spread the example too wide: %.0f", s.Width)
	}
	// The horizontal relationship corridor stays clear of surrounding attributes.
	for _, e := range s.Edges {
		if !strings.HasPrefix(e.ID, "$edge:") {
			continue
		}
		for _, n := range s.Nodes {
			if !strings.Contains(n.Shape, "ttribute") {
				continue
			}
			for step := 1; step < 200; step++ {
				p := pointAt(e.Points, float64(step)/200)
				x := (p.X - n.X - n.Width/2) / (n.Width / 2)
				y := (p.Y - n.Y - n.Height/2) / (n.Height / 2)
				if x*x+y*y < 1 {
					t.Fatalf("relationship %s crosses attribute %s", e.ID, n.ID)
				}
			}
		}
	}
}

func TestERRowsIncludeAttributesBelowEntities(t *testing.T) {
	d := Templates()[0]
	d.Model.Relations = nil
	for i := 3; i < 6; i++ {
		d.Model.Entities = append(d.Model.Entities, Item{ID: fmt.Sprintf("e%d", i), Name: "实体", Fields: []Field{{Name: "编号"}, {Name: "名称"}}})
	}
	for i := 0; i < 30; i++ {
		d.Model.Entities[0].Fields = append(d.Model.Entities[0].Fields, Field{Name: fmt.Sprintf("扩展属性%d", i)})
	}
	d.Model.Entities[0].Fields = d.Model.Entities[0].Fields[:30]
	s := Generate(d)
	for i, a := range s.Nodes {
		for _, b := range s.Nodes[i+1:] {
			if a.X < b.X+b.Width && a.X+a.Width > b.X && a.Y < b.Y+b.Height && a.Y+a.Height > b.Y {
				t.Fatalf("surrounding attributes overlap another row: %s / %s", a.ID, b.ID)
			}
		}
	}
}

func TestSequenceHeadersAndActivationAlignment(t *testing.T) {
	d := Templates()[4]
	d.View.Positions["page"] = Point{0, 300}
	d.View.Positions["service"] = Point{0, 125}
	s := Generate(d)
	right := 0.0
	for _, p := range d.Model.Participants {
		n := sceneNode(s, p.ID)
		if n.Y != 64 || n.X < right {
			t.Fatal("sequence header misaligned", n)
		}
		right = n.X + n.Width
	}
	for _, e := range s.Edges {
		if e.ID == "m3" {
			n := sceneNode(s, "$activation:m3")
			end := e.Points[len(e.Points)-1]
			if math.Abs(end.X-(n.X+1)) > .01 || end.Y != n.Y {
				t.Fatal("call misses activation", end, n)
			}
			if e.Arrow != "filled" {
				t.Fatal("synchronous call needs a filled arrow")
			}
		}
		if e.ID == "m4" && (!e.Dashed || e.Arrow != "open") {
			t.Fatal("return message style incorrect")
		}
	}
}
