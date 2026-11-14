package core

import (
	"math"
	"unicode/utf8"
)

func Generate(d Diagram) Scene {
	s := Scene{Nodes: []Node{}, Edges: []Edge{}, Type: d.Type, Title: d.Title, Theme: d.View.Theme}
	stroke, fill := "#475569", "#eff4ff"
	if d.View.Theme == "mono" {
		stroke = "#333333"
		fill = "#ffffff"
	}
	if d.View.Theme == "sage" {
		stroke = "#5f7a71"
		fill = "#eef6f0"
	}
	add := func(n Node) {
		if p, ok := d.View.Positions[n.ID]; ok && n.Movable {
			n.X = p.X
			n.Y = p.Y
		}
		n.X = math.Max(0, n.X)
		n.Y = math.Max(0, n.Y)
		n.Fill = fill
		n.Stroke = stroke
		s.Nodes = append(s.Nodes, n)
	}
	find := func(id string) Node {
		for _, n := range s.Nodes {
			if n.ID == id {
				return n
			}
		}
		return Node{}
	}
	connect := func(r Relation, arrow string) {
		a, b := find(r.Source), find(r.Target)
		label := r.Label
		if r.Kind == "include" || r.Kind == "extend" {
			label = "«" + r.Kind + "»"
		}
		if r.Kind == "inheritance" {
			arrow = "triangle"
		}
		points := route(a, b, d.View.Direction)
		var labelPosition *Point
		if d.Type == "flowchart" {
			points, labelPosition = routeFlowchart(a, b, d.View.Direction, .5)
		}
		s.Edges = append(s.Edges, Edge{ID: r.ID, Source: r.Source, Target: r.Target, Points: points, Label: label, LabelPosition: labelPosition, SourceLabel: r.SourceCardinality, TargetLabel: r.TargetCardinality, Arrow: arrow, Dashed: r.Kind == "dependency" || r.Kind == "include" || r.Kind == "extend"})
	}
	switch d.Type {
	case "er":
		layoutER(d, &s, add, find)
	case "usecase":
		layoutUseCases(d, &s, add, find)
	case "sequence":
		layoutSequence(d, &s, add, find)
	case "architecture":
		layoutArchitecture(d, &s, add, find)
	default:
		items := d.Items()
		levels := layers(items, d.Model.Relations)
		maxCount := 1
		for _, row := range levels {
			if len(row) > maxCount {
				maxCount = len(row)
			}
		}
		maxW, maxH := 220.0, 74.0
		dims := map[string]Point{}
		for _, n := range items {
			w, h := textWidth(n.Name, 220, 360), 74.0
			if d.Type != "table" && d.Type != "class" && n.Shape != "decision" && n.Shape != "terminal" {
				w, h = boxTitleSize(n.Name, n.Shape == "database")
			}
			if d.Type == "table" || d.Type == "class" {
				w = 260
				for _, f := range n.Fields {
					w = math.Max(w, textWidth(f.Name+" : "+f.Type, 260, 450))
				}
				for _, m := range n.Methods {
					w = math.Max(w, textWidth(m, 260, 450))
				}
				h = 50 + float64(len(n.Fields))*32 + 18
				if len(n.Methods) > 0 {
					h += 20 + float64(len(n.Methods))*32
				}
				if d.Type == "class" {
					degree := 0
					for _, r := range d.Model.Relations {
						if r.Source == n.ID {
							degree++
						}
						if r.Target == n.ID {
							degree++
						}
					}
					h = math.Max(h, float64(degree)*20+28)
				}
			}
			if n.Shape == "decision" {
				h = 105
				w = 230
			}
			if d.Type == "flowchart" {
				switch n.Shape {
				case "terminal":
					w, h = textWidth(n.Name, 180, 300), 48
				case "decision":
					h = 86
				default:
					h = math.Max(56, float64(len(wrapNodeTitle(n.Name, w-36)))*nodeTitleLineHeight+20)
				}
			}
			dims[n.ID] = Point{w, h}
			maxW = math.Max(maxW, w)
			maxH = math.Max(maxH, h)
		}
		horizontalGap := 110.0
		if d.Type == "table" {
			horizontalGap = 160
			for _, r := range d.Model.Relations {
				horizontalGap = math.Max(horizontalGap, labelWidth(r.Label)+32)
				horizontalGap = math.Max(horizontalGap, labelWidth(r.SourceCardinality)+labelWidth(r.TargetCardinality)+40)
			}
		}
		var layoutPositions map[string]Point
		if d.Type == "flowchart" {
			layoutPositions = flowchartPositions(d, levels, dims)
		} else if d.Type == "class" && d.View.Direction == "LR" {
			layoutPositions = classPositions(d, dims)
		}
		for level, row := range levels {
			for order, id := range row {
				var item Item
				for _, v := range items {
					if v.ID == id {
						item = v
						break
					}
				}
				w, h := dims[id].X, dims[id].Y
				x := 64 + (float64(maxCount-len(row))/2+float64(order))*(maxW+64) + (maxW-w)/2
				y := 64 + float64(level)*(maxH+80)
				if d.View.Direction == "LR" {
					x = 64 + float64(level)*(maxW+horizontalGap)
					y = 64 + (float64(maxCount-len(row))/2+float64(order))*(maxH+64) + (maxH-h)/2
					if d.Type == "table" {
						// Table headers share a baseline even when field counts differ.
						y -= (maxH - h) / 2
					}
				}
				if p, ok := layoutPositions[id]; ok {
					x, y = p.X, p.Y
				}
				shape := "rect"
				if d.Type == "table" || d.Type == "class" {
					shape = "table"
				}
				switch item.Shape {
				case "decision":
					shape = "diamond"
				case "terminal":
					shape = "terminal"
				case "database":
					shape = "database"
				}
				lines := []string{}
				for _, f := range item.Fields {
					prefix := ""
					if f.Primary {
						prefix = "PK  "
					}
					line := prefix + f.Name
					if f.Type != "" {
						line += " : " + f.Type
					}
					lines = append(lines, line)
				}
				add(Node{ID: id, X: x, Y: y, Width: w, Height: h, Shape: shape, Title: item.Name, Lines: lines, Methods: item.Methods, Movable: true})
			}
		}
		if d.Type == "structure" {
			layoutStructureEdges(d, &s, find)
		} else if d.Type == "class" {
			layoutClassEdges(d, &s, find)
		} else if d.Type == "table" || d.Type == "architecture" {
			layoutBoxEdges(d, &s, find)
		} else {
			for _, r := range d.Model.Relations {
				arrow := "open"
				if d.Type == "table" || d.Type == "structure" || d.Type == "class" {
					arrow = "none"
				}
				if r.Kind == "dependency" {
					arrow = "open"
				}
				connect(r, arrow)
			}
		}
		if d.Type == "flowchart" {
			mergeFlowchartEdges(d, &s, add, find)
		}
	}
	placeEdgeLabels(&s)
	// A shared final scene drives both the editor and all vector exporters.
	for i := range s.Nodes {
		n := &s.Nodes[i]
		s.Width = math.Max(s.Width, n.X+n.Width+70)
		s.Height = math.Max(s.Height, n.Y+n.Height+70)
	}
	for i := range s.Edges {
		for j := range s.Edges[i].Points {
			p := &s.Edges[i].Points[j]
			s.Width = math.Max(s.Width, p.X+70)
			s.Height = math.Max(s.Height, p.Y+70)
		}
	}
	s.Width = math.Max(s.Width, 600)
	s.Height = math.Max(s.Height, 420)
	return s
}
func textWidth(s string, min, max float64) float64 {
	return math.Min(max, math.Max(min, float64(utf8.RuneCountInString(s))*10+45))
}
func center(n Node) Point { return Point{n.X + n.Width/2, n.Y + n.Height/2} }
func straight(a, b Node) []Point {
	return []Point{shapeBoundary(a, connectionCenter(b)), shapeBoundary(b, connectionCenter(a))}
}
func route(a, b Node, direction string) []Point {
	ac := center(a)
	if a.ID == b.ID {
		return []Point{shapePort(a, "right", .5), {a.X + a.Width + 70, ac.Y}, {a.X + a.Width + 70, math.Max(0, a.Y-45)}, {ac.X, math.Max(0, a.Y-45)}, shapePort(a, "top", .5)}
	}
	if direction == "LR" {
		start, end := shapePort(a, "right", .5), shapePort(b, "left", .5)
		if b.X >= a.X+a.Width {
			if math.Abs(start.Y-end.Y) < 0.01 {
				return []Point{start, end}
			}
			mid := (start.X + end.X) / 2
			return []Point{start, {mid, start.Y}, {mid, end.Y}, end}
		}
		return straight(a, b)
	}
	start, end := shapePort(a, "bottom", .5), shapePort(b, "top", .5)
	if b.Y >= a.Y+a.Height {
		if math.Abs(start.X-end.X) < 0.01 {
			return []Point{start, end}
		}
		mid := (start.Y + end.Y) / 2
		return []Point{start, {start.X, mid}, {end.X, mid}, end}
	}
	return straight(a, b)
}
func layers(items []Item, relations []Relation) [][]string {
	indeg := map[string]int{}
	adj := map[string][]string{}
	depth := map[string]int{}
	for _, n := range items {
		indeg[n.ID] = 0
	}
	for _, r := range relations {
		if r.Source != r.Target {
			adj[r.Source] = append(adj[r.Source], r.Target)
			indeg[r.Target]++
		}
	}
	q := []string{}
	for _, n := range items {
		if indeg[n.ID] == 0 {
			q = append(q, n.ID)
		}
	}
	seen := map[string]bool{}
	for len(q) > 0 {
		id := q[0]
		q = q[1:]
		seen[id] = true
		for _, next := range adj[id] {
			if depth[next] < depth[id]+1 {
				depth[next] = depth[id] + 1
			}
			indeg[next]--
			if indeg[next] == 0 {
				q = append(q, next)
			}
		}
	}
	maxDepth := 0
	for _, v := range depth {
		if v > maxDepth {
			maxDepth = v
		}
	}
	for _, n := range items {
		if !seen[n.ID] {
			depth[n.ID] = maxDepth + 1
		}
	}
	result := [][]string{}
	for _, n := range items {
		lev := depth[n.ID]
		for len(result) <= lev {
			result = append(result, []string{})
		}
		result[lev] = append(result[lev], n.ID)
	}
	filtered := [][]string{}
	for _, row := range result {
		if len(row) > 0 {
			filtered = append(filtered, row)
		}
	}
	return filtered
}
