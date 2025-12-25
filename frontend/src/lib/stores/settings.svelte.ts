// Settings store for app configuration
import * as SettingsService from '$bindings/term/settingsservice';

export interface AppSettings {
  theme: string;
  fontFamily: string;
  fontSize: number;
  autoLaunch: boolean;
  restoreTabsOnStartup: boolean;
  confirmTabClose: boolean;
  showStatusBar: boolean;
  recordingDefaultCaptureInput: boolean;
  recordingDefaultEncrypt: boolean;
}

class SettingsStore {
  settings = $state<AppSettings>({
    theme: 'dark',
    fontFamily: 'monospace',
    fontSize: 14,
    autoLaunch: true,
    restoreTabsOnStartup: true,
    confirmTabClose: false,
    showStatusBar: true,
    recordingDefaultCaptureInput: false,
    recordingDefaultEncrypt: true
  });
  loading = $state(false);

  async loadSettings() {
    this.loading = true;
    try {
      console.log('=== STORE loadSettings START ===');
      const allSettings = await SettingsService.GetAllSettings();
      console.log('Raw settings from backend:', allSettings);

      this.settings = {
        theme: allSettings.theme || 'dark',
        fontFamily: allSettings.font_family || 'monospace',
        fontSize: parseInt(allSettings.font_size || '14'),
        autoLaunch: allSettings.auto_launch === 'true',
        restoreTabsOnStartup: (allSettings.restore_tabs_on_startup || 'true') === 'true',
        confirmTabClose: (allSettings.confirm_tab_close || 'false') === 'true',
        showStatusBar: (allSettings.show_status_bar || 'true') === 'true',
        recordingDefaultCaptureInput: (allSettings.recording_default_capture_input || 'false') === 'true',
        recordingDefaultEncrypt: (allSettings.recording_default_encrypt || 'true') === 'true'
      };

      console.log('Parsed settings:', this.settings);
      console.log('=== STORE loadSettings END ===');
    } catch (error) {
      console.error('Failed to load settings:', error);
    } finally {
      this.loading = false;
    }
  }

  async setRecordingDefaultCaptureInput(v: boolean) {
    try {
      await SettingsService.SetSetting('recording_default_capture_input', v.toString(), 'bool');
      this.settings.recordingDefaultCaptureInput = v;
    } catch (error) {
      console.error('Failed to set recording default capture input:', error);
      throw error;
    }
  }

  async setRecordingDefaultEncrypt(v: boolean) {
    try {
      await SettingsService.SetSetting('recording_default_encrypt', v.toString(), 'bool');
      this.settings.recordingDefaultEncrypt = v;
    } catch (error) {
      console.error('Failed to set recording default encrypt:', error);
      throw error;
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

  async setRestoreTabsOnStartup(restore: boolean) {
    try {
      console.log(`Setting restoreTabsOnStartup to: ${restore}`);
      await SettingsService.SetRestoreTabsOnStartup(restore.toString());
      this.settings.restoreTabsOnStartup = restore;
      console.log(`restoreTabsOnStartup saved successfully`);
    } catch (error) {
      console.error('Failed to set restore tabs on startup:', error);
      throw error;
    }
  }

  async setConfirmTabClose(confirm: boolean) {
    try {
      console.log(`Setting confirmTabClose to: ${confirm}`);
      await SettingsService.SetConfirmTabClose(confirm.toString());
      this.settings.confirmTabClose = confirm;
      console.log(`confirmTabClose saved successfully`);
    } catch (error) {
      console.error('Failed to set confirm tab close:', error);
      throw error;
    }
  }

  async setShowStatusBar(show: boolean) {
    try {
      await SettingsService.SetShowStatusBar(show.toString());
      this.settings.showStatusBar = show;
    } catch (error) {
      console.error('Failed to set show status bar:', error);
      throw error;
    }
  }

  async saveTabSnapshots(tabs: Array<{sessionId: string, sessionName: string, sessionType: string}>) {
    try {
      await SettingsService.SaveTabSnapshots(JSON.stringify(tabs));
    } catch (error) {
      console.error('Failed to save tab snapshots:', error);
    }
  }

  async getTabSnapshots(): Promise<Array<{sessionId: string, sessionName: string, sessionType: string}>> {
    try {
      const snapshots = await SettingsService.GetTabSnapshots();
      return JSON.parse(snapshots);
    } catch (error) {
      console.error('Failed to get tab snapshots:', error);
      return [];
    }
  }
}

export const settingsStore = new SettingsStore();
