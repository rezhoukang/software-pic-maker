package core

import (
	"container/heap"
	"math"
	"sort"
)

type routeBox struct {
	left, right, top, bottom float64
	id                       string
}

func (b routeBox) contains(p Point) bool {
	return p.X > b.left+.001 && p.X < b.right-.001 && p.Y > b.top+.001 && p.Y < b.bottom-.001
}
func (b routeBox) crosses(a, z Point) bool {
	lo, hi := 0.0, 1.0
	for _, v := range [][4]float64{{a.X, z.X - a.X, b.left + .001, b.right - .001}, {a.Y, z.Y - a.Y, b.top + .001, b.bottom - .001}} {
		if math.Abs(v[1]) < .00001 {
			if v[0] <= v[2] || v[0] >= v[3] {
				return false
			}
			continue
		}
		x, y := (v[2]-v[0])/v[1], (v[3]-v[0])/v[1]
		if x > y {
			x, y = y, x
		}
		lo = math.Max(lo, x)
		hi = math.Min(hi, y)
		if lo >= hi {
			return false
		}
	}
	return hi > 0 && lo < 1
}

type routePort struct {
	point, stub Point
	axis        int
}

func useCasePorts(n Node) []routePort {
	if n.Shape == "actor" {
		c := connectionCenter(n)
		return []routePort{{shapePort(n, "left", .5), Point{c.X - 67, c.Y}, 0}, {shapePort(n, "right", .5), Point{c.X + 67, c.Y}, 0}}
	}
	return []routePort{portWithStub(n, "left", .5), portWithStub(n, "right", .5), portWithStub(n, "top", .5), portWithStub(n, "bottom", .5)}
}

type routeArc struct {
	to, axis int
	length   float64
}
type useCaseRouter struct {
	nodes          []Node
	extraPorts     []routePort
	boxes          []routeBox
	xs, ys         []float64
	xIndex, yIndex map[float64]int
	points         []Point
	free           []bool
	adj            [][]routeArc
}

func newUseCaseRouter(nodes []Node) *useCaseRouter {
	r := &useCaseRouter{}
	for _, n := range nodes {
		if n.Shape != "boundary" {
			r.nodes = append(r.nodes, n)
			r.boxes = append(r.boxes, routeBox{n.X - 16, n.X + n.Width + 16, n.Y - 16, n.Y + n.Height + 16, n.ID})
		}
	}
	return r
}
func (r *useCaseRouter) clear(a, b Point, skipA, skipB string) bool {
	for _, box := range r.boxes {
		if box.id != skipA && box.id != skipB && box.crosses(a, b) {
			return false
		}
	}
	return true
}

func (r *useCaseRouter) directClear(a, b Point, skipA, skipB string) bool {
	for _, box := range r.boxes {
		if box.id == skipA || box.id == skipB {
			continue
		}
		// A visible straight association should not be detoured just because it
		// grazes the wider clearance reserved for orthogonal routing.
		tight := routeBox{left: box.left + 12, right: box.right - 12, top: box.top + 12, bottom: box.bottom - 12}
		if tight.crosses(a, b) {
			return false
		}
	}
	return true
}
func (r *useCaseRouter) blocked(p Point) bool {
	for _, box := range r.boxes {
		if box.contains(p) {
			return true
		}
	}
	return false
}
func uniqueCoordinates(values []float64) []float64 {
	sort.Float64s(values)
	out := []float64{}
	for _, x := range values {
		if x >= 0 && (len(out) == 0 || x != out[len(out)-1]) {
			out = append(out, x)
		}
	}
	return out
}
func (r *useCaseRouter) prepare() {
	if r.points != nil {
		return
	}
	xs, ys := []float64{0}, []float64{0}
	maxX, maxY := 0.0, 0.0
	for _, box := range r.boxes {
		xs = append(xs, box.left, box.right)
		ys = append(ys, box.top, box.bottom)
		maxX = math.Max(maxX, box.right)
		maxY = math.Max(maxY, box.bottom)
	}
	for _, n := range r.nodes {
		for _, p := range useCasePorts(n) {
			xs = append(xs, p.stub.X)
			ys = append(ys, p.stub.Y)
		}
	}
	for _, p := range r.extraPorts {
		xs = append(xs, p.stub.X)
		ys = append(ys, p.stub.Y)
	}
	xs = append(xs, maxX+40)
	ys = append(ys, maxY+40)
	r.xs, r.ys = uniqueCoordinates(xs), uniqueCoordinates(ys)
	r.xIndex, r.yIndex = map[float64]int{}, map[float64]int{}
	for i, x := range r.xs {
		r.xIndex[x] = i
	}
	for i, y := range r.ys {
		r.yIndex[y] = i
	}
	width := len(r.xs)
	total := width * len(r.ys)
	r.points = make([]Point, total)
	r.free = make([]bool, total)
	r.adj = make([][]routeArc, total)
	for y, py := range r.ys {
		for x, px := range r.xs {
			id := y*width + x
			p := Point{px, py}
			r.points[id] = p
			r.free[id] = !r.blocked(p)
		}
	}
	link := func(a, b, axis int) {
		if !r.free[a] || !r.free[b] {
			return
		}
		p, q := r.points[a], r.points[b]
		if r.blocked(Point{(p.X + q.X) / 2, (p.Y + q.Y) / 2}) {
			return
		}
		length := math.Abs(p.X-q.X) + math.Abs(p.Y-q.Y)
		r.adj[a] = append(r.adj[a], routeArc{b, axis, length})
		r.adj[b] = append(r.adj[b], routeArc{a, axis, length})
	}
	for y := range r.ys {
		for x := range r.xs {
			id := y*width + x
			if x+1 < width {
				link(id, id+1, 0)
			}
			if y+1 < len(r.ys) {
				link(id, id+width, 1)
			}
		}
	}
}
func (r *useCaseRouter) index(p Point) (int, bool) {
	x, ok := r.xIndex[p.X]
	if !ok {
		return 0, false
	}
	y, ok := r.yIndex[p.Y]
	if !ok {
		return 0, false
	}
	i := y*len(r.xs) + x
	return i, r.free[i]
}

