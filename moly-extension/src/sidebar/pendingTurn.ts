/**
 * A message that was sent and whose reply has not arrived. The engine saves a turn (the person's message and Moly's reply)
 * only when the reply is finished, so while Moly is writing there is nothing in the history. On the phone the app can be
 * closed or killed in that time; this note, kept on the device, lets the next start find the reply and wait for it.
 */
const PENDING_KEY = 'moly_pending_turn';
const LAST_KEY = 'moly_last_conversation';
const UNSENT_KEY = 'moly_unsent_message';

/** Within this long away, Moly opens on the conversation the person left; after it, on a blank chat (the history is one tap away). */
export const RESUME_WITHIN_MS = 10 * 60 * 1000;

/** How long a reply is waited for before giving up (the engine may have been stopped, and then it never comes). */
export const PENDING_GIVE_UP_MS = 10 * 60 * 1000;
export const PENDING_POLL_MS = 4000;
// The engine and the screens share one clock on the phone; a little slack covers rounding to whole seconds.
const CLOCK_SLACK_MS = 3000;

export interface PendingTurn {
  /** Empty when the message started a new conversation (its id is not known until the reply). */
  conversationId: string;
  text: string;
  startedAt: number;
}

export interface StoredMessage {
  role: string;
  timestamp: number; // ms
}

function read(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function write(key: string, value: string | null): void {
  try {
    if (value === null) localStorage.removeItem(key);
    else localStorage.setItem(key, value);
  } catch {
    /* storage unavailable: the reply is then simply not found after a restart */
  }
}

export function savePending(p: PendingTurn): void {
  write(PENDING_KEY, JSON.stringify(p));
}

export function clearPending(): void {
  write(PENDING_KEY, null);
}

export function loadPending(now: number = Date.now()): PendingTurn | null {
  const raw = read(PENDING_KEY);
  if (!raw) return null;
  try {
    const p = JSON.parse(raw) as PendingTurn;
    if (typeof p.text !== 'string' || typeof p.startedAt !== 'number') return null;
    if (now - p.startedAt > PENDING_GIVE_UP_MS) return null;
    return { conversationId: typeof p.conversationId === 'string' ? p.conversationId : '', text: p.text, startedAt: p.startedAt };
  } catch {
    return null;
  }
}

/** Continue the last conversation only after a short time away (0: not known, which counts as short). */
export function resumeAfter(awayMs: number): boolean {
  return !(awayMs >= RESUME_WITHIN_MS);
}

export function rememberConversation(id: string): void {
  write(LAST_KEY, id || null);
}

export function lastConversation(): string {
  return read(LAST_KEY) || '';
}

/** Moly's reply to the pending message is in the history once an assistant message is newer than the moment it was sent. */
export function replyArrived(messages: StoredMessage[], p: PendingTurn): boolean {
  return messages.some((m) => m.role === 'assistant' && m.timestamp >= p.startedAt - CLOCK_SLACK_MS);
}

/** For a message that began a new conversation: the conversation the engine opened for it (the newest one touched since). */
export function pickNewConversation(list: Array<{ id: string; updatedAt: number }>, p: PendingTurn): string | null {
  const fresh = list.filter((c) => c.updatedAt * 1000 >= p.startedAt - CLOCK_SLACK_MS);
  if (fresh.length === 0) return null;
  return fresh.reduce((a, b) => (b.updatedAt > a.updatedAt ? b : a)).id;
}

export interface TurnStatus {
  state: 'none' | 'running' | 'done' | 'failed';
  startedAt: number; // ms
  error?: string;
}

// A request in flight is recorded by the engine at once, so a turn that is not recorded after this long is gone.
const UNRECORDED_GRACE_MS = 15000;

/**
 * What the engine says about the last turn, read against the pending message: keep waiting, or the reply will never come
 * (the turn failed, or the engine was restarted and forgot it). Only called when no reply is in the history.
 */
export function turnVerdict(status: TurnStatus, p: PendingTurn, now: number = Date.now()): { dead: false } | { dead: true; message: string } {
  const ours = status.startedAt >= p.startedAt - CLOCK_SLACK_MS;
  if (status.state === 'failed' && ours) {
    return { dead: true, message: status.error ? `Moly could not finish the reply: ${status.error}` : 'Moly could not finish the reply.' };
  }
  const old = now - p.startedAt > UNRECORDED_GRACE_MS;
  if (old && (status.state === 'none' || (!ours && status.state !== 'running'))) {
    return { dead: true, message: 'Moly was restarted while it was writing, so the reply is lost.' };
  }
  return { dead: false };
}

/**
 * The phone's HTTP layer says "Failed to connect to /127.0.0.1:11436" (not a TypeError) when the engine is not running. Nothing
 * reached the engine then, so the message was not sent and the person gets it back.
 */
export function engineUnreachable(err: unknown): boolean {
  const m = err instanceof Error ? err.message : String(err ?? '');
  return /failed to connect|connection refused|econnrefused|failed to fetch|network request failed/i.test(m);
}

export const UNSENT_NOTE = 'Moly had stopped and is starting again. Your message is back in the box: send it again.';

/** Keeps a message that was not sent: the engine restart that follows rebuilds the screen and would lose it. */
export function saveUnsent(text: string): void {
  write(UNSENT_KEY, text);
}

/** The message kept by saveUnsent, once; null when there is none. */
export function takeUnsent(): string | null {
  const t = read(UNSENT_KEY);
  write(UNSENT_KEY, null);
  return t && t.trim() ? t : null;
}
