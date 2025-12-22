<script lang="ts">
    import { LoggingService } from '$bindings/term';
  import { sessionsStore } from '../stores/sessions.svelte';
  import Modal from './common/Modal.svelte';

  interface Props {
    show: boolean;
    onClose: () => void;
    defaultType?: 'folder' | 'session';
    defaultParentId?: string;
  }

  let { show, onClose, defaultType, defaultParentId }: Props = $props();

  let itemType = $derived<'folder' | 'session'>(defaultType || 'session');
  let sessionName = $state('');
  let sessionType = $state<'ssh' | 'bash' | 'zsh' | 'fish' | 'pwsh' | 'rdp' | 'vnc' | 'telnet'>('bash');
  let parentId = $derived<string | null>(defaultParentId || null);

  // SSH-specific fields
  let sshHost = $state('');
  let sshPort = $state('22');
  let sshUsername = $state('');
  let sshAuthMethod = $state<'password' | 'key'>('key');
  let sshPassword = $state('');
  let sshKeyPath = $state('');

  // General session fields
  let workingDirectory = $state('');
  let startupCommands = $state('');
  let environmentVariables = $state('');

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
  let activeTab = $state<'connection' | 'session' | 'display'>('connection');

  // Reset active tab when session type changes
  $effect(() => {
    if (itemType === 'session') {
      activeTab = 'connection';
    }
  });

  async function handleCreate() {
    LoggingService.Log(`[NewSessionDialog] handleCreate called: itemType=${itemType}, sessionType=${sessionType}, sessionName=${sessionName}`, "DEBUG");

    if (!sessionName.trim()) {
      LoggingService.Log(`[NewSessionDialog] Aborting - empty session name`, "DEBUG");
      return;
    }

    // Basic validation for SSH - only host is truly required
    // (username, auth, etc. can be inherited from parent folder)
    if (itemType === 'session' && sessionType === 'ssh') {
      if (!sshHost.trim()) {
        alert('SSH host is required (other fields can be inherited from folder)');
        return;
      }
    }

    // Validation for RDP
    if (itemType === 'session' && sessionType === 'rdp') {
      if (!rdpHost.trim()) {
        alert('RDP host is required');
        return;
      }
    }

    // Validation for VNC
    if (itemType === 'session' && sessionType === 'vnc') {
      if (!vncHost.trim()) {
        alert('VNC host is required');
        return;
      }
    }

    // Validation for Telnet
    if (itemType === 'session' && sessionType === 'telnet') {
      if (!telnetHost.trim()) {
        alert('Telnet host is required');
        return;
      }
    }

    try {
      const newItem = {
        id: `${itemType}-${Date.now()}`,
        parentId: parentId,
        name: sessionName,
        type: itemType,
        sessionType: itemType === 'session' ? sessionType : undefined,
        position: 0,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString()
      };

      await sessionsStore.createSession(newItem);

      // Save SSH config if SSH session (only save non-empty values)
      if (itemType === 'session' && sessionType === 'ssh') {
        const sessionId = newItem.id;

        // Host is required
        if (sshHost.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'ssh_host', sshHost.toString());
        }

        // Optional fields (can be inherited from parent folder)
        if (sshPort.trim() && sshPort !== '22') {
          await sessionsStore.setSessionConfig(sessionId, 'ssh_port', sshPort.toString());
        }
        if (sshUsername.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'ssh_username', sshUsername.toString());
        }

        // Save auth method (always, since it's a user choice)
        await sessionsStore.setSessionConfig(sessionId, 'ssh_auth_method', sshAuthMethod.toString());

        // Save credentials only if provided
        if (sshAuthMethod === 'password' && sshPassword.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'ssh_password', sshPassword.toString());
        } else if (sshAuthMethod === 'key' && sshKeyPath.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'ssh_key_path', sshKeyPath.toString());
        }
      }

      // Save RDP config if RDP session
      if (itemType === 'session' && sessionType === 'rdp') {
        const sessionId = newItem.id;

        if (rdpHost.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'rdp_host', rdpHost.toString());
        }
        if (rdpPort.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'rdp_port', rdpPort.toString());
        }
        if (rdpUsername.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'rdp_username', rdpUsername.toString());
        }
        if (rdpPassword.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'rdp_password', rdpPassword.toString());
        }
        if (rdpDomain.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'rdp_domain', rdpDomain.toString());
        }
        await sessionsStore.setSessionConfig(sessionId, 'rdp_security', rdpSecurity.toString());
        await sessionsStore.setSessionConfig(sessionId, 'desktop_width', desktopWidth.toString());
        await sessionsStore.setSessionConfig(sessionId, 'desktop_height', desktopHeight.toString());
        await sessionsStore.setSessionConfig(sessionId, 'desktop_color_depth', desktopColorDepth.toString());
      }

      // Save VNC config if VNC session
      if (itemType === 'session' && sessionType === 'vnc') {
        const sessionId = newItem.id;

        if (vncHost.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'vnc_host', vncHost.toString());
        }
        if (vncPort.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'vnc_port', vncPort.toString());
        }
        if (vncPassword.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'vnc_password', vncPassword.toString());
        }
        await sessionsStore.setSessionConfig(sessionId, 'desktop_width', desktopWidth.toString());
        await sessionsStore.setSessionConfig(sessionId, 'desktop_height', desktopHeight.toString());
        await sessionsStore.setSessionConfig(sessionId, 'desktop_color_depth', desktopColorDepth.toString());
      }

      // Save Telnet config if Telnet session
      if (itemType === 'session' && sessionType === 'telnet') {
        const sessionId = newItem.id;

        if (telnetHost.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'telnet_host', telnetHost.toString());
        }
        if (telnetPort.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'telnet_port', telnetPort.toString());
        }
        if (telnetUsername.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'telnet_username', telnetUsername.toString());
        }
        if (telnetPassword.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'telnet_password', telnetPassword.toString());
        }
      }

      // Save general session config (for terminal session types only)
      if (itemType === 'session' && !['rdp', 'vnc', 'telnet'].includes(sessionType)) {
        const sessionId = newItem.id;

        if (workingDirectory.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'working_directory', workingDirectory.toString());
        }
        if (startupCommands.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'startup_commands', startupCommands.toString());
        }
        if (environmentVariables.trim()) {
          await sessionsStore.setSessionConfig(sessionId, 'environment_variables', environmentVariables.toString());
        }
      }

      resetForm();
      onClose();
    } catch (error) {
      LoggingService.Log(`Failed to create ${itemType}: ${error}`, "ERROR");
      alert(`Failed to create ${itemType}: ` + error);
    }
  }

  function handleCancel() {
    resetForm();
    onClose();
  }

  function resetForm() {
    itemType = 'session';
    sessionName = '';
    sessionType = 'bash';
    parentId = null;
    sshHost = '';
    sshPort = '22';
    sshUsername = '';
    sshAuthMethod = 'password';
    sshPassword = '';
    sshKeyPath = '';
    workingDirectory = '';
    startupCommands = '';
    environmentVariables = '';
    rdpHost = '';
    rdpPort = '3389';
    rdpUsername = '';
    rdpPassword = '';
    rdpDomain = '';
    rdpSecurity = 'any';
    vncHost = '';
    vncPort = '5900';
    vncPassword = '';
    telnetHost = '';
    telnetPort = '23';
    telnetUsername = '';
    telnetPassword = '';
    desktopWidth = '1920';
    desktopHeight = '1080';
    desktopColorDepth = '16';
  }
