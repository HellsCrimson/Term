<script lang="ts">
  import { settingsStore } from '../stores/settings.svelte';
  import { themeStore } from '../stores/themeStore';
  import { Browser, Dialogs } from '@wailsio/runtime'
  import { alertsStore } from '$lib/stores/alerts.svelte';
  import Modal from './common/Modal.svelte';
  import ToggleSwitch from './common/ToggleSwitch.svelte';
  import { Events } from '@wailsio/runtime';

  interface Props {
    show: boolean;
    onClose: () => void;
  }

  let { show, onClose }: Props = $props();

  // Local state for settings
  let theme = $state(settingsStore.settings.theme);
  let selectedThemeId = $state($themeStore.activeTheme?.id || 'dark');
  let fontFamily = $state(settingsStore.settings.fontFamily);
  let fontSize = $state(settingsStore.settings.fontSize);
  let autoLaunch = $state(settingsStore.settings.autoLaunch);
  let restoreTabsOnStartup = $state(settingsStore.settings.restoreTabsOnStartup);
  let confirmTabClose = $state(settingsStore.settings.confirmTabClose);
  let showStatusBar = $state(settingsStore.settings.showStatusBar);
  let saving = $state(false);
  // Theme import/export helpers
  let importPath = $state('');
  let exportPath = $state('');
  let exporting = $state(false);

  // Known hosts management state
  let knownHosts: Array<any> = $state([]);
  let knownHostsLoaded = $state(false);
  let knownHostsUnsub: (() => void) | null = null;

  // Tabs state
  type TabKey = 'appearance' | 'typography' | 'behavior' | 'security';
  let activeTab: TabKey = $state('appearance');

  // Live theme preview helper
  function previewSelectedTheme() {
    const store = $themeStore;
    const themes = store.themes || [];
    const preview = themes.find(t => t.id === selectedThemeId) || null;
    themeStore.setPreviewTheme(preview);
  }

  // Update local state when settings change (but not while saving)
  // Do NOT overwrite selectedThemeId here; the user is interacting with it.
  $effect(() => {
    if (!saving) {
      theme = settingsStore.settings.theme;
      fontFamily = settingsStore.settings.fontFamily;
      fontSize = settingsStore.settings.fontSize;
      autoLaunch = settingsStore.settings.autoLaunch;
      restoreTabsOnStartup = settingsStore.settings.restoreTabsOnStartup;
      confirmTabClose = settingsStore.settings.confirmTabClose;
      showStatusBar = settingsStore.settings.showStatusBar;
    }
  });

  // Initialize the selection from the active theme when the dialog opens
  let initializedSelection = $state(false);
  $effect(() => {
    if (show && !initializedSelection) {
      selectedThemeId = $themeStore.activeTheme?.id || 'dark';
      initializedSelection = true;
    }
    if (!show && initializedSelection) {
      initializedSelection = false;
    }
  });

  // Load known hosts when dialog opens
  $effect(() => {
    if (show) {
      // Subscribe once per open
      knownHostsUnsub = Events.On('ssh:known_hosts:list', (ev: any) => {
        const items = ev.data?.items || [];
        knownHosts = items;
        knownHostsLoaded = true;
      });
      Events.Emit('ssh:known_hosts:list:request');
    } else {
      if (knownHostsUnsub) {
        knownHostsUnsub();
        knownHostsUnsub = null;
      }
    }
  });

  async function deleteKnownHost(item: any) {
    await Events.Emit('ssh:known_hosts:delete', { id: item.id });
  }

  function hostWithPort(h: string, p: number | string) {
    const hs = String(h);
    const ps = String(p);
    return hs.endsWith(':' + ps) ? hs : `${hs}:${ps}`;
  }

  async function handleSave() {
    saving = true;
    try {
      console.log('=== DIALOG handleSave START ===');
      console.log(`Local values: theme=${theme}, fontFamily=${fontFamily}, fontSize=${fontSize}`);
      console.log(`Local values: autoLaunch=${autoLaunch}, restoreTabsOnStartup=${restoreTabsOnStartup}, confirmTabClose=${confirmTabClose}`);

      // Save sequentially to avoid database locks
      await settingsStore.setTheme(theme);
      console.log('Theme saved');

      await themeStore.setTheme(selectedThemeId);
      console.log('File-based theme saved');

      await settingsStore.setFontFamily(fontFamily);
      console.log('FontFamily saved');

      await settingsStore.setFontSize(fontSize);
      console.log('FontSize saved');

      await settingsStore.setAutoLaunch(autoLaunch);
      console.log('AutoLaunch saved');

      await settingsStore.setRestoreTabsOnStartup(restoreTabsOnStartup);
      console.log('RestoreTabsOnStartup saved');

      await settingsStore.setConfirmTabClose(confirmTabClose);
      console.log('ConfirmTabClose saved');

      await settingsStore.setShowStatusBar(showStatusBar);
      console.log('ShowStatusBar saved');

      console.log('=== DIALOG handleSave END - calling onClose ===');
      onClose();
    } catch (error) {
      console.error('Failed to save settings:', error);
      await alertsStore.alert('Failed to save settings: ' + error, 'Error');
    } finally {
      saving = false;
    }
  }

  function handleCancel() {
    // Reset to current values
    theme = settingsStore.settings.theme;
    selectedThemeId = $themeStore.activeTheme?.id || 'dark';
    fontFamily = settingsStore.settings.fontFamily;
    fontSize = settingsStore.settings.fontSize;
    autoLaunch = settingsStore.settings.autoLaunch;
    restoreTabsOnStartup = settingsStore.settings.restoreTabsOnStartup;
    confirmTabClose = settingsStore.settings.confirmTabClose;
    // Re-apply the active theme to undo any live preview
    themeStore.setPreviewTheme(null);
    if ($themeStore.activeTheme) themeStore.applyTheme($themeStore.activeTheme);
    onClose();
  }

  const fontFamilies = [
    { value: 'monospace', label: 'Monospace (default)' },
    { value: '"FiraCode Nerd Font", "Fira Code", monospace', label: 'Fira Code Nerd Font' },
    { value: '"JetBrainsMono Nerd Font", "JetBrains Mono", monospace', label: 'JetBrains Mono Nerd Font' },
    { value: '"MesloLGS NF", "Meslo LG S", monospace', label: 'MesloLGS Nerd Font (p10k recommended)' },
    { value: '"Hack Nerd Font", "Hack", monospace', label: 'Hack Nerd Font' },
    { value: '"CascadiaCode Nerd Font", "Cascadia Code", monospace', label: 'Cascadia Code Nerd Font' },
    { value: '"SourceCodePro Nerd Font", "Source Code Pro", monospace', label: 'Source Code Pro Nerd Font' },
    { value: '"UbuntuMono Nerd Font", "Ubuntu Mono", monospace', label: 'Ubuntu Mono Nerd Font' },
    { value: '"DejaVuSansMono Nerd Font", "DejaVu Sans Mono", monospace', label: 'DejaVu Sans Mono Nerd Font' },
    { value: '"Consolas", monospace', label: 'Consolas' },
    { value: '"Courier New", monospace', label: 'Courier New' }
  ];

