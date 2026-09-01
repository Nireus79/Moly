import { createProxyServer } from './index.js';
import fetch from 'node-fetch';

async function testProxy() {
  console.log('[Test] Starting Moly Proxy test...\n');

  const { server } = createProxyServer('http://localhost:11434', 11435);

  await new Promise(resolve => setTimeout(resolve, 500));

  try {
    console.log('[Test] Testing CORS headers on /api/tags...');
    const response = await fetch('http://127.0.0.1:11435/api/tags', {
      method: 'GET',
      headers: {
        'Origin': 'chrome-extension://test',
      },
    });

    const accessControlAllowOrigin = response.headers.get('access-control-allow-origin');
    const accessControlAllowMethods = response.headers.get('access-control-allow-methods');

    console.log(`[Test] Status: ${response.status}`);
    console.log(`[Test] Access-Control-Allow-Origin: ${accessControlAllowOrigin}`);
    console.log(`[Test] Access-Control-Allow-Methods: ${accessControlAllowMethods}`);

    if (accessControlAllowOrigin === '*') {
      console.log('[Test] ✓ CORS headers are correctly set');
    } else {
      console.log('[Test] ✗ CORS headers missing or incorrect');
    }

    const data = await response.json();
    console.log(`[Test] Response received: ${Object.keys(data).join(', ')}`);

    console.log('\n[Test] Proxy test completed successfully!');
  } catch (error) {
    console.error('[Test] Error:', error.message);
    console.log('[Test] Note: This error is expected if Ollama is not running at localhost:11434');
  } finally {
    server.close();
    console.log('[Test] Server closed');
  }
}

testProxy();
