package core

import (
	"math"
	"sort"
)

// Architecture layout uses only topology and measured content. No role names,
// layer annotations, dummy model nodes, or agent-supplied coordinates are needed.
func architectureLayers(items []Item, relations []Relation) [][]string {
	index, adjacency := map[string]int{}, map[string][]string{}
	for i, n := range items {
		index[n.ID] = i
	}
	seenPair := map[[2]string]bool{}
	for _, r := range relations {
		_, source := index[r.Source]
		_, target := index[r.Target]
		pair := [2]string{r.Source, r.Target}
		if source && target && r.Source != r.Target && r.Kind != "return" && !seenPair[pair] {
			adjacency[r.Source] = append(adjacency[r.Source], r.Target)
			seenPair[pair] = true
		}
	}
	for id := range adjacency {
		sort.Slice(adjacency[id], func(i, j int) bool { return index[adjacency[id][i]] < index[adjacency[id][j]] })
	}
	// Exclude DFS back edges from ranking, but keep every relation for routing.
	// Otherwise one feedback relation collapses all downstream layers together.
	color, forward := map[string]int{}, map[string][]string{}
	var visit func(string)
	visit = func(id string) {
		color[id] = 1
		for _, next := range adjacency[id] {
			if color[next] == 1 {
				continue
			}
			forward[id] = append(forward[id], next)
			if color[next] == 0 {
				visit(next)
			}
		}
		color[id] = 2
	}
	for _, n := range items {
		if color[n.ID] == 0 {
			visit(n.ID)
		}
	}
	indegree, depth := map[string]int{}, map[string]int{}
	for _, targets := range forward {
		for _, id := range targets {
			indegree[id]++
		}
	}
	queue := []string{}
	for _, n := range items {
		if indegree[n.ID] == 0 {
			queue = append(queue, n.ID)
		}
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, next := range forward[id] {
			depth[next] = max(depth[next], depth[id]+1)
			indegree[next]--
			if indegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	rows := [][]string{}
	for _, n := range items {
		for len(rows) <= depth[n.ID] {
			rows = append(rows, []string{})
		}
		rows[depth[n.ID]] = append(rows[depth[n.ID]], n.ID)
	}
	architectureOrder(rows, relations)
	return rows
}

func architectureOrder(rows [][]string, relations []Relation) {
	positions := func() map[string]Point {
		p := map[string]Point{}
		for level, row := range rows {
			for order, id := range row {
				p[id] = Point{float64(level), float64(order+1) / float64(len(row)+1)}
			}
		}
		return p
	}
	connections := [][2]string{}
	initial := positions()
	for _, r := range relations {
		if initial[r.Source].X < initial[r.Target].X && r.Kind != "return" {
			connections = append(connections, [2]string{r.Source, r.Target})
		}
	}
	crossings := func() int {
		p, count := positions(), 0
		for i, edge := range connections {
			a, b := p[edge[0]], p[edge[1]]
			for _, other := range connections[i+1:] {
				if edge[0] == other[0] || edge[1] == other[1] || edge[0] == other[1] || edge[1] == other[0] {
					continue
				}
				c, d := p[other[0]], p[other[1]]
				cross := func(a, b, c Point) float64 { return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X) }
				if cross(a, b, c)*cross(a, b, d) < -.000001 && cross(c, d, a)*cross(c, d, b) < -.000001 {
					count++
				}
			}
		}
		return count
	}
	best := crossings()
	for pass := 0; pass < 4; pass++ {
		for step := range len(rows) {
			level := step
			if pass%2 == 1 {
				level = len(rows) - step - 1
			}
			row, before, p := rows[level], append([]string{}, rows[level]...), positions()
			centers := map[string]float64{}
			for _, id := range row {
				total, count := 0.0, 0.0
				for _, edge := range connections {
					if pass%2 == 0 && edge[1] == id || pass%2 == 1 && edge[0] == id {
						other := edge[0]
						if pass%2 == 1 {
							other = edge[1]
						}
						total += p[other].Y
						count++
					}
				}
				centers[id] = p[id].Y
				if count > 0 {
					centers[id] = total / count
				}
			}
			sort.SliceStable(row, func(i, j int) bool { return centers[row[i]] < centers[row[j]] })
			if score := crossings(); score <= best {
				best = score
			} else {
				copy(row, before)
			}
			for i := 1; i < len(row); i++ {
				row[i-1], row[i] = row[i], row[i-1]
				if score := crossings(); score < best {
					best = score
				} else {
					row[i-1], row[i] = row[i], row[i-1]
				}
			}
		}
	}
}

func layoutArchitecture(d Diagram, s *Scene, add func(Node), find func(string) Node) {
	rows := architectureLayers(d.Model.Nodes, d.Model.Relations)
	if len(rows) == 0 {
		return
	}
	rank, dimensions, items := map[string]int{}, map[string]Point{}, map[string]Item{}
	for level, row := range rows {
		for _, id := range row {
			rank[id] = level
		}
	}
	incoming, outgoing := map[string]int{}, map[string]int{}
	for _, r := range d.Model.Relations {
		outgoing[r.Source]++
		incoming[r.Target]++
	}
	horizontal := d.View.Direction == "LR"
	for _, n := range d.Model.Nodes {
		items[n.ID] = n
		w := math.Max(176, math.Min(360, math.Ceil(titleMeasuredWidth(n.Name)+36)))
		padding := 24.0
		if n.Shape == "database" {
			padding = 50
		}
		h := math.Max(64, float64(len(wrapNodeTitle(n.Name, w-36)))*nodeTitleLineHeight+padding)
		if n.Shape == "decision" {
			w, h = math.Max(w, 230), math.Max(h, 105)
		}
		// Grow high-degree outlines so independent arrowheads have room.
		ports := float64(incoming[n.ID]+outgoing[n.ID]+1)*16 + 14
		w, h = math.Max(w, ports), math.Max(h, ports)
		dimensions[n.ID] = Point{w, h}
	}
	primary, cross, sizes := make([]float64, len(rows)), map[string]float64{}, make([]float64, len(rows))
	maxCross := 0.0
	for level, row := range rows {
		for i, id := range row {
			dim := dimensions[id]
			length, width := dim.X, dim.Y
			if !horizontal {
				length, width = dim.Y, dim.X
			}
			sizes[level] = math.Max(sizes[level], length)
			cross[id] = primary[level]
			primary[level] += width
			if i+1 < len(row) {
				primary[level] += 80
			}
		}
		maxCross = math.Max(maxCross, primary[level])
	}
	for level, row := range rows {
		for _, id := range row {
			cross[id] += 64 + (maxCross-primary[level])/2
		}
	}
	maxCross = alignArchitectureContinuations(d, rows, dimensions, cross)
	// Leave enough room for text and a small number of short detours; straight
	// connectors no longer need a separate vertical maze track for every edge.
	primary[0] = 64
	for level := 0; level+1 < len(rows); level++ {
		count, label := 0, 0.0
		for _, r := range d.Model.Relations {
			a, b := rank[r.Source], rank[r.Target]
			if min(a, b) <= level && max(a, b) > level {
				count++
				label = math.Max(label, labelWidth(r.Label))
			}
		}
		gap := math.Max(96, math.Max(32+float64(count)*14, label+40))
		primary[level+1] = primary[level] + sizes[level] + gap
	}
	firstNode, firstEdge := len(s.Nodes), len(s.Edges)
	for attempt := 0; attempt < 4; attempt++ {
		s.Nodes, s.Edges = s.Nodes[:firstNode], s.Edges[:firstEdge]
		for level, row := range rows {
			for _, id := range row {
				n, dim := items[id], dimensions[id]
				along := primary[level] + float64(attempt*level)*24
				across := cross[id] + float64(attempt*(level%2))*(maxCross/2+40)
				x, y := along, across
				if !horizontal {
					x, y = across, along
				}
				shape := "rect"
				switch n.Shape {
				case "database":
					shape = "database"
				case "decision":
					shape = "diamond"
				case "terminal":
					shape = "terminal"
				}
				add(Node{ID: id, X: x, Y: y, Width: dim.X, Height: dim.Y, Shape: shape, Title: n.Name, Movable: true})
			}
		}
		if layoutArchitectureEdges(d, s, find, rows, false) {
			return
		}
		if len(d.View.Positions) > 0 {
			// User-defined overlapping positions may make the constraints incompatible.
			// Preserve those positions and logical edges; automatic layout clears them.
			layoutArchitectureEdges(d, s, find, rows, true)
			return
		}
	}
	// A sparse strip always admits a two-corner outside route. Use it only when
	// compact layered placements cannot clear the components within the bend cap.
	s.Nodes, s.Edges = s.Nodes[:firstNode], s.Edges[:firstEdge]
	cursor := 64.0
	for _, row := range rows {
		for _, id := range row {
			n, dim := items[id], dimensions[id]
			x, y := cursor, 64.0
			if !horizontal {
				x, y = 64, cursor
			}
			shape := "rect"
			if n.Shape == "database" {
				shape = "database"
			} else if n.Shape == "decision" {
				shape = "diamond"
			} else if n.Shape == "terminal" {
				shape = "terminal"
			}
			add(Node{ID: id, X: x, Y: y, Width: dim.X, Height: dim.Y, Shape: shape, Title: n.Name, Movable: true})
			if horizontal {
				cursor += dim.X + 96
			} else {
				cursor += dim.Y + 96
			}
		}
	}
	layoutArchitectureEdges(d, s, find, rows, false)
}
