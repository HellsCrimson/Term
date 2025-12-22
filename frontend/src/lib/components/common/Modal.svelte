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
    /** Default slot */
    children?: () => any;
    /** Named footer slot */
    footer?: () => any;
  }

  let { show, title, onClose, panelClass = 'w-96', closeOnOverlay = true, children, footer }: Props = $props();

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
    tabindex="0"
    onkeydown={(e) => { if (e.key === 'Escape') { onClose(); } }}
    aria-modal="true"
  >
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <div
      bind:this={panelEl}
      class={`rounded-lg p-6 max-h-[80vh] overflow-y-auto panel ${panelClass || ''}`}
      style="background: var(--bg-secondary); border: 1px solid var(--border-color)"
      onclick={(e) => e.stopPropagation()}
      role="document"
      onkeydown={(e) => { if (e.key === 'Escape') { e.stopPropagation(); onClose(); } }}
    >
      {#if title}
        <h2 class="text-xl font-semibold mb-4">{title}</h2>
      {/if}

      {@render children?.()}

      {@render footer?.()}
    </div>
  </div>
{/if}

<style>
  :global(.modal-actions-divider) {
    border-top: 1px solid var(--border-color);
  }
</style>
