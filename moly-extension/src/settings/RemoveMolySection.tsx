import React from 'react';
import { detectOS, uninstallInfo } from '@/api/hostErrors';
import { UNINSTALL_URL } from '@/config/links';

/**
 * How to remove Moly completely. Removing the extension in the browser does not remove the Moly app: a browser extension may not delete programs
 * from the computer, so this says how, for this system. (After the extension is removed the browser opens the same steps on the project's page.)
 */
export const RemoveMolySection: React.FC = () => {
  const info = uninstallInfo(detectOS());
  return (
    <section className="settings-section">
      <h2>Remove Moly</h2>
      <p className="section-description">
        Removing the extension in your browser does not remove the Moly app on your computer. To remove everything, remove the extension and then the app.
      </p>
      <h3>{info.heading}</h3>
      <ol className="section-description">
        {info.steps.map((s) => (
          <li key={s}>{s}</li>
        ))}
      </ol>
      <p className="section-info">
        The same steps are on <a href={UNINSTALL_URL} target="_blank" rel="noreferrer">the Moly website</a>, which your browser also opens when you remove the extension.
      </p>
    </section>
  );
};

export default RemoveMolySection;
