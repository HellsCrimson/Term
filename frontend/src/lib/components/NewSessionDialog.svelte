<script lang="ts">
    import { LoggingService } from '$bindings/term';
  import { sessionsStore } from '../stores/sessions.svelte';
  import Modal from './common/Modal.svelte';
  import Tabs from './common/Tabs.svelte';
  import LabeledInput from './common/LabeledInput.svelte';
  import LabeledSelect from './common/LabeledSelect.svelte';
  import LabeledTextarea from './common/LabeledTextarea.svelte';
  import DisplaySettings from './common/DisplaySettings.svelte';
  import RDPConnectionForm from './common/RDPConnectionForm.svelte';
  import VNCConnectionForm from './common/VNCConnectionForm.svelte';
  import TelnetConnectionForm from './common/TelnetConnectionForm.svelte';
  import TerminalSessionForm from './common/TerminalSessionForm.svelte';

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
            <Tabs items={[{ id: 'connection', label: 'Connection' }, { id: 'session', label: 'Session' }]} active={activeTab} on:change={(e) => activeTab = e.detail as "session" | "connection" | "display"} />

            <!-- Tab Content -->
            {#if activeTab === 'connection'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <div class="flex items-center justify-between">
                  <h4 class="text-sm font-medium text-blue-400">SSH Connection</h4>
                  <p class="text-xs text-gray-400">* Only host is required</p>
                </div>
                <div class="grid grid-cols-2 gap-3">
                  <div class="col-span-2">
                    <LabeledInput id="ssh_host" label="Host *" bind:value={sshHost} placeholder="192.168.1.100 or example.com" />
                  </div>
                  <LabeledInput id="ssh_port" label="Port" bind:value={sshPort} placeholder="22" />
                  <LabeledInput id="ssh_username" label="Username" bind:value={sshUsername} placeholder="root" />
                </div>
                <LabeledSelect id="ssh_auth_method" label="Authentication" bind:value={sshAuthMethod} options={[{ value: 'password', label: 'Password' }, { value: 'key', label: 'SSH Key' }]} />
                {#if sshAuthMethod === 'password'}
                  <LabeledInput id="ssh_password" label="Password" type="password" bind:value={sshPassword} placeholder="••••••••" />
                {:else}
                  <LabeledInput id="ssh_key_path" label="Key Path" bind:value={sshKeyPath} placeholder="~/.ssh/id_rsa" />
                {/if}
                <p class="text-xs text-gray-400 mt-2 pt-2 border-t border-gray-600">💡 Tip: Leave fields empty to inherit values from the parent folder</p>
              </div>
            {:else if activeTab === 'session'}
              <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
                <h4 class="text-sm font-medium text-purple-400">Session Configuration</h4>
                <TerminalSessionForm bind:workingDirectory={workingDirectory} bind:startupCommands={startupCommands} bind:environmentVariables={environmentVariables} />
                <p class="text-xs text-gray-400 mt-2 pt-2 border-t border-gray-600">💡 Tip: Leave fields empty to inherit values from the parent folder</p>
              </div>
            {/if}
          {/if}

          {#if sessionType === 'rdp'}
            <Tabs items={[{ id: 'connection', label: 'Connection' }, { id: 'display', label: 'Display' }]} active={activeTab} on:change={(e) => activeTab = e.detail as "session" | "connection" | "display"} />

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

          {#if sessionType === 'vnc'}
            <Tabs items={[{ id: 'connection', label: 'Connection' }, { id: 'display', label: 'Display' }]} active={activeTab} on:change={(e) => activeTab = e.detail as "session" | "connection" | "display"} />

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

          {#if sessionType === 'telnet'}
            <Tabs items={[{ id: 'connection', label: 'Connection' }]} active={'connection'} />
            <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
              <h4 class="text-sm font-medium text-orange-400">Telnet Connection</h4>
              <TelnetConnectionForm bind:host={telnetHost} bind:port={telnetPort} bind:username={telnetUsername} bind:password={telnetPassword} />
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
                <TerminalSessionForm bind:workingDirectory={workingDirectory} bind:startupCommands={startupCommands} bind:environmentVariables={environmentVariables} />
                <p class="text-xs text-gray-400 mt-2 pt-2 border-t border-gray-600">💡 Tip: Leave fields empty to inherit values from the parent folder</p>
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
