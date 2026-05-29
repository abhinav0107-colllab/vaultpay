import http from 'k6/http';
import { check, sleep } from 'k6';

// Define the ramping configuration to achieve 500 RPS safely
export const options = {
    stages: [
        { duration: '10s', target: 100 }, // Ramp up from 0 to 100 virtual users
        { duration: '30s', target: 500 }, // Aggressively scale up to 500 requests/sec
        { duration: '10s', target: 0 },   // Cool down back to zero
    ],
    thresholds: {
        // Enforce Day 19 Dashboard Requirement: 99% of requests must complete under 50ms
        http_req_duration: ['p(99)<50'], 
    },
};

export default function () {
// Change this line inside load_test.js:
const url = 'http://host.docker.internal:8080/v1/charges';    
    const payload = JSON.stringify({
        user_id: 'merchant_india@gmail.com',
        amount: 1000, // 1000 minor units (10.00 INR/USD)
        currency: 'INR'
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const res = http.post(url, payload, params);

    check(res, {
        'status is 200': (r) => r.status === 200,
    });

    sleep(0.1); // Short pacing gap to sustain continuous throughput loops
}