/**
 * Popup Entry Point
 */

import React from 'react';
import ReactDOM from 'react-dom/client';
import PopupEnhanced from './PopupEnhanced';

import { installBackendAuth } from '@/api/backendAuth';

// The backend token must be known before the first request, so the app renders once the stored connection is loaded.
installBackendAuth().finally(() => {
  const root = ReactDOM.createRoot(document.getElementById('root')!);
  root.render(React.createElement(PopupEnhanced));
});
