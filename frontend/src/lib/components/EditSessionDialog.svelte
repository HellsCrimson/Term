<script lang="ts">
  import { sessionsStore } from '../stores/sessions.svelte';
  import type { SessionNode } from '../types';

  interface Props {
    show: boolean;
    session: SessionNode | null;
    onClose: () => void;
  }

  let { show, session, onClose }: Props = $props();

  let sessionName = $state('');
  let sshHost = $state('');
  let sshPort = $state('22');
  let sshUsername = $state('');
  let sshAuthMethod = $state<'password' | 'key'>('key');
  let sshPassword = $state('');
  let sshKeyPath = $state('');
  let workingDirectory = $state('');
  let startupCommands = $state('');
  let environmentVariables = $state('');
  let loading = $state(false);
  let inheritedConfig = $state<Record<string, string>>({});

  // RDP-specific fields
  let rdpHost = $state('');
  let rdpPort = $state('3389');
  let rdpUsername = $state('');
  let rdpPassword = $state('');
  let rdpDomain = $state('');
  let rdpSecurity = $state<'any' | 'nla' | 'tls' | 'rdp'>('any');

  // VNC-specific fields
  let vncHost = $state('');
  let vncPort = $state('5900');
  let vncPassword = $state('');

  // Telnet-specific fields
  let telnetHost = $state('');
  let telnetPort = $state('23');
  let telnetUsername = $state('');
  let telnetPassword = $state('');

  // Desktop configuration (for RDP/VNC)
  let desktopWidth = $state('1920');
  let desktopHeight = $state('1080');
  let desktopColorDepth = $state<'8' | '16' | '24' | '32'>('32');

  // Tab state
  let activeTab = $state<'connection' | 'session' | 'display' | 'vnc'>('connection');

  // Load session config when dialog opens
  $effect(() => {
    if (show && session) {
      sessionName = session.name;
      loadConfig();
    }
  });

  async function loadConfig() {
    if (!session) return;

    loading = true;
    try {
      // Load both direct config and effective (inherited) config
      const [directConfig, effectiveConfig] = await Promise.all([
        sessionsStore.getSessionConfig(session.id),
        sessionsStore.getEffectiveConfig(session.id)
      ]);

      // Determine which values are inherited
      inheritedConfig = {};
      for (const key in effectiveConfig) {
        if (!(key in directConfig)) {
          inheritedConfig[key] = effectiveConfig[key];
        }
      }

      // Set form values to direct config (what's actually set on this session/folder)
      sshHost = directConfig.ssh_host || '';
      sshPort = directConfig.ssh_port || '';
      sshUsername = directConfig.ssh_username || '';
      sshAuthMethod = (directConfig.ssh_auth_method as 'password' | 'key') || 'password';
      sshPassword = directConfig.ssh_password || '';
      sshKeyPath = directConfig.ssh_key_path || '';
      workingDirectory = directConfig.working_directory || '';
      startupCommands = directConfig.startup_commands || '';
      environmentVariables = directConfig.environment_variables || '';

      // Load RDP config
      rdpHost = directConfig.rdp_host || '';
      rdpPort = directConfig.rdp_port || '3389';
      rdpUsername = directConfig.rdp_username || '';
      rdpPassword = directConfig.rdp_password || '';
      rdpDomain = directConfig.rdp_domain || '';
      rdpSecurity = (directConfig.rdp_security as 'any' | 'nla' | 'tls' | 'rdp') || 'any';

      // Load VNC config
      vncHost = directConfig.vnc_host || '';
      vncPort = directConfig.vnc_port || '5900';
      vncPassword = directConfig.vnc_password || '';

      // Load Telnet config
      telnetHost = directConfig.telnet_host || '';
      telnetPort = directConfig.telnet_port || '23';
      telnetUsername = directConfig.telnet_username || '';
      telnetPassword = directConfig.telnet_password || '';

      // Load desktop config (for RDP/VNC)
      desktopWidth = directConfig.desktop_width || '1920';
      desktopHeight = directConfig.desktop_height || '1080';
      desktopColorDepth = (directConfig.desktop_color_depth as '8' | '16' | '24' | '32') || '32';
    } catch (error) {
      console.error('Failed to load config:', error);
    } finally {
      loading = false;
    }
  }

  async function handleSave() {
    if (!session) return;

    if (!sessionName.trim()) {
      alert('Name is required');
      return;
    }

    // Validate SSH fields if it's an SSH session
    // Only host is required - other fields can be inherited
    if (session.sessionType === 'ssh') {
      if (!sshHost.trim()) {
        alert('SSH host is required (other fields can be inherited from folder)');
        return;
      }
    }

    // Validation for RDP
    if (session.sessionType === 'rdp') {
      if (!rdpHost.trim()) {
        alert('RDP host is required');
        return;
      }
    }

    // Validation for VNC
    if (session.sessionType === 'vnc') {
      if (!vncHost.trim()) {
        alert('VNC host is required');
        return;
      }
    }

    // Validation for Telnet
    if (session.sessionType === 'telnet') {
      if (!telnetHost.trim()) {
        alert('Telnet host is required');
        return;
      }
    }

    try {
      // Update session name if changed
      if (sessionName !== session.name) {
        await sessionsStore.updateSession({
          ...session,
          name: sessionName.trim()
        });
      }

      // Save general session config (for terminal session types only)
      if (session.type === 'session' && !['rdp', 'vnc', 'telnet'].includes(session.sessionType || '')) {
        if (workingDirectory.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'working_directory', workingDirectory.toString());
        }
        if (startupCommands.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'startup_commands', startupCommands.toString());
        }
        if (environmentVariables.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'environment_variables', environmentVariables.toString());
        }
      }

      // Save SSH config if SSH session (only save non-empty values)
      if (session.sessionType === 'ssh') {
        // Host is required
        if (sshHost.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'ssh_host', sshHost.toString());
        }

        // Optional fields - only save if not empty
        if (sshPort.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'ssh_port', sshPort.toString());
        }
        if (sshUsername.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'ssh_username', sshUsername.toString());
        }

        // Save auth method (always, since it's a user choice)
        await sessionsStore.setSessionConfig(session.id, 'ssh_auth_method', sshAuthMethod.toString());

        // Save credentials only if provided
        if (sshAuthMethod === 'password' && sshPassword.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'ssh_password', sshPassword.toString());
        } else if (sshAuthMethod === 'key' && sshKeyPath.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'ssh_key_path', sshKeyPath.toString());
        }
      }

      // Save RDP config if RDP session
      if (session.sessionType === 'rdp') {
        if (rdpHost.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'rdp_host', rdpHost.toString());
        }
        if (rdpPort.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'rdp_port', rdpPort.toString());
        }
        if (rdpUsername.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'rdp_username', rdpUsername.toString());
        }
        if (rdpPassword.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'rdp_password', rdpPassword.toString());
        }
        if (rdpDomain.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'rdp_domain', rdpDomain.toString());
        }
        await sessionsStore.setSessionConfig(session.id, 'rdp_security', rdpSecurity.toString());
        await sessionsStore.setSessionConfig(session.id, 'desktop_width', desktopWidth.toString());
        await sessionsStore.setSessionConfig(session.id, 'desktop_height', desktopHeight.toString());
        await sessionsStore.setSessionConfig(session.id, 'desktop_color_depth', desktopColorDepth.toString());
      }

      // Save VNC config if VNC session
      if (session.sessionType === 'vnc') {
        if (vncHost.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'vnc_host', vncHost.toString());
        }
        if (vncPort.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'vnc_port', vncPort.toString());
        }
        if (vncPassword.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'vnc_password', vncPassword.toString());
        }
        await sessionsStore.setSessionConfig(session.id, 'desktop_width', desktopWidth.toString());
        await sessionsStore.setSessionConfig(session.id, 'desktop_height', desktopHeight.toString());
        await sessionsStore.setSessionConfig(session.id, 'desktop_color_depth', desktopColorDepth.toString());
      }

      // Save Telnet config if Telnet session
      if (session.sessionType === 'telnet') {
        if (telnetHost.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'telnet_host', telnetHost.toString());
        }
        if (telnetPort.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'telnet_port', telnetPort.toString());
        }
        if (telnetUsername.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'telnet_username', telnetUsername.toString());
        }
        if (telnetPassword.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'telnet_password', telnetPassword.toString());
        }
      }

      // Save folder config (inherited by children)
      if (session.type === 'folder') {
        // SSH config
        if (sshUsername.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'ssh_username', sshUsername.toString());
        }
        if (sshPort.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'ssh_port', sshPort.toString());
        }
        if (sshKeyPath.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'ssh_key_path', sshKeyPath.toString());
        }

        // Terminal config
        if (workingDirectory.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'working_directory', workingDirectory.toString());
        }
        if (startupCommands.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'startup_commands', startupCommands.toString());
        }
        if (environmentVariables.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'environment_variables', environmentVariables.toString());
        }

        // RDP/VNC config
        if (rdpUsername.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'rdp_username', rdpUsername.toString());
        }
        if (rdpDomain.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'rdp_domain', rdpDomain.toString());
        }
        if (rdpSecurity) {
          await sessionsStore.setSessionConfig(session.id, 'rdp_security', rdpSecurity.toString());
        }
        if (desktopWidth.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'desktop_width', desktopWidth.toString());
        }
        if (desktopHeight.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'desktop_height', desktopHeight.toString());
        }
        if (desktopColorDepth) {
          await sessionsStore.setSessionConfig(session.id, 'desktop_color_depth', desktopColorDepth.toString());
        }
      }

      onClose();
    } catch (error) {
      console.error('Failed to save session:', error);
      alert('Failed to save session: ' + error);
    }
  }

  function handleCancel() {
    onClose();
  }
