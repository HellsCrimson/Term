<script lang="ts">
  import type { TreeNode } from '../types';
  import { createEventDispatcher } from 'svelte';
  import ContextMenu, { type MenuItem } from './ContextMenu.svelte';
  import EditSessionDialog from './EditSessionDialog.svelte';
  import { sessionsStore } from '../stores/sessions.svelte';
  import * as LoggingService from '$bindings/term/loggingservice';

  interface Props {
    node: TreeNode;
    selected?: boolean;
    level?: number;
  }

  let { node, selected = false, level = 0 }: Props = $props();

  const dispatch = createEventDispatcher();

  let expanded = $state(true);
  let showContextMenu = $state(false);
  let contextMenuX = $state(0);
  let contextMenuY = $state(0);
  let editingName = $state(false);
  let newName = $state(node.session.name);
  let isDragging = $state(false);
  let isDragOver = $state(false);
  let dragOverPosition = $state<'before' | 'inside' | 'after' | null>(null);
  let showEditDialog = $state(false);
  let hasMoved = $state(false);

  function toggleExpand() {
    if (node.session.type === 'folder') {
      expanded = !expanded;
    }
  }

  function getIcon(node: TreeNode): string {
    if (node.session.type === 'folder') {
      return expanded ? '📂' : '📁';
    }

    switch (node.session.sessionType) {
      case 'ssh':
        return '🔗';
      case 'bash':
      case 'zsh':
      case 'fish':
        return '💻';
      case 'pwsh':
        return '⚡';
      case 'git-bash':
        return '🌿';
      default:
        return '📟';
    }
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      toggleExpand();
      dispatch('click');
    } else if (e.key === ' ') {
      e.preventDefault();
      dispatch('dblclick');
    }
  }

  function handleContextMenu(e: MouseEvent) {
    e.preventDefault();
    contextMenuX = e.clientX;
    contextMenuY = e.clientY;
    showContextMenu = true;
  }

  async function handleRename() {
    if (newName.trim() && newName !== node.session.name) {
      try {
        await sessionsStore.updateSession({
          ...node.session,
          name: newName.trim()
        });
      } catch (error) {
        console.error('Failed to rename:', error);
        newName = node.session.name;
      }
    }
    editingName = false;
  }

  async function handleDelete() {
    const confirmMsg = node.session.type === 'folder'
      ? `Delete folder "${node.session.name}"? This will also delete all items inside.`
      : `Delete session "${node.session.name}"?`;

    if (confirm(confirmMsg)) {
      try {
        await sessionsStore.deleteSession(node.session.id, true);
      } catch (error) {
        console.error('Failed to delete:', error);
        alert('Failed to delete: ' + error);
      }
    }
  }

  async function handleDuplicate() {
    if (node.session.type === 'session') {
      try {
        const newId = `session-${Date.now()}`;
        const newSession = {
          ...node.session,
          id: newId,
          name: `${node.session.name} (Copy)`,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString()
        };
        await sessionsStore.createSession(newSession);
      } catch (error) {
        console.error('Failed to duplicate:', error);
        alert('Failed to duplicate: ' + error);
      }
    }
  }

  const contextMenuItems = $derived.by((): MenuItem[] => {
    const items: MenuItem[] = [
      {
        label: 'Edit',
        icon: '⚙️',
        action: () => {
          showEditDialog = true;
        }
      },
      {
        label: 'Rename',
        icon: '✏️',
        action: () => {
          newName = node.session.name;
          editingName = true;
        }
      }
    ];

    if (node.session.type === 'session') {
      items.push({
        label: 'Duplicate',
        icon: '📋',
        action: handleDuplicate
      });
    }

    items.push(
      { separator: true } as MenuItem,
      {
        label: 'Delete',
        icon: '🗑️',
        action: handleDelete,
        danger: true
      }
    );

    return items;
  });

  // Drag and drop handlers
  function handleDragStart(e: DragEvent) {
    if (editingName) {
      e.preventDefault();
      return;
    }

    hasMoved = false;
    isDragging = false;

    e.dataTransfer!.effectAllowed = 'move';
    e.dataTransfer!.setData('application/json', JSON.stringify({
      id: node.session.id,
      type: node.session.type,
      name: node.session.name
    }));
  }

  function handleDrag(e: DragEvent) {
    // Once we detect actual dragging movement, set the flag
    if (!hasMoved && (e.clientX !== 0 || e.clientY !== 0)) {
      hasMoved = true;
      isDragging = true;
    }
  }

  function handleDragEnd() {
    isDragging = false;
    hasMoved = false;
  }

  function handleDragOver(e: DragEvent) {
    if (editingName) return;

    e.preventDefault();
    e.dataTransfer!.dropEffect = 'move';

    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const y = e.clientY - rect.top;
    const height = rect.height;

    // Determine drop position based on mouse Y position
    if (node.session.type === 'folder') {
      if (y < height * 0.25) {
        dragOverPosition = 'before';
      } else if (y > height * 0.75) {
        dragOverPosition = 'after';
      } else {
        dragOverPosition = 'inside';
      }
    } else {
      dragOverPosition = y < height / 2 ? 'before' : 'after';
    }

    isDragOver = true;
  }

  function handleDragLeave() {
    isDragOver = false;
    dragOverPosition = null;
  }

  async function handleDrop(e: DragEvent) {
    e.preventDefault();
    e.stopPropagation();

    isDragOver = false;
    const position = dragOverPosition;
    dragOverPosition = null;

    if (!position) return;

    try {
      const data = JSON.parse(e.dataTransfer!.getData('application/json'));
      const draggedId = data.id;

      // Don't drop on self
      if (draggedId === node.session.id) return;

      // Don't drop parent into child
      if (isDescendant(draggedId, node.session.id)) {
        alert('Cannot move a folder into its own child');
        return;
      }

      const draggedNode = sessionsStore.sessions.find(s => s.id === draggedId);
      if (!draggedNode) return;

      let newParentId: string | null = null;
      let newPosition = 0;

      if (position === 'inside' && node.session.type === 'folder') {
        // Drop inside folder - add to end
        newParentId = node.session.id;
        const childrenInFolder = sessionsStore.sessions.filter(
          s => {
            const matchesParent = s.parentId === newParentId;
            const notDragged = s.id !== draggedId;
            return matchesParent && notDragged;
          }
        );
        newPosition = childrenInFolder.length;
        LoggingService.Log(`Drop inside folder: ${newParentId}, position: ${newPosition}, children: ${childrenInFolder.length}`, "INFO");
      } else {
        // Drop before or after - calculate position within siblings
        newParentId = node.session.parentId;

        // Get all siblings (excluding the dragged node)
        const siblings = sessionsStore.sessions
          .filter(s => {
            // Check if parentId matches
            const sameParent = (s.parentId === null && newParentId === null) ||
                              (s.parentId !== null && newParentId !== null && s.parentId === newParentId);
            const notDragged = s.id !== draggedId;
            return sameParent && notDragged;
          })
          .sort((a, b) => a.position - b.position);

        // Find where the target node is in the sorted siblings
        const targetIndex = siblings.findIndex(s => s.id === node.session.id);

        LoggingService.Log(`Drop ${position}: parent=${newParentId}, siblings=${siblings.length}, targetIndex=${targetIndex}`, "INFO");

        if (position === 'before') {
          newPosition = targetIndex;
        } else {
          newPosition = targetIndex + 1;
        }
      }

      LoggingService.Log(`BEFORE MOVE - DraggedId: ${draggedId}, NewParent: ${newParentId}, NewPosition: ${newPosition}`, "INFO");
      LoggingService.Log(`BEFORE MOVE - Total sessions: ${sessionsStore.sessions.length}`, "INFO");

      // Move and reload tree atomically
      await sessionsStore.moveSession(draggedId, newParentId, newPosition);

      LoggingService.Log(`AFTER MOVE - Total sessions: ${sessionsStore.sessions.length}`, "INFO");
      const movedSession = sessionsStore.sessions.find(s => s.id === draggedId);
      if (movedSession) {
        LoggingService.Log(`AFTER MOVE - Session found: parent=${movedSession.parentId}, position=${movedSession.position}`, "INFO");
      } else {
        LoggingService.Log(`AFTER MOVE - WARNING: Session ${draggedId} not found!`, "INFO");
      }

    } catch (error) {
      LoggingService.Log('Drop failed: ' + error);
      alert('Failed to move: ' + error);
    }
  }

  function isDescendant(ancestorId: string, descendantId: string): boolean {
    const findNode = (id: string, nodes: typeof sessionsStore.sessions): typeof node.session | undefined => {
      return nodes.find(n => n.id === id);
    };

    let current = findNode(descendantId, sessionsStore.sessions);
    while (current) {
      if (current.parentId === ancestorId) return true;
      current = current.parentId ? findNode(current.parentId, sessionsStore.sessions) : undefined;
    }
    return false;
  }
