interface AppNotification {
  id: number;
  message: string;
  type: 'success' | 'info' | 'error';
}

let _counter = 0;
let notifications = $state<AppNotification[]>([]);
let pendingNotifications = $state<AppNotification[]>([]);

export const notificationStore = {
  get items() { return notifications; },

  show(message: string, type: 'success' | 'info' | 'error' = 'info') {
    if (!document.hasFocus()) {
      pendingNotifications.push({ id: ++_counter, message, type });
      return;
    }
    const id = ++_counter;
    notifications.push({ id, message, type });
    setTimeout(() => {
      notifications = notifications.filter(n => n.id !== id);
    }, 4000);
  },

  flushPending() {
    for (const n of pendingNotifications) {
      notifications.push(n);
      const id = n.id;
      setTimeout(() => {
        notifications = notifications.filter(x => x.id !== id);
      }, 4000);
    }
    pendingNotifications = [];
  }
};
