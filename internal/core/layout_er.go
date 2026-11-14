package core

import (
	"fmt"
	"math"
)

type erAttribute struct{ offset, port Point }
type erCluster struct {
	attributes               []erAttribute
	left, right, top, bottom float64
}

// Attribute bodies stay outside a rectangular frame. Their inward-facing ports
// lie on the frame, keeping every spoke inside it until it reaches its own node.
func erAttributes(fields []Field, entityWidth float64) erCluster {
	count := len(fields)
	best := erCluster{left: entityWidth/2 + 18, right: entityWidth/2 + 18, top: 48, bottom: 48}
	if count == 0 {
		return best
	}
	width := 130.0
	for _, f := range fields {
		width = math.Max(width, textWidth(f.Name, 130, 250))
	}
	bestScore := math.Inf(1)
	entity := Node{X: -entityWidth / 2, Y: -30, Width: entityWidth, Height: 60}
	for horizontal := 0; horizontal <= count; horizontal++ {
		if count < 4 && horizontal != count {
			continue
		}
		if count >= 4 && (horizontal < 2 || horizontal > count-2) {
			continue
		}
		top, bottom := (horizontal+1)/2, horizontal/2
		left, right := (count-horizontal+1)/2, (count-horizontal)/2
		maxX := float64(max(top, bottom)-1) * (width + 20) / 2
		maxY := 0.0
		for i := 0; i < max(left, right); i++ {
			maxY = math.Max(maxY, 74+float64(i/2)*60)
		}
		innerX := math.Max(entityWidth/2+45, maxX+width/2+20)
		innerY := math.Max(75, maxY+23+20)
		candidate := erCluster{left: entityWidth / 2, right: entityWidth / 2, top: 30, bottom: 30}
		appendPort := func(port Point, side string) {
			index := len(candidate.attributes)
			w := textWidth(fields[index].Name, 130, 250)
			offset := port
			switch side {
			case "top":
				offset.Y -= 23
			case "bottom":
				offset.Y += 23
			case "left":
				offset.X -= w / 2
			case "right":
				offset.X += w / 2
			}
			candidate.attributes = append(candidate.attributes, erAttribute{offset: offset, port: port})
			candidate.left = math.Max(candidate.left, -offset.X+w/2)
			candidate.right = math.Max(candidate.right, offset.X+w/2)
			candidate.top = math.Max(candidate.top, -offset.Y+23)
			candidate.bottom = math.Max(candidate.bottom, offset.Y+23)
		}
		for i := 0; i < top; i++ {
			appendPort(Point{(float64(i) - float64(top-1)/2) * (width + 20), -innerY}, "top")
		}
		for i := 0; i < bottom; i++ {
			appendPort(Point{(float64(i) - float64(bottom-1)/2) * (width + 20), innerY}, "bottom")
		}
		for i := 0; i < left; i++ {
			y := 74 + float64(i/2)*60
			if i%2 == 0 {
				y = -y
			}
			appendPort(Point{-innerX, y}, "left")
		}
		for i := 0; i < right; i++ {
			y := 74 + float64(i/2)*60
			if i%2 == 1 {
				y = -y
			}
			appendPort(Point{innerX, y}, "right")
		}
		longest, total := 0.0, 0.0
		for _, a := range candidate.attributes {
			start := straight(entity, Node{X: a.port.X, Y: a.port.Y})[0]
			length := math.Hypot(a.port.X-start.X, a.port.Y-start.Y)
			longest = math.Max(longest, length)
			total += length
		}
		// Prefer shorter spokes, using the occupied size to break close ties.
		score := longest + .25*total/float64(count) + .03*(candidate.left+candidate.right+candidate.top+candidate.bottom)
		if score < bestScore {
			bestScore = score
			best = candidate
		}
	}
	best.left += 18
	best.right += 18
	best.top += 18
	best.bottom += 18
	return best
}

func layoutER(d Diagram, s *Scene, add func(Node), find func(string) Node) {
	clusters := map[string]erCluster{}
	for _, e := range d.Model.Entities {
		clusters[e.ID] = erAttributes(e.Fields, textWidth(e.Name, 160, 260))
	}
	columns := min(3, len(d.Model.Entities))
	rowY, rowX, rowTop, rowBottom := 48.0, 48.0, 0.0, 0.0
	rights := map[string]float64{}
	for i, e := range d.Model.Entities {
		if i%columns == 0 {
			if i > 0 {
				rowY += rowTop + rowBottom + 100
			}
			rowX, rowTop, rowBottom = 48, 0, 0
			for j := i; j < min(i+columns, len(d.Model.Entities)); j++ {
				c := clusters[d.Model.Entities[j].ID]
				rowTop = math.Max(rowTop, c.top)
				rowBottom = math.Max(rowBottom, c.bottom)
			}
		}
		c := clusters[e.ID]
		cx, cy := rowX+c.left, rowY+rowTop
		rowX += c.left + c.right + 120
		w := textWidth(e.Name, 160, 260)
		add(Node{ID: e.ID, X: cx - w/2, Y: cy - 30, Width: w, Height: 60, Shape: "entity", Title: e.Name, Movable: true})
		n := find(e.ID)
		nc := center(n)
		rights[e.ID] = nc.X + c.right
		for j, f := range e.Fields {
			p := c.attributes[j]
			w := textWidth(f.Name, 130, 250)
			shape := "attribute"
			if f.Primary {
				shape = "keyAttribute"
			}
			id := fmt.Sprintf("$attr:%s:%d", e.ID, j)
			add(Node{ID: id, X: nc.X + p.offset.X - w/2, Y: nc.Y + p.offset.Y - 23, Width: w, Height: 46, Shape: shape, Title: f.Name, Movable: true})
			a := find(id)
			actual := center(a)
			port := Point{nc.X + p.port.X + actual.X - (nc.X + p.offset.X), nc.Y + p.port.Y + actual.Y - (nc.Y + p.offset.Y)}
			port = shapeBoundary(a, port)
			points := []Point{port, straight(n, Node{X: port.X, Y: port.Y})[0]}
			if _, manual := d.View.Positions[id]; manual {
				points = straight(a, n)
			}
			s.Edges = append(s.Edges, Edge{ID: id + "-edge", Source: id, Target: e.ID, Points: points, Arrow: "none"})
		}
	}
	for i, r := range d.Model.Relations {
		a, b := find(r.Source), find(r.Target)
		ac, bc := center(a), center(b)
		title := r.Label
		if title == "" {
			title = "联系"
		}
		x, y := (ac.X+bc.X)/2-60, (ac.Y+bc.Y)/2-38
		if math.Abs(ac.Y-bc.Y) > 1 {
			x = math.Max(rights[a.ID], rights[b.ID]) + 35 + float64(i%3)*35
		}
		rn := Node{ID: "$relation:" + r.ID, X: x, Y: y, Width: 120, Height: 76, Shape: "diamond", Title: title, Movable: true}
		add(rn)
		rn = find(rn.ID)
		s.Edges = append(s.Edges,
			Edge{ID: "$edge:" + r.ID + ":a", Source: a.ID, Target: rn.ID, Points: straight(a, rn), SourceLabel: r.SourceCardinality, Arrow: "none"},
			Edge{ID: "$edge:" + r.ID + ":b", Source: rn.ID, Target: b.ID, Points: straight(rn, b), TargetLabel: r.TargetCardinality, Arrow: "none"})
	}
}
