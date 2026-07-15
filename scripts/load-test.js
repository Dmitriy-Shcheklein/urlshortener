import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter } from 'k6/metrics';

export const options = {
  vus: 100,
  duration: '30s',
};

const successCounter = new Counter('successful_requests');

export default function () {
  const baseUrl = __ENV.BASE_URL || 'http://localhost:8080';

  // Make initial request to get auth cookie
  const initRes = http.get(`${baseUrl}/`);
  const authCookie = initRes.cookies['auth'];

  if (!authCookie || authCookie.length === 0) {
    console.error('Failed to get auth cookie');
    return;
  }

  // Use the auth cookie for subsequent requests
  const cookieValue = authCookie[0].value;
  const headers = {
    'Content-Type': 'application/json',
    'Cookie': `auth=${cookieValue}`,
  };

  // Create short URL
  const payload = JSON.stringify({
    url: `https://example.com/path/${Math.random().toString(36).substring(7)}`,
  });

  const res = http.post(`${baseUrl}/api/shorten`, payload, { headers });

  check(res, {
    'status is 201 or 409': (r) => r.status === 201 || r.status === 409,
    'has result': (r) => r.body && r.body.length > 0,
  });

  if (res.status === 201 || res.status === 409) {
    successCounter.add(1);
  }

  sleep(0.1);
}
