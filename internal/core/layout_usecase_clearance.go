package core

import "math"

// When straight connections are blocked, adjust automatic node placement rather
// than silently switching the connector back to a polyline. Saved manual nodes
// stay fixed; the automatic-layout command clears those positions as usual.
func relaxUseCasePlacement(d Diagram, s *Scene) {
	index := map[string]int{}
	left, right := math.Inf(1), 0.0
	for i, n := range s.Nodes {
		index[n.ID] = i
		if n.Shape == "ellipse" {
			left = math.Min(left, n.X)
			right = math.Max(right, n.X)
		}
	}
	score := func() (float64, map[string]bool) {
		cost := 0.0
		blocked := map[string]bool{}
		for i, a := range s.Nodes {
			if a.Shape == "boundary" {
				continue
			}
			for _, b := range s.Nodes[i+1:] {
				if b.Shape != "boundary" && a.X < b.X+b.Width+12 && a.X+a.Width+12 > b.X && a.Y < b.Y+b.Height+12 && a.Y+a.Height+12 > b.Y {
					cost += 10000
					blocked[a.ID] = true
					blocked[b.ID] = true
				}
			}
		}
		for _, r := range d.Model.Relations {
			a, b := s.Nodes[index[r.Source]], s.Nodes[index[r.Target]]
			best := math.Inf(1)
			var hits []string
			for _, end := range []Point{shapeBoundary(b, connectionCenter(a)), ellipseNormalBoundary(b, connectionCenter(a))} {
				start := shapeBoundary(a, end)
				count := 0.0
				ids := []string{}
				for _, n := range s.Nodes {
					if n.Shape == "boundary" || n.ID == a.ID || n.ID == b.ID {
						continue
					}
					if (routeBox{left: n.X - 2, right: n.X + n.Width + 2, top: n.Y - 2, bottom: n.Y + n.Height + 2}).crosses(start, end) {
						count++
						ids = append(ids, n.ID)
					}
				}
				if count < best {
					best, hits = count, ids
				}
			}
			cost += best * 1000
			if best > 0 {
				blocked[a.ID] = true
				blocked[b.ID] = true
				for _, id := range hits {
					blocked[id] = true
				}
			}
		}
		return cost, blocked
	}
	current, blocked := score()
	verticalOffsets := []float64{0, -24, 24, -48, 48, -72, 72}
	for row := 1; row <= len(d.Model.UseCases); row++ {
		for _, fine := range []float64{-24, 0, 24} {
			offset := float64(row)*104 + fine
			verticalOffsets = append(verticalOffsets, -offset, offset)
		}
	}
	for pass := 0; current > 0 && pass < 48; pass++ {
		bestCost := current
		bestIndex, swapIndex := -1, -1
		var best, swap Point
		for i, n := range s.Nodes {
			if n.Shape != "ellipse" || !blocked[n.ID] {
				continue
			}
			if _, fixed := d.View.Positions[n.ID]; fixed {
				continue
			}
			for _, dx := range []float64{0, -360, 360} {
				for _, dy := range verticalOffsets {
					x, y := n.X+dx, n.Y+dy
					if x < left || x > right || y < 140 {
						continue
					}
					s.Nodes[i].X, s.Nodes[i].Y = x, y
					cost, _ := score()
					// Prefer small repairs among solutions with equal obstruction counts.
					cost += math.Hypot(dx, dy) * .001
					if cost < bestCost-.001 {
						bestCost, bestIndex, best = cost, i, Point{x, y}
					}
				}
			}
			s.Nodes[i] = n
		}
		if bestIndex < 0 {
			for i, a := range s.Nodes {
				if a.Shape != "ellipse" || !blocked[a.ID] {
					continue
				}
				if _, fixed := d.View.Positions[a.ID]; fixed {
					continue
				}
				for j, b := range s.Nodes {
					if j <= i || b.Shape != "ellipse" {
						continue
					}
					if _, fixed := d.View.Positions[b.ID]; fixed {
						continue
					}
					s.Nodes[i].X, s.Nodes[i].Y = b.X, b.Y
					s.Nodes[j].X, s.Nodes[j].Y = a.X, a.Y
					cost, _ := score()
					if cost < bestCost-.001 {
						bestCost, bestIndex, swapIndex, best, swap = cost, i, j, Point{b.X, b.Y}, Point{a.X, a.Y}
					}
					s.Nodes[i], s.Nodes[j] = a, b
				}
			}
		}
		if bestIndex < 0 {
			break
		}
		s.Nodes[bestIndex].X, s.Nodes[bestIndex].Y = best.X, best.Y
		if swapIndex >= 0 {
			s.Nodes[swapIndex].X, s.Nodes[swapIndex].Y = swap.X, swap.Y
		}
		current, blocked = score()
	}
	bottom := 0.0
	for _, n := range s.Nodes {
		if n.Shape != "boundary" {
			bottom = math.Max(bottom, n.Y+n.Height+50)
		}
	}
	if i, ok := index["$boundary"]; ok {
		s.Nodes[i].Height = math.Max(s.Nodes[i].Height, bottom-s.Nodes[i].Y)
	}
}
