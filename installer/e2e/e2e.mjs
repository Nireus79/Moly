// Drives a real Chromium: loads the built extension, opens the Moly side panel page in a tab and checks what a person
// would see in each situation. Usage: node e2e.mjs <scenario>   (see run.sh for the setup of each scenario)
import puppeteer from 'puppeteer-core';
import fs from 'node:fs';

const scenario = process.argv[2];
const EXT = '/ext';
const ID = process.env.EXT_ID;
const results = [];
const check = (name, ok, detail = '') => { results.push(ok); console.log(`${ok ? 'PASS' : 'FAIL'}  ${name}${ok ? '' : '  -> ' + detail}`); };
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));

async function textOf(page) { return (await page.evaluate(() => document.body.innerText)).replace(/\s+/g, ' ').trim(); }
async function waitFor(page, needle, ms = 60000) {
  const end = Date.now() + ms;
  let t = '';
  while (Date.now() < end) { t = await textOf(page); if (t.includes(needle)) return t; await sleep(500); }
  return t;
}
async function waitForGone(page, needles, ms = 30000) {
  const end = Date.now() + ms;
  let t = '';
  while (Date.now() < end) { t = await textOf(page); if (needles.every((n) => !t.includes(n)) && t.length > 0) return t; await sleep(500); }
  return t;
}
async function clickByText(page, selector, text) {
  const handle = await page.evaluateHandle((sel, txt) => [...document.querySelectorAll(sel)].find((e) => e.textContent.trim().startsWith(txt)), selector, text);
  const el = handle.asElement();
  if (!el) throw new Error(`no ${selector} with text "${text}"`);
  await el.click();
}

