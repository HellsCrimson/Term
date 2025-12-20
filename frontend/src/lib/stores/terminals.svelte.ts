import type { Terminal } from 'ghostty-web';
import * as TerminalService from '$bindings/term/terminalservice';
import * as RemoteStatsService from '$bindings/term/remotestatsservice';
import * as SystemStatsService from '$bindings/term/systemstatsservice';
import { Events } from '@wailsio/runtime';
import { settingsStore } from './settings.svelte';
import * as LoggingService from '$bindings/term/loggingservice';

export interface TerminalTab {
  id: string;
  sessionId: string; // Session node ID from sidebar (for config lookup)
  backendSessionId: string; // Unique backend PTY session ID (tab.id)
  sessionName: string;
  sessionType: string;
  terminal: Terminal | null;
  active: boolean;
  exited: boolean;
  exitCode?: number;
  pinned?: boolean;
}

class TerminalsStore {
  tabs = $state<TerminalTab[]>([]);
  activeTabId = $state<string | null>(null);

  constructor() {
    // Listen to terminal events from backend
    // Note: Events.On receives the full event object with structure: {name: string, data: {...}}
    Events.On('terminal:data', (event: any) => {
      const { id, data } = event.data;
      this.handleTerminalData(id, data);
    });

    Events.On('terminal:exit', (event: any) => {
      const { id, exitCode } = event.data;
      this.handleTerminalExit(id, exitCode);
    });

    Events.On('terminal:error', (event: any) => {
      const { id, error } = event.data;
      console.error('Terminal error:', id, error);
    });
  }

