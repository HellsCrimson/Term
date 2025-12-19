<script lang="ts">
  import { onMount } from 'svelte';
  import SessionTree from '$lib/components/SessionTree.svelte';
  import TerminalArea from '$lib/components/TerminalArea.svelte';
  import { sessionsStore } from '$lib/stores/sessions.svelte';
  import { settingsStore } from '$lib/stores/settings.svelte';

  let sidebarWidth = $state(250);
  let resizing = $state(false);

  onMount(async () => {
    // Load sessions and settings on mount
    await sessionsStore.loadSessions();
    await settingsStore.loadSettings();
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
        <button
          class="px-2 py-1 text-sm bg-blue-600 hover:bg-blue-700 rounded"
          onclick={() => {/* TODO: Add new session */}}
        >
          + New
        </button>
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
    role="separator"
    tabindex="0"
  ></div>

  <!-- Main Content Area -->
  <div class="flex-1 flex flex-col overflow-hidden">
    <TerminalArea />
  </div>
</div>
