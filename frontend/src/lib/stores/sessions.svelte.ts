import type { SessionNode, TreeNode } from '../types';
import * as SessionService from '$bindings/term/sessionservice';
import { LoggingService } from '$bindings/term';

type sessionType = 'ssh' | 'bash' | 'zsh' | 'fish' | 'pwsh' | 'git-bash' | 'rdp' | 'vnc' | 'telnet' | 'custom' | 'powershell' | 'cmd' | 'serial' | undefined;

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
      // Cast the data to proper types (backend returns string, we need literal types)
      this.sessions = (sessions || []).map(s => ({
        ...s,
        type: s.type as 'folder' | 'session',
        sessionType: s.sessionType as sessionType
      }));

      const treeData = await SessionService.GetSessionTree();
      // Cast tree data recursively
      this.tree = (treeData || []).map(node => this.castTreeNode(node));

      // Log what we received
      LoggingService.Log(`loadSessions: Got ${this.sessions.length} sessions, ${this.tree.length} root nodes`, "DEBUG");
      const sessionIds = this.sessions.map(s => `${s.id}(parent=${s.parentId},pos=${s.position},type=${s.type},sessionType=${s.sessionType})`);
      LoggingService.Log(`loadSessions: Sessions: ${sessionIds.join(', ')}`, "DEBUG");
    } catch (error) {
      LoggingService.Log(`Failed to load sessions: ${error}`, "ERROR");
      this.sessions = [];
      this.tree = [];
    } finally {
      this.loading = false;
    }
  }

  private castTreeNode(node: any): TreeNode {
    return {
      session: {
        ...node.session,
        type: node.session.type as 'folder' | 'session',
        sessionType: node.session.sessionType as sessionType
      },
      children: (node.children || []).map((child: any) => this.castTreeNode(child))
    };
  }

  async createSession(session: Partial<SessionNode>) {
    try {
      await SessionService.CreateSession(session as any);
      await this.loadSessions();
    } catch (error) {
      LoggingService.Log(`Failed to create session: ${error}`, "ERROR");
      throw error;
    }
  }

  async updateSession(session: SessionNode) {
    try {
      await SessionService.UpdateSession(session as any);
      await this.loadSessions();
    } catch (error) {
      LoggingService.Log(`Failed to update session: ${error}`, "ERROR");
      throw error;
    }
  }

  async deleteSession(id: string, cascade: boolean = false) {
    try {
      await SessionService.DeleteSession(id, cascade);
      await this.loadSessions();
    } catch (error) {
      LoggingService.Log(`Failed to delete session: ${error}`, "ERROR");
      throw error;
    }
  }

  async getEffectiveConfig(sessionId: string): Promise<Record<string, string>> {
    try {
      return await SessionService.GetEffectiveConfig(sessionId);
    } catch (error) {
      LoggingService.Log(`Failed to get effective config: ${error}`, "ERROR");
      return {};
    }
  }

  async getSessionConfig(sessionId: string): Promise<Record<string, string>> {
    try {
      return await SessionService.GetSessionConfig(sessionId);
    } catch (error) {
      LoggingService.Log(`Failed to get session config: ${error}`, "ERROR");
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
      LoggingService.Log(`[Frontend] setSessionConfig: sessionId=${sessionId}, key=${key}, value=${value}, valueType=${valueType}`, "DEBUG");
      await SessionService.SetSessionConfig(sessionId, key, value, valueType);
      LoggingService.Log(`[Frontend] setSessionConfig SUCCESS: ${key}=${value}`, "DEBUG");
    } catch (error) {
      LoggingService.Log(`Failed to set session config: ${error}`, "ERROR");
      throw error;
    }
  }

  async moveSession(sessionId: string, newParentId: string | null, newPosition: number) {
    try {
      await SessionService.MoveSession(sessionId, newParentId, newPosition);
      await this.loadSessions();
    } catch (error) {
      LoggingService.Log(`Failed to move session: ${error}`, "ERROR");
      throw error;
    }
  }
}

export const sessionsStore = new SessionsStore();
