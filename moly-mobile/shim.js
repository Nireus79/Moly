// Stand-ins for browser features that old Android web views (Chrome 61, e.g. a phone without Play services updates) lack.
(function () {
  if (!Promise.prototype.finally) Promise.prototype.finally = function (f) { return this.then((v) => Promise.resolve(f()).then(() => v), (e) => Promise.resolve(f()).then(() => { throw e; })); };
  if (typeof AbortSignal !== 'undefined' && !AbortSignal.timeout) AbortSignal.timeout = function (ms) { const c = new AbortController(); setTimeout(() => c.abort(), ms); return c.signal; };
  if (!Object.fromEntries) Object.fromEntries = function (it) { const o = {}; for (const [k, v] of it) o[k] = v; return o; };
  if (!Array.prototype.flat) Array.prototype.flat = function (d) { d = d === undefined ? 1 : d; return d < 1 ? this.slice() : this.reduce((a, x) => a.concat(Array.isArray(x) ? x.flat(d - 1) : x), []); };
  if (!Array.prototype.flatMap) Array.prototype.flatMap = function (f, t) { return this.map(f, t).flat(); };
  if (!Array.prototype.at) Array.prototype.at = function (i) { i = Math.trunc(i) || 0; if (i < 0) i += this.length; return this[i]; };
  if (!String.prototype.at) String.prototype.at = function (i) { i = Math.trunc(i) || 0; if (i < 0) i += this.length; return this.charAt(i); };
  if (!String.prototype.replaceAll) String.prototype.replaceAll = function (a, b) { return a instanceof RegExp ? this.replace(a, b) : this.split(a).join(b); };
  if (typeof structuredClone === 'undefined') window.structuredClone = function (v) { return JSON.parse(JSON.stringify(v)); };
})();

