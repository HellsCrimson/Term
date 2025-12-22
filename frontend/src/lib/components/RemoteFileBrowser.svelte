<script lang="ts">
  import type { TerminalTab } from '../stores/terminals.svelte';
  import { Dialogs, Events } from '@wailsio/runtime';
  import { LoggingService, SftpService } from '$bindings/term';
  import { formatBytes } from '$lib/utils/format';

  interface Props { tab: TerminalTab }
  let { tab }: Props = $props();

  interface FileEntry {
    name: string;
    path: string;
    size: number;
    mode: string;
    isDir: boolean;
    modTime: number; // unix
  }

  let currentPath = $state<string>('');
  let entries = $state<FileEntry[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);
  let selected: FileEntry | null = $state(null);
  // Progress state
  let uploading = $state(false);
  let uploadProgress = $state(0);
  let downloading = $state(false);
  let downloadLabel = $state('');

  async function list(path?: string) {
    loading = true;
    error = null;
    try {
      const res = await SftpService.HandleSSHFSList(tab.backendSessionId, path || currentPath || "");
      currentPath = res.remote_path || path || currentPath || "";
      entries = (res.files || []).sort((a, b) => Number(b.isDir) - Number(a.isDir) || a.name.localeCompare(b.name));
    } catch (e: any) {
      error = e.message || String(e);
    } finally {
      loading = false;
    }
  }

  // Use shared formatter

  function parentDir(pathStr: string): string {
    if (!pathStr || pathStr === '/') return '/';
    const parts = pathStr.split('/').filter(Boolean);
    parts.pop();
    return '/' + parts.join('/');
  }

  async function handleDownload(entry: FileEntry) {
    try {
      const suggested = entry.name;
      const dest = await Dialogs.SaveFile({ Filename: suggested });
      if (!dest) return;
      downloading = true;
      downloadLabel = `Downloading ${entry.name}…`;
      await SftpService.HandleSSHFSDownload(tab.backendSessionId, entry.path, dest);
    } catch (e: any) {
      error = e.message || String(e);
    } finally {
      downloading = false;
      downloadLabel = '';
    }
  }

  function genId(): string {
    // Simple unique-ish id for correlating SSE and upload request
    return 'upl-' + Date.now().toString(36) + '-' + Math.random().toString(36).slice(2);
  }

  async function chooseAndUpload() {
    const localPath = await Dialogs.OpenFile({});
    if (!localPath) return;
    const jobId = genId();

    uploading = true;
    uploadProgress = 0;
    error = null;

    try {
      Events.On(`sshfs-upload-progress-${jobId}`, (data) => {
        const d = data.data as { total?: number; transferred?: number; done?: boolean; error?: string }
        if (typeof d.total === 'number' && d.total > 0 && typeof d.transferred === 'number') {
          uploadProgress = Math.max(0, Math.min(100, Math.round((d.transferred / d.total) * 100)));
        }
        if (d.error) { error = d.error; }
        if (d.done) {
          uploading = false;
          list(currentPath);
          Events.Off(`sshfs-upload-progress-${jobId}`);
        }
      });

      SftpService.HandleSSHFSUpload(tab.backendSessionId, localPath, currentPath, jobId).then(() => {
        Events.Off(`sshfs-upload-progress-${jobId}`);
        uploading = false;
        uploadProgress = 0;
      })
    } catch (e: any) {
      error = e.message || String(e);
      uploading = false;
      uploadProgress = 0;
      Events.Off(`sshfs-upload-progress-${jobId}`);
    }
  }

  $effect(() => {
    // Load when tab becomes active and is ssh
    if (tab.active && tab.sessionType === 'ssh' && entries.length === 0 && !loading) {
      list();
    }
  });

  async function mkdirPrompt() {
    const name = prompt('New folder name');
    if (!name) return;
    await SftpService.HandleSSHFSMkdir(tab.backendSessionId, currentPath.endsWith('/') ? currentPath + name : currentPath + '/' + name, false);
    await list(currentPath);
  }

  async function deleteSelected() {
    if (!selected) return;
    if (!confirm(`Delete ${selected.isDir ? 'folder' : 'file'} "${selected.name}"?`)) return;
    await SftpService.HandleSSHFSDelete(tab.backendSessionId, selected.path);
    selected = null;
    await list(currentPath);
  }

  async function renameSelected() {
    if (!selected) return;
    const newName = prompt('Rename to', selected.name);
    if (!newName || newName === selected.name) return;
    const newPath = parentDir(selected.path).replace(/\/$/, '') + '/' + newName;
    await SftpService.HandleSSHFSRename(tab.backendSessionId, selected.path, newPath);
    await list(currentPath);
  }

  async function downloadDirSelected() {
    const dir = selected && selected.isDir ? selected.path : currentPath;
    const base = dir.split('/').filter(Boolean).pop() || 'archive';
    const dest = await Dialogs.SaveFile({ Filename: `${base}.zip` });
    if (!dest) return;
    downloading = true;
    downloadLabel = `Downloading ${base}.zip...`;
    try {
      await SftpService.HandleSSHFSDownloadDir(tab.backendSessionId, dir, dest);
    } catch (e: any) {
      error = e.message || String(e);
    } finally {
      downloading = false; downloadLabel = '';
    }
  }
