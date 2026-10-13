/**
 * Settings Page Entry Point
 */

import '@/styles/theme.css';
import '@/styles/brand.css';
import '@/styles/marks.css';
import React from 'react';
import ReactDOM from 'react-dom/client';
import { Settings } from './Settings';
import { applyTheme } from '@/styles/theme';

applyTheme();

import { installBackendAuth } from '@/api/backendAuth';
import { installSessionRenewal } from '@/api/sessionRenewal';
import { useAuthStore } from '@/stores/authStore';

// The backend token must be known before the first request, so the app renders once the stored connection is loaded.
installBackendAuth().finally(() => {
  // The settings page can be opened on its own (not from the sidebar): it must know who is signed in.
  useAuthStore.getState().initFromStorage();
  installSessionRenewal();
  const root = ReactDOM.createRoot(document.getElementById('root')!);
  root.render(React.createElement(Settings));
});
