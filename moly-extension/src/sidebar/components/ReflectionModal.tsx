import React, { useState } from 'react';
import type { ExtractedContactContext } from '@/utils/contextExtractor';

interface ReflectionModalProps {
  isOpen: boolean;
  conversationName: string;
  onClose: () => void;
  onSave: (notes: string) => void;
  onSaveContactContext?: (context: ExtractedContactContext) => void;
  extractedContext?: ExtractedContactContext;
  isExtractingContext?: boolean;
}

export const ReflectionModal: React.FC<ReflectionModalProps> = ({
  isOpen,
  conversationName,
  onClose,
  onSave,
  onSaveContactContext,
  extractedContext,
  isExtractingContext = false,
}) => {
  const [notes, setNotes] = useState('');
  const [saved, setSaved] = useState(false);
  const [contactContextApproved, setContactContextApproved] = useState(false);
  const [step, setStep] = useState<'context' | 'reflection'>('context');
  const [editedContext, setEditedContext] = useState<ExtractedContactContext>(extractedContext || {});
  const [editingItem, setEditingItem] = useState<{ type: 'characteristics' | 'intentions' | 'behaviors'; index: number } | null>(null);
  const [editingValue, setEditingValue] = useState('');

  if (!isOpen) return null;

  const hasExtractedContext =
    extractedContext &&
    (extractedContext.characteristics?.length ||
      extractedContext.intentions?.length ||
      extractedContext.behaviors?.length);

  // Sync edited context when extracted context changes
  React.useEffect(() => {
    if (extractedContext) {
      setEditedContext(extractedContext);
    }
  }, [extractedContext, isOpen]);

  const handleRemoveItem = (type: 'characteristics' | 'intentions' | 'behaviors', index: number) => {
    setEditedContext(prev => ({
      ...prev,
      [type]: prev[type]?.filter((_, i) => i !== index) || [],
    }));
  };

  const handleEditItem = (type: 'characteristics' | 'intentions' | 'behaviors', index: number, value: string) => {
    setEditingItem({ type, index });
    setEditingValue(value);
  };

  const handleSaveEdit = () => {
    if (editingItem) {
      setEditedContext(prev => {
        const array = [...(prev[editingItem.type] || [])];
        array[editingItem.index] = editingValue;
        return {
          ...prev,
          [editingItem.type]: array,
        };
      });
      setEditingItem(null);
      setEditingValue('');
    }
  };

  const handleApproveContext = async () => {
    if (onSaveContactContext) {
      onSaveContactContext(editedContext);
      setContactContextApproved(true);
      setTimeout(() => {
        setStep('reflection');
      }, 600);
    }
  };

  const handleSkipContext = () => {
    setStep('reflection');
  };

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
    setStep('context');
    setContactContextApproved(false);
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
        zIndex: 10002,
        pointerEvents: 'auto',
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
          maxHeight: '90vh',
          overflow: 'auto',
          boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.1)',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        {step === 'context' && hasExtractedContext ? (
          <>
            <h2 style={{ marginTop: 0, marginBottom: '16px', fontSize: '18px' }}>
              Save What I Learned About {conversationName}
            </h2>

            {isExtractingContext ? (
              <div style={{ textAlign: 'center', padding: '24px' }}>
                <div style={{ color: '#6366f1', marginBottom: '8px' }}>Analyzing conversation...</div>
              </div>
            ) : (
              <>
                <div
                  style={{
                    background: '#f0f9ff',
                    border: '1px solid #bfdbfe',
                    borderRadius: '4px',
                    padding: '12px',
                    marginBottom: '16px',
                    fontSize: '13px',
                    color: '#0369a1',
                  }}
                >
                  I found some useful information about {conversationName} in our conversation. Should I save this to their profile?
                </div>

                <div
                  style={{
                    background: '#f3f4f6',
                    borderRadius: '4px',
                    padding: '12px',
                    marginBottom: '16px',
                    fontSize: '13px',
                    lineHeight: '1.6',
                    color: '#666',
                    maxHeight: '300px',
                    overflowY: 'auto',
                  }}
                >
                  {editingItem && (
                    <div style={{
                      background: 'white',
                      border: '1px solid #bfdbfe',
                      borderRadius: '4px',
                      padding: '12px',
                      marginBottom: '12px',
                    }}>
                      <label style={{ display: 'block', fontSize: '12px', fontWeight: '600', marginBottom: '6px' }}>
                        Edit {editingItem.type}:
                      </label>
                      <input
                        type="text"
                        value={editingValue}
                        onChange={(e) => setEditingValue(e.target.value)}
                        style={{
                          width: '100%',
                          padding: '8px',
                          border: '1px solid #ddd',
                          borderRadius: '4px',
                          fontSize: '12px',
                          boxSizing: 'border-box',
                          marginBottom: '8px',
                        }}
                      />
                      <div style={{ display: 'flex', gap: '6px' }}>
                        <button
                          onClick={handleSaveEdit}
                          style={{
                            flex: 1,
                            padding: '6px',
                            background: '#10b981',
                            color: 'white',
                            border: 'none',
                            borderRadius: '4px',
                            cursor: 'pointer',
                            fontSize: '12px',
                          }}
                        >
                          Save
                        </button>
                        <button
                          onClick={() => setEditingItem(null)}
                          style={{
                            flex: 1,
                            padding: '6px',
                            background: '#e5e7eb',
                            border: 'none',
                            borderRadius: '4px',
                            cursor: 'pointer',
                            fontSize: '12px',
                          }}
                        >
                          Cancel
                        </button>
                      </div>
                    </div>
                  )}

                  {editedContext.characteristics && editedContext.characteristics.length > 0 && (
                    <div style={{ marginBottom: '12px' }}>
                      <strong style={{ display: 'block', marginBottom: '6px', fontSize: '12px' }}>Characteristics:</strong>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                        {editedContext.characteristics.map((item, idx) => (
                          <div
                            key={idx}
                            style={{
                              background: 'white',
                              padding: '8px',
                              borderRadius: '4px',
                              display: 'flex',
                              justifyContent: 'space-between',
                              alignItems: 'center',
                              fontSize: '12px',
                              border: '1px solid #e5e7eb',
                            }}
                          >
                            <span>{item}</span>
                            <div style={{ display: 'flex', gap: '4px' }}>
                              <button
                                onClick={() => handleEditItem('characteristics', idx, item)}
                                style={{
                                  padding: '2px 6px',
                                  background: '#6366f1',
                                  color: 'white',
                                  border: 'none',
                                  borderRadius: '3px',
                                  cursor: 'pointer',
                                  fontSize: '11px',
                                }}
                              >
                                Edit
                              </button>
                              <button
                                onClick={() => handleRemoveItem('characteristics', idx)}
                                style={{
                                  padding: '2px 6px',
                                  background: '#ef4444',
                                  color: 'white',
                                  border: 'none',
                                  borderRadius: '3px',
                                  cursor: 'pointer',
                                  fontSize: '11px',
                                }}
                              >
                                Remove
                              </button>
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {editedContext.intentions && editedContext.intentions.length > 0 && (
                    <div style={{ marginBottom: '12px' }}>
                      <strong style={{ display: 'block', marginBottom: '6px', fontSize: '12px' }}>Intentions:</strong>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                        {editedContext.intentions.map((item, idx) => (
                          <div
                            key={idx}
                            style={{
                              background: 'white',
                              padding: '8px',
                              borderRadius: '4px',
                              display: 'flex',
                              justifyContent: 'space-between',
                              alignItems: 'center',
                              fontSize: '12px',
                              border: '1px solid #e5e7eb',
                            }}
                          >
                            <span>{item}</span>
                            <div style={{ display: 'flex', gap: '4px' }}>
                              <button
                                onClick={() => handleEditItem('intentions', idx, item)}
                                style={{
                                  padding: '2px 6px',
                                  background: '#6366f1',
                                  color: 'white',
                                  border: 'none',
                                  borderRadius: '3px',
                                  cursor: 'pointer',
                                  fontSize: '11px',
                                }}
                              >
                                Edit
                              </button>
                              <button
                                onClick={() => handleRemoveItem('intentions', idx)}
                                style={{
                                  padding: '2px 6px',
                                  background: '#ef4444',
                                  color: 'white',
                                  border: 'none',
                                  borderRadius: '3px',
                                  cursor: 'pointer',
                                  fontSize: '11px',
                                }}
                              >
                                Remove
                              </button>
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {editedContext.behaviors && editedContext.behaviors.length > 0 && (
                    <div>
                      <strong style={{ display: 'block', marginBottom: '6px', fontSize: '12px' }}>Behaviors:</strong>
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                        {editedContext.behaviors.map((item, idx) => (
                          <div
                            key={idx}
                            style={{
                              background: 'white',
                              padding: '8px',
                              borderRadius: '4px',
                              display: 'flex',
                              justifyContent: 'space-between',
                              alignItems: 'center',
                              fontSize: '12px',
                              border: '1px solid #e5e7eb',
                            }}
                          >
                            <span>{item}</span>
                            <div style={{ display: 'flex', gap: '4px' }}>
                              <button
                                onClick={() => handleEditItem('behaviors', idx, item)}
                                style={{
                                  padding: '2px 6px',
                                  background: '#6366f1',
                                  color: 'white',
                                  border: 'none',
                                  borderRadius: '3px',
                                  cursor: 'pointer',
                                  fontSize: '11px',
                                }}
                              >
                                Edit
                              </button>
                              <button
                                onClick={() => handleRemoveItem('behaviors', idx)}
                                style={{
                                  padding: '2px 6px',
                                  background: '#ef4444',
                                  color: 'white',
                                  border: 'none',
                                  borderRadius: '3px',
                                  cursor: 'pointer',
                                  fontSize: '11px',
                                }}
                              >
                                Remove
                              </button>
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>

                <div style={{ display: 'flex', gap: '8px', marginBottom: '16px' }}>
                  <button
                    onClick={handleApproveContext}
                    disabled={contactContextApproved}
                    style={{
                      flex: 1,
                      padding: '10px',
                      background: contactContextApproved ? '#10b981' : '#6366f1',
                      color: 'white',
                      border: 'none',
                      borderRadius: '4px',
                      cursor: contactContextApproved ? 'default' : 'pointer',
                      fontSize: '13px',
                      fontWeight: '600',
                    }}
                  >
                    {contactContextApproved ? '✓ Saved to Profile' : 'Save to Profile'}
                  </button>
                  <button
                    onClick={handleSkipContext}
                    style={{
                      flex: 1,
                      padding: '10px',
                      background: '#e5e7eb',
                      color: '#333',
                      border: 'none',
                      borderRadius: '4px',
                      cursor: 'pointer',
                      fontSize: '13px',
                      fontWeight: '600',
                    }}
                  >
                    Skip
                  </button>
                </div>
              </>
            )}
          </>
        ) : (
          <>
            <h2 style={{ marginTop: 0, marginBottom: '16px', fontSize: '18px' }}>
              Any Other Notes?
            </h2>

            <p style={{ fontSize: '13px', color: '#666', marginBottom: '16px' }}>
              Anything else you'd like Moly to remember about this conversation?
            </p>

            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              placeholder="e.g., 'They were stressed about deadline' or 'Prefers calls over texts'"
              style={{
                width: '100%',
                padding: '12px',
                border: '1px solid #ddd',
                borderRadius: '4px',
                fontSize: '13px',
                fontFamily: 'inherit',
                boxSizing: 'border-box',
                minHeight: '80px',
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
                Done
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
};
