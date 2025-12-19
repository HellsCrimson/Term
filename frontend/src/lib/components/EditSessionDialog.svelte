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
  let sshAuthMethod = $state<'password' | 'key'>('password');
  let sshPassword = $state('');
  let sshKeyPath = $state('');
  let loading = $state(false);

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
      const config = await sessionsStore.getEffectiveConfig(session.id);

      sshHost = config.ssh_host || '';
      sshPort = config.ssh_port || '22';
      sshUsername = config.ssh_username || '';
      sshAuthMethod = (config.ssh_auth_method as 'password' | 'key') || 'password';
      sshPassword = config.ssh_password || '';
      sshKeyPath = config.ssh_key_path || '';
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
    if (session.sessionType === 'ssh') {
      if (!sshHost.trim() || !sshUsername.trim()) {
        alert('SSH host and username are required');
        return;
      }
      if (sshAuthMethod === 'password' && !sshPassword) {
        alert('SSH password is required');
        return;
      }
      if (sshAuthMethod === 'key' && !sshKeyPath.trim()) {
        alert('SSH key path is required');
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

      // Save SSH config if SSH session
      if (session.sessionType === 'ssh') {
        await sessionsStore.setSessionConfig(session.id, 'ssh_host', sshHost.toString());
        await sessionsStore.setSessionConfig(session.id, 'ssh_port', sshPort.toString());
        await sessionsStore.setSessionConfig(session.id, 'ssh_username', sshUsername.toString());
        await sessionsStore.setSessionConfig(session.id, 'ssh_auth_method', sshAuthMethod.toString());

        if (sshAuthMethod === 'password') {
          await sessionsStore.setSessionConfig(session.id, 'ssh_password', sshPassword.toString());
        } else {
          await sessionsStore.setSessionConfig(session.id, 'ssh_key_path', sshKeyPath.toString());
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
            <label class="block text-sm font-medium mb-1">Name</label>
            <input
              type="text"
              bind:value={sessionName}
              class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
              placeholder="My Terminal"
            />
          </div>

          {#if session.sessionType === 'ssh'}
            <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
              <h4 class="text-sm font-medium text-blue-400">SSH Connection</h4>

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
                  <label class="block text-xs font-medium mb-1">Username *</label>
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
                  <label class="block text-xs font-medium mb-1">Password *</label>
                  <input
                    type="password"
                    bind:value={sshPassword}
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    placeholder="••••••••"
                  />
                </div>
              {:else}
                <div>
                  <label class="block text-xs font-medium mb-1">Key Path *</label>
                  <input
                    type="text"
                    bind:value={sshKeyPath}
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    placeholder="~/.ssh/id_rsa"
                  />
                  <p class="text-xs text-gray-500 mt-1">Path to your private key file</p>
                </div>
              {/if}
            </div>
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
