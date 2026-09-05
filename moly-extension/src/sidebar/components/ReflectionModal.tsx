import React, { useState } from 'react';

interface ReflectionModalProps {
  isOpen: boolean;
  conversationName: string;
  onClose: () => void;
  onSave: (notes: string) => void;
}

export const ReflectionModal: React.FC<ReflectionModalProps> = ({
  isOpen,
  conversationName,
  onClose,
  onSave,
}) => {
  const [notes, setNotes] = useState('');
  const [saved, setSaved] = useState(false);

  if (!isOpen) return null;

  const handleSave = () => {
    onSave(notes);
    setSaved(true);
    setTimeout(() => {
      setNotes('');
      setSaved(false);
      onClose();
    }, 800);
  };

  const handleClose = () => {
    setNotes('');
    setSaved(false);
    onClose();
  };

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0, 0, 0, 0.5)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        zIndex: 1001,
      }}
      onClick={handleClose}
    >
      <div
        style={{
          background: 'white',
          borderRadius: '8px',
          padding: '24px',
          maxWidth: '500px',
          width: '90%',
          boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.1)',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <h2 style={{ marginTop: 0, marginBottom: '16px', fontSize: '18px' }}>
          Save What You Learned
        </h2>

        <p style={{ fontSize: '13px', color: '#666', marginBottom: '16px' }}>
          Anything you'd like Moly to remember about this conversation with {conversationName}?
        </p>

        <textarea
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
          placeholder="e.g., 'They mentioned job change' or 'Prefers direct advice' or 'Talk about hiking next time'"
          style={{
            width: '100%',
            padding: '12px',
            border: '1px solid #ddd',
            borderRadius: '4px',
            fontSize: '13px',
            fontFamily: 'inherit',
            boxSizing: 'border-box',
            minHeight: '100px',
            marginBottom: '16px',
            resize: 'vertical',
          }}
        />

        <div style={{ display: 'flex', gap: '8px' }}>
          <button
            onClick={handleSave}
            disabled={saved}
            style={{
              flex: 1,
              padding: '10px',
              background: saved ? '#10b981' : '#6366f1',
              color: 'white',
              border: 'none',
              borderRadius: '4px',
              cursor: saved ? 'default' : 'pointer',
              fontSize: '14px',
              fontWeight: '600',
            }}
          >
            {saved ? '✓ Saved' : 'Save Notes'}
          </button>
          <button
            onClick={handleClose}
            style={{
              flex: 1,
              padding: '10px',
              background: '#e5e7eb',
              color: '#333',
              border: 'none',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '14px',
              fontWeight: '600',
            }}
          >
            Skip
          </button>
        </div>
      </div>
    </div>
  );
};
