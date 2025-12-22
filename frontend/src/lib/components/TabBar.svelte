<script lang="ts">
  import { terminalsStore } from '../stores/terminals.svelte';
  import type { TerminalTab } from '../stores/terminals.svelte';
  import ContextMenu, { type MenuItem } from './ContextMenu.svelte';

  const { tabs } = $derived.by(() => ({
    tabs: terminalsStore.tabs
  }));

  let showContextMenu = $state(false);
  let contextMenuX = $state(0);
  let contextMenuY = $state(0);
  let contextMenuTab: TerminalTab | null = $state(null);
  let renamingTab: TerminalTab | null = $state(null);
  let newTabName = $state('');

  function handleTabClick(tab: TerminalTab) {
    terminalsStore.setActiveTab(tab.id);
  }

  function handleTabClose(e: MouseEvent, tab: TerminalTab) {
    e.stopPropagation();
    terminalsStore.closeTab(tab.id);
  }

  function handleTabContextMenu(e: MouseEvent, tab: TerminalTab) {
    e.preventDefault();
    contextMenuX = e.clientX;
    contextMenuY = e.clientY;
    contextMenuTab = tab;
    showContextMenu = true;
  }

  function handleTabKeyDown(e: KeyboardEvent, tab: TerminalTab) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      handleTabClick(tab);
    }
  }

  function handleRenameTab(tab: TerminalTab) {
    newTabName = tab.sessionName;
    renamingTab = tab;
  }

  function finishRename() {
    if (renamingTab && newTabName.trim()) {
      terminalsStore.renameTab(renamingTab.id, newTabName.trim());
    }
    renamingTab = null;
  }

  function handleDuplicateTab(tab: TerminalTab) {
    terminalsStore.createTab(tab.sessionId, tab.sessionName, tab.sessionType);
  }

  function handleReconnect(tab: TerminalTab) {
    // Close and recreate the terminal
    terminalsStore.closeTab(tab.id);
    terminalsStore.createTab(tab.sessionId, tab.sessionName, tab.sessionType);
  }

  function handleClearBuffer(tab: TerminalTab) {
    if (tab.terminal) {
      tab.terminal.clear();
    }
  }

  function handleCloseOthers(tab: TerminalTab) {
    terminalsStore.closeOtherTabs(tab.id);
  }

  function handleCloseAllExited() {
    terminalsStore.closeAllExited();
  }

  const contextMenuItems = $derived.by((): MenuItem[] => {
    if (!contextMenuTab) return [];

    const items: MenuItem[] = [
      {
        label: contextMenuTab.pinned ? 'Unpin Tab' : 'Pin Tab',
        icon: contextMenuTab.pinned ? '📌' : '📍',
        action: () => terminalsStore.togglePin(contextMenuTab!.id)
      },
      { separator: true } as MenuItem,
      {
        label: 'Rename',
        icon: '✏️',
        action: () => handleRenameTab(contextMenuTab!)
      },
      {
        label: 'Duplicate',
        icon: '📋',
        action: () => handleDuplicateTab(contextMenuTab!)
      }
    ];

    if (contextMenuTab.exited) {
      items.push({
        label: 'Reconnect',
        icon: '🔄',
        action: () => handleReconnect(contextMenuTab!)
      });
    } else {
      items.push({
        label: 'Clear Buffer',
        icon: '🧹',
        action: () => handleClearBuffer(contextMenuTab!)
      });
    }

    items.push(
      { separator: true } as MenuItem,
      {
        label: 'Close Others',
        icon: '✖️',
        action: () => handleCloseOthers(contextMenuTab!),
        disabled: tabs.length <= 1
      },
      {
        label: 'Close All Exited',
        icon: '🗑️',
        action: handleCloseAllExited,
        disabled: !tabs.some(t => t.exited)
      },
      { separator: true } as MenuItem,
      {
        label: 'Close',
        icon: '❌',
        action: () => terminalsStore.closeTab(contextMenuTab!.id),
        danger: true,
        disabled: contextMenuTab.pinned
      }
    );

    return items;
  });
</script>

<div class="flex overflow-x-auto" style="background: var(--bg-secondary); border-bottom: 1px solid var(--border-color)">
  {#each tabs as tab (tab.id)}
    <div
      class="terminal-tab flex items-center min-w-0 gap-2 {tab.active ? 'active' : ''} {tab.exited ? 'exited' : ''}"
      onclick={() => renamingTab !== tab && handleTabClick(tab)}
      oncontextmenu={(e) => handleTabContextMenu(e, tab)}
      onkeydown={(e) => handleTabKeyDown(e, tab)}
      role="tab"
      aria-selected={tab.active}
      tabindex="0"
    >
      {#if renamingTab === tab}
        <!-- svelte-ignore a11y_autofocus -->
        <input
          type="text"
          bind:value={newTabName}
          class="flex-1 px-2 py-0.5 text-sm rounded focus:outline-none min-w-24 border"
          style="background: var(--bg-tertiary); border-color: var(--accent-blue)"
          onblur={finishRename}
          onkeydown={(e) => {
            if (e.key === 'Enter') {
              finishRename();
            } else if (e.key === 'Escape') {
              renamingTab = null;
            }
            e.stopPropagation();
          }}
          onclick={(e) => e.stopPropagation()}
          autofocus
        />
      {:else}
        <span class="flex-1 truncate text-sm flex items-center gap-1">
          {#if tab.pinned}
            <span class="text-xs" title="Pinned">📌</span>
          {/if}
          {tab.sessionName}
          {#if tab.exited}
            <span class="text-xs ml-1">(exited {tab.exitCode ?? ''})</span>
          {/if}
        </span>
      {/if}

      {#if !tab.pinned}
        <button
          class="ml-2 rounded p-0.5 hover:bg-[var(--hover-bg)]"
          onclick={(e) => handleTabClose(e, tab)}
          aria-label="Close tab"
        >
          <svg class="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
            <path
              fill-rule="evenodd"
              d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
              clip-rule="evenodd"
            />
          </svg>
        </button>
      {/if}
    </div>
  {/each}
</div>

<ContextMenu
  show={showContextMenu}
  x={contextMenuX}
  y={contextMenuY}
  items={contextMenuItems}
  onClose={() => {
    showContextMenu = false;
    contextMenuTab = null;
  }}
/>
