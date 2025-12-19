<script lang="ts">
  import '../app.css';
  import { onMount } from 'svelte';

  // Import this for Wails runtime (will be available in built app)
  let ready = $state(false);

  onMount(() => {
    ready = true;
  });
</script>

{#if ready}
  <slot />
{:else}
  <div class="flex items-center justify-center h-screen bg-gray-900 text-white">
    <p>Loading...</p>
  </div>
{/if}
