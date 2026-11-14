package core

import (
	"math"
	"sort"
	"strings"
)

// Keep the normal path on one axis; alternative paths occupy lanes to its
// right (or below it for LR). Explicit yes/no labels take priority over order.
func flowchartPositions(d Diagram, levels [][]string, dims map[string]Point) map[string]Point {
	items := d.Items()
	indegree := map[string]int{}
	outgoing := map[string][]Relation{}
	for _, r := range d.Model.Relations {
		if r.Source != r.Target {
			indegree[r.Target]++
			outgoing[r.Source] = append(outgoing[r.Source], r)
		}
	}
	rank := func(label string) int {
		switch strings.ToLower(strings.TrimSpace(label)) {
		case "是", "yes", "true", "通过", "成功", "满足", "正常", "有效":
			return 0
		case "否", "no", "false", "失败", "不满足", "异常", "无效":
			return 2
		}
		return 1
	}
	preferred := map[string]string{}
	for source, relations := range outgoing {
		best := relations[0]
		for _, r := range relations[1:] {
			if rank(r.Label) < rank(best.Label) {
				best = r
			}
		}
		preferred[source] = best.Target
	}
	root := ""
	for _, n := range items {
		if indegree[n.ID] == 0 {
			root = n.ID
			break
		}
	}
	if root == "" && len(items) > 0 {
		root = items[0].ID
	}
	main := map[string]bool{}
	for id := root; id != "" && !main[id]; id = preferred[id] {
		main[id] = true
	}
	maxW, maxH := 0.0, 0.0
	for _, size := range dims {
		maxW, maxH = math.Max(maxW, size.X), math.Max(maxH, size.Y)
	}
	positions, lanes := map[string]Point{}, map[string]int{}
	offset := 48.0
	for _, level := range levels {
		row := append([]string{}, level...)
		sort.SliceStable(row, func(i, j int) bool { return main[row[i]] && !main[row[j]] })
		occupied := map[int]bool{}
		extent := 0.0
		for _, id := range row {
			lane := 0
			if !main[id] {
				lane = 1
				for _, r := range d.Model.Relations {
					if r.Target == id {
						lane = max(lane, lanes[r.Source])
					}
				}
			}
			for occupied[lane] {
				lane++
			}
			occupied[lane], lanes[id] = true, lane
			size := dims[id]
			positions[id] = Point{64 + float64(lane)*(maxW+64) + (maxW-size.X)/2, offset}
			extent = math.Max(extent, size.Y)
			if d.View.Direction == "LR" {
				positions[id] = Point{offset, 64 + float64(lane)*(maxH+56) + (maxH-size.Y)/2}
				extent = math.Max(extent, size.X)
			}
		}
		gap := 32.0
		if d.View.Direction == "LR" {
			gap = 48
		}
		offset += extent + gap
	}
	return positions
}
