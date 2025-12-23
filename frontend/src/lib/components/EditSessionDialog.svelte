<script lang="ts">
  import { sessionsStore } from '../stores/sessions.svelte';
  import type { SessionNode } from '../types';
  import LabeledInput from './common/LabeledInput.svelte';
  import LabeledSelect from './common/LabeledSelect.svelte';
  import Modal from './common/Modal.svelte';
  import Tabs from './common/Tabs.svelte';
  import DisplaySettings from './common/DisplaySettings.svelte';
  import RDPConnectionForm from './common/RDPConnectionForm.svelte';
  import VNCConnectionForm from './common/VNCConnectionForm.svelte';
  import TelnetConnectionForm from './common/TelnetConnectionForm.svelte';
    import TerminalSessionForm from './common/TerminalSessionForm.svelte';
    import SSHDefaultsForm from './common/SSHDefaultsForm.svelte';
  import { alertsStore } from '$lib/stores/alerts.svelte';

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
  let customCommand = $state('');
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
      customCommand = directConfig.command || '';

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
      await alertsStore.alert('Name is required', 'Validation');
      return;
    }

    // Validate SSH fields if it's an SSH session
    // Only host is required - other fields can be inherited
    if (session.sessionType === 'ssh') {
      if (!sshHost.trim()) {
        await alertsStore.alert('SSH host is required (other fields can be inherited from folder)', 'Validation');
        return;
      }
    }

    // Validation for RDP
    if (session.sessionType === 'rdp') {
      if (!rdpHost.trim()) {
        await alertsStore.alert('RDP host is required', 'Validation');
        return;
      }
    }

    // Validation for VNC
    if (session.sessionType === 'vnc') {
      if (!vncHost.trim()) {
        await alertsStore.alert('VNC host is required', 'Validation');
        return;
      }
    }

    // Validation for Telnet
    if (session.sessionType === 'telnet') {
      if (!telnetHost.trim()) {
        await alertsStore.alert('Telnet host is required', 'Validation');
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
        if (session.sessionType === 'custom' && customCommand.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'command', customCommand.toString());
        }
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
        if (customCommand.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'command', customCommand.toString());
        }
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
      await alertsStore.alert('Failed to save session: ' + error, 'Error');
    }
  }

  function handleCancel() {
    onClose();
  }
</script>

