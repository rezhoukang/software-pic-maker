package core

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func esc(s string) string  { var b bytes.Buffer; _ = xml.EscapeText(&b, []byte(s)); return b.String() }
func num(n float64) string { return strconv.FormatFloat(n, 'f', 2, 64) }

// NodeMarkup is also used by X6 via a data SVG, preserving vector export fidelity.
func NodeMarkup(n Node) string {
	theme := "blue"
	if n.Stroke == "#333333" {
		theme = "mono"
	}
	if n.Stroke == "#5f7a71" {
		theme = "sage"
	}
	return colorize(nodeMarkup(n), theme)
}
func colorize(s, theme string) string {
	if theme == "mono" {
		return strings.NewReplacer("#475569", "#333333", "#52627c", "#333333", "#263650", "#222222", "#475770", "#333333").Replace(s)
	}
	if theme == "sage" {
		return strings.NewReplacer("#475569", "#5f7a71", "#52627c", "#49665c").Replace(s)
	}
	return s
}
func nodeMarkup(n Node) string {
	var b strings.Builder
	w, h := n.Width, n.Height
	stroke := n.Stroke
	fill := n.Fill
	if n.Shape == "boundary" {
		fmt.Fprintf(&b, `<rect x="1" y="1" width="%s" height="%s" fill="white" stroke="%s" stroke-width="1.5" rx="4"/>`, num(w-2), num(h-2), stroke)
		textSVG(&b, w/2, 26, n.Title, 16, "#52627c", "middle", false)
		return b.String()
	}
	switch n.Shape {
	case "junction":
		// An internal merge anchor has no visible flowchart symbol.
		return b.String()
	case "ellipse", "attribute", "keyAttribute":
		fmt.Fprintf(&b, `<ellipse cx="%s" cy="%s" rx="%s" ry="%s" fill="white" stroke="%s" stroke-width="1.6"/>`, num(w/2), num(h/2), num(w/2-2), num(h/2-2), stroke)
	case "diamond":
		fmt.Fprintf(&b, `<path d="M %s 2 L %s %s L %s %s L 2 %s Z" fill="%s" stroke="%s" stroke-width="1.6"/>`, num(w/2), num(w-2), num(h/2), num(w/2), num(h-2), num(h/2), fill, stroke)
	case "actor":
		fmt.Fprintf(&b, `<g fill="none" stroke="%s" stroke-width="2" stroke-linecap="round"><circle cx="%s" cy="18" r="13"/><path d="M %s 31 V 74 M %s 48 H %s M %s 74 L %s 102 M %s 74 L %s 102"/></g>`, stroke, num(w/2), num(w/2), num(w/2-28), num(w/2+28), num(w/2), num(w/2-25), num(w/2), num(w/2+25))
		textSVG(&b, w/2, 122, n.Title, 16, "#263650", "middle", false)
		return b.String()
	case "activation":
		fmt.Fprintf(&b, `<rect x="1" y="0" width="%s" height="%s" fill="white" stroke="%s" stroke-width="1.6"/>`, num(w-2), num(h), stroke)
		return b.String()
	case "database":
		fmt.Fprintf(&b, `<path d="M 2 16 C 2 -2 %s -2 %s 16 V %s C %s %s 2 %s 2 %s Z M 2 16 C 2 34 %s 34 %s 16" fill="%s" stroke="%s" stroke-width="1.6"/>`, num(w-2), num(w-2), num(h-16), num(w-2), num(h+2), num(h+2), num(h-16), num(w-2), num(w-2), fill, stroke)
	default:
		radius := 6.0
		if n.Shape == "terminal" {
			radius = h / 2
		}
		if n.Shape == "entity" {
			radius = 0
		}
		fmt.Fprintf(&b, `<rect x="1" y="1" width="%s" height="%s" rx="%s" fill="%s" stroke="%s" stroke-width="1.6"/>`, num(w-2), num(h-2), num(radius), fill, stroke)
	}
	if n.Shape == "table" {
		fmt.Fprintf(&b, `<path d="M 1 44 H %s" stroke="%s" stroke-width="1.2"/>`, num(w-1), stroke)
		textSVG(&b, w/2, 26, n.Title, 16, "#263650", "middle", true)
		y := 67.0
		for _, line := range n.Lines {
			textSVG(&b, 16, y, line, 14, "#475770", "start", false)
			y += 32
		}
		if len(n.Methods) > 0 {
			fmt.Fprintf(&b, `<path d="M 1 %s H %s" stroke="%s"/>`, num(y-10), num(w-1), stroke)
			y += 15
			for _, line := range n.Methods {
				textSVG(&b, 16, y, line, 14, "#475770", "start", false)
				y += 32
			}
		}
	} else {
		if n.Shape == "rect" || n.Shape == "database" {
			lines := wrapNodeTitle(n.Title, w-36)
			y := nodeTitleCenter(n) - float64(len(lines)-1)*nodeTitleLineHeight/2
			b.WriteString(`<g class="node-title">`)
			for _, line := range lines {
				textSVG(&b, w/2, y, line, 16, "#263650", "middle", false)
				y += nodeTitleLineHeight
			}
			b.WriteString(`</g>`)
			return b.String()
		}
		if n.Shape == "keyAttribute" {
			b.WriteString(`<g text-decoration="underline">`)
		}
		textSVG(&b, w/2, h/2+1, n.Title, 16, "#263650", "middle", n.Shape == "entity")
		if n.Shape == "keyAttribute" {
			b.WriteString(`</g>`)
		}
	}
	return b.String()
}
func textSVG(b *strings.Builder, x, y float64, text string, size int, color, anchor string, bold bool) {
	weight := "400"
	if bold {
		weight = "600"
	}
	fmt.Fprintf(b, `<text x="%s" y="%s" fill="%s" font-size="%d" font-weight="%s" text-anchor="%s" dominant-baseline="central">%s</text>`, num(x), num(y), color, size, weight, anchor, esc(text))
}
func SVG(s Scene) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%s" height="%s" viewBox="0 0 %s %s"><title>%s</title><style>text{font-family:'Microsoft YaHei','Noto Sans CJK SC',Arial,sans-serif}</style><defs><marker id="open" markerWidth="12" markerHeight="12" refX="10" refY="6" orient="auto" markerUnits="userSpaceOnUse"><path d="M 1 1 L 10 6 L 1 11" fill="none" stroke="#475569" stroke-width="1.6"/></marker><marker id="filled" markerWidth="12" markerHeight="12" refX="10" refY="6" orient="auto" markerUnits="userSpaceOnUse"><path d="M 1 1 L 10 6 L 1 11 Z" fill="#475569" stroke="#475569"/></marker><marker id="triangle" markerWidth="16" markerHeight="14" refX="14" refY="7" orient="auto" markerUnits="userSpaceOnUse"><path d="M 1 1 L 14 7 L 1 13 Z" fill="white" stroke="#475569" stroke-width="1.5"/></marker></defs><rect width="100%%" height="100%%" fill="white"/>`, num(s.Width), num(s.Height), num(s.Width), num(s.Height), esc(s.Title))
	for _, n := range s.Nodes {
		if n.Shape == "boundary" {
			fmt.Fprintf(&b, `<g transform="translate(%s,%s)">%s</g>`, num(n.X), num(n.Y), NodeMarkup(n))
		}
	}
	for _, e := range s.Edges {
		if len(e.Points) < 2 {
			continue
		}
		pts := []string{}
		for _, p := range e.Points {
			pts = append(pts, num(p.X)+","+num(p.Y))
		}
		dash := ""
		if e.Dashed {
			dash = ` stroke-dasharray="7 5"`
		}
		marker := ""
		if e.Arrow == "open" || e.Arrow == "triangle" || e.Arrow == "filled" {
			marker = ` marker-end="url(#` + e.Arrow + `)"`
		}
		fmt.Fprintf(&b, `<polyline points="%s" fill="none" stroke="#475569" stroke-width="1.6" stroke-linejoin="round"%s%s/>`, strings.Join(pts, " "), dash, marker)
		labelSVG(&b, e)
	}
	for _, n := range s.Nodes {
		if n.Shape != "boundary" {
			fmt.Fprintf(&b, `<g transform="translate(%s,%s)">%s</g>`, num(n.X), num(n.Y), NodeMarkup(n))
		}
	}
	b.WriteString(`</svg>`)
	return colorize(b.String(), s.Theme)
}
func labelSVG(b *strings.Builder, e Edge) {
	pts := e.Points
	mid := pointAt(pts, 0.5)
	if e.Label != "" {
		label := Point{mid.X, mid.Y - 13}
		if e.LabelPosition != nil {
			label = *e.LabelPosition
		}
		fmt.Fprintf(b, `<g style="paint-order:stroke;stroke:white;stroke-width:7;stroke-linejoin:round">`)
		textSVG(b, label.X, label.Y, e.Label, 14, "#52627c", "middle", false)
		b.WriteString(`</g>`)
	}
	for _, v := range []struct {
		label    string
		ratio    float64
		position *Point
	}{{e.SourceLabel, .12, e.SourceLabelPosition}, {e.TargetLabel, .88, e.TargetLabelPosition}} {
		if v.label != "" {
			p := pointAt(pts, v.ratio)
			y := p.Y - 15
			if e.Label != "" {
				y = p.Y + 18
			}
			if v.position != nil {
				p = *v.position
				y = p.Y
			}
			textSVG(b, p.X, y, v.label, 14, "#52627c", "middle", false)
		}
	}
}
func pointAt(pts []Point, ratio float64) Point {
	length := 0.0
	for i := 1; i < len(pts); i++ {
		length += math.Hypot(pts[i].X-pts[i-1].X, pts[i].Y-pts[i-1].Y)
	}
	remaining := length * ratio
	for i := 1; i < len(pts); i++ {
		seg := math.Hypot(pts[i].X-pts[i-1].X, pts[i].Y-pts[i-1].Y)
		if seg > 0 && remaining <= seg {
			t := remaining / seg
			return Point{pts[i-1].X + (pts[i].X-pts[i-1].X)*t, pts[i-1].Y + (pts[i].Y-pts[i-1].Y)*t}
		}
		remaining -= seg
	}
	return pts[len(pts)-1]
}

