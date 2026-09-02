import React, { useState } from 'react';
import type { LocalModelStatus } from '@/api/detection';
import { ModelSelectionDialog } from './ModelSelectionDialog';

interface ModelManagementProps {
  status: LocalModelStatus;
  currentModel?: string;
  onModelSelect?: (model: string) => void;
}

export const ModelManagement: React.FC<ModelManagementProps> = ({
  status,
  currentModel,
  onModelSelect,
}) => {
  const [showAddModel, setShowAddModel] = useState(false);

  const allModels = [...new Set([...status.ollama.models])];

  if (allModels.length === 0) {
    return null; // Only show if models are available
  }

  if (showAddModel) {
    return (
      <ModelSelectionDialog
        onClose={() => setShowAddModel(false)}
        onSuccess={() => {
          setShowAddModel(false);
          // Refresh models list after successful installation
          window.location.reload();
        }}
      />
    );
  }

  return (
    <div
      style={{
        padding: '16px',
        background: '#fafafa',
        border: '1px solid #e0e0e0',
        borderRadius: '6px',
        marginBottom: '16px',
      }}
    >
      <div
        style={{
          fontSize: '14px',
          fontWeight: '600',
          color: '#333',
          marginBottom: '12px',
        }}
      >
        Available Models ({allModels.length})
      </div>

      {/* Model List */}
      <div
        style={{
          display: 'flex',
          flexDirection: 'column',
          gap: '8px',
          marginBottom: '12px',
        }}
      >
        {allModels.map((model) => (
          <div
            key={model}
            onClick={() => onModelSelect?.(model)}
            style={{
              padding: '10px 12px',
              background: currentModel === model ? '#e3f2fd' : 'white',
              border:
                currentModel === model
                  ? '2px solid #1976d2'
                  : '1px solid #e0e0e0',
              borderRadius: '4px',
              cursor: 'pointer',
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              transition: 'all 0.2s',
            }}
          >
            <div>
              <div
                style={{
                  fontSize: '13px',
                  fontWeight: currentModel === model ? '600' : '500',
                  color: '#333',
                }}
              >
                {model}
              </div>
              {currentModel === model && (
                <div
                  style={{
                    fontSize: '11px',
                    color: '#1976d2',
                    marginTop: '2px',
                  }}
                >
                  Currently using
                </div>
              )}
            </div>
            {currentModel === model && (
              <div
                style={{
                  fontSize: '18px',
                  color: '#1976d2',
                }}
              >
                ✓
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Add Model Section */}
      <button
        onClick={() => setShowAddModel(true)}
        style={{
          width: '100%',
          padding: '8px 12px',
          fontSize: '12px',
          background: '#1976d2',
          color: 'white',
          border: 'none',
          borderRadius: '4px',
          cursor: 'pointer',
          fontWeight: '600',
          transition: 'all 0.2s',
        }}
      >
        + Add Another Model
      </button>

      <div
        style={{
          marginTop: '12px',
          fontSize: '11px',
          color: '#999',
          lineHeight: '1.4',
        }}
      >
        Click a model to use it. To add new models, follow the instructions or
        run <code>ollama pull model-name</code> in terminal.
      </div>
    </div>
  );
};

export default ModelManagement;
