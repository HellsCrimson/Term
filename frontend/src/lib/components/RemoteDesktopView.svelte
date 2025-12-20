<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import Guacamole from 'guacamole-common-js';
  import type { TerminalTab } from '../stores/terminals.svelte';
  import { terminalsStore } from '../stores/terminals.svelte';
  import { sessionsStore } from '../stores/sessions.svelte';
  import { LoggingService } from '$bindings/term';
  import StatusBar from './StatusBar.svelte';

  interface Props {
    tab: TerminalTab;
  }

  let { tab }: Props = $props();

  let displayElement: HTMLDivElement;
  let client: any = null;
  let resizeObserver: ResizeObserver | null = null;
  let lastContainerWidth = 0;
  let lastContainerHeight = 0;
  let isResizing = false;
  let currentScale = 1.0;
  let mouse: any = null;

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
      // Use sessionId (sidebar node) for configuration lookup
      const wsUrl = `ws://localhost:3000/api/guacamole/${tab.sessionId}`;

      // Create Guacamole tunnel
      const tunnel = new Guacamole.WebSocketTunnel(wsUrl);

      // Create Guacamole client
      client = new Guacamole.Client(tunnel);

      // Get display element from client
      const display = client.getDisplay();

      // Add display to DOM
      displayElement.appendChild(display.getElement());

      // Set initial display scale
      display.scale(1.0);
      currentScale = 1.0;

      // Initialize container size tracking
      lastContainerWidth = displayElement.clientWidth;
      lastContainerHeight = displayElement.clientHeight;

      // Error handler
      client.onerror = (error: any) => {
        LoggingService.Log(`Guacamole client error: ${error.message || error}`, "ERROR");
        if (displayElement) {
          const errorDiv = document.createElement('div');
          errorDiv.className = 'flex items-center justify-center h-full text-red-400';
          errorDiv.textContent = `Connection error: ${error.message || 'Unknown error'}`;
          displayElement.innerHTML = '';
          displayElement.appendChild(errorDiv);
        }
      };

      // Debounce resize handling to prevent flickering
      let resizeTimeout: number | null = null;
      const handleResize = () => {
        if (resizeTimeout) {
          clearTimeout(resizeTimeout);
        }
        resizeTimeout = setTimeout(() => {
          // Prevent re-entrant resize calls
          if (isResizing) {
            resizeTimeout = null;
            return;
          }

          if (client && displayElement) {
            const displayDiv = display.getElement();
            const containerWidth = displayElement.clientWidth;
            const containerHeight = displayElement.clientHeight;

            // Only resize if container size changed by more than 5 pixels to prevent feedback loop
            const widthDiff = Math.abs(containerWidth - lastContainerWidth);
            const heightDiff = Math.abs(containerHeight - lastContainerHeight);

            if (widthDiff > 5 || heightDiff > 5) {
              LoggingService.Log(`Handling resize event for Guacamole display: ${containerHeight}, ${containerWidth} (diff: ${heightDiff}, ${widthDiff})`, "DEBUG");

              isResizing = true;
              lastContainerWidth = containerWidth;
              lastContainerHeight = containerHeight;

              const displayWidth = displayDiv.offsetWidth;
              const displayHeight = displayDiv.offsetHeight;

              if (displayWidth > 0 && displayHeight > 0) {
                const scaleX = containerWidth / displayWidth;
                const scaleY = containerHeight / displayHeight;
                const scale = Math.min(scaleX, scaleY, 1.0);
                display.scale(scale);
                currentScale = scale;
                LoggingService.Log(`Applied scale: ${scale}`, "DEBUG");
              }

              // Clear the resizing flag after a delay to allow the DOM to settle
              setTimeout(() => {
                isResizing = false;
              }, 200);
            }
          }
          resizeTimeout = null;
        }, 100); // 100ms debounce
      };

      // State change handler
      client.onstatechange = (state: number) => {
        LoggingService.Log(`Guacamole state changed: ${state}`, "DEBUG");

        if (state === 3) { // CONNECTED
          LoggingService.Log('Guacamole client connected', "INFO");
        } else if (state === 5) { // DISCONNECTED
          LoggingService.Log('Guacamole client disconnected', "INFO");
          if (!tab.exited) {
            tab.exited = true;
            tab.exitCode = 0;
          }
        }
      };

      // Name handler (for window title)
      client.onname = (name: string) => {
        LoggingService.Log(`Remote desktop name: ${name}`, "DEBUG");
      };

      // Clipboard handler
      client.onclipboard = (stream: any, mimetype: string) => {
        LoggingService.Log(`Clipboard received: ${mimetype}`, "DEBUG");
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
              LoggingService.Log(`Failed to write to clipboard: ${err}`, "ERROR");
            });
          };
        }
      };

      // Mouse handling
      mouse = new Guacamole.Mouse(display.getElement());

      mouse.onmousedown = (mouseState: any) => {
        // Create a scaled copy of mouse state to avoid modifying the original
        const scaledState = {
          ...mouseState,
          x: mouseState.x / currentScale,
          y: mouseState.y / currentScale
        };
        client.sendMouseState(scaledState);
      };

      mouse.onmouseup = (mouseState: any) => {
        // Create a scaled copy of mouse state to avoid modifying the original
        const scaledState = {
          ...mouseState,
          x: mouseState.x / currentScale,
          y: mouseState.y / currentScale
        };
        client.sendMouseState(scaledState);
      };

      mouse.onmousemove = (mouseState: any) => {
        // Create a scaled copy of mouse state to avoid modifying the original
        const scaledState = {
          ...mouseState,
          x: mouseState.x / currentScale,
          y: mouseState.y / currentScale
        };
        client.sendMouseState(scaledState);
      };

      // Keyboard handling
      const keyboard = new Guacamole.Keyboard(document);

      keyboard.onkeydown = (keysym: number) => {
        client.sendKeyEvent(1, keysym);
      };

      keyboard.onkeyup = (keysym: number) => {
        client.sendKeyEvent(0, keysym);
      };

      // Set up resize observer to scale display
      resizeObserver = new ResizeObserver(handleResize);
      resizeObserver.observe(displayElement);

      // Connect to the server with configuration
      const connectionParams = buildConnectionParams(config, tab.sessionType);
      client.connect(connectionParams);

    } catch (error) {
      LoggingService.Log(`Failed to create Guacamole client: ${error}`, "ERROR");
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
        LoggingService.Log(`Error disconnecting Guacamole client: ${e}`, "ERROR");
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
    position: relative;
  }

  :global(.desktop-container canvas) {
    max-width: 100%;
    max-height: 100%;
    object-fit: contain;
  }
</style>
