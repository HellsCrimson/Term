<script lang="ts">
  import './app.css';
  import { onMount } from 'svelte';
  import { init as initGhostty } from 'ghostty-web';
  import SessionTree from './lib/components/SessionTree.svelte';
  import TerminalArea from './lib/components/TerminalArea.svelte';
  import NewSessionDialog from './lib/components/NewSessionDialog.svelte';
  import SettingsDialog from './lib/components/SettingsDialog.svelte';
  import { sessionsStore } from './lib/stores/sessions.svelte';
  import { settingsStore } from './lib/stores/settings.svelte';

  let sidebarWidth = $state(250);
  let resizing = $state(false);
  let ready = $state(false);
  let showNewSessionDialog = $state(false);
  let showSettingsDialog = $state(false);

  onMount(async () => {
    // Initialize ghostty-web WASM (only once globally)
    await initGhostty();

    // Load sessions and settings on mount
    await Promise.all([
      sessionsStore.loadSessions(),
      settingsStore.loadSettings()
    ]);
    ready = true;
  });

  function handleMouseDown(e: MouseEvent) {
    resizing = true;
    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  }

  function handleMouseMove(e: MouseEvent) {
    if (resizing) {
      sidebarWidth = Math.max(200, Math.min(500, e.clientX));
    }
  }

  function handleMouseUp() {
    resizing = false;
    document.removeEventListener('mousemove', handleMouseMove);
    document.removeEventListener('mouseup', handleMouseUp);
  }
</script>

{#if !ready}
  <div class="flex items-center justify-center h-screen bg-gray-900 text-white">
    <p>Loading...</p>
  </div>
{:else}
  <div class="flex h-screen w-screen overflow-hidden bg-gray-900 text-gray-100">
    <!-- Sidebar -->
    <div
      class="flex-shrink-0 bg-gray-800 border-r border-gray-700 overflow-hidden"
      style="width: {sidebarWidth}px"
    >
      <div class="h-full flex flex-col">
        <!-- Sidebar Header -->
        <div class="px-4 py-3 border-b border-gray-700 flex items-center justify-between">
          <h2 class="text-lg font-semibold">Sessions</h2>
          <div class="flex gap-2">
            <button
              class="px-2 py-1 text-sm bg-blue-600 hover:bg-blue-700 rounded transition-colors"
              onclick={() => showNewSessionDialog = true}
              aria-label="Create new session or folder"
            >
              + New
            </button>
            <button
              class="px-2 py-1 text-sm bg-gray-700 hover:bg-gray-600 rounded transition-colors"
              onclick={() => showSettingsDialog = true}
              aria-label="Open settings"
              title="Settings"
            >
              ⚙️
            </button>
          </div>
        </div>

        <!-- Session Tree -->
        <div class="flex-1 overflow-y-auto">
          <SessionTree />
        </div>
      </div>
    </div>

    <!-- Resizer -->
    <div
      class="w-1 bg-gray-700 hover:bg-blue-500 cursor-col-resize transition-colors"
      onmousedown={handleMouseDown}
      onkeydown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          handleMouseDown(e as any);
        }
      }}
      role="separator"
      aria-label="Resize sidebar"
      tabindex="0"
    ></div>

    <!-- Main Content Area -->
    <div class="flex-1 flex flex-col overflow-hidden">
      <TerminalArea />
    </div>
  </div>

  <!-- New Session Dialog -->
  <NewSessionDialog show={showNewSessionDialog} onClose={() => showNewSessionDialog = false} />

  <!-- Settings Dialog -->
  <SettingsDialog show={showSettingsDialog} onClose={() => showSettingsDialog = false} />
{/if}
