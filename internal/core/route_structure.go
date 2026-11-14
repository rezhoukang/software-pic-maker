package core

import "math"

// A functional tree has one branch rail per parent, including after a child is
// dragged to a different depth. Pairwise midpoint elbows split that rail.
func layoutStructureEdges(d Diagram, s *Scene, find func(string) Node) {
	layoutBoxEdges(d, s, find)
	groups := map[string][]int{}
	order := []string{}
	for i, e := range s.Edges {
		if _, ok := groups[e.Source]; !ok {
			order = append(order, e.Source)
		}
		groups[e.Source] = append(groups[e.Source], i)
	}
	for _, id := range order {
		indices := groups[id]
		parent := find(id)
		start := shapePort(parent, "bottom", .5)
		near := math.Inf(1)
		for _, i := range indices {
			child := find(s.Edges[i].Target)
			v := child.Y
			if d.View.Direction == "LR" {
				v = child.X
			}
			near = math.Min(near, v)
		}
		front := parent.Y + parent.Height
		if d.View.Direction == "LR" {
			front = parent.X + parent.Width
			start = shapePort(parent, "right", .5)
		}
		// A child dragged behind/beside its parent uses the independent obstacle
		// router above, rather than forcing a backwards branch through a box.
		if near-front < 36 {
			continue
		}
		rail := (front + near) / 2
		paths := make([][]Point, len(indices))
		safe := true
		for j, i := range indices {
			child := find(s.Edges[i].Target)
			end := shapePort(child, "top", .5)
			points := []Point{start, {start.X, rail}, {end.X, rail}, end}
			if d.View.Direction == "LR" {
				end = shapePort(child, "left", .5)
				points = []Point{start, {rail, start.Y}, {rail, end.Y}, end}
			}
			paths[j] = simplifyRoute(points)
			for k := 1; k < len(paths[j]); k++ {
				for _, n := range s.Nodes {
					if n.ID == parent.ID && k == 1 || n.ID == child.ID && k == len(paths[j])-1 {
						continue
					}
					box := routeBox{left: n.X - 6, right: n.X + n.Width + 6, top: n.Y - 6, bottom: n.Y + n.Height + 6}
					if box.crosses(paths[j][k-1], paths[j][k]) {
						safe = false
					}
				}
			}
		}
		if safe {
			for j, i := range indices {
				s.Edges[i].Points = paths[j]
			}
		}
	}
}
