package core

import "math"

// Includes the name below the stick figure, with room for font descenders.
const actorNodeHeight = 140.0

type useCaseGroup struct {
	items         []Item
	positions     map[string]Point
	width, height float64
	side          int
}

func arrangeUseCaseGroup(items []Item, relations []Relation) useCaseGroup {
	parents := map[string][]string{}
	children := map[string]bool{}
	contains := map[string]bool{}
	for _, n := range items {
		contains[n.ID] = true
	}
	for _, r := range relations {
		base, detail := "", ""
		if r.Kind == "include" {
			base, detail = r.Source, r.Target
		}
		if r.Kind == "extend" {
			base, detail = r.Target, r.Source
		}
		if contains[base] && contains[detail] {
			parents[detail] = append(parents[detail], base)
			children[detail] = true
		}
	}
	positions := map[string]Point{}
	occupied := map[int]map[int]bool{}
	place := func(id string, column, row int) {
		if occupied[column] == nil {
			occupied[column] = map[int]bool{}
		}
		preferred := row
		for offset := 0; ; offset++ {
			if !occupied[column][preferred+offset] {
				row = preferred + offset
				break
			}
			if preferred-offset >= 0 && !occupied[column][preferred-offset] {
				row = preferred - offset
				break
			}
		}
		occupied[column][row] = true
		positions[id] = Point{float64(column), float64(row)}
	}
	rootIndex := 0
	for _, n := range items {
		if !children[n.ID] {
			place(n.ID, rootIndex/8, rootIndex%8)
			rootIndex++
		}
	}
	if rootIndex == 0 {
		place(items[0].ID, 0, 0)
	}
	for len(positions) < len(items) {
		progress := false
		for _, n := range items {
			if _, ok := positions[n.ID]; ok {
				continue
			}
			column, total, count := 0, 0.0, 0
			for _, parent := range parents[n.ID] {
				if p, ok := positions[parent]; ok {
					column = max(column, int(p.X)+1)
					total += p.Y
					count++
				}
			}
			if count > 0 {
				place(n.ID, min(column, 3), int(math.Round(total/float64(count))))
				progress = true
			}
		}
		if !progress {
			for _, n := range items {
				if _, ok := positions[n.ID]; !ok {
					place(n.ID, 0, 0)
					break
				}
			}
		}
	}
	group := useCaseGroup{items: items, positions: map[string]Point{}, side: 0}
	for id, p := range positions {
		position := Point{p.X * 330, p.Y * 104}
		group.positions[id] = position
		group.width = math.Max(group.width, position.X+230)
		group.height = math.Max(group.height, position.Y+62)
	}
	return group
}

