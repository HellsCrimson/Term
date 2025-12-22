export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  const value = bytes / Math.pow(k, i);
  return `${value.toFixed(1)} ${sizes[i]}`;
}

export function formatRate(bytesPerInterval: number, intervalSeconds = 2): string {
  const perSecond = bytesPerInterval / Math.max(1, intervalSeconds);
  return `${formatBytes(perSecond)}/s`;
}

// Alias to mirror older helper naming
export const formatSize = formatBytes;

