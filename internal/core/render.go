package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type RenderResult struct {
	Validation Validation        `json:"validation"`
	Scene      Scene             `json:"scene"`
	SVG        string            `json:"svg"`
	NodeSVGs   map[string]string `json:"node_svgs"`
}

func Render(raw string) RenderResult {
	d, v := ValidateJSON([]byte(raw))
	r := RenderResult{Validation: v, NodeSVGs: map[string]string{}}
	if !v.Valid {
		return r
	}
	r.Scene = Generate(d)
	r.SVG = SVG(r.Scene)
	for _, n := range r.Scene.Nodes {
		r.NodeSVGs[n.ID] = fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%s" height="%s" viewBox="0 0 %s %s"><style>text{font-family:'Microsoft YaHei','Noto Sans CJK SC',Arial,sans-serif}</style>%s</svg>`, num(n.Width), num(n.Height), num(n.Width), num(n.Height), NodeMarkup(n))
	}
	return r
}
func Export(raw, format string) (string, error) {
	d, v := ValidateJSON([]byte(raw))
	if !v.Valid {
		return "", fmt.Errorf("%s", v.Issues[0].Message)
	}
	switch format {
	case "svg":
		return SVG(Generate(d)), nil
	case "drawio":
		return DrawIO(Generate(d)), nil
	case "json":
		b, _ := json.MarshalIndent(d, "", "  ")
		return string(b), nil
	default:
		return "", fmt.Errorf("不支持的导出格式：%s", format)
	}
}
func AtomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".tuxu-*")
	if err != nil {
		return err
	}
	temp := f.Name()
	defer os.Remove(temp)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(temp, path)
}
