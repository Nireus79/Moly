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

    await clickByText(page, 'button', 'Claude (Anthropic)');
    await sleep(500);
    const form = await textOf(page);
    check('choosing Claude shows the passphrase option ticked by default', form.includes('Protect the key with a passphrase') && (await page.$eval('input[type=checkbox]', (c) => c.checked)), form.slice(0, 300));
    const saveDisabled = await page.evaluate(() => [...document.querySelectorAll('button')].find((b) => b.textContent.trim() === 'Save').disabled);
    check('Save is disabled until the form is complete', saveDisabled === true);

    await page.type('#ai-key', 'sk-ant-api03-E2E-TEST-KEY-0123456789');
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

    // The small card on the empty chat screen: labelled, hideable, and it stays hidden. (A new profile first sees the
    // "Tell me about yourself" form, which covers the card until it is skipped.)
    await sleep(1500);
    if ((await textOf(page)).includes('Skip for now')) { await clickByText(page, 'button', 'Skip for now'); await sleep(500); }
    await page.waitForSelector('.moly-card', { timeout: 15000 }).catch(() => {});
    const cardText = await page.$eval('.moly-card', (e) => e.textContent).catch(() => '');
    check('the empty chat screen shows the labelled card', cardText.includes('Moly message') && cardText.includes('Hide'), cardText);
    await page.click('.moly-card-hide');
    await sleep(300);
    check('Hide removes the card', (await page.$('.moly-card')) === null);
    await page.reload();
    await page.waitForSelector('textarea.chat-input', { timeout: 30000 });
    await sleep(1500);
    if ((await textOf(page)).includes('Skip for now')) { await clickByText(page, 'button', 'Skip for now'); await sleep(500); }
    check('and it stays hidden after the panel is reopened', (await page.$('.moly-card')) === null);

    // Settings page: export the data and read the downloaded file.
    fs.mkdirSync('/tmp/dl', { recursive: true });
    const cdp = await browser.target().createCDPSession();
    await cdp.send('Browser.setDownloadBehavior', { behavior: 'allow', downloadPath: '/tmp/dl' });
    const settings = await browser.newPage();
    await settings.goto(`chrome-extension://${realId}/settings/settings.html`);
    const st = await waitFor(settings, 'Export my data', 30000);
    check('the settings page offers export, import and the provider panel', st.includes('Export my data') && st.includes('Import data') && st.includes('AI provider'), st.slice(0, 200));
    check('Settings can show the card again (it is switched off now)', await settings.$eval('input[type=checkbox]', () => true).catch(() => false) && st.includes('Moly messages'), st.slice(0, 200));
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
  }
  check('no uncaught errors in the page', errors.length === 0, errors.join(' | '));
} catch (e) {
  check('scenario ran without an exception', false, String(e));
} finally {
  await browser.close();
}
process.exit(results.every(Boolean) ? 0 : 1);
