<script lang="ts">
  import Modal from './common/Modal.svelte';
  import { Events } from '@wailsio/runtime';
  import { onMount } from 'svelte';
  import * as LoggingService from '$bindings/term/loggingservice';

  interface Props { show: boolean; onClose: () => void; }
  let { show, onClose }: Props = $props();

  let items: Array<any> = $state([]);
  let unsubList: (() => void) | null = null;
  let deleting = $state<number | null>(null);

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

  async function deleteItem(id: number) {
    deleting = id;
    LoggingService.Log(`[RecordingsDialog] delete id=${id}`, 'DEBUG');
    await Events.Emit('recording:delete', { id });
    deleting = null;
  }

  function playItem(item: any) {
    const passphrase = item.encrypted ? prompt('Enter passphrase to decrypt recording:') || '' : '';
    const speed = 1.0;
    LoggingService.Log(`[RecordingsDialog] play id=${item.id} enc=${item.encrypted}`, 'DEBUG');
    Events.Emit('recording:replay:start', { id: item.id, speed, passphrase } as any);
    // The ReplayViewer will subscribe and render.
  }
</script>

<Modal show={show} title="Recordings" onClose={onClose} panelClass="w-[720px] h-[480px] flex flex-col">
  <div class="flex-1 overflow-auto">
    <table class="w-full text-sm" style="border-collapse: collapse">
      <thead>
        <tr style="background: var(--bg-tertiary)">
          <th class="text-left p-2">Name</th>
          <th class="text-left p-2">Size</th>
          <th class="text-left p-2">Encrypted</th>
          <th class="text-right p-2">Actions</th>
        </tr>
      </thead>
      <tbody>
        {#each items as item (item.id)}
          <tr style="border-top: 1px solid var(--border-color)">
            <td class="p-2">{item.sessionName}</td>
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
  </div>
  {#snippet footer()}
    <div class="flex justify-end pt-3" style="border-top: 1px solid var(--border-color)">
      <button class="px-3 py-1.5 rounded" style="background: var(--bg-tertiary)" onclick={onClose}>Close</button>
    </div>
  {/snippet}
</Modal>
