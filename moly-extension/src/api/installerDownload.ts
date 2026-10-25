import type { OSName } from './hostErrors';

/** One command for Linux and Mac: finds the newest release, checks it and installs it (see site/install.sh). */
export const INSTALL_COMMAND = 'curl -fsSL https://nireus79.github.io/Moly/install.sh | sh';

const RELEASES_API = 'https://api.github.com/repos/Nireus79/Moly/releases?per_page=30';

export type InstallAction =
  | { kind: 'download'; url: string; fileName: string; version: string }
  | { kind: 'command'; command: string }
  | { kind: 'page' };

interface ReleaseAsset { name: string; browser_download_url: string }
interface Release { tag_name: string; draft: boolean; assets: ReleaseAsset[] }

/**
 * What the install button does on this system. Windows gets the newest installer file (the browser cannot run it for the
 * person: they open it from the download list). Linux and Mac get a command to paste. If the newest release cannot be
 * looked up, or the system is unknown, the person is sent to the download page, which explains every system.
 */
export async function installAction(os: OSName, get: typeof fetch = fetch): Promise<InstallAction> {
  if (os === 'linux' || os === 'macos') return { kind: 'command', command: INSTALL_COMMAND };
  if (os !== 'windows') return { kind: 'page' };
  try {
    const res = await get(RELEASES_API);
    if (!res.ok) return { kind: 'page' };
    // Android releases are tagged mobile-v...; the desktop ones v...
    const releases = (await res.json()) as Release[];
    const release = releases.find((r) => !r.draft && /^v\d/.test(r.tag_name));
    const exe = release?.assets.find((a) => /^moly-setup-.*\.exe$/.test(a.name));
    if (!release || !exe) return { kind: 'page' };
    return { kind: 'download', url: exe.browser_download_url, fileName: exe.name, version: release.tag_name.replace(/^v/, '') };
  } catch {
    return { kind: 'page' };
  }
}
