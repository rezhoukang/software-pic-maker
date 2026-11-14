package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"ezruantu/internal/core"
	"ezruantu/internal/workspace"
)

type Preferences struct {
	Theme string `json:"theme"`
}

func validTheme(theme string) bool {
	return theme == "blue" || theme == "mono" || theme == "sage"
}

func Load(dataDir string) (Preferences, error) {
	p := Preferences{Theme: "blue"}
	b, err := os.ReadFile(filepath.Join(dataDir, workspace.Internal, "settings.json"))
	if os.IsNotExist(err) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return Preferences{Theme: "blue"}, fmt.Errorf("全局设置无法读取：%w", err)
	}
	if !validTheme(p.Theme) {
		return Preferences{Theme: "blue"}, fmt.Errorf("无效的全局风格：%s", p.Theme)
	}
	return p, nil
}

func SetTheme(dataDir, theme string) error {
	if !validTheme(theme) {
		return fmt.Errorf("风格必须是 blue、mono 或 sage")
	}
	b, err := json.Marshal(Preferences{Theme: theme})
	if err != nil {
		return err
	}
	return core.AtomicWrite(filepath.Join(dataDir, workspace.Internal, "settings.json"), b)
}
