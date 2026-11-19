import http from 'node:http';
import { AggrepayClient } from '../src/index.js';

const client = new AggrepayClient(
  process.env.AGGREPAY_API_KEY || 'YOUR_API_KEY',
  process.env.AGGREPAY_SECRET_KEY || 'YOUR_SECRET_KEY',
);

const server = http.createServer(async (req, res) => {
  if (req.method !== 'POST') {
    res.writeHead(405).end();
    return;
  }
  const chunks = [];
  for await (const c of req) chunks.push(c);
  const raw = Buffer.concat(chunks).toString('utf8');
  const ts = req.headers['x-timestamp'] || '';
  const sig = req.headers['x-signature'] || '';

  if (!client.verifyCallback(raw, ts, sig)) {
    res.writeHead(401, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ ok: false, error: 'invalid signature' }));
    return;
  }

  const payload = JSON.parse(raw);
  console.log('callback ok', payload.order_id, payload.status);
  res.writeHead(200, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify({ ok: true }));
});

server.listen(8787, () => console.log('callback listener :8787'));
