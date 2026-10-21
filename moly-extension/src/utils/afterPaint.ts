/**
 * Resolves once the page has had a chance to paint, or after maxWaitMs, whichever comes first. A page that is not being painted
 * (the phone's screen is off, or the person switched to another app straight after tapping Send) never calls
 * requestAnimationFrame, and whatever waits for it would wait until they come back.
 */
export function afterPaint(maxWaitMs = 150): Promise<void> {
  return new Promise((resolve) => {
    let done = false;
    const finish = () => {
      if (!done) {
        done = true;
        resolve();
      }
    };
    setTimeout(finish, maxWaitMs);
    try {
      requestAnimationFrame(finish);
    } catch {
      finish();
    }
  });
}