</script>

{#if show && session}
  <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
    <div class="bg-gray-800 rounded-lg p-6 w-96 border border-gray-700">
      <h2 class="text-xl font-semibold mb-4">
        Edit {session.type === 'folder' ? 'Folder' : 'Session'}
      </h2>

      {#if loading}
        <div class="text-center py-8">
          <div class="text-gray-400">Loading configuration...</div>
        </div>
      {:else}
        <div class="space-y-4">
          <div>
            <label for="session_name" class="block text-sm font-medium mb-1">Name</label>
            <input
            id="session_name"
              type="text"
              bind:value={sessionName}
              class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
              placeholder="My Terminal"
            />
          </div>

          {#if session.sessionType === 'ssh'}
            <!-- Tab Navigation -->
            <div class="flex border-b border-gray-600">
              <button
                type="button"
                class="px-4 py-2 text-sm font-medium transition-colors {activeTab === 'connection' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-gray-300'}"
                onclick={() => activeTab = 'connection'}
              >
                Connection
              </button>
              <button
                type="button"
                class="px-4 py-2 text-sm font-medium transition-colors {activeTab === 'session' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-gray-300'}"
                onclick={() => activeTab = 'session'}
              >
                Session
              </button>
            </div>

            <!-- Tab Content -->
            {#if activeTab === 'connection'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-blue-400">SSH Connection</h4>

                <div class="grid grid-cols-2 gap-3">
                  <div class="col-span-2">
                    <label for="ssh_host" class="block text-xs font-medium mb-1">Host *</label>
                    <input
                      id="ssh_host"
                      type="text"
                      bind:value={sshHost}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder="192.168.1.100 or example.com"
                    />
                  </div>

                  <div>
                    <label for="ssh_port" class="block text-xs font-medium mb-1">Port</label>
                    <input
                      id="ssh_port"
                      type="text"
                      bind:value={sshPort}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder={inheritedConfig.ssh_port ? `Inherited: ${inheritedConfig.ssh_port}` : '22'}
                    />
                    {#if inheritedConfig.ssh_port && !sshPort}
                      <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inheritedConfig.ssh_port}</p>
                    {/if}
                  </div>

                  <div>
                    <label for="ssh_username" class="block text-xs font-medium mb-1">Username</label>
                    <input
                      id="ssh_username"
                      type="text"
                      bind:value={sshUsername}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder={inheritedConfig.ssh_username ? `Inherited: ${inheritedConfig.ssh_username}` : 'root'}
                    />
                    {#if inheritedConfig.ssh_username && !sshUsername}
                      <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inheritedConfig.ssh_username}</p>
                    {/if}
                  </div>
                </div>

                <div>
                  <label for="ssh_auth_method" class="block text-xs font-medium mb-1">Authentication</label>
                  <select
                    id="ssh_auth_method"
                    bind:value={sshAuthMethod}
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                  >
                    <option value="password">Password</option>
                    <option value="key">SSH Key</option>
                  </select>
                </div>

                {#if sshAuthMethod === 'password'}
                  <div>
                    <label for="ssh_password" class="block text-xs font-medium mb-1">Password *</label>
                    <input
                      id="ssh_password"
                      type="password"
                      bind:value={sshPassword}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder="••••••••"
                    />
                  </div>
                {:else}
                  <div>
                    <label for="ssh_key_path" class="block text-xs font-medium mb-1">Key Path</label>
                    <input
                      id="ssh_key_path"
                      type="text"
                      bind:value={sshKeyPath}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder={inheritedConfig.ssh_key_path ? `Inherited: ${inheritedConfig.ssh_key_path}` : '~/.ssh/id_rsa'}
                    />
                    {#if inheritedConfig.ssh_key_path && !sshKeyPath}
                      <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inheritedConfig.ssh_key_path}</p>
                    {:else}
                      <p class="text-xs text-gray-500 mt-1">Path to your private key file</p>
                    {/if}
                  </div>
                {/if}
              </div>
            {:else if activeTab === 'session'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-purple-400">Session Configuration</h4>

                <div>
                  <label for="working_directory" class="block text-xs font-medium mb-1">Working Directory</label>
                  <input
                    id="working_directory"
                    type="text"
                    bind:value={workingDirectory}
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    placeholder={inheritedConfig.working_directory ? `Inherited: ${inheritedConfig.working_directory}` : '~/projects or /home/user'}
                  />
                  {#if inheritedConfig.working_directory && !workingDirectory}
                    <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inheritedConfig.working_directory}</p>
                  {:else}
                    <p class="text-xs text-gray-400">Directory where the session will start (supports ~ for home)</p>
                  {/if}
                </div>

                <div>
                  <label for="startup_command" class="block text-xs font-medium mb-1">Startup Commands</label>
                  <textarea
                    id="startup_command"
                    bind:value={startupCommands}
                    rows="3"
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500 font-mono"
                    placeholder={inheritedConfig.startup_commands ? `Inherited: ${inheritedConfig.startup_commands}` : 'cd ~/project; source .env'}
                  ></textarea>
                  {#if inheritedConfig.startup_commands && !startupCommands}
                    <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inheritedConfig.startup_commands}</p>
                  {:else}
                    <p class="text-xs text-gray-400">Commands to run when the session starts (separated by semicolons)</p>
                  {/if}
                </div>

                <div>
                  <label for="environment_variables" class="block text-xs font-medium mb-1">Environment Variables</label>
                  <textarea
                    id="environment_variables"
                    bind:value={environmentVariables}
                    rows="3"
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500 font-mono"
                    placeholder={inheritedConfig.environment_variables ? `Inherited: ${inheritedConfig.environment_variables}` : 'KEY1=value1; KEY2=value2'}
                  ></textarea>
                  {#if inheritedConfig.environment_variables && !environmentVariables}
                    <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inheritedConfig.environment_variables}</p>
                  {:else}
                    <p class="text-xs text-gray-400">Environment variables (KEY=value; separated by semicolons)</p>
                  {/if}
                </div>
              </div>
            {/if}
          {/if}

          <!-- RDP Session Configuration -->
          {#if session.type === 'session' && session.sessionType === 'rdp'}
            <!-- Tab Navigation -->
            <div class="flex border-b border-gray-600">
              <button
                type="button"
                class="px-4 py-2 text-sm font-medium transition-colors {activeTab === 'connection' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-gray-300'}"
                onclick={() => activeTab = 'connection'}
              >
                Connection
              </button>
              <button
                type="button"
                class="px-4 py-2 text-sm font-medium transition-colors {activeTab === 'display' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-gray-300'}"
                onclick={() => activeTab = 'display'}
              >
                Display
              </button>
            </div>

            <!-- Tab Content -->
            {#if activeTab === 'connection'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-cyan-400">RDP Connection</h4>

                <div class="grid grid-cols-2 gap-3">
                  <div class="col-span-2">
                    <label for="rdp_host" class="block text-xs font-medium mb-1">Host *</label>
                    <input
                      id="rdp_host"
                      type="text"
                      bind:value={rdpHost}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder="192.168.1.100 or windows-server.local"
                    />
                  </div>

                  <div>
                    <label for="rdp_port" class="block text-xs font-medium mb-1">Port</label>
                    <input
                      id="rdp_port"
                      type="text"
                      bind:value={rdpPort}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder="3389"
                    />
                  </div>

                  <div>
                    <label for="rdp_security" class="block text-xs font-medium mb-1">Security</label>
                    <select
                      id="rdp_security"
                      bind:value={rdpSecurity}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    >
                      <option value="any">Any</option>
                      <option value="nla">NLA</option>
                      <option value="tls">TLS</option>
                      <option value="rdp">RDP</option>
                    </select>
                  </div>

                  <div>
                    <label for="rdp_username" class="block text-xs font-medium mb-1">Username</label>
                    <input
                      id="rdp_username"
                      type="text"
                      bind:value={rdpUsername}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder="administrator"
                    />
                  </div>

                  <div>
                    <label for="rdp_password" class="block text-xs font-medium mb-1">Password</label>
                    <input
                      id="rdp_password"
                      type="password"
                      bind:value={rdpPassword}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder="••••••••"
                    />
                  </div>

                  <div class="col-span-2">
                    <label for="rdp_domain" class="block text-xs font-medium mb-1">Domain (Optional)</label>
                    <input
                      id="rdp_domain"
                      type="text"
                      bind:value={rdpDomain}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder="CORP or corp.local"
                    />
                  </div>
                </div>
              </div>
            {:else if activeTab === 'display'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-cyan-400">Display Settings</h4>

                <div class="grid grid-cols-3 gap-3">
                  <div>
                    <label for="desktop_width" class="block text-xs font-medium mb-1">Width</label>
                    <input
                      id="desktop_width"
                      type="text"
                      bind:value={desktopWidth}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder="1920"
                    />
                  </div>
                  <div>
                    <label for="desktop_height" class="block text-xs font-medium mb-1">Height</label>
                    <input
                      id="desktop_height"
                      type="text"
                      bind:value={desktopHeight}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder="1080"
                    />
                  </div>
                  <div>
                    <label for="desktop_color_depth" class="block text-xs font-medium mb-1">Color Depth</label>
                    <select
                      id="desktop_color_depth"
                      bind:value={desktopColorDepth}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    >
                      <option value="8">8-bit</option>
                      <option value="16">16-bit</option>
                      <option value="24">24-bit</option>
                      <option value="32">32-bit</option>
                    </select>
                  </div>
                </div>
              </div>
            {/if}
          {/if}

          <!-- VNC Session Configuration -->
          {#if session.type === 'session' && session.sessionType === 'vnc'}
            <!-- Tab Navigation -->
            <div class="flex border-b border-gray-600">
              <button
                type="button"
                class="px-4 py-2 text-sm font-medium transition-colors {activeTab === 'connection' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-gray-300'}"
                onclick={() => activeTab = 'connection'}
              >
                Connection
              </button>
              <button
                type="button"
                class="px-4 py-2 text-sm font-medium transition-colors {activeTab === 'display' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-gray-300'}"
                onclick={() => activeTab = 'display'}
              >
                Display
              </button>
            </div>

            <!-- Tab Content -->
            {#if activeTab === 'connection'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-green-400">VNC Connection</h4>

                <div class="grid grid-cols-2 gap-3">
                  <div class="col-span-2">
                    <label for="vnc_host" class="block text-xs font-medium mb-1">Host *</label>
                    <input
                      id="vnc_host"
                      type="text"
                      bind:value={vncHost}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder="192.168.1.100 or vnc-server.local"
                    />
                  </div>

                  <div>
                    <label for="vnc_port" class="block text-xs font-medium mb-1">Port</label>
                    <input
                      id="vnc_port"
                      type="text"
                      bind:value={vncPort}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder="5900"
                    />
                  </div>

                  <div>
                    <label for="vnc_password" class="block text-xs font-medium mb-1">Password</label>
                    <input
                      id="vnc_password"
                      type="password"
                      bind:value={vncPassword}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder="••••••••"
                    />
                  </div>
                </div>
              </div>
            {:else if activeTab === 'display'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-green-400">Display Settings</h4>

                <div class="grid grid-cols-3 gap-3">
                  <div>
                    <label for="desktop_width" class="block text-xs font-medium mb-1">Width</label>
                    <input
                      id="desktop_width"
                      type="text"
                      bind:value={desktopWidth}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder="1920"
                    />
                  </div>
                  <div>
                    <label for="desktop_height" class="block text-xs font-medium mb-1">Height</label>
                    <input
                      id="desktop_height"
                      type="text"
                      bind:value={desktopHeight}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder="1080"
                    />
                  </div>
                  <div>
                    <label for="desktop_color_depth" class="block text-xs font-medium mb-1">Color Depth</label>
                    <select
                      id="desktop_color_depth"
                      bind:value={desktopColorDepth}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    >
                      <option value="8">8-bit</option>
                      <option value="16">16-bit</option>
                      <option value="24">24-bit</option>
                      <option value="32">32-bit</option>
                    </select>
                  </div>
                </div>
              </div>
            {/if}
          {/if}

          <!-- Telnet Session Configuration -->
          {#if session.type === 'session' && session.sessionType === 'telnet'}
            <!-- Tab Navigation -->
            <div class="flex border-b border-gray-600">
              <button
                type="button"
                class="px-4 py-2 text-sm font-medium text-blue-400 border-b-2 border-blue-400"
              >
                Connection
              </button>
            </div>

            <!-- Tab Content -->
            <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
              <h4 class="text-sm font-medium text-orange-400">Telnet Connection</h4>

              <div class="grid grid-cols-2 gap-3">
                <div class="col-span-2">
                  <label for="telnet_host" class="block text-xs font-medium mb-1">Host *</label>
                  <input
                    id="telnet_host"
                    type="text"
                    bind:value={telnetHost}
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    placeholder="192.168.1.100 or telnet-server.local"
                  />
                </div>

                <div>
                  <label for="telnet_port" class="block text-xs font-medium mb-1">Port</label>
                  <input
                    id="telnet_port"
                    type="text"
                    bind:value={telnetPort}
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    placeholder="23"
                  />
                </div>

                <div>
                  <label for="telnet_username" class="block text-xs font-medium mb-1">Username (Optional)</label>
                  <input
                    id="telnet_username"
                    type="text"
                    bind:value={telnetUsername}
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    placeholder="admin"
                  />
                </div>

                <div class="col-span-2">
                  <label for="telnet_password" class="block text-xs font-medium mb-1">Password (Optional)</label>
                  <input
                    id="telnet_password"
                    type="password"
                    bind:value={telnetPassword}
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    placeholder="••••••••"
                  />
                </div>
              </div>
              <p class="text-xs text-gray-400 mt-2">Note: Telnet is unencrypted. Use SSH when possible.</p>
            </div>
          {/if}

          <!-- Terminal Session Configuration (bash/zsh/fish/pwsh) -->
          {#if session.type === 'session' && !['ssh', 'rdp', 'vnc', 'telnet'].includes(session.sessionType || '')}
            <!-- Tab Navigation -->
            <div class="flex border-b border-gray-600">
              <button
                type="button"
                class="px-4 py-2 text-sm font-medium text-blue-400 border-b-2 border-blue-400"
              >
                Session
              </button>
            </div>

            <!-- Tab Content -->
            <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
              <h4 class="text-sm font-medium text-purple-400">Session Configuration</h4>

              <div>
                <label for="working_directory" class="block text-xs font-medium mb-1">Working Directory</label>
                <input
                  id="working_directory"
                  type="text"
                  bind:value={workingDirectory}
                  class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                  placeholder={inheritedConfig.working_directory ? `Inherited: ${inheritedConfig.working_directory}` : '~/projects or /home/user'}
                />
                {#if inheritedConfig.working_directory && !workingDirectory}
                  <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inheritedConfig.working_directory}</p>
                {:else}
                  <p class="text-xs text-gray-400">Directory where the session will start (supports ~ for home)</p>
                {/if}
              </div>

              <div>
                <label for="startup_commands" class="block text-xs font-medium mb-1">Startup Commands</label>
                <textarea
                  id="startup_commands"
                  bind:value={startupCommands}
                  rows="3"
                  class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500 font-mono"
                  placeholder={inheritedConfig.startup_commands ? `Inherited: ${inheritedConfig.startup_commands}` : 'cd ~/project; source .env'}
                ></textarea>
                {#if inheritedConfig.startup_commands && !startupCommands}
                  <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inheritedConfig.startup_commands}</p>
                {:else}
                  <p class="text-xs text-gray-400">Commands to run when the session starts (separated by semicolons)</p>
                {/if}
              </div>

              <div>
                <label for="environment_variables" class="block text-xs font-medium mb-1">Environment Variables</label>
                <textarea
                  id="environment_variables"
                  bind:value={environmentVariables}
                  rows="3"
                  class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500 font-mono"
                  placeholder={inheritedConfig.environment_variables ? `Inherited: ${inheritedConfig.environment_variables}` : 'KEY1=value1; KEY2=value2'}
                ></textarea>
                {#if inheritedConfig.environment_variables && !environmentVariables}
                  <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inheritedConfig.environment_variables}</p>
                {:else}
                  <p class="text-xs text-gray-400">Environment variables (KEY=value; separated by semicolons)</p>
                {/if}
              </div>
            </div>
          {/if}

          {#if session.type === 'folder'}
            <div class="mb-2">
              <h4 class="text-sm font-medium text-green-400">Folder Configuration (Inherited by Children)</h4>
              <p class="text-xs text-gray-400">These settings will be inherited by all sessions and subfolders inside this folder.</p>
            </div>

            <!-- Tab Navigation -->
            <div class="flex border-b border-gray-600 overflow-x-auto">
              <button
                type="button"
                class="px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors {activeTab === 'connection' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-gray-300'}"
                onclick={() => activeTab = 'connection'}
              >
                SSH
              </button>
              <button
                type="button"
                class="px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors {activeTab === 'session' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-gray-300'}"
                onclick={() => activeTab = 'session'}
              >
                Terminal
              </button>
              <button
                type="button"
                class="px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors {activeTab === 'display' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-gray-300'}"
                onclick={() => activeTab = 'display'}
              >
                RDP
              </button>
              <button
                type="button"
                class="px-4 py-2 text-sm font-medium whitespace-nowrap transition-colors {activeTab === 'vnc' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-gray-300'}"
                onclick={() => activeTab = 'vnc'}
              >
                VNC
              </button>
            </div>

            <!-- Tab Content -->
            {#if activeTab === 'connection'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-blue-400">SSH Configuration</h4>
                <p class="text-xs text-gray-400">Settings inherited by SSH sessions</p>

                <div>
                  <label for="ssh_username" class="block text-xs font-medium mb-1">SSH Username</label>
                  <input
                    id="ssh_username"
                    type="text"
                    bind:value={sshUsername}
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    placeholder={inheritedConfig.ssh_username ? `Inherited: ${inheritedConfig.ssh_username}` : 'root (inherited by SSH sessions)'}
                  />
                  {#if inheritedConfig.ssh_username && !sshUsername}
                    <p class="text-xs text-yellow-400 mt-1">↓ Inherited from parent: {inheritedConfig.ssh_username}</p>
                  {/if}
                </div>

                <div>
                  <label for="ssh_port" class="block text-xs font-medium mb-1">SSH Port</label>
                  <input
                    id="ssh_port"
                    type="text"
                    bind:value={sshPort}
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    placeholder={inheritedConfig.ssh_port ? `Inherited: ${inheritedConfig.ssh_port}` : '22 (inherited by SSH sessions)'}
                  />
                  {#if inheritedConfig.ssh_port && !sshPort}
                    <p class="text-xs text-yellow-400 mt-1">↓ Inherited from parent: {inheritedConfig.ssh_port}</p>
                  {/if}
                </div>

                <div>
                  <label for="ssh_key_path" class="block text-xs font-medium mb-1">SSH Key Path</label>
                  <input
                    id="ssh_key_path"
                    type="text"
                    bind:value={sshKeyPath}
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    placeholder={inheritedConfig.ssh_key_path ? `Inherited: ${inheritedConfig.ssh_key_path}` : '~/.ssh/id_rsa (inherited by SSH sessions)'}
                  />
                  {#if inheritedConfig.ssh_key_path && !sshKeyPath}
                    <p class="text-xs text-yellow-400 mt-1">↓ Inherited from parent: {inheritedConfig.ssh_key_path}</p>
                  {/if}
                </div>
              </div>
            {:else if activeTab === 'session'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-purple-400">Terminal Configuration</h4>
                <p class="text-xs text-gray-400">Settings inherited by terminal sessions (bash, zsh, fish, pwsh)</p>

                <div>
                  <label for="working_directory" class="block text-xs font-medium mb-1">Working Directory</label>
                  <input
                    id="working_directory"
                    type="text"
                    bind:value={workingDirectory}
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    placeholder={inheritedConfig.working_directory ? `Inherited: ${inheritedConfig.working_directory}` : '~/projects'}
                  />
                  {#if inheritedConfig.working_directory && !workingDirectory}
                    <p class="text-xs text-yellow-400 mt-1">↓ Inherited from parent: {inheritedConfig.working_directory}</p>
                  {/if}
                </div>

                <div>
                  <label for="startup_commands" class="block text-xs font-medium mb-1">Startup Commands</label>
                  <textarea
                    id="startup_commands"
                    bind:value={startupCommands}
                    rows="2"
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500 font-mono"
                    placeholder={inheritedConfig.startup_commands ? `Inherited: ${inheritedConfig.startup_commands}` : 'cd ~/project; source .env'}
                  ></textarea>
                  {#if inheritedConfig.startup_commands && !startupCommands}
                    <p class="text-xs text-yellow-400 mt-1">↓ Inherited from parent: {inheritedConfig.startup_commands}</p>
                  {:else}
                    <p class="text-xs text-gray-500 mt-1">Commands to run when sessions start</p>
                  {/if}
                </div>

                <div>
                  <label for="environment_variables" class="block text-xs font-medium mb-1">Environment Variables</label>
                  <textarea
                    id="environment_variables"
                    bind:value={environmentVariables}
                    rows="2"
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500 font-mono"
                    placeholder={inheritedConfig.environment_variables ? `Inherited: ${inheritedConfig.environment_variables}` : 'KEY1=value1; KEY2=value2'}
                  ></textarea>
                  {#if inheritedConfig.environment_variables && !environmentVariables}
                    <p class="text-xs text-yellow-400 mt-1">↓ Inherited from parent: {inheritedConfig.environment_variables}</p>
                  {:else}
                    <p class="text-xs text-gray-500 mt-1">Environment variables for sessions (KEY=value; separated)</p>
                  {/if}
                </div>
              </div>
            {:else if activeTab === 'display'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-cyan-400">RDP Configuration</h4>
                <p class="text-xs text-gray-400">Settings inherited by RDP sessions</p>

                <div class="grid grid-cols-2 gap-3">
                  <div>
                    <label for="rdp_username" class="block text-xs font-medium mb-1">RDP Username</label>
                    <input
                      id="rdp_username"
                      type="text"
                      bind:value={rdpUsername}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder={inheritedConfig.rdp_username ? `Inherited: ${inheritedConfig.rdp_username}` : 'administrator'}
                    />
                    {#if inheritedConfig.rdp_username && !rdpUsername}
                      <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inheritedConfig.rdp_username}</p>
                    {/if}
                  </div>

                  <div>
                    <label for="rdp_domain" class="block text-xs font-medium mb-1">RDP Domain</label>
                    <input
                      id="rdp_domain"
                      type="text"
                      bind:value={rdpDomain}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder={inheritedConfig.rdp_domain ? `Inherited: ${inheritedConfig.rdp_domain}` : 'CORP'}
                    />
                    {#if inheritedConfig.rdp_domain && !rdpDomain}
                      <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inheritedConfig.rdp_domain}</p>
                    {/if}
                  </div>

                  <div class="col-span-2">
                    <label for="rdp_security" class="block text-xs font-medium mb-1">RDP Security</label>
                    <select
                      id="rdp_security"
                      bind:value={rdpSecurity}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    >
                      <option value="any">Any</option>
                      <option value="nla">NLA</option>
                      <option value="tls">TLS</option>
                      <option value="rdp">RDP</option>
                    </select>
                  </div>
                </div>

                <div class="pt-3 border-t border-gray-600">
                  <h5 class="text-xs font-medium text-gray-300 mb-2">Display Settings</h5>
                  <div class="grid grid-cols-3 gap-3">
                    <div>
                      <label for="desktop_width" class="block text-xs font-medium mb-1">Width</label>
                      <input
                        id="desktop_width"
                        type="text"
                        bind:value={desktopWidth}
                        class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                        placeholder={inheritedConfig.desktop_width ? `Inherited: ${inheritedConfig.desktop_width}` : '1920'}
                      />
                    </div>
                    <div>
                      <label for="desktop_height" class="block text-xs font-medium mb-1">Height</label>
                      <input
                        id="desktop_height"
                        type="text"
                        bind:value={desktopHeight}
                        class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                        placeholder={inheritedConfig.desktop_height ? `Inherited: ${inheritedConfig.desktop_height}` : '1080'}
                      />
                    </div>
                    <div>
                      <label for="desktop_color_depth" class="block text-xs font-medium mb-1">Color Depth</label>
                      <select
                        id="desktop_color_depth"
                        bind:value={desktopColorDepth}
                        class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      >
                        <option value="8">8-bit</option>
                        <option value="16">16-bit</option>
                        <option value="24">24-bit</option>
                        <option value="32">32-bit</option>
                      </select>
                    </div>
                  </div>
                </div>
              </div>
            {:else if activeTab === 'vnc'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-green-400">VNC Configuration</h4>
                <p class="text-xs text-gray-400">Settings inherited by VNC sessions</p>

                <div class="pt-3 border-t border-gray-600">
                  <h5 class="text-xs font-medium text-gray-300 mb-2">Display Settings</h5>
                  <div class="grid grid-cols-3 gap-3">
                    <div>
                      <label for="desktop_width" class="block text-xs font-medium mb-1">Width</label>
                      <input
                        id="desktop_width"
                        type="text"
                        bind:value={desktopWidth}
                        class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                        placeholder={inheritedConfig.desktop_width ? `Inherited: ${inheritedConfig.desktop_width}` : '1920'}
                      />
                      {#if inheritedConfig.desktop_width && !desktopWidth}
                        <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inheritedConfig.desktop_width}</p>
                      {/if}
                    </div>
                    <div>
                      <label for="desktop_height" class="block text-xs font-medium mb-1">Height</label>
                      <input
                        id="desktop_height"
                        type="text"
                        bind:value={desktopHeight}
                        class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                        placeholder={inheritedConfig.desktop_height ? `Inherited: ${inheritedConfig.desktop_height}` : '1080'}
                      />
                      {#if inheritedConfig.desktop_height && !desktopHeight}
                        <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inheritedConfig.desktop_height}</p>
                      {/if}
                    </div>
                    <div>
                      <label for="desktop_color_depth" class="block text-xs font-medium mb-1">Color Depth</label>
                      <select
                        id="desktop_color_depth"
                        bind:value={desktopColorDepth}
                        class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      >
                        <option value="8">8-bit</option>
                        <option value="16">16-bit</option>
                        <option value="24">24-bit</option>
                        <option value="32">32-bit</option>
                      </select>
                    </div>
                  </div>
                  <p class="text-xs text-gray-400 mt-2">These settings are shared with RDP sessions</p>
                </div>
              </div>
            {/if}
          {/if}
        </div>

        <div class="flex gap-2 mt-6">
          <button
            onclick={handleSave}
            class="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded font-medium"
          >
            Save
          </button>
          <button
            onclick={handleCancel}
            class="flex-1 px-4 py-2 bg-gray-600 hover:bg-gray-700 rounded font-medium"
          >
            Cancel
          </button>
        </div>
      {/if}
    </div>
  </div>
{/if}
