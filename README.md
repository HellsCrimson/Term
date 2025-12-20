# Terminal Manager

Terminal Manager is a desktop terminal and remote-session manager built with Wails v3 (Go backend + Svelte 5 frontend). It lets you organize sessions in a tree with folders and inheritance, open multiple terminals in tabs, and connect to SSH. It also integrates Apache Guacamole (via `guacd`) for RDP, VNC, and Telnet sessions streamed into the app.

## Highlights

- Session tree with folders and inheritance (configs cascade parent → child)
- Local shells: Bash, Zsh, Fish, PowerShell, Git Bash, plus custom commands
- SSH sessions (password or key auth)
- Remote desktop via Guacamole: RDP, VNC, Telnet
- Tabbed interface with pin/rename/duplicate/reconnect/close options
- Drag-and-drop reorder/move sessions and folders
- Search sessions by name/type; context menus for nodes and tabs
- Persisted settings and session snapshots (SQLite)
- Live system stats status bar (CPU, RAM, Disk, Net, Load)

## How It Works

- Backend (Go, Wails v3):
  - Manages sessions, configs, and settings in a SQLite database located under the OS config directory (e.g. `~/.config/term/term.db` on Linux).
  - Spawns local shells and SSH sessions using PTYs; streams I/O via Wails events.
  - Exposes an HTTP server on `localhost:3000` for Guacamole WebSocket tunneling.
- Frontend (Svelte 5 + Tailwind):
  - Renders the session tree, terminal tabs, and remote desktops.
  - Uses `ghostty-web` for a fast WebAssembly terminal emulator.
  - Uses `guacamole-common-js` to render RDP/VNC/Telnet sessions.

## Features In Detail

### Session Tree & Inheritance
- Create folders and sessions; reorder and reparent via drag-and-drop.
- Each node can define key/value configuration; effective config is resolved by merging parents into children (child overrides parent).
- Context menu actions on nodes: New session/subfolder, Rename, Duplicate (for sessions), Delete (with cascade for folders).

### Terminal Sessions
- Supported types: `bash`, `zsh`, `fish`, `pwsh` (PowerShell), `git-bash` (Windows), and `custom`.
- Config options:
  - `working_directory`: absolute path (supports `~` expansion)
  - `environment_variables`: semicolon-separated `KEY=value;KEY2=value2`
  - `startup_commands`: semicolon-separated commands run after the shell starts

### SSH Sessions
- Config options:
  - `ssh_host` (required), `ssh_port` (default `22`)
  - `ssh_username`
  - `ssh_auth_method`: `password` or `key`
  - If `password`: `ssh_password`
  - If `key`: `ssh_key_path` (supports `~` expansion)

Note: SSH currently skips host key verification (uses `InsecureIgnoreHostKey`) — add verification before production use.

### Remote Desktop (RDP/VNC/Telnet via Guacamole)
- Requires a running `guacd` on `localhost:4822`.
- The app opens a WebSocket tunnel to `ws://localhost:3000/api/guacamole/:sessionId` and streams the remote display.
- Session type-specific config keys:
  - RDP: `rdp_host`, `rdp_port` (default `3389`), `rdp_username`, `rdp_password`, `rdp_domain`, `rdp_security` (`any|nla|tls|rdp`)
  - VNC: `vnc_host`, `vnc_port` (default `5900`), `vnc_password`
  - Telnet: `telnet_host`, `telnet_port` (default `23`), `telnet_username`, `telnet_password`
- Desktop parameters (RDP/VNC): `desktop_width` (default `1920`), `desktop_height` (default `1080`), `desktop_color_depth` (`8|16|24|32`)

### Tabs, Shortcuts, and UX
- Tabs: pin, rename, duplicate, reconnect (if exited), clear buffer, close others, close all exited, close.
- Keyboard shortcuts:
  - `Ctrl+T`: New terminal from selected session
  - `Ctrl+W`: Close active tab
  - `Ctrl+Tab` / `Ctrl+Shift+Tab`: Cycle tabs
- Session selection can auto-launch a tab; double-click always opens a new tab.
- Tab snapshots persist across restarts (optional restore on startup).

### Settings (SQLite-backed)
- Theme, font family/size
- Auto-launch behavior, restore tabs on startup, confirm tab close
- Show/hide status bar

### System Stats Bar
- Emits `system:stats` every 2s (CPU, memory, disk, net speeds, load averages) and shows a compact HUD.

## Quick Start (Development)

Prerequisites:
- Go 1.21+
- Node.js 18+
- Wails v3 CLI (`go install github.com/wailsapp/wails/v3/cmd/wails3@latest`)
- For RDP/VNC/Telnet: `guacd` running on `localhost:4822`

Install frontend deps (on the first run or when `frontend/package.json` changes):

```
cd frontend && npm install
```

Run the app with hot reload:

```
wails3 dev
```

Build a production binary:

```
wails3 build
```

The app also starts a local HTTP server on port `3000` (used for Guacamole tunnels).

## Data & Paths

- Database: SQLite at `os.UserConfigDir()/term/term.db` (e.g., Linux: `~/.config/term/term.db`).
- Default bootstrap content includes example folders and sessions, plus sane default settings.

## Project Structure

- `main.go`: App bootstrap, services registration, window creation
- `terminalservice.go`: Local shell + SSH PTY management and I/O
- `sessionservice.go`: CRUD, tree building, move/duplicate, config inheritance
- `settingsservice.go`: App settings get/set, tab snapshot persistence
- `systemstatsservice.go`: Periodic system metrics emitter
- `guacamoleservice.go` + `httpserver.go`: Guacamole tunnel and WebSocket endpoint
- `database/`: SQLite schema, models, migrations, bootstrap
- `frontend/`: Svelte 5 app (components, stores, bindings, Tailwind config)

## Notes & Limitations

- SSH host key verification is currently disabled. For production, implement verification.
- `git-bash` is only applicable on Windows and must be installed locally.
- Remote desktop requires `guacd` reachable at `localhost:4822`.
- Some of the values (local port, guacd port) are not configurable and aren't using dynamic ports, so the port must be available