const browser = await puppeteer.launch({
  executablePath: '/usr/bin/chromium',
  headless: 'new',
  // The browser looks for native messaging hosts in its profile folder; puppeteer would use a temporary one.
  userDataDir: '/home/tester/.config/chromium',
  args: ['--no-sandbox', `--disable-extensions-except=${EXT}`, `--load-extension=${EXT}`, '--no-first-run', '--disable-gpu'],
});
const errors = [];
try {
  // The extension's service worker shows the real ID the browser gave it.
  let worker;
  for (let i = 0; i < 40 && !worker; i++) { worker = browser.targets().find((t) => t.type() === 'service_worker' && t.url().startsWith('chrome-extension://')); if (!worker) await sleep(500); }
  check('the extension loaded', !!worker, 'no service worker');
  const realId = worker ? new URL(worker.url()).host : '';
  check(`the extension ID is the expected one (${ID})`, realId === ID, `got ${realId}`);

  const page = await browser.newPage();
  page.on('pageerror', (e) => errors.push(String(e)));
  await page.goto(`chrome-extension://${realId}/sidebar/sidebar.html`);

  if (scenario === 'missing') {
    const t = await waitFor(page, 'Moly needs a small app');
    check('no app installed: the install screen is shown', t.includes('Install Moly for Linux') && t.includes('Moly needs a small app on your computer'), t.slice(0, 300));
    check('it offers Download and Check again', t.includes('Download') && t.includes('Check again'), t);
    check('support address shown', t.includes('Hermes_creative@proton.me'), t);
  } else if (scenario === 'notallowed') {
    const t = await waitFor(page, 'does not recognise this extension');
    check('app registered for another extension: says so', t.includes('does not recognise this extension'), t.slice(0, 300));
  } else if (scenario === 'firstrun') {
    const t = await waitFor(page, 'Welcome to Moly');
    check('the app started through the native host and the first-run screen appears', t.includes('Welcome to Moly'), t.slice(0, 400));
    check('the not-a-crisis-service notice is shown', t.includes('not a crisis, medical or legal service'), t);
    check('the three providers are offered', t.includes('Ollama') && t.includes('Claude') && t.includes('OpenAI'), t);

    await clickByText(page, 'button', 'Claude');
    await sleep(500);
    const form = await textOf(page);
    check('choosing Claude shows the passphrase option ticked by default', form.includes('Protect the key with a passphrase') && (await page.$eval('input[type=checkbox]', (c) => c.checked)), form.slice(0, 300));
    const saveDisabled = await page.evaluate(() => [...document.querySelectorAll('button')].find((b) => b.textContent.trim() === 'Save').disabled);
    check('Save is disabled until the form is complete', saveDisabled === true);

    await page.type('#ai-key', 'sk-ant-api03-E2E-TEST-KEY-0123456789');
    await page.type('#ai-model', 'claude-sonnet-5-5'); // the container has no network: the list of models cannot be fetched, a model is typed
    await page.type('#ai-pass', 'correct horse battery');
    await page.type('#ai-pass2', 'correct horse battery');
    const boxes = await page.$$('input[type=checkbox]');
    await boxes[1].click(); // consent
    await sleep(300);
    const saveEnabled = await page.evaluate(() => ![...document.querySelectorAll('button')].find((b) => b.textContent.trim() === 'Save').disabled);
    check('Save becomes available when key, passphrase and consent are given', saveEnabled);
    await clickByText(page, 'button', 'Save');
    const after = await waitForGone(page, ['Welcome to Moly', 'Opening Moly'], 30000);
    console.log('   screen after saving:', after.slice(0, 200));
    check('after saving, the app opens by itself: no login or registration screen', after.length > 0 && !/Login|Log in|Sign in|Register|password/i.test(after), after.slice(0, 300));

    const stored = await page.evaluate(() => JSON.stringify([localStorage, Object.fromEntries(Object.entries(localStorage))]));
    const local = await page.evaluate(() => new Promise((r) => chrome.storage.local.get(null, r)));
    check('no API key in the browser storage', !JSON.stringify(local).includes('E2E-TEST-KEY') && !stored.includes('E2E-TEST-KEY'));
    fs.writeFileSync('/tmp/e2e-saved', 'yes');
  } else if (scenario === 'devlogin') {
    // A development backend started by hand (no native host, no installed app): the person registers an account, then saves a
    // Claude key without a passphrase. The AI provider belongs to that account, so every settings call must carry its session.
    const settingsCalls = [];
    page.on('response', (r) => { if (r.url().includes('/api/settings/llm')) settingsCalls.push(`${r.request().method()} ${new URL(r.url()).pathname} ${r.status()}`); });
    // the browser profile is shared with the earlier scenarios: forget the local-profile session they left
    await page.evaluate(() => { localStorage.clear(); sessionStorage.clear(); });
    await page.reload();
    let t = await waitFor(page, 'Register', 30000);
    check('a development backend shows the login screen', t.includes('Log in') && t.includes('Register'), t.slice(0, 300));
    await clickByText(page, 'button[role=tab]', 'Register');
    await sleep(300);
    await page.type('#reg-name', 'Dev Tester');
    await page.type('#reg-email', 'dev-tester@example.invalid');
    await page.type('#reg-password', 'a-long-test-password-123');
    await page.type('#reg-confirm', 'a-long-test-password-123');
    await clickByText(page, 'button', 'Create account');
    t = await waitFor(page, 'Welcome to Moly', 40000);
    check('after the login the first-run provider screen follows (the provider belongs to the signed-in user)', t.includes('Welcome to Moly'), t.slice(0, 400));
    await clickByText(page, 'button', 'Claude');
    await sleep(500);
    await page.type('#ai-key', 'sk-ant-api03-E2E-DEV-KEY-0123456789');
    await page.type('#ai-model', 'claude-sonnet-5-5');
    const boxes = await page.$$('input[type=checkbox]');
    await boxes[0].click(); // untick the passphrase
    await sleep(200);
    const warn = await textOf(page);
    check('without the passphrase the form says the key is saved without one', warn.includes('saved without a passphrase'), warn.slice(0, 400));
    check('the passphrase fields are gone', !(await page.$('#ai-pass')));
    await boxes[1].click(); // consent
    await sleep(300);
    const saveEnabled = await page.evaluate(() => ![...document.querySelectorAll('button')].find((b) => b.textContent.trim() === 'Save').disabled);
    check('Save is available with a key and consent, without any passphrase', saveEnabled);
    await clickByText(page, 'button', 'Save');
    const after = await waitForGone(page, ['Welcome to Moly'], 30000);
    console.log('   screen after saving:', after.slice(0, 200));
    console.log('   settings calls:', settingsCalls.join(' | '));
    check('saving does not say the app is unreachable or that the user is not signed in', !/not reachable|not signed in/i.test(after), after.slice(0, 300));
    check('after saving, the chat opens', !!(await page.$('textarea.chat-input')) || after.length > 0 && !after.includes('Welcome to Moly'), after.slice(0, 300));
    check('every settings call carried the session of the account', settingsCalls.length > 0 && settingsCalls.every((c) => !c.endsWith(' 401')), settingsCalls.join(' | '));
    // Found live (2026-10-14): the profile was deleted while the panel stayed open; the panel kept a session the app no longer
    // knew and every request failed. The panel must go back to the login.
    const deletedStatus = await page.evaluate(async () => {
      const s = JSON.parse(sessionStorage.getItem('moly_session') || '{}');
      const r = await fetch('http://127.0.0.1:11436/api/user/delete', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + s.sessionId },
        body: JSON.stringify({ password: 'a-long-test-password-123' }),
      });
      return r.status;
    });
    check('the account is deleted on the app', deletedStatus === 200, String(deletedStatus));
    await page.evaluate(() => window.dispatchEvent(new Event('focus'))); // the panel asks the app again, as it does when it gets the focus
    t = await waitFor(page, 'Register', 30000);
    check('the open panel goes back to the login screen once the app no longer knows the session', t.includes('Log in') && t.includes('Register'), t.slice(0, 300));
    check('the dead session is gone from the browser', await page.evaluate(() => !sessionStorage.getItem('moly_session') && !localStorage.getItem('moly_session')));
  } else if (scenario === 'chat') {
    // A real conversation: provider = a small Ollama model in another container, then one message and its answer.
    // With a running Ollama the backend finds it by itself and no setup is needed; otherwise the first-run screen appears.
    let first = '';
    for (let i = 0; i < 120 && !first; i++) {
      if (await page.$('textarea.chat-input')) first = 'chat';
      else if ((await textOf(page)).includes('Welcome to Moly')) first = 'setup';
      else await sleep(500);
    }
    check('either the chat or the first-run screen appears', first !== '', (await textOf(page)).slice(0, 300));
    console.log(`   starting point: ${first === 'chat' ? 'Ollama found automatically, chat open' : 'first-run setup'}`);
    if (first === 'setup') {
      await page.click('#ai-endpoint', { clickCount: 3 });
      await page.type('#ai-endpoint', process.env.OLLAMA_URL);
      await page.waitForSelector('#ai-ollama-model', { timeout: 30000 });
      const models = await page.$$eval('#ai-ollama-model option', (o) => o.map((x) => x.value));
      check('the app lists the models of the Ollama server', models.length > 0, JSON.stringify(models));
      await clickByText(page, 'button', 'Save');
    }
    await page.waitForSelector('textarea.chat-input', { timeout: 30000 });
    check('the chat is open (no login, no registration)', true);
    // A new profile is first offered the "Tell me about yourself" form; it can be skipped.
    await sleep(1500);
    if ((await textOf(page)).includes('Skip for now')) {
      await clickByText(page, 'button', 'Skip for now');
      await sleep(500);
      check('the "about you" form can be skipped', !(await textOf(page)).includes('Skip for now'));
    }

    const question = process.env.CHAT_MESSAGE;
    await page.type('textarea.chat-input', question);
    await page.click('button.send-button');
    check('the message appears in the conversation', (await waitFor(page, question.slice(0, 20), 5000)).includes(question.slice(0, 20)));
    const started = Date.now();
    let reply = '';
    while (Date.now() - started < 600000) {
      reply = await page.evaluate(() => [...document.querySelectorAll('.message-assistant .message-text')].map((e) => e.textContent.trim()).filter(Boolean).pop() || '');
      if (reply) break;
      await sleep(2000);
    }
    console.log(`   answer after ${Math.round((Date.now() - started) / 1000)} s: ${reply.slice(0, 400)}`);
    check('Moly answers', reply.length > 0, 'no assistant message within 10 minutes');
    check('the answer is not an error message', reply.length > 0 && !/^(error|failed|something went wrong|backend)/i.test(reply) && !/not reachable|no AI provider/i.test(reply), reply);
    const bodyText = await textOf(page);
    check('no error banner on the page', !/Error:|Failed to|Backend server not responding/i.test(bodyText), bodyText.slice(0, 300));
  } else if (scenario === 'unlock') {
    const t = await waitFor(page, 'Unlock Moly');
    check('after a restart the sealed key asks for the passphrase', t.includes('Unlock Moly'), t.slice(0, 300));
    await page.type('#ai-unlock', 'not the passphrase');
    await clickByText(page, 'button', 'Unlock');
    const bad = await waitFor(page, 'not right', 15000);
    check('a wrong passphrase is refused with a message', bad.includes('That passphrase is not right'), bad.slice(0, 300));
    await page.click('#ai-unlock', { clickCount: 3 });
    await page.type('#ai-unlock', 'correct horse battery');
    await clickByText(page, 'button', 'Unlock');
    const ok = await waitForGone(page, ['Unlock Moly', 'Opening Moly'], 30000);
    check('the right passphrase unlocks and the app opens by itself', ok.length > 0 && !/Login|Log in|Sign in|Register|password/i.test(ok), ok.slice(0, 300));
    check('it is the same local profile as before', await page.evaluate(() => JSON.parse(localStorage.getItem('moly_session') || '{}').userId === 'user_local'));

    // The small card in its own area below the writing box: labelled, hideable, and it stays hidden. (A new profile first sees the
    // "Tell me about yourself" form, which covers the card until it is skipped.)
    await sleep(1500);
    if ((await textOf(page)).includes('Skip for now')) { await clickByText(page, 'button', 'Skip for now'); await sleep(500); }
    await page.waitForSelector('.moly-card', { timeout: 15000 }).catch(() => {});
    const cardText = await page.$eval('.moly-card', (e) => e.textContent).catch(() => '');
    check('the chat shows the card below the writing box', cardText.includes('Μώλυ was developed by Hermes') && cardText.includes('Hide') && cardText.includes('Sponsored'), cardText);
    await page.click('.moly-card-hide');
    await sleep(300);
    check('Hide removes the card', (await page.$('.moly-card')) === null);
    await page.reload();
    await page.waitForSelector('textarea.chat-input', { timeout: 30000 });
    await sleep(1500);
    if ((await textOf(page)).includes('Skip for now')) { await clickByText(page, 'button', 'Skip for now'); await sleep(500); }
    check('a reload of the open panel keeps it hidden', (await page.$('.moly-card')) === null);

    // Settings page: export the data and read the downloaded file.
    fs.mkdirSync('/tmp/dl', { recursive: true });
    const cdp = await browser.target().createCDPSession();
    await cdp.send('Browser.setDownloadBehavior', { behavior: 'allow', downloadPath: '/tmp/dl' });
    const settings = await browser.newPage();
    await settings.goto(`chrome-extension://${realId}/settings/settings.html`);
    const st = await waitFor(settings, 'Export my data', 30000);
    check('the settings page offers export, import and the provider panel', st.includes('Export my data') && st.includes('Import data') && st.includes('AI provider'), st.slice(0, 200));
    check('Settings has no switch for the card (Hide lasts only until Moly is opened again)', !st.includes('Show the card') && !st.includes('Moly messages'), st.slice(0, 200));
    await clickByText(settings, 'button', 'About Moly');
    const about = await waitFor(settings, 'Hermes Software', 10000);
    check('About Moly opens as its own panel with the contact details', about.includes('Hermes_creative@proton.me') && about.includes('Socrates AI') && about.includes('MIT') && !about.includes('Export my data'), about.slice(0, 200));
    await clickByText(settings, 'button', 'Settings');
    await clickByText(settings, 'button', 'User manual');
    const manual = await waitFor(settings, 'Getting started', 10000);
    check('the User manual opens as its own panel', manual.includes('What Moly does') && manual.includes('Something is wrong') && !manual.includes('Export my data'), manual.slice(0, 200));
    await clickByText(settings, 'button', 'Settings');
    await waitFor(settings, 'Export my data', 10000);
    const panel = await waitFor(settings, 'protected by your passphrase', 20000);
    check('the provider panel shows the sealed key without ever showing the key', panel.includes('protected by your passphrase') && !panel.includes('sk-ant'), panel.slice(0, 400));
    await clickByText(settings, 'button', 'Export my data');
    let file = '';
    for (let i = 0; i < 40 && !file; i++) { file = (fs.readdirSync('/tmp/dl').find((f) => f.startsWith('moly-export-') && f.endsWith('.json')) || ''); if (!file) await sleep(500); }
    check('a file moly-export-<date>.json was downloaded', !!file);
    if (file) {
      const data = JSON.parse(fs.readFileSync(`/tmp/dl/${file}`, 'utf8'));
      check('the export is for the local profile and has the expected shape', data.format === 'moly-export-1' && data.profile?.id === 'user_local' && typeof data.tables === 'object', JSON.stringify(data).slice(0, 200));
      check('the export holds no password hash, session token or API key', !/password_hash|sessions|sk-ant|E2E-TEST-KEY/.test(JSON.stringify(data)));
    }
  } else if (scenario === 'keystore-setup') {
    // This computer has no keychain and no MOLY_DB_KEY: the app must ask how to protect the database key.
    const t = await waitFor(page, 'Protect your data');
    check('no keychain: the screen "Protect your data" is shown, nothing was decided silently', t.includes('Protect your data') && t.includes('this computer has none'), t.slice(0, 300));
    check('the passphrase is the recommended choice', t.includes('A passphrase (recommended)'), t);
    check('no key file was created without asking', !fs.existsSync('/home/tester/.moly/db.key') && !fs.existsSync('/home/tester/.moly/keystore.json'));
    const disabled = await page.evaluate(() => [...document.querySelectorAll('button')].find((b) => b.textContent.trim() === 'Protect with this passphrase').disabled);
    check('the button waits for a valid passphrase', disabled === true);
    await clickByText(page, 'button', 'Use a key file instead');
    const warn = await textOf(page);
    check('the key file choice carries a clear warning and needs confirmation', warn.includes('less secure') && warn.includes('I understand and accept this'), warn.slice(0, 300));
    await clickByText(page, 'button', 'Use a passphrase instead');
    await page.type('#ks-pass', 'e2e keystore passphrase');
    await page.type('#ks-again', 'e2e keystore passphrase');
    await clickByText(page, 'button', 'Protect with this passphrase');
    const next = await waitFor(page, 'Welcome to Moly', 90000);
    check('after choosing, the database opens and the first-run screen follows', next.includes('Welcome to Moly'), next.slice(0, 300));
    const local = await page.evaluate(() => new Promise((r) => chrome.storage.local.get(null, r)));
    const stored = await page.evaluate(() => JSON.stringify(Object.fromEntries(Object.entries(localStorage))));
    check('the passphrase is not stored in the browser', !JSON.stringify(local).includes('e2e keystore passphrase') && !stored.includes('e2e keystore passphrase'));
    check('the sealed key, not the key, is on disk', fs.readFileSync('/home/tester/.moly/keystore.json', 'utf8').includes('"backend": "passphrase"') && !fs.existsSync('/home/tester/.moly/db.key'));
  } else if (scenario === 'keystore-unlock') {
    const t = await waitFor(page, 'Unlock Moly');
    check('after a restart the protected data asks for the passphrase', t.includes('Unlock Moly') && t.includes('protected by a passphrase'), t.slice(0, 300));
    await page.type('#ks-unlock', 'not the passphrase');
    await clickByText(page, 'button', 'Unlock');
    const bad = await waitFor(page, 'not right', 20000);
    check('a wrong passphrase is refused', bad.includes('That passphrase is not right'), bad.slice(0, 300));
    await page.click('#ks-unlock', { clickCount: 3 });
    await page.type('#ks-unlock', 'e2e keystore passphrase');
    await clickByText(page, 'button', 'Unlock');
    const ok = await waitFor(page, 'Welcome to Moly', 90000);
    check('the right passphrase opens the same data', ok.includes('Welcome to Moly'), ok.slice(0, 300));
  }
  check('no uncaught errors in the page', errors.length === 0, errors.join(' | '));
} catch (e) {
  check('scenario ran without an exception', false, String(e));
} finally {
  await browser.close();
}
process.exit(results.every(Boolean) ? 0 : 1);
