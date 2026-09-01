import express from 'express';
import cors from 'cors';
import fetch from 'node-fetch';

export function createProxyServer(ollamaUrl = 'http://localhost:11434', proxyPort = 11435) {
  const app = express();

  app.use(cors({
    origin: '*',
    methods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'],
    allowedHeaders: ['Content-Type', 'Authorization'],
    credentials: false,
  }));

  app.use(express.json({ limit: '50mb' }));
  app.use(express.text({ limit: '50mb' }));
  app.use(express.raw({ limit: '50mb' }));

  app.all('*', async (req, res) => {
    try {
      const targetUrl = new URL(req.originalUrl.substring(1) || '/', ollamaUrl);

      const fetchOptions = {
        method: req.method,
        headers: {
          ...req.headers,
          host: targetUrl.host,
        },
      };

      if (['POST', 'PUT', 'PATCH'].includes(req.method) && req.body) {
        if (Buffer.isBuffer(req.body)) {
          fetchOptions.body = req.body;
        } else if (typeof req.body === 'object') {
          fetchOptions.body = JSON.stringify(req.body);
          fetchOptions.headers['Content-Type'] = 'application/json';
        } else {
          fetchOptions.body = req.body;
        }
      }

      delete fetchOptions.headers['content-length'];
      delete fetchOptions.headers['host'];

      const response = await fetch(targetUrl.toString(), fetchOptions);
      const contentType = response.headers.get('content-type');

      if (contentType && contentType.includes('application/json')) {
        const data = await response.json();
        res.status(response.status)
          .set('Content-Type', 'application/json')
          .json(data);
      } else {
        const buffer = await response.buffer();
        res.status(response.status)
          .set('Content-Type', contentType || 'text/plain')
          .send(buffer);
      }
    } catch (error) {
      console.error('[Moly Proxy] Error:', error.message);
      res.status(502).json({
        error: `Proxy error: ${error.message}`,
        details: 'Failed to reach Ollama server at ' + ollamaUrl,
      });
    }
  });

  const server = app.listen(proxyPort, '127.0.0.1', () => {
    console.log(`[Moly Proxy] Started on http://127.0.0.1:${proxyPort}`);
    console.log(`[Moly Proxy] Proxying to ${ollamaUrl}`);
  });

  return { app, server };
}
