<script lang="ts">
  import type { TreeNode, SessionNode } from '../types';
  import { sessionsStore } from '../stores/sessions.svelte';
  import { terminalsStore } from '../stores/terminals.svelte';
  import TreeNodeComponent from './TreeNodeComponent.svelte';

  const { tree, selectedNodeId } = $derived.by(() => ({
    tree: sessionsStore.tree,
    selectedNodeId: sessionsStore.selectedNodeId
  }));

  async function handleNodeClick(node: SessionNode) {
    sessionsStore.selectNode(node.id);

    // If it's a session and auto-launch is enabled, create a tab
    if (node.type === 'session' && node.sessionType) {
      // Check if already has an active tab
      const existingTab = terminalsStore.tabs.find(t => t.sessionId === node.id);
      if (existingTab) {
        terminalsStore.setActiveTab(existingTab.id);
      } else {
        // Create new tab
        terminalsStore.createTab(node.id, node.name, node.sessionType);
      }
    }
  }

  async function handleNodeDoubleClick(node: SessionNode) {
    if (node.type === 'session' && node.sessionType) {
      // Always create a new tab on double-click
      terminalsStore.createTab(node.id, node.name, node.sessionType);
    }
  }
</script>

<div class="session-tree p-2">
  {#if tree.length === 0}
    <div class="text-gray-500 text-sm p-4 text-center">
      No sessions available.<br />
      <button class="text-blue-500 hover:underline mt-2">Create your first session</button>
    </div>
  {:else}
    {#each tree as node}
      <TreeNodeComponent
        {node}
        selected={selectedNodeId === node.session.id}
        on:click={() => handleNodeClick(node.session)}
        on:dblclick={() => handleNodeDoubleClick(node.session)}
      />
    {/each}
  {/if}
</div>
