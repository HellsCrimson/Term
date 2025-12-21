package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type ThemeColors struct {
	Bg struct {
		Primary    string `json:"primary"`
		Secondary  string `json:"secondary"`
		Tertiary   string `json:"tertiary"`
		Quaternary string `json:"quaternary"`
	} `json:"bg"`
	Text struct {
		Primary   string `json:"primary"`
		Secondary string `json:"secondary"`
		Muted     string `json:"muted"`
	} `json:"text"`
	Accent struct {
		Blue   string `json:"blue"`
		Green  string `json:"green"`
		Red    string `json:"red"`
		Yellow string `json:"yellow"`
		Purple string `json:"purple"`
		Pink   string `json:"pink"`
		Cyan   string `json:"cyan"`
		Orange string `json:"orange"`
	} `json:"accent"`
	Border    string `json:"border"`
	Hover     string `json:"hover"`
	Active    string `json:"active"`
	Selection string `json:"selection"`
}

type TerminalColors struct {
	Background          string `json:"background"`
	Foreground          string `json:"foreground"`
	Cursor              string `json:"cursor"`
	SelectionBackground string `json:"selectionBackground"`
	Black               string `json:"black"`
	Red                 string `json:"red"`
	Green               string `json:"green"`
	Yellow              string `json:"yellow"`
	Blue                string `json:"blue"`
	Magenta             string `json:"magenta"`
	Cyan                string `json:"cyan"`
	White               string `json:"white"`
	BrightBlack         string `json:"brightBlack"`
	BrightRed           string `json:"brightRed"`
	BrightGreen         string `json:"brightGreen"`
	BrightYellow        string `json:"brightYellow"`
	BrightBlue          string `json:"brightBlue"`
	BrightMagenta       string `json:"brightMagenta"`
	BrightCyan          string `json:"brightCyan"`
	BrightWhite         string `json:"brightWhite"`
}

type Theme struct {
	Name     string         `json:"name"`
	ID       string         `json:"id"`
	Type     string         `json:"type"` // "dark" or "light"
	Colors   ThemeColors    `json:"colors"`
	Terminal TerminalColors `json:"terminal"`
}

type ThemeService struct {
	ctx           context.Context
	settingsSvc   *SettingsService
	builtInPath   string
	userThemePath string
}

func NewThemeService(ctx context.Context, settingsSvc *SettingsService) *ThemeService {
	// Get executable path to find built-in themes
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	builtInPath := filepath.Join(exeDir, "..", "themes")

	// Get user config directory for custom themes
	configDir, _ := os.UserConfigDir()
	userThemePath := filepath.Join(configDir, "term", "themes")

	// Create user themes directory if it doesn't exist
	os.MkdirAll(userThemePath, 0755)

	return &ThemeService{
		ctx:           ctx,
		settingsSvc:   settingsSvc,
		builtInPath:   builtInPath,
		userThemePath: userThemePath,
	}
}

// GetAllThemes returns all available themes (built-in + user)
func (s *ThemeService) GetAllThemes() ([]Theme, error) {
	themes := []Theme{}

	// Load built-in themes
	builtInThemes, err := s.loadThemesFromDirectory(s.builtInPath)
	if err == nil {
		themes = append(themes, builtInThemes...)
	}

	// Load user themes
	userThemes, err := s.loadThemesFromDirectory(s.userThemePath)
	if err == nil {
		themes = append(themes, userThemes...)
	}

	return themes, nil
}

// GetTheme returns a specific theme by ID
func (s *ThemeService) GetTheme(id string) (*Theme, error) {
	themes, err := s.GetAllThemes()
	if err != nil {
		return nil, err
	}

	for _, theme := range themes {
		if theme.ID == id {
			return &theme, nil
		}
	}

	return nil, fmt.Errorf("theme not found: %s", id)
}

// GetActiveTheme returns the currently active theme
func (s *ThemeService) GetActiveTheme() (*Theme, error) {
	// Get active theme ID from settings
	setting, err := s.settingsSvc.GetSetting("active_theme")
	themeID := "dark" // default
	if err == nil && setting.Value != "" {
		themeID = setting.Value
	}

	return s.GetTheme(themeID)
}

// SetActiveTheme sets the active theme
func (s *ThemeService) SetActiveTheme(id string) error {
	// Verify theme exists
	_, err := s.GetTheme(id)
	if err != nil {
		return err
	}

	// Save to settings
	return s.settingsSvc.SetSetting("active_theme", id, "string")
}

// ImportTheme imports a theme from a JSON file
func (s *ThemeService) ImportTheme(sourcePath string) error {
	// Read the theme file
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read theme file: %w", err)
	}

	// Parse theme
	var theme Theme
	if err := json.Unmarshal(data, &theme); err != nil {
		return fmt.Errorf("failed to parse theme: %w", err)
	}

	// Validate theme
	if theme.ID == "" || theme.Name == "" {
		return fmt.Errorf("invalid theme: missing ID or name")
	}

	// Copy to user themes directory
	destPath := filepath.Join(s.userThemePath, theme.ID+".json")
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save theme: %w", err)
	}

	return nil
}

// ExportTheme exports a theme to a JSON file
func (s *ThemeService) ExportTheme(id string, destPath string) error {
	theme, err := s.GetTheme(id)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(theme, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal theme: %w", err)
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write theme file: %w", err)
	}

	return nil
}

// loadThemesFromDirectory loads all themes from a directory
func (s *ThemeService) loadThemesFromDirectory(dir string) ([]Theme, error) {
	themes := []Theme{}

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return themes, nil
	}

	// Read all JSON files
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var theme Theme
		if err := json.Unmarshal(data, &theme); err != nil {
			continue
		}

		themes = append(themes, theme)
	}

	return themes, nil
}
