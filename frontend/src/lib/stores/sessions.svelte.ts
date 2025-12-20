import type { SessionNode, TreeNode } from '../types';
import * as SessionService from '$bindings/term/sessionservice';

class SessionsStore {
  sessions = $state<SessionNode[]>([]);
  tree = $state<TreeNode[]>([]);
  selectedNodeId = $state<string | null>(null);
  loading = $state(false);

  async loadSessions() {
    this.loading = true;
    try {
      // Call Wails backend
      const sessions = await SessionService.GetAllSessions();
      this.sessions = sessions || [];
      const treeData = await SessionService.GetSessionTree();
      this.tree = treeData || [];

      // Log what we received
      await import('$bindings/term/loggingservice').then(log => {
        log.Log(`loadSessions: Got ${this.sessions.length} sessions, ${this.tree.length} root nodes`);
        const sessionIds = this.sessions.map(s => `${s.id}(parent=${s.parentId},pos=${s.position})`);
        log.Log(`loadSessions: Sessions: ${sessionIds.join(', ')}`);
      });
    } catch (error) {
      console.error('Failed to load sessions:', error);
      this.sessions = [];
      this.tree = [];
    } finally {
      this.loading = false;
    }
  }

  async createSession(session: Partial<SessionNode>) {
    try {
      await SessionService.CreateSession(session as any);
      await this.loadSessions();
    } catch (error) {
      console.error('Failed to create session:', error);
      throw error;
    }
  }

  async updateSession(session: SessionNode) {
    try {
      await SessionService.UpdateSession(session as any);
      await this.loadSessions();
    } catch (error) {
      console.error('Failed to update session:', error);
      throw error;
    }
  }

  async deleteSession(id: string, cascade: boolean = false) {
    try {
      await SessionService.DeleteSession(id, cascade);
      await this.loadSessions();
    } catch (error) {
      console.error('Failed to delete session:', error);
      throw error;
    }
  }

  async getEffectiveConfig(sessionId: string): Promise<Record<string, string>> {
    try {
      return await SessionService.GetEffectiveConfig(sessionId);
    } catch (error) {
      console.error('Failed to get effective config:', error);
      return {};
    }
  }

  async getSessionConfig(sessionId: string): Promise<Record<string, string>> {
    try {
      return await SessionService.GetSessionConfig(sessionId);
    } catch (error) {
      console.error('Failed to get session config:', error);
      return {};
    }
  }

  selectNode(id: string | null) {
    this.selectedNodeId = id;
  }

  getSelectedNode(): SessionNode | null {
    if (!this.selectedNodeId) return null;
    return this.sessions.find(s => s.id === this.selectedNodeId) || null;
  }

  async setSessionConfig(sessionId: string, key: string, value: string, valueType: string = 'string') {
    try {
      await SessionService.SetSessionConfig(sessionId, key, value, valueType);
    } catch (error) {
      console.error('Failed to set session config:', error);
      throw error;
    }
  }

  async moveSession(sessionId: string, newParentId: string | null, newPosition: number) {
    try {
      await SessionService.MoveSession(sessionId, newParentId, newPosition);
      await this.loadSessions();
    } catch (error) {
      console.error('Failed to move session:', error);
      throw error;
    }
  }
}

export const sessionsStore = new SessionsStore();
