package core

import (
	"math"
	"sort"
)

type architecturePort struct {
	point Point
	side  string
}

type architectureEnd struct {
	node, other Node
	id          string
	preferred   string
	portRange   *[2]float64
	ports       []architecturePort
}

type architectureSegment struct {
	a, b  Point
	owner string
}

func architectureNormal(side string) Point {
	switch side {
	case "left":
		return Point{-1, 0}
	case "right":
		return Point{1, 0}
	case "top":
		return Point{0, -1}
	default:
		return Point{0, 1}
	}
}

func architectureCross(a, b Point) float64 { return a.X*b.Y - a.Y*b.X }

func architectureOverlap(a, b, c, d Point) float64 {
	vector, other := Point{b.X - a.X, b.Y - a.Y}, Point{d.X - c.X, d.Y - c.Y}
	length := math.Hypot(vector.X, vector.Y)
	otherLength := math.Hypot(other.X, other.Y)
	if length < .001 || otherLength < .001 || math.Abs(architectureCross(vector, other))/(length*otherLength) > .00001 || math.Abs(architectureCross(vector, Point{c.X - a.X, c.Y - a.Y}))/length > .001 {
		return 0
	}
	project := func(p Point) float64 { return ((p.X-a.X)*vector.X + (p.Y-a.Y)*vector.Y) / length }
	x, y := project(c), project(d)
	return math.Max(0, math.Min(length, math.Max(x, y))-math.Max(0, math.Min(x, y)))
}

func architectureIntersects(a, b, c, d Point) bool {
	v, w := Point{b.X - a.X, b.Y - a.Y}, Point{d.X - c.X, d.Y - c.Y}
	denominator := architectureCross(v, w)
	if math.Abs(denominator) < .001 {
		return false
	}
	offset := Point{c.X - a.X, c.Y - a.Y}
	t, u := architectureCross(offset, w)/denominator, architectureCross(offset, v)/denominator
	return t >= -.001 && t <= 1.001 && u >= -.001 && u <= 1.001
}

func architectureRouteCost(owner string, points []Point, used []architectureSegment) float64 {
	cost := float64(len(points)-2) * 24
	for i := 1; i < len(points); i++ {
		a, b := points[i-1], points[i]
		cost += math.Hypot(a.X-b.X, a.Y-b.Y)
		for _, other := range used {
			if other.owner == owner {
				continue
			}
			if overlap := architectureOverlap(a, b, other.a, other.b); overlap > .01 {
				cost += 1000000 + overlap*100
			}
			// Distinct rails must also look distinct at ordinary zoom levels.
			if math.Abs(a.X-b.X) < .001 && math.Abs(other.a.X-other.b.X) < .001 {
				gap := math.Abs(a.X - other.a.X)
				if gap > .001 && gap < 12 && math.Min(math.Max(a.Y, b.Y), math.Max(other.a.Y, other.b.Y)) > math.Max(math.Min(a.Y, b.Y), math.Min(other.a.Y, other.b.Y)) {
					cost += 20000 + (12-gap)*100
				}
			} else if math.Abs(a.Y-b.Y) < .001 && math.Abs(other.a.Y-other.b.Y) < .001 {
				gap := math.Abs(a.Y - other.a.Y)
				if gap > .001 && gap < 12 && math.Min(math.Max(a.X, b.X), math.Max(other.a.X, other.b.X)) > math.Max(math.Min(a.X, b.X), math.Min(other.a.X, other.b.X)) {
					cost += 20000 + (12-gap)*100
				}
			}
			if architectureIntersects(a, b, other.a, other.b) {
				cost += 320
			}
		}
	}
	return cost
}

