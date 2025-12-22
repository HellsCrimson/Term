<script lang="ts">
  interface Props {
    id: string;
    label: string;
    value: string;
    type?: string;
    placeholder?: string;
    inherited?: string;
    hint?: string;
    inputClass?: string;
    extra?: () => any;
  }

  let {
    id,
    label,
    value = $bindable(''),
    type = 'text',
    placeholder = '',
    inherited = '',
    hint = '',
    inputClass = '',
    extra
  }: Props = $props();
</script>

<div>
  <label for={id} class="block text-xs font-medium mb-1">{label}</label>
  <input
    id={id}
    type={type}
    bind:value
    class={`w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500 ${inputClass}`}
    placeholder={placeholder}
  />
  {#if inherited && !value}
    <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inherited}</p>
  {/if}
  {#if hint}
    <p class="text-xs text-gray-500 mt-1">{hint}</p>
  {/if}
  {@render extra?.()}
  
</div>
