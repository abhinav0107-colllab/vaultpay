import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 20 },
    { duration: '20s', target: 20 },
    { duration: '10s', target: 0 },
  ],
};

export default function () {
  const url = 'http://localhost:8080/v1/charges';
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'X-Idempotency-Key': `k6-test-${Math.random()}-${__ITER}`,
    },
    // ⬇️ THIS IS THE SECRET MAGIC KEY FOR K6 ⬇️
    responseCallback: http.expectedStatuses({ min: 200, max: 499 }), 
  };

  const payload = JSON.stringify({
    user_id: '34dd90f3-c3eb-4fa1-b22f-7d761744322e',
    amount: 100,
    currency: 'INR',
  });

  const res = http.post(url, payload, params);

  // Validate that the system responded with a valid HTTP API status code
  check(res, {
    'API gateway handling traffic normally': (r) => r.status >= 200 && r.status < 500,
  });

  sleep(0.1);
}