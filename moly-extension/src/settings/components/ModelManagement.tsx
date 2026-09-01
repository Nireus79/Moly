import React, { useState } from 'react';
import type { LocalModelStatus } from '@/api/detection';

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
  const [selectedNewModel, setSelectedNewModel] = useState('mistral');
  const [isAdding, setIsAdding] = useState(false);

  const allModels = [...new Set([...status.ollama.models])];

  if (allModels.length === 0) {
    return null; // Only show if models are available
  }

  const handleAddModel = async () => {
    setIsAdding(true);
    try {
      const command = `ollama pull ${selectedNewModel}`;

      try {
        await navigator.clipboard.writeText(command);
        alert(
          `Command copied to clipboard!\n\n${command}\n\n` +
          `1. Open terminal\n` +
          `2. Paste and run the command\n` +
          `3. Wait for download to complete\n` +
          `4. Moly will detect it automatically when you refresh\n` +
          `5. Click "Refresh Models" button above to rescan`
        );
      } catch {
        alert(
          `Copy this command to terminal:\n\n${command}\n\n` +
          `1. Open terminal\n` +
          `2. Paste and run the command\n` +
          `3. Wait for download to complete\n` +
          `4. Click "Refresh Models" button above to rescan`
        );
      }

      setShowAddModel(false);
    } finally {
      setIsAdding(false);
    }
  };

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
      {!showAddModel ? (
        <button
          onClick={() => setShowAddModel(true)}
          style={{
            width: '100%',
            padding: '8px 12px',
            fontSize: '12px',
            background: '#f5f5f5',
            border: '1px solid #ddd',
            borderRadius: '4px',
            cursor: 'pointer',
            color: '#666',
            fontWeight: '500',
            transition: 'all 0.2s',
          }}
        >
          + Add Another Model
        </button>
      ) : (
        <div
          style={{
            padding: '12px',
            background: '#fff9c4',
            border: '1px solid #fff59d',
            borderRadius: '4px',
          }}
        >
          <div
            style={{
              fontSize: '12px',
              fontWeight: '600',
              color: '#f57f17',
              marginBottom: '8px',
            }}
          >
            Pull Another Model
          </div>

          <select
            value={selectedNewModel}
            onChange={(e) => setSelectedNewModel(e.target.value)}
            style={{
              width: '100%',
              padding: '6px',
              fontSize: '12px',
              marginBottom: '8px',
              borderRadius: '4px',
              border: '1px solid #ddd',
            }}
          >
            <optgroup label="Popular">
              <option value="mistral">Mistral 7B (4GB) - Recommended</option>
              <option value="llama2">Llama 2 7B (4GB)</option>
              <option value="neural-chat">Neural Chat 7B (4GB)</option>
              <option value="stable-code">Stable Code 3B (2GB)</option>
            </optgroup>
            <optgroup label="Larger">
              <option value="neural-chat-7b">Neural Chat 7B</option>
              <option value="dolphin-mixtral">Dolphin Mixtral 8x7B</option>
            </optgroup>
            <optgroup label="Smaller">
              <option value="tinyllama">TinyLlama 1.1B</option>
              <option value="phi">Phi 2.7B</option>
            </optgroup>
          </select>

          <div style={{ display: 'flex', gap: '8px' }}>
            <button
              onClick={handleAddModel}
              disabled={isAdding}
              style={{
                flex: 1,
                padding: '6px 12px',
                fontSize: '12px',
                background: '#f57f17',
                color: 'white',
                border: 'none',
                borderRadius: '4px',
                cursor: isAdding ? 'not-allowed' : 'pointer',
                opacity: isAdding ? 0.7 : 1,
              }}
            >
              {isAdding ? 'Copying...' : 'Copy Command'}
            </button>
            <button
              onClick={() => setShowAddModel(false)}
              style={{
                flex: 1,
                padding: '6px 12px',
                fontSize: '12px',
                background: 'white',
                color: '#666',
                border: '1px solid #ddd',
                borderRadius: '4px',
                cursor: 'pointer',
              }}
            >
              Cancel
            </button>
          </div>
        </div>
      )}

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
