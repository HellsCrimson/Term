<script lang="ts">
  import type { TerminalTab } from '../stores/terminals.svelte';
  import { Dialogs } from '@wailsio/runtime';

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

  async function list(path?: string) {
    loading = true; error = null;
    try {
      const url = new URL(`http://localhost:3000/api/sshfs/list/${encodeURIComponent(tab.backendSessionId)}`);
      if (path) url.searchParams.set('path', path);
      const res = await fetch(url.toString(), { method: 'GET' });
      if (!res.ok) throw new Error(await res.text());
      const payload = await res.json() as { path: string; entries: FileEntry[] };
      currentPath = payload.path || path || currentPath || '/';
      entries = (payload.entries || []).sort((a, b) => Number(b.isDir) - Number(a.isDir) || a.name.localeCompare(b.name));
    } catch (e: any) {
      error = e.message || String(e);
    } finally {
      loading = false;
    }
  }

  function formatSize(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return (bytes / Math.pow(k, i)).toFixed(1) + ' ' + sizes[i];
  }

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
      const url = new URL(`http://localhost:3000/api/sshfs/save/${encodeURIComponent(tab.backendSessionId)}`);
      const res = await fetch(url.toString(), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: entry.path, dest })
      });
      if (!res.ok) throw new Error(await res.text());
    } catch (e: any) {
      error = e.message || String(e);
    }
  }

  async function handleUpload(e: Event) {
    const input = e.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;
    const file = input.files[0];
    const form = new FormData();
    form.append('file', file, file.name);
    const url = new URL(`http://localhost:3000/api/sshfs/upload/${encodeURIComponent(tab.backendSessionId)}`);
    url.searchParams.set('dir', currentPath || '/');
    try {
      const res = await fetch(url.toString(), { method: 'POST', body: form });
      if (!res.ok) throw new Error(await res.text());
      await list(currentPath);
    } catch (e: any) {
      error = e.message || String(e);
    } finally {
      input.value = '';
    }
  }

  $effect(() => {
    // Load when tab becomes active and is ssh
    if (tab.active && tab.sessionType === 'ssh' && entries.length === 0 && !loading) {
      list();
    }
  });
</script>

<div class="border-t px-2 py-2 text-sm h-full" style="border-color: var(--border-color); color: var(--text-primary)">
  <div class="flex items-center gap-2 mb-2">
    <div class="flex-1 truncate" title={currentPath}>
      <span style="color: var(--text-muted)">Path:</span> {currentPath || '/'}
    </div>
    <button class="px-2 py-1 rounded text-white" style="background: var(--accent-blue)" onclick={() => list(currentPath)}>Refresh</button>
    <button class="px-2 py-1 rounded" style="background: var(--bg-tertiary)" onclick={() => list(parentDir(currentPath))}>Up</button>
    <label class="px-2 py-1 rounded cursor-pointer text-white" style="background: var(--accent-green)">
      Upload<input type="file" class="hidden" onchange={handleUpload} />
    </label>
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
            <tr class="hover:bg-[var(--hover-bg)]">
              <td class="px-2 py-1">
                {#if e.isDir}
                  <button class="underline" onclick={() => list(e.path)} title={e.path}>📁 {e.name}</button>
                {:else}
                  <span title={e.path}>📄 {e.name}</span>
                {/if}
              </td>
              <td class="px-2 py-1">{e.isDir ? '-' : formatSize(e.size)}</td>
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
</div>
