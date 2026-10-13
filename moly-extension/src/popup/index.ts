/**
 * Popup Entry Point
 */

import React from 'react';
import ReactDOM from 'react-dom/client';
import PopupEnhanced from './PopupEnhanced';

import { installBackendAuth } from '@/api/backendAuth';
import { installSessionRenewal } from '@/api/sessionRenewal';

// The backend token must be known before the first request, so the app renders once the stored connection is loaded.
installBackendAuth().finally(() => {
  installSessionRenewal();
  const root = ReactDOM.createRoot(document.getElementById('root')!);
  root.render(React.createElement(PopupEnhanced));
});
