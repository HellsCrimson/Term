import { writable, get } from 'svelte/store';

export type AlertType = 'alert' | 'confirm';

export interface AlertItem {
  id: number;
  type: AlertType;
  title: string;
  message: string;
  resolve: (value: any) => void;
}

const queue = writable<AlertItem[]>([]);

export const alertsStore = {
  subscribe: queue.subscribe,
  // Show a simple OK alert. Returns a promise that resolves when dismissed.
  async alert(message: string, title = 'Notice'): Promise<void> {
    return new Promise<void>((resolve) => {
      const id = Date.now() + Math.floor(Math.random() * 1000);
      queue.update((q) => [...q, { id, type: 'alert', title, message, resolve }]);
    });
  },
  // Show a confirm dialog. Resolves to true for confirm, false for cancel.
  async confirm(message: string, title = 'Confirm'): Promise<boolean> {
    return new Promise<boolean>((resolve) => {
      const id = Date.now() + Math.floor(Math.random() * 1000);
      queue.update((q) => [...q, { id, type: 'confirm', title, message, resolve }]);
    });
  },
  // Dismiss the current alert with a value (true/false/void)
  dismissCurrent(value?: any) {
    const current = get(queue)[0];
    if (current) {
      current.resolve(value);
      queue.update((q) => q.slice(1));
    }
  }
};

