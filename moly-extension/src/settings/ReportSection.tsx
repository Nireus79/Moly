import React, { useState } from 'react';
import { phoneBridge } from '@/utils/platform';

/**
 * Phone app only: "Report a problem". The report is built on the phone, shown here in full, and leaves the phone only when the
 * person taps Share and picks where it goes. Without a bridge that can build one (browser extension), nothing is shown.
 */
export const ReportSection: React.FC = () => {
  const bridge = phoneBridge();
  const [report, setReport] = useState<string | null>(null);
  const [message, setMessage] = useState('');
  if (!bridge || !bridge.buildReport || !bridge.shareReport) return null;
  const { buildReport, shareReport } = bridge;

  const prepare = async () => {
    setMessage('');
    try {
      setReport(await buildReport());
    } catch {
      setReport(null);
      setMessage('Could not prepare the report.');
    }
  };
  const share = async () => {
    try {
      await shareReport(report ?? '');
    } catch {
      setMessage('Could not open the share sheet.');
    }
  };

  return (
    <section className="settings-section">
      <h2>Report a problem</h2>
      <p className="section-description">
        Prepares a short note about how Moly is doing: your phone model and Android version, when Moly started and stopped, and error
        lines from its engine. Read it first. It goes nowhere unless you tap Share and choose where to send it.
      </p>
      {report === null ? (
        <button className="btn btn-secondary" onClick={() => void prepare()}>Prepare report</button>
      ) : (
        <>
          <textarea className="report-text" readOnly rows={10} value={report} aria-label="The report" />
          <div className="button-row phone-buttons">
            <button className="btn btn-primary" onClick={() => void share()}>Share report</button>
            <button className="btn btn-secondary" onClick={() => setReport(null)}>Close</button>
          </div>
        </>
      )}
      {message && <div className="notice danger"><span>{message}</span></div>}
    </section>
  );
};

export default ReportSection;
