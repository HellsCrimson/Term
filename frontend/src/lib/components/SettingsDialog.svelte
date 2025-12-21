<script lang="ts">
  import { settingsStore } from '../stores/settings.svelte';
  import { themeStore } from '../stores/themeStore';

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

  // Update local state when settings change (but not while saving)
  $effect(() => {
    if (!saving) {
      theme = settingsStore.settings.theme;
      selectedThemeId = $themeStore.activeTheme?.id || 'dark';
      fontFamily = settingsStore.settings.fontFamily;
      fontSize = settingsStore.settings.fontSize;
      autoLaunch = settingsStore.settings.autoLaunch;
      restoreTabsOnStartup = settingsStore.settings.restoreTabsOnStartup;
      confirmTabClose = settingsStore.settings.confirmTabClose;
      showStatusBar = settingsStore.settings.showStatusBar;
    }
  });

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
      alert('Failed to save settings: ' + error);
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

{#if show}
  <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
    <div class="bg-gray-800 rounded-lg p-6 w-[500px] max-h-[80vh] overflow-y-auto border border-gray-700">
      <h2 class="text-2xl font-semibold mb-6">Settings</h2>

      <div class="space-y-6">
        <!-- Theme -->
        <div>
          <h3 class="text-lg font-medium mb-3">Appearance</h3>
          <div class="space-y-4">
            <div>
              <label for="selected_theme" class="block text-sm font-medium mb-2">Color Theme</label>
              <select
                id="selected_theme"
                bind:value={selectedThemeId}
                class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
              >
                {#each $themeStore.themes as themeOption}
                  <option value={themeOption.id}>
                    {themeOption.name} {themeOption.type === 'dark' ? '🌙' : '☀️'}
                  </option>
                {/each}
              </select>
              <p class="text-xs text-gray-400 mt-1">
                {#if $themeStore.activeTheme}
                  Currently active: {$themeStore.activeTheme.name}
                {/if}
              </p>
            </div>
          </div>
        </div>

        <!-- Typography -->
        <div>
          <h3 class="text-lg font-medium mb-3">Typography</h3>
          <div class="space-y-4">
            <div>
              <label for="font_family" class="block text-sm font-medium mb-2">Font Family</label>
              <select
                id="font_family"
                bind:value={fontFamily}
                class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                style="font-family: {fontFamily}"
              >
                {#each fontFamilies as font}
                  <option value={font.value} style="font-family: {font.value}">
                    {font.label}
                  </option>
                {/each}
              </select>
              <p class="text-xs text-gray-400 mt-1">
                Font preview: <span style="font-family: {fontFamily}">The quick brown fox   󰜴 0O1lI</span>
              </p>
              <p class="text-xs text-gray-500 mt-2">
                💡 Nerd Fonts include powerline symbols and icons for oh-my-zsh, powerlevel10k, etc.
                <br />
                Download from <a href="https://www.nerdfonts.com/" class="text-blue-400 hover:underline" onclick={(e) => { e.preventDefault(); }}>nerdfonts.com</a>
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
              <div class="flex justify-between text-xs text-gray-400 mt-1">
                <span>8px</span>
                <span>24px</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Behavior -->
        <div>
          <h3 class="text-lg font-medium mb-3">Behavior</h3>
          <div class="space-y-4">
            <div class="flex items-center justify-between">
              <div>
                <!-- svelte-ignore a11y_label_has_associated_control -->
                <label class="block text-sm font-medium">Auto-launch tabs</label>
                <p class="text-xs text-gray-400">
                  Automatically open a terminal tab when selecting a session
                </p>
              </div>
              <label class="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  bind:checked={autoLaunch}
                  class="sr-only peer"
                />
                <div
                  class="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-500 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"
                ></div>
              </label>
            </div>

            <div class="flex items-center justify-between">
              <div>
                <!-- svelte-ignore a11y_label_has_associated_control -->
                <label class="block text-sm font-medium">Restore tabs on startup</label>
                <p class="text-xs text-gray-400">
                  Automatically restore previously open tabs when app starts
                </p>
              </div>
              <label class="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  bind:checked={restoreTabsOnStartup}
                  class="sr-only peer"
                />
                <div
                  class="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-500 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"
                ></div>
              </label>
            </div>

            <div class="flex items-center justify-between">
              <div>
                <!-- svelte-ignore a11y_label_has_associated_control -->
                <label class="block text-sm font-medium">Confirm tab close</label>
                <p class="text-xs text-gray-400">
                  Show confirmation dialog when closing active tabs
                </p>
              </div>
              <label class="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  bind:checked={confirmTabClose}
                  class="sr-only peer"
                />
                <div
                  class="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-500 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"
                ></div>
              </label>
            </div>

            <div class="flex items-center justify-between">
              <div>
                <!-- svelte-ignore a11y_label_has_associated_control -->
                <label class="block text-sm font-medium">Show status bar</label>
                <p class="text-xs text-gray-400">
                  Display system resource monitoring bar (CPU, RAM, Disk, Network)
                </p>
              </div>
              <label class="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  bind:checked={showStatusBar}
                  class="sr-only peer"
                />
                <div
                  class="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-500 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"
                ></div>
              </label>
            </div>
          </div>
        </div>
      </div>

      <div class="flex gap-2 mt-6 pt-6 border-t border-gray-700">
        <button
          onclick={handleSave}
          class="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded font-medium transition-colors"
        >
          Save
        </button>
        <button
          onclick={handleCancel}
          class="flex-1 px-4 py-2 bg-gray-600 hover:bg-gray-700 rounded font-medium transition-colors"
        >
          Cancel
        </button>
      </div>
    </div>
  </div>
{/if}