// Align facing ports when the components share a horizontal/vertical corridor.
// Keep each endpoint separated from its neighbors instead of adding tiny elbows
// merely because two independently distributed ports differ by a few pixels.
func architectureAlignPorts(ends []*architectureEnd, nodes []Node) {
	for i := 0; i+1 < len(ends); i += 2 {
		a, b := ends[i], ends[i+1]
		if a.node.ID == b.node.ID || a.preferred != oppositeSide(b.preferred) {
			continue
		}
		vertical := architectureNormal(a.preferred).Y != 0
		coordinate := func(p Point) float64 {
			if vertical {
				return p.X
			}
			return p.Y
		}
		indices := [2]int{}
		limits := [2][2]float64{}
		current := [2]float64{}
		for k, end := range []*architectureEnd{a, b} {
			for j, p := range end.ports {
				if p.side == end.preferred {
					indices[k] = j
					current[k] = coordinate(p.point)
					break
				}
			}
			lo := coordinate(shapePort(end.node, end.preferred, .08))
			hi := coordinate(shapePort(end.node, end.preferred, .92))
			if end.portRange != nil {
				lo, hi = end.portRange[0], end.portRange[1]
			}
			for _, other := range ends {
				if other == end || other.node.ID != end.node.ID || other.preferred != end.preferred {
					continue
				}
				for _, p := range other.ports {
					if p.side != end.preferred {
						continue
					}
					v := coordinate(p.point)
					if v < current[k] {
						lo = math.Max(lo, v+14)
					} else {
						hi = math.Min(hi, v-14)
					}
				}
			}
			limits[k] = [2]float64{lo, hi}
		}
		lo, hi := math.Max(limits[0][0], limits[1][0]), math.Min(limits[0][1], limits[1][1])
		if lo > hi {
			continue
		}
		value := math.Max(lo, math.Min(hi, (current[0]+current[1])/2))
		if a.portRange == nil && b.portRange == nil && math.Abs(coordinate(center(a.node))-coordinate(center(b.node))) < .001 {
			value = math.Max(lo, math.Min(hi, coordinate(center(a.node))))
		}
		aligned := [2]architecturePort{}
		for k, end := range []*architectureEnd{a, b} {
			p, q := shapePort(end.node, end.preferred, 0), shapePort(end.node, end.preferred, 1)
			span := coordinate(q) - coordinate(p)
			if span > .001 {
				aligned[k] = architecturePort{point: shapePort(end.node, end.preferred, (value-coordinate(p))/span), side: end.preferred}
			}
		}
		if architectureRouteClear([]Point{aligned[0].point, aligned[1].point}, a.node, b.node, nodes, aligned[0], aligned[1]) {
			a.ports[indices[0]], b.ports[indices[1]] = aligned[0], aligned[1]
		}
	}
}

func architectureRouteClear(points []Point, source, target Node, nodes []Node, p, q architecturePort) bool {
	if len(points) < 2 || len(points) > 4 {
		return false
	}
	for _, end := range []struct {
		a, b Point
		side string
	}{{points[0], points[1], p.side}, {points[len(points)-1], points[len(points)-2], q.side}} {
		normal := architectureNormal(end.side)
		dx, dy := end.b.X-end.a.X, end.b.Y-end.a.Y
		length := math.Hypot(dx, dy)
		if length < 14 || (dx*normal.X+dy*normal.Y)/length < .99 {
			return false
		}
	}
	for i := 1; i < len(points); i++ {
		a, b := points[i-1], points[i]
		length := math.Hypot(b.X-a.X, b.Y-a.Y)
		if length < .001 || a.X < 12 || a.Y < 12 || b.X < 12 || b.Y < 12 {
			return false
		}
		if math.Abs(a.X-b.X) > .001 && math.Abs(a.Y-b.Y) > .001 {
			return false
		}
		for _, n := range nodes {
			if n.ID == source.ID && i == 1 || n.ID == target.ID && i == len(points)-1 {
				continue
			}
			if (routeBox{left: n.X - 6, right: n.X + n.Width + 6, top: n.Y - 6, bottom: n.Y + n.Height + 6}).crosses(a, b) {
				return false
			}
		}
	}
	// Convex shapes must be left/entered from outside their painted outline,
	// including diamond and rounded corners rather than just the image box.
	for _, end := range []struct {
		node     Node
		at, next Point
	}{{source, points[0], points[1]}, {target, points[len(points)-1], points[len(points)-2]}} {
		dx, dy := end.next.X-end.at.X, end.next.Y-end.at.Y
		length := math.Hypot(dx, dy)
		sample := Point{end.at.X + dx/length, end.at.Y + dy/length}
		boundary, c := shapeBoundary(end.node, sample), center(end.node)
		if math.Hypot(sample.X-c.X, sample.Y-c.Y) < math.Hypot(boundary.X-c.X, boundary.Y-c.Y)-.01 {
			return false
		}
	}
	return true
}

