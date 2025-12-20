<script lang="ts">
  import { sessionsStore } from '../stores/sessions.svelte';

  interface Props {
    show: boolean;
    onClose: () => void;
    defaultType?: 'folder' | 'session';
    defaultParentId?: string;
  }

  let { show, onClose, defaultType, defaultParentId }: Props = $props();

  let itemType = $state<'folder' | 'session'>(defaultType || 'session');
  let sessionName = $state('');
  let sessionType = $state<'ssh' | 'bash' | 'zsh' | 'fish' | 'pwsh'>('bash');
  let parentId = $state<string | null>(defaultParentId || null);

  // Reset when dialog opens
  $effect(() => {
    if (show) {
      itemType = defaultType || 'session';
      parentId = defaultParentId || null;
      sessionName = '';
      sessionType = 'bash';
      sshHost = '';
      sshPort = '22';
      sshUsername = '';
      sshAuthMethod = 'password';
      sshPassword = '';
      sshKeyPath = '';
      workingDirectory = '';
      startupCommands = '';
      environmentVariables = '';
    }
  });

  // SSH-specific fields
  let sshHost = $state('');
  let sshPort = $state('22');
  let sshUsername = $state('');
  let sshAuthMethod = $state<'password' | 'key'>('password');
  let sshPassword = $state('');
  let sshKeyPath = $state('');

  // General session fields
  let workingDirectory = $state('');
  let startupCommands = $state('');
  let environmentVariables = $state('');

  async function handleCreate() {
    if (!sessionName.trim()) {
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

      // Save general session config (for all session types)
      if (itemType === 'session') {
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
      console.error(`Failed to create ${itemType}:`, error);
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
  }

  // Helper to check if we need setSessionConfig
  async function setSessionConfig(sessionId: string, key: string, value: string) {
    try {
      await sessionsStore.setSessionConfig(sessionId, key, value);
    } catch (error) {
      console.error(`Failed to set config ${key}:`, error);
    }
  }

  // Alias for cleaner code
  const setConfig = setSessionConfig;
</script>

{#if show}
  <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
    <div class="bg-gray-800 rounded-lg p-6 w-96 border border-gray-700">
      <h2 class="text-xl font-semibold mb-4">
        Create New {itemType === 'folder' ? 'Folder' : 'Session'}
      </h2>

      <div class="space-y-4">
        <div>
          <label class="block text-sm font-medium mb-1">Type</label>
          <select
            bind:value={itemType}
            class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
          >
            <option value="session">Session</option>
            <option value="folder">Folder</option>
          </select>
        </div>

        <div>
          <label class="block text-sm font-medium mb-1">Name</label>
          <input
            type="text"
            bind:value={sessionName}
            class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
            placeholder={itemType === 'folder' ? 'My Folder' : 'My Terminal'}
            autofocus
          />
        </div>

        {#if itemType === 'session'}
          <div>
            <label class="block text-sm font-medium mb-1">Session Type</label>
            <select
              bind:value={sessionType}
              class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
            >
              <option value="bash">Bash</option>
              <option value="zsh">Zsh</option>
              <option value="fish">Fish</option>
              <option value="pwsh">PowerShell</option>
              <option value="ssh">SSH</option>
            </select>
          </div>

          {#if sessionType === 'ssh'}
            <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
              <div class="flex items-center justify-between">
                <h4 class="text-sm font-medium text-blue-400">SSH Connection</h4>
                <p class="text-xs text-gray-400">* Only host is required</p>
              </div>

              <div class="grid grid-cols-2 gap-3">
                <div class="col-span-2">
                  <label class="block text-xs font-medium mb-1">Host *</label>
                  <input
                    type="text"
                    bind:value={sshHost}
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    placeholder="192.168.1.100 or example.com"
                  />
                </div>

                <div>
                  <label class="block text-xs font-medium mb-1">Port</label>
                  <input
                    type="text"
                    bind:value={sshPort}
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    placeholder="22"
                  />
                </div>

                <div>
                  <label class="block text-xs font-medium mb-1">Username</label>
                  <input
                    type="text"
                    bind:value={sshUsername}
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    placeholder="root"
                  />
                </div>
              </div>

              <div>
                <label class="block text-xs font-medium mb-1">Authentication</label>
                <select
                  bind:value={sshAuthMethod}
                  class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                >
                  <option value="password">Password</option>
                  <option value="key">SSH Key</option>
                </select>
              </div>

              {#if sshAuthMethod === 'password'}
                <div>
                  <label class="block text-xs font-medium mb-1">Password</label>
                  <input
                    type="password"
                    bind:value={sshPassword}
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    placeholder="••••••••"
                  />
                </div>
              {:else}
                <div>
                  <label class="block text-xs font-medium mb-1">Key Path</label>
                  <input
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
          {/if}

          <!-- General session configuration (for all session types) -->
          <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
            <h4 class="text-sm font-medium text-purple-400">Session Configuration</h4>

            <div>
              <label class="block text-xs font-medium mb-1">Working Directory</label>
              <input
                type="text"
                bind:value={workingDirectory}
                class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                placeholder="~/projects or /home/user"
              />
              <p class="text-xs text-gray-500 mt-1">Directory where the session will start (supports ~)</p>
            </div>

            <div>
              <label class="block text-xs font-medium mb-1">Startup Commands</label>
              <textarea
                bind:value={startupCommands}
                rows="2"
                class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500 font-mono"
                placeholder="cd ~/project; source .env"
              ></textarea>
              <p class="text-xs text-gray-500 mt-1">Commands to run when session starts (separated by semicolons)</p>
            </div>

            <div>
              <label class="block text-xs font-medium mb-1">Environment Variables</label>
              <textarea
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
      </div>

      <div class="flex gap-2 mt-6">
        <button
          onclick={handleCreate}
          class="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded font-medium"
        >
          Create
        </button>
        <button
          onclick={handleCancel}
          class="flex-1 px-4 py-2 bg-gray-600 hover:bg-gray-700 rounded font-medium"
        >
          Cancel
        </button>
      </div>
    </div>
  </div>
{/if}
