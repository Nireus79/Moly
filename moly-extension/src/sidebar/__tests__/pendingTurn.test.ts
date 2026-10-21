import { beforeEach, describe, expect, it } from 'vitest';
import {
  PENDING_GIVE_UP_MS,
  clearPending,
  lastConversation,
  loadPending,
  pickNewConversation,
  rememberConversation,
  replyArrived,
  savePending,
  turnVerdict,
} from '../pendingTurn';

const p = { conversationId: 'c1', text: 'Hello', startedAt: 1_000_000 };

describe('pending turn', () => {
  beforeEach(() => localStorage.clear());

  it('is kept, found again and cleared', () => {
    expect(loadPending()).toBeNull();
    savePending({ ...p, startedAt: Date.now() });
    expect(loadPending()?.text).toBe('Hello');
    clearPending();
    expect(loadPending()).toBeNull();
  });

  it('is forgotten after the give-up time', () => {
    savePending(p);
    expect(loadPending(p.startedAt + PENDING_GIVE_UP_MS + 1)).toBeNull();
    expect(loadPending(p.startedAt + 1000)?.conversationId).toBe('c1');
  });

  it('ignores a damaged note', () => {
    localStorage.setItem('moly_pending_turn', '{nonsense');
    expect(loadPending()).toBeNull();
    localStorage.setItem('moly_pending_turn', '{"text":5}');
    expect(loadPending()).toBeNull();
  });

  it('remembers the last conversation', () => {
    expect(lastConversation()).toBe('');
    rememberConversation('c9');
    expect(lastConversation()).toBe('c9');
    rememberConversation('');
    expect(lastConversation()).toBe('');
  });

  it('finds the reply only when it is newer than the message', () => {
    expect(replyArrived([{ role: 'assistant', timestamp: 500_000 }, { role: 'user', timestamp: 2_000_000 }], p)).toBe(false);
    expect(replyArrived([{ role: 'user', timestamp: 1_000_500 }], p)).toBe(false);
    expect(replyArrived([{ role: 'user', timestamp: 1_000_500 }, { role: 'assistant', timestamp: 1_050_000 }], p)).toBe(true);
    // stored times are whole seconds: a reply a moment "before" the start still counts within the slack
    expect(replyArrived([{ role: 'assistant', timestamp: 998_500 }], p)).toBe(true);
  });

  it('picks the conversation opened for a first message', () => {
    expect(pickNewConversation([{ id: 'old', updatedAt: 100 }], p)).toBeNull();
    expect(pickNewConversation([{ id: 'old', updatedAt: 100 }, { id: 'new', updatedAt: 1001 }, { id: 'newer', updatedAt: 1005 }], p)).toBe('newer');
  });

  it('knows from the engine that a reply will never come', () => {
    const later = p.startedAt + 60_000;
    // our turn failed: the engine's reason is passed on
    const failed = turnVerdict({ state: 'failed', startedAt: p.startedAt + 100, error: 'the AI model is not answering' }, p, p.startedAt + 2000);
    expect(failed).toEqual({ dead: true, message: 'Moly could not finish the reply: the AI model is not answering' });
    // an older turn's failure is not ours
    expect(turnVerdict({ state: 'failed', startedAt: p.startedAt - 60_000 }, p, p.startedAt + 2000)).toEqual({ dead: false });
    // still running: wait
    expect(turnVerdict({ state: 'running', startedAt: p.startedAt + 100 }, p, later)).toEqual({ dead: false });
    expect(turnVerdict({ state: 'running', startedAt: p.startedAt - 60_000 }, p, later)).toEqual({ dead: false });
    // done and the reply is not in the history yet: wait for the next look
    expect(turnVerdict({ state: 'done', startedAt: p.startedAt + 100 }, p, later)).toEqual({ dead: false });
    // the engine knows no turn (restarted), or only an older one: lost, but not in the first seconds
    expect(turnVerdict({ state: 'none', startedAt: 0 }, p, p.startedAt + 3000)).toEqual({ dead: false });
    expect(turnVerdict({ state: 'none', startedAt: 0 }, p, later).dead).toBe(true);
    expect(turnVerdict({ state: 'done', startedAt: p.startedAt - 60_000 }, p, later).dead).toBe(true);
  });
});