// Phone replacement for the parts of chrome.* the Moly screens use. Storage lives in localStorage; the "background"
// answers MOLY_ENSURE_BACKEND by asking the native plugin (MolyEngine) to start the engine and give {url, token}.
(function () {
  function area(prefix) {
    const listeners = [];
    const read = () => { try { return JSON.parse(localStorage.getItem(prefix) || '{}'); } catch (e) { return {}; } };
    const write = (o) => localStorage.setItem(prefix, JSON.stringify(o));
    return {
      get: async (keys) => {
        const all = read();
        if (keys == null) return all;
        if (typeof keys === 'string') keys = [keys];
        if (Array.isArray(keys)) { const out = {}; keys.forEach((k) => { if (k in all) out[k] = all[k]; }); return out; }
        const out = {}; Object.keys(keys).forEach((k) => { out[k] = k in all ? all[k] : keys[k]; }); return out;
      },
      set: async (items) => {
        const all = read(); const changes = {};
        Object.keys(items).forEach((k) => { changes[k] = { oldValue: all[k], newValue: items[k] }; all[k] = items[k]; });
        write(all); listeners.forEach((l) => l(changes, prefix));
      },
      remove: async (keys) => { const all = read(); [].concat(keys).forEach((k) => delete all[k]); write(all); },
      clear: async () => write({}),
      onChanged: { addListener: (l) => listeners.push(l), removeListener: (l) => { const i = listeners.indexOf(l); if (i >= 0) listeners.splice(i, 1); } },
    };
  }
  const local = area('moly.local');
  const session = area('moly.session');
  const noop = { addListener() {}, removeListener() {} };
  // This install's engine port: random, chosen once and kept, so no other app (or other copy of Moly) is already on it. The
  // screens read it from window.molyPhone.enginePort when they load; the native side starts the engine on it.
  const PORT_KEY = 'moly.enginePort';
  function pickPort(fresh) {
    let p = fresh ? 0 : Number(localStorage.getItem(PORT_KEY));
    if (!(p >= 20000 && p < 32000)) { p = 20000 + Math.floor(Math.random() * 12000); try { localStorage.setItem(PORT_KEY, String(p)); } catch (e) {} }
    return p;
  }
  const enginePort = pickPort(false);
  local.remove('detectedBackendUrl'); // a cached address from an older port
  async function ensureBackend() {
    try {
      const plugin = window.Capacitor && window.Capacitor.Plugins && window.Capacitor.Plugins.MolyEngine;
      if (!plugin) return { ok: false, error: 'No engine plugin (not running inside the app).' };
      const c = await plugin.start({ port: enginePort });
      await session.set({ molyConnection: { url: c.url, token: c.token, version: c.version || '' } });
      return { ok: true, url: c.url, version: c.version };
    } catch (e) {
      const text = String(e && e.message || e);
      // The engine could not take its port (another program has it): choose another one and load the screens again, twice at most.
      if (text.indexOf('ENGINE_STOPPED') >= 0) {
        const tries = Number(sessionStorage.getItem('moly.portTries') || '0');
        if (tries < 2) { sessionStorage.setItem('moly.portTries', String(tries + 1)); pickPort(true); location.reload(); }
      }
      return { ok: false, error: text };
    }
  }
  // The phone's service stops the engine after a few minutes with the app off the screen. When the person comes back (the native
  // side says so with the window event molyReturned), start it again and take its new token before they type, instead of
  // failing on their first message.
  window.addEventListener('molyReturned', () => { ensureBackend(); });
  // Lets the screens know they run inside the phone app, and read the clipboard when a Paste button is tapped.
  window.molyPhone = {
    enginePort,
    readClipboard: async () => {
      const c = window.Capacitor && window.Capacitor.Plugins && window.Capacitor.Plugins.Clipboard;
      if (!c) return '';
      try { const r = await c.read(); return (r && r.value) || ''; } catch (e) { return ''; } // the plugin rejects when the clipboard is empty
    },
    // How long the app was off the screen before it last came back, in ms (0 when not known).
    awayMs: async () => {
      const m = window.Capacitor && window.Capacitor.Plugins && window.Capacitor.Plugins.MolyEngine;
      if (!m) return 0;
      try { const r = await m.away(); return (r && Number(r.ms)) || 0; } catch (e) { return 0; }
    },
    // Text shared to Moly from another app: the native side hands it over once.
    takeSharedText: async () => {
      const m = window.Capacitor && window.Capacitor.Plugins && window.Capacitor.Plugins.MolyEngine;
      if (!m) return '';
      try { const r = await m.getSharedText(); return (r && r.text) || ''; } catch (e) { return ''; }
    },
    // "Report a problem": the text is built on the phone, shown, and shared only when the person taps Share.
    buildReport: async () => {
      const m = window.Capacitor && window.Capacitor.Plugins && window.Capacitor.Plugins.MolyEngine;
      if (!m) throw new Error('no engine bridge');
      const r = await m.buildReport();
      return (r && r.text) || '';
    },
    // "Your data": save a copy through Android's "save as" picker, restore through its file picker.
    saveFile: async (name, text) => {
      const m = window.Capacitor && window.Capacitor.Plugins && window.Capacitor.Plugins.MolyEngine;
      if (!m) throw new Error('no engine bridge');
      const r = await m.saveFile({ name, text });
      return !!(r && r.saved);
    },
    pickFile: async () => {
      const m = window.Capacitor && window.Capacitor.Plugins && window.Capacitor.Plugins.MolyEngine;
      if (!m) throw new Error('no engine bridge');
      const r = await m.pickFile();
      return r && typeof r.text === 'string' ? r.text : null;
    },
    shareReport: async (text) => {
      const m = window.Capacitor && window.Capacitor.Plugins && window.Capacitor.Plugins.MolyEngine;
      if (!m) throw new Error('no engine bridge');
      await m.shareReport({ text });
    },
  };
  window.chrome = {
    storage: { local, session },
    runtime: {
      id: 'moly-mobile',
      lastError: undefined,
      getManifest: () => ({ version: '0.1.0-beta.2', name: 'Moly' }),
      sendMessage: async (msg) => {
        if (msg && msg.type === 'MOLY_ENSURE_BACKEND') return ensureBackend();
        return undefined;
      },
      onMessage: noop, onStartup: noop, onInstalled: noop,
      connectNative: () => { throw new Error('no native host on the phone'); },
    },
    management: { getSelf: async () => ({ installType: 'normal' }) },
    sidePanel: { setOptions: async () => {}, open: async () => {} },
    action: { onClicked: noop },
  };
})();
