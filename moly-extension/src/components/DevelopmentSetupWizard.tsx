import React, { useState } from 'react';

interface DevelopmentSetupWizardProps {
  extensionId: string;
  setupCommand: string;
  onDismiss: () => void;
}

export const DevelopmentSetupWizard: React.FC<DevelopmentSetupWizardProps> = ({
  extensionId,
  setupCommand,
  onDismiss,
}) => {
  const [copied, setCopied] = useState(false);

  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(setupCommand);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
        <div className="mb-4">
          <h2 className="text-xl font-bold text-gray-900 dark:text-white mb-2">
            🔧 Backend Setup Required
          </h2>
          <p className="text-sm text-gray-600 dark:text-gray-300">
            Development Mode - Configure native messaging for your extension
          </p>
        </div>

        <div className="bg-blue-50 dark:bg-blue-900 border border-blue-200 dark:border-blue-700 rounded p-3 mb-4">
          <p className="text-xs font-mono text-gray-700 dark:text-gray-300 mb-1">
            Your Extension ID:
          </p>
          <p className="text-sm font-mono font-bold text-blue-600 dark:text-blue-300 break-all">
            {extensionId}
          </p>
        </div>

        <div className="mb-4">
          <label className="block text-sm font-semibold text-gray-700 dark:text-gray-300 mb-2">
            Run this command in your terminal:
          </label>
          <div className="relative">
            <input
              type="text"
              value={setupCommand}
              readOnly
              className="w-full px-3 py-2 text-sm font-mono bg-gray-100 dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded text-gray-900 dark:text-gray-100"
            />
            <button
              onClick={copyToClipboard}
              className="absolute right-2 top-1/2 -translate-y-1/2 px-3 py-1 bg-blue-500 hover:bg-blue-600 text-white text-xs font-medium rounded transition"
            >
              {copied ? '✓ Copied' : 'Copy'}
            </button>
          </div>
        </div>

        <div className="bg-yellow-50 dark:bg-yellow-900 border border-yellow-200 dark:border-yellow-700 rounded p-3 mb-4">
          <p className="text-sm text-yellow-800 dark:text-yellow-200 font-semibold mb-2">
            Steps:
          </p>
          <ol className="text-xs text-yellow-700 dark:text-yellow-300 space-y-1 list-decimal list-inside">
            <li>Open a terminal or command prompt</li>
            <li>Copy the command above (click Copy button)</li>
            <li>Paste and run the command</li>
            <li>Wait for setup to complete</li>
            <li>Reload this extension (F5 or Cmd+R)</li>
          </ol>
        </div>

        <div className="flex gap-3">
          <button
            onClick={onDismiss}
            className="flex-1 px-4 py-2 bg-gray-200 dark:bg-gray-700 hover:bg-gray-300 dark:hover:bg-gray-600 text-gray-900 dark:text-white font-medium rounded transition"
          >
            Dismiss
          </button>
          <button
            onClick={copyToClipboard}
            className="flex-1 px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white font-medium rounded transition"
          >
            {copied ? '✓ Copied!' : 'Copy Command'}
          </button>
        </div>

        <p className="text-xs text-gray-500 dark:text-gray-400 mt-4">
          💡 Tip: This wizard only appears in development mode. In production, setup is automatic.
        </p>
      </div>
    </div>
  );
};
