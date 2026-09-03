#!/usr/bin/env node

/**
 * Moly CORS Proxy
 * Forwards requests from browser extension to local Ollama
 * Strips browser security headers to allow communication
 */

const http = require('http');
const process = require('process');

const PORT = 11435;
const OLLAMA_HOST = '127.0.0.1';
const OLLAMA_PORT = 11434;

// List of headers to strip (browser security headers)
const HEADERS_TO_STRIP = [
  'host',
  'origin',
  'sec-fetch-site',
  'sec-fetch-mode',
  'sec-fetch-dest',
  'sec-fetch-storage-access',
  'sec-ch-ua',
  'sec-ch-ua-platform',
  'sec-ch-ua-mobile',
  'sec-gpc',
];

const server = http.createServer((req, res) => {
  // Enable CORS headers for browser
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS, HEAD');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type, Accept');
  res.setHeader('Access-Control-Max-Age', '3600');

  // Handle CORS preflight
  if (req.method === 'OPTIONS') {
    res.writeHead(200);
    res.end();
    return;
  }

  // Prepare headers, stripping problematic ones
  const headers = { ...req.headers };
  HEADERS_TO_STRIP.forEach(header => {
    delete headers[header.toLowerCase()];
  });

  // Forward request to Ollama
  const options = {
    hostname: OLLAMA_HOST,
    port: OLLAMA_PORT,
    path: req.url,
    method: req.method,
    headers,
  };

  const ollama_req = http.request(options, (ollama_res) => {
    res.writeHead(ollama_res.statusCode, ollama_res.headers);
    ollama_res.pipe(res);
  });

  ollama_req.on('error', (err) => {
    res.writeHead(502);
    res.end(JSON.stringify({
      error: 'Bad Gateway',
      message: 'Failed to connect to Ollama at ' + OLLAMA_HOST + ':' + OLLAMA_PORT,
      details: err.message
    }));
  });

  req.pipe(ollama_req);
});

server.listen(PORT, '127.0.0.1', () => {
  console.log(`[Moly CORS Proxy] Listening on http://127.0.0.1:${PORT}`);
  console.log(`[Moly CORS Proxy] Forwarding to Ollama at http://${OLLAMA_HOST}:${OLLAMA_PORT}`);
});

server.on('error', (err) => {
  if (err.code === 'EADDRINUSE') {
    console.error(`[Moly CORS Proxy] ERROR: Port ${PORT} is already in use`);
    console.error(`[Moly CORS Proxy] Make sure only one instance of moly-proxy is running`);
    process.exit(1);
  } else {
    console.error(`[Moly CORS Proxy] Server error:`, err);
    process.exit(1);
  }
});

// Handle shutdown gracefully
process.on('SIGTERM', () => {
  console.log('[Moly CORS Proxy] Received SIGTERM, shutting down gracefully');
  server.close(() => {
    console.log('[Moly CORS Proxy] Server closed');
    process.exit(0);
  });
});

process.on('SIGINT', () => {
  console.log('[Moly CORS Proxy] Received SIGINT, shutting down gracefully');
  server.close(() => {
    console.log('[Moly CORS Proxy] Server closed');
    process.exit(0);
  });
});
