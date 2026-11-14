package core

import (
	"math"
	"sort"
)

// Independent relations need independent ports and elbows; only a hierarchy is
// allowed to intentionally share a bus.
func layoutBoxEdges(d Diagram, s *Scene, find func(string) Node) {
	type end struct {
		node, other Node
		side        string
		port        routePort
	}
	ends := make([]*end, 0, len(d.Model.Relations)*2)
	groups := map[string][]*end{}
	for _, r := range d.Model.Relations {
		a, b := find(r.Source), find(r.Target)
		side := "bottom"
		if d.View.Direction == "LR" {
			side = "right"
		}
		if b.X >= a.X+a.Width {
			side = "right"
		} else if a.X >= b.X+b.Width {
			side = "left"
		} else if b.Y >= a.Y+a.Height {
			side = "bottom"
		} else if a.Y >= b.Y+b.Height {
			side = "top"
		}
		targetSide := oppositeSide(side)
		if a.ID == b.ID {
			side = "right"
			targetSide = "top"
		}
		for _, e := range []*end{{node: a, other: b, side: side}, {node: b, other: a, side: targetSide}} {
			ends = append(ends, e)
			key := e.node.ID + ":" + e.side
			groups[key] = append(groups[key], e)
		}
	}
	// Header side ports stay above attributes and methods. Overflow goes to the
	// top outline, where there is room to keep arrowheads separated.
	overflow := []*end{}
	for _, group := range groups {
		sort.SliceStable(group, func(i, j int) bool {
			if group[i].side == "left" || group[i].side == "right" {
				return center(group[i].other).Y < center(group[j].other).Y
			}
			return center(group[i].other).X < center(group[j].other).X
		})
		for i, e := range group {
			if e.node.Shape == "table" && (e.side == "left" || e.side == "right") && i >= 2 {
				overflow = append(overflow, e)
			}
		}
	}
	sort.SliceStable(overflow, func(i, j int) bool {
		a, b := overflow[i], overflow[j]
		if a.node.ID != b.node.ID {
			return a.node.ID < b.node.ID
		}
		if a.side != b.side {
			return a.side < b.side
		}
		return a.other.ID < b.other.ID
	})
	for _, e := range overflow {
		key := e.node.ID + ":" + e.side
		g := groups[key]
		for i, v := range g {
			if v == e {
				groups[key] = append(g[:i], g[i+1:]...)
				break
			}
		}
		e.side = "top"
		groups[e.node.ID+":top"] = append(groups[e.node.ID+":top"], e)
	}
	router := newUseCaseRouter(s.Nodes)
	for i := range router.boxes {
		b := &router.boxes[i]
		b.left += 8
		b.right -= 8
		b.top += 8
		b.bottom -= 8
	}
	for _, group := range groups {
		for i, e := range group {
			ratio := float64(i+1) / float64(len(group)+1)
			e.port = portWithStub(e.node, e.side, ratio)
			switch e.side {
			case "left":
				e.port.stub.X = e.node.X - 20
			case "right":
				e.port.stub.X = e.node.X + e.node.Width + 20
			case "top":
				e.port.stub.Y = e.node.Y - 20
			case "bottom":
				e.port.stub.Y = e.node.Y + e.node.Height + 20
			}
			if e.node.Shape == "table" && (e.side == "left" || e.side == "right") {
				y := e.node.Y + 22
				if len(group) > 1 {
					y = e.node.Y + 14 + float64(i)*16
				}
				e.port.point.Y = y
				e.port.stub.Y = y
			}
			router.extraPorts = append(router.extraPorts, e.port)
		}
	}
	for i, r := range d.Model.Relations {
		a, b := ends[2*i], ends[2*i+1]
		p, q := a.port, b.port
		points := []Point{p.point, p.stub}
		if a.node.ID == b.node.ID {
			points = append(points, Point{p.stub.X, q.stub.Y}, q.stub)
		} else if p.axis == q.axis {
			if p.axis == 0 {
				lane := (p.stub.X+q.stub.X)/2 + float64(i%3-1)*12
				points = append(points, Point{lane, p.stub.Y}, Point{lane, q.stub.Y}, q.stub)
			} else {
				lane := (p.stub.Y+q.stub.Y)/2 + float64(i%3-1)*12
				points = append(points, Point{p.stub.X, lane}, Point{q.stub.X, lane}, q.stub)
			}
		} else if p.axis == 0 {
			points = append(points, Point{q.stub.X, p.stub.Y}, q.stub)
		} else {
			points = append(points, Point{p.stub.X, q.stub.Y}, q.stub)
		}
		points = simplifyRoute(append(points, q.point))
		clear := true
		for j := 1; j < len(points); j++ {
			for _, box := range router.boxes {
				// Test endpoint boxes too, except for the terminal segment attached there.
				if box.id == a.node.ID && j == 1 || box.id == b.node.ID && j == len(points)-1 {
					continue
				}
				if box.crosses(points[j-1], points[j]) {
					clear = false
				}
			}
		}
		if !clear {
			points = router.routePorts(a.node, b.node, []routePort{p}, []routePort{q})
		}
		arrow := "none"
		if d.Type == "architecture" || r.Kind == "dependency" {
			arrow = "open"
		}
		if r.Kind == "inheritance" {
			arrow = "triangle"
		}
		edge := Edge{ID: r.ID, Source: r.Source, Target: r.Target, Points: points, Label: r.Label, SourceLabel: r.SourceCardinality, TargetLabel: r.TargetCardinality, Arrow: arrow, Dashed: r.Kind == "dependency"}
		if d.Type == "table" && len(points) == 2 && math.Abs(points[0].Y-points[1].Y) < .01 {
			// Keep the relation name above the line, and multiplicities below it
			// beside their own table. Each label has its own horizontal space.
			start, end := points[0], points[1]
			direction := 1.0
			if end.X < start.X {
				direction = -1
			}
			edge.LabelPosition = &Point{(start.X + end.X) / 2, start.Y - 18}
			edge.SourceLabelPosition = &Point{start.X + direction*(labelWidth(r.SourceCardinality)/2+12), start.Y + 18}
			edge.TargetLabelPosition = &Point{end.X - direction*(labelWidth(r.TargetCardinality)/2+12), end.Y + 18}
		}
		s.Edges = append(s.Edges, edge)
	}
}