type routeItem struct {
	state          int
	cost, estimate float64
}
type routeQueue []routeItem

func (q routeQueue) Len() int { return len(q) }
func (q routeQueue) Less(i, j int) bool {
	if q[i].estimate == q[j].estimate {
		return q[i].state < q[j].state
	}
	return q[i].estimate < q[j].estimate
}
func (q routeQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q *routeQueue) Push(v any)   { *q = append(*q, v.(routeItem)) }
func (q *routeQueue) Pop() any     { old := *q; v := old[len(old)-1]; *q = old[:len(old)-1]; return v }
func simplifyRoute(points []Point) []Point {
	out := []Point{}
	for _, p := range points {
		if len(out) > 0 && math.Hypot(out[len(out)-1].X-p.X, out[len(out)-1].Y-p.Y) < .001 {
			continue
		}
		for len(out) >= 2 {
			a, b := out[len(out)-2], out[len(out)-1]
			cross := (b.X-a.X)*(p.Y-b.Y) - (b.Y-a.Y)*(p.X-b.X)
			dot := (b.X-a.X)*(p.X-b.X) + (b.Y-a.Y)*(p.Y-b.Y)
			if math.Abs(cross) > .001 || dot < 0 {
				break
			}
			out = out[:len(out)-1]
		}
		out = append(out, p)
	}
	return out
}
func (r *useCaseRouter) route(a, b Node) []Point {
	direct := straight(a, b)
	if r.directClear(direct[0], direct[1], a.ID, b.ID) {
		return direct
	}
	return r.obliqueRoute(a, b, r.routePorts(a, b, useCasePorts(a), useCasePorts(b)), false)
}
func (r *useCaseRouter) routeArrow(a, b Node) []Point {
	if b.Shape == "ellipse" {
		end := ellipseNormalBoundary(b, connectionCenter(a))
		start := shapeBoundary(a, end)
		if r.directClear(start, end, a.ID, b.ID) {
			return []Point{start, end}
		}
	}
	return r.obliqueRoute(a, b, r.routePorts(a, b, useCasePorts(a), useCasePorts(b)), true)
}

