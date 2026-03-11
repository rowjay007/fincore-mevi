import http from 'k6/http';
import { check, sleep } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

export const options = {
  stages: [
    { duration: '30s', target: 50 }, // ramp up
    { duration: '1m', target: 50 },  // stay at 50 users
    { duration: '30s', target: 0 },  // ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<200'], // 95% of requests must complete below 200ms
  },
};

const BASE_URL = __ENV.ACCOUNT_SERVICE_URL || 'http://localhost:8080';

export default function () {
  const customerId = uuidv4();
  const idempotencyKey = uuidv4();

  // 1. Open Account
  const openRes = http.post(`${BASE_URL}/v1/accounts`, JSON.stringify({
    customer_id: customerId,
    idempotency_key: idempotencyKey,
  }), { headers: { 'Content-Type': 'application/json' } });

  check(openRes, {
    'open account status is 200': (r) => r.status === 200,
  });

  if (openRes.status !== 200) return;
  const accountId = openRes.json().account_id;

  sleep(1);

  // 2. Deposit
  const depositRes = http.post(`${BASE_URL}/v1/accounts/${accountId}/deposit`, JSON.stringify({
    idempotency_key: uuidv4(),
    amount: { currency: 'NGN', amount_kobo: 100000 }, // 1000 NGN
    narration: 'Initial deposit',
  }), { headers: { 'Content-Type': 'application/json' } });

  check(depositRes, {
    'deposit status is 200': (r) => r.status === 200,
  });

  sleep(1);

  // 3. Withdraw
  const withdrawRes = http.post(`${BASE_URL}/v1/accounts/${accountId}/withdraw`, JSON.stringify({
    idempotency_key: uuidv4(),
    amount: { currency: 'NGN', amount_kobo: 50000 }, // 500 NGN
    narration: 'ATM withdrawal',
  }), { headers: { 'Content-Type': 'application/json' } });

  check(withdrawRes, {
    'withdraw status is 200': (r) => r.status === 200,
  });

  sleep(1);

  // 4. Get Account (Balance check)
  const getRes = http.get(`${BASE_URL}/v1/accounts/${accountId}`);
  check(getRes, {
    'get account status is 200': (r) => r.status === 200,
    'balance is correct': (r) => r.json().available_balance.amount_kobo === 50000,
  });
}
