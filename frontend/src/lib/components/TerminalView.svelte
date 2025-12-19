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

  // Focus terminal when tab becomes active
  $effect(() => {
    if (tab.active && terminal) {
      terminal.focus();
    }
  });

  onMount(async () => {

    // Create terminal instance
    terminal = new Terminal({
      fontFamily: settingsStore.settings.fontFamily,
      fontSize: settingsStore.settings.fontSize,
      cursorBlink: true,
      theme: {
        background: '#000000',
        foreground: '#ffffff',
        cursor: '#ffffff',
        cursorAccent: '#000000',
        selectionBackground: 'rgba(255, 255, 255, 0.3)',
        black: '#000000',
        red: '#e06c75',
        green: '#98c379',
        yellow: '#d19a66',
        blue: '#61afef',
        magenta: '#c678dd',
        cyan: '#56b6c2',
        white: '#abb2bf',
        brightBlack: '#5c6370',
        brightRed: '#e06c75',
        brightGreen: '#98c379',
        brightYellow: '#d19a66',
        brightBlue: '#61afef',
        brightMagenta: '#c678dd',
        brightCyan: '#56b6c2',
        brightWhite: '#ffffff'
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
      terminal.dispose();
    }
  });
</script>

<div class="terminal-container h-full" bind:this={terminalElement}></div>
