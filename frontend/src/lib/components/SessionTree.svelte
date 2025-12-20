<script lang="ts">
  import type { TreeNode, SessionNode } from '../types';
  import { sessionsStore } from '../stores/sessions.svelte';
  import { terminalsStore } from '../stores/terminals.svelte';
  import TreeNodeComponent from './TreeNodeComponent.svelte';

  let searchQuery = $state('');

  const { tree, selectedNodeId } = $derived.by(() => ({
    tree: sessionsStore.tree,
    selectedNodeId: sessionsStore.selectedNodeId
  }));

  // Filter tree based on search query
  const filteredTree = $derived.by(() => {
    if (!searchQuery.trim()) {
      return tree;
    }

    const query = searchQuery.toLowerCase();

    function filterNode(node: TreeNode): TreeNode | null {
      const nameMatch = node.session.name.toLowerCase().includes(query);

      // For sessions, check name and type
      if (node.session.type === 'session') {
        const typeMatch = node.session.sessionType?.toLowerCase().includes(query);
        if (nameMatch || typeMatch) {
          return node;
        }
        return null;
      }

      // For folders, check name and filter children
      if (node.session.type === 'folder') {
        const filteredChildren = node.children
          .map(child => filterNode(child))
          .filter(child => child !== null) as TreeNode[];

        // Show folder if name matches or has matching children
        if (nameMatch || filteredChildren.length > 0) {
          return {
            ...node,
            children: filteredChildren
          };
        }
      }

      return null;
    }

    return tree.map(node => filterNode(node)).filter(node => node !== null) as TreeNode[];
  });

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

  function clearSearch() {
    searchQuery = '';
  }
</script>

<div class="session-tree flex flex-col h-full">
  <!-- Search input -->
  <div class="p-2 border-b border-gray-700">
    <div class="relative">
      <input
        type="text"
        bind:value={searchQuery}
        placeholder="Search sessions..."
        class="w-full px-3 py-1.5 pr-8 bg-gray-700 border border-gray-600 rounded text-sm focus:outline-none focus:border-blue-500"
      />
      {#if searchQuery}
        <button
          onclick={clearSearch}
          class="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-200"
          aria-label="Clear search"
        >
          ✕
        </button>
      {/if}
    </div>
  </div>

  <!-- Session tree -->
  <div class="flex-1 overflow-y-auto p-2">
    {#if tree.length === 0}
      <div class="text-gray-500 text-sm p-4 text-center">
        No sessions available.<br />
        <button class="text-blue-500 hover:underline mt-2">Create your first session</button>
      </div>
    {:else if filteredTree.length === 0}
      <div class="text-gray-500 text-sm p-4 text-center">
        No sessions match "{searchQuery}"
      </div>
    {:else}
      {#each filteredTree as node}
        <TreeNodeComponent
          {node}
          selected={selectedNodeId === node.session.id}
          on:click={(e) => handleNodeClick(e.detail)}
          on:dblclick={(e) => handleNodeDoubleClick(e.detail)}
        />
      {/each}
    {/if}
  </div>
</div>
