package main

import (
	"embed"
	_ "embed"
	"log"
	"os"
	"path/filepath"

	"term/database"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	// Register terminal events
	application.RegisterEvent[map[string]interface{}]("terminal:data")
	application.RegisterEvent[map[string]interface{}]("terminal:exit")
	application.RegisterEvent[map[string]interface{}]("terminal:error")

	// Register system stats event
	application.RegisterEvent[SystemStats]("system:stats")
}

func main() {
	// Get data directory for database
	dataDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal("Failed to get config dir:", err)
	}
	dbPath := filepath.Join(dataDir, "term", "term.db")

	// Initialize database
	db, err := database.New(dbPath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Create services
	sessionService := NewSessionService(db)
	settingsService := NewSettingsService(db)
	loggingService := &LoggingService{}

	// Create Wails application
	app := application.New(application.Options{
		Name:        "Terminal Manager",
		Description: "A desktop terminal manager with session management",
		Services: []application.Service{
			application.NewService(sessionService),
			application.NewService(settingsService),
			application.NewService(loggingService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	// Create terminal service (needs app instance for events)
	terminalService := NewTerminalService(app)
	app.RegisterService(application.NewService(terminalService))

	// Create and start system stats service
	systemStatsService := NewSystemStatsService()
	systemStatsService.SetApp(app)
	app.RegisterService(application.NewService(systemStatsService))
	systemStatsService.Start()

	// Create main window
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Terminal Manager",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
		Width:            1200,
		Height:           800,
	})

	// Run the application
	err = app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
