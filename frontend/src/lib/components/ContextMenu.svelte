<script lang="ts">
  import { onMount } from 'svelte';

  export interface MenuItem {
    label: string;
    icon?: string;
    action: () => void;
    disabled?: boolean;
    separator?: boolean;
    danger?: boolean;
  }

  interface Props {
    show: boolean;
    x: number;
    y: number;
    items: MenuItem[];
    onClose: () => void;
  }

  let { show, x, y, items, onClose }: Props = $props();

  let menuElement: HTMLDivElement | undefined = $state();

  onMount(() => {
    function handleClickOutside(e: MouseEvent) {
      if (show && menuElement && !menuElement.contains(e.target as Node)) {
        onClose();
      }
    }

    function handleEscape(e: KeyboardEvent) {
      if (show && e.key === 'Escape') {
        onClose();
      }
    }

    document.addEventListener('click', handleClickOutside);
    document.addEventListener('keydown', handleEscape);

    return () => {
      document.removeEventListener('click', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  });

  function handleItemClick(item: MenuItem) {
    if (!item.disabled && !item.separator) {
      item.action();
      onClose();
    }
  }

  function handleKeyDown(e: KeyboardEvent, item: MenuItem) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      handleItemClick(item);
    }
  }
</script>

{#if show}
  <div
    bind:this={menuElement}
    class="fixed bg-gray-800 border border-gray-600 rounded-lg shadow-xl py-1 min-w-48 z-50"
    style="left: {x}px; top: {y}px;"
  >
    {#each items as item}
      {#if item.separator}
        <div class="h-px bg-gray-600 my-1"></div>
      {:else}
        <div
          class="px-4 py-2 hover:bg-gray-700 cursor-pointer flex items-center gap-2 {item.disabled ? 'opacity-50 cursor-not-allowed' : ''} {item.danger ? 'text-red-400 hover:bg-red-900/30' : ''}"
          onclick={() => handleItemClick(item)}
          onkeydown={(e) => handleKeyDown(e, item)}
          role="menuitem"
          tabindex={item.disabled ? -1 : 0}
        >
          {#if item.icon}
            <span class="text-base">{item.icon}</span>
          {/if}
          <span class="text-sm">{item.label}</span>
        </div>
      {/if}
    {/each}
  </div>
{/if}
