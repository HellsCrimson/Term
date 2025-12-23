<script lang="ts">
  import './app.css';
  import { onMount } from 'svelte';
  import { init as initGhostty } from 'ghostty-web';
  import SessionTree from './lib/components/SessionTree.svelte';
  import TerminalArea from './lib/components/TerminalArea.svelte';
  import NewSessionDialog from './lib/components/NewSessionDialog.svelte';
  import SettingsDialog from './lib/components/SettingsDialog.svelte';
  import { sessionsStore } from './lib/stores/sessions.svelte';
  import { settingsStore } from './lib/stores/settings.svelte';
  import { terminalsStore } from './lib/stores/terminals.svelte';
  import { themeStore } from './lib/stores/themeStore';
  import * as LoggingService from '$bindings/term/loggingservice';
  import AlertHost from '$lib/components/common/AlertHost.svelte';
  import { Events } from '@wailsio/runtime';
  import RecordingsDialog from '$lib/components/RecordingsDialog.svelte';
  import ReplayViewer from '$lib/components/ReplayViewer.svelte';

  let sidebarWidth = $state(250);
  let resizing = $state(false);
  let ready = $state(false);
  let showNewSessionDialog = $state(false);
  let showSettingsDialog = $state(false);
  let showRecordingsDialog = $state(false);
  let showReplayViewer = $state(false);
  let currentReplayId = $state<string | null>(null);
  let lastKeyPressed = $state('');
  // SSH host key prompt state
  let showHostKeyPrompt = $state(false);
  let hostKeyPrompt: any = $state(null);

  onMount(() => {
    console.log('App mounting - loading sessions and settings');
    LoggingService.Log('App mounting - loading sessions and settings', "INFO");

    // Load sessions and settings on mount (async fire-and-forget)
    (async () => {
      // Initialize ghostty-web WASM (only once globally)
      await initGhostty();

      // Load sessions, settings, and themes
      await Promise.all([
        sessionsStore.loadSessions(),
        settingsStore.loadSettings(),
        themeStore.loadThemes()
      ]);

      console.log(`Settings loaded: restoreTabsOnStartup=${settingsStore.settings.restoreTabsOnStartup}, confirmTabClose=${settingsStore.settings.confirmTabClose}`);
      LoggingService.Log(`Settings loaded: restoreTabsOnStartup=${settingsStore.settings.restoreTabsOnStartup}, confirmTabClose=${settingsStore.settings.confirmTabClose}`, "INFO");

      // Restore tabs if enabled
      await terminalsStore.restoreTabs();

      console.log('Keyboard shortcuts registered on document');
      LoggingService.Log('Keyboard shortcuts registered on document', "INFO");

      ready = true;
    })();

    // Setup keyboard shortcuts on document
    document.addEventListener('keydown', handleKeyDown, true);

    // Open replay viewer when a replay starts emitting
    Events.On('recording:replay:header', (ev: any) => {
      LoggingService.Log('[App] replay header received, opening viewer', 'DEBUG');
      currentReplayId = ev?.data?.replayId || null;
      showReplayViewer = true;
    });

    // Also open viewer immediately when a replay is requested
    Events.On('recording:replay:start', (_ev: any) => {
      LoggingService.Log('[App] replay start received, opening viewer', 'DEBUG');
      // Stop previous replay to avoid double playback
      if (currentReplayId) {
        Events.Emit('recording:replay:stop', { replayId: currentReplayId } as any);
      }
      showReplayViewer = true;
    });

    // Listen for SSH host key verification prompts
    Events.On('ssh:hostkey_prompt', (event: any) => {
      const data = event.data || {};
      hostKeyPrompt = data;
      showHostKeyPrompt = true;
    });

    // Return cleanup function
    return () => {
      document.removeEventListener('keydown', handleKeyDown, true);
    };
  });

  function handleKeyDown(e: KeyboardEvent) {
    lastKeyPressed = `${e.ctrlKey ? 'Ctrl+' : ''}${e.shiftKey ? 'Shift+' : ''}${e.key}`;
    console.log(`KeyDown: ${lastKeyPressed}`);
    LoggingService.Log(`KeyDown: ctrl=${e.ctrlKey} shift=${e.shiftKey} key="${e.key}" code="${e.code}"`, "INFO");

    // Ctrl+T: New terminal
    if (e.ctrlKey && e.key === 't') {
      e.preventDefault();
      LoggingService.Log('Ctrl+T pressed - attempting to create new tab', "INFO");
      const selectedNode = sessionsStore.getSelectedNode();
      LoggingService.Log(`Selected node: ${selectedNode ? selectedNode.id : 'null'}`, "INFO");

      if (selectedNode && selectedNode.type === 'session' && selectedNode.sessionType) {
        LoggingService.Log(`Creating tab for selected session: ${selectedNode.name}`, "INFO");
        terminalsStore.createTab(selectedNode.id, selectedNode.name, selectedNode.sessionType);
      } else {
        // Fallback: duplicate the active tab's session
        const activeTab = terminalsStore.getActiveTab();
        if (activeTab) {
          LoggingService.Log(`No session selected, duplicating active tab: ${activeTab.sessionName}`, "INFO");
          terminalsStore.createTab(activeTab.sessionId, activeTab.sessionName, activeTab.sessionType);
        } else {
          LoggingService.Log('No valid session selected and no active tab', "INFO");
        }
      }
      return;
    }

    // Ctrl+W: Close tab
    if (e.ctrlKey && e.key === 'w') {
      e.preventDefault();
      LoggingService.Log('Ctrl+W pressed - attempting to close tab', "INFO");
      const activeTab = terminalsStore.getActiveTab();
      LoggingService.Log(`Active tab: ${activeTab ? activeTab.id : 'null'}`, "INFO");
      if (activeTab) {
        terminalsStore.closeTab(activeTab.id);
      }
      return;
    }

    // Ctrl+Tab: Next tab
    if (e.ctrlKey && e.key === 'Tab' && !e.shiftKey) {
      e.preventDefault();
      LoggingService.Log('Ctrl+Tab pressed - switching to next tab', "INFO");
      const tabs = terminalsStore.tabs;
      const activeIndex = tabs.findIndex(t => t.active);
      LoggingService.Log(`Current tab index: ${activeIndex}, total tabs: ${tabs.length}`, "INFO");
      if (activeIndex !== -1 && tabs.length > 1) {
        const nextIndex = (activeIndex + 1) % tabs.length;
        LoggingService.Log(`Switching to tab index: ${nextIndex}`, "INFO");
        terminalsStore.setActiveTab(tabs[nextIndex].id);
      }
      return;
    }

    // Ctrl+Shift+Tab: Previous tab
    if (e.ctrlKey && e.shiftKey && e.key === 'Tab') {
      e.preventDefault();
      LoggingService.Log('Ctrl+Shift+Tab pressed - switching to previous tab', "INFO");
      const tabs = terminalsStore.tabs;
      const activeIndex = tabs.findIndex(t => t.active);
      LoggingService.Log(`Current tab index: ${activeIndex}, total tabs: ${tabs.length}`, "INFO");
      if (activeIndex !== -1 && tabs.length > 1) {
        const prevIndex = (activeIndex - 1 + tabs.length) % tabs.length;
        LoggingService.Log(`Switching to tab index: ${prevIndex}`, "INFO");
        terminalsStore.setActiveTab(tabs[prevIndex].id);
      }
      return;
    }

    // Ctrl+N: New session dialog
    if (e.ctrlKey && e.key === 'n') {
      e.preventDefault();
      showNewSessionDialog = true;
      return;
    }

    // Ctrl+Plus/Equal: Increase font size
    if (e.ctrlKey && (e.key === '+' || e.key === '=')) {
      e.preventDefault();
      e.stopPropagation();
      const currentSize = settingsStore.settings.fontSize;
      if (currentSize < 32) {
        settingsStore.setFontSize(currentSize + 1);
      }
      return;
    }

    // Ctrl+Minus: Decrease font size
    if (e.ctrlKey && e.key === '-') {
      e.preventDefault();
      e.stopPropagation();
      const currentSize = settingsStore.settings.fontSize;
      if (currentSize > 8) {
        settingsStore.setFontSize(currentSize - 1);
      }
      return;
    }

    // Ctrl+0: Reset font size
    if (e.ctrlKey && e.key === '0') {
      e.preventDefault();
      e.stopPropagation();
      settingsStore.setFontSize(14);
      return;
    }

    // Ctrl+Shift+C: Copy from terminal
    if (e.ctrlKey && e.shiftKey && e.key === 'C') {
      e.preventDefault();
      e.stopPropagation();
      const activeTab = terminalsStore.getActiveTab();
      if (activeTab?.terminal) {
        const selection = activeTab.terminal.getSelection();
        if (selection) {
          navigator.clipboard.writeText(selection).catch(err =>
            console.error('Failed to copy:', err)
          );
        }
      }
      return;
    }

    // Ctrl+Shift+V: Paste to terminal
    if (e.ctrlKey && e.shiftKey && e.key === 'V') {
      e.preventDefault();
      e.stopPropagation();
      const activeTab = terminalsStore.getActiveTab();
      if (activeTab) {
        navigator.clipboard.readText().then(text => {
          terminalsStore.writeToSession(activeTab.backendSessionId, text);
        }).catch(err => console.error('Failed to paste:', err));
      }
      return;
    }
  }

  function handleMouseDown(e: MouseEvent) {
    resizing = true;
    document.addEventListener('mousemove', handleMouseMove);
    document.addEventListener('mouseup', handleMouseUp);
  }

  function handleMouseMove(e: MouseEvent) {
    if (resizing) {
      sidebarWidth = Math.max(200, Math.min(500, e.clientX));
    }
  }

  function handleMouseUp() {
    resizing = false;
    document.removeEventListener('mousemove', handleMouseMove);
    document.removeEventListener('mouseup', handleMouseUp);
  }

  async function respondToHostKeyPrompt(action: 'accept_once' | 'trust' | 'reject') {
    if (!hostKeyPrompt) return;
    const payload = { id: hostKeyPrompt.id, action };
    await Events.Emit('ssh:hostkey_response', payload);
    showHostKeyPrompt = false;
    hostKeyPrompt = null;
  }

  function hostWithPort(h: string, p: number | string) {
    const hs = String(h);
    const ps = String(p);
    return hs.endsWith(':' + ps) ? hs : `${hs}:${ps}`;
  }