// Keep an actor's connected cases together. Unconnected cases fill a separate
// lower area instead of being inserted between a base case and its extensions.
func layoutUseCases(d Diagram, s *Scene, add func(Node), find func(string) Node) {
	parent := map[string]string{}
	caseIDs := map[string]bool{}
	actorSide := map[string]int{}
	for _, n := range d.Model.UseCases {
		parent[n.ID] = n.ID
		caseIDs[n.ID] = true
	}
	for i, a := range d.Model.Actors {
		actorSide[a.ID] = i % 2
	}
	var root func(string) string
	root = func(id string) string {
		if parent[id] != id {
			parent[id] = root(parent[id])
		}
		return parent[id]
	}
	union := func(a, b string) {
		ra, rb := root(a), root(b)
		if ra != rb {
			parent[rb] = ra
		}
	}
	neighbors := map[string][]string{}
	linked := map[string]bool{}
	for _, r := range d.Model.Relations {
		if caseIDs[r.Source] && caseIDs[r.Target] {
			union(r.Source, r.Target)
			linked[r.Source] = true
			linked[r.Target] = true
		}
		if _, ok := actorSide[r.Source]; ok && caseIDs[r.Target] {
			neighbors[r.Source] = append(neighbors[r.Source], r.Target)
			linked[r.Target] = true
		}
		if _, ok := actorSide[r.Target]; ok && caseIDs[r.Source] {
			neighbors[r.Target] = append(neighbors[r.Target], r.Source)
			linked[r.Source] = true
		}
	}
	for _, a := range d.Model.Actors {
		list := neighbors[a.ID]
		for i := 1; i < len(list); i++ {
			union(list[0], list[i])
		}
	}
	components := map[string][]Item{}
	order := []string{}
	orphans := []Item{}
	for _, n := range d.Model.UseCases {
		if !linked[n.ID] {
			orphans = append(orphans, n)
			continue
		}
		key := root(n.ID)
		if _, ok := components[key]; !ok {
			order = append(order, key)
		}
		components[key] = append(components[key], n)
	}
	groups := []useCaseGroup{}
	stackWidth, stackHeight := [2]float64{}, [2]float64{}
	for _, key := range order {
		group := arrangeUseCaseGroup(components[key], d.Model.Relations)
		owners := []Item{}
		leftOwners, rightOwners := 0, 0
		for _, a := range d.Model.Actors {
			for _, id := range neighbors[a.ID] {
				if root(id) == key {
					owners = append(owners, a)
					if actorSide[a.ID] == 0 {
						leftOwners++
					} else {
						rightOwners++
					}
					break
				}
			}
		}
		if len(owners) <= 3 {
			// Actors sharing a small use-case group belong on the same side.
			// Alternating by actor index sends their associations across the system.
			if stackHeight[1] < stackHeight[0] {
				group.side = 1
			}
			for _, a := range owners {
				actorSide[a.ID] = group.side
			}
		} else if rightOwners > leftOwners || leftOwners == rightOwners && stackHeight[1] < stackHeight[0] {
			group.side = 1
		}
		stackWidth[group.side] = math.Max(stackWidth[group.side], group.width)
		if stackHeight[group.side] > 0 {
			stackHeight[group.side] += 56
		}
		stackHeight[group.side] += group.height
		groups = append(groups, group)
	}
	innerWidth := math.Max(stackWidth[0], stackWidth[1])
	if stackWidth[0] > 0 && stackWidth[1] > 0 {
		innerWidth = stackWidth[0] + stackWidth[1] + 100
	}
	orphanColumns := min(4, max(1, int(math.Ceil(math.Sqrt(float64(len(orphans))*104/330)))))
	if len(orphans) > 0 {
		innerWidth = math.Max(innerWidth, float64(orphanColumns-1)*330+230)
	}
	innerWidth = math.Max(innerWidth, 330)
	if len(orphans) > 0 {
		orphanColumns = min(len(orphans), min(4, max(1, int((innerWidth+100)/330))))
	}
	positions := map[string]Point{}
	stackY := [2]float64{140, 140}
	for _, group := range groups {
		for _, n := range group.items {
			p := group.positions[n.ID]
			x := 270 + p.X
			if group.side == 1 {
				x = 270 + innerWidth - 230 - p.X
			}
			positions[n.ID] = Point{x, stackY[group.side] + p.Y}
		}
		stackY[group.side] += group.height + 56
	}
	orphanY := 140.0
	if len(groups) > 0 {
		orphanY += math.Max(stackHeight[0], stackHeight[1]) + 56
	}
	for i, n := range orphans {
		rowStart := i / orphanColumns * orphanColumns
		rowCount := min(orphanColumns, len(orphans)-rowStart)
		x := 270 + (float64(i%orphanColumns)+float64(orphanColumns-rowCount)/2)*330
		positions[n.ID] = Point{x, orphanY + float64(i/orphanColumns)*92}
	}
	left, right, top, bottom := 220.0, 270+innerWidth+50, 80.0, 240.0
	for _, n := range d.Model.UseCases {
		p := positions[n.ID]
		add(Node{ID: n.ID, X: p.X, Y: p.Y, Width: 230, Height: 62, Shape: "ellipse", Title: n.Name, Movable: true})
		actual := find(n.ID)
		left = math.Min(left, actual.X-50)
		right = math.Max(right, actual.X+actual.Width+50)
		top = math.Min(top, actual.Y-50)
		bottom = math.Max(bottom, actual.Y+actual.Height+50)
	}
	left = math.Max(0, left)
	top = math.Max(0, top)
	counts := [2]int{}
	for _, a := range d.Model.Actors {
		if len(neighbors[a.ID]) > 0 {
			counts[actorSide[a.ID]]++
		}
	}
	for _, a := range d.Model.Actors {
		if len(neighbors[a.ID]) == 0 {
			side := 0
			if counts[1] < counts[0] {
				side = 1
			}
			actorSide[a.ID] = side
			counts[side]++
		}
	}
	actorY := useCaseActorPositions(d.Model.Actors, neighbors, actorSide, top, find)
	for _, a := range d.Model.Actors {
		x := math.Max(0, left-165)
		if actorSide[a.ID] == 1 {
			x = right + 65
		}
		y := actorY[a.ID]
		add(Node{ID: a.ID, X: x, Y: y, Width: 90, Height: actorNodeHeight, Shape: "actor", Title: a.Name, Movable: true})
		actual := find(a.ID)
		bottom = math.Max(bottom, actual.Y+actual.Height+35)
	}
	add(Node{ID: "$boundary", X: left, Y: top, Width: right - left, Height: bottom - top, Shape: "boundary", Title: d.Model.SystemName, Movable: false})
	router := newUseCaseRouter(s.Nodes)
	for _, r := range d.Model.Relations {
		label, arrow, dashed := r.Label, "none", false
		if r.Kind == "include" || r.Kind == "extend" {
			label = "«" + r.Kind + "»"
			arrow = "open"
			dashed = true
		}
		a, b := find(r.Source), find(r.Target)
		var points []Point
		if arrow == "none" {
			points = router.route(a, b)
		} else {
			points = router.routeArrow(a, b)
		}
		s.Edges = append(s.Edges, Edge{ID: r.ID, Source: r.Source, Target: r.Target, Points: points, Label: label, Arrow: arrow, Dashed: dashed})
	}
}
