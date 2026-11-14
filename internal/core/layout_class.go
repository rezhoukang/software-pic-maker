package core

import (
	"math"
	"sort"
)

// A class reference is not a mandatory left-to-right rank. Center the most
// connected class and balance its branches on both sides, using only topology.
func classPositions(d Diagram, dims map[string]Point) map[string]Point {
	adj, order := map[string][]string{}, map[string]int{}
	maxWidth, gap := 260.0, 160.0
	for i, n := range d.Model.Classes {
		order[n.ID] = i
		maxWidth = math.Max(maxWidth, dims[n.ID].X)
	}
	for _, r := range d.Model.Relations {
		if r.Source != r.Target {
			adj[r.Source] = append(adj[r.Source], r.Target)
			adj[r.Target] = append(adj[r.Target], r.Source)
		}
		gap = math.Max(gap, labelWidth(r.Label)+80)
	}
	for id := range adj {
		sort.SliceStable(adj[id], func(i, j int) bool { return order[adj[id][i]] < order[adj[id][j]] })
	}
	positions, seen := map[string]Point{}, map[string]bool{}
	offsetY := 64.0
	for len(seen) < len(d.Model.Classes) {
		root := ""
		for _, n := range d.Model.Classes {
			if !seen[n.ID] && (root == "" || len(adj[n.ID]) > len(adj[root])) {
				root = n.ID
			}
		}
		children := map[string][]string{}
		var visit func(string)
		visit = func(id string) {
			seen[id] = true
			for _, next := range adj[id] {
				if !seen[next] {
					children[id] = append(children[id], next)
					visit(next)
				}
			}
		}
		visit(root)
		heights := map[string]float64{}
		var measure func(string) float64
		measure = func(id string) float64 {
			total := 0.0
			for i, child := range children[id] {
				if i > 0 {
					total += 80
				}
				total += measure(child)
			}
			heights[id] = math.Max(dims[id].Y, total)
			return heights[id]
		}
		measure(root)
		branches := append([]string{}, children[root]...)
		sort.SliceStable(branches, func(i, j int) bool { return heights[branches[i]] > heights[branches[j]] })
		sides, totals := [2][]string{}, [2]float64{}
		for _, id := range branches {
			side := 0
			if totals[1] < totals[0] {
				side = 1
			}
			if len(sides[side]) > 0 {
				totals[side] += 80
			}
			sides[side] = append(sides[side], id)
			totals[side] += heights[id]
		}
		span := math.Max(dims[root].Y, math.Max(totals[0], totals[1]))
		positions[root] = Point{0, offsetY + (span-dims[root].Y)/2}
		var place func(string, int, int, float64)
		place = func(idString string, side, depth int, top float64) {
			x := float64(side*depth) * (maxWidth + gap)
			y := top + (heights[idString]-dims[idString].Y)/2
			positions[idString] = Point{x, y}
			childTotal := 0.0
			for i, child := range children[idString] {
				if i > 0 {
					childTotal += 80
				}
				childTotal += heights[child]
			}
			cursor := top + (heights[idString]-childTotal)/2
			if len(children[idString]) == 1 && dims[idString].Y >= heights[children[idString][0]] {
				child := children[idString][0]
				cursor = y - (heights[child]-dims[child].Y)/2
			}
			for _, child := range children[idString] {
				place(child, side, depth+1, cursor)
				cursor += heights[child] + 80
			}
		}
		for side, ids := range sides {
			sort.SliceStable(ids, func(i, j int) bool { return order[ids[i]] < order[ids[j]] })
			cursor := offsetY + (span-totals[side])/2
			sign := -1
			if side == 1 {
				sign = 1
			}
			for _, id := range ids {
				place(id, sign, 1, cursor)
				cursor += heights[id] + 80
			}
		}
		offsetY += span + 120
	}
	minX, minY := 0.0, 64.0
	for _, p := range positions {
		minX = math.Min(minX, p.X)
		minY = math.Min(minY, p.Y)
	}
	for id, p := range positions {
		positions[id] = Point{p.X - minX + 64, p.Y - minY + 64}
	}
	return positions
}
