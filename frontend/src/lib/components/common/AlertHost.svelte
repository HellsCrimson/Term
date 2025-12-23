<script lang="ts">
  import Modal from './Modal.svelte';
  import { alertsStore } from '$lib/stores/alerts.svelte';
  import { derived } from 'svelte/store';

  const current = derived(alertsStore, ($q) => $q[0]);

  function handleClose() {
    // For alert, closing is equivalent to OK; for confirm, treat as cancel
    alertsStore.dismissCurrent(false);
  }

  function handleOK() {
    alertsStore.dismissCurrent(true);
  }
</script>

{#if $current}
  <Modal show={true} title={$current.title} onClose={handleClose} panelClass="w-[420px]" zIndex={1000}>
    <div class="text-sm" style="color: var(--text-primary)">{$current.message}</div>

    {#snippet footer()}
      <div class="flex justify-end gap-2 pt-4" style="border-top: 1px solid var(--border-color)">
        {#if $current.type === 'confirm'}
          <button class="px-3 py-1.5 rounded" style="background: var(--bg-tertiary)" onclick={handleClose}>Cancel</button>
          <button class="px-3 py-1.5 rounded text-white" style="background: var(--accent-blue)" onclick={handleOK}>Confirm</button>
        {:else}
          <button class="px-3 py-1.5 rounded text-white" style="background: var(--accent-blue)" onclick={handleOK}>OK</button>
        {/if}
      </div>
    {/snippet}
  </Modal>
{/if}
