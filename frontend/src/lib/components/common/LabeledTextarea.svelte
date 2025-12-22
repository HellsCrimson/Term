<script lang="ts">
  interface Props {
    id: string;
    label: string;
    value: string;
    rows?: number;
    placeholder?: string;
    hint?: string;
    inherited?: string;
  }

  let {
    id,
    label,
    value = $bindable(""),
    rows = 2,
    placeholder = '',
    hint = '',
    inherited = ''
  }: Props = $props();
</script>

<div>
  <label for={id} class="block text-xs font-medium mb-1">{label}</label>
  <textarea
    id={id}
    bind:value
    rows={rows}
    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500 font-mono"
    placeholder={placeholder}
  ></textarea>
  {#if inherited && !value}
    <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inherited}</p>
  {:else if hint}
    <p class="text-xs text-gray-500 mt-1">{hint}</p>
  {/if}
</div>
