import type { Terminal } from 'ghostty-web';
import * as TerminalService from '$bindings/term/terminalservice';
import { Events } from '@wailsio/runtime';

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

    return tab;
  }

  setActiveTab(id: string) {
    this.tabs.forEach(tab => {
      tab.active = tab.id === id;
    });
    this.activeTabId = id;
  }

  getActiveTab(): TerminalTab | null {
    return this.tabs.find(tab => tab.active) || null;
  }

  getTab(id: string): TerminalTab | undefined {
    return this.tabs.find(tab => tab.id === id);
  }

  closeTab(id: string) {
    const tab = this.getTab(id);
    if (tab) {
      // Close backend session
      if (!tab.exited) {
        this.closeSession(tab.backendSessionId);
      }

      // Dispose terminal
      if (tab.terminal) {
        tab.terminal.dispose();
      }
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
    }
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
}

export const terminalsStore = new TerminalsStore();