func architectureChooseRoute(id string, source, target *architectureEnd, nodes []Node, used []architectureSegment, xs, ys []float64) []Point {
	best, bestCost := []Point(nil), math.Inf(1)
	consider := func(points []Point, p, q architecturePort) {
		points = simplifyRoute(points)
		if !architectureRouteClear(points, source.node, target.node, nodes, p, q) {
			return
		}
		cost := architectureRouteCost(id, points, used)
		if p.side != source.preferred {
			cost += 2000
		}
		if q.side != target.preferred {
			cost += 2000
		}
		if cost < bestCost-.001 {
			best, bestCost = points, cost
		}
	}
	// Straight means horizontal or vertical, never diagonal. A direct connector
	// takes precedence when its ports keep input and output on separate faces.
	if source.node.ID != target.node.ID {
		alternateOut, alternateIn := "bottom", "top"
		if source.preferred == "bottom" {
			alternateOut, alternateIn = "right", "left"
		}
		for _, p := range source.ports {
			for _, q := range target.ports {
				if p.side != source.preferred && p.side != alternateOut || q.side != target.preferred && q.side != alternateIn {
					continue
				}
				consider([]Point{p.point, q.point}, p, q)
			}
		}
		if best != nil && bestCost < 1000000 {
			return best
		}
	}
	straight, straightCost := best, bestCost
	best, bestCost = nil, math.Inf(1)
	for phase := 0; phase < 2; phase++ {
		for _, p := range source.ports {
			for _, q := range target.ports {
				if phase == 0 && (p.side != source.preferred || q.side != target.preferred) {
					continue
				}
				pa, qa := architectureNormal(p.side), architectureNormal(q.side)
				if pa.X != 0 && qa.X != 0 {
					for _, x := range xs {
						consider([]Point{p.point, {x, p.point.Y}, {x, q.point.Y}, q.point}, p, q)
					}
				} else if pa.Y != 0 && qa.Y != 0 {
					for _, y := range ys {
						consider([]Point{p.point, {p.point.X, y}, {q.point.X, y}, q.point}, p, q)
					}
				} else if pa.X != 0 {
					consider([]Point{p.point, {q.point.X, p.point.Y}, q.point}, p, q)
				} else {
					consider([]Point{p.point, {p.point.X, q.point.Y}, q.point}, p, q)
				}
			}
		}
		if best != nil && bestCost < 1000000 {
			break
		}
	}
	if straight != nil && straightCost < bestCost {
		return straight
	}
	return best
}

