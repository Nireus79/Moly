import { describe, it, expect, afterEach } from 'vitest';
import React from 'react';
import { createRoot } from 'react-dom/client';
import { act } from 'react-dom/test-utils';
import { AboutPanel } from '../AboutPanel';
import { ManualPanel } from '../ManualPanel';
import { SUPPORT_EMAIL } from '@/config/links';

(globalThis as any).IS_REACT_ACT_ENVIRONMENT = true;

let host: HTMLDivElement | null = null;
async function render(node: React.ReactElement) {
  host = document.createElement('div');
  document.body.appendChild(host);
  await act(async () => createRoot(host!).render(node));
  return host;
}
afterEach(() => { host?.remove(); host = null; });

describe('About Moly panel', () => {
  it('gives the contact details, the developer, the license and the sponsor link', async () => {
    const el = await render(<AboutPanel />);
    const t = el.textContent || '';
    expect(t).toContain('Μώλυ');
    expect(t).toContain('Hermes Software');
    expect(t).toContain('Socrates AI');
    expect(t).toContain(SUPPORT_EMAIL);
    expect(t).toContain('MIT');
    expect(t).toContain('not a crisis, medical or legal service');
    expect(el.querySelector(`a[href="mailto:${SUPPORT_EMAIL}"]`)).not.toBeNull();
    expect(el.querySelector('a[href="https://github.com/sponsors/Nireus79"]')).not.toBeNull();
    expect(el.querySelector('a[href="https://nireus79.github.io/Moly/privacy.html"]')).not.toBeNull();
  });

  it('opens every web link in a new tab without giving the page access to Moly', async () => {
    const el = await render(<AboutPanel />);
    const web = [...el.querySelectorAll<HTMLAnchorElement>('a')].filter((a) => a.href.startsWith('http'));
    expect(web.length).toBeGreaterThan(5);
    for (const a of web) {
      expect(a.href.startsWith('https://')).toBe(true);
      expect(a.getAttribute('target')).toBe('_blank');
      expect(a.getAttribute('rel')).toContain('noopener');
    }
  });
});

describe('User manual panel', () => {
  it('covers the whole path from the first start to troubleshooting', async () => {
    const el = await render(<ManualPanel />);
    const titles = [...el.querySelectorAll('summary')].map((s) => s.textContent);
    for (const wanted of ['What Moly does', 'Getting started', 'Writing a message', 'Skip questions', 'Previous conversations', 'Pending insights', 'Your data and your privacy', 'Something is wrong', 'What Moly is not']) {
      expect(titles).toContain(wanted);
    }
  });

  it('opens with the first section expanded and the rest collapsed', async () => {
    const el = await render(<ManualPanel />);
    const sections = [...el.querySelectorAll('details')];
    expect(sections[0].open).toBe(true);
    expect(sections.slice(1).every((d) => !d.open)).toBe(true);
  });

  it('tells nothing that contradicts the product: no crisis help lines, no account', async () => {
    const t = (await render(<ManualPanel />)).textContent || '';
    expect(t).not.toMatch(/hotline|helpline|suicide/i);
    expect(t).toContain('needs no account');
  });
});
