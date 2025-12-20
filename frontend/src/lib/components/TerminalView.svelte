<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Terminal, FitAddon } from 'ghostty-web';
  import type { TerminalTab } from '../stores/terminals.svelte';
  import { terminalsStore } from '../stores/terminals.svelte';
  import { sessionsStore } from '../stores/sessions.svelte';
  import { settingsStore } from '../stores/settings.svelte';

  interface Props {
    tab: TerminalTab;
  }

  let { tab }: Props = $props();

  let terminalElement: HTMLDivElement;
  let terminal: Terminal | null = null;
  let fitAddon: FitAddon | null = null;
  let resizeObserver: ResizeObserver | null = null;
  let currentFontSize = $state(settingsStore.settings.fontSize);

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
          terminal.refresh(0, terminal.rows - 1);
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

    // Create terminal instance
    terminal = new Terminal({
      fontFamily: settingsStore.settings.fontFamily,
      fontSize: settingsStore.settings.fontSize,
      cursorBlink: true,
      theme: {
        background: '#1a1b26',
        foreground: '#c0caf5',
        cursor: '#c0caf5',
        cursorAccent: '#1a1b26',
        selectionBackground: 'rgba(122, 162, 247, 0.3)',
        black: '#15161e',
        red: '#f7768e',
        green: '#9ece6a',
        yellow: '#e0af68',
        blue: '#7aa2f7',
        magenta: '#bb9af7',
        cyan: '#7dcfff',
        white: '#a9b1d6',
        brightBlack: '#414868',
        brightRed: '#f7768e',
        brightGreen: '#9ece6a',
        brightYellow: '#e0af68',
        brightBlue: '#7aa2f7',
        brightMagenta: '#bb9af7',
        brightCyan: '#7dcfff',
        brightWhite: '#c0caf5'
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
</script>

<div class="terminal-wrapper h-full bg-[#1a1b26] p-4">
  <div class="terminal-container h-full" bind:this={terminalElement}></div>
</div>
