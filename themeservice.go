package main

import (
    "context"
    "encoding/json"
    "fmt"
    "embed"
    "os"
    "path/filepath"
    "io/fs"
    "strings"
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
    // Ensure defaults are present if this is first run
    _ = s.bootstrapDefaultThemes()

    // Load built-in and user themes
    builtInThemes, _ := s.loadThemesFromDirectory(s.builtInPath)
    userThemes, _ := s.loadThemesFromDirectory(s.userThemePath)

    // Deduplicate by ID (case-insensitive). User themes override built-in on conflict.
    byID := make(map[string]Theme)
    order := []string{}
    add := func(list []Theme) {
        for _, t := range list {
            key := strings.ToLower(strings.TrimSpace(t.ID))
            if key == "" {
                // Fallback to name if ID missing (shouldn't happen for built-ins)
                key = "name:" + strings.ToLower(strings.TrimSpace(t.Name))
            }
            if _, exists := byID[key]; !exists {
                order = append(order, key)
            }
            // Insert/override (user themes processed later will override built-in)
            byID[key] = t
        }
    }
    add(builtInThemes)
    add(userThemes)

    // Rebuild ordered list
    result := make([]Theme, 0, len(byID))
    for _, k := range order {
        result = append(result, byID[k])
    }
    return result, nil
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

	// Ensure built-in defaults exist if nothing is available yet
	if err := s.bootstrapDefaultThemes(); err != nil {
		// Ignore bootstrap error, still try to return a theme
	}

	th, getErr := s.GetTheme(themeID)
	if getErr == nil {
		return th, nil
	}
	// If requested theme is missing, fall back to dark and persist
	if _, derr := s.GetTheme("dark"); derr == nil {
		_ = s.settingsSvc.SetSetting("active_theme", "dark", "string")
		return s.GetTheme("dark")
	}
	return nil, getErr
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

    // Enforce uniqueness by ID and Name (case-insensitive) across all themes
    existing, _ := s.GetAllThemes()
    idLower := strings.ToLower(strings.TrimSpace(theme.ID))
    nameLower := strings.ToLower(strings.TrimSpace(theme.Name))
    for _, t := range existing {
        if strings.ToLower(strings.TrimSpace(t.ID)) == idLower {
            return fmt.Errorf("a theme with the same ID already exists: %s", theme.ID)
        }
        if strings.ToLower(strings.TrimSpace(t.Name)) == nameLower {
            return fmt.Errorf("a theme with the same name already exists: %s", theme.Name)
        }
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

// Embedded default themes for bootstrapping on first run
//go:embed themes/*.json
var embeddedThemesFS embed.FS

// bootstrapDefaultThemes copies embedded default themes to the user theme folder
// if no themes are currently available. It also ensures there is a valid
// "active_theme" setting.
func (s *ThemeService) bootstrapDefaultThemes() error {
    // Avoid recursion by checking the filesystem directly instead of calling GetAllThemes
    // If the user theme directory already contains any JSON themes, do nothing
    if matches, _ := filepath.Glob(filepath.Join(s.userThemePath, "*.json")); len(matches) > 0 {
        // Ensure active theme exists or set default
        if st, err := s.settingsSvc.GetSetting("active_theme"); err != nil || st.Value == "" {
            _ = s.settingsSvc.SetSetting("active_theme", "dark", "string")
        }
        return nil
    }

    // Install embedded defaults into user theme directory
    entries, err := fs.ReadDir(embeddedThemesFS, "themes")
    if err == nil {
        for _, e := range entries {
            if e.IsDir() {
                continue
            }
            data, rerr := embeddedThemesFS.ReadFile("themes/" + e.Name())
            if rerr != nil {
                continue
            }
            dest := filepath.Join(s.userThemePath, e.Name())
            _ = os.WriteFile(dest, data, 0644)
        }
    }

    // Set default active theme
    _ = s.settingsSvc.SetSetting("active_theme", "dark", "string")
    return nil
}
