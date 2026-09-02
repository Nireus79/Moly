/**
 * Uninstall page - handles model removal decision
 */

// Connect to background script to access native host
const bg = chrome.extension.getBackgroundPage();

interface Message {
  action: string;
  [key: string]: any;
}

// Send message to background (which can talk to native host)
function sendToBackground(message: Message): Promise<any> {
  return new Promise((resolve) => {
    if (bg && typeof (bg as any).handleUninstallMessage === 'function') {
      (bg as any).handleUninstallMessage(message, resolve);
    } else {
      resolve({ error: 'Background not available' });
    }
  });
}

async function loadModels() {
  try {
    const response = await new Promise<any>((resolve) => {
      chrome.runtime.sendNativeMessage(
        'com.moly.native_host',
        { action: 'get-models' },
        (response) => {
          if (chrome.runtime.lastError) {
            resolve({ error: chrome.runtime.lastError.message });
          } else {
            resolve(response || {});
          }
        }
      );
    });

    const modelsList = document.getElementById('modelsList');
    const question = document.getElementById('question');

    if (response.success && response.models && response.models.length > 0) {
      const html = response.models
        .map(
          (model: any) =>
            `<li>${model.name}</li>`
        )
        .join('');
      modelsList!.innerHTML = `<ul class="models-list">${html}</ul>`;
    } else {
      modelsList!.innerHTML = '<div class="no-models">No local models installed</div>';
      question!.textContent = 'No local models to remove.';
      document.getElementById('btnRemove')!.style.display = 'none';
    }
  } catch (error) {
    console.error('Failed to load models:', error);
    const modelsList = document.getElementById('modelsList');
    modelsList!.innerHTML =
      '<div class="no-models">Could not check for models (native host not available)</div>';
  }
}

async function cleanup(keepModels: boolean) {
  const statusEl = document.getElementById('status');
  const btnKeep = document.getElementById('btnKeep') as HTMLButtonElement | null;
  const btnRemove = document.getElementById('btnRemove') as HTMLButtonElement | null;

  // Disable buttons
  if (btnKeep) btnKeep.disabled = true;
  if (btnRemove) btnRemove.disabled = true;

  // Show status
  statusEl!.textContent = keepModels
    ? 'Removing Moly data (keeping models)...'
    : 'Removing Moly data and models...';
  statusEl!.classList.add('show');

  try {
    const response = await new Promise<any>((resolve) => {
      chrome.runtime.sendNativeMessage(
        'com.moly.native_host',
        { action: 'cleanup', keep_models: keepModels },
        (response) => {
          if (chrome.runtime.lastError) {
            resolve({ error: chrome.runtime.lastError.message });
          } else {
            resolve(response || {});
          }
        }
      );
    });

    if (response.success) {
      statusEl!.textContent = keepModels
        ? '✓ Cleanup complete. Your models have been preserved.'
        : '✓ Cleanup complete. Models have been removed.';
      statusEl!.classList.add('success');

      // Wait a moment then close the tab
      setTimeout(() => {
        window.close();
      }, 2000);
    } else {
      throw new Error(response.error || 'Cleanup failed');
    }
  } catch (error) {
    statusEl!.textContent = `✗ ${error instanceof Error ? error.message : 'Cleanup failed'}`;
    statusEl!.classList.add('error');
    if (btnKeep) btnKeep.disabled = false;
    if (btnRemove) btnRemove.disabled = false;
  }
}

// Setup event listeners
document.addEventListener('DOMContentLoaded', () => {
  loadModels();

  document.getElementById('btnKeep')?.addEventListener('click', () => {
    cleanup(true);
  });

  document.getElementById('btnRemove')?.addEventListener('click', () => {
    if (window.confirm('Are you sure? Removing models cannot be undone.')) {
      cleanup(false);
    }
  });
});
