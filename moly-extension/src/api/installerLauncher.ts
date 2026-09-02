/**
 * Cross-platform Native Host Installer
 * Moly orchestrates native host installation through the extension
 *
 * New Self-Install Flow:
 * 1. Extension detects native host missing
 * 2. Extension downloads native host binary from GitHub
 * 3. Extension invokes native messaging setup (if host becomes available)
 * 4. Native host self-installs to system location
 * 5. Extension verifies installation and enables service control
 */

export type Platform = 'macos' | 'linux' | 'windows' | 'unknown';

export interface InstallerStatus {
  platform: Platform;
  canLaunchNative: boolean;
  downloadUrl?: string;
  instructions?: string[];
}

/**
 * Detect the user's platform
 */
export function detectPlatform(): Platform {
  const ua = navigator.userAgent.toLowerCase();

  if (ua.indexOf('mac') > -1) {
    return 'macos';
  } else if (ua.indexOf('linux') > -1) {
    return 'linux';
  } else if (ua.indexOf('win') > -1) {
    return 'windows';
  }

  return 'unknown';
}

/**
 * Get native host binary download URL for platform
 */
export function getNativeHostDownloadUrl(platform: Platform): string {
  const baseUrl =
    'https://github.com/Nireus79/Moly/releases/download/v1.0.0';

  switch (platform) {
    case 'macos':
      // Intel: moly-installer-macos.tar.gz contains moly-native-host
      // ARM64: moly-installer-macos-arm64.tar.gz contains moly-native-host
      return navigator.hardwareConcurrency > 4
        ? `${baseUrl}/moly-installer-macos-arm64.tar.gz`
        : `${baseUrl}/moly-installer-macos.tar.gz`;
    case 'linux':
      return `${baseUrl}/moly-installer-linux-x64.tar.gz`;
    case 'windows':
      return `${baseUrl}/moly-installer-windows-x64.exe`;
    default:
      return '';
  }
}

/**
 * Get installer download URL for platform (legacy, for fallback)
 */
export function getInstallerDownloadUrl(platform: Platform): string {
  const baseUrl =
    'https://github.com/Nireus79/Moly/releases/download/v1.0.0';

  switch (platform) {
    case 'macos':
      return `${baseUrl}/moly-installer-macos-arm64.dmg`;
    case 'linux':
      return `${baseUrl}/moly-installer-linux-x64`;
    case 'windows':
      return `${baseUrl}/moly-installer-windows-x64.exe`;
    default:
      return '';
  }
}

/**
 * Get platform-specific setup instructions
 */
export function getSetupInstructions(platform: Platform): string[] {
  switch (platform) {
    case 'macos':
      return [
        '1. Download moly-installer-macos-arm64.dmg',
        '2. Double-click to mount the disk image',
        '3. Drag Moly Installer to Applications folder',
        '4. Open Applications, launch Moly Installer',
        '5. Follow the installation wizard',
      ];

    case 'linux':
      return [
        '1. Download moly-installer-linux-x64',
        '2. Open terminal in Downloads folder',
        '3. Run: chmod +x moly-installer-linux-x64',
        '4. Run: ./moly-installer-linux-x64',
        '5. Follow the installation wizard',
      ];

    case 'windows':
      return [
        '1. Download moly-installer-windows-x64.exe',
        '2. Double-click the .exe file',
        '3. Click "Yes" if prompted by User Account Control',
        '4. Follow the installation wizard',
        '5. Moly will start automatically after setup',
      ];

    default:
      return [
        '1. Visit https://github.com/user/moly-installer/releases',
        '2. Download installer for your platform',
        '3. Run the installer',
        '4. Follow the installation wizard',
      ];
  }
}

/**
 * Send install message to native host
 * Native host self-installs to system location
 */
