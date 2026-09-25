import test from 'node:test';
import assert from 'node:assert';
import http from 'node:http';
import app from '../src/server';

test('Worker Health Endpoint returns 200 and UP status', async () => {
  const address = app.listen(0).address();
  const port = typeof address === 'object' && address !== null ? address.port : 8083;

  const res = await fetch(`http://localhost:${port}/health`);
  assert.strictEqual(res.status, 200);
  const data = await res.json();
  assert.strictEqual(data.service, 'kather_baksho-worker-ts');
  assert.strictEqual(data.status, 'UP');
});

