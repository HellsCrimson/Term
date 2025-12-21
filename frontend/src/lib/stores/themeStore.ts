import { writable } from 'svelte/store';

export interface ThemeColors {
  bg: {
    primary: string;
    secondary: string;
    tertiary: string;
    quaternary: string;
  };
  text: {
    primary: string;
    secondary: string;
    muted: string;
  };
  accent: {
    blue: string;
    green: string;
    red: string;
    yellow: string;
    purple: string;
    pink: string;
    cyan: string;
    orange: string;
  };
  border: string;
  hover: string;
  active: string;
  selection: string;
}

export interface TerminalColors {
  background: string;
  foreground: string;
  cursor: string;
  selectionBackground: string;
  black: string;
  red: string;
  green: string;
  yellow: string;
  blue: string;
  magenta: string;
  cyan: string;
  white: string;
  brightBlack: string;
  brightRed: string;
  brightGreen: string;
  brightYellow: string;
  brightBlue: string;
  brightMagenta: string;
  brightCyan: string;
  brightWhite: string;
}

export interface Theme {
  name: string;
  id: string;
  type: 'dark' | 'light';
  colors: ThemeColors;
  terminal: TerminalColors;
}

interface ThemeStore {
  themes: Theme[];
  activeTheme: Theme | null;
}

function createThemeStore() {
  const { subscribe, set, update } = writable<ThemeStore>({
    themes: [],
    activeTheme: null
  });

  return {
    subscribe,

    // Load all available themes
    async loadThemes() {
      try {
        const ThemeService = await import('$bindings/term/themeservice');
        const themes = await ThemeService.GetAllThemes();
        const activeTheme = await ThemeService.GetActiveTheme();

        update(state => ({
          themes: themes || [],
          activeTheme: activeTheme || null
        }));

        // Apply the active theme
        if (activeTheme) {
          this.applyTheme(activeTheme);
        }
      } catch (error) {
        console.error('Failed to load themes:', error);
      }
    },

    // Set the active theme
    async setTheme(themeId: string) {
      try {
        const ThemeService = await import('$bindings/term/themeservice');
        await ThemeService.SetActiveTheme(themeId);

        const theme = await ThemeService.GetTheme(themeId);

        update(state => ({
          ...state,
          activeTheme: theme
        }));

        this.applyTheme(theme);
      } catch (error) {
        console.error('Failed to set theme:', error);
      }
    },

    // Apply theme colors to CSS variables
    applyTheme(theme: Theme) {
      const root = document.documentElement;

      // Apply background colors
      root.style.setProperty('--bg-primary', theme.colors.bg.primary);
      root.style.setProperty('--bg-secondary', theme.colors.bg.secondary);
      root.style.setProperty('--bg-tertiary', theme.colors.bg.tertiary);
      root.style.setProperty('--bg-quaternary', theme.colors.bg.quaternary);

      // Apply text colors
      root.style.setProperty('--text-primary', theme.colors.text.primary);
      root.style.setProperty('--text-secondary', theme.colors.text.secondary);
      root.style.setProperty('--text-muted', theme.colors.text.muted);

      // Apply accent colors
      root.style.setProperty('--accent-blue', theme.colors.accent.blue);
      root.style.setProperty('--accent-green', theme.colors.accent.green);
      root.style.setProperty('--accent-red', theme.colors.accent.red);
      root.style.setProperty('--accent-yellow', theme.colors.accent.yellow);
      root.style.setProperty('--accent-purple', theme.colors.accent.purple);
      root.style.setProperty('--accent-pink', theme.colors.accent.pink);
      root.style.setProperty('--accent-cyan', theme.colors.accent.cyan);
      root.style.setProperty('--accent-orange', theme.colors.accent.orange);

      // Apply UI colors
      root.style.setProperty('--border-color', theme.colors.border);
      root.style.setProperty('--hover-bg', theme.colors.hover);
      root.style.setProperty('--active-bg', theme.colors.active);
      root.style.setProperty('--selection-bg', theme.colors.selection);

      // Store terminal colors for terminal instances
      root.setAttribute('data-theme-type', theme.type);
      root.setAttribute('data-theme-id', theme.id);

      // Also expose terminal palette as CSS variables for styling wrappers/previews
      root.style.setProperty('--term-background', theme.terminal.background);
      root.style.setProperty('--term-foreground', theme.terminal.foreground);
      root.style.setProperty('--term-cursor', theme.terminal.cursor);
      root.style.setProperty('--term-selection', theme.terminal.selectionBackground);
      root.style.setProperty('--term-black', theme.terminal.black);
      root.style.setProperty('--term-red', theme.terminal.red);
      root.style.setProperty('--term-green', theme.terminal.green);
      root.style.setProperty('--term-yellow', theme.terminal.yellow);
      root.style.setProperty('--term-blue', theme.terminal.blue);
      root.style.setProperty('--term-magenta', theme.terminal.magenta);
      root.style.setProperty('--term-cyan', theme.terminal.cyan);
      root.style.setProperty('--term-white', theme.terminal.white);
      root.style.setProperty('--term-bright-black', theme.terminal.brightBlack);
      root.style.setProperty('--term-bright-red', theme.terminal.brightRed);
      root.style.setProperty('--term-bright-green', theme.terminal.brightGreen);
      root.style.setProperty('--term-bright-yellow', theme.terminal.brightYellow);
      root.style.setProperty('--term-bright-blue', theme.terminal.brightBlue);
      root.style.setProperty('--term-bright-magenta', theme.terminal.brightMagenta);
      root.style.setProperty('--term-bright-cyan', theme.terminal.brightCyan);
      root.style.setProperty('--term-bright-white', theme.terminal.brightWhite);
    },

    // Import a theme from file
    async importTheme(filePath: string) {
      try {
        const ThemeService = await import('$bindings/term/themeservice');
        await ThemeService.ImportTheme(filePath);
        await this.loadThemes();
      } catch (error) {
        console.error('Failed to import theme:', error);
        throw error;
      }
    },

    // Export a theme to file
    async exportTheme(themeId: string, destPath: string) {
      try {
        const ThemeService = await import('$bindings/term/themeservice');
        await ThemeService.ExportTheme(themeId, destPath);
      } catch (error) {
        console.error('Failed to export theme:', error);
        throw error;
      }
    }
  };
}

export const themeStore = createThemeStore();