</script>

<div class="tree-node" style="padding-left: {level * 16}px">
  <div
    class="session-node flex items-center gap-2 {selected ? 'active' : ''} {isDragging ? 'opacity-50' : ''} {isDragOver && dragOverPosition === 'before' ? 'border-t-2 border-blue-500' : ''} {isDragOver && dragOverPosition === 'after' ? 'border-b-2 border-blue-500' : ''} {isDragOver && dragOverPosition === 'inside' ? 'bg-blue-500/20' : ''}"
    draggable="true"
    ondragstart={handleDragStart}
    ondrag={handleDrag}
    ondragend={handleDragEnd}
    ondragover={handleDragOver}
    ondragleave={handleDragLeave}
    ondrop={handleDrop}
    onclick={(e) => {
      if (!editingName && !hasMoved) {
        toggleExpand();
        dispatch('click');
      }
    }}
    ondblclick={(e) => {
      if (!editingName && !hasMoved) {
        dispatch('dblclick');
      }
    }}
    oncontextmenu={handleContextMenu}
    onkeydown={handleKeyDown}
    role="button"
    aria-label="{node.session.type === 'folder' ? 'Folder' : 'Session'}: {node.session.name}"
    tabindex="0"
  >
    <span class="text-base">{getIcon(node)}</span>
    {#if editingName}
      <input
        type="text"
        bind:value={newName}
        class="flex-1 px-2 py-0.5 text-sm bg-gray-700 border border-blue-500 rounded focus:outline-none"
        onblur={handleRename}
        onkeydown={(e) => {
          if (e.key === 'Enter') {
            handleRename();
          } else if (e.key === 'Escape') {
            newName = node.session.name;
            editingName = false;
          }
          e.stopPropagation();
        }}
        onclick={(e) => e.stopPropagation()}
        autofocus
      />
    {:else}
      <span class="flex-1 truncate text-sm">{node.session.name}</span>
    {/if}
    {#if node.session.type === 'folder' && node.children.length > 0}
      <span class="text-xs text-gray-500">
        {expanded ? '▼' : '▶'}
      </span>
    {/if}
  </div>

  {#if node.session.type === 'folder' && expanded && node.children.length > 0}
    <div class="children">
      {#each node.children as child}
        <svelte:self
          node={child}
          selected={selected && child.session.id === node.session.id}
          level={level + 1}
          on:click
          on:dblclick
        />
      {/each}
    </div>
  {/if}
</div>

<ContextMenu
  show={showContextMenu}
  x={contextMenuX}
  y={contextMenuY}
  items={contextMenuItems}
  onClose={() => showContextMenu = false}
/>

<EditSessionDialog
  show={showEditDialog}
  session={node.session}
  onClose={() => showEditDialog = false}
/>

<style>
  .tree-node {
    user-select: none;
  }

  .session-node[draggable="true"] {
    cursor: grab;
  }

  .session-node[draggable="true"]:active {
    cursor: grabbing;
  }
</style>
