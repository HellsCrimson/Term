<script lang="ts">
  import Modal from './common/Modal.svelte';

  interface Props {
    show: boolean;
    title?: string;
    message?: string;
    onSubmit: (passphrase: string) => void;
    onClose: () => void;
  }

  let { show, title = 'Enter Passphrase', message = 'Enter passphrase to decrypt recording:', onSubmit, onClose }: Props = $props();

  let passphrase = $state('');
  let showPassword = $state(false);
  let inputRef: HTMLInputElement | undefined = $state();

  // Reset and focus when dialog opens
  $effect(() => {
    if (show) {
      passphrase = '';
      showPassword = false;
      setTimeout(() => inputRef?.focus(), 100);
    }
  });

  function handleSubmit() {
    if (passphrase.trim()) {
      onSubmit(passphrase);
      passphrase = '';
      onClose();
    }
  }

  function handleCancel() {
    passphrase = '';
    onClose();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleSubmit();
    } else if (e.key === 'Escape') {
      e.preventDefault();
      handleCancel();
    }
  }
</script>

<Modal show={show} title={title} onClose={handleCancel} panelClass="w-[460px]">
  <div class="space-y-4">
    <p class="text-sm" style="color: var(--text-secondary)">{message}</p>

    <div class="relative">
      <input
        bind:this={inputRef}
        bind:value={passphrase}
        type={showPassword ? 'text' : 'password'}
        class="w-full px-3 py-2 pr-10 rounded focus:outline-none border"
        style="background: var(--bg-tertiary); border-color: var(--border-color)"
        placeholder="Enter passphrase"
        onkeydown={handleKeydown}
      />
      <button
        type="button"
        class="absolute right-2 top-1/2 -translate-y-1/2 px-2 py-1 text-xs rounded"
        style="background: var(--bg-secondary); color: var(--text-muted)"
        onclick={() => showPassword = !showPassword}
      >
        {showPassword ? '🙈' : '👁️'}
      </button>
    </div>

    <div class="text-xs" style="color: var(--text-muted)">
      💡 Tip: Use a strong passphrase to protect your recordings
    </div>
  </div>

  {#snippet footer()}
    <div class="flex gap-2 mt-6 pt-6" style="border-top: 1px solid var(--border-color)">
      <button
        onclick={handleSubmit}
        disabled={!passphrase.trim()}
        class="flex-1 px-4 py-2 rounded font-medium transition-colors text-white disabled:opacity-50"
        style="background: var(--accent-blue)"
      >
        OK
      </button>
      <button
        onclick={handleCancel}
        class="flex-1 px-4 py-2 rounded font-medium transition-colors"
        style="background: var(--bg-tertiary)"
      >
        Cancel
      </button>
    </div>
  {/snippet}
</Modal>
