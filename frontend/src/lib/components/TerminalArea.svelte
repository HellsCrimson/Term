<script lang="ts">
  import { terminalsStore } from '../stores/terminals.svelte';
  import TabBar from './TabBar.svelte';
  import TerminalView from './TerminalView.svelte';
  import RemoteDesktopView from './RemoteDesktopView.svelte';

  const { tabs, activeTabId } = $derived.by(() => ({
    tabs: terminalsStore.tabs,
    activeTabId: terminalsStore.activeTabId
  }));

  const activeTab = $derived(terminalsStore.getActiveTab());

  function isRemoteDesktopSession(sessionType: string): boolean {
    return ['rdp', 'vnc', 'telnet'].includes(sessionType);
  }
</script>

<div class="flex flex-col h-full" style="background: var(--bg-primary)">
  {#if tabs.length > 0}
    <!-- Tab Bar -->
    <TabBar />

    <!-- Terminal Views - Keep all mounted but only show active -->
    <div class="flex-1 overflow-hidden relative">
      {#each tabs as tab (tab.id)}
        <div class="absolute inset-0" style="display: {tab.active ? 'block' : 'none'}">
          {#key tab.id}
            {#if isRemoteDesktopSession(tab.sessionType)}
              <RemoteDesktopView {tab} />
            {:else}
              <TerminalView {tab} />
            {/if}
          {/key}
        </div>
      {/each}
    </div>
  {:else}
    <!-- Empty State -->
    <div class="flex flex-col items-center justify-center h-full" style="color: var(--text-muted)">
      <svg
        class="w-24 h-24 mb-4"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="1"
          d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
        />
      </svg>
      <h3 class="text-xl mb-2">No Active Terminals</h3>
      <p class="text-sm">Select or double-click a session from the sidebar to open a terminal</p>
    </div>
  {/if}
</div>
