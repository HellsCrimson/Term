<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  export interface TabItem { id: string; label: string }

  interface Props {
    items: TabItem[];
    active: string;
    after?: () => any;
  }

  let {
    items,
    active,
    after
  }: Props = $props();
  const dispatch = createEventDispatcher<{ change: string }>();

  function setActive(id: string) {
    if (active === id) return;
    active = id;
    dispatch('change', id);
  }

  function classes(id: string): string {
    const selected = id === active;
    return `px-4 py-2 text-sm font-medium transition-colors ${selected ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-gray-300'}`;
  }
</script>

<div class="flex border-b border-gray-600" role="tablist">
  {#each items as item}
    <button
      type="button"
      class={classes(item.id)}
      aria-selected={item.id === active}
      onclick={() => setActive(item.id)}
      onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); setActive(item.id); } }}
      role="tab"
      tabindex="0"
    >
      {item.label}
    </button>
  {/each}
  {@render after?.()}
</div>
