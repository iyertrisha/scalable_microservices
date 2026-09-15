import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    mixed: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '15s', target: 20 },
        { duration: '20s', target: 50 },
        { duration: '10s', target: 0 },
      ],
    },
  },
  thresholds: {
    // Only count order API requests toward failure budget.
    'http_req_failed{name:order}': ['rate<0.05'],
    'http_req_duration{name:order}': ['p(95)<1500'],
  },
};

const BASE = __ENV.BASE_URL || 'http://localhost:8080';
const DRIVER = __ENV.DRIVER_URL || 'http://localhost:8081';
const LOCATION = __ENV.LOCATION_URL || 'http://localhost:8084';
const TRACKING = __ENV.TRACKING_URL || 'http://localhost:8083';

export function setup() {
  const res = http.post(
    `${DRIVER}/drivers`,
    JSON.stringify({ name: 'k6-driver', vehicle_type: 'bike', capacity: 1 }),
    { headers: { 'Content-Type': 'application/json' }, tags: { name: 'setup' } },
  );
  if (res.status === 201) {
    const id = res.json('driver_id');
    http.patch(`${DRIVER}/drivers/${id}/status`, JSON.stringify({ status: 'AVAILABLE' }), {
      headers: { 'Content-Type': 'application/json' },
      tags: { name: 'setup' },
    });
    return { driverId: id };
  }
  return { driverId: '' };
}

export default function (data) {
  const orderPayload = JSON.stringify({
    pickup_location: { lat: 12.97 + Math.random() * 0.02, lng: 77.59 + Math.random() * 0.02 },
    delivery_location: { lat: 12.98, lng: 77.60 },
    package_size: 'M',
    priority: Math.random() > 0.7 ? 'high' : 'normal',
  });

  const create = http.post(`${BASE}/orders`, orderPayload, {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'order' },
  });
  check(create, { 'order created': (r) => r.status === 201 });

  let orderId = '';
  try {
    orderId = create.json('order_id');
  } catch (_) {}

  if (orderId) {
    const get = http.get(`${BASE}/orders/${orderId}`, { tags: { name: 'order' } });
    check(get, { 'order get ok': (r) => r.status === 200 });
    http.get(`${TRACKING}/orders/${orderId}/tracking`, { tags: { name: 'tracking' } });
  }

  if (data.driverId) {
    http.post(
      `${LOCATION}/drivers/${data.driverId}/location`,
      JSON.stringify({ lat: 12.97 + Math.random() * 0.01, lng: 77.59 + Math.random() * 0.01 }),
      { headers: { 'Content-Type': 'application/json' }, tags: { name: 'location' } },
    );
  }

  sleep(0.2);
}
