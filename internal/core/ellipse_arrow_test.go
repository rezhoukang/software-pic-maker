package core

import (
	"math"
	"testing"
)

func TestUseCaseArrowWingsStayOutsideEllipse(t *testing.T) {
	s := Generate(Templates()[2])
	for _, e := range s.Edges {
		if e.Arrow == "none" {
			continue
		}
		n := sceneNode(s, e.Target)
		end, prior := e.Points[len(e.Points)-1], e.Points[len(e.Points)-2]
		c := center(n)
		rx, ry := n.Width/2-2, n.Height/2-2
		implicit := func(p Point) float64 { return math.Pow((p.X-c.X)/rx, 2) + math.Pow((p.Y-c.Y)/ry, 2) }
		if math.Abs(implicit(end)-1) > 1e-6 {
			t.Fatal("arrow tip misses ellipse", e.ID, end)
		}
		dx, dy := prior.X-end.X, prior.Y-end.Y
		length := math.Hypot(dx, dy)
		dx, dy = dx/length, dy/length
		nx, ny := (end.X-c.X)/(rx*rx), (end.Y-c.Y)/(ry*ry)
		cosine := (nx*dx + ny*dy) / math.Hypot(nx, ny)
		if cosine < .5 {
			t.Fatal("arrow should not graze the ellipse", e.ID, cosine)
		}
		for _, side := range []float64{-1, 1} {
			for step := 2.0; step <= 10; step++ {
				wing := Point{end.X + dx*step - side*dy*step/2, end.Y + dy*step + side*dx*step/2}
				if implicit(wing) <= 1 {
					t.Fatal("arrow wing overlaps the ellipse", e.ID, wing)
				}
			}
		}
	}
}
