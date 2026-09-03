const { app, BrowserWindow, Menu, ipcMain, dialog } = require('electron');
const path = require('path');
const { spawn } = require('child_process');
const os = require('os');
const fs = require('fs');
const MolyInstaller = require('./services/installer');

let mainWindow;
let sidebarWindow;
let nativeHostProcess;
const installer = new MolyInstaller();

// Paths
const HOME = os.homedir();
const PLATFORM = process.platform;
const INSTALL_DIR = path.join(HOME, '.local', 'bin');
const NATIVE_HOST_PATH = path.join(INSTALL_DIR, 'moly-native-host');

function createWindow() {
  const iconPath = path.join(__dirname, '../assets/icon.png');
  const windowConfig = {
    width: 1200,
    height: 800,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      nodeIntegration: false,
      contextIsolation: true,
    }
  };

  if (fs.existsSync(iconPath)) {
    windowConfig.icon = iconPath;
  }

  mainWindow = new BrowserWindow(windowConfig);

  const startUrl = `file://${path.join(__dirname, '../build/index.html')}`;
  mainWindow.loadURL(startUrl);

  mainWindow.on('closed', () => {
    mainWindow = null;
  });
}

function createSidebar() {
  if (sidebarWindow) {
    sidebarWindow.focus();
    return;
  }

  const iconPath = path.join(__dirname, '../assets/icon.png');
  sidebarWindow = new BrowserWindow({
    width: 420,
    height: 600,
    x: -420,
    y: 0,
    alwaysOnTop: true,
    frame: true,
    skipTaskbar: false,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      nodeIntegration: false,
      contextIsolation: true,
    }
  });

  if (fs.existsSync(iconPath)) {
    sidebarWindow.setIcon(iconPath);
  }

  const startUrl = `file://${path.join(__dirname, '../build/index.html')}?mode=sidebar`;
  sidebarWindow.loadURL(startUrl);

  sidebarWindow.on('closed', () => {
    sidebarWindow = null;
  });
}

function startNativeHost() {
  // No longer needed - Electron app calls Ollama directly
  // Leaving this function empty for backwards compatibility
}

function stopNativeHost() {
  if (nativeHostProcess) {
    nativeHostProcess.kill();
    nativeHostProcess = null;
  }
}

// IPC Handlers
ipcMain.handle('check-native-host', () => {
  return fs.existsSync(NATIVE_HOST_PATH);
});

ipcMain.handle('get-native-host-path', () => {
  return NATIVE_HOST_PATH;
});

ipcMain.handle('get-proxy-status', async () => {
  try {
    const response = await fetch('http://127.0.0.1:11435/api/tags');
    return response.status === 200;
  } catch {
    return false;
  }
});

ipcMain.handle('install-native-host', async () => {
  // No longer needed - app calls Ollama directly
  return { success: true };
});

ipcMain.handle('get-system-info', () => {
  return {
    platform: PLATFORM,
    arch: os.arch(),
    version: app.getVersion(),
  };
});

ipcMain.handle('start-setup', async () => {
  return { setupComplete: fs.existsSync(NATIVE_HOST_PATH) };
});

ipcMain.handle('open-sidebar', () => {
  createSidebar();
  return { success: true };
});

// App lifecycle
app.on('ready', () => {
  startNativeHost();
  createWindow();
});

app.on('window-all-closed', () => {
  stopNativeHost();
  if (process.platform !== 'darwin') {
    app.quit();
  }
});

app.on('activate', () => {
  if (mainWindow === null) {
    createWindow();
  }
});

console.log('[Moly] Desktop app initialized');
