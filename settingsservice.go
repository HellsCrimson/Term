package main

import (
	"term/database"
)

type SettingsService struct {
	db *database.DB
}

// NewSettingsService creates a new settings service
func NewSettingsService(db *database.DB) *SettingsService {
	return &SettingsService{db: db}
}

// GetSetting retrieves a single setting
func (s *SettingsService) GetSetting(key string) (*database.Setting, error) {
	return s.db.GetSetting(key)
}

// GetAllSettings retrieves all settings as a map
func (s *SettingsService) GetAllSettings() (map[string]string, error) {
	return s.db.GetAllSettings()
}

// SetSetting sets or updates a setting
func (s *SettingsService) SetSetting(key, value, valueType string) error {
	return s.db.SetSetting(key, value, valueType)
}

// GetTheme retrieves the current theme setting
func (s *SettingsService) GetTheme() (string, error) {
	setting, err := s.db.GetSetting("theme")
	if err != nil {
		return "dark", err // default to dark
	}
	return setting.Value, nil
}

// SetTheme updates the theme setting
func (s *SettingsService) SetTheme(theme string) error {
	return s.db.SetSetting("theme", theme, "string")
}

// GetFontFamily retrieves the font family setting
func (s *SettingsService) GetFontFamily() (string, error) {
	setting, err := s.db.GetSetting("font_family")
	if err != nil {
		return "monospace", err
	}
	return setting.Value, nil
}

// SetFontFamily updates the font family setting
func (s *SettingsService) SetFontFamily(fontFamily string) error {
	return s.db.SetSetting("font_family", fontFamily, "string")
}

// GetFontSize retrieves the font size setting
func (s *SettingsService) GetFontSize() (string, error) {
	setting, err := s.db.GetSetting("font_size")
	if err != nil {
		return "14", err
	}
	return setting.Value, nil
}

// SetFontSize updates the font size setting
func (s *SettingsService) SetFontSize(fontSize string) error {
	return s.db.SetSetting("font_size", fontSize, "int")
}

// GetAutoLaunch retrieves the auto-launch setting
func (s *SettingsService) GetAutoLaunch() (string, error) {
	setting, err := s.db.GetSetting("auto_launch")
	if err != nil {
		return "true", err
	}
	return setting.Value, nil
}

// SetAutoLaunch updates the auto-launch setting
func (s *SettingsService) SetAutoLaunch(autoLaunch string) error {
	return s.db.SetSetting("auto_launch", autoLaunch, "bool")
}

// SaveTabSnapshots saves the current tab snapshots
func (s *SettingsService) SaveTabSnapshots(snapshots string) error {
	return s.db.SetSetting("tab_snapshots", snapshots, "json")
}

// GetTabSnapshots retrieves the saved tab snapshots
func (s *SettingsService) GetTabSnapshots() (string, error) {
	setting, err := s.db.GetSetting("tab_snapshots")
	if err != nil {
		return "[]", nil // return empty array if not found
	}
	return setting.Value, nil
}
