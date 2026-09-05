import React from 'react';
import type { SafetyCheckResult } from '@/api/molyAgent';

interface SafetyAlertProps {
  alert: SafetyCheckResult;
  onDismiss?: () => void;
}

export const SafetyAlert: React.FC<SafetyAlertProps> = ({ alert, onDismiss }) => {
  if (alert.alert_type === 'none') {
    return null;
  }

  const isImmediate = alert.severity === 'immediate';
  const isIllegal = alert.alert_type === 'illegal';

  return (
    <div
      style={{
        padding: '16px',
        marginBottom: '12px',
        borderRadius: '8px',
        border: `2px solid ${isImmediate ? '#ef4444' : '#f59e0b'}`,
        background: isImmediate ? '#fef2f2' : '#fffbeb',
        color: isImmediate ? '#7f1d1d' : '#92400e',
      }}
    >
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <div>
          <h3 style={{ margin: '0 0 8px 0', fontSize: '16px', fontWeight: '600' }}>
            {isImmediate ? '⚠️ ' : '⚡ '}
            {alert.title}
          </h3>
          <p style={{ margin: '0 0 12px 0', fontSize: '14px', lineHeight: '1.4' }}>
            {alert.message}
          </p>

          {alert.indicators && alert.indicators.length > 0 && (
            <div style={{ marginBottom: '12px' }}>
              <p style={{ margin: '0 0 6px 0', fontSize: '12px', fontWeight: '500' }}>
                Detected indicators:
              </p>
              <ul style={{ margin: '0', paddingLeft: '20px', fontSize: '13px' }}>
                {alert.indicators.map((indicator, i) => (
                  <li key={i}>{indicator}</li>
                ))}
              </ul>
            </div>
          )}

          {alert.resources && alert.resources.length > 0 && (
            <div style={{ marginBottom: '12px' }}>
              <p style={{ margin: '0 0 8px 0', fontSize: '12px', fontWeight: '600' }}>
                Resources:
              </p>
              {alert.resources.map((resource, i) => (
                <div
                  key={i}
                  style={{
                    marginBottom: '8px',
                    padding: '8px',
                    background: isImmediate ? '#fee2e2' : '#fef3c7',
                    borderRadius: '4px',
                    fontSize: '13px',
                  }}
                >
                  <div style={{ fontWeight: '500' }}>{resource.name}</div>
                  <div style={{ fontSize: '12px', opacity: 0.8, marginTop: '4px' }}>
                    {resource.number && (
                      <div>
                        Call: <strong>{resource.number}</strong>
                      </div>
                    )}
                    {resource.url && (
                      <div>
                        <a href={resource.url} target="_blank" rel="noopener noreferrer">
                          {resource.description}
                        </a>
                      </div>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}

          {alert.recommendations && alert.recommendations.length > 0 && (
            <div>
              <p style={{ margin: '0 0 6px 0', fontSize: '12px', fontWeight: '500' }}>
                Recommendations:
              </p>
              <ul style={{ margin: '0', paddingLeft: '20px', fontSize: '12px' }}>
                {alert.recommendations.slice(0, 3).map((rec, i) => (
                  <li key={i}>{rec}</li>
                ))}
              </ul>
            </div>
          )}
        </div>

        {onDismiss && (
          <button
            onClick={onDismiss}
            style={{
              background: 'transparent',
              border: 'none',
              cursor: 'pointer',
              fontSize: '16px',
              padding: '0',
              marginLeft: '12px',
              opacity: 0.7,
            }}
            title="Dismiss"
          >
            ✕
          </button>
        )}
      </div>
    </div>
  );
};

export default SafetyAlert;
