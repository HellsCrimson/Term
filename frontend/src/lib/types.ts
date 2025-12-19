export interface SessionNode {
  id: string;
  parentId: string | null;
  name: string;
  type: 'folder' | 'session';
  sessionType?: 'ssh' | 'bash' | 'zsh' | 'fish' | 'pwsh' | 'git-bash' | 'custom';
  position: number;
  createdAt: string;
  updatedAt: string;
}

export interface TreeNode {
  session: SessionNode;
  children: TreeNode[];
}

export interface Config {
  id: number;
  sessionId: string;
  key: string;
  value: string;
  valueType: 'string' | 'int' | 'bool' | 'json';
  createdAt: string;
  updatedAt: string;
}

export interface Setting {
  key: string;
  value: string;
  valueType: 'string' | 'int' | 'bool' | 'json';
  createdAt: string;
  updatedAt: string;
}