</script>

<Modal show={show} title="Settings" onClose={handleCancel} panelClass="w-[560px] max-h-[80vh] overflow-y-auto">
  <!-- Tabs Nav -->
  <div class="flex gap-2 mb-4" style="border-bottom: 1px solid var(--border-color)">
    <button class="px-3 py-2 text-sm rounded-t" style="background: {activeTab === 'appearance' ? 'var(--bg-tertiary)' : 'transparent'}; border: 1px solid var(--border-color); border-bottom: none;" onclick={() => activeTab = 'appearance'}>Appearance</button>
    <button class="px-3 py-2 text-sm rounded-t" style="background: {activeTab === 'typography' ? 'var(--bg-tertiary)' : 'transparent'}; border: 1px solid var(--border-color); border-bottom: none;" onclick={() => activeTab = 'typography'}>Typography</button>
    <button class="px-3 py-2 text-sm rounded-t" style="background: {activeTab === 'behavior' ? 'var(--bg-tertiary)' : 'transparent'}; border: 1px solid var(--border-color); border-bottom: none;" onclick={() => activeTab = 'behavior'}>Behavior</button>
    <button class="px-3 py-2 text-sm rounded-t" style="background: {activeTab === 'security' ? 'var(--bg-tertiary)' : 'transparent'}; border: 1px solid var(--border-color); border-bottom: none;" onclick={() => activeTab = 'security'}>Security</button>
  </div>
  <div class="space-y-6">
        <!-- Theme -->
        <div style="display: {activeTab === 'appearance' ? 'block' : 'none'}">
          <h3 class="text-lg font-medium mb-3">Appearance</h3>
          <div class="space-y-4">
            <div>
              <label for="selected_theme" class="block text-sm font-medium mb-2">Color Theme</label>
              <select
                id="selected_theme"
                bind:value={selectedThemeId}
                class="w-full px-3 py-2 rounded focus:outline-none border"
                style="background: var(--bg-tertiary); border-color: var(--border-color)"
                onchange={previewSelectedTheme}
              >
                {#each $themeStore.themes as themeOption}
                  <option value={themeOption.id}>
                    {themeOption.name} {themeOption.type === 'dark' ? '🌙' : '☀️'}
                  </option>
                {/each}
              </select>
              <p class="text-xs" style="color: var(--text-muted)">
                {#if $themeStore.activeTheme}
                  Currently active: {$themeStore.activeTheme.name}
                {/if}
              </p>
              <p class="text-xs mt-1" style="color: var(--text-muted)">
                Live preview updates as you change selection. Click Save to keep it, or Cancel to revert.
              </p>
            </div>

            <!-- Theme preview -->
            {#if $themeStore.activeTheme}
              <div class="mt-3">
                <div class="text-xs mb-1" style="color: var(--text-muted)">Preview</div>
                <div class="grid grid-cols-8 gap-1 p-2 rounded"
                     style="background: var(--bg-tertiary); border: 1px solid var(--border-color)">
                  <div class="h-5 rounded" style="background: var(--term-black)" title="black"></div>
                  <div class="h-5 rounded" style="background: var(--term-red)" title="red"></div>
                  <div class="h-5 rounded" style="background: var(--term-green)" title="green"></div>
                  <div class="h-5 rounded" style="background: var(--term-yellow)" title="yellow"></div>
                  <div class="h-5 rounded" style="background: var(--term-blue)" title="blue"></div>
                  <div class="h-5 rounded" style="background: var(--term-magenta)" title="magenta"></div>
                  <div class="h-5 rounded" style="background: var(--term-cyan)" title="cyan"></div>
                  <div class="h-5 rounded" style="background: var(--term-white)" title="white"></div>
                  <div class="h-5 rounded" style="background: var(--term-bright-black)" title="brightBlack"></div>
                  <div class="h-5 rounded" style="background: var(--term-bright-red)" title="brightRed"></div>
                  <div class="h-5 rounded" style="background: var(--term-bright-green)" title="brightGreen"></div>
                  <div class="h-5 rounded" style="background: var(--term-bright-yellow)" title="brightYellow"></div>
                  <div class="h-5 rounded" style="background: var(--term-bright-blue)" title="brightBlue"></div>
                  <div class="h-5 rounded" style="background: var(--term-bright-magenta)" title="brightMagenta"></div>
                  <div class="h-5 rounded" style="background: var(--term-bright-cyan)" title="brightCyan"></div>
                  <div class="h-5 rounded" style="background: var(--term-bright-white)" title="brightWhite"></div>
                </div>
              </div>
            {/if}

            <!-- Import/Export themes -->
            <div class="mt-4 space-y-3">
              <div>
                <label for="import_theme" class="block text-sm font-medium mb-1">Import theme from JSON</label>
                <div class="flex gap-2">
                  <button id="import_theme" class="px-3 py-2 rounded text-white"
                          style="background: var(--accent-green)"
                          onclick={async () => {
                            const path = await Dialogs.OpenFile({
                              Title: 'Select theme JSON',
                              Filters: [{ DisplayName: 'JSON files', Pattern: '*.json' }]
                            } as any);
                            if (!path) return;
                            await themeStore.importTheme(String(path));
                          }}>
                    Choose File…
                  </button>
                </div>
                <p class="text-xs mt-1" style="color: var(--text-muted)">
                  You can also add custom themes in ~/.config/term/themes/
                </p>
              </div>
              <div>
                <label for="export_theme" class="block text-sm font-medium mb-1">Export current theme</label>
                <div class="flex gap-2">
                  <button id="export_theme" class="px-3 py-2 rounded text-white disabled:opacity-60"
                          style="background: var(--accent-blue)"
                          disabled={exporting || !$themeStore.activeTheme}
                          onclick={async () => {
                            if (!$themeStore.activeTheme) return;
                            const suggested = `${$themeStore.activeTheme.id || 'theme'}.json`;
                            const dest = await Dialogs.SaveFile({ Filename: suggested } as any);
                            if (!dest) return;
                            exporting = true;
                            try {
                              await themeStore.exportTheme($themeStore.activeTheme.id, String(dest));
                            } finally {
                              exporting = false;
                            }
                          }}>
                    Save As…
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Typography -->
        <div style="display: {activeTab === 'typography' ? 'block' : 'none'}">
          <h3 class="text-lg font-medium mb-3">Typography</h3>
          <div class="space-y-4">
            <div>
              <label for="font_family" class="block text-sm font-medium mb-2">Font Family</label>
              <select
                id="font_family"
                bind:value={fontFamily}
                class="w-full px-3 py-2 rounded focus:outline-none border"
                style="background: var(--bg-tertiary); border-color: var(--border-color) font-family: {fontFamily}"
              >
                {#each fontFamilies as font}
                  <option value={font.value} style="font-family: {font.value}">
                    {font.label}
                  </option>
                {/each}
              </select>
              <p class="text-xs mt-1" style="color: var(--text-muted)">
                Font preview: <span style="font-family: {fontFamily}">The quick brown fox   󰜴 0O1lI</span>
              </p>
              <p class="text-xs mt-2" style="color: var(--text-muted)">
                💡 Nerd Fonts include powerline symbols and icons for oh-my-zsh, powerlevel10k, etc.
                <br />
                Download from <a href="https://www.nerdfonts.com/" class="text-blue-400 hover:underline" onclick={(e) => {
                  e.preventDefault();
                  Browser.OpenURL('https://www.nerdfonts.com/');
                }}>nerdfonts.com</a>
              </p>
            </div>

            <div>
              <label for="font_size" class="block text-sm font-medium mb-2">
                Font Size: {fontSize}px
              </label>
              <input
                id="font_size"
                type="range"
                bind:value={fontSize}
                min="8"
                max="24"
                step="1"
                class="w-full"
              />
              <div class="flex justify-between text-xs mt-1" style="color: var(--text-muted)">
                <span>8px</span>
                <span>24px</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Behavior -->
        <div style="display: {activeTab === 'behavior' ? 'block' : 'none'}">
          <h3 class="text-lg font-medium mb-3">Behavior</h3>
          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <div>
                <!-- svelte-ignore a11y_label_has_associated_control -->
                <label class="block text-sm font-medium">Auto-launch tabs</label>
                <p class="text-xs" style="color: var(--text-muted)">
                  Automatically open a terminal tab when selecting a session
                </p>
              </div>
              <ToggleSwitch checked={autoLaunch} ariaLabel="Auto-launch tabs" on:change={(e) => autoLaunch = e.detail} />
            </div>

            <div class="flex items-center justify-between">
              <div>
                <!-- svelte-ignore a11y_label_has_associated_control -->
                <label class="block text-sm font-medium">Restore tabs on startup</label>
                <p class="text-xs" style="color: var(--text-muted)">
                  Automatically restore previously open tabs when app starts
                </p>
              </div>
              <ToggleSwitch checked={restoreTabsOnStartup} ariaLabel="Restore tabs on startup" on:change={(e) => restoreTabsOnStartup = e.detail} />
            </div>

            <div class="flex items-center justify-between">
              <div>
                <!-- svelte-ignore a11y_label_has_associated_control -->
                <label class="block text-sm font-medium">Confirm tab close</label>
                <p class="text-xs" style="color: var(--text-muted)">
                  Show confirmation dialog when closing active tabs
                </p>
              </div>
              <ToggleSwitch checked={confirmTabClose} ariaLabel="Confirm tab close" on:change={(e) => confirmTabClose = e.detail} />
            </div>

            <div class="flex items-center justify-between">
              <div>
                <!-- svelte-ignore a11y_label_has_associated_control -->
                <label class="block text-sm font-medium">Show status bar</label>
                <p class="text-xs" style="color: var(--text-muted)">
                  Display system resource monitoring bar (CPU, RAM, Disk, Network)
                </p>
              </div>
              <ToggleSwitch checked={showStatusBar} ariaLabel="Show status bar" on:change={(e) => showStatusBar = e.detail} />
            </div>
          </div>
        </div>

        <!-- Known Hosts -->
        <div style="display: {activeTab === 'security' ? 'block' : 'none'}">
          <h3 class="text-lg font-medium mb-3">Known Hosts</h3>
          <div class="space-y-2">
            {#if !knownHostsLoaded}
              <div class="text-sm" style="color: var(--text-muted)">Loading known hosts…</div>
            {:else if knownHosts.length === 0}
              <div class="text-sm" style="color: var(--text-muted)">No known hosts saved yet.</div>
            {:else}
              <div class="max-h-60 overflow-auto rounded border" style="border-color: var(--border-color)">
                <table class="w-full text-sm" style="border-collapse: collapse">
                  <thead>
                    <tr style="background: var(--bg-tertiary)">
                      <th class="text-left p-2 font-medium">Host</th>
                      <th class="text-left p-2 font-medium">Key Type</th>
                      <th class="text-left p-2 font-medium">Fingerprint</th>
                      <th class="text-right p-2 font-medium">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each knownHosts as item (item.id)}
                      <tr style="border-top: 1px solid var(--border-color)">
                        <td class="p-2">{hostWithPort(item.host, item.port)}</td>
                        <td class="p-2">{item.keyType}</td>
                        <td class="p-2" style="font-family: monospace">{item.fingerprint}</td>
                        <td class="p-2 text-right">
                          <button class="px-2 py-1 text-xs rounded" style="background: var(--bg-tertiary)" onclick={() => deleteKnownHost(item)}>Delete</button>
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </div>
              <p class="text-xs" style="color: var(--text-muted)">Use this list to remove stale or reused IP entries (e.g., recycled VM IPs).</p>
            {/if}
          </div>
        </div>

        <!-- Recording Defaults -->
        <div style="display: {activeTab === 'security' ? 'block' : 'none'}">
          <h3 class="text-lg font-medium mb-3">Recording Defaults</h3>
          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <div>
                <label class="block text-sm font-medium">Capture input by default</label>
                <p class="text-xs" style="color: var(--text-muted)">Keystrokes may include sensitive data; off by default</p>
              </div>
              <ToggleSwitch checked={settingsStore.settings.recordingDefaultCaptureInput} ariaLabel="Default capture input" on:change={(e) => settingsStore.setRecordingDefaultCaptureInput(e.detail)} />
            </div>
            <div class="flex items-center justify-between">
              <div>
                <label class="block text-sm font-medium">Encrypt recordings at rest by default</label>
                <p class="text-xs" style="color: var(--text-muted)">AES-GCM per-file key; passphrase will be requested when starting</p>
              </div>
              <ToggleSwitch checked={settingsStore.settings.recordingDefaultEncrypt} ariaLabel="Default encrypt recordings" on:change={(e) => settingsStore.setRecordingDefaultEncrypt(e.detail)} />
            </div>
          </div>
        </div>
  </div>

  {#snippet footer()}
    <div class="flex gap-2 mt-6 pt-6" style="border-top: 1px solid var(--border-color)">
      <button
        onclick={handleSave}
        class="flex-1 px-4 py-2 rounded font-medium transition-colors text-white"
        style="background: var(--accent-blue)"
      >
        Save
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
