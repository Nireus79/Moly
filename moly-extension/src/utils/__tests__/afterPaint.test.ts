import { describe, it, expect, vi, afterEach } from 'vitest';
import { afterPaint } from '../afterPaint';

afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals(); });

describe('afterPaint', () => {
  it('resolves when the page paints', async () => {
    vi.useFakeTimers();
    vi.stubGlobal('requestAnimationFrame', (cb: () => void) => { setTimeout(cb, 16); return 1; });
    let done = false;
    void afterPaint().then(() => { done = true; });
    await vi.advanceTimersByTimeAsync(20);
    expect(done).toBe(true);
  });

  it('does not wait for ever on a page that is not painted', async () => {
    vi.useFakeTimers();
    vi.stubGlobal('requestAnimationFrame', () => 1); // never calls back, like a hidden page
    let done = false;
    void afterPaint(150).then(() => { done = true; });
    await vi.advanceTimersByTimeAsync(100);
    expect(done).toBe(false);
    await vi.advanceTimersByTimeAsync(60);
    expect(done).toBe(true);
  });
});
