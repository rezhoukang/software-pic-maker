package core

import (
	"math"
	"sort"
)

// Reserve a straight continuation through the longer downstream branch instead
// of independently centering every rank. A wide sibling must not pull a leaf's
// parent and child away from their common axis. Layer order and gaps stay fixed.
func alignArchitectureContinuations(d Diagram, rows [][]string, dims map[string]Point, cross map[string]float64) float64 {
	rank, index := map[string]int{}, map[string]int{}
	for level, row := range rows {
		for _, id := range row {
			rank[id] = level
		}
	}
	for i, n := range d.Model.Nodes {
		index[n.ID] = i
	}
	length := func(id string) float64 {
		if d.View.Direction == "LR" {
			return dims[id].Y
		}
		return dims[id].X
	}
	depth, next := map[string]int{}, map[string]string{}
	for level := len(rows) - 1; level >= 0; level-- {
		for _, id := range rows[level] {
			best := -1
			for _, r := range d.Model.Relations {
				if r.Source == id && rank[r.Target] > level && r.Kind != "return" && depth[r.Target] > best {
					best = depth[r.Target]
					next[id] = r.Target
				}
			}
			depth[id] = best + 1
		}
	}
	type pair struct{ a, b string }
	pairs := []pair{}
	for _, n := range d.Model.Nodes {
		if target, ok := next[n.ID]; ok {
			pairs = append(pairs, pair{n.ID, target})
		}
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		a, b := pairs[i], pairs[j]
		rigidity := func(p pair) int { return max(len(rows[rank[p.a]]), len(rows[rank[p.b]])) }
		if rigidity(a) != rigidity(b) {
			return rigidity(a) > rigidity(b)
		}
		if depth[a.a] != depth[b.a] {
			return depth[a.a] > depth[b.a]
		}
		return index[a.a] < index[b.a]
	})
	fits := func(id string, position float64) bool {
		row := rows[rank[id]]
		for i, other := range row {
			if other != id {
				continue
			}
			if i > 0 && position < cross[row[i-1]]+length(row[i-1])+80-.001 {
				return false
			}
			if i+1 < len(row) && position+length(id)+80 > cross[row[i+1]]+.001 {
				return false
			}
		}
		return true
	}
	locked := map[string]bool{}
	for pass := 0; pass < 3; pass++ {
		changed := false
		for _, p := range pairs {
			a, b := cross[p.a]+length(p.a)/2, cross[p.b]+length(p.b)/2
			if math.Abs(a-b) < .001 {
				locked[p.a], locked[p.b] = true, true
				continue
			}
			move, anchor := p.b, p.a
			if locked[p.b] || !locked[p.a] && len(rows[rank[p.a]]) < len(rows[rank[p.b]]) {
				move, anchor = p.a, p.b
			}
			for attempt := 0; attempt < 2; attempt++ {
				position := cross[anchor] + (length(anchor)-length(move))/2
				if !locked[move] && fits(move, position) {
					cross[move] = position
					locked[move], locked[anchor] = true, true
					changed = true
					break
				}
				move, anchor = anchor, move
			}
		}
		if !changed {
			break
		}
	}
	minimum, maximum := math.Inf(1), math.Inf(-1)
	for id, position := range cross {
		minimum = math.Min(minimum, position)
		maximum = math.Max(maximum, position+length(id))
	}
	for id := range cross {
		cross[id] += 64 - minimum
	}
	return maximum - minimum
}
