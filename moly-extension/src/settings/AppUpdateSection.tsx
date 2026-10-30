import React, { useEffect, useState } from 'react';
import { getBackendManager } from '@/api/backendManager';
import { compareVersions, latestVersion } from '@/api/installerDownload';
import { InstallRunner } from '@/components/InstallRunner';

type Check = { state: 'checking' } | { state: 'unknown'; app?: string } | { state: 'current'; app: string } | { state: 'newer'; app: string; latest: string };

/**
 * The Moly app on this computer: its version, and, when a newer release exists, the update button (it downloads the installer and opens it).
 * The newest version is looked up when this section is shown (one request to api.github.com; nothing about the person is sent).
 */
export const AppUpdateSection: React.FC = () => {
  const [check, setCheck] = useState<Check>({ state: 'checking' });

  useEffect(() => {
    let alive = true;
    (async () => {
      const status = await getBackendManager().getStatus().catch(() => null);
      const app = status?.version;
      const latest = await latestVersion();
      if (!alive) return;
      if (!app || !latest) return setCheck({ state: 'unknown', app });
      const c = compareVersions(latest, app);
      setCheck(c === null ? { state: 'unknown', app } : c > 0 ? { state: 'newer', app, latest } : { state: 'current', app });
    })();
    return () => {
      alive = false;
    };
  }, []);

  return (
    <section className="settings-section">
      <h2>Moly app</h2>
      {check.state === 'checking' && <p className="section-description">Looking for a newer version...</p>}
      {check.state === 'current' && <p className="section-description">Version {check.app}. This is the newest version.</p>}
      {check.state === 'unknown' && (
        <p className="section-description">
          {check.app ? `Version ${check.app}. ` : ''}Moly could not tell whether a newer version exists (a development build, or no connection to GitHub).
        </p>
      )}
      {check.state === 'newer' && (
        <>
          <p className="section-description">
            Version {check.app} is installed; version {check.latest} is available. The button downloads the installer and opens it. Your conversations stay.
          </p>
          <div className="button-row">
            <InstallRunner label={`Update to ${check.latest}`} />
          </div>
        </>
      )}
    </section>
  );
};

export default AppUpdateSection;