func labelWidth(text string) float64 {
	w := 0.0
	for _, r := range text {
		if r < 128 {
			w += 7.5
		} else {
			w += 14
		}
	}
	return w + 10
}

func placeEdgeLabels(s *Scene) {
	if s.Type == "usecase" {
		placeUseCaseLabels(s)
		return
	}
	occupied := []routeBox{}
	for _, n := range s.Nodes {
		if n.Shape != "boundary" && n.Shape != "activation" {
			occupied = append(occupied, routeBox{left: n.X - 4, right: n.X + n.Width + 4, top: n.Y - 4, bottom: n.Y + n.Height + 4})
		}
	}
	for i := range s.Edges {
		e := &s.Edges[i]
		if len(e.Points) < 2 {
			continue
		}
		for index, text := range []string{e.Label, e.SourceLabel, e.TargetLabel} {
			if text == "" {
				continue
			}
			slot := &e.LabelPosition
			ratio := .5
			if index == 1 {
				slot = &e.SourceLabelPosition
				ratio = .12
			}
			if index == 2 {
				slot = &e.TargetLabelPosition
				ratio = .88
			}
			width := labelWidth(text)
			candidates := []Point{}
			if *slot != nil {
				candidates = append(candidates, **slot)
			}
			if index > 0 {
				p := pointAt(e.Points, ratio)
				candidates = append(candidates, Point{p.X, p.Y - 17}, Point{p.X, p.Y + 18}, Point{p.X - width/2 - 12, p.Y}, Point{p.X + width/2 + 12, p.Y})
			}
			type segment struct {
				a, b   Point
				length float64
			}
			segments := []segment{}
			for j := 1; j < len(e.Points); j++ {
				a, b := e.Points[j-1], e.Points[j]
				segments = append(segments, segment{a, b, math.Hypot(a.X-b.X, a.Y-b.Y)})
			}
			sort.SliceStable(segments, func(i, j int) bool { return segments[i].length > segments[j].length })
			for _, seg := range segments {
				a, b := seg.a, seg.b
				mid := Point{(a.X + b.X) / 2, (a.Y + b.Y) / 2}
				if math.Abs(a.Y-b.Y) < .01 {
					candidates = append(candidates, Point{mid.X, mid.Y - 16}, Point{mid.X, mid.Y + 18})
				} else if math.Abs(a.X-b.X) > .01 && seg.length > 0 {
					// A fixed vertical offset can leave a wide label crossing a
					// diagonal. Clear the entire label rectangle along the normal.
					nx, ny := -(b.Y-a.Y)/seg.length, (b.X-a.X)/seg.length
					if ny > 0 {
						nx, ny = -nx, -ny
					}
					distance := math.Abs(nx)*width/2 + math.Abs(ny)*11 + 8
					for _, t := range []float64{.5, .45, .55, .4, .6, .35, .65, .25, .75, .2, .8} {
						p := Point{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t}
						candidates = append(candidates, Point{p.X + nx*distance, p.Y + ny*distance}, Point{p.X - nx*distance, p.Y - ny*distance})
					}
				}
			}
			for _, seg := range segments {
				a, b := seg.a, seg.b
				mid := Point{(a.X + b.X) / 2, (a.Y + b.Y) / 2}
				candidates = append(candidates, Point{mid.X - width/2 - 10, mid.Y}, Point{mid.X + width/2 + 10, mid.Y})
			}
			best := pointAt(e.Points, .5)
			best.Y -= 16
			bestScore := math.Inf(1)
			for k, p := range candidates {
				box := routeBox{left: p.X - width/2, right: p.X + width/2, top: p.Y - 11, bottom: p.Y + 11}
				score := float64(k) * .01
				if box.left < 0 || box.top < 0 {
					score += 10000
				}
				for _, o := range occupied {
					if box.left < o.right && box.right > o.left && box.top < o.bottom && box.bottom > o.top {
						score += 10000
					}
				}
				// Labels should also avoid hiding another relation's arrow or line.
				for _, other := range s.Edges {
					for j := 1; j < len(other.Points); j++ {
						if box.crosses(other.Points[j-1], other.Points[j]) {
							score += 100
						}
					}
				}
				if score < bestScore {
					bestScore = score
					best = p
				}
			}
			*slot = &best
			occupied = append(occupied, routeBox{left: best.X - width/2, right: best.X + width/2, top: best.Y - 12, bottom: best.Y + 12})
			s.Width = math.Max(s.Width, best.X+width/2+40)
			s.Height = math.Max(s.Height, best.Y+40)
		}
	}
}
