package core

import (
	"math"
	"testing"
)

func TestStructureDraggedChildrenKeepOneSharedRail(t *testing.T) {
	for _, direction := range []string{"TB", "LR"} {
		d := Templates()[7]
		d.View.Direction = direction
		if direction == "TB" {
			d.View.Positions = map[string]Point{"system": {140, 20}, "readers": {248, 218}, "admin": {532, 218}, "search": {0, 490}, "loans": {248, 372}, "books": {532, 372}, "categories": {816, 372}}
		} else {
			d.View.Positions = map[string]Point{"system": {20, 140}, "readers": {340, 248}, "admin": {340, 532}, "search": {800, 0}, "loans": {700, 248}, "books": {700, 532}, "categories": {700, 816}}
		}
		s := Generate(d)
		var a, b Edge
		for _, e := range s.Edges {
			if e.ID == "s3" {
				a = e
			}
			if e.ID == "s4" {
				b = e
			}
		}
		if len(a.Points) < 3 {
			t.Fatal("missing tree rail", a)
		}
		start := a.Points[0]
		if start != b.Points[0] {
			t.Fatal("siblings do not share parent's stem")
		}
		rail := a.Points[1].Y
		if direction == "LR" {
			rail = a.Points[1].X
		}
		// One child lies in line with the parent so simplification may remove its
		// explicit rail vertex; the other edge must still traverse the same rail.
		p := Point{start.X, rail}
		if direction == "LR" {
			p = Point{rail, start.Y}
		}
		onRail := false
		for j := 1; j < len(b.Points); j++ {
			x, y := b.Points[j-1], b.Points[j]
			if math.Abs(math.Hypot(x.X-p.X, x.Y-p.Y)+math.Hypot(y.X-p.X, y.Y-p.Y)-math.Hypot(y.X-x.X, y.Y-x.Y)) < .01 {
				onRail = true
			}
		}
		if !onRail {
			t.Fatal("staggered siblings split shared rail", direction, a.Points, b.Points)
		}
		for _, e := range s.Edges {
			if e.Arrow != "none" {
				t.Fatal("functional hierarchy should not acquire UML arrows")
			}
			for i := 1; i < len(e.Points); i++ {
				for _, n := range s.Nodes {
					if n.ID == e.Source && i == 1 || n.ID == e.Target && i == len(e.Points)-1 {
						continue
					}
					box := routeBox{left: n.X, right: n.X + n.Width, top: n.Y, bottom: n.Y + n.Height}
					if box.crosses(e.Points[i-1], e.Points[i]) {
						t.Fatal("dragged tree crosses node", direction, e.ID, n.ID)
					}
				}
			}
		}
		moved := sceneNode(s, "search")
		if moved.X != d.View.Positions["search"].X || moved.Y != d.View.Positions["search"].Y {
			t.Fatal("manual position overwritten")
		}
	}
}

func TestStructureChildDraggedBehindParentUsesClearSidePorts(t *testing.T) {
	d := Templates()[7]
	d.View.Positions["search"] = Point{1100, 40}
	s := Generate(d)
	for _, e := range s.Edges {
		for i := 1; i < len(e.Points); i++ {
			for _, n := range s.Nodes {
				if n.ID == e.Source && i == 1 || n.ID == e.Target && i == len(e.Points)-1 {
					continue
				}
				box := routeBox{left: n.X, right: n.X + n.Width, top: n.Y, bottom: n.Y + n.Height}
				if box.crosses(e.Points[i-1], e.Points[i]) {
					t.Fatal("reverse drag crosses node", e.ID, n.ID, e.Points)
				}
			}
		}
	}
}