func DrawIO(s Scene) string {
	var b strings.Builder
	nodes := map[string]Node{}
	for _, n := range s.Nodes {
		nodes[n.ID] = n
	}
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><mxfile host="EZ软图" version="1.0"><diagram id="diagram-1" name="` + esc(s.Title) + `"><mxGraphModel grid="1" gridSize="10" page="1" pageWidth="` + num(s.Width) + `" pageHeight="` + num(s.Height) + `"><root><mxCell id="0"/><mxCell id="1" parent="0"/>`)
	for _, n := range s.Nodes {
		style := "rounded=1;whiteSpace=wrap;html=0;align=center;verticalAlign=middle;fontFamily=Microsoft YaHei;fontSize=16;strokeWidth=1.5;fillColor=" + n.Fill + ";strokeColor=" + n.Stroke + ";"
		switch n.Shape {
		case "junction":
			style += "shape=ellipse;fillColor=none;strokeColor=none;strokeWidth=0;opacity=0;"
		case "ellipse", "attribute", "keyAttribute":
			style += "shape=ellipse;fillColor=#ffffff;"
			if n.Shape == "keyAttribute" {
				style += "fontStyle=4;"
			}
		case "diamond":
			style += "shape=rhombus;"
		case "actor":
			style += "shape=umlActor;verticalLabelPosition=bottom;verticalAlign=top;fillColor=none;"
		case "table":
			style += "shape=swimlane;startSize=44;rounded=0;"
		case "boundary":
			style += "fillColor=none;rounded=0;verticalAlign=top;spacingTop=12;"
		case "database":
			style += "shape=cylinder3;size=15;spacingTop=14;"
		case "entity":
			style += "rounded=0;"
		case "activation":
			style += "rounded=0;fillColor=#ffffff;"
		case "terminal":
			style += "arcSize=50;"
		}
		h := n.Height
		if n.Shape == "actor" {
			// draw.io places its label outside the figure's own geometry.
			h = 103
		}
		title := n.Title
		if n.Shape == "rect" || n.Shape == "database" {
			title = strings.Join(wrapNodeTitle(n.Title, n.Width-36), "\n")
			style += "spacingLeft=18;spacingRight=18;"
		}
		fmt.Fprintf(&b, `<mxCell id="%s" value="%s" style="%s" vertex="1" parent="1"><mxGeometry x="%s" y="%s" width="%s" height="%s" as="geometry"/></mxCell>`, esc(n.ID), esc(title), esc(style), num(n.X), num(n.Y), num(n.Width), num(h))
		if n.Shape == "table" {
			y := 50.0
			lines := append(append([]string{}, n.Lines...), n.Methods...)
			for i, line := range lines {
				if i == len(n.Lines) && len(n.Methods) > 0 {
					y += 20
				}
				fmt.Fprintf(&b, `<mxCell id="$field:%s:%d" value="%s" style="text;html=0;align=left;verticalAlign=middle;whiteSpace=wrap;fontFamily=Microsoft YaHei;fontSize=14;spacingLeft=14;" vertex="1" parent="%s"><mxGeometry x="0" y="%s" width="%s" height="32" as="geometry"/></mxCell>`, esc(n.ID), i, esc(line), esc(n.ID), num(y), num(n.Width))
				y += 32
			}
		}
	}
	for _, e := range s.Edges {
		if len(e.Points) < 2 {
			continue
		}
		style := "edgeStyle=none;html=0;fontFamily=Microsoft YaHei;fontSize=14;strokeColor=#475569;strokeWidth=1.5;labelBackgroundColor=#ffffff;endArrow=none;"
		if e.Dashed {
			style += "dashed=1;dashPattern=7 5;"
		}
		if e.Arrow == "open" {
			style += "endArrow=open;endFill=0;"
		}
		if e.Arrow == "triangle" {
			style += "endArrow=block;endFill=0;"
		}
		if e.Arrow == "filled" {
			style += "endArrow=block;endFill=1;"
		}
		// Preserve the same shape-specific endpoints used by SVG and the editor.
		perimeter := "0"
		if s.Type == "flowchart" {
			perimeter = "1"
		}
		if n, ok := nodes[e.Source]; ok {
			p := e.Points[0]
			style += "exitX=" + num((p.X-n.X)/n.Width) + ";exitY=" + num((p.Y-n.Y)/n.Height) + ";exitPerimeter=" + perimeter + ";"
		}
		if n, ok := nodes[e.Target]; ok {
			p := e.Points[len(e.Points)-1]
			style += "entryX=" + num((p.X-n.X)/n.Width) + ";entryY=" + num((p.Y-n.Y)/n.Height) + ";entryPerimeter=" + perimeter + ";"
		}
		connections := ""
		if e.Source != "" {
			connections += ` source="` + esc(e.Source) + `"`
		}
		if e.Target != "" {
			connections += ` target="` + esc(e.Target) + `"`
		}
		fmt.Fprintf(&b, `<mxCell id="%s" value="%s" style="%s" edge="1" parent="1"%s><mxGeometry relative="1" as="geometry">`, esc(e.ID), esc(e.Label), esc(style), connections)
		if e.LabelPosition != nil {
			mid := pointAt(e.Points, .5)
			fmt.Fprintf(&b, `<mxPoint x="%s" y="%s" as="offset"/>`, num(e.LabelPosition.X-mid.X), num(e.LabelPosition.Y-mid.Y))
		}
		a, z := e.Points[0], e.Points[len(e.Points)-1]
		fmt.Fprintf(&b, `<mxPoint x="%s" y="%s" as="sourcePoint"/><mxPoint x="%s" y="%s" as="targetPoint"/>`, num(a.X), num(a.Y), num(z.X), num(z.Y))
		if len(e.Points) > 2 {
			b.WriteString(`<Array as="points">`)
			for _, p := range e.Points[1 : len(e.Points)-1] {
				fmt.Fprintf(&b, `<mxPoint x="%s" y="%s"/>`, num(p.X), num(p.Y))
			}
			b.WriteString(`</Array>`)
		}
		b.WriteString(`</mxGeometry></mxCell>`)
		for i, v := range []struct {
			text     string
			ratio    float64
			position *Point
		}{{e.SourceLabel, .12, e.SourceLabelPosition}, {e.TargetLabel, .88, e.TargetLabelPosition}} {
			if v.text != "" {
				p := pointAt(e.Points, v.ratio)
				if e.Label != "" {
					p.Y += 40
				}
				fmt.Fprintf(&b, `<mxCell id="$card:%s:%d" value="%s" style="text;html=0;align=center;fontSize=14;" vertex="1" parent="1"><mxGeometry x="%s" y="%s" width="60" height="24" as="geometry"/></mxCell>`, esc(e.ID), i, esc(v.text), num(cardPosition(v.position, p).X-30), num(cardPosition(v.position, p).Y-12))
			}
		}
	}
	b.WriteString(`</root></mxGraphModel></diagram></mxfile>`)
	return colorize(b.String(), s.Theme)
}

func cardPosition(explicit *Point, fallback Point) Point {
	if explicit != nil {
		return *explicit
	}
	fallback.Y -= 15
	return fallback
}
