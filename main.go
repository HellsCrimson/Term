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

	// SSH host key verification events
	application.RegisterEvent[map[string]interface{}]("ssh:hostkey_prompt")
	application.RegisterEvent[map[string]interface{}]("ssh:hostkey_response")
	application.RegisterEvent[map[string]interface{}]("ssh:known_hosts:list:request")
	application.RegisterEvent[map[string]interface{}]("ssh:known_hosts:list")
    application.RegisterEvent[map[string]interface{}]("ssh:known_hosts:delete")

    // Recording events
    application.RegisterEvent[map[string]interface{}]("recording:start")
    application.RegisterEvent[map[string]interface{}]("recording:stop")
    application.RegisterEvent[map[string]interface{}]("recording:started")
    application.RegisterEvent[map[string]interface{}]("recording:stopped")
    application.RegisterEvent[map[string]interface{}]("recording:list:request")
    application.RegisterEvent[map[string]interface{}]("recording:list")
    application.RegisterEvent[map[string]interface{}]("recording:delete")
    application.RegisterEvent[map[string]interface{}]("recording:list:error")
    application.RegisterEvent[map[string]interface{}]("recording:replay:start")
    application.RegisterEvent[map[string]interface{}]("recording:replay:stop")
    application.RegisterEvent[map[string]interface{}]("recording:replay:header")
    application.RegisterEvent[map[string]interface{}]("recording:replay:output")
    application.RegisterEvent[map[string]interface{}]("recording:replay:resize")
    application.RegisterEvent[map[string]interface{}]("recording:replay:ended")
    application.RegisterEvent[map[string]interface{}]("recording:replay:meta")
    application.RegisterEvent[map[string]interface{}]("recording:replay:progress")
    application.RegisterEvent[map[string]interface{}]("recording:replay:pause")
    application.RegisterEvent[map[string]interface{}]("recording:replay:resume")
    application.RegisterEvent[map[string]interface{}]("recording:replay:rewind")
    application.RegisterEvent[map[string]interface{}]("recording:replay:setSpeed")
    application.RegisterEvent[map[string]interface{}]("recording:replay:seek")

    // Key management events
    application.RegisterEvent[map[string]interface{}]("keys:generate")
    application.RegisterEvent[map[string]interface{}]("keys:generated")
    application.RegisterEvent[map[string]interface{}]("keys:import")
    application.RegisterEvent[map[string]interface{}]("keys:imported")
    application.RegisterEvent[map[string]interface{}]("keys:list:request")
    application.RegisterEvent[map[string]interface{}]("keys:list")
    application.RegisterEvent[map[string]interface{}]("keys:delete")
    application.RegisterEvent[map[string]interface{}]("keys:deleted")
    application.RegisterEvent[map[string]interface{}]("keys:export:public")
    application.RegisterEvent[map[string]interface{}]("keys:public_key")
    application.RegisterEvent[map[string]interface{}]("keys:error")
    application.RegisterEvent[map[string]interface{}]("recording:share")
    application.RegisterEvent[map[string]interface{}]("recording:shared")
    application.RegisterEvent[map[string]interface{}]("recording:share:error")
    application.RegisterEvent[map[string]interface{}]("recording:shared_with:request")
    application.RegisterEvent[map[string]interface{}]("recording:shared_with")
    application.RegisterEvent[map[string]interface{}]("recording:shared_with:error")
    application.RegisterEvent[map[string]interface{}]("recording:revoke_share")
    application.RegisterEvent[map[string]interface{}]("recording:share_revoked")
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
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID:      "dd231868-8745-42a5-a173-4b2f7565b82c",
			EncryptionKey: [32]byte{}, // TODO: set encryption key at build time
		},
	})

    // Host key service for SSH verification
    hostKeyService := NewHostKeyService(app, db)

    // Recording service for binary terminal recordings
    recordingService := NewRecordingService(app, db)
    app.RegisterService(application.NewService(recordingService))

    // Key management service for secure recording sharing
    keyMgmtService := NewKeyManagementService(db, app)
    keyMgmtService.Setup()
    app.RegisterService(application.NewService(keyMgmtService))

    // Create terminal service (needs app instance for events and host key verification and recorder)
    terminalService := NewTerminalService(app, hostKeyService, recordingService)
    app.RegisterService(application.NewService(terminalService))

	sftpService := NewSFTPService(app, terminalService)
	app.RegisterService(application.NewService(sftpService))

    // Create theme service (needs app context)
    themeService := NewThemeService(app.Context(), settingsService)
    app.RegisterService(application.NewService(themeService))

	// Create and start system stats service (needs terminal service to check session types)
	systemStatsService := NewSystemStatsService(terminalService)
	systemStatsService.SetApp(app)
	app.RegisterService(application.NewService(systemStatsService))
	systemStatsService.Start()

	// Create and start remote stats service (for monitoring SSH remote machines)
	remoteStatsService := NewRemoteStatsService(terminalService)
	remoteStatsService.SetApp(app)
	app.RegisterService(application.NewService(remoteStatsService))
	remoteStatsService.Start()

	// Create Guacamole service and HTTP server
	guacService := NewGuacamoleService(sessionService)
	httpServer := NewHTTPServer(3000, guacService, terminalService)
	if err := httpServer.Start(); err != nil {
		log.Printf("Failed to start HTTP server: %v", err)
	}
	defer httpServer.Stop()

	// Create main window
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Terminal Manager",
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarDefault,
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
