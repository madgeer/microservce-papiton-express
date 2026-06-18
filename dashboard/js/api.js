/* PAPITON Express — API Service Module */

const PROXY_URL = '';

export async function getDwhMetrics() {
  const response = await fetch(`${PROXY_URL}/api/metrics`);
  if (!response.ok) throw new Error('DWH API unreachable');
  return response.json();
}

export async function apiCalculateTariff(payload) {
  const response = await fetch(`${PROXY_URL}/api/v1/tariff/calculate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  return response.json();
}

export async function apiCreateOrder(payload) {
  const response = await fetch(`${PROXY_URL}/api/v1/orders`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  if (!response.ok) throw new Error('Order creation failed');
  return response.json();
}

export async function apiRegisterCourier(payload) {
  const response = await fetch(`${PROXY_URL}/api/v1/couriers/register`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  if (!response.ok) throw new Error('Courier registration failed');
  return response.json();
}

export async function apiAutoDispatch(payload) {
  const response = await fetch(`${PROXY_URL}/api/v1/dispatch`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  if (!response.ok) {
    const err = await response.json();
    throw new Error(err.message || 'Auto dispatch failed');
  }
  return response.json();
}

export async function apiConfirmPickUp(dispatchId) {
  const response = await fetch(`${PROXY_URL}/api/v1/dispatches/confirm`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ dispatch_id: dispatchId })
  });
  return response.json();
}

export async function apiProcessInbound(payload) {
  const response = await fetch(`${PROXY_URL}/api/v1/inbound`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  return response.json();
}

export async function apiCreateManifest(payload) {
  const response = await fetch(`${PROXY_URL}/api/v1/manifest/create`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  return response.json();
}

export async function apiAddToManifest(payload) {
  const response = await fetch(`${PROXY_URL}/api/v1/manifest/add`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  return response.json();
}

export async function apiUpdateManifest(payload) {
  const response = await fetch(`${PROXY_URL}/api/v1/manifest/update`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  return response.json();
}

export async function apiGetTrackingHistory(resi) {
  const response = await fetch(`${PROXY_URL}/api/v1/tracking?resi_id=${resi}`);
  if (!response.ok) throw new Error('Tracking data not found');
  return response.json();
}

export async function apiSendManualScan(payload) {
  const response = await fetch(`${PROXY_URL}/api/v1/tracking/scan`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  });
  return response.json();
}

export async function apiGetCourierLocation(courierID) {
  const response = await fetch(`${PROXY_URL}/api/v1/couriers/location?courier_id=${courierID}`);
  if (!response.ok) throw new Error('Location not found');
  return response.json();
}
