<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Terminal, FitAddon } from 'ghostty-web';
  import type { TerminalTab } from '../stores/terminals.svelte';
  import { terminalsStore } from '../stores/terminals.svelte';
  import { sessionsStore } from '../stores/sessions.svelte';
  import { settingsStore } from '../stores/settings.svelte';
  import { themeStore } from '../stores/themeStore';
  import StatusBar from './StatusBar.svelte';
  import RemoteFileBrowser from './RemoteFileBrowser.svelte';
  import PassphraseDialog from './PassphraseDialog.svelte';
  import Modal from './common/Modal.svelte';
  import { Events } from '@wailsio/runtime';

  interface Props {
    tab: TerminalTab;
  }

  let { tab }: Props = $props();

  let terminalElement: HTMLDivElement;
  let terminal: Terminal | null = null;
  let fitAddon: FitAddon | null = null;
  let resizeObserver: ResizeObserver | null = null;
  let currentFontSize = $state(settingsStore.settings.fontSize);
  let showFileOverlay = $state(false);
  let recordActive = $state(false);
  let showPassphraseDialog = $state(false);
  let pendingRecordingOptions: any = $state(null);
  

  // Focus terminal when tab becomes active
  $effect(() => {
    if (tab.active && terminal) {
      terminal.focus();
    }
  });

  // Update font size when settings change
  $effect(() => {
    currentFontSize = settingsStore.settings.fontSize;
    if (terminal && fitAddon) {
      terminal.options.fontSize = currentFontSize;

      // Fit the terminal to the new font size
      fitAddon.fit();

      // Wait a tick for the terminal to update its dimensions, then notify backend
      setTimeout(() => {
        if (terminal) {
          terminalsStore.resizeSession(
            tab.backendSessionId,
            terminal.cols,
            terminal.rows
          );

          // Force a terminal refresh to redraw content
          // terminal.refresh(0, terminal.rows - 1);
        }
      }, 0);
    }
  });

  onMount(async () => {
    // Clear any existing content in the terminal element
    if (terminalElement) {
      terminalElement.innerHTML = '';
    }

    // Ensure any existing terminal instance is disposed
    if (terminal) {
      try {
        terminal.clear();
        terminal.reset();
        terminal.dispose();
      } catch (e) {
        // Ignore cleanup errors
      }
      terminal = null;
    }

    // Also clear any terminal reference in the tab
    if (tab.terminal) {
      try {
        tab.terminal.clear();
        tab.terminal.reset();
        tab.terminal.dispose();
      } catch (e) {
        // Ignore cleanup errors
      }
      tab.terminal = null;
    }

    // Resolve current theme colors (fallbacks in case not loaded yet)
    const liveTheme = $themeStore.previewTheme || $themeStore.activeTheme;
    const t = liveTheme?.terminal;

    // Create terminal instance
    terminal = new Terminal({
      fontFamily: settingsStore.settings.fontFamily,
      fontSize: settingsStore.settings.fontSize,
      cursorBlink: true,
      theme: {
        background: t?.background || '#1f2937',
        foreground: t?.foreground || '#f9fafb',
        cursor: t?.cursor || (t?.foreground || '#f9fafb'),
        cursorAccent: t?.background || '#1f2937',
        selectionBackground: t?.selectionBackground || 'rgba(59,130,246,0.25)',
        black: t?.black || '#111827',
        red: t?.red || '#ef4444',
        green: t?.green || '#10b981',
        yellow: t?.yellow || '#f59e0b',
        blue: t?.blue || '#3b82f6',
        magenta: t?.magenta || '#ec4899',
        cyan: t?.cyan || '#06b6d4',
        white: t?.white || '#f9fafb',
        brightBlack: t?.brightBlack || '#6b7280',
        brightRed: t?.brightRed || '#f87171',
        brightGreen: t?.brightGreen || '#34d399',
        brightYellow: t?.brightYellow || '#fbbf24',
        brightBlue: t?.brightBlue || '#60a5fa',
        brightMagenta: t?.brightMagenta || '#f472b6',
        brightCyan: t?.brightCyan || '#22d3ee',
        brightWhite: t?.brightWhite || '#ffffff'
      }
    });

    // Add fit addon
    fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);

    // Open terminal in DOM
    terminal.open(terminalElement);
    // Ensure a fully clean buffer on fresh open
    try { terminal.reset(); terminal.clear(); } catch { /* ignore */ }

    // Fit terminal to container
    fitAddon.fit();

    // Focus the terminal to receive input
    terminal.focus();

    // Store terminal reference in tab
    tab.terminal = terminal;

    // Set up resize observer
    resizeObserver = new ResizeObserver(() => {
      if (fitAddon && terminal) {
        fitAddon.fit();
        // Notify backend of new size
        terminalsStore.resizeSession(
          tab.backendSessionId,
          terminal.cols,
          terminal.rows
        );
      }
    });
    resizeObserver.observe(terminalElement);

    // Handle terminal input
    terminal.onData((data) => {
      terminalsStore.writeToSession(tab.backendSessionId, data);
    });

    // Start the backend session immediately; user can start recording via button
    if (!tab.exited) {
      try {
        const config = await sessionsStore.getEffectiveConfig(tab.sessionId);
        await terminalsStore.startSession(
          tab.backendSessionId,
          tab.sessionType,
          config,
          terminal.cols,
          terminal.rows
        );
      } catch (error) {
        console.error('Error starting session:', error);
        terminal.write(`\r\n\x1b[1;31mError starting session: ${error}\x1b[0m\r\n`);
      }
    }
  });

  onDestroy(() => {
    if (resizeObserver) {
      resizeObserver.disconnect();
    }

    if (terminal) {
      // Clear the terminal buffer before disposing
      try {
        terminal.clear();
        terminal.reset();
      } catch (e) {
        // Ignore errors during cleanup
      }
      terminal.dispose();
      terminal = null;
    }

    // Clear the terminal element
    if (terminalElement) {
      terminalElement.innerHTML = '';
    }

    // Clear the terminal reference from the tab
    if (tab.terminal) {
      tab.terminal = null;
    }
    // Clean listeners
    try { unsubStarted?.(); unsubStopped?.(); } catch {}
  });

  async function startRecording() {
    if (!terminal) return;
    const cols = terminal.cols;
    const rows = terminal.rows;
    const captureInput = settingsStore.settings.recordingDefaultCaptureInput;
    const encrypt = settingsStore.settings.recordingDefaultEncrypt;

    if (encrypt) {
      // Store recording options and show passphrase dialog
      pendingRecordingOptions = { cols, rows, captureInput, encrypt };
      showPassphraseDialog = true;
    } else {
      // Start recording without encryption
      await doStartRecording('', cols, rows, captureInput, false);
    }
  }

  async function doStartRecording(passphrase: string, cols: number, rows: number, captureInput: boolean, encrypt: boolean) {
    await Events.Emit('recording:start', {
      sessionId: tab.backendSessionId,
      sessionName: tab.sessionName,
      sessionType: tab.sessionType,
      cols,
      rows,
      captureInput,
      encrypt: encrypt && !!passphrase,
      passphrase
    } as any);
  }

  function handlePassphraseSubmit(passphrase: string) {
    if (pendingRecordingOptions) {
      const opts = pendingRecordingOptions;
      doStartRecording(passphrase, opts.cols, opts.rows, opts.captureInput, opts.encrypt);
      pendingRecordingOptions = null;
    }
  }

  function handlePassphraseCancel() {
    pendingRecordingOptions = null;
    showPassphraseDialog = false;
  }

  // React to theme changes and update the live terminal instance
  $effect(() => {
    if (!terminal) return;
    const liveTheme = $themeStore.previewTheme || $themeStore.activeTheme;
    if (!liveTheme) return;
    const t = liveTheme.terminal;
    // Update theme on the existing terminal instance
    terminal.options.theme = {
      background: t.background,
      foreground: t.foreground,
      cursor: t.cursor,
      cursorAccent: t.background,
      selectionBackground: t.selectionBackground,
      black: t.black,
      red: t.red,
      green: t.green,
      yellow: t.yellow,
      blue: t.blue,
      magenta: t.magenta,
      cyan: t.cyan,
      white: t.white,
      brightBlack: t.brightBlack,
      brightRed: t.brightRed,
      brightGreen: t.brightGreen,
      brightYellow: t.brightYellow,
      brightBlue: t.brightBlue,
      brightMagenta: t.brightMagenta,
      brightCyan: t.brightCyan,
      brightWhite: t.brightWhite
    } as any;
  });

  // Listen to recording status
  const unsubStarted = Events.On('recording:started', (ev: any) => {
    if (ev.data?.sessionId === tab.backendSessionId) recordActive = true;
  });
  const unsubStopped = Events.On('recording:stopped', (ev: any) => {
    if (ev.data?.sessionId === tab.backendSessionId) recordActive = false;
  });
