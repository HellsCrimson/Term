<script lang="ts">
  import Modal from './common/Modal.svelte';
  import PassphraseDialog from './PassphraseDialog.svelte';
  import { Events } from '@wailsio/runtime';
  import { onMount } from 'svelte';
  import * as LoggingService from '$bindings/term/loggingservice';

  interface Props { show: boolean; onClose: () => void; }
  let { show, onClose }: Props = $props();

  let items: Array<any> = $state([]);
  let unsubList: (() => void) | null = null;
  let deleting = $state<number | null>(null);
  let showPassphraseDialog = $state(false);
  let pendingPlayItem: any = $state(null);
  let searchQuery = $state('');

  // Filtered items based on search query
  let filteredItems = $derived(
    searchQuery.trim()
      ? items.filter(item =>
          item.sessionName?.toLowerCase().includes(searchQuery.toLowerCase()) ||
          item.sessionType?.toLowerCase().includes(searchQuery.toLowerCase())
        )
      : items
  );

  onMount(() => {
    unsubList = Events.On('recording:list', (ev: any) => {
      LoggingService.Log(`[RecordingsDialog] received list with ${ev?.data?.items?.length || 0} items`, 'DEBUG');
      items = ev.data?.items || [];
    });
    Events.Emit('recording:list:request');
    return () => { if (unsubList) unsubList(); };
  });

  function formatSize(n: number) {
    if (!n) return '0 B';
    const units = ['B','KB','MB','GB'];
    let i = 0; let v = n;
    while (v >= 1024 && i < units.length-1) { v /= 1024; i++; }
    return `${v.toFixed(1)} ${units[i]}`;
  }

  function formatDate(dateStr: string) {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    // If today, show time
    if (diffDays === 0) {
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }
    // If yesterday
    if (diffDays === 1) {
      return 'Yesterday ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }
    // If within a week
    if (diffDays < 7) {
      return date.toLocaleDateString([], { weekday: 'short', hour: '2-digit', minute: '2-digit' });
    }
    // Otherwise full date
    return date.toLocaleDateString([], { month: 'short', day: 'numeric', year: 'numeric' });
  }

  async function deleteItem(id: number) {
    deleting = id;
    LoggingService.Log(`[RecordingsDialog] delete id=${id}`, 'DEBUG');
    await Events.Emit('recording:delete', { id });
    deleting = null;
  }

  function playItem(item: any) {
    if (item.encrypted) {
      pendingPlayItem = item;
      showPassphraseDialog = true;
    } else {
      doPlayItem(item, '');
    }
  }

  function doPlayItem(item: any, passphrase: string) {
    const speed = 1.0;
    LoggingService.Log(`[RecordingsDialog] play id=${item.id} enc=${item.encrypted}`, 'DEBUG');
    Events.Emit('recording:replay:start', { id: item.id, speed, passphrase } as any);
    // The ReplayViewer will subscribe and render.
  }

  function handlePassphraseSubmit(passphrase: string) {
    if (pendingPlayItem) {
      doPlayItem(pendingPlayItem, passphrase);
      pendingPlayItem = null;
    }
  }

  function handlePassphraseCancel() {
    pendingPlayItem = null;
    showPassphraseDialog = false;
  }
</script>

<Modal show={show} title="Recordings" onClose={onClose} panelClass="w-[720px] h-[480px] flex flex-col">
  <!-- Search bar -->
  <div class="mb-3">
    <input
      type="text"
      bind:value={searchQuery}
      placeholder="Search recordings by name or type..."
      class="w-full px-3 py-2 rounded focus:outline-none border text-sm"
      style="background: var(--bg-tertiary); border-color: var(--border-color)"
    />
  </div>

  <div class="flex-1 overflow-auto">
    {#if filteredItems.length === 0}
      <div class="text-center py-8 text-sm" style="color: var(--text-muted)">
        {#if items.length === 0}
          No recordings found. Start recording a session to see them here.
        {:else}
          No recordings match your search query.
        {/if}
      </div>
    {:else}
      <table class="w-full text-sm" style="border-collapse: collapse">
        <thead>
          <tr style="background: var(--bg-tertiary)">
            <th class="text-left p-2">Name</th>
            <th class="text-left p-2">Date</th>
            <th class="text-left p-2">Size</th>
            <th class="text-left p-2">Encrypted</th>
            <th class="text-right p-2">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each filteredItems as item (item.id)}
            <tr style="border-top: 1px solid var(--border-color)">
              <td class="p-2">{item.sessionName}</td>
              <td class="p-2" style="color: var(--text-muted)">{formatDate(item.startedAt)}</td>
              <td class="p-2">{formatSize(item.size)}</td>
              <td class="p-2">{item.encrypted ? 'Yes' : 'No'}</td>
              <td class="p-2 text-right">
                <button class="px-2 py-1 text-xs rounded text-white" style="background: var(--accent-blue)" onclick={() => playItem(item)}>Play</button>
                <button class="ml-2 px-2 py-1 text-xs rounded" style="background: var(--bg-tertiary)" onclick={() => navigator.clipboard.writeText(item.path)}>Copy Path</button>
                <button class="ml-2 px-2 py-1 text-xs rounded disabled:opacity-60" style="background: var(--bg-tertiary)" disabled={deleting === item.id} onclick={() => deleteItem(item.id)}>Delete</button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
  {#snippet footer()}
    <div class="flex justify-end pt-3" style="border-top: 1px solid var(--border-color)">
      <button class="px-3 py-1.5 rounded" style="background: var(--bg-tertiary)" onclick={onClose}>Close</button>
    </div>
  {/snippet}
</Modal>

<PassphraseDialog
  show={showPassphraseDialog}
  onSubmit={handlePassphraseSubmit}
  onClose={handlePassphraseCancel}
/>
