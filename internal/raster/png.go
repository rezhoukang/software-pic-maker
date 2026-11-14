package raster

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Browser uses an installed browser and a private temporary profile. No model,
// desktop editor, remote service, or browser download is involved.
func Browser() string {
	for _, name := range []string{"msedge", "chrome", "chromium", "chromium-browser", "google-chrome"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	for _, root := range []string{os.Getenv("ProgramFiles(x86)"), os.Getenv("ProgramFiles"), os.Getenv("LOCALAPPDATA")} {
		if root == "" {
			continue
		}
		for _, name := range []string{"Microsoft/Edge/Application/msedge.exe", "Google/Chrome/Application/chrome.exe"} {
			p := filepath.Join(root, filepath.FromSlash(name))
			if info, err := os.Stat(p); err == nil && !info.IsDir() {
				return p
			}
		}
	}
	return ""
}

func PNG(ctx context.Context, svg string, width, height float64) ([]byte, error) {
	browser := Browser()
	if browser == "" {
		return nil, fmt.Errorf("PNG 导出需要本机安装 Microsoft Edge 或 Chrome；也可改用 SVG")
	}
	if width <= 0 || height <= 0 || math.IsNaN(width+height) || math.IsInf(width+height, 0) {
		return nil, fmt.Errorf("无效图像尺寸")
	}
	scale := math.Min(2, math.Min(8000/math.Max(width, height), math.Sqrt(24000000/(width*height))))
	w, h := int(math.Ceil(width*scale)), int(math.Ceil(height*scale))
	dir, err := os.MkdirTemp("", "ezruantu-png-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	html := fmt.Sprintf(`<!doctype html><meta charset="utf-8"><style>html,body{margin:0;padding:0;background:white;overflow:hidden}svg{display:block;width:%dpx!important;height:%dpx!important}</style>%s`, w, h, svg)
	input, output := filepath.Join(dir, "diagram.html"), filepath.Join(dir, "diagram.png")
	if err = os.WriteFile(input, []byte(html), 0600); err != nil {
		return nil, err
	}
	uri := (&url.URL{Scheme: "file", Path: "/" + strings.TrimPrefix(filepath.ToSlash(input), "/")}).String()
	ctx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, browser, "--headless=new", "--disable-gpu", "--no-first-run", "--no-default-browser-check", "--disable-extensions", "--disable-background-networking", "--hide-scrollbars", "--force-device-scale-factor=1", "--user-data-dir="+filepath.Join(dir, "profile"), fmt.Sprintf("--window-size=%d,%d", w, h), "--screenshot="+output, uri)
	hideWindow(cmd)
	log, runErr := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("PNG 渲染取消或超时：%w", ctx.Err())
	}
	if runErr != nil {
		if len(log) > 1000 {
			log = log[:1000]
		}
		return nil, fmt.Errorf("PNG 渲染失败：%v %s", runErr, log)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		return nil, fmt.Errorf("PNG 未生成：%w", err)
	}
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width != w || config.Height != h {
		return nil, fmt.Errorf("PNG 尺寸或内容无效：%dx%d，预期 %dx%d", config.Width, config.Height, w, h)
	}
	return data, nil
}