</script>

<div class="terminal-wrapper h-full flex flex-col relative" style="background: var(--term-background)">
  <!-- Floating actions -->
<div class="absolute right-2 top-2 flex gap-2 z-20">
  {#if tab.sessionType === 'ssh'}
    <button
      class="px-2 py-1 text-xs rounded text-white"
      style="background: var(--accent-blue)"
      aria-label="Toggle remote files"
      onclick={() => showFileOverlay = true}
    >
      Files
    </button>
  {/if}
  {#if !recordActive}
    <button
      class="px-2 py-1 text-xs rounded text-white"
      style="background: var(--accent-blue)"
      aria-label="Start recording"
      onclick={startRecording}
    >
      Start Rec
    </button>
  {:else}
    <button
      class="px-2 py-1 text-xs rounded text-white"
      style="background: var(--accent-blue)"
      aria-label="Stop recording"
      onclick={() => Events.Emit('recording:stop', { sessionId: tab.backendSessionId } as any)}
    >
      Stop Rec
    </button>
  {/if}
</div>

  <div class="terminal-container flex-1 bg-transparent" bind:this={terminalElement}></div>

  

  <!-- Overlay for remote file browser -->
  {#if tab.sessionType === 'ssh' && showFileOverlay}
    <Modal show={showFileOverlay} title="Remote Files" onClose={() => showFileOverlay = false} panelClass="w-[80%] h-[75%] flex flex-col">
      <div class="flex-1 overflow-hidden">
        <RemoteFileBrowser {tab} />
      </div>

      {#snippet footer()}
        <div class="flex justify-end mt-4 pt-2" style="border-top: 1px solid var(--border-color)">
          <button class="px-2 py-1 text-xs rounded" style="background: var(--bg-tertiary)" onclick={() => showFileOverlay = false} aria-label="Close">Close</button>
        </div>
      {/snippet}
    </Modal>
  {/if}

  <PassphraseDialog
    show={showPassphraseDialog}
    title="Recording Encryption"
    message="Enter passphrase for recording encryption (Argon2-derived):"
    onSubmit={handlePassphraseSubmit}
    onClose={handlePassphraseCancel}
  />

  <StatusBar />
</div>
