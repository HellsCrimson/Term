<script lang="ts">
  export interface Option { value: string; label: string }

  interface Props {
    id: string;
    label: string;
    value: string;
    options: Option[];
    placeholder?: string;
    inherited?: string;
    hint?: string;
  }

  let {
    id,
    label,
    value = $bindable(''),
    options,
    placeholder = '',
    inherited = '',
    hint = ''
  }: Props = $props();
</script>

<div>
  <label for={id} class="block text-xs font-medium mb-1">{label}</label>
  <select
    id={id}
    bind:value
    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
  >
    {#if placeholder}
      <option value="" disabled selected={value === ''}>{placeholder}</option>
    {/if}
    {#each options as opt}
      <option value={opt.value}>{opt.label}</option>
    {/each}
  </select>
  {#if inherited && !value}
    <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inherited}</p>
  {:else if hint}
    <p class="text-xs text-gray-500 mt-1">{hint}</p>
  {/if}
</div>
