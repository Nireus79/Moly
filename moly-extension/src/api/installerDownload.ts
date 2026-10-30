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

/** The file of a release that installs on each system by a double-click (Linux: a .deb, Mac: a .pkg, Windows: the setup program). */
const INSTALLER_FILE: Partial<Record<OSName, RegExp>> = {
  windows: /^moly-setup-.*\.exe$/,
  linux: /^moly[-_].*\.deb$/,
  macos: /^moly-.*\.pkg$/,
};

/** Only files of the project's own releases are ever downloaded by the button. */
export const RELEASE_URL_PREFIX = 'https://github.com/Nireus79/Moly/releases/download/';
export const isReleaseUrl = (url: string): boolean => url.startsWith(RELEASE_URL_PREFIX);

async function newestDesktopRelease(get: typeof fetch): Promise<Release | null> {
  const res = await get(RELEASES_API);
  if (!res.ok) return null;
  // Android releases before 0.1.2 are tagged mobile-v...; the desktop ones v...
  const releases = (await res.json()) as Release[];
  return releases.find((r) => !r.draft && /^v\d/.test(r.tag_name)) ?? null;
}

/**
 * What the install button does on this system. It finds the installer file of the newest release for this system (the setup program on
 * Windows, a .deb on Linux, a .pkg on Mac); the extension downloads it and asks the browser to open it. Where a release has no such file
 * (Linux and Mac before it was built), Linux and Mac get the command to paste. If the newest release cannot be looked up, Windows is sent
 * to the download page, which explains every system.
 */
export async function installAction(os: OSName, get: typeof fetch = fetch): Promise<InstallAction> {
  const fallback: InstallAction = os === 'linux' || os === 'macos' ? { kind: 'command', command: INSTALL_COMMAND } : { kind: 'page' };
  const pattern = INSTALLER_FILE[os];
  if (!pattern) return fallback;
  try {
    const release = await newestDesktopRelease(get);
    const file = release?.assets.find((a) => pattern.test(a.name) && isReleaseUrl(a.browser_download_url));
    if (!release || !file) return fallback;
    return { kind: 'download', url: file.browser_download_url, fileName: file.name, version: release.tag_name.replace(/^v/, '') };
  } catch {
    return fallback;
  }
}

/** The version number of the newest desktop release, or null when it cannot be looked up. */
export async function latestVersion(get: typeof fetch = fetch): Promise<string | null> {
  try {
    const release = await newestDesktopRelease(get);
    return release ? release.tag_name.replace(/^v/, '') : null;
  } catch {
    return null;
  }
}

/**
 * Compares two versions like 0.1.2 and 0.1.0-beta.4: the numbers first, and a version with a pre-release part (-beta.4) is older than the
 * same numbers without it. Returns negative, 0 or positive; null when either is not a version (a development build says "dev").
 */
export function compareVersions(a: string, b: string): number | null {
  const parse = (v: string) => {
    const m = /^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$/.exec(v.trim());
    return m ? { n: [Number(m[1]), Number(m[2]), Number(m[3])], pre: m[4] ?? '' } : null;
  };
  const x = parse(a);
  const y = parse(b);
  if (!x || !y) return null;
  for (let i = 0; i < 3; i++) if (x.n[i] !== y.n[i]) return x.n[i] - y.n[i];
  if (x.pre === y.pre) return 0;
  if (x.pre === '') return 1;
  if (y.pre === '') return -1;
  const px = x.pre.split('.');
  const py = y.pre.split('.');
  for (let i = 0; i < Math.max(px.length, py.length); i++) {
    const u = px[i];
    const v = py[i];
    if (u === undefined) return -1;
    if (v === undefined) return 1;
    const nu = /^\d+$/.test(u);
    const nv = /^\d+$/.test(v);
    if (nu && nv && Number(u) !== Number(v)) return Number(u) - Number(v);
    if (nu !== nv) return nu ? -1 : 1;
    if (!nu && u !== v) return u < v ? -1 : 1;
  }
  return 0;
}
