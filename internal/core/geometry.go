package core

import "math"

// Closest point on the painted ellipse: its outward normal points toward the
// incoming segment, keeping both arrow wings outside the outline.
func ellipseNormalBoundary(n Node, towards Point) Point {
	c := center(n)
	rx, ry := n.Width/2-2, n.Height/2-2
	dx, dy := towards.X-c.X, towards.Y-c.Y
	if rx <= 0 || ry <= 0 || dx*dx/(rx*rx)+dy*dy/(ry*ry) <= 1 {
		return shapeBoundary(n, towards)
	}
	lo, hi := 0.0, math.Hypot(rx*dx, ry*dy)
	for i := 0; i < 48; i++ {
		lambda := (lo + hi) / 2
		x, y := rx*dx/(lambda+rx*rx), ry*dy/(lambda+ry*ry)
		if x*x+y*y > 1 {
			lo = lambda
		} else {
			hi = lambda
		}
	}
	return Point{c.X + rx*rx*dx/(hi+rx*rx), c.Y + ry*ry*dy/(hi+ry*ry)}
}

// All attachment points refer to the painted SVG outline, not its image box.
func shapePort(n Node, side string, ratio float64) Point {
	c := center(n)
	if n.Shape == "junction" {
		return c
	}
	if n.Shape == "actor" {
		c = connectionCenter(n)
		if side == "left" {
			return Point{c.X - 28, c.Y}
		}
		return Point{c.X + 28, c.Y}
	}
	if n.Shape == "ellipse" || n.Shape == "attribute" || n.Shape == "keyAttribute" || n.Shape == "junction" {
		rx, ry := n.Width/2-2, n.Height/2-2
		if n.Shape == "junction" {
			rx, ry = 3, 3
		}
		t := (ratio - .5) * 1.6
		switch side {
		case "left":
			return Point{c.X - rx*math.Sqrt(1-t*t), c.Y + ry*t}
		case "right":
			return Point{c.X + rx*math.Sqrt(1-t*t), c.Y + ry*t}
		case "top":
			return Point{c.X + rx*t, c.Y - ry*math.Sqrt(1-t*t)}
		case "bottom":
			return Point{c.X + rx*t, c.Y + ry*math.Sqrt(1-t*t)}
		}
	}
	inset := 1.0
	if n.Shape == "diamond" || n.Shape == "database" {
		inset = 2
	}
	left, right, top, bottom := n.X+inset, n.X+n.Width-inset, n.Y+inset, n.Y+n.Height-inset
	if n.Shape == "activation" {
		top, bottom = n.Y, n.Y+n.Height
	}
	cornerX, cornerY := 6.0, 6.0
	if n.Shape == "entity" || n.Shape == "activation" {
		cornerX, cornerY = 0, 0
	}
	if n.Shape == "terminal" {
		cornerX = math.Min(n.Height/2, (n.Width-2)/2)
		cornerY = math.Min(n.Height/2, (n.Height-2)/2)
	}
	switch side {
	case "top", "bottom":
		x := left + cornerX + ratio*(right-left-2*cornerX)
		y := top
		if side == "bottom" {
			y = bottom
		}
		if n.Shape == "diamond" {
			x = left + ratio*(right-left)
			delta := math.Abs(x-c.X) / (n.Width/2 - 2) * (n.Height/2 - 2)
			if side == "top" {
				y += delta
			} else {
				y -= delta
			}
		}
		if n.Shape == "database" {
			x = left + ratio*(right-left)
			top, bottom = databaseLimits(n, x)
			y = top
			if side == "bottom" {
				y = bottom
			}
		}
		return Point{x, y}
	case "left", "right":
		y := top + cornerY + ratio*(bottom-top-2*cornerY)
		x := left
		if side == "right" {
			x = right
		}
		if n.Shape == "terminal" {
			y = top + ratio*(bottom-top)
			t := (y - c.Y) / cornerY
			delta := cornerX * (1 - math.Sqrt(math.Max(0, 1-t*t)))
			if side == "left" {
				x += delta
			} else {
				x -= delta
			}
		}
		if n.Shape == "diamond" {
			y = top + ratio*(bottom-top)
			delta := math.Abs(y-c.Y) / (n.Height/2 - 2) * (n.Width/2 - 2)
			if side == "left" {
				x += delta
			} else {
				x -= delta
			}
		}
		if n.Shape == "database" {
			y = n.Y + 16 + ratio*(n.Height-32)
		}
		return Point{x, y}
	}
	return c
}

