package core

import (
	"math"
	"sort"
)

// Pack actors near their own cases instead of spreading them across the full
// height of the system boundary. Pooling crowded rows balances the displacement
// around their preferred centers while reserving room for each actor's label.
func useCaseActorPositions(actors []Item, neighbors map[string][]string, sides map[string]int, top float64, find func(string) Node) map[string]float64 {
	type row struct {
		id      string
		desired float64
		weight  float64
	}
	type block struct {
		start, count int
		sum          float64
		weight       float64
	}
	result := map[string]float64{}
	spacing := actorNodeHeight + 12
	for side := 0; side < 2; side++ {
		rows := []row{}
		unlinked := 0
		for _, a := range actors {
			if sides[a.ID] != side {
				continue
			}
			desired := top + 24 + float64(unlinked)*spacing
			list := neighbors[a.ID]
			if len(list) > 0 {
				desired = 0
				for _, id := range list {
					desired += center(find(id)).Y
				}
				desired = desired/float64(len(list)) - 48
			} else {
				unlinked++
			}
			// Moving an actor with a broad fan too far from its center can send
			// its lines through sibling cases. Give that center more weight.
			weight := math.Max(1, float64(len(list)*len(list)))
			rows = append(rows, row{a.ID, desired, weight})
		}
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].desired < rows[j].desired })
		blocks := []block{}
		for i, r := range rows {
			blocks = append(blocks, block{i, 1, (r.desired - float64(i)*spacing) * r.weight, r.weight})
			for len(blocks) > 1 {
				last := len(blocks) - 1
				a, b := blocks[last-1], blocks[last]
				if a.sum/a.weight <= b.sum/b.weight {
					break
				}
				blocks[last-1] = block{a.start, a.count + b.count, a.sum + b.sum, a.weight + b.weight}
				blocks = blocks[:last]
			}
		}
		for _, b := range blocks {
			base := math.Max(top+16, b.sum/b.weight)
			for i := b.start; i < b.start+b.count; i++ {
				result[rows[i].id] = base + float64(i)*spacing
			}
		}
	}
	return result
}