export async function installNativeHost(
  extensionId: string
): Promise<{ success: boolean; error?: string; install_path?: string }> {
  return new Promise((resolve) => {
    const timeout = setTimeout(
      () => resolve({ success: false, error: 'Installation timeout' }),
      30000
    );

    chrome.runtime.sendNativeMessage(
      'com.moly.native_host',
      { action: 'install', extension_id: extensionId },
      (response) => {
        clearTimeout(timeout);
        if (chrome.runtime.lastError) {
          resolve({
            success: false,
            error: chrome.runtime.lastError.message,
          });
        } else {
          resolve(response || { success: false, error: 'No response' });
        }
      }
    );
  });
}

/**
 * Pull an Ollama model via native host
 */
export async function pullModel(
  modelName: string
): Promise<{ success: boolean; message?: string; error?: string }> {
  return new Promise((resolve) => {
    const timeout = setTimeout(
      () => resolve({ success: false, error: 'Model pull timeout' }),
      3600000 // 1 hour timeout for large models
    );

    chrome.runtime.sendNativeMessage(
      'com.moly.native_host',
      { action: 'pull-model', model: modelName },
      (response) => {
        clearTimeout(timeout);
        if (chrome.runtime.lastError) {
          resolve({
            success: false,
            error: chrome.runtime.lastError.message,
          });
        } else {
          resolve(response || { success: false, error: 'No response' });
        }
      }
    );
  });
}

/**
 * Install CORS proxy via native host
 */
export async function installCORSProxy(): Promise<{
  success: boolean;
  message?: string;
  error?: string;
}> {
  return new Promise((resolve) => {
    const timeout = setTimeout(
      () => resolve({ success: false, error: 'CORS proxy install timeout' }),
      300000 // 5 minute timeout
    );

    chrome.runtime.sendNativeMessage(
      'com.moly.native_host',
      { action: 'install-cors-proxy' },
      (response) => {
        clearTimeout(timeout);
        if (chrome.runtime.lastError) {
          resolve({
            success: false,
            error: chrome.runtime.lastError.message,
          });
        } else {
          resolve(response || { success: false, error: 'No response' });
        }
      }
    );
  });
}

/**
 * Configure auto-start for services via native host
 */
export async function setupAutoStart(): Promise<{
  success: boolean;
  error?: string;
}> {
  return new Promise((resolve) => {
    const timeout = setTimeout(
      () => resolve({ success: false, error: 'Setup timeout' }),
      15000
    );

    chrome.runtime.sendNativeMessage(
      'com.moly.native_host',
      { action: 'setup-autostart' },
      (response) => {
        clearTimeout(timeout);
        if (chrome.runtime.lastError) {
          resolve({
            success: false,
            error: chrome.runtime.lastError.message,
          });
        } else {
          resolve(response || { success: false, error: 'No response' });
        }
      }
    );
  });
}

/**
 * Attempt to launch installer using native messaging
 * Requires moly-native-host to be installed
 */
export async function launchInstallerNative(): Promise<boolean> {
  try {
    // Send message to native host
    const response = await new Promise<{ success?: boolean; error?: string }>(
      (resolve) => {
        chrome.runtime.sendNativeMessage(
          'com.moly.native_host',
          { action: 'launch' },
          (response) => {
            if (chrome.runtime.lastError) {
              resolve({ error: chrome.runtime.lastError.message });
            } else {
              resolve(response || { success: false });
            }
          }
        );
      }
    );

    return response.success === true;
  } catch {
    return false;
  }
}

/**
 * Download native host binary (new self-install flow)
 */
export async function downloadNativeHost(platform: Platform): Promise<void> {
  const url = getNativeHostDownloadUrl(platform);
  if (!url) {
    throw new Error(`Unsupported platform: ${platform}`);
  }

  // Trigger download
  chrome.downloads.download({
    url,
    filename: getNativeHostFilename(platform),
    saveAs: false,
  });
}

/**
 * Download installer and trigger browser download (legacy)
 */
export async function downloadInstaller(platform: Platform): Promise<void> {
  const url = getInstallerDownloadUrl(platform);
  if (!url) {
    throw new Error(`Unsupported platform: ${platform}`);
  }

  // Trigger download
  chrome.downloads.download({
    url,
    filename: getInstallerFilename(platform),
    saveAs: false,
  });
}

