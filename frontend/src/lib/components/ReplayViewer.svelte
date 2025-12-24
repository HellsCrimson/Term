<script lang="ts">
  import Modal from './common/Modal.svelte';
  import { Events } from '@wailsio/runtime';
  import { Terminal, FitAddon } from 'ghostty-web';
  import * as LoggingService from '$bindings/term/loggingservice';
  import { onMount, onDestroy } from 'svelte';
  import { themeStore } from '../stores/themeStore';
  import { settingsStore } from '../stores/settings.svelte';

  interface Props {
    show: boolean;
    onClose: () => void;
    replayId?: string | null;
  }

  let {
    show,
    onClose,
    replayId: replayIdProp
  }: Props = $props();

  let terminalEl: HTMLDivElement;
  let terminal: Terminal | null = null;
  let fitAddon: FitAddon | null = null;
  let replayId: string | null = null;
  let speed = $state(1.0);
  let pending: string[] = [];
  let unsubHeader: (() => void) | null = null;
  let unsubOutput: (() => void) | null = null;
  let unsubResize: (() => void) | null = null;
  let unsubEnded: (() => void) | null = null;
  let resizeObserver: ResizeObserver | null = null;
  let playing = $state(true);
  let elapsedNs = $state(0);
  let totalNs = $state(0);

  onMount(() => {
    const liveTheme = $themeStore.previewTheme || $themeStore.activeTheme;
    const t = liveTheme?.terminal;
    LoggingService.Log('[ReplayViewer] mounting; creating terminal', 'DEBUG');
    terminal = new Terminal({
      fontFamily: settingsStore.settings.fontFamily,
      fontSize: settingsStore.settings.fontSize,
      cursorBlink: true,
      theme: t ? {
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
      } : undefined as any
    });
    fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.open(terminalEl);
    try {
      terminal.reset();
      terminal.clear();
    } catch {}
    fitAddon.fit();
    terminal.write('\x1b[32mLoading replay...\x1b[0m\r\n');
    // Flush any buffered output that came in before terminal was ready
    if (pending.length > 0) {
      try {
        terminal.write(pending.join(''));
      } catch {}
      pending = [];
    }

    // Observe container size and refit
    resizeObserver = new ResizeObserver(() => {
      try { fitAddon?.fit(); } catch {}
    });
    resizeObserver.observe(terminalEl);

    unsubHeader = Events.On('recording:replay:header', (ev: any) => {
      LoggingService.Log('[ReplayViewer] header received', 'DEBUG');
      if (!replayId && ev.data?.replayId) replayId = ev.data.replayId;
      // Adjust size on header
      try { fitAddon?.fit(); } catch {}
    });
    Events.On('recording:replay:meta', (ev: any) => {
      if (replayId && ev.data?.replayId && ev.data.replayId !== replayId) return;
      totalNs = ev.data?.totalNs || 0;
      playing = true;
    });
    Events.On('recording:replay:progress', (ev: any) => {
      if (!replayId && ev.data?.replayId) replayId = ev.data.replayId;
      if (replayId && ev.data?.replayId !== replayId) return;
      elapsedNs = ev.data?.elapsedNs || 0;
      totalNs = ev.data?.totalNs || totalNs;
    });
    unsubOutput = Events.On('recording:replay:output', (ev: any) => {
      LoggingService.Log(`[ReplayViewer] output event ${ev?.data?.data?.length || 0} bytes`, 'DEBUG');
      if (!replayId && ev.data?.replayId) replayId = ev.data.replayId;
      if (replayId && ev.data?.replayId !== replayId) return;
      const text = ev.data?.data || '';
      if (terminal) {
        try { terminal.write(text); } catch {}
      } else {
        pending.push(text);
      }
    });
    unsubResize = Events.On('recording:replay:resize', (ev: any) => {
      LoggingService.Log('[ReplayViewer] resize event', 'DEBUG');
      if (!replayId && ev.data?.replayId) replayId = ev.data.replayId;
      if (replayId && ev.data?.replayId !== replayId) return;
      // Optionally adjust terminal if needed
      fitAddon?.fit();
    });
    unsubEnded = Events.On('recording:replay:ended', (ev: any) => {
      LoggingService.Log('[ReplayViewer] ended', 'DEBUG');
      if (!replayId && ev.data?.replayId) replayId = ev.data.replayId;
      if (replayId && ev.data?.replayId !== replayId) return;
      // Nothing special for now
      playing = false;
    });
  });

  // Update theme live
  $effect(() => {
    if (!terminal) return;
    const liveTheme = $themeStore.previewTheme || $themeStore.activeTheme;
    if (!liveTheme) return;
    const t = liveTheme.terminal;
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

  // Update font settings live
  $effect(() => {
    if (!terminal) return;
    // Family
    terminal.options.fontFamily = settingsStore.settings.fontFamily;
    // Size and refit
    terminal.options.fontSize = settingsStore.settings.fontSize;
    try { fitAddon?.fit(); } catch {}
  });

  $effect(() => {
    if (replayIdProp && replayIdProp !== replayId) {
      // Switch to new replay id: clear terminal to avoid mixing outputs
      replayId = replayIdProp;
      if (terminal) {
        try { terminal.reset(); terminal.clear(); } catch {}
        try { terminal.write('\x1b[32mLoading replay...\x1b[0m\r\n'); } catch {}
      }
      pending = [];
    }
  });

  onDestroy(() => {
    if (resizeObserver) { try { resizeObserver.disconnect(); } catch {} }
    if (replayId) Events.Emit('recording:replay:stop', { replayId });
    unsubHeader && unsubHeader();
    unsubOutput && unsubOutput();
    unsubResize && unsubResize();
    unsubEnded && unsubEnded();
    if (terminal) { try { terminal.dispose(); } catch {}
      terminal = null; }
  });

  function close() {
    if (replayId) Events.Emit('recording:replay:stop', { replayId });
    onClose();
  }

  function fmtTime(ns: number): string {
    const s = Math.floor(ns / 1e9);
    const m = Math.floor(s / 60);
    const ss = s % 60;
    return `${m}:${ss.toString().padStart(2, '0')}`;
  }

  function onPlayPause() {
    if (!replayId) return;
    if (playing) {
      Events.Emit('recording:replay:pause', { replayId });
      playing = false;
    } else {
      Events.Emit('recording:replay:resume', { replayId });
      playing = true;
    }
  }

  function onRewind() {
    if (!replayId) return;
    Events.Emit('recording:replay:rewind', { replayId });
    playing = true;
    elapsedNs = 0;
  }

  function onSeek(event: MouseEvent) {
    if (!replayId || !totalNs) return;
    const target = event.currentTarget as HTMLElement;
    const rect = target.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const percent = Math.max(0, Math.min(1, x / rect.width));
    const targetNs = Math.floor(percent * totalNs);

    LoggingService.Log(`[ReplayViewer] seek to ${percent * 100}% (${targetNs}ns / ${totalNs}ns)`, 'DEBUG');
    Events.Emit('recording:replay:seek', { replayId, targetNs } as any);
    elapsedNs = targetNs;
  }

  $effect(() => {
    if (!replayId && replayIdProp) replayId = replayIdProp;
  });
</script>

<Modal show={show} title="Replay Viewer" onClose={close} panelClass="w-[80%] h-[75%] flex flex-col">
  <div class="flex items-center gap-3 p-2" style="border-bottom: 1px solid var(--border-color)">
    <button class="px-2 py-1 text-xs rounded text-white" style="background: var(--accent-blue)" onclick={onPlayPause}>
      {playing ? 'Pause' : 'Play'}
    </button>
    <button class="px-2 py-1 text-xs rounded" style="background: var(--bg-tertiary)" onclick={onRewind}>
      Rewind
    </button>
    <div
      class="flex-1 h-2 rounded overflow-hidden cursor-pointer"
      style="background: var(--bg-tertiary)"
      onclick={onSeek}
      role="slider"
      aria-label="Seek timeline"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={totalNs ? Math.min(100, Math.floor((elapsedNs/Math.max(1,totalNs))*100)) : 0}
      tabindex="0"
    >
      <div style={`width: ${totalNs ? Math.min(100, Math.floor((elapsedNs/Math.max(1,totalNs))*100)) : 0}%; height: 100%; background: var(--accent-blue); transition: width 0.1s ease`}></div>
    </div>
    <div class="text-xs" style="color: var(--text-muted)">{fmtTime(elapsedNs)} / {fmtTime(totalNs)}</div>
    <label for="speed_selector" class="text-sm">Speed</label>
    <select id="speed_selector" bind:value={speed} class="px-2 py-1 rounded border" style="background: var(--bg-tertiary); border-color: var(--border-color)"
            onchange={() => replayId && Events.Emit('recording:replay:setSpeed', { replayId, speed } as any)}>
      <option value={0.5}>0.5x</option>
      <option value={1.0}>1x</option>
      <option value={2.0}>2x</option>
      <option value={4.0}>4x</option>
    </select>
  </div>
  <div class="flex-1 overflow-hidden" style="min-height: 0">
    <div class="h-full" style="width: 100%" bind:this={terminalEl}></div>
  </div>
  {#snippet footer()}
    <div class="flex justify-end p-2" style="border-top: 1px solid var(--border-color)">
      <button class="px-3 py-1.5 rounded" style="background: var(--bg-tertiary)" onclick={close}>Close</button>
    </div>
  {/snippet}
</Modal>
