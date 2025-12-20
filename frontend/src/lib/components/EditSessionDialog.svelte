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
  let workingDirectory = $state('');
  let startupCommands = $state('');
  let environmentVariables = $state('');
  let loading = $state(false);
  let inheritedConfig = $state<Record<string, string>>({});

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

    try {
      // Update session name if changed
      if (sessionName !== session.name) {
        await sessionsStore.updateSession({
          ...session,
          name: sessionName.trim()
        });
      }

      // Save general session config (for all session types)
      if (session.type === 'session') {
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

      // Save folder config (inherited by children)
      if (session.type === 'folder') {
        // Only save non-empty values
        if (sshUsername.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'ssh_username', sshUsername.toString());
        }
        if (sshPort.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'ssh_port', sshPort.toString());
        }
        if (sshKeyPath.trim()) {
          await sessionsStore.setSessionConfig(session.id, 'ssh_key_path', sshKeyPath.toString());
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
                    placeholder={inheritedConfig.ssh_port ? `Inherited: ${inheritedConfig.ssh_port}` : '22'}
                  />
                  {#if inheritedConfig.ssh_port && !sshPort}
                    <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inheritedConfig.ssh_port}</p>
                  {/if}
                </div>

                <div>
                  <label class="block text-xs font-medium mb-1">Username</label>
                  <input
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
                  <label class="block text-xs font-medium mb-1">Key Path</label>
                  <input
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
          {/if}

          <!-- Working Directory (for all session types) -->
          {#if session.type === 'session'}
            <div class="space-y-2">
              <label class="block text-sm font-medium">Working Directory</label>
              <input
                type="text"
                bind:value={workingDirectory}
                class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                placeholder={inheritedConfig.working_directory ? `Inherited: ${inheritedConfig.working_directory}` : '~/projects or /home/user'}
              />
              {#if inheritedConfig.working_directory && !workingDirectory}
                <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inheritedConfig.working_directory}</p>
              {:else}
                <p class="text-xs text-gray-400">Directory where the session will start (supports ~ for home)</p>
              {/if}
            </div>

            <div class="space-y-2">
              <label class="block text-sm font-medium">Startup Commands</label>
              <textarea
                bind:value={startupCommands}
                rows="3"
                class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500 font-mono text-sm"
                placeholder={inheritedConfig.startup_commands ? `Inherited: ${inheritedConfig.startup_commands}` : 'cd ~/project; source .env'}
              ></textarea>
              {#if inheritedConfig.startup_commands && !startupCommands}
                <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inheritedConfig.startup_commands}</p>
              {:else}
                <p class="text-xs text-gray-400">Commands to run when the session starts (separated by semicolons)</p>
              {/if}
            </div>

            <div class="space-y-2">
              <label class="block text-sm font-medium">Environment Variables</label>
              <textarea
                bind:value={environmentVariables}
                rows="3"
                class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500 font-mono text-sm"
                placeholder={inheritedConfig.environment_variables ? `Inherited: ${inheritedConfig.environment_variables}` : 'KEY1=value1; KEY2=value2'}
              ></textarea>
              {#if inheritedConfig.environment_variables && !environmentVariables}
                <p class="text-xs text-yellow-400 mt-1">↓ Inherited: {inheritedConfig.environment_variables}</p>
              {:else}
                <p class="text-xs text-gray-400">Environment variables (KEY=value; separated by semicolons)</p>
              {/if}
            </div>
          {/if}

          {#if session.type === 'folder'}
            <div class="space-y-3 p-3 bg-gray-700/50 rounded border border-gray-600">
              <h4 class="text-sm font-medium text-green-400">Folder Configuration (Inherited by Children)</h4>
              <p class="text-xs text-gray-400">These settings will be inherited by all sessions and subfolders inside this folder.</p>

              <div class="space-y-3">
                <div>
                  <label class="block text-xs font-medium mb-1">SSH Username</label>
                  <input
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
                  <label class="block text-xs font-medium mb-1">SSH Port</label>
                  <input
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
                  <label class="block text-xs font-medium mb-1">SSH Key Path</label>
                  <input
                    type="text"
                    bind:value={sshKeyPath}
                    class="w-full px-2 py-1.5 text-sm bg-gray-700 border border-gray-600 rounded focus:outline-none focus:border-blue-500"
                    placeholder={inheritedConfig.ssh_key_path ? `Inherited: ${inheritedConfig.ssh_key_path}` : '~/.ssh/id_rsa (inherited by SSH sessions)'}
                  />
                  {#if inheritedConfig.ssh_key_path && !sshKeyPath}
                    <p class="text-xs text-yellow-400 mt-1">↓ Inherited from parent: {inheritedConfig.ssh_key_path}</p>
                  {/if}
                </div>

                <div>
                  <label class="block text-xs font-medium mb-1">Working Directory</label>
                  <input
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
                  <label class="block text-xs font-medium mb-1">Startup Commands</label>
                  <textarea
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
                  <label class="block text-xs font-medium mb-1">Environment Variables</label>
                  <textarea
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
