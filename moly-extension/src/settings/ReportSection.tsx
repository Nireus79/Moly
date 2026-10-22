import React, { useState } from 'react';
import { phoneBridge } from '@/utils/platform';

/**
 * Phone app only: "Report a problem". The report is built on the phone, shown here in full, and leaves the phone only when the
 * person taps Send and their mail app opens with it. The person can add a note of their own, for example a reply they want to
 * report. Without a bridge that can build one (browser extension), nothing is shown.
 */
export const ReportSection: React.FC = () => {
  const bridge = phoneBridge();
  const [report, setReport] = useState<string | null>(null);
  const [note, setNote] = useState('');
  const [message, setMessage] = useState('');
  if (!bridge || !bridge.buildReport || !bridge.shareReport) return null;
  const { buildReport, shareReport } = bridge;

  const prepare = async () => {
    setMessage('');
    try {
      setNote('');
      setReport(await buildReport());
    } catch {
      setReport(null);
      setMessage('Could not prepare the report.');
    }
  };
  const share = async () => {
    try {
      const text = note.trim() ? `${note.trim()}\n\n---\n${report ?? ''}` : report ?? '';
      await shareReport(text);
    } catch {
      setMessage('Could not open a mail app.');
    }
  };

  return (
    <section className="settings-section">
      <h2>Report a problem or a reply</h2>
      <p className="section-description">
        Prepares a short note about how Moly is doing: your phone model and Android version, when Moly started and stopped, and error
        lines from its engine. You can add your own note, for example a reply from Moly that you want to report: paste it there. Read
        everything first. It goes nowhere unless you tap Send and your mail app opens with it; you press Send there.
      </p>
      {report === null ? (
        <button className="btn btn-secondary" onClick={() => void prepare()}>Prepare report</button>
      ) : (
        <>
          <textarea
            className="report-text"
            rows={4}
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="Your note: describe the problem, or paste the reply you want to report (optional)"
            aria-label="Your note"
          />
          <textarea className="report-text" readOnly rows={10} value={report} aria-label="The report" />
          <div className="button-row phone-buttons">
            <button className="btn btn-primary" onClick={() => void share()}>Send report</button>
            <button className="btn btn-secondary" onClick={() => setReport(null)}>Close</button>
          </div>
        </>
      )}
      {message && <div className="notice danger"><span>{message}</span></div>}
    </section>
  );
};

export default ReportSection;
