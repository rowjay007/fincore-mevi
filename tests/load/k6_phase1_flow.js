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
    http_req_duration: ['p(95)<400'],
    http_req_failed: ['rate<0.01'],
  },
};

const AUTH_URL = __ENV.AUTH_SERVICE_URL || 'http://localhost:8082';
const ACCOUNT_URL = __ENV.ACCOUNT_SERVICE_URL || 'http://localhost:8080';
const LEDGER_URL = __ENV.LEDGER_SERVICE_URL || 'http://localhost:8083';

function getAccessToken() {
  if (__ENV.K6_ACCESS_TOKEN && __ENV.K6_ACCESS_TOKEN.trim() !== '') {
    return __ENV.K6_ACCESS_TOKEN.trim();
  }
  const email = (__ENV.K6_AUTH_EMAIL || '').trim();
  const password = (__ENV.K6_AUTH_PASSWORD || '').trim();
  if (email === '' || password === '') {
    throw new Error('missing auth config: set K6_ACCESS_TOKEN or (K6_AUTH_EMAIL and K6_AUTH_PASSWORD)');
  }
  const res = http.post(`${AUTH_URL}/v1/auth/login`, JSON.stringify({ email, password }), {
    headers: { 'Content-Type': 'application/json' },
  });
  check(res, { 'login status is 200': (r) => r.status === 200 });
  if (res.status !== 200) {
    throw new Error(`login failed: ${res.status}`);
  }
  const tok = res.json('access_token');
  if (!tok) {
    throw new Error('login response missing access_token');
  }
  return tok;
}

const accessToken = getAccessToken();
const authzHeaders = {
  'Content-Type': 'application/json',
  Authorization: `Bearer ${accessToken}`,
};

export default function () {
  const customerId = uuidv4();
  const idempotencyKey = uuidv4();

  // 1. Open Account
  const openRes = http.post(`${ACCOUNT_URL}/v1/accounts`, JSON.stringify({
    customer_id: customerId,
    idempotency_key: idempotencyKey,
  }), { headers: authzHeaders });

  check(openRes, {
    'open account status is 200': (r) => r.status === 200,
  });

  if (openRes.status !== 200) return;
  const accountId = openRes.json().account_id;

  sleep(1);

  // 2. Deposit
  const depositRes = http.post(`${ACCOUNT_URL}/v1/accounts/${accountId}/deposit`, JSON.stringify({
    idempotency_key: uuidv4(),
    amount: { currency: 'NGN', amount_kobo: 100000 }, // 1000 NGN
    narration: 'Initial deposit',
  }), { headers: authzHeaders });

  check(depositRes, {
    'deposit status is 200': (r) => r.status === 200,
  });

  sleep(1);

  // 3. Withdraw
  const withdrawRes = http.post(`${ACCOUNT_URL}/v1/accounts/${accountId}/withdraw`, JSON.stringify({
    idempotency_key: uuidv4(),
    amount: { currency: 'NGN', amount_kobo: 50000 }, // 500 NGN
    narration: 'ATM withdrawal',
  }), { headers: authzHeaders });

  check(withdrawRes, {
    'withdraw status is 200': (r) => r.status === 200,
  });

  sleep(1);

  // 4. Get Account (Balance check)
  const getRes = http.get(`${ACCOUNT_URL}/v1/accounts/${accountId}`, { headers: { Authorization: `Bearer ${accessToken}` } });
  check(getRes, {
    'get account status is 200': (r) => r.status === 200,
    'balance is correct': (r) => r.json().available_balance.amount_kobo === 50000,
  });

  sleep(1);

  // 5. Ledger balance should match as source of truth
  const ledgerBalRes = http.get(`${LEDGER_URL}/v1/ledger/accounts/${accountId}/balance`, { headers: { Authorization: `Bearer ${accessToken}` } });
  check(ledgerBalRes, {
    'ledger get balance status is 200': (r) => r.status === 200,
    'ledger balance is correct': (r) => r.json().available_balance.amount_kobo === 50000,
  });
}
