#!/usr/bin/env node
/**
 * Webhook receiver for SAI AUROSY events with HMAC verification.
 * Run: npm install && node server.js
 * Env: WEBHOOK_SECRET for HMAC verification
 * Port: PORT (default 5000)
 *
 * E2E: GET /last-event returns the last received webhook (event + payload).
 */
const crypto = require('crypto');
const express = require('express');

const app = express();
app.use(express.raw({ type: 'application/json' }));

const SECRET = process.env.WEBHOOK_SECRET || '';

let lastEvent = null;

function verifySignature(body, signature, secret) {
  if (!secret || !signature.startsWith('sha256=')) return false;
  const expected = 'sha256=' + crypto
    .createHmac('sha256', secret)
    .update(body)
    .digest('hex');
  return crypto.timingSafeEqual(Buffer.from(expected), Buffer.from(signature));
}

app.post('/webhooks/sai-aurosy', (req, res) => {
  const body = req.body;
  const sig = req.headers['x-webhook-signature'] || '';
  const event = req.headers['x-webhook-event'] || '';

  if (SECRET && !verifySignature(body, sig, SECRET)) {
    return res.status(401).json({ error: 'Invalid signature' });
  }

  let payload;
  try {
    payload = JSON.parse(body.toString());
  } catch {
    payload = {};
  }
  lastEvent = { event, payload };
  console.log(`[${event}]`, JSON.stringify(payload, null, 2));

  res.status(200).end();
});

app.get('/last-event', (req, res) => {
  res.setHeader('Content-Type', 'application/json');
  res.end(JSON.stringify(lastEvent || {}));
});

const port = process.env.PORT || 5000;
app.listen(port, () => {
  console.log(`Webhook receiver on http://localhost:${port}/webhooks/sai-aurosy`);
  console.log('Set WEBHOOK_SECRET env for HMAC verification');
});
