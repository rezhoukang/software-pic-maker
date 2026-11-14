package core

import "math"

type rotatedLabel struct {
	center               Point
	width, height, angle float64
}

func (b rotatedLabel) local(p Point) Point {
	a := b.angle * math.Pi / 180
	x, y := p.X-b.center.X, p.Y-b.center.Y
	return Point{x*math.Cos(a) + y*math.Sin(a), -x*math.Sin(a) + y*math.Cos(a)}
}

func (b rotatedLabel) crosses(a, z Point) bool {
	return (routeBox{left: -b.width / 2, right: b.width / 2, top: -b.height / 2, bottom: b.height / 2}).crosses(b.local(a), b.local(z))
}

func (b rotatedLabel) bounds() routeBox {
	a := b.angle * math.Pi / 180
	x := (math.Abs(math.Cos(a))*b.width + math.Abs(math.Sin(a))*b.height) / 2
	y := (math.Abs(math.Sin(a))*b.width + math.Abs(math.Cos(a))*b.height) / 2
	return routeBox{left: b.center.X - x, right: b.center.X + x, top: b.center.Y - y, bottom: b.center.Y + y}
}

func useCaseLabelAngle(a, b Point) float64 {
	angle := math.Atan2(b.Y-a.Y, b.X-a.X) * 180 / math.Pi
	for angle > 90 {
		angle -= 180
	}
	for angle < -90 {
		angle += 180
	}
	return math.Max(-85, math.Min(85, angle))
}

func placeUseCaseLabels(s *Scene) {
	occupied := []routeBox{}
	for _, n := range s.Nodes {
		if n.Shape != "boundary" {
			occupied = append(occupied, routeBox{left: n.X - 4, right: n.X + n.Width + 4, top: n.Y - 4, bottom: n.Y + n.Height + 4})
		}
	}
	for i := range s.Edges {
		e := &s.Edges[i]
		if e.Label == "" || len(e.Points) < 2 {
			continue
		}
		a, b := e.Points[0], e.Points[len(e.Points)-1]
		e.LabelAngle = useCaseLabelAngle(a, b)
		angle := e.LabelAngle * math.Pi / 180
		length := math.Hypot(b.X-a.X, b.Y-a.Y)
		if length < .001 {
			continue
		}
		nx, ny := -(b.Y-a.Y)/length, (b.X-a.X)/length
		if ny > 0 || math.Abs(ny) < .001 && nx < 0 {
			nx, ny = -nx, -ny
		}
		width := labelWidth(e.Label)
		// Account for the small angle difference when a near-vertical line is
		// clamped to 85 degrees. The label remains beside its own connector.
		distance := math.Abs(nx*math.Cos(angle)+ny*math.Sin(angle))*width/2 + math.Abs(-nx*math.Sin(angle)+ny*math.Cos(angle))*11 + 6
		bestCost := math.Inf(1)
		var best rotatedLabel
		for k, t := range []float64{.5, .4, .6, .3, .7, .2, .8} {
			for _, side := range []float64{1, -1} {
				p := Point{a.X + (b.X-a.X)*t + nx*distance*side, a.Y + (b.Y-a.Y)*t + ny*distance*side}
				box := rotatedLabel{p, width + 6, 24, e.LabelAngle}
				bounds := box.bounds()
				cost := float64(k) + (.5-side*.5)*.1
				if bounds.left < 0 || bounds.top < 0 {
					cost += 10000
				}
				for _, o := range occupied {
					if bounds.left < o.right && bounds.right > o.left && bounds.top < o.bottom && bounds.bottom > o.top {
						cost += 10000
					}
				}
				for _, other := range s.Edges {
					for j := 1; j < len(other.Points); j++ {
						if box.crosses(other.Points[j-1], other.Points[j]) {
							cost += 1000
						}
					}
				}
				if cost < bestCost {
					bestCost, best = cost, box
				}
			}
		}
		e.LabelPosition = &best.center
		occupied = append(occupied, best.bounds())
		s.Width = math.Max(s.Width, best.bounds().right+40)
		s.Height = math.Max(s.Height, best.bounds().bottom+40)
	}
}
