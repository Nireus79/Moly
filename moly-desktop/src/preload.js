const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('moly', {
  // Check if native host is installed
  checkNativeHost: () => ipcRenderer.invoke('check-native-host'),

  // Get native host installation path
  getNativeHostPath: () => ipcRenderer.invoke('get-native-host-path'),

  // Check if CORS proxy is running
  getProxyStatus: () => ipcRenderer.invoke('get-proxy-status'),

  // Download and install native host
  installNativeHost: () => ipcRenderer.invoke('install-native-host'),

  // Get system info
  getSystemInfo: () => ipcRenderer.invoke('get-system-info'),

  // Start setup wizard
  startSetup: () => ipcRenderer.invoke('start-setup'),

  // Open floating sidebar
  openSidebar: () => ipcRenderer.invoke('open-sidebar'),
});
