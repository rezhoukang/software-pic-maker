package core

import (
	"math"
	"sort"
)

func layoutClassEdges(d Diagram, s *Scene, find func(string) Node) {
	ends := make([]*architectureEnd, 0, len(d.Model.Relations)*2)
	groups := map[string][]*architectureEnd{}
	for _, r := range d.Model.Relations {
		a, b := find(r.Source), find(r.Target)
		out, in := "right", "left"
		if b.X+b.Width <= a.X {
			out, in = "left", "right"
		} else if b.X < a.X+a.Width {
			out, in = "right", "right"
		}
		for _, end := range []*architectureEnd{{node: a, other: b, id: r.ID + ":source", preferred: out}, {node: b, other: a, id: r.ID + ":target", preferred: in}} {
			ends = append(ends, end)
			key := end.node.ID + ":" + end.preferred
			groups[key] = append(groups[key], end)
		}
	}
	for _, group := range groups {
		sort.SliceStable(group, func(i, j int) bool {
			a, b := center(group[i].other), center(group[j].other)
			if a.Y != b.Y {
				return a.Y < b.Y
			}
			return group[i].id < group[j].id
		})
		for i, end := range group {
			p := shapePort(end.node, end.preferred, .5)
			p.Y = end.node.Y + 22
			if len(group) > 1 {
				p.Y = end.node.Y + 14 + float64(i)*20
			}
			end.ports = []architecturePort{{point: p, side: end.preferred}}
			end.portRange = &[2]float64{end.node.Y + 8, end.node.Y + math.Max(40, 22+float64(len(group)-1)*20)}
		}
	}
	architectureAlignPorts(ends, s.Nodes)
	xs := []float64{16}
	maxX := 0.0
	for _, n := range s.Nodes {
		xs = append(xs, math.Max(16, n.X-24), n.X+n.Width+24)
		maxX = math.Max(maxX, n.X+n.Width)
	}
	for x := 30.0; x <= maxX+32+float64(len(d.Model.Relations))*16; x += 14 {
		xs = append(xs, x)
	}
	xs = uniqueCoordinates(xs)
	order := make([]int, len(d.Model.Relations))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(i, j int) bool {
		a, b := order[i], order[j]
		distance := func(k int) float64 {
			p, q := center(ends[2*k].node), center(ends[2*k+1].node)
			return math.Hypot(p.X-q.X, p.Y-q.Y)
		}
		if distance(a) != distance(b) {
			return distance(a) < distance(b)
		}
		return d.Model.Relations[a].ID < d.Model.Relations[b].ID
	})
	edges := make([]Edge, len(order))
	used := []architectureSegment{}
	var fallback *useCaseRouter
	for _, i := range order {
		r := d.Model.Relations[i]
		a, b := ends[2*i], ends[2*i+1]
		points := architectureChooseRoute(r.ID, a, b, s.Nodes, used, xs, nil)
		if points == nil {
			// Preserve saved positions even when obstacles require a longer detour;
			// the terminal ports still stay exclusively on the left/right faces.
			if fallback == nil {
				fallback = newUseCaseRouter(s.Nodes)
				for _, end := range ends {
					p := end.ports[0]
					stub := p.point
					stub.X += architectureNormal(p.side).X * 24
					fallback.extraPorts = append(fallback.extraPorts, routePort{p.point, stub, 0})
				}
			}
			port := func(end *architectureEnd) routePort {
				p := end.ports[0]
				stub := p.point
				stub.X += architectureNormal(p.side).X * 24
				return routePort{p.point, stub, 0}
			}
			points = fallback.routePorts(a.node, b.node, []routePort{port(a)}, []routePort{port(b)})
		}
		for j := 1; j < len(points); j++ {
			used = append(used, architectureSegment{points[j-1], points[j], r.ID})
		}
		arrow := "none"
		if r.Kind == "inheritance" {
			arrow = "triangle"
		} else if r.Kind == "dependency" {
			arrow = "open"
		}
		edges[i] = Edge{ID: r.ID, Source: r.Source, Target: r.Target, Points: points, Label: r.Label, SourceLabel: r.SourceCardinality, TargetLabel: r.TargetCardinality, Arrow: arrow, Dashed: r.Kind == "dependency"}
	}
	s.Edges = append(s.Edges, edges...)
}
