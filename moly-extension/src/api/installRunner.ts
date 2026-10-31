/**
 * Downloads the installer of the newest release and asks the browser to open it. This needs the browser's downloads permission, which the
 * extension does not hold until the person presses the install or update button (it is an optional permission: nobody who never presses the
 * button is asked for it, and an update of the extension does not switch it off). Without the permission, or where the browser has no
 * downloads interface, the button falls back to an ordinary download link.
 *
 * A browser opens a downloaded file only in answer to a click, and a download takes longer than a click lasts; so the file is opened at once
 * when the browser allows it, and otherwise the button changes to "Open the installer" and the next click does it.
 */

import { isReleaseUrl } from './installerDownload';

export type RunState =
  | { phase: 'idle' }
  | { phase: 'downloading'; percent: number | null }
  | { phase: 'ready'; id: number }
  | { phase: 'opened' }
  | { phase: 'saved' } // downloaded by an ordinary link: the person opens it from the download list
  | { phase: 'failed'; message: string };

const PERMISSIONS = ['downloads', 'downloads.open'];
const POLL_MS = 500;
const GIVE_UP_MS = 15 * 60 * 1000;

type DownloadsApi = {
  download(options: { url: string; filename?: string; conflictAction?: string }): Promise<number>;
  search(query: { id: number }): Promise<Array<{ state: string; bytesReceived: number; totalBytes: number; error?: string }>>;
  open(id: number): void | Promise<void>;
};

function api(): DownloadsApi | null {
  const d = (globalThis as { chrome?: { downloads?: DownloadsApi } }).chrome?.downloads;
  return d && typeof d.download === 'function' ? d : null;
}

/** True when the extension holds, or was just given, the downloads permission. Must be called straight from the click. */
export async function ensureDownloadsPermission(): Promise<boolean> {
  const perms = (globalThis as { chrome?: { permissions?: { contains(p: object): Promise<boolean>; request(p: object): Promise<boolean> } } }).chrome?.permissions;
  if (!perms) return api() !== null;
  try {
    if (await perms.contains({ permissions: PERMISSIONS })) return true;
    return await perms.request({ permissions: PERMISSIONS });
  } catch {
    return false;
  }
}

function plainLink(url: string, fileName: string): void {
  const link = document.createElement('a');
  link.href = url;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  link.remove();
}

/** Opens a finished download. Resolves true when the browser accepted it. */
export async function openDownload(id: number): Promise<boolean> {
  try {
    await api()?.open(id);
    return api() !== null;
  } catch {
    return false;
  }
}

/**
 * Downloads and, if the browser lets it, opens. Reports each step through onState. `permitted` is the answer of ensureDownloadsPermission,
 * asked by the caller inside the click.
 */
export async function downloadAndOpen(url: string, fileName: string, permitted: boolean, onState: (s: RunState) => void, wait: (ms: number) => Promise<void> = (ms) => new Promise((r) => setTimeout(r, ms))): Promise<void> {
  if (!isReleaseUrl(url)) {
    onState({ phase: 'failed', message: 'That file is not from the Moly releases, so it was not downloaded.' });
    return;
  }
  const d = api();
  if (!permitted || !d) {
    plainLink(url, fileName);
    onState({ phase: 'saved' });
    return;
  }
  try {
    onState({ phase: 'downloading', percent: null });
    const id = await d.download({ url, filename: fileName, conflictAction: 'uniquify' });
    const started = Date.now();
    for (;;) {
      const [item] = await d.search({ id });
      if (!item) {
        onState({ phase: 'failed', message: 'The download disappeared from the browser. Try again.' });
        return;
      }
      if (item.state === 'complete') break;
      if (item.state === 'interrupted') {
        onState({ phase: 'failed', message: 'The download was interrupted. Check your connection and try again.' });
        return;
      }
      if (Date.now() - started > GIVE_UP_MS) {
        onState({ phase: 'failed', message: 'The download takes too long. Try again.' });
        return;
      }
      onState({ phase: 'downloading', percent: item.totalBytes > 0 ? Math.round((item.bytesReceived / item.totalBytes) * 100) : null });
      await wait(POLL_MS);
    }
    onState((await openDownload(id)) ? { phase: 'opened' } : { phase: 'ready', id });
  } catch (e) {
    onState({ phase: 'failed', message: e instanceof Error ? e.message : 'The download failed.' });
  }
}
