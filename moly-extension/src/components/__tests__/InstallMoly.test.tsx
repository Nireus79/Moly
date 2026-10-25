import { describe, it, expect, vi, afterEach } from 'vitest';
import React from 'react';
import { createRoot } from 'react-dom/client';
import { act } from 'react-dom/test-utils';
import { InstallMoly } from '../InstallMoly';

(globalThis as any).IS_REACT_ACT_ENVIRONMENT = true;

let host: HTMLDivElement | null = null;
async function render(el: React.ReactElement) {
  host = document.createElement('div');
  document.body.appendChild(host);
  const root = createRoot(host);
  await act(async () => root.render(el));
  return host;
}
afterEach(() => {
  host?.remove();
  host = null;
});

describe('InstallMoly', () => {
  it('a missing app shows the install steps, a download link and Check again', async () => {
    const onCheck = vi.fn();
    const el = await render(<InstallMoly problem="not-installed" detail="Specified native messaging host not found." checking={false} onCheckAgain={onCheck} />);
    expect(el.textContent).toContain('Moly needs a small app on your computer');
    const install = Array.from(el.querySelectorAll('button')).find((b) => b.textContent === 'Download latest version');
    expect(install).toBeDefined();
    const button = Array.from(el.querySelectorAll('button')).find((b) => b.textContent === 'Check again')!;
    await act(async () => button.click());
    expect(onCheck).toHaveBeenCalledTimes(1);
    expect(el.textContent).toContain('Hermes_creative@proton.me');
  });

  it('technical details stay hidden until asked for', async () => {
    const el = await render(<InstallMoly problem="not-installed" detail="Specified native messaging host not found." checking={false} onCheckAgain={() => {}} />);
    expect(el.textContent).not.toContain('Specified native messaging host not found.');
    const more = Array.from(el.querySelectorAll('button')).find((b) => b.textContent === 'Technical details')!;
    await act(async () => more.click());
    expect(el.textContent).toContain('Specified native messaging host not found.');
  });

  it('a start problem does not tell the user to install again, and points to the log', async () => {
    const el = await render(<InstallMoly problem="start-timeout" checking={false} onCheckAgain={() => {}} />);
    expect(el.textContent).toContain('taking too long to start');
    expect(el.textContent).not.toContain('Moly needs a small app');
    expect(el.textContent).toContain('moly.log');
    expect(Array.from(el.querySelectorAll('button')).some((b) => b.textContent === 'Download latest version')).toBe(false);
  });

  it('an unrecognised extension offers the latest app', async () => {
    const el = await render(<InstallMoly problem="not-allowed" checking={false} onCheckAgain={() => {}} />);
    expect(el.textContent).toContain('does not recognise this extension');
    expect(Array.from(el.querySelectorAll('button')).some((b) => b.textContent === 'Download latest version')).toBe(true);
  });

  it('while checking, the button is disabled', async () => {
    const el = await render(<InstallMoly problem="unknown" checking={true} onCheckAgain={() => {}} />);
    const button = Array.from(el.querySelectorAll('button')).find((b) => b.textContent === 'Checking...') as HTMLButtonElement;
    expect(button.disabled).toBe(true);
  });
});
