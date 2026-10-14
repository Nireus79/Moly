import { describe, it, expect, beforeEach, vi } from 'vitest';

const URL_ = 'http://127.0.0.1:11436';

function mockSession(initial: Record<string, any>) {
  const store: Record<string, any> = { ...initial };
  const listeners: Array<(c: any) => void> = [];
  (window as any).chrome.storage.session = {
    get: vi.fn(async (key: string) => ({ [key]: store[key] })),
    set: vi.fn(async (data: Record<string, any>) => Object.assign(store, data)),
    onChanged: { addListener: (fn: (c: any) => void) => listeners.push(fn) },
  };
  return {
    store,
    change(key: string, value: any) {
      store[key] = value;
      listeners.forEach((fn) => fn({ [key]: { newValue: value } }));
    },
  };
}

describe('backendAuth: token header on backend requests', () => {
  let calls: Array<{ url: string; headers: Headers }>;
  let responses: Response[];

  beforeEach(() => {
    vi.resetModules();
    calls = [];
    responses = [];
    (window as any).fetch = vi.fn(async (input: any, init?: RequestInit) => {
      calls.push({ url: typeof input === 'string' ? input : input.url, headers: new Headers(init?.headers) });
      return responses.shift() ?? new Response('{}', { status: 200 });
    });
  });

  it('adds X-Moly-Token to requests to the backend and keeps the caller headers', async () => {
    mockSession({ molyConnection: { url: URL_, token: 'tok1', version: '2.1' } });
    const { installBackendAuth } = await import('../backendAuth');
    await installBackendAuth();
    await fetch(`${URL_}/api/conversations`, { headers: { Authorization: 'Bearer abc', 'Content-Type': 'application/json' } });
    expect(calls).toHaveLength(1);
    expect(calls[0].headers.get('X-Moly-Token')).toBe('tok1');
    expect(calls[0].headers.get('Authorization')).toBe('Bearer abc');
    expect(calls[0].headers.get('Content-Type')).toBe('application/json');
  });

  it('never sends the token to other addresses', async () => {
    mockSession({ molyConnection: { url: URL_, token: 'tok1', version: '2.1' } });
    const { installBackendAuth } = await import('../backendAuth');
    await installBackendAuth();
    for (const other of ['http://127.0.0.1:11434/api/tags', 'https://api.anthropic.com/v1/models', 'http://127.0.0.1:11436.evil.example/api/x', 'http://127.0.0.1:114360/api/x']) {
      await fetch(other);
    }
    expect(calls).toHaveLength(4);
    calls.forEach((c) => expect(c.headers.has('X-Moly-Token')).toBe(false));
  });

  it('treats localhost and 127.0.0.1 with the same port as the same backend, and updates BACKEND_CONFIG', async () => {
    mockSession({ molyConnection: { url: 'http://127.0.0.1:11999', token: 'tokx', version: '' } });
    const { installBackendAuth } = await import('../backendAuth');
    const { BACKEND_CONFIG } = await import('../backendConfig');
    await installBackendAuth();
    expect(BACKEND_CONFIG.baseUrl).toBe('http://127.0.0.1:11999');
    await fetch('http://localhost:11999/api/about-me');
    await fetch('http://localhost:11998/api/about-me'); // another port: not the backend
    expect(calls[0].headers.get('X-Moly-Token')).toBe('tokx');
    expect(calls[1].headers.has('X-Moly-Token')).toBe(false);
  });

  it('leaves requests alone when there is no connection (development backend)', async () => {
    mockSession({});
    const { installBackendAuth } = await import('../backendAuth');
    await installBackendAuth();
    await fetch(`${URL_}/api/status`);
    expect(calls[0].headers.has('X-Moly-Token')).toBe(false);
  });

  it('picks up a token that appears later', async () => {
    const session = mockSession({});
    const { installBackendAuth } = await import('../backendAuth');
    await installBackendAuth();
    session.change('molyConnection', { url: URL_, token: 'late', version: '' });
    await fetch(`${URL_}/api/messages`);
    expect(calls[0].headers.get('X-Moly-Token')).toBe('late');
  });

  it('refreshes the connection once on 401 and retries only when the token changed', async () => {
    const session = mockSession({ molyConnection: { url: URL_, token: 'old', version: '' } });
    (window as any).chrome.runtime.sendMessage = vi.fn(async () => {
      session.change('molyConnection', { url: URL_, token: 'new', version: '' });
      return { ok: true, url: URL_ };
    });
    responses.push(new Response('{}', { status: 401 }));
    const { installBackendAuth } = await import('../backendAuth');
    await installBackendAuth();
    const res = await fetch(`${URL_}/api/conversations`);
    expect(res.status).toBe(200);
    expect(calls.map((c) => c.headers.get('X-Moly-Token'))).toEqual(['old', 'new']);
  });

  it('does not retry a 401 that is not about the token (expired user session)', async () => {
    const session = mockSession({ molyConnection: { url: URL_, token: 'same', version: '' } });
    (window as any).chrome.runtime.sendMessage = vi.fn(async () => {
      session.change('molyConnection', { url: URL_, token: 'same', version: '' });
      return { ok: true, url: URL_ };
    });
    responses.push(new Response('{"error":"session expired"}', { status: 401 }));
    const { installBackendAuth } = await import('../backendAuth');
    await installBackendAuth();
    const res = await fetch(`${URL_}/api/conversations`);
    expect(res.status).toBe(401);
    expect(calls).toHaveLength(1);
  });

  it('ensureBackendConnection reports a missing app instead of throwing', async () => {
    mockSession({});
    (window as any).chrome.runtime.sendMessage = vi.fn(async () => {
      throw new Error('Could not establish connection.');
    });
    const { ensureBackendConnection } = await import('../backendAuth');
    const r = await ensureBackendConnection();
    expect(r.ok).toBe(false);
    expect(r.error).toContain('Could not establish');
  });
});