/**
 * Get appropriate filename for native host binary
 */
function getNativeHostFilename(platform: Platform): string {
  switch (platform) {
    case 'macos':
      return navigator.hardwareConcurrency > 4
        ? 'moly-installer-macos-arm64.tar.gz'
        : 'moly-installer-macos.tar.gz';
    case 'linux':
      return 'moly-installer-linux-x64.tar.gz';
    case 'windows':
      return 'moly-installer-windows-x64.exe';
    default:
      return 'moly-native-host';
  }
}

/**
 * Get appropriate filename for installer (legacy)
 */
function getInstallerFilename(platform: Platform): string {
  switch (platform) {
    case 'macos':
      return 'moly-installer-macos.dmg';
    case 'linux':
      return 'moly-installer-linux';
    case 'windows':
      return 'moly-installer.exe';
    default:
      return 'moly-installer';
  }
}

/**
 * Get complete installer status for a platform
 */
export async function getInstallerStatus(
  platform: Platform
): Promise<InstallerStatus> {
  // Try native launch first
  const nativeAvailable = await testNativeHost();

  return {
    platform,
    canLaunchNative: nativeAvailable,
    downloadUrl: getInstallerDownloadUrl(platform),
    instructions: getSetupInstructions(platform),
  };
}

/**
 * Test if native host is available
 */
export async function testNativeHost(): Promise<boolean> {
  return new Promise((resolve) => {
    const timeout = setTimeout(() => resolve(false), 2000);

    chrome.runtime.sendNativeMessage(
      'com.moly.native_host',
      { action: 'ping' },
      (response) => {
        clearTimeout(timeout);
        if (chrome.runtime.lastError) {
          resolve(false);
        } else {
          resolve(response?.pong === true);
        }
      }
    );
  });
}

/**
 * Complete orchestrated setup flow
 * Coordinates downloading and installing native host
 */
export async function orchestrateSetup(
  platform: Platform,
  extensionId: string
): Promise<{
  success: boolean;
  step?: string;
  error?: string;
}> {
  try {
    // Step 1: Check if native host already installed
    const hostAvailable = await testNativeHost();
    if (hostAvailable) {
      return {
        success: true,
        step: 'already-installed',
      };
    }

    // Step 2: Download native host binary
    // This returns immediately - browser handles download
    await downloadNativeHost(platform);

    return {
      success: true,
      step: 'downloaded',
      error: 'User must run the downloaded installer to complete setup',
    };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Setup failed',
    };
  }
}

/**
 * Try to complete setup after native host becomes available
 * Call this after user manually runs the native host binary
 */
export async function completeSetupAfterInstall(
  extensionId: string
): Promise<{ success: boolean; error?: string }> {
  try {
    // Verify native host is now available
    const available = await testNativeHost();
    if (!available) {
      return {
        success: false,
        error: 'Native host still not available',
      };
    }

    // Trigger self-install within the native host
    const installResult = await installNativeHost(extensionId);
    if (!installResult.success) {
      return {
        success: false,
        error: installResult.error || 'Installation failed',
      };
    }

    // Install CORS proxy for better performance
    // Non-blocking: if it fails, continue anyway (direct connection works as fallback)
    const proxyResult = await installCORSProxy();
    if (!proxyResult.success) {
      console.warn('[Setup] CORS proxy install skipped:', proxyResult.error);
      // Continue anyway - direct Ollama connection works as fallback
    }

    // Configure auto-start
    const autoStartResult = await setupAutoStart();
    if (!autoStartResult.success) {
      return {
        success: false,
        error: autoStartResult.error || 'Auto-start setup failed',
      };
    }

    return { success: true };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : 'Completion failed',
    };
  }
}

/**
 * Open installer release page
 */
export function openInstallerPage(): void {
  chrome.tabs.create({
    url: 'https://github.com/Nireus79/Moly/releases',
  });
}
