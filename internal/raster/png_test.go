package raster

import (
	"bytes"
	"context"
	"image/png"
	"testing"
)

func TestPNGIncludesTheWholeDiagram(t *testing.T) {
	if Browser() == "" {
		t.Skip("no local browser installed")
	}
	data, err := PNG(context.Background(), `<svg xmlns="http://www.w3.org/2000/svg" width="320" height="180" viewBox="0 0 320 180"><rect width="320" height="180" fill="white"/><text x="15" y="40" font-size="26">中文图表测试</text><rect x="300" y="160" width="20" height="20" fill="red"/></svg>`, 320, 180)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil || img.Bounds().Dx() != 640 || img.Bounds().Dy() != 360 {
		t.Fatal("unexpected bitmap", err)
	}
	r, g, b, _ := img.At(630, 350).RGBA()
	if r < 60000 || g > 1000 || b > 1000 {
		t.Fatal("lower diagram content was cropped", r, g, b)
	}
	ink := 0
	for y := 20; y < 90; y++ {
		for x := 20; x < 400; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r+g+b < 30000 {
				ink++
			}
		}
	}
	if ink < 100 {
		t.Fatal("text was not rendered")
	}
}
