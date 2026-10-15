// A stand-in for the two AI vendors' model lists (a test container has no network and no key). Anthropic is told by the
// x-api-key header, OpenAI by the Authorization header. The Anthropic list is refused when it carries the filter parameter the
// real API rejected (2026-10-14), so a wrong request fails here too.
import http from 'node:http';

const port = Number(process.argv[2] || 11999);
http.createServer((req, res) => {
  const url = new URL(req.url, 'http://x');
  const send = (code, body) => { res.writeHead(code, { 'Content-Type': 'application/json' }); res.end(JSON.stringify(body)); };
  if (req.method === 'GET' && url.pathname === '/v1/models') {
    if (req.headers['x-api-key']) {
      if (url.searchParams.has('lifecycle')) return send(400, { type: 'error', error: { type: 'invalid_request_error', message: 'the parameter name is `lifecycle[]`' } });
      return send(200, { data: [
        { id: 'claude-opus-e2e', display_name: 'Claude Opus E2E', line: 'opus', lifecycle: 'active' },
        { id: 'claude-sonnet-5-5', display_name: 'Claude Sonnet 5.5', line: 'sonnet', lifecycle: 'active' },
        { id: 'claude-haiku-5-5', display_name: 'Claude Haiku 5.5', line: 'haiku', lifecycle: 'active' },
        { id: 'claude-old-e2e', display_name: 'Claude Old E2E', line: 'haiku', lifecycle: 'retired' },
      ], has_more: false });
    }
    if ((req.headers.authorization || '').startsWith('Bearer ')) {
      return send(200, { data: [
        { id: 'gpt-e2e-new', created: 300 }, { id: 'text-embedding-e2e', created: 400 }, { id: 'gpt-e2e-old', created: 100 },
      ] });
    }
    return send(401, { error: 'no key' });
  }
  send(404, { error: 'not found' });
}).listen(port, '127.0.0.1', () => console.log('fake vendors on', port));
