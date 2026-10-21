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
});
