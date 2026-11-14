package core

import (
	"math"
	"sort"
)

func flowPort(n Node, side string, ratio float64) Point { return shapePort(n, side, ratio) }
func flowRay(n Node, towards Point) Point               { return shapeBoundary(n, towards) }

func routeFlowchart(a, b Node, direction string, entryRatio float64) ([]Point, *Point) {
	ac, bc := center(a), center(b)
	if a.ID == b.ID {
		start, end := flowPort(a, "right", .5), flowPort(a, "top", .5)
		x, y := a.X+a.Width+60, math.Max(0, a.Y-45)
		return []Point{start, {x, start.Y}, {x, y}, {end.X, y}, end}, nil
	}
	if direction == "LR" && b.X >= a.X+a.Width {
		start, end := flowPort(a, "right", .5), flowPort(b, "left", entryRatio)
		if a.Shape == "diamond" && (end.Y < a.Y || end.Y > a.Y+a.Height) {
			side := "bottom"
			if end.Y < ac.Y {
				side = "top"
			}
			start = flowPort(a, side, .5)
			label := Point{start.X - 17, (start.Y + end.Y) / 2}
			return simplifyRoute([]Point{start, {start.X, end.Y}, end}), &label
		}
		if math.Abs(start.Y-end.Y) < .001 {
			return []Point{start, end}, nil
		}
		mid := (start.X + end.X) / 2
		return simplifyRoute([]Point{start, {mid, start.Y}, {mid, end.Y}, end}), nil
	}
	if direction != "LR" && b.Y >= a.Y+a.Height {
		start, end := flowPort(a, "bottom", .5), flowPort(b, "top", entryRatio)
		if a.Shape == "diamond" && (end.X < a.X || end.X > a.X+a.Width) {
			side := "right"
			if end.X < ac.X {
				side = "left"
			}
			start = flowPort(a, side, .5)
			label := Point{(start.X + end.X) / 2, start.Y - 14}
			return simplifyRoute([]Point{start, {end.X, start.Y}, end}), &label
		}
		if math.Abs(start.X-end.X) < .001 {
			return []Point{start, end}, nil
		}
		mid := (start.Y + end.Y) / 2
		return simplifyRoute([]Point{start, {start.X, mid}, {end.X, mid}, end}), nil
	}
	return []Point{flowRay(a, bc), flowRay(b, ac)}, nil
}

func mergeFlowchartEdges(d Diagram, s *Scene, add func(Node), find func(string) Node) {
	groups := map[string][]int{}
	order := []string{}
	for i, e := range s.Edges {
		if _, ok := groups[e.Target]; !ok {
			order = append(order, e.Target)
		}
		groups[e.Target] = append(groups[e.Target], i)
	}
	for _, targetID := range order {
		indices := groups[targetID]
		if len(indices) < 2 {
			continue
		}
		sort.SliceStable(indices, func(i, j int) bool {
			a, b := center(find(s.Edges[indices[i]].Source)), center(find(s.Edges[indices[j]].Source))
			if d.View.Direction == "LR" {
				return a.Y < b.Y
			}
			return a.X < b.X
		})
		target := find(targetID)
		tc := center(target)
		front := 0.0
		for _, i := range indices {
			n := find(s.Edges[i].Source)
			value := n.Y + n.Height
			if d.View.Direction == "LR" {
				value = n.X + n.Width
			}
			front = math.Max(front, value)
		}
		join := Point{tc.X, (front + target.Y) / 2}
		room := target.Y - front
		if d.View.Direction == "LR" {
			join = Point{(front + target.X) / 2, tc.Y}
			room = target.X - front
		}
		paths := make([][]Point, len(indices))
		safe := room >= 28
		for j, i := range indices {
			e := s.Edges[i]
			source := find(e.Source)
			start := flowPort(source, "bottom", .5)
			turn := Point{start.X, join.Y}
			if d.View.Direction == "LR" {
				start = flowPort(source, "right", .5)
				turn = Point{join.X, start.Y}
			}
			paths[j] = simplifyRoute([]Point{start, turn, join})
			for _, n := range s.Nodes {
				if n.ID == source.ID || n.ID == targetID {
					continue
				}
				box := routeBox{left: n.X, right: n.X + n.Width, top: n.Y, bottom: n.Y + n.Height}
				for k := 1; k < len(paths[j]); k++ {
					if box.crosses(paths[j][k-1], paths[j][k]) {
						safe = false
					}
				}
			}
		}
		if !safe {
			// If a merge cannot fit, give independent arrows distinct entries.
			for j, i := range indices {
				source := find(s.Edges[i].Source)
				s.Edges[i].Points, s.Edges[i].LabelPosition = routeFlowchart(source, target, d.View.Direction, float64(j+1)/float64(len(indices)+1))
			}
			continue
		}
		id := "$junction:" + targetID
		add(Node{ID: id, X: join.X - 4, Y: join.Y - 4, Width: 8, Height: 8, Shape: "junction", Movable: false})
		for j, i := range indices {
			s.Edges[i].Target = id
			s.Edges[i].Points = paths[j]
			s.Edges[i].Arrow = "none"
			s.Edges[i].LabelPosition = nil
		}
		points, _ := routeFlowchart(find(id), target, d.View.Direction, .5)
		s.Edges = append(s.Edges, Edge{ID: "$junction-edge:" + targetID, Source: id, Target: targetID, Points: points, Arrow: "open"})
	}
}