  createTab(sessionId: string, sessionName: string, sessionType: string): TerminalTab {
    const id = `tab-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;

    const tab: TerminalTab = {
      id,
      sessionId, // Session node ID from sidebar
      backendSessionId: id, // Use tab ID as unique backend session ID
      sessionName,
      sessionType,
      terminal: null,
      active: false,
      exited: false
    };

    this.tabs.push(tab);
    this.setActiveTab(id);
    this.saveTabSnapshots();

    return tab;
  }

  setActiveTab(id: string) {
    this.tabs.forEach(tab => {
      tab.active = tab.id === id;
    });
    this.activeTabId = id;

    // Notify both stats services about the active session
    const activeTab = this.tabs.find(tab => tab.id === id);
    if (activeTab) {
      const sessionId = activeTab.backendSessionId;
      RemoteStatsService.SetActiveSession(sessionId);
      SystemStatsService.SetActiveSession(sessionId);
    }
  }

  getActiveTab(): TerminalTab | null {
    return this.tabs.find(tab => tab.active) || null;
  }

  getTab(id: string): TerminalTab | undefined {
    return this.tabs.find(tab => tab.id === id);
  }

  closeTab(id: string, skipConfirmation: boolean = false) {
    LoggingService.Log(`closeTab called: id=${id}, skipConfirmation=${skipConfirmation}`, "INFO");
    const tab = this.getTab(id);
    if (!tab) {
      LoggingService.Log('Tab not found', "INFO");
      return;
    }

    LoggingService.Log(`Tab found: ${tab.sessionName}, exited=${tab.exited}, pinned=${tab.pinned}`, "INFO");

    // Prevent closing pinned tabs unless explicitly skipping confirmation
    if (tab.pinned && !skipConfirmation) {
      LoggingService.Log('Cannot close pinned tab', "INFO");
      alert(`Tab "${tab.sessionName}" is pinned. Unpin it first to close.`);
      return;
    }

    LoggingService.Log(`confirmTabClose setting: ${settingsStore.settings.confirmTabClose}`, "INFO");

    // Check for confirmation if enabled and not exited
    if (!skipConfirmation && settingsStore.settings.confirmTabClose && !tab.exited) {
      LoggingService.Log('Showing confirmation dialog', "INFO");
      if (!confirm(`Close tab "${tab.sessionName}"?`)) {
        LoggingService.Log('User cancelled close', "INFO");
        return;
      }
      LoggingService.Log('User confirmed close', "INFO");
    } else {
      LoggingService.Log(`Skipping confirmation: skipConfirmation=${skipConfirmation}, setting=${settingsStore.settings.confirmTabClose}, exited=${tab.exited}`, "INFO");
    }

    // Close backend session
    if (!tab.exited) {
      this.closeSession(tab.backendSessionId);
    }

    // Dispose terminal
    if (tab.terminal) {
      try {
        tab.terminal.clear();
        tab.terminal.reset();
        tab.terminal.dispose();
      } catch (e) {
        console.error('Error disposing terminal:', e);
      }
      tab.terminal = null;
    }

    const index = this.tabs.findIndex(t => t.id === id);
    if (index !== -1) {
      this.tabs.splice(index, 1);
    }

    // Activate another tab if this was active
    if (this.activeTabId === id && this.tabs.length > 0) {
      this.setActiveTab(this.tabs[0].id);
    } else if (this.tabs.length === 0) {
      this.activeTabId = null;
      // Clear stats services when no tabs are open
      RemoteStatsService.SetActiveSession("");
      SystemStatsService.SetActiveSession("");
    }

    LoggingService.Log(`Tab closed successfully, ${this.tabs.length} tabs remaining`, "INFO");
    this.saveTabSnapshots();
  }

  closeOtherTabs(id: string) {
    const tabsToClose = this.tabs.filter(t => t.id !== id);
    tabsToClose.forEach(tab => this.closeTab(tab.id));
  }

  closeAllExited() {
    const exitedTabs = this.tabs.filter(t => t.exited);
    exitedTabs.forEach(tab => this.closeTab(tab.id));
  }

  renameTab(id: string, newName: string) {
    const tab = this.getTab(id);
    if (tab) {
      tab.sessionName = newName;
    }
  }

  togglePin(id: string) {
    const tab = this.getTab(id);
    if (tab) {
      tab.pinned = !tab.pinned;
      LoggingService.Log(`Tab ${tab.sessionName} ${tab.pinned ? 'pinned' : 'unpinned'}`, "INFO");
    }
  }

  handleTerminalData(backendSessionId: string, data: string) {
    const tab = this.tabs.find(t => t.backendSessionId === backendSessionId);
    if (tab && tab.terminal) {
      try {
        tab.terminal.write(data);
      } catch (error) {
        console.error('Error writing to terminal:', error);
      }
    }
  }

  handleTerminalExit(backendSessionId: string, exitCode: number) {
    const tab = this.tabs.find(t => t.backendSessionId === backendSessionId);
    if (tab) {
      tab.exited = true;
      tab.exitCode = exitCode;

      // Show exit message in terminal
      if (tab.terminal) {
        const msg = `\r\n\r\n[Process exited with code ${exitCode}]\r\n`;
        tab.terminal.write(msg);
      }
    }
  }

  async startSession(
    sessionId: string,
    sessionType: string,
    config: Record<string, string>,
    cols: number,
    rows: number
  ) {
    try {
      await TerminalService.StartSession({
        id: sessionId,
        sessionType,
        config,
        cols,
        rows
      } as any);
    } catch (error) {
      console.error('Failed to start session:', error);
      throw error;
    }
  }

  async writeToSession(backendSessionId: string, data: string) {
    try {
      await TerminalService.WriteToSession(backendSessionId, data);
    } catch (error) {
      console.error('Failed to write to session:', error);
    }
  }

  async resizeSession(backendSessionId: string, cols: number, rows: number) {
    try {
      await TerminalService.ResizeSession(backendSessionId, cols, rows);
    } catch (error) {
      console.error('Failed to resize session:', error);
    }
  }

  async closeSession(backendSessionId: string) {
    try {
      await TerminalService.CloseSession(backendSessionId);
    } catch (error) {
      console.error('Failed to close session:', error);
    }
  }

  saveTabSnapshots() {
    // Save only non-exited tabs
    const snapshots = this.tabs
      .filter(tab => !tab.exited)
      .map(tab => ({
        sessionId: tab.sessionId,
        sessionName: tab.sessionName,
        sessionType: tab.sessionType
      }));
    settingsStore.saveTabSnapshots(snapshots);
  }

  async restoreTabs() {
    LoggingService.Log(`restoreTabs called, restoreTabsOnStartup=${settingsStore.settings.restoreTabsOnStartup}`, "INFO");
    if (!settingsStore.settings.restoreTabsOnStartup) {
      LoggingService.Log('Tab restoration disabled', "INFO");
      return;
    }

    const snapshots = await settingsStore.getTabSnapshots();
    LoggingService.Log(`Found ${snapshots.length} tab snapshots to restore`, "INFO");
    for (const snapshot of snapshots) {
      LoggingService.Log(`Restoring tab: ${snapshot.sessionName} (${snapshot.sessionType})`, "INFO");
      this.createTab(snapshot.sessionId, snapshot.sessionName, snapshot.sessionType);
    }
    LoggingService.Log(`Tab restoration complete, ${this.tabs.length} tabs created`, "INFO");
  }
}

export const terminalsStore = new TerminalsStore();