// Bounded connector candidates, rather than an unrestricted maze router, enforce
// straight-first and no more than two right-angle bends in editor and exports.
func layoutArchitectureEdges(d Diagram, s *Scene, find func(string) Node, rows [][]string, allowObstruction bool) bool {
	ends := make([]*architectureEnd, 0, len(d.Model.Relations)*2)
	groups := map[string][]*architectureEnd{}
	for _, r := range d.Model.Relations {
		a, b := find(r.Source), find(r.Target)
		out, in := "right", "left"
		if d.View.Direction != "LR" {
			out, in = "bottom", "top"
		}
		for _, end := range []*architectureEnd{{node: a, other: b, id: r.ID + ":source", preferred: out}, {node: b, other: a, id: r.ID + ":target", preferred: in}} {
			ends = append(ends, end)
			groups[end.node.ID] = append(groups[end.node.ID], end)
		}
	}
	for _, group := range groups {
		for _, side := range []string{"right", "left", "bottom", "top"} {
			normal := architectureNormal(side)
			positive, negative := []*architectureEnd{}, []*architectureEnd{}
			for _, end := range group {
				if end.preferred == side {
					positive = append(positive, end)
				} else {
					negative = append(negative, end)
				}
			}
			less := func(a, b *architectureEnd) bool {
				p, q := center(a.other), center(b.other)
				if normal.Y != 0 {
					p.Y, q.Y = p.X, q.X
				}
				if p.Y != q.Y {
					return p.Y < q.Y
				}
				return a.id < b.id
			}
			sort.SliceStable(positive, func(i, j int) bool { return less(positive[i], positive[j]) })
			sort.SliceStable(negative, func(i, j int) bool { return less(negative[i], negative[j]) })
			ratios := []float64{0, 1}
			for i, end := range positive {
				ratio := float64(i+1) / float64(len(positive)+1)
				if len(positive) == 2 {
					a, b := shapePort(end.node, side, 0), shapePort(end.node, side, 1)
					span := math.Abs(b.Y - a.Y)
					if normal.Y != 0 {
						span = math.Abs(b.X - a.X)
					}
					// Leave a free center slot for a straight continuation even on
					// the narrow side of a vertically compact component.
					ratio = .5 + float64(2*i-1)*math.Min(.5, 16/math.Max(1, span))
				}
				end.ports = append(end.ports, architecturePort{shapePort(end.node, side, ratio), side})
				ratios = append(ratios, ratio)
			}
			// Opposite-face options are only needed for short obstacle detours. Fill
			// spare gaps, so they don't displace an ordinary one-to-one center port.
			for _, end := range negative {
				sort.Float64s(ratios)
				index := 0
				for i := 1; i+1 < len(ratios); i++ {
					if ratios[i+1]-ratios[i] > ratios[index+1]-ratios[index] {
						index = i
					}
				}
				ratio := (ratios[index] + ratios[index+1]) / 2
				end.ports = append(end.ports, architecturePort{shapePort(end.node, side, ratio), side})
				ratios = append(ratios, ratio)
			}
		}
	}
	architectureAlignPorts(ends, s.Nodes)
	xs, ys := []float64{16}, []float64{16}
	maxX, maxY := 0.0, 0.0
	for _, n := range s.Nodes {
		xs = append(xs, n.X-24, n.X+n.Width+24)
		ys = append(ys, n.Y-24, n.Y+n.Height+24)
		maxX, maxY = math.Max(maxX, n.X+n.Width), math.Max(maxY, n.Y+n.Height)
	}
	for i := 0; i <= len(d.Model.Relations); i++ {
		offset := 24 + float64(i)*16
		xs = append(xs, maxX+offset)
		ys = append(ys, maxY+offset)
	}
	for level := 0; level+1 < len(rows); level++ {
		xEnd, yEnd, xStart, yStart := 0.0, 0.0, math.Inf(1), math.Inf(1)
		for _, id := range rows[level] {
			n := find(id)
			xEnd = math.Max(xEnd, n.X+n.Width)
			yEnd = math.Max(yEnd, n.Y+n.Height)
		}
		for _, id := range rows[level+1] {
			n := find(id)
			xStart = math.Min(xStart, n.X)
			yStart = math.Min(yStart, n.Y)
		}
		if xStart > xEnd {
			xs = append(xs, (xStart+xEnd)/2)
			for x := xEnd + 20; x <= xStart-20; x += 14 {
				xs = append(xs, x)
			}
		}
		if yStart > yEnd {
			ys = append(ys, (yStart+yEnd)/2)
			for y := yEnd + 20; y <= yStart-20; y += 14 {
				ys = append(ys, y)
			}
		}
	}
	xs, ys = uniqueCoordinates(xs), uniqueCoordinates(ys)
	order := make([]int, len(d.Model.Relations))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, b := order[i], order[j]
		length := func(i int) float64 {
			p, q := center(ends[2*i].node), center(ends[2*i+1].node)
			return math.Hypot(p.X-q.X, p.Y-q.Y)
		}
		if length(a) != length(b) {
			return length(a) > length(b)
		}
		return d.Model.Relations[a].ID < d.Model.Relations[b].ID
	})
	obstacles := s.Nodes
	if allowObstruction {
		obstacles = nil
	}
	plan := func() ([]Edge, bool) {
		edges := make([]Edge, len(order))
		used := []architectureSegment{}
		for _, i := range order {
			r := d.Model.Relations[i]
			points := architectureChooseRoute(r.ID, ends[2*i], ends[2*i+1], obstacles, used, xs, ys)
			if points == nil {
				return nil, false
			}
			for j := 1; j < len(points); j++ {
				used = append(used, architectureSegment{points[j-1], points[j], r.ID})
			}
			arrow := "open"
			if r.Kind == "inheritance" {
				arrow = "triangle"
			}
			edges[i] = Edge{ID: r.ID, Source: r.Source, Target: r.Target, Points: points, Label: r.Label, SourceLabel: r.SourceCardinality, TargetLabel: r.TargetCardinality, Arrow: arrow, Dashed: r.Kind == "dependency" || r.Kind == "return"}
		}
		// Revisit early choices with all later connectors present. Otherwise an edge
		// routed first cannot account for crossings introduced by the remaining fanout.
		for pass := 0; pass < 2; pass++ {
			changed := false
			for step := range order {
				i := order[len(order)-step-1]
				others := make([]architectureSegment, 0, len(used))
				for j, edge := range edges {
					if j == i {
						continue
					}
					for k := 1; k < len(edge.Points); k++ {
						others = append(others, architectureSegment{edge.Points[k-1], edge.Points[k], edge.ID})
					}
				}
				points := architectureChooseRoute(edges[i].ID, ends[2*i], ends[2*i+1], obstacles, others, xs, ys)
				if points != nil && architectureRouteCost(edges[i].ID, points, others) < architectureRouteCost(edges[i].ID, edges[i].Points, others)-.001 {
					edges[i].Points, changed = points, true
				}
			}
			if !changed {
				break
			}
		}
		return edges, true
	}
	// Compare both insertion orders: a locally optimal early rail can otherwise
	// trap a later fanout in an unnecessary crossing. Select the whole-diagram
	// result, without requiring the caller to annotate ports or routing channels.
	var best []Edge
	bestCost := math.Inf(1)
	for attempt := 0; attempt < 2; attempt++ {
		if edges, ok := plan(); ok {
			segments := []architectureSegment{}
			for _, edge := range edges {
				for j := 1; j < len(edge.Points); j++ {
					segments = append(segments, architectureSegment{edge.Points[j-1], edge.Points[j], edge.ID})
				}
			}
			cost := 0.0
			for _, edge := range edges {
				cost += architectureRouteCost(edge.ID, edge.Points, segments)
			}
			if cost < bestCost-.001 {
				best, bestCost = edges, cost
			}
		}
		for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
			order[i], order[j] = order[j], order[i]
		}
	}
	if best == nil {
		return false
	}
	s.Edges = append(s.Edges, best...)
	return true
}
