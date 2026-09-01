#!/usr/bin/env node

import { createProxyServer } from './index.js';

const ollamaUrl = process.env.OLLAMA_URL || 'http://localhost:11434';
const proxyPort = parseInt(process.env.MOLY_PROXY_PORT || '11435', 10);

const args = process.argv.slice(2);
if (args.includes('--help') || args.includes('-h')) {
  console.log(`
Moly CORS Proxy - Local Ollama Bridge

Usage:
  moly-proxy [options]

Options:
  --ollama-url <url>     Ollama server URL (default: http://localhost:11434)
  --port <port>          Proxy port (default: 11435)
  --help, -h             Show this help message

Environment Variables:
  OLLAMA_URL            Override Ollama server URL
  MOLY_PROXY_PORT       Override proxy port

Example:
  moly-proxy --port 11435 --ollama-url http://localhost:11434

  `);
  process.exit(0);
}

let customOllamaUrl = ollamaUrl;
let customProxyPort = proxyPort;

for (let i = 0; i < args.length; i++) {
  if ((args[i] === '--ollama-url' || args[i] === '-u') && args[i + 1]) {
    customOllamaUrl = args[i + 1];
    i++;
  } else if ((args[i] === '--port' || args[i] === '-p') && args[i + 1]) {
    customProxyPort = parseInt(args[i + 1], 10);
    i++;
  }
}

console.log(`
╔════════════════════════════════════════╗
║      Moly CORS Proxy - Starting        ║
╚════════════════════════════════════════╝

Configuration:
  Ollama URL: ${customOllamaUrl}
  Proxy Port: ${customProxyPort}
  Proxy URL:  http://127.0.0.1:${customProxyPort}

Status:
  ✓ Proxy server starting...
  ✓ CORS headers will be added to all requests
  ✓ Moly extension can now communicate with Ollama

Instructions:
  1. Make sure Ollama is running at ${customOllamaUrl}
  2. Configure Moly extension with base URL: http://127.0.0.1:${customProxyPort}
  3. Auto-detection should find this proxy automatically

To stop the proxy, press Ctrl+C

`);

const { server } = createProxyServer(customOllamaUrl, customProxyPort);

process.on('SIGINT', () => {
  console.log('\n[Moly Proxy] Shutting down gracefully...');
  server.close(() => {
    console.log('[Moly Proxy] Closed');
    process.exit(0);
  });
});

process.on('SIGTERM', () => {
  server.close(() => {
    process.exit(0);
  });
});
