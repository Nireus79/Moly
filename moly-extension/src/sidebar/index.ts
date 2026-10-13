/**
 * Sidebar Entry Point -.1 Auth-gated Chat Interface
 */

import React from 'react';
import ReactDOM from 'react-dom/client';
import { SidebarApp } from './SidebarApp';

import { installBackendAuth } from '@/api/backendAuth';

// The backend token must be known before the first request, so the app renders once the stored connection is loaded.
installBackendAuth().finally(() => {
  const root = ReactDOM.createRoot(document.getElementById('root')!);
  root.render(React.createElement(SidebarApp));
});
