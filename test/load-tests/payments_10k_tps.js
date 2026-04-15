import http from 'k6/http';
import { check, sleep } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

export const options = {
  scenarios: {
    constant_request_rate: {
      executor: 'constant-arrival-rate',
      rate: 10000,
      timeUnit: '1s',
      duration: '1m',
      preAllocatedVUs: 500,
      maxVUs: 2000,
    },
  },
  thresholds: {
    http_req_duration: ['p(99)<50'], // Blueprint mandate: p99 < 50ms
    http_req_failed: ['rate<0.001'], // 99.9% success rate requirement
  },
};

const BASE_URL = __ENV.API_GATEWAY_URL || 'http://localhost:8080';

export default function () {
  const payload = JSON.stringify({
    id: uuidv4(),
    amount: '150.00',
    currency: 'USD',
    source_account: 'acc_123456',
    destination_account: 'acc_789012',
    description: 'k6 load test payment',
    idempotency_key: uuidv4(),
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${__ENV.TEST_TOKEN}`,
      'X-Correlation-ID': uuidv4(),
    },
  };

  const res = http.post(`${BASE_URL}/v1/payments`, payload, params);

  check(res, {
    'status is 201': (r) => r.status === 201,
    'has transaction_id': (r) => r.json('transaction_id') !== undefined,
  });
}
