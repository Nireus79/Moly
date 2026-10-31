/**
 * What went wrong when the extension tried to reach the Moly app (native messaging host), in words a person can act on.
 * The browser's own messages are matched loosely: they are not a stable API.
 */

/** The protocol of the messages between this extension and the Moly app. It rises only when they stop understanding each other (host.go: hostVersion). */
export const HOST_PROTOCOL = '1';

/** The error text when the app and the extension speak different protocols (either side may find it). */
export const protocolMismatch = (appProtocol: string) => `The Moly app and this extension speak different versions of the protocol (app ${appProtocol}, extension ${HOST_PROTOCOL}).`;

export type HostProblem = 'app-mismatch' | 'not-installed' | 'not-allowed' | 'host-crashed' | 'start-timeout' | 'unknown' | 'none';
export type OSName = 'windows' | 'macos' | 'linux' | 'unknown';

export function classifyHostError(message: string | undefined): HostProblem {
  if (!message) return 'none';
  const m = message.toLowerCase();
  if (m.includes('speak different versions')) return 'app-mismatch';
  if (m.includes('not found') || m.includes('not registered') || m.includes('not installed')) return 'not-installed';
  if (m.includes('forbidden') || m.includes('not allowed')) return 'not-allowed';
  if (m.includes('has exited') || m.includes('exited')) return 'host-crashed';
  if (m.includes('did not answer') || m.includes('not become ready') || m.includes('in time')) return 'start-timeout';
  return 'unknown';
}

/** Problems that cannot fix themselves: retrying in the background is pointless until the user acts. */
export const needsUserAction = (p: HostProblem) => p === 'not-installed' || p === 'not-allowed' || p === 'app-mismatch';

export function detectOS(userAgent: string = typeof navigator !== 'undefined' ? navigator.userAgent : ''): OSName {
  const ua = userAgent.toLowerCase();
  if (ua.includes('windows')) return 'windows';
  if (ua.includes('mac os') || ua.includes('macintosh')) return 'macos';
  if (ua.includes('linux') || ua.includes('x11') || ua.includes('cros')) return 'linux';
  return 'unknown';
}

export interface InstallInfo {
  heading: string;
  steps: string[];
  note?: string;
}

/** Steps shown for each system. The beta installers are not signed yet; the texts say what the user will see. */
export function installInfo(os: OSName): InstallInfo {
  switch (os) {
    case 'windows':
      return {
        heading: 'Install Moly for Windows',
        steps: [
          'Press "Download latest version". The installer downloads and opens; follow it.',
          'If Windows says "Windows protected your PC", choose "More info", then "Run anyway". The beta installer is not signed yet.',
          'Come back here and press "Check again".',
        ],
      };
    case 'macos':
      return {
        heading: 'Install Moly for Mac (beta)',
        steps: [
          'Press "Download latest version". The installer downloads and opens; follow it.',
          'If macOS blocks it, open System Settings, then Privacy & Security, and choose "Open Anyway" next to Moly.',
          'Come back here and press "Check again".',
        ],
        note: 'Mac support is in beta. If something does not work, write to us: it helps a lot.',
      };
    case 'linux':
      return {
        heading: 'Install Moly for Linux',
        steps: [
          'Press "Download latest version". The installer downloads and opens in your software installer. If a line of text appears instead, paste it in a terminal and press Enter.',
          'This screen continues by itself when Moly is installed.',
          'If it does not, restart your browser and press "Check again".',
        ],
      };
    default:
      return {
        heading: 'Install the Moly app',
        steps: ['Press "Download latest version" and follow the installer.', 'This screen continues by itself when Moly is installed.', 'If it does not, restart your browser and press "Check again".'],
      };
  }
}

export interface UninstallInfo {
  heading: string;
  steps: string[];
}

/** How to remove the Moly app for each system. The extension cannot do it (a browser extension may not delete programs), so it says how. site/uninstall.html says the same. */
export function uninstallInfo(os: OSName): UninstallInfo {
  const data = 'Your conversations stay in the .moly folder of your user folder unless you delete that folder too. Export your data first in Settings if you may want it back.';
  switch (os) {
    case 'windows':
      return { heading: 'Remove the Moly app from Windows', steps: ['Open Settings, then Apps, then Installed apps.', 'Find Moly and choose Uninstall.', data] };
    case 'macos':
      return {
        heading: 'Remove the Moly app from your Mac',
        steps: [
          'Open Terminal and paste: "$HOME/Library/Application Support/moly-program/uninstall.sh" (add --purge to delete your data too). If you installed the .pkg, paste: sudo /usr/local/lib/moly/uninstall.sh',
          data,
        ],
      };
    case 'linux':
      return {
        heading: 'Remove the Moly app from Linux',
        steps: [
          'If you installed the .deb: open a terminal and paste "sudo apt remove moly". Otherwise paste "~/.local/share/moly/uninstall.sh" (add --purge to delete your data too).',
          data,
        ],
      };
    default:
      return { heading: 'Remove the Moly app', steps: ['Run the uninstaller that came with the Moly app, or see the page linked below.', data] };
  }
}

export interface ProblemText {
  title: string;
  body: string;
}

export function problemText(p: HostProblem): ProblemText {
  switch (p) {
    case 'app-mismatch':
      return {
        title: 'The Moly app and the extension do not match',
        body: 'They were made for different versions of Moly. Install the latest Moly app and update the extension to the latest version, then press "Check again".',
      };
    case 'not-allowed':
      return {
        title: 'Moly does not recognise this extension',
        body: 'The Moly app is installed, but it was set up for a different copy of the extension. Install the latest Moly app, which registers this one, then press "Check again".',
      };
    case 'host-crashed':
      return {
        title: 'The Moly app stopped',
        body: 'It closed unexpectedly while starting. Press "Check again". If it keeps happening, send us the log file.',
      };
    case 'start-timeout':
      return {
        title: 'Moly is taking too long to start',
        body: 'This can happen the first time, or on a slow computer. Wait a minute and press "Check again".',
      };
    default:
      return {
        title: 'Moly could not start',
        body: 'Press "Check again". If it keeps happening, send us the log file.',
      };
  }
}