</script>

<Modal show={show} title={`Create New ${itemType === 'folder' ? 'Folder' : 'Session'}`} onClose={handleCancel} panelClass="w-96">
  <div class="space-y-4">
        <div>
          <label for="item_type" class="block text-sm font-medium mb-1">Type</label>
          <select
            id="item_type"
            bind:value={itemType}
            class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
          >
            <option value="session">Session</option>
            <option value="folder">Folder</option>
          </select>
        </div>

        <div>
          <label for="session_name" class="block text-sm font-medium mb-1">Name</label>
          <!-- svelte-ignore a11y_autofocus -->
          <input
            id="session_name"
            type="text"
            bind:value={sessionName}
            class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
            placeholder={itemType === 'folder' ? 'My Folder' : 'My Terminal'}
            autofocus
          />
        </div>

        {#if itemType === 'session'}
          <div>
            <label for="session_type" class="block text-sm font-medium mb-1">Session Type</label>
            <select
              id="session_type"
              bind:value={sessionType}
              class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
            >
              <optgroup label="Terminal">
                <option value="bash">Bash</option>
                <option value="zsh">Zsh</option>
                <option value="fish">Fish</option>
                <option value="pwsh">PowerShell</option>
                <option value="ssh">SSH</option>
              </optgroup>
              <optgroup label="Remote Desktop">
                <option value="rdp">RDP (Remote Desktop)</option>
                <option value="vnc">VNC</option>
              </optgroup>
              <optgroup label="Other">
                <option value="telnet">Telnet</option>
              </optgroup>
            </select>
          </div>

          {#if sessionType === 'ssh'}
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
                <div class="flex items-center justify-between">
                  <h4 class="text-sm font-medium text-blue-400">SSH Connection</h4>
                  <p class="text-xs text-gray-400">* Only host is required</p>
                </div>

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
                      placeholder="22"
                    />
                  </div>

                  <div>
                    <label for="ssh_username" class="block text-xs font-medium mb-1">Username</label>
                    <input
                      id="ssh_username"
                      type="text"
                      bind:value={sshUsername}
                      class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                      placeholder="root"
                    />
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
                    <label for="ssh_password" class="block text-xs font-medium mb-1">Password</label>
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
                      placeholder="~/.ssh/id_rsa"
                    />
                    <p class="text-xs text-gray-500 mt-1">Path to your private key file</p>
                  </div>
                {/if}

                <p class="text-xs text-gray-400 mt-2 pt-2 border-t border-gray-600">
                  💡 Tip: Leave fields empty to inherit values from the parent folder
                </p>
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
                    placeholder="~/projects or /home/user"
                  />
                  <p class="text-xs text-gray-500 mt-1">Directory where the session will start (supports ~)</p>
                </div>

                <div>
                  <label for="startup_commands" class="block text-xs font-medium mb-1">Startup Commands</label>
                  <textarea
                    id="startup_commands"
                    bind:value={startupCommands}
                    rows="2"
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500 font-mono"
                    placeholder="cd ~/project; source .env"
                  ></textarea>
                  <p class="text-xs text-gray-500 mt-1">Commands to run when session starts (separated by semicolons)</p>
                </div>

                <div>
                  <label for="environment_variables" class="block text-xs font-medium mb-1">Environment Variables</label>
                  <textarea
                    id="environment_variables"
                    bind:value={environmentVariables}
                    rows="2"
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500 font-mono"
                    placeholder="KEY1=value1; KEY2=value2"
                  ></textarea>
                  <p class="text-xs text-gray-500 mt-1">Environment variables (KEY=value; separated by semicolons)</p>
                </div>

                <p class="text-xs text-gray-400 mt-2 pt-2 border-t border-gray-600">
                  💡 Tip: Leave fields empty to inherit values from the parent folder
                </p>
              </div>
            {/if}
          {/if}

          {#if sessionType === 'rdp'}
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
                    <label for="rdp_host" class="block text-xs font-medium mb-1">Port</label>
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

          {#if sessionType === 'vnc'}
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

          {#if sessionType === 'telnet'}
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

          <!-- General session configuration (for terminal session types only) -->
          {#if !['ssh', 'rdp', 'vnc', 'telnet'].includes(sessionType)}
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
                  placeholder="~/projects or /home/user"
                />
                <p class="text-xs text-gray-500 mt-1">Directory where the session will start (supports ~)</p>
              </div>

              <div>
                <label for="startup_commands" class="block text-xs font-medium mb-1">Startup Commands</label>
                <textarea
                  id="startup_commands"
                  bind:value={startupCommands}
                  rows="2"
                  class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500 font-mono"
                  placeholder="cd ~/project; source .env"
                ></textarea>
                <p class="text-xs text-gray-500 mt-1">Commands to run when session starts (separated by semicolons)</p>
              </div>

              <div>
                <label for="environment_variables" class="block text-xs font-medium mb-1">Environment Variables</label>
                <textarea
                  id="environment_variables"
                  bind:value={environmentVariables}
                  rows="2"
                  class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500 font-mono"
                  placeholder="KEY1=value1; KEY2=value2"
                ></textarea>
                <p class="text-xs text-gray-500 mt-1">Environment variables (KEY=value; separated by semicolons)</p>
              </div>

              <p class="text-xs text-gray-400 mt-2 pt-2 border-t border-gray-600">
                💡 Tip: Leave fields empty to inherit values from the parent folder
              </p>
            </div>
          {/if}
        {/if}
      </div>

    <div class="flex gap-2 mt-6">
      <button
        onclick={handleCreate}
        class="flex-1 px-4 py-2 rounded font-medium text-white"
        style="background: var(--accent-blue)"
      >
        Create
      </button>
      <button
        onclick={handleCancel}
        class="flex-1 px-4 py-2 rounded font-medium"
        style="background: var(--bg-tertiary)"
      >
        Cancel
      </button>
    </div>
</Modal>
