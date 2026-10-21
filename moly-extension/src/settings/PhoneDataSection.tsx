import React, { useState } from 'react';
import { fetchExport, importExport } from '@/api/dataTransfer';
import { phoneBridge } from '@/utils/platform';
import { Icon } from '@/components/Icon';

/**
 * Phone app: "Your data". The desktop saves a downloaded file and picks one from a file input; on the phone both go through
 * Android's own pickers, so the person chooses where the copy goes and which one to restore. Without a bridge that can do
 * that (browser extension), nothing is shown.
 */
export const PhoneDataSection: React.FC = () => {
  const bridge = phoneBridge();
  const [working, setWorking] = useState(false);
  const [message, setMessage] = useState('');
  if (!bridge || !bridge.saveFile || !bridge.pickFile) return null;
  const { saveFile, pickFile } = bridge;

  const save = async () => {
    setWorking(true);
    setMessage('');
    try {
      const result = await fetchExport();
      if (!result.ok) {
        setMessage(`Could not save a copy: ${result.message}`);
      } else {
        const saved = await saveFile(`moly-copy-${new Date().toISOString().slice(0, 10)}.json`, result.text);
        setMessage(saved ? 'The copy was saved.' : 'Nothing was saved.');
      }
    } catch {
      setMessage('Could not save a copy: the phone could not write the file.');
    }
    setWorking(false);
  };

  const restore = async () => {
    setWorking(true);
    setMessage('');
    try {
      const text = await pickFile();
      if (text === null) setMessage('Nothing was restored.');
      else {
        const result = await importExport(text);
        // The desktop message tells the person to reopen the panel; on the phone the conversations are listed at once.
        setMessage(result.ok ? result.message.replace('Close and reopen the Moly panel to see them.', 'Your conversations are under Previous conversations.') : `Could not restore: ${result.message}`);
      }
    } catch {
      setMessage('Could not restore: the phone could not read the file.');
    }
    setWorking(false);
  };

  const failed = message.startsWith('Could not');
  return (
    <section className="settings-section">
      <h2>Your data</h2>
      <p className="section-description">
        Save everything Moly stored about you (conversations, contacts, what Moly learned) as one file, in a place you choose on the
        phone. Moly sends it nowhere. Your AI key is not in the file: after a reinstall, enter the key first, then restore.
      </p>
      <div className="button-row phone-buttons">
        <button className="btn btn-secondary" onClick={() => void save()} disabled={working}>
          <Icon name="download" /> {working ? 'Working...' : 'Save a copy'}
        </button>
        <button className="btn btn-secondary" onClick={() => void restore()} disabled={working}>
          <Icon name="upload" /> Restore from a copy
        </button>
      </div>
      <p className="section-info">Restoring works on an empty profile, for example right after a reinstall. Without a copy, uninstalling Moly deletes your conversations.</p>
      {message && (
        <div className={`notice ${failed ? 'danger' : 'ok'}`}>
          <Icon name={failed ? 'alert' : 'check'} /><span>{message}</span>
        </div>
      )}
    </section>
  );
};

export default PhoneDataSection;