func databaseLimits(n Node, x float64) (float64, float64) {
	fraction := math.Max(0, math.Min(1, (x-n.X-2)/(n.Width-4)))
	lo, hi := 0.0, 1.0
	for i := 0; i < 36; i++ {
		t := (lo + hi) / 2
		if 3*t*t-2*t*t*t < fraction {
			lo = t
		} else {
			hi = t
		}
	}
	t := (lo + hi) / 2
	curve := 54 * t * (1 - t)
	return n.Y + 16 - curve, n.Y + n.Height - 16 + curve
}

func shapeBoundary(n Node, towards Point) Point {
	c := center(n)
	if n.Shape == "junction" {
		return c
	}
	if n.Width <= 0 || n.Height <= 0 {
		return c
	}
	if n.Shape == "actor" {
		c = connectionCenter(n)
		return Point{c.X + math.Copysign(28, towards.X-c.X), c.Y}
	}
	dx, dy := towards.X-c.X, towards.Y-c.Y
	if dx == 0 && dy == 0 {
		return c
	}
	if n.Shape == "ellipse" || n.Shape == "attribute" || n.Shape == "keyAttribute" || n.Shape == "junction" {
		rx, ry := n.Width/2-2, n.Height/2-2
		if n.Shape == "junction" {
			rx, ry = 3, 3
		}
		scale := 1 / math.Sqrt(dx*dx/(rx*rx)+dy*dy/(ry*ry))
		return Point{c.X + dx*scale, c.Y + dy*scale}
	}
	if n.Shape == "diamond" {
		scale := 1 / (math.Abs(dx)/(n.Width/2-2) + math.Abs(dy)/(n.Height/2-2))
		return Point{c.X + dx*scale, c.Y + dy*scale}
	}
	rx, ry := (n.Width-2)/2, (n.Height-2)/2
	cornerX, cornerY := math.Min(6, rx), math.Min(6, ry)
	if n.Shape == "entity" || n.Shape == "activation" {
		cornerX, cornerY = 0, 0
	}
	if n.Shape == "activation" {
		ry = n.Height / 2
	}
	if n.Shape == "terminal" {
		cornerX = math.Min(n.Height/2, rx)
		cornerY = math.Min(n.Height/2, ry)
	}
	if n.Shape == "database" {
		rx = n.Width/2 - 2
		ry = n.Height/2 - 2.5
		cornerX, cornerY = 0, 0
	}
	inside := func(t float64) bool {
		x, y := math.Abs(dx*t), math.Abs(dy*t)
		if x > rx || y > ry {
			return false
		}
		if n.Shape == "database" {
			top, bottom := databaseLimits(n, c.X+dx*t)
			py := c.Y + dy*t
			return py >= top && py <= bottom
		}
		if cornerX == 0 || cornerY == 0 || x <= rx-cornerX || y <= ry-cornerY {
			return true
		}
		x = (x - (rx - cornerX)) / cornerX
		y = (y - (ry - cornerY)) / cornerY
		return x*x+y*y <= 1
	}
	lo, hi := 0.0, math.Min(rx/math.Max(math.Abs(dx), .00001), ry/math.Max(math.Abs(dy), .00001))
	for i := 0; i < 36; i++ {
		mid := (lo + hi) / 2
		if inside(mid) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return Point{c.X + dx*lo, c.Y + dy*lo}
}

func oppositeSide(side string) string {
	switch side {
	case "left":
		return "right"
	case "right":
		return "left"
	case "top":
		return "bottom"
	default:
		return "top"
	}
}
func portWithStub(n Node, side string, ratio float64) routePort {
	p := shapePort(n, side, ratio)
	stub := p
	axis := 0
	switch side {
	case "left":
		stub.X = n.X - 36
	case "right":
		stub.X = n.X + n.Width + 36
	case "top":
		stub.Y = n.Y - 36
		axis = 1
	case "bottom":
		stub.Y = n.Y + n.Height + 36
		axis = 1
	}
	return routePort{p, stub, axis}
}
