import React, { useState, useEffect } from 'react';
import { useAuth } from '@/hooks/useAuth';
import { getBackendManager } from '@/api/backendManager';
import './reflections-panel.css';

interface PendingReflection {
  id: string;
  characteristics?: string[];
  interests?: string[];
  intentions?: string[];
  status: string;
  createdAt: number;
}

export const ReflectionsPanel: React.FC = () => {
  const { session } = useAuth();
  const [reflections, setReflections] = useState<PendingReflection[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [processingId, setProcessingId] = useState<string | null>(null);

  useEffect(() => {
    const fetchReflections = async () => {
      if (!session) return;

      try {
        setLoading(true);
        const apiBase = getBackendManager().getBackendUrl();
        const response = await fetch(`${apiBase}/api/v2/reflections`, {
          method: 'GET',
          headers: {
            'Authorization': `Bearer ${session.sessionId}`,
          },
        });

        // 404 is expected when no reflections exist — treat as empty list
        if (response.status === 404) {
          console.log('[ReflectionsPanel] No pending reflections (404 is normal when empty)');
          setReflections([]);
          setLoading(false);
          return;
        }

        if (!response.ok) {
          if (response.status === 401) {
            setError('Session expired');
            setLoading(false);
            return;
          }
          throw new Error(`Failed to fetch reflections: ${response.statusText}`);
        }

        const data = await response.json();
        const reflectionList = data.reflections || [];
        setReflections(reflectionList);
        console.log('[ReflectionsPanel] Loaded', reflectionList.length, 'pending reflections');
      } catch (err) {
        let message = 'Failed to load reflections';
        if (err instanceof Error) {
          message = err.message;
        } else if (typeof err === 'string') {
          message = err;
        }
        // Ensure we never set an object as error state (would cause React error)
        setError(String(message));
        console.error('[ReflectionsPanel] Fetch error:', message);
      } finally {
        setLoading(false);
      }
    };

    fetchReflections();
  }, [session]);

  const handleApprove = async (reflectionId: string) => {
    if (!session) return;

    try {
      setProcessingId(reflectionId);
      const apiBase = getBackendManager().getBackendUrl();
      const response = await fetch(`${apiBase}/api/v2/reflections`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${session.sessionId}`,
        },
        body: JSON.stringify({
          id: parseInt(reflectionId, 10),
          action: 'approve',
        }),
      });

      if (!response.ok) {
        throw new Error(`Failed to approve reflection: ${response.statusText}`);
      }

      // Remove from list
      setReflections(prev => prev.filter(r => r.id !== reflectionId));
      console.log('[ReflectionsPanel] Approved reflection', reflectionId);
    } catch (err) {
      let message = 'Failed to approve';
      if (err instanceof Error) {
        message = err.message;
      } else if (typeof err === 'string') {
        message = err;
      }
      setError(String(message));
      console.error('[ReflectionsPanel] Approve error:', message);
    } finally {
      setProcessingId(null);
    }
  };

  const handleReject = async (reflectionId: string) => {
    if (!session) return;

    try {
      setProcessingId(reflectionId);
      const apiBase = getBackendManager().getBackendUrl();
      const response = await fetch(`${apiBase}/api/v2/reflections`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${session.sessionId}`,
        },
        body: JSON.stringify({
          id: parseInt(reflectionId, 10),
          action: 'reject',
        }),
      });

      if (!response.ok) {
        throw new Error(`Failed to reject reflection: ${response.statusText}`);
      }

      // Remove from list
      setReflections(prev => prev.filter(r => r.id !== reflectionId));
      console.log('[ReflectionsPanel] Rejected reflection', reflectionId);
    } catch (err) {
      let message = 'Failed to reject';
      if (err instanceof Error) {
        message = err.message;
      } else if (typeof err === 'string') {
        message = err;
      }
      setError(String(message));
      console.error('[ReflectionsPanel] Reject error:', message);
    } finally {
      setProcessingId(null);
    }
  };

  if (loading) {
    return (
      <div className="reflections-panel">
        <div className="reflections-loading">
          <div className="spinner">⏳</div>
          <p>Loading your insights...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="reflections-panel">
      <div className="reflections-header">
        <h3>Pending Insights</h3>
        <p className="reflections-subtitle">Review and approve what I've learned about you</p>
      </div>

      {error && (
        <div className="reflections-error">
          <span>⚠️</span>
          <p>{error}</p>
        </div>
      )}

      {reflections.length === 0 ? (
        <div className="reflections-empty">
          <div className="empty-icon">✨</div>
          <p>No pending insights yet</p>
          <p className="empty-subtitle">Share more with me and I'll learn</p>
        </div>
      ) : (
        <div className="reflections-list">
          {reflections.map(reflection => (
            <div key={reflection.id} className="reflection-card">
              <div className="reflection-content">
                {reflection.characteristics && reflection.characteristics.length > 0 && (
                  <div className="reflection-section">
                    <label>I think you are:</label>
                    <ul className="reflection-list">
                      {reflection.characteristics.map((char, idx) => (
                        <li key={idx}>{char}</li>
                      ))}
                    </ul>
                  </div>
                )}

                {reflection.interests && reflection.interests.length > 0 && (
                  <div className="reflection-section">
                    <label>Your interests:</label>
                    <ul className="reflection-list">
                      {reflection.interests.map((interest, idx) => (
                        <li key={idx}>{interest}</li>
                      ))}
                    </ul>
                  </div>
                )}

                {reflection.intentions && reflection.intentions.length > 0 && (
                  <div className="reflection-section">
                    <label>What you want:</label>
                    <ul className="reflection-list">
                      {reflection.intentions.map((intention, idx) => (
                        <li key={idx}>{intention}</li>
                      ))}
                    </ul>
                  </div>
                )}

                <div className="reflection-date">
                  {new Date(reflection.createdAt * 1000).toLocaleDateString([], {
                    month: 'short',
                    day: 'numeric',
                    hour: '2-digit',
                    minute: '2-digit',
                  })}
                </div>
              </div>

              <div className="reflection-actions">
                <button
                  className="reflection-btn reflection-btn-approve"
                  onClick={() => handleApprove(reflection.id)}
                  disabled={processingId !== null}
                  title="This is accurate"
                >
                  {processingId === reflection.id ? '⏳' : '✓'}
                </button>
                <button
                  className="reflection-btn reflection-btn-reject"
                  onClick={() => handleReject(reflection.id)}
                  disabled={processingId !== null}
                  title="This is not accurate"
                >
                  {processingId === reflection.id ? '⏳' : '✕'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};
