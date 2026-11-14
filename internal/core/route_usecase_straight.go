package core

import "math"

// Use-case connectors have exactly two endpoints. Obstacles affect endpoint
// selection and automatic placement, never introduce intermediate vertices.
func useCaseStraightPorts(n, other Node) []Point {
	if n.Shape == "actor" {
		return []Point{shapeBoundary(n, connectionCenter(other))}
	}
	ports := []Point{ellipseNormalBoundary(n, connectionCenter(other)), shapeBoundary(n, connectionCenter(other))}
	c := center(n)
	for i := range 32 {
		angle := float64(i) * math.Pi / 16
		p := Point{c.X + (n.Width/2-2)*math.Cos(angle), c.Y + (n.Height/2-2)*math.Sin(angle)}
		ports = append(ports, p)
	}
	return ports
}

func useCaseIncidence(n Node, at, other Point) float64 {
	if n.Shape == "actor" {
		return 1
	}
	c := center(n)
	normal := Point{(at.X - c.X) / math.Pow(n.Width/2-2, 2), (at.Y - c.Y) / math.Pow(n.Height/2-2, 2)}
	delta := Point{other.X - at.X, other.Y - at.Y}
	return (normal.X*delta.X + normal.Y*delta.Y) / math.Max(.0001, math.Hypot(normal.X, normal.Y)*math.Hypot(delta.X, delta.Y))
}

func useCaseArrowClear(n Node, tip, from Point) bool {
	if n.Shape != "ellipse" {
		return true
	}
	c := center(n)
	dx, dy := from.X-tip.X, from.Y-tip.Y
	length := math.Hypot(dx, dy)
	if length < 20 {
		return false
	}
	dx, dy = dx/length, dy/length
	for _, side := range []float64{-1, 1} {
		for _, step := range []float64{2, 5, 10} {
			p := Point{tip.X + dx*step - side*dy*step/2, tip.Y + dy*step + side*dx*step/2}
			if math.Pow((p.X-c.X)/(n.Width/2-2), 2)+math.Pow((p.Y-c.Y)/(n.Height/2-2), 2) <= 1 {
				return false
			}
		}
	}
	return true
}

func routeUseCaseStraight(a, b Node, arrow bool, nodes []Node, used []Edge) []Point {
	best, bestCost := straight(a, b), math.Inf(1)
	for _, start := range useCaseStraightPorts(a, b) {
		for _, end := range useCaseStraightPorts(b, a) {
			incidenceA, incidenceB := useCaseIncidence(a, start, end), useCaseIncidence(b, end, start)
			if incidenceA < .2 || incidenceB < .2 || arrow && !useCaseArrowClear(b, end, start) {
				continue
			}
			cost := math.Hypot(end.X-start.X, end.Y-start.Y)*.08 + (2-incidenceA-incidenceB)*30
			for _, n := range nodes {
				if n.Shape == "boundary" || n.ID == a.ID || n.ID == b.ID {
					continue
				}
				if (routeBox{left: n.X - 3, right: n.X + n.Width + 3, top: n.Y - 3, bottom: n.Y + n.Height + 3}).crosses(start, end) {
					cost += 100000000
				}
			}
			for _, other := range used {
				p, q := other.Points[0], other.Points[len(other.Points)-1]
				if overlap := architectureOverlap(start, end, p, q); overlap > .01 {
					cost += 1000000 + overlap*100
				}
				if architectureIntersects(start, end, p, q) {
					cost += 80
				}
				// Arrow tips need independent landing points, including where an
				// association and an include/extend relationship share a use case.
				for _, attached := range []struct {
					id string
					p  Point
				}{{other.Source, p}, {other.Target, q}} {
					if attached.id == b.ID && (arrow || other.Arrow != "none") {
						gap := math.Hypot(end.X-attached.p.X, end.Y-attached.p.Y)
						if gap < 18 {
							cost += (18 - gap) * 5000
						}
					}
					if attached.id == a.ID && a.Shape != "actor" {
						gap := math.Hypot(start.X-attached.p.X, start.Y-attached.p.Y)
						if gap < 14 {
							cost += (14 - gap) * 2000
						}
					}
				}
			}
			if cost < bestCost-.001 {
				best, bestCost = []Point{start, end}, cost
			}
		}
	}
	return best
}
