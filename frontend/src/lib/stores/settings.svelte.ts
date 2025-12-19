// Settings store for app configuration
import * as SettingsService from '$bindings/term/settingsservice';

export interface AppSettings {
  theme: string;
  fontFamily: string;
  fontSize: number;
  autoLaunch: boolean;
}

class SettingsStore {
  settings = $state<AppSettings>({
    theme: 'dark',
    fontFamily: 'monospace',
    fontSize: 14,
    autoLaunch: true
  });
  loading = $state(false);

  async loadSettings() {
    this.loading = true;
    try {
      const allSettings = await SettingsService.GetAllSettings();
      this.settings = {
        theme: allSettings.theme || 'dark',
        fontFamily: allSettings.font_family || 'monospace',
        fontSize: parseInt(allSettings.font_size || '14'),
        autoLaunch: allSettings.auto_launch === 'true'
      };
    } catch (error) {
      console.error('Failed to load settings:', error);
    } finally {
      this.loading = false;
    }
  }

  async setTheme(theme: string) {
    try {
      await SettingsService.SetTheme(theme);
      this.settings.theme = theme;
    } catch (error) {
      console.error('Failed to set theme:', error);
    }
  }

  async setFontFamily(fontFamily: string) {
    try {
      await SettingsService.SetFontFamily(fontFamily);
      this.settings.fontFamily = fontFamily;
    } catch (error) {
      console.error('Failed to set font family:', error);
    }
  }

  async setFontSize(fontSize: number) {
    try {
      await SettingsService.SetFontSize(fontSize.toString());
      this.settings.fontSize = fontSize;
    } catch (error) {
      console.error('Failed to set font size:', error);
    }
  }

  async setAutoLaunch(autoLaunch: boolean) {
    try {
      await SettingsService.SetAutoLaunch(autoLaunch.toString());
      this.settings.autoLaunch = autoLaunch;
    } catch (error) {
      console.error('Failed to set auto launch:', error);
    }
  }
}

export const settingsStore = new SettingsStore();