// Use-case relationships should read as straight or oblique lines. The shared
// obstacle router also serves box diagrams, so remove its orthogonal detours
// here only, while retaining obstacle clearance and normal arrow approaches.
func (r *useCaseRouter) obliqueRoute(a, b Node, points []Point, arrow bool) []Point {
	if len(points) < 3 {
		return points
	}
	visible := func(i, j int) bool {
		p, q := points[i], points[j]
		skipA, skipB := "", ""
		if i == 0 {
			skipA = a.ID
			c := connectionCenter(a)
			if (q.X-p.X)*(p.X-c.X)+(q.Y-p.Y)*(p.Y-c.Y) < -.001 {
				return false
			}
		}
		if j == len(points)-1 {
			if arrow && i != j-1 {
				return false
			}
			skipB = b.ID
			c := connectionCenter(b)
			if (p.X-q.X)*(q.X-c.X)+(p.Y-q.Y)*(q.Y-c.Y) < -.001 {
				return false
			}
		}
		return r.clear(p, q, skipA, skipB)
	}
	short := []Point{points[0]}
	for i := 0; i < len(points)-1; {
		j := len(points) - 1
		for j > i+1 && !visible(i, j) {
			j--
		}
		short = append(short, points[j])
		i = j
	}
	// Rare tight passages can still retain a right angle. A small diagonal
	// through its reserved 16px clearance keeps the actual shapes unobstructed.
	out := []Point{short[0]}
	for i := 1; i < len(short)-1; i++ {
		p, q, z := short[i-1], short[i], short[i+1]
		dx, dy, ex, ey := q.X-p.X, q.Y-p.Y, z.X-q.X, z.Y-q.Y
		before, after := math.Hypot(dx, dy), math.Hypot(ex, ey)
		if before > .001 && after > .001 && math.Abs(dx*ex+dy*ey) < .00001*before*after {
			cut := math.Min(6, math.Min(before, after)/3)
			out = append(out, Point{q.X - dx/before*cut, q.Y - dy/before*cut}, Point{q.X + ex/after*cut, q.Y + ey/after*cut})
		} else {
			out = append(out, q)
		}
	}
	return simplifyRoute(append(out, short[len(short)-1]))
}

func (r *useCaseRouter) routePorts(a, b Node, sources, targets []routePort) []Point {
	direct := straight(a, b)
	if len(sources) == 1 && len(targets) == 1 {
		p, q := sources[0], targets[0]
		direct = []Point{p.point, p.stub}
		if p.axis == 0 {
			direct = append(direct, Point{q.stub.X, p.stub.Y})
		} else {
			direct = append(direct, Point{p.stub.X, q.stub.Y})
		}
		direct = simplifyRoute(append(direct, q.stub, q.point))
	}
	r.prepare()
	goals := map[int][]int{}
	for i, p := range targets {
		if id, ok := r.index(p.stub); ok && r.clear(p.point, p.stub, b.ID, "") {
			goals[id] = append(goals[id], i)
		}
	}
	if len(goals) == 0 {
		return direct
	}
	heuristic := func(p Point) float64 {
		best := math.Inf(1)
		for id := range goals {
			q := r.points[id]
			best = math.Min(best, math.Abs(p.X-q.X)+math.Abs(p.Y-q.Y))
		}
		return best
	}
	distance := make([]float64, len(r.points)*2)
	previous := make([]int, len(distance))
	origin := make([]int, len(distance))
	for i := range distance {
		distance[i] = math.Inf(1)
		previous[i] = -1
		origin[i] = -1
	}
	queue := &routeQueue{}
	heap.Init(queue)
	for i, p := range sources {
		if id, ok := r.index(p.stub); ok && r.clear(p.point, p.stub, a.ID, "") {
			state := id*2 + p.axis
			cost := math.Hypot(p.stub.X-p.point.X, p.stub.Y-p.point.Y)
			if cost < distance[state] {
				distance[state] = cost
				origin[state] = i
				heap.Push(queue, routeItem{state, cost, cost + heuristic(p.stub)})
			}
		}
	}
	best, state, target := math.Inf(1), -1, -1
	for queue.Len() > 0 {
		current := heap.Pop(queue).(routeItem)
		if current.cost > distance[current.state]+.001 {
			continue
		}
		if current.estimate >= best {
			break
		}
		id, axis := current.state/2, current.state%2
		for _, i := range goals[id] {
			p := targets[i]
			cost := current.cost + math.Hypot(p.stub.X-p.point.X, p.stub.Y-p.point.Y)
			if axis != p.axis {
				cost += 32
			}
			if cost < best {
				best, state, target = cost, current.state, i
			}
		}
		for _, arc := range r.adj[id] {
			next := arc.to*2 + arc.axis
			cost := current.cost + arc.length
			if axis != arc.axis {
				cost += 32
			}
			if cost+.001 < distance[next] {
				distance[next] = cost
				previous[next] = current.state
				origin[next] = origin[current.state]
				heap.Push(queue, routeItem{next, cost, cost + heuristic(r.points[arc.to])})
			}
		}
	}
	if state < 0 {
		return direct
	}
	path := []Point{}
	start := sources[origin[state]]
	for cursor := state; cursor >= 0; cursor = previous[cursor] {
		path = append(path, r.points[cursor/2])
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	points := append([]Point{start.point, start.stub}, path...)
	points = append(points, targets[target].stub, targets[target].point)
	return simplifyRoute(points)
}