{#if show && session}
  <Modal show={show} title={`Edit ${session.type === 'folder' ? 'Folder' : 'Session'}`} onClose={handleCancel} panelClass="w-96">
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
            <Tabs items={[{ id: 'connection', label: 'Connection' }, { id: 'session', label: 'Session' }]} active={activeTab} on:change={(e) => activeTab = e.detail as "session" | "connection" | "display" | "vnc"} />

            <!-- Tab Content -->
            {#if activeTab === 'connection'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-blue-400">SSH Connection</h4>

                <div class="grid grid-cols-2 gap-3">
                  <div class="col-span-2">
                    <LabeledInput id="ssh_host" label="Host *" bind:value={sshHost} placeholder="192.168.1.100 or example.com" />
                  </div>
                  <LabeledInput id="ssh_port" label="Port" bind:value={sshPort} placeholder={inheritedConfig.ssh_port ? `Inherited: ${inheritedConfig.ssh_port}` : '22'} inherited={inheritedConfig.ssh_port} />
                  <LabeledInput id="ssh_username" label="Username" bind:value={sshUsername} placeholder={inheritedConfig.ssh_username ? `Inherited: ${inheritedConfig.ssh_username}` : 'root'} inherited={inheritedConfig.ssh_username} />
                </div>
                <LabeledSelect id="ssh_auth_method" label="Authentication" bind:value={sshAuthMethod} options={[{ value: 'password', label: 'Password' }, { value: 'key', label: 'SSH Key' }]} />

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
                    {:else}"session" | "connection" | "display"
                      <p class="text-xs text-gray-500 mt-1">Path to your private key file</p>
                    {/if}
                  </div>
                {/if}
              </div>
            {:else if activeTab === 'session'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-purple-400">Session Configuration</h4>
                <TerminalSessionForm bind:workingDirectory={workingDirectory} bind:startupCommands={startupCommands} bind:environmentVariables={environmentVariables} inherited={inheritedConfig} rowsCommands={3} rowsEnv={3} />
              </div>
            {/if}
          {/if}

          <!-- RDP Session Configuration -->
          {#if session.type === 'session' && session.sessionType === 'rdp'}
            <Tabs items={[{ id: 'connection', label: 'Connection' }, { id: 'display', label: 'Display' }]} active={activeTab} on:change={(e) => activeTab = e.detail as "session" | "connection" | "display" | "vnc"} />

            <!-- Tab Content -->
            {#if activeTab === 'connection'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-cyan-400">RDP Connection</h4>

                <RDPConnectionForm bind:host={rdpHost} bind:port={rdpPort} bind:security={rdpSecurity} bind:username={rdpUsername} bind:password={rdpPassword} bind:domain={rdpDomain} />
              </div>
            {:else if activeTab === 'display'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-cyan-400">Display Settings</h4>

                <DisplaySettings bind:width={desktopWidth} bind:height={desktopHeight} bind:colorDepth={desktopColorDepth} />
              </div>
            {/if}
          {/if}

          <!-- VNC Session Configuration -->
          {#if session.type === 'session' && session.sessionType === 'vnc'}
            <Tabs items={[{ id: 'connection', label: 'Connection' }, { id: 'display', label: 'Display' }]} active={activeTab} on:change={(e) => activeTab = e.detail as "session" | "connection" | "display" | "vnc"} />

            <!-- Tab Content -->
            {#if activeTab === 'connection'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-green-400">VNC Connection</h4>

                <VNCConnectionForm bind:host={vncHost} bind:port={vncPort} bind:password={vncPassword} />
              </div>
            {:else if activeTab === 'display'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-green-400">Display Settings</h4>

                <DisplaySettings bind:width={desktopWidth} bind:height={desktopHeight} bind:colorDepth={desktopColorDepth} />
              </div>
            {/if}
          {/if}

          <!-- Telnet Session Configuration -->
          {#if session.type === 'session' && session.sessionType === 'telnet'}
            <Tabs items={[{ id: 'connection', label: 'Connection' }]} active={'connection'} />
            <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
              <h4 class="text-sm font-medium text-orange-400">Telnet Connection</h4>
              <TelnetConnectionForm bind:host={telnetHost} bind:port={telnetPort} bind:username={telnetUsername} bind:password={telnetPassword} />
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
              {#if session.sessionType === 'custom'}
                <LabeledInput id="custom_command" label="Custom Command" bind:value={customCommand} placeholder={inheritedConfig.command ? `Inherited: ${inheritedConfig.command}` : ''} inherited={inheritedConfig.command || ''} hint="Full path to the executable (no arguments)." />
              {/if}
              <TerminalSessionForm bind:workingDirectory={workingDirectory} bind:startupCommands={startupCommands} bind:environmentVariables={environmentVariables} inherited={inheritedConfig} rowsCommands={3} rowsEnv={3} />
            </div>
          {/if}

          {#if session.type === 'folder'}
            <div class="mb-2">
              <h4 class="text-sm font-medium text-green-400">Folder Configuration (Inherited by Children)</h4>
              <p class="text-xs text-gray-400">These settings will be inherited by all sessions and subfolders inside this folder.</p>
            </div>

            <Tabs items={[{ id: 'connection', label: 'SSH' }, { id: 'session', label: 'Terminal' }, { id: 'display', label: 'RDP' }, { id: 'vnc', label: 'VNC' }]} active={activeTab} on:change={(e) => activeTab = e.detail as "session" | "connection" | "display" | "vnc"} />

            <!-- Tab Content -->
            {#if activeTab === 'connection'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-blue-400">SSH Configuration</h4>
                <p class="text-xs text-gray-400">Settings inherited by SSH sessions</p>
                <SSHDefaultsForm bind:username={sshUsername} bind:port={sshPort} bind:keyPath={sshKeyPath} inherited={inheritedConfig} />
              </div>
            {:else if activeTab === 'session'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-purple-400">Terminal Configuration</h4>
                <LabeledInput id="custom_command_folder" label="Default Custom Command" bind:value={customCommand} placeholder={inheritedConfig.command ? `Inherited: ${inheritedConfig.command}` : ''} inherited={inheritedConfig.command || ''} hint="Set a default custom executable for child sessions (no arguments)." />
                <p class="text-xs text-gray-400">Settings inherited by terminal sessions (bash, zsh, fish, pwsh)</p>
                <TerminalSessionForm bind:workingDirectory={workingDirectory} bind:startupCommands={startupCommands} bind:environmentVariables={environmentVariables} inherited={inheritedConfig} />
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
                  <DisplaySettings bind:width={desktopWidth} bind:height={desktopHeight} bind:colorDepth={desktopColorDepth} inherited={inheritedConfig} />
                  <p class="text-xs text-gray-400 mt-2">These settings are shared with VNC sessions</p>
                </div>
              </div>
            {/if}
          {/if}
        

        <div class="flex gap-2 mt-6">
          <button
            onclick={handleSave}
            class="flex-1 px-4 py-2 rounded font-medium text-white"
            style="background: var(--accent-blue)"
          >
            Save
          </button>
          <button
            onclick={handleCancel}
            class="flex-1 px-4 py-2 rounded font-medium"
            style="background: var(--bg-tertiary)"
          >
            Cancel
          </button>
        </div>
      </div>
    {/if}
  </Modal>
{/if}
