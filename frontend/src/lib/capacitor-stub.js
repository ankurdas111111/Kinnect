const listenerHandle = { remove: () => {} };

export const App = {
  addListener: async () => listenerHandle,
  removeAllListeners: async () => {}
};

export const Geolocation = {
  checkPermissions: async () => ({ location: "prompt" }),
  requestPermissions: async () => ({ location: "prompt" }),
  getCurrentPosition: async () => {
    throw new Error("Capacitor geolocation is unavailable in web builds.");
  },
  watchPosition: async () => "web-stub-watch",
  clearWatch: async () => {}
};

export const Device = {
  getBatteryInfo: async () => ({ batteryLevel: null, isCharging: false }),
  getInfo: async () => ({ platform: 'web', operatingSystem: 'unknown' })
};

export const Network = {
  getStatus: async () => ({ connected: true, connectionType: 'unknown' }),
  addListener: async () => listenerHandle
};

export const Share = {
  canShare: async () => ({ value: false }),
  share: async () => {}
};

export const Haptics = {
  impact: async () => {},
  notification: async () => {},
  vibrate: async () => {}
};

export const ImpactStyle = { Heavy: 'HEAVY', Medium: 'MEDIUM', Light: 'LIGHT' };
export const NotificationType = { Success: 'SUCCESS', Warning: 'WARNING', Error: 'ERROR' };

export const LocalNotifications = {
  checkPermissions: async () => ({ display: 'denied' }),
  requestPermissions: async () => ({ display: 'denied' }),
  schedule: async () => {},
  addListener: async () => listenerHandle
};

export const registerPlugin = () => ({});

// Minimal shims for plugins that import these from @capacitor/core
// (e.g. @capgo/capacitor-live-activities) — never used at runtime on web.
export const Capacitor = {
  getPlatform: () => 'web',
  isNativePlatform: () => false
};
export class WebPlugin {
  addListener() { return Promise.resolve(listenerHandle); }
  removeAllListeners() { return Promise.resolve(); }
  notifyListeners() {}
}

export default {};
