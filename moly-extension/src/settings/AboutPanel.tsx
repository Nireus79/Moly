import React from 'react';
import { Icon } from '@/components/Icon';
import { isPhoneApp } from '@/utils/platform';
import {
  HERMES_URL,
  ISSUES_URL,
  PRIVACY_URL,
  REPO_URL,
  SITE_URL,
  SOCRATES_URL,
  SPONSOR_URL,
  SUPPORT_EMAIL,
} from '@/config/links';

const Ext: React.FC<{ href: string; children: React.ReactNode }> = ({ href, children }) => (
  <a href={href} target="_blank" rel="noopener noreferrer">{children}</a>
);

/** About Moly: what it is, who makes it, how to reach us, and under what terms. Opened from Settings. */
export const AboutPanel: React.FC = () => {
  const phone = isPhoneApp();
  let version = '';
  try {
    version = chrome.runtime.getManifest().version;
  } catch {
    /* not running as an extension */
  }

  return (
    <div className="doc-panel">
      <section className="settings-section doc-hero">
        <span className="brand-mark" aria-hidden="true" />
        <h2 lang="el">Μώλυ</h2>
        <p className="doc-tagline">Think before you send.</p>
        {version && <p className="version">Version {version}</p>}
      </section>

      <section className="settings-section">
        <h2>What Moly is</h2>
        <p>
          Moly is a writing coach for the messages that matter: to a landlord, a colleague, a partner, a customer. Before you send,
          it asks the one question that is missing, then helps you write. It never decides for you, and you can skip its questions
          at any time.
        </p>
        {phone ? (
          <p>
            Moly runs on this phone. It keeps your conversations in an encrypted database on this phone and talks to the AI model you
            choose: your own Claude or OpenAI key. There is no Moly server, no sign-up, no analytics and no tracking.
          </p>
        ) : (
          <p>
            It is made of this browser extension and a small app on your computer. The app keeps your conversations in an encrypted
            database on your computer and talks to the AI model you choose: a model on your computer (Ollama), or your own Claude or
            OpenAI key. There is no Moly server, no sign-up, no analytics and no tracking.
          </p>
        )}
        <p className="doc-note">
          <Icon name="info" />
          <span>Moly is a writing coach. It is not a crisis, medical or legal service, and it gives no emergency advice.</span>
        </p>
      </section>

      <section className="settings-section">
        <h2>The name</h2>
        <p>
          In Homer's <em>Odyssey</em>, <em>moly</em> (Μώλυ) is the herb that Hermes gave to Odysseus so that he could meet Circe with a
          clear head and not be changed by her drink. Moly the program tries to do the same for a message: keep your head clear
          before you send.
        </p>
      </section>

      <section className="settings-section">
        <h2>Who makes it</h2>
        <p>
          Moly is developed by <Ext href={HERMES_URL}>Hermes Software</Ext>, using solutions inherited from{' '}
          <Ext href={SOCRATES_URL}>Socrates AI</Ext>, its multi-agent platform.
        </p>
        <div className="about-logos" aria-hidden="true">
          <span className="mark mark-hermes" />
          <span className="mark mark-socrates" />
        </div>
      </section>

      <section className="settings-section">
        <h2>Contact</h2>
        <dl className="contact-list">
          <dt>Email</dt>
          <dd><a href={`mailto:${SUPPORT_EMAIL}`}>{SUPPORT_EMAIL}</a></dd>
          <dt>Questions and problems</dt>
          <dd><Ext href={ISSUES_URL}>github.com/Nireus79/Moly/issues</Ext></dd>
          <dt>Source code</dt>
          <dd><Ext href={REPO_URL}>github.com/Nireus79/Moly</Ext></dd>
          <dt>Website</dt>
          <dd><Ext href={SITE_URL}>nireus79.github.io/Moly</Ext></dd>
          <dt>Developer</dt>
          <dd><Ext href={HERMES_URL}>Hermes Software</Ext></dd>
          <dt>Privacy policy</dt>
          <dd><Ext href={PRIVACY_URL}>nireus79.github.io/Moly/privacy.html</Ext></dd>
        </dl>
        <p className="section-info">When you write for help, please do not include your conversations or any key.{!phone && ' The log file of the app (see the user manual) holds no message text.'}</p>
      </section>

      <section className="settings-section">
        <h2>Support the project</h2>
        <p>Moly is free. If it helps you, please sponsor its development, or tell a friend.</p>
        <div className="button-row">
          <a className="btn btn-primary" href={SPONSOR_URL} target="_blank" rel="noopener noreferrer">Sponsor Moly</a>
          <a className="btn btn-secondary" href={REPO_URL} target="_blank" rel="noopener noreferrer">Star it on GitHub</a>
        </div>
      </section>

      <section className="settings-section">
        <h2>License</h2>
        <p>
          Moly is free and open source under the MIT license: you may use, copy and change it, including for work, as long as the
          license notice stays with it. It comes without warranty.
        </p>
      </section>
    </div>
  );
};

export default AboutPanel;
