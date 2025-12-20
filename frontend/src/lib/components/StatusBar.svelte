<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Events } from '@wailsio/runtime';
  import { settingsStore } from '../stores/settings.svelte';

  interface SystemStats {
    cpuPercent: number;
    memoryPercent: number;
    memoryUsed: number;
    memoryTotal: number;
    diskPercent: number;
    diskUsed: number;
    diskTotal: number;
    networkSent: number;
    networkRecv: number;
    loadAvg1: number;
    loadAvg5: number;
    loadAvg15: number;
  }

  let stats = $state<SystemStats>({
    cpuPercent: 0,
    memoryPercent: 0,
    memoryUsed: 0,
    memoryTotal: 0,
    diskPercent: 0,
    diskUsed: 0,
    diskTotal: 0,
    networkSent: 0,
    networkRecv: 0,
    loadAvg1: 0,
    loadAvg5: 0,
    loadAvg15: 0
  });

  let unsubscribe: (() => void) | null = null;

  onMount(() => {
    // Listen to system stats events
    unsubscribe = Events.On('system:stats', (event: any) => {
      stats = event.data;
    });
  });

  onDestroy(() => {
    if (unsubscribe) {
      unsubscribe();
    }
  });

  function formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return (bytes / Math.pow(k, i)).toFixed(1) + ' ' + sizes[i];
  }

  function formatRate(bytesPerInterval: number): string {
    // Convert to bytes per second (interval is 2 seconds)
    const bytesPerSecond = bytesPerInterval / 2;
    return formatBytes(bytesPerSecond) + '/s';
  }

  function getUsageColor(percent: number): string {
    if (percent < 50) return 'text-green-400';
    if (percent < 75) return 'text-yellow-400';
    return 'text-red-400';
  }
</script>

{#if settingsStore.settings.showStatusBar}
<div class="status-bar bg-gray-800 border-t border-gray-700 px-4 py-1.5 flex items-center gap-6 text-xs font-mono">
  <!-- CPU -->
  <div class="flex items-center gap-2">
    <span class="text-gray-400">CPU:</span>
    <span class="{getUsageColor(stats.cpuPercent)} font-semibold">
      {stats.cpuPercent.toFixed(1)}%
    </span>
    <div class="w-16 h-2 bg-gray-700 rounded-full overflow-hidden">
      <div
        class="h-full transition-all duration-300 {stats.cpuPercent < 50 ? 'bg-green-500' : stats.cpuPercent < 75 ? 'bg-yellow-500' : 'bg-red-500'}"
        style="width: {Math.min(stats.cpuPercent, 100)}%"
      ></div>
    </div>
  </div>

  <!-- Memory -->
  <div class="flex items-center gap-2">
    <span class="text-gray-400">RAM:</span>
    <span class="{getUsageColor(stats.memoryPercent)} font-semibold">
      {stats.memoryPercent.toFixed(1)}%
    </span>
    <span class="text-gray-500">
      ({formatBytes(stats.memoryUsed)} / {formatBytes(stats.memoryTotal)})
    </span>
    <div class="w-16 h-2 bg-gray-700 rounded-full overflow-hidden">
      <div
        class="h-full transition-all duration-300 {stats.memoryPercent < 50 ? 'bg-green-500' : stats.memoryPercent < 75 ? 'bg-yellow-500' : 'bg-red-500'}"
        style="width: {Math.min(stats.memoryPercent, 100)}%"
      ></div>
    </div>
  </div>

  <!-- Disk -->
  <div class="flex items-center gap-2">
    <span class="text-gray-400">Disk:</span>
    <span class="{getUsageColor(stats.diskPercent)} font-semibold">
      {stats.diskPercent.toFixed(1)}%
    </span>
    <span class="text-gray-500">
      ({formatBytes(stats.diskUsed)} / {formatBytes(stats.diskTotal)})
    </span>
    <div class="w-16 h-2 bg-gray-700 rounded-full overflow-hidden">
      <div
        class="h-full transition-all duration-300 {stats.diskPercent < 50 ? 'bg-green-500' : stats.diskPercent < 75 ? 'bg-yellow-500' : 'bg-red-500'}"
        style="width: {Math.min(stats.diskPercent, 100)}%"
      ></div>
    </div>
  </div>

  <!-- Network -->
  <div class="flex items-center gap-2">
    <span class="text-gray-400">Net:</span>
    <span class="text-blue-400">
      ↓ {formatRate(stats.networkRecv)}
    </span>
    <span class="text-purple-400">
      ↑ {formatRate(stats.networkSent)}
    </span>
  </div>

  <!-- Load Average -->
  <div class="flex items-center gap-2">
    <span class="text-gray-400">Load:</span>
    <span class="text-cyan-400">
      {stats.loadAvg1.toFixed(2)}
    </span>
    <span class="text-gray-500">
      {stats.loadAvg5.toFixed(2)}
    </span>
    <span class="text-gray-600">
      {stats.loadAvg15.toFixed(2)}
    </span>
  </div>
</div>
{/if}

<style>
  .status-bar {
    user-select: none;
  }
</style>