</script>

<div class="border-t px-2 py-2 text-sm h-full" style="border-color: var(--border-color); color: var(--text-primary)">
  <div class="flex items-center gap-2 mb-2">
    <div class="flex-1 truncate" title={currentPath}>
      <span style="color: var(--text-muted)">Path:</span> {currentPath || '/'}
    </div>
    <button class="px-2 py-1 rounded text-white" style="background: var(--accent-blue)" onclick={() => list(currentPath)}>Refresh</button>
    <button class="px-2 py-1 rounded" style="background: var(--bg-tertiary)" onclick={() => list(parentDir(currentPath))}>Up</button>
    <button class="px-2 py-1 rounded text-white" style="background: var(--accent-green)" onclick={chooseAndUpload}>Upload</button>
    <button class="px-2 py-1 rounded" style="background: var(--bg-tertiary)" onclick={mkdirPrompt}>New Folder</button>
    <button class="px-2 py-1 rounded disabled:opacity-60" style="background: var(--bg-tertiary)" disabled={!selected} onclick={renameSelected}>Rename</button>
    <button class="px-2 py-1 rounded disabled:opacity-60 text-white" style="background: var(--accent-red)" disabled={!selected} onclick={deleteSelected}>Delete</button>
    <button class="px-2 py-1 rounded" style="background: var(--bg-tertiary)" onclick={downloadDirSelected}>Download Dir (ZIP)</button>
  </div>

  {#if loading}
    <div style="color: var(--text-muted)">Loading…</div>
  {:else if error}
    <div style="color: var(--accent-red)">Error: {error}</div>
  {:else}
    <div class="max-h-full overflow-auto rounded" style="border: 1px solid var(--border-color); background: var(--bg-secondary)">
      <table class="w-full text-xs">
        <thead style="position: sticky; top: 0; background: var(--bg-tertiary)">
          <tr>
            <th class="text-left px-2 py-1">Name</th>
            <th class="text-left px-2 py-1">Size</th>
            <th class="text-left px-2 py-1">Modified</th>
            <th class="text-left px-2 py-1">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each entries as e}
            <tr class="hover:bg-[var(--hover-bg)] cursor-pointer {selected === e ? 'outline outline-1' : ''}"
                onclick={() => selected = (selected === e ? null : e)}
            >
              <td class="px-2 py-1">
                {#if e.isDir}
                  <button class="underline" onclick={() => list(e.path)} title={e.path}>📁 {e.name}</button>
                {:else}
                  <span title={e.path}>📄 {e.name}</span>
                {/if}
              </td>
              <td class="px-2 py-1">{e.isDir ? '-' : formatBytes(e.size)}</td>
              <td class="px-2 py-1">{new Date(e.modTime * 1000).toLocaleString()}</td>
              <td class="px-2 py-1">
                {#if !e.isDir}
                  <button class="px-2 py-1 rounded text-white" style="background: var(--accent-blue)" onclick={() => handleDownload(e)}>Download</button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}

  <!-- Progress indicators -->
  {#if uploading}
    <div class="mt-2 text-xs">
      <div>Uploading… {uploadProgress}%</div>
      <div class="h-2 rounded overflow-hidden" style="background: var(--bg-tertiary)">
        <div class="h-full" style="background: var(--accent-blue); width: {uploadProgress}%"></div>
      </div>
    </div>
  {/if}
  {#if downloading}
    <div class="mt-2 text-xs" style="color: var(--text-muted)">{downloadLabel}</div>
  {/if}
</div>