describe('backendAuth: renewing the local session', () => {
  let calls: Array<{ url: string; headers: Headers }>;
  let responses: Response[];

  beforeEach(() => {
    vi.resetModules();
    calls = [];
    responses = [];
    (window as any).fetch = vi.fn(async (input: any, init?: RequestInit) => {
      calls.push({ url: typeof input === 'string' ? input : input.url, headers: new Headers(init?.headers) });
      return responses.shift() ?? new Response('{}', { status: 200 });
    });
    mockSession({ molyConnection: { url: URL_, token: 'tok', version: '' } });
    (window as any).chrome.runtime.sendMessage = vi.fn(async () => ({ ok: true, url: URL_ })); // connection unchanged
  });

  it('repeats a request once with a new session when the session ended', async () => {
    const { installBackendAuth, setSessionRenewer } = await import('../backendAuth');
    await installBackendAuth();
    setSessionRenewer(async () => 'new-session');
    responses.push(new Response('{"error":"expired"}', { status: 401 }));
    const res = await fetch(`${URL_}/api/conversations`, { headers: { Authorization: 'Bearer old-session' } });
    expect(res.status).toBe(200);
    expect(calls.map((c) => c.headers.get('Authorization'))).toEqual(['Bearer old-session', 'Bearer new-session']);
    expect(calls[1].headers.get('X-Moly-Token')).toBe('tok');
  });

  it('does nothing when no renewer is registered (an account with a password)', async () => {
    const { installBackendAuth } = await import('../backendAuth');
    await installBackendAuth();
    responses.push(new Response('{}', { status: 401 }));
    const res = await fetch(`${URL_}/api/conversations`, { headers: { Authorization: 'Bearer old' } });
    expect(res.status).toBe(401);
    expect(calls).toHaveLength(1);
  });

  it('never renews for the auth routes or for requests without a user session', async () => {
    const { installBackendAuth, setSessionRenewer } = await import('../backendAuth');
    await installBackendAuth();
    const renew = vi.fn(async () => 'x');
    setSessionRenewer(renew);
    responses.push(new Response('{}', { status: 401 }), new Response('{}', { status: 401 }));
    await fetch(`${URL_}/api/auth/login`, { headers: { Authorization: 'Bearer old' } });
    await fetch(`${URL_}/api/conversations`);
    expect(renew).not.toHaveBeenCalled();
  });

  it('gives back the original 401 when no new session can be had, and does not loop', async () => {
    const { installBackendAuth, setSessionRenewer } = await import('../backendAuth');
    await installBackendAuth();
    setSessionRenewer(async () => null);
    responses.push(new Response('{}', { status: 401 }));
    const res = await fetch(`${URL_}/api/conversations`, { headers: { Authorization: 'Bearer old' } });
    expect(res.status).toBe(401);
    expect(calls).toHaveLength(1);
  });

  // Found live (2026-10-14): after the profile was deleted the open panel kept a session the app no longer knew, and every
  // request failed. A session the app rejects is forgotten, so the panel starts a new profile or shows the login.
  it('tells the app of the extension when the user session is rejected and cannot be renewed', async () => {
    const { installBackendAuth, setSessionRejectedHandler } = await import('../backendAuth');
    await installBackendAuth();
    const handler = vi.fn();
    setSessionRejectedHandler(handler);
    responses.push(new Response('{"error":"Invalid token"}', { status: 401 }));
    await fetch(`${URL_}/api/conversations`, { headers: { Authorization: 'Bearer gone' } });
    expect(handler).toHaveBeenCalledTimes(1);
  });

  it('does not forget the session for the auth routes, for requests without a session, or when the app token is the problem', async () => {
    const { installBackendAuth, setSessionRejectedHandler } = await import('../backendAuth');
    await installBackendAuth();
    const handler = vi.fn();
    setSessionRejectedHandler(handler);
    responses.push(new Response('{}', { status: 401 }), new Response('{}', { status: 401 }), new Response('{"error":"missing or wrong token"}', { status: 401 }));
    await fetch(`${URL_}/api/auth/login`, { headers: { Authorization: 'Bearer x' } });
    await fetch(`${URL_}/api/conversations`);
    await fetch(`${URL_}/api/conversations`, { headers: { Authorization: 'Bearer x' } });
    expect(handler).not.toHaveBeenCalled();
  });

  it('a renewed session is not a rejected one', async () => {
    const { installBackendAuth, setSessionRenewer, setSessionRejectedHandler } = await import('../backendAuth');
    await installBackendAuth();
    const handler = vi.fn();
    setSessionRejectedHandler(handler);
    setSessionRenewer(async () => 'new-session');
    responses.push(new Response('{"error":"Invalid token"}', { status: 401 }));
    const res = await fetch(`${URL_}/api/conversations`, { headers: { Authorization: 'Bearer old' } });
    expect(res.status).toBe(200);
    expect(handler).not.toHaveBeenCalled();
  });
});

