import React from 'react';
import { ISSUES_URL, LOG_HINT, SUPPORT_EMAIL } from '@/config/links';
import { Icon } from '@/components/Icon';
import { isPhoneApp } from '@/utils/platform';

const Section: React.FC<{ title: string; open?: boolean; children: React.ReactNode }> = ({ title, open, children }) => (
  <details className="manual-section" open={open}>
    <summary>{title}</summary>
    <div className="manual-body">{children}</div>
  </details>
);

/** The user's manual: what Moly does and how to use it, in the order a person meets it. Opened from Settings. */
export const ManualPanel: React.FC = () => {
  const phone = isPhoneApp();
  return (
  <div className="doc-panel">
    <section className="settings-section">
      <h2>User manual</h2>
      <p className="section-description">How to use Moly, from the first start to the last conversation. Open a section to read it.</p>
    </section>

    <div className="manual">
      <Section title="What Moly does" open>
        <p>
          You tell Moly what you want to say, to whom and why. Moly asks one question at a time about what is missing, for example who the
          person is or what you hope will happen. When it knows enough, it helps you write the message. Moly never decides for you: the
          words are yours to change, copy and send, and you can skip its questions whenever you wish.
        </p>
        <p>
          Moly writes its questions and replies in the language you write in. Its buttons, settings and this manual are in English for now.
        </p>
      </Section>

      {phone ? (
        <>
          <Section title="Getting started">
            <ol>
              <li>Open Moly and choose Claude or OpenAI: Moly works with your own account at one of them.</li>
              <li>No key yet? Under the key box, tap <strong>Get a Claude key</strong> or <strong>Get an OpenAI key</strong>: it opens the provider's page, with the steps beside it. A chat subscription is not an API key. Copy your API key from the provider's website, then tap <strong>Paste key</strong>. You pay the provider for what you use; Moly never sees your key on any server. Create a separate key for Moly and set a spending limit there.</li>
              <li>Tick the box that says your messages are sent to the provider, and press Save. Moly chooses the models itself.</li>
            </ol>
            <p className="doc-note"><Icon name="info" /><span>Moly keeps working while you leave the app. A reply can take a minute; when you come back the reply is there, or Moly says that it could not finish.</span></p>
          </Section>

          <Section title="Share a message to Moly">
            <p>
              To answer a message you received, press and hold its text in the other app (WhatsApp, e-mail, messages), choose Share, then
              Moly. The text appears in the writing box. You can also copy it and tap <strong>Paste</strong> under the box. Moly reads the
              clipboard only when you tap Paste, and never looks at any other app.
            </p>
          </Section>
        </>
      ) : (
      <Section title="Getting started">
        <ol>
          <li>Install the Moly app on your computer (the extension shows the steps and a Download button if it is missing). It starts and stops by itself with your browser.</li>
          <li>Open the side panel with the Moly icon in the toolbar (in Firefox: the sidebar, or View &gt; Sidebar &gt; Moly).</li>
          <li>
            On the first start choose where the thinking happens:
            <ul>
              <li><strong>Ollama</strong>: a model on your own computer. Nothing leaves it. Ollama must be installed and a model downloaded; Moly lists the models it finds.</li>
              <li><strong>Claude</strong> or <strong>OpenAI</strong>: your own account. Paste your API key. You pay the provider for what you use; Moly never sees your key on any server.</li>
            </ul>
          </li>
          <li>For a cloud key, keep "Protect the key with a passphrase" ticked. You will type the passphrase once each time the app starts.</li>
        </ol>
        <p className="doc-note"><Icon name="info" /><span>A model on a computer without a graphics card can take minutes to answer. Moly shows your message at once and a typing sign while it works.</span></p>
      </Section>
      )}

      <Section title="Writing a message">
        <ol>
          <li>Type in the box at the bottom and press Enter (Shift+Enter starts a new line), or press the arrow button.</li>
          <li>Moly replies with one question, sometimes with a short sentence of understanding first. A greeting gets a greeting.</li>
          <li>Answer in your own words. To write to someone, Moly needs a name for the person.</li>
          <li>When Moly offers a draft, comment on it ("shorter", "warmer", "mention that it was cold last week") and Moly revises it first, before asking anything else.</li>
        </ol>
        <p>Moly remembers the conversation and what you told it about the people involved, so you do not have to repeat yourself.</p>
      </Section>

      <Section title="Skip questions">
        <p>
          Under the writing box, <strong>Skip questions</strong> tells Moly to go ahead with what it already knows. It becomes available
          after you have answered at least one question, while Moly is asking a question you may skip. The skipped questions stay open
          and Moly may come back to them later if they are still worth asking.
        </p>
      </Section>

      <Section title="Previous conversations">
        <p>
          The clock icon in the top bar lists your earlier conversations. Choose one to continue it. The bin icon on a conversation
          deletes it, after a confirmation, so it is not deleted by accident.
        </p>
      </Section>

      <Section title="Pending insights">
        <p>
          The light bulb icon shows what Moly has guessed about you (your characteristics, interests and aims) and waits for your
          answer. Press the tick if it is right and the cross if it is not.
        </p>
      </Section>

      {!phone && (
        <>
        <Section title="Removing Moly">
          <p>
            Removing the extension in your browser does not remove the Moly app on your computer, because a browser extension cannot delete programs. Settings, "Remove
            Moly" lists the steps for your system, and your browser opens the same steps on the Moly website when you remove the extension. Your conversations stay on
            your computer unless you delete the .moly folder too; export them first in Settings if you may want them back.
          </p>
        </Section>

        <Section title="Profiles">
          <p>
            On a computer Moly always asks who is using it. The first time you make your profile (a name and a password); after that you choose your
            name and type the password each time the browser has been closed. If someone else uses Moly on this computer, Settings, Profiles, "Make
            another profile" makes one for them. Each profile has its own conversations, people, settings and AI key. Conversations that an older
            version kept on this computer become those of the first profile you make. The password keeps other people of the computer out of your
            conversations in Moly; people who use the same computer account can still open its files, so for real separation use a separate
            computer account for each person. The phone has one owner and no password of its own.
          </p>
          <p>
            When you make a profile Moly shows a recovery code once. Keep it: if you forget the password, the code is the only way back in
            ("I forgot my password" on the start screen), because Moly has no server and cannot send you anything. Use "Sign out"
            in Settings to change who is using Moly.
          </p>
        </Section>
        </>
      )}

      <Section title="About you and your preferences">
        <p>
          The first time you may see "Tell me about yourself": your style, values and preferred tone. All of it is optional; "Skip for
          now" hides it for a week. You can fill it in or change it at any time in Settings, under Communication preferences.
        </p>
      </Section>

      {phone ? (
        <Section title="Your data and your privacy">
          <ul>
            <li>Everything is stored on this phone, in an encrypted database that only Moly can open. Moly has no server and no analytics.</li>
            <li>The key to the database is kept by Android's secure key storage.</li>
            <li>Your messages are sent to the provider whose key you gave (Claude or OpenAI), and only to that provider, to write the replies.</li>
            <li>Passwords, card numbers and similar secrets in your messages are cut out before anything is stored.</li>
            <li><strong>Save a copy</strong> (Settings, Your data) saves everything Moly stored about you in one file, in a place you choose. <strong>Restore from a copy</strong> puts such a file back into an empty profile, for example after a reinstall: enter your key first, then restore. Without a copy, uninstalling Moly deletes your conversations.</li>
            <li><strong>Delete profile permanently</strong> (Settings) removes your profile and everything stored with it. It cannot be undone.</li>
          </ul>
        </Section>
      ) : (
      <Section title="Your data and your privacy">
        <ul>
          <li>Everything is stored on your computer, in an encrypted database in the <code>.moly</code> folder of your user folder. Moly has no server and no analytics.</li>
          <li>Text you write is sent to an AI provider only if you chose Claude or OpenAI, and only to that provider. With Ollama nothing leaves your computer.</li>
          <li>Passwords, card numbers and similar secrets in your messages are cut out before anything is stored.</li>
          <li><strong>Export my data</strong> (Settings, Your data) saves everything Moly stored about you in one file. <strong>Import data</strong> restores such a file into an empty profile, for example after reinstalling.</li>
          <li><strong>Delete profile permanently</strong> removes your profile and everything stored with it. It cannot be undone, so export first if you may want it.</li>
          <li>The key to the database is kept in your system keychain. On a computer without one, Moly asks you to protect it with a passphrase (recommended) or, knowingly, a key file. A forgotten passphrase cannot be recovered.</li>
        </ul>
      </Section>
      )}

      <Section title="Appearance and the Moly message">
        <p>
          Settings, Appearance: System (the default), Light or Dark. The small card at the bottom of the panel is a message from Moly or
          from a sponsor, and is always labelled "Sponsored". Its text is a small data file that Moly fetches from the project website at most once a day, with nothing about you; the built-in text shows when it is unavailable. It never appears between messages. Its Hide button hides it
          until you open Moly again.
        </p>
      </Section>

      {phone ? (
        <Section title="Something is wrong">
          <dl className="manual-faq">
            <dt>"Moly could not start" or the screen stays empty</dt>
            <dd>Press Try again. If it persists, close Moly completely and open it again.</dd>
            <dt>"The key could not be checked"</dt>
            <dd>Check the key (copy it again from the provider's website) and your connection, then press Try again. A key from Claude starts with sk-ant, one from OpenAI with sk-.</dd>
            <dt>"Moly is still writing"</dt>
            <dd>The reply is being written, even if you left the app. Wait, or come back later. After 10 minutes without a reply Moly says so, and you can send the message again.</dd>
            <dt>"Moly could not finish the reply"</dt>
            <dd>The AI provider did not answer (a wrong or empty account, a spending limit, no connection). Fix that and send the message again.</dd>
            <dt>Answers take very long</dt>
            <dd>That depends on the provider and your connection. Moly keeps working while you do something else.</dd>
            <dt>"Moly had stopped and is starting again"</dt>
            <dd>The phone had closed Moly's engine. It starts again by itself; your message is back in the box, so send it again.</dd>
            <dt>Moly stops while the phone is idle</dt>
            <dd>Some phones close apps to save battery. In the phone's battery settings, allow Moly to run in the background.</dd>
            <dt>Something else, or it keeps happening</dt>
            <dd>Settings, Report a problem. It shows a short note about your phone and Moly's engine (it is built to leave your messages out). You read it first, and it goes nowhere unless you share it.</dd>
          </dl>
        </Section>
      ) : (
      <Section title="Something is wrong">
        <dl className="manual-faq">
          <dt>"Install Moly" appears</dt>
          <dd>The app is missing. Install it, restart the browser and press Check again.</dd>
          <dt>"Moly does not recognise this extension"</dt>
          <dd>The app was installed for another copy of the extension. Install the latest app from the Download button.</dd>
          <dt>"Moly could not start"</dt>
          <dd>Press Check again. If it persists, open the log file at {LOG_HINT}; it contains no message text. Writing to us with that file helps.</dd>
          <dt>Moly asks for a passphrase</dt>
          <dd>Your data or your AI key is protected by one. Enter it. Wrong guesses are slowed down on purpose.</dd>
          <dt>"Your keychain is not reachable"</dt>
          <dd>Log in to your desktop session or unlock the keychain, then press Check again. Do not make a new key: that would lock your data away.</dd>
          <dt>No Ollama models listed</dt>
          <dd>Start Ollama and download a model (for example <code>ollama pull mistral</code>), then reopen the page.</dd>
          <dt>Answers take very long</dt>
          <dd>A local model on a slow computer needs time. A smaller model, or Claude or OpenAI with your own key, is faster.</dd>
          <dt>Sign-in appears</dt>
          <dd>That screen is only for a development backend started by hand; the installed Moly needs no account.</dd>
        </dl>
      </Section>
      )}

      <Section title="What Moly is not">
        <p>
          Moly is a writing coach. It is not a crisis, medical or legal service, and it gives no emergency advice. If a request is
          clearly harmful or illegal, Moly declines it and suggests asking a specialist.
        </p>
      </Section>

      <Section title="More help">
        <p>
          Write to <a href={`mailto:${SUPPORT_EMAIL}`}>{SUPPORT_EMAIL}</a>, or open an issue at{' '}
          <a href={ISSUES_URL} target="_blank" rel="noopener noreferrer">github.com/Nireus79/Moly/issues</a>. Please do not send your conversations or any key.
        </p>
      </Section>
    </div>
  </div>
  );
};

export default ManualPanel;
