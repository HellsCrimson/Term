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
    const activeTheme = $themeStore.activeTheme;
    const t = activeTheme?.terminal;

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

    // Start the backend session if not already running
    if (!tab.exited) {
      try {
        // Use sessionId (sidebar node) for config, but backendSessionId for PTY
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
  });

  // React to theme changes and update the live terminal instance
  $effect(() => {
    if (!terminal) return;
    const activeTheme = $themeStore.activeTheme;
    if (!activeTheme) return;
    const t = activeTheme.terminal;
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
</script>

<div class="terminal-wrapper h-full flex flex-col relative" style="background: var(--term-background)">
  <!-- Floating actions -->
  {#if tab.sessionType === 'ssh'}
    <div class="absolute right-2 top-2 flex gap-2 z-20">
      <button
        class="px-2 py-1 text-xs rounded text-white"
        style="background: var(--accent-blue)"
        aria-label="Toggle remote files"
        onclick={() => showFileOverlay = true}
      >
        Files
      </button>
    </div>
  {/if}

  <div class="terminal-container flex-1 bg-transparent" bind:this={terminalElement}></div>

  <!-- Overlay for remote file browser -->
  {#if tab.sessionType === 'ssh' && showFileOverlay}
    <div class="absolute inset-0 z-30 flex items-center justify-center" style="background: rgba(0,0,0,0.5)">
      <div class="rounded-lg shadow-xl overflow-hidden w-[80%] h-[75%] flex flex-col" style="background: var(--bg-secondary); border: 1px solid var(--border-color)">
        <div class="flex items-center justify-between px-3 py-2" style="border-bottom: 1px solid var(--border-color)">
          <div class="text-sm font-medium">Remote Files</div>
          <div class="flex items-center gap-2">
            <button class="px-2 py-1 text-xs rounded" style="background: var(--bg-tertiary)" onclick={() => showFileOverlay = false} aria-label="Close">Close</button>
          </div>
        </div>
        <div class="flex-1 overflow-hidden">
          <RemoteFileBrowser {tab} />
        </div>
      </div>
    </div>
  {/if}

  <StatusBar />
</div>
