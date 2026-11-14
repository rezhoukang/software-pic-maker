package main

import "ezruantu/internal/settings"

func (a *App) GetSettings() (settings.Preferences, error) {
	return settings.Load(a.dataDir)
}

func (a *App) SetGlobalTheme(theme string) error {
	return settings.SetTheme(a.dataDir, theme)
}
