<script lang="ts">
  import { onMount } from 'svelte';

  interface Props {
    show: boolean;
    title?: string;
    onClose: () => void;
    /** Optional extra classes to control size (e.g., w-96, w-[560px], w-[80%] h-[75%]) */
    panelClass?: string;
    /** If true, clicking outside the panel closes the modal */
    closeOnOverlay?: boolean;
  }

  let { show, title, onClose, panelClass = 'w-96', closeOnOverlay = true }: Props = $props();

  let panelEl: HTMLDivElement | null = $state(null);

  onMount(() => {
    function onKey(e: KeyboardEvent) {
      if (!show) return;
      if (e.key === 'Escape') {
        onClose();
      }
    }
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  });

  function handleOverlayClick(e: MouseEvent) {
    if (!closeOnOverlay) return;
    if (!panelEl) return;
    if (!panelEl.contains(e.target as Node)) {
      onClose();
    }
  }
</script>

{#if show}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center"
    style="background: rgba(0,0,0,0.5)"
    onclick={handleOverlayClick}
    role="dialog"
    aria-modal="true"
  >
    <div
      bind:this={panelEl}
      class={`rounded-lg p-6 max-h-[80vh] overflow-y-auto panel ${panelClass || ''}`}
      style="background: var(--bg-secondary); border: 1px solid var(--border-color)"
      onclick={(e) => e.stopPropagation()}
    >
      {#if title}
        <h2 class="text-xl font-semibold mb-4">{title}</h2>
      {/if}

      <slot />

      <slot name="footer" />
    </div>
  </div>
{/if}

<style>
  .panel {
    /* utility class hook if needed */
  }
  :global(.modal-actions-divider) {
    border-top: 1px solid var(--border-color);
  }
</style>
