import React, { useState } from 'react';

/**
 * Shows the recovery code of a profile once. The code is the only way back in after a forgotten password (Moly has no server and no email),
 * so the person confirms they have kept it before they can go on.
 */
/** `bare`: no card of its own, because a frame around it already has one. */
export const RecoveryCode: React.FC<{ code: string; onDone: () => void; again?: boolean; bare?: boolean }> = ({ code, onDone, again, bare }) => {
  const [kept, setKept] = useState(false);
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
    } catch {
      setCopied(false); // the code is on the screen; copying is a convenience
    }
  };
  return (
    <div className={bare ? 'recovery-bare' : 'ai-card'} role="alert">
      <h2>{again ? 'Your new recovery code' : 'Your recovery code'}</h2>
      <p>
        If you forget your password, this code is the only way back in. Moly has no server and cannot send you anything. Write it down or save it somewhere safe. It is shown
        only now{again ? ', and the old code no longer works' : ''}.
      </p>
      <p className="recovery-box"><strong className="recovery-code">{code}</strong></p>
      <div className="ai-actions">
        <button className="ai-btn" onClick={copy}>{copied ? 'Copied' : 'Copy'}</button>
      </div>
      <label className="form-label" style={{ display: 'flex', gap: 8, alignItems: 'center', marginTop: 12 }}>
        <input type="checkbox" checked={kept} onChange={(e) => setKept(e.target.checked)} /> I have saved the code
      </label>
      <div className="ai-actions">
        <button className="ai-btn primary" disabled={!kept} onClick={onDone}>Continue</button>
      </div>
    </div>
  );
};

export default RecoveryCode;