</script>

{#if !ready}
  <div class="flex items-center justify-center h-screen" style="background: var(--bg-primary); color: var(--text-primary)">
    <p>Loading...</p>
  </div>
{:else}
  <div class="flex h-screen w-screen overflow-hidden" style="background: var(--bg-primary); color: var(--text-primary)">
    <!-- Sidebar -->
    <div
      class="flex-shrink-0 overflow-hidden"
      style="background: var(--bg-secondary); border-right: 1px solid var(--border-color) width: {sidebarWidth}px"
    >
      <div class="h-full flex flex-col">
        <!-- Sidebar Header -->
        <div class="px-4 py-3 flex flex-col gap-1" style="border-bottom: 1px solid var(--border-color)">
          <div class="flex items-center justify-between">
            <h2 class="text-lg font-semibold">Sessions</h2>
            <div class="flex gap-2">
              <button
                class="px-2 py-1 text-sm rounded transition-colors text-white"
                style="background: var(--accent-blue)"
                onclick={() => showNewSessionDialog = true}
                aria-label="Create new session or folder"
              >
                + New
              </button>
              <button
                class="px-2 py-1 text-sm rounded transition-colors"
                style="background: var(--bg-tertiary)"
                onclick={() => showSettingsDialog = true}
                aria-label="Open settings"
                title="Settings"
              >
                ⚙️
              </button>
              <button
                class="px-2 py-1 text-sm rounded transition-colors"
                style="background: var(--bg-tertiary)"
                onclick={() => showRecordingsDialog = true}
                aria-label="Open recordings"
                title="Recordings"
              >
                ⏺️
              </button>
            </div>
          </div>
          {#if lastKeyPressed}
            <div class="text-xs" style="color: var(--text-muted)">
              Last key: {lastKeyPressed}
  </div>
  <AlertHost />
{/if}
          <div class="text-xs" style="color: var(--text-muted)">
            Restore: {settingsStore.settings.restoreTabsOnStartup ? 'ON' : 'OFF'} |
            Confirm: {settingsStore.settings.confirmTabClose ? 'ON' : 'OFF'}
          </div>
        </div>

        <!-- Session Tree -->
        <div class="flex-1 overflow-hidden">
          <SessionTree />
        </div>
      </div>
    </div>

    <!-- Resizer -->
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
    <div
      class="w-1 cursor-col-resize transition-colors"
      style="background: var(--border-color)"
      onmousedown={handleMouseDown}
      onkeydown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          handleMouseDown(e as any);
        }
      }}
      role="separator"
      aria-label="Resize sidebar"
      tabindex="0"
    ></div>

    <!-- Main Content Area -->
    <div class="flex-1 flex flex-col overflow-hidden">
      <TerminalArea />
    </div>
  </div>

  <!-- New Session Dialog -->
  <NewSessionDialog show={showNewSessionDialog} onClose={() => showNewSessionDialog = false} />

  <!-- Settings Dialog -->
  <SettingsDialog show={showSettingsDialog} onClose={() => showSettingsDialog = false} />
  <RecordingsDialog show={showRecordingsDialog} onClose={() => showRecordingsDialog = false} />
  {#if showReplayViewer}
    <ReplayViewer show={true} onClose={() => showReplayViewer = false} replayId={currentReplayId} />
  {/if}

  {#if showHostKeyPrompt && hostKeyPrompt}
    <div class="fixed inset-0 z-[1100] flex items-center justify-center" style="background: rgba(0,0,0,0.5)">
      <div class="w-[540px] max-w-[90%] rounded shadow-lg p-4"
           style="background: var(--bg-secondary); color: var(--text-primary); border: 1px solid var(--border-color)">
        <h3 class="text-lg font-semibold mb-2">
          {hostKeyPrompt.status === 'mismatch' ? 'Host Key Changed' : 'Unknown Host' }
        </h3>
        <div class="text-sm space-y-1 mb-3">
          <div><span class="font-medium">Host:</span> {hostWithPort(hostKeyPrompt.host, hostKeyPrompt.port)}</div>
          <div><span class="font-medium">Key Type:</span> {hostKeyPrompt.keyType}</div>
          <div><span class="font-medium">Fingerprint:</span> {hostKeyPrompt.fingerprint}</div>
          {#if hostKeyPrompt.status === 'mismatch'}
            <div class="text-yellow-400"><span class="font-medium">Previous:</span> {hostKeyPrompt.oldFingerprint}</div>
          {/if}
          {#if hostKeyPrompt.status === 'mismatch'}
            <p class="text-xs mt-2" style="color: var(--text-muted)">Warning: The host key has changed. This may indicate a man-in-the-middle attack, or the host was reinstalled.</p>
          {:else}
            <p class="text-xs mt-2" style="color: var(--text-muted)">First time connecting to this host. Verify the fingerprint with the server admin.</p>
          {/if}
        </div>

        <div class="flex justify-end gap-2 pt-3" style="border-top: 1px solid var(--border-color)">
          <button class="px-3 py-1.5 rounded" style="background: var(--bg-tertiary)" onclick={() => respondToHostKeyPrompt('reject')}>Cancel</button>
          <button class="px-3 py-1.5 rounded" style="background: var(--bg-tertiary)" onclick={() => respondToHostKeyPrompt('accept_once')}>Accept Once</button>
          <button class="px-3 py-1.5 rounded text-white" style="background: var(--accent-green)" onclick={() => respondToHostKeyPrompt('trust')}>Trust and Save</button>
        </div>
      </div>
    </div>
  {/if}
{/if}
