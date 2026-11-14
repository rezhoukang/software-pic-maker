package main

import (
	"ezruantu/internal/exportbundle"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) ExportBundle(name string, files []exportbundle.File, folder, zipped bool) ([]string, error) {
	if !folder && !zipped {
		return nil, fmt.Errorf("请至少选择一种输出方式")
	}
	destination, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "选择导出保存位置"})
	if err != nil || destination == "" {
		return nil, err
	}
	return exportbundle.Write(destination, "导出-"+safeName(name), files, folder, zipped)
}
