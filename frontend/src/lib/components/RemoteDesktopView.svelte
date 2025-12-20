<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import Guacamole from 'guacamole-common-js';
  import type { TerminalTab } from '../stores/terminals.svelte';
  import { terminalsStore } from '../stores/terminals.svelte';
  import { sessionsStore } from '../stores/sessions.svelte';
  import StatusBar from './StatusBar.svelte';

  interface Props {
    tab: TerminalTab;
  }

  let { tab }: Props = $props();

  let displayElement: HTMLDivElement;
  let client: any = null;
  let resizeObserver: ResizeObserver | null = null;

  // Focus desktop when tab becomes active
  $effect(() => {
    if (tab.active && client) {
      client.focus();
    }
  });

  onMount(async () => {
    // Clear any existing content
    if (displayElement) {
      displayElement.innerHTML = '';
    }

    try {
      // Get session configuration
      const config = await sessionsStore.getEffectiveConfig(tab.sessionId);

      // Create WebSocket tunnel URL
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = `${protocol}//${window.location.host}/api/guacamole/${tab.backendSessionId}`;

      // Create Guacamole tunnel
      const tunnel = new Guacamole.WebSocketTunnel(wsUrl);

      // Create Guacamole client
      client = new Guacamole.Client(tunnel);

      // Get display element from client
      const display = client.getDisplay();

      // Add display to DOM
      displayElement.appendChild(display.getElement());

      // Set display scale mode
      display.scale(1.0);

      // Error handler
      client.onerror = (error: any) => {
        console.error('Guacamole client error:', error);
        if (displayElement) {
          const errorDiv = document.createElement('div');
          errorDiv.className = 'flex items-center justify-center h-full text-red-400';
          errorDiv.textContent = `Connection error: ${error.message || 'Unknown error'}`;
          displayElement.innerHTML = '';
          displayElement.appendChild(errorDiv);
        }
      };

      // State change handler
      client.onstatechange = (state: number) => {
        console.log('Guacamole state changed:', state);

        if (state === 3) { // CONNECTED
          console.log('Guacamole client connected');
        } else if (state === 5) { // DISCONNECTED
          console.log('Guacamole client disconnected');
          if (!tab.exited) {
            tab.exited = true;
            tab.exitCode = 0;
          }
        }
      };

      // Name handler (for window title)
      client.onname = (name: string) => {
        console.log('Remote desktop name:', name);
      };

      // Clipboard handler
      client.onclipboard = (stream: any, mimetype: string) => {
        console.log('Clipboard received:', mimetype);
        // Handle clipboard data
        if (mimetype === 'text/plain') {
          const reader = new Guacamole.StringReader(stream);
          let text = '';

          reader.ontext = (data: string) => {
            text += data;
          };

          reader.onend = () => {
            // Copy to system clipboard
            navigator.clipboard.writeText(text).catch(err => {
              console.error('Failed to write to clipboard:', err);
            });
          };
        }
      };

      // Mouse handling
      const mouse = new Guacamole.Mouse(display.getElement());

      // mouse.onmousedown =
      // mouse.onmouseup =
      // mouse.onmousemove = (mouseState: any) => {
      //   client.sendMouseState(mouseState);
      // };

      // Keyboard handling
      const keyboard = new Guacamole.Keyboard(document);

      keyboard.onkeydown = (keysym: number) => {
        client.sendKeyEvent(1, keysym);
      };

      keyboard.onkeyup = (keysym: number) => {
        client.sendKeyEvent(0, keysym);
      };

      // Connect to the server with configuration
      const connectionParams = buildConnectionParams(config, tab.sessionType);
      client.connect(connectionParams);

      // Set up resize observer to scale display
      resizeObserver = new ResizeObserver(() => {
        if (client && displayElement) {
          const displayDiv = display.getElement();
          const containerWidth = displayElement.clientWidth;
          const containerHeight = displayElement.clientHeight;
          const displayWidth = displayDiv.offsetWidth;
          const displayHeight = displayDiv.offsetHeight;

          if (displayWidth > 0 && displayHeight > 0) {
            const scaleX = containerWidth / displayWidth;
            const scaleY = containerHeight / displayHeight;
            const scale = Math.min(scaleX, scaleY, 1.0);
            display.scale(scale);
          }
        }
      });
      resizeObserver.observe(displayElement);

    } catch (error) {
      console.error('Failed to create Guacamole client:', error);
      if (displayElement) {
        const errorDiv = document.createElement('div');
        errorDiv.className = 'flex items-center justify-center h-full text-red-400';
        errorDiv.textContent = `Failed to initialize: ${error}`;
        displayElement.appendChild(errorDiv);
      }
    }
  });

  onDestroy(() => {
    if (resizeObserver) {
      resizeObserver.disconnect();
    }

    if (client) {
      try {
        client.disconnect();
      } catch (e) {
        console.error('Error disconnecting Guacamole client:', e);
      }
      client = null;
    }

    // Clear the display element
    if (displayElement) {
      displayElement.innerHTML = '';
    }
  });

  function buildConnectionParams(config: Record<string, string>, sessionType: string): string {
    const params: Record<string, string> = {};

    if (sessionType === 'rdp') {
      params.hostname = config.rdp_host || '';
      params.port = config.rdp_port || '3389';
      params.username = config.rdp_username || '';
      params.password = config.rdp_password || '';
      params.domain = config.rdp_domain || '';
      params.security = config.rdp_security || 'any';
      params['ignore-cert'] = 'true';
      params.width = config.desktop_width || '1920';
      params.height = config.desktop_height || '1080';
      params['color-depth'] = config.desktop_color_depth || '16';
      params['enable-wallpaper'] = 'false';
      params['enable-theming'] = 'false';
      params['enable-font-smoothing'] = 'false';
      params['enable-full-window-drag'] = 'false';
      params['enable-desktop-composition'] = 'false';
      params['enable-menu-animations'] = 'false';
    } else if (sessionType === 'vnc') {
      params.hostname = config.vnc_host || '';
      params.port = config.vnc_port || '5900';
      params.password = config.vnc_password || '';
      params.width = config.desktop_width || '1920';
      params.height = config.desktop_height || '1080';
      params['color-depth'] = config.desktop_color_depth || '16';
    } else if (sessionType === 'telnet') {
      params.hostname = config.telnet_host || '';
      params.port = config.telnet_port || '23';
      params.username = config.telnet_username || '';
      params.password = config.telnet_password || '';
    }

    // Convert to Guacamole connection string format
    return Object.entries(params)
      .map(([key, value]) => `${key}=${encodeURIComponent(value)}`)
      .join('&');
  }
</script>

<div class="desktop-wrapper h-full bg-[#1a1b26] flex flex-col overflow-hidden">
  <div class="desktop-container flex-1 flex items-center justify-center" bind:this={displayElement}></div>
  <StatusBar />
</div>

<style>
  .desktop-wrapper {
    position: relative;
  }

  .desktop-container {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  :global(.desktop-container canvas) {
    max-width: 100%;
    max-height: 100%;
  }
</style>
