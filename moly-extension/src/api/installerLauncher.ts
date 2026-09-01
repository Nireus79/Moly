/**
 * Cross-platform Installer Launcher
 * Handles downloading and launching moly-installer on Mac, Linux, Windows
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
 * Get installer download URL for platform
 */
export function getInstallerDownloadUrl(platform: Platform): string {
  const baseUrl =
    'https://github.com/user/moly-installer/releases/download/v1.0.0';

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
 * Attempt to launch installer using native messaging
 * Requires moly-native-host to be installed
 */
export async function launchInstallerNative(): Promise<boolean> {
  try {
    // Send message to native host
    const response = await new Promise<{ success?: boolean; error?: string }>(
      (resolve) => {
        chrome.runtime.sendNativeMessage(
          'com.moly.installer',
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
 * Download installer and trigger browser download
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
 * Get appropriate filename for installer
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
async function testNativeHost(): Promise<boolean> {
  return new Promise((resolve) => {
    const timeout = setTimeout(() => resolve(false), 2000);

    chrome.runtime.sendNativeMessage(
      'com.moly.installer',
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
 * Open installer release page
 */
export function openInstallerPage(): void {
  chrome.tabs.create({
    url: 'https://github.com/user/moly-installer/releases',
  });
}
