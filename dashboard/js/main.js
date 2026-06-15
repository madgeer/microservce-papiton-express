/* PAPITON Express — Main Coordinator Module */

import * as api from './api.js';
import * as ui from './ui.js';

// Fallback Mock Data in case DWH backend is offline
const fallbackData = {
  total_delivery: 12540,
  total_revenue: 1845000000,
  avg_duration_hours: 55.2,
  notification_count: 24500,
  notification_success_rate: 98.2,
  driver_avg_earnings: 17500,
  driver_avg_rating: 4.75,
  driver_active_count: 12,
  monthly_volumes: [600, 850, 920, 1050, 1200, 1500, 1400, 1800, 2000, 2200, 2500, 2800],
  monthly_labels: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'],
  service_types: ['REGULAR', 'EXPRESS', 'CARGO'],
  service_counts: [6897, 3762, 1881],
  warehouse_ids: ['WH-BDG', 'WH-JKT', 'WH-SUB', 'WH-UPI'],
  warehouse_counts: [4500, 3900, 2800, 1340],
  notif_channels: ['email', 'push'],
  notif_success_rates: [99.1, 97.3],
  notif_failure_rates: [0.9, 2.7]
};

// 1. Fetch DWH metrics
async function fetchMetrics(isManual = false) {
  const syncText = document.getElementById('syncText');
  if (isManual) syncText.innerText = "DWH Live Sync: Refreshing...";
  
  try {
    const data = await api.getDwhMetrics();
    ui.updateDashboardDOM(data);
    syncText.innerText = "DWH Live Sync: Connected";
    syncText.parentElement.style.borderColor = 'rgba(16, 185, 129, 0.2)';
  } catch (e) {
    console.warn('API error, falling back to mock data:', e);
    ui.updateDashboardDOM(fallbackData);
    syncText.innerText = "DWH Live Sync: Offline (Mock Mode)";
    syncText.parentElement.style.borderColor = 'rgba(239, 68, 68, 0.2)';
  }
}

// 2. Calculate Tariff
async function calculateTariff() {
  const respBox = document.getElementById('customerResponse');
  respBox.classList.add('active');
  respBox.innerText = "Mengajukan estimasi tarif...";
  
  const payload = getOrderFormPayload();
  try {
    const data = await api.apiCalculateTariff(payload);
    respBox.innerText = JSON.stringify(data, null, 2);
  } catch (e) {
    respBox.innerText = "Koneksi API Gagal: " + e.message;
  }
}

// 3. Create Order
async function createOrder() {
  const respBox = document.getElementById('customerResponse');
  respBox.classList.add('active');
  respBox.innerText = "Membuat pesanan baru...";
  
  const payload = getOrderFormPayload();
  try {
    const data = await api.apiCreateOrder(payload);
    respBox.innerText = JSON.stringify(data, null, 2);
    if (data.awb) {
      ui.updateState('activeResi', data.awb);
      ui.triggerAlert('customerAlert');
    }
  } catch (e) {
    respBox.innerText = "Koneksi API Gagal: " + e.message;
  }
}

function getOrderFormPayload() {
  return {
    sender: {
      name: document.getElementById('senderName').value,
      phone: "08123456789",
      email: "pengirim@gmail.com",
      full_address: "Alamat Pengirim",
      city: document.getElementById('senderCity').value,
      coordinate: { latitude: -6.8915, longitude: 107.6106 }
    },
    recipient: {
      name: document.getElementById('recipientName').value,
      phone: "08987654321",
      email: "penerima@gmail.com",
      full_address: "Alamat Penerima",
      city: document.getElementById('recipientCity').value,
      coordinate: { latitude: -6.2088, longitude: 106.8456 }
    },
    package: {
      length: 30, width: 20, height: 10,
      actual_weight: parseFloat(document.getElementById('pkgWeight').value)
    },
    service_type: document.getElementById('pkgService').value,
    has_insurance: document.getElementById('hasInsurance').checked,
    has_packing: document.getElementById('hasPacking').checked
  };
}

// 4. Register Driver
async function registerDriver() {
  const payload = {
    id: document.getElementById('driverId').value,
    name: document.getElementById('driverName').value,
    phone_number: "081223344",
    zone: document.getElementById('driverZone').value,
    status: "AVAILABLE",
    vehicle_type: document.getElementById('driverVehicle').value
  };
  
  try {
    await api.apiRegisterCourier(payload);
    ui.triggerAlert('driverRegisterAlert');
  } catch (e) {
    alert("Gagal registrasi driver: " + e.message);
  }
}

// 5. Auto Dispatch Courier
async function autoDispatch() {
  const resBox = document.getElementById('driverResponse');
  resBox.classList.add('active');
  resBox.innerText = "Mencari kurir...";

  const payload = {
    order_id: document.getElementById('dispatchAwb').value,
    pickup_zone: document.getElementById('dispatchZone').value
  };

  try {
    const data = await api.apiAutoDispatch(payload);
    resBox.innerText = JSON.stringify(data, null, 2);
    if (data.id) {
      ui.updateState('activeDsp', data.id);
      ui.triggerAlert('driverDispatchAlert');
    }
  } catch (e) {
    resBox.innerText = "Error penugasan kurir: " + e.message;
  }
}

// 6. Confirm Pick Up
async function confirmPickUp() {
  const resBox = document.getElementById('driverResponse');
  resBox.classList.add('active');
  resBox.innerText = "Mengirimkan konfirmasi pengambilan kurir...";

  try {
    const data = await api.apiConfirmPickUp(ui.state.activeDsp);
    resBox.innerText = JSON.stringify(data, null, 2);
  } catch (e) {
    resBox.innerText = "Error PickUp: " + e.message;
  }
}

// 7. Process Scan Inbound
async function processInbound() {
  const resBox = document.getElementById('warehouseResponse');
  resBox.classList.add('active');
  resBox.innerText = "Memproses scan inbound...";

  const payload = {
    resi: document.getElementById('inboundAwb').value,
    warehouse_id: document.getElementById('inboundWh').value
  };

  try {
    const data = await api.apiProcessInbound(payload);
    resBox.innerText = JSON.stringify(data, null, 2);
    ui.triggerAlert('warehouseAlert');
  } catch (e) {
    resBox.innerText = "Error Inbound: " + e.message;
  }
}

// 8. Track & Trace Awb
async function trackAwb() {
  const resi = document.getElementById('searchAwb').value;
  const errAlert = document.getElementById('trackErrorAlert');
  const resultArea = document.getElementById('trackingResultArea');

  errAlert.classList.remove('active');
  resultArea.style.display = 'none';

  if (!resi) return;

  try {
    const data = await api.apiGetTrackingHistory(resi);
    if (!data.history || data.history.length === 0) {
      errAlert.classList.add('active');
      return;
    }
    resultArea.style.display = 'block';
    ui.renderTrackingTimeline(data.history);
  } catch (e) {
    errAlert.classList.add('active');
  }
}

// 9. Send Manual Scan
async function sendManualScan() {
  const payload = {
    resi_id: document.getElementById('searchAwb').value,
    activity_code: document.getElementById('manualActivity').value,
    location_code: document.getElementById('manualLocation').value,
    timestamp: new Date().toISOString()
  };

  try {
    await api.apiSendManualScan(payload);
    ui.triggerAlert('manualScanAlert');
    trackAwb(); // Refresh timeline
  } catch (e) {
    alert("Gagal kirim scan log: " + e.message);
  }
}

// Bind module functions to the global window scope
window.switchTab = ui.switchTab;
window.calculateTariff = calculateTariff;
window.createOrder = createOrder;
window.registerDriver = registerDriver;
window.autoDispatch = autoDispatch;
window.confirmPickUp = confirmPickUp;
window.processInbound = processInbound;
window.trackAwb = trackAwb;
window.sendManualScan = sendManualScan;
window.fetchMetrics = fetchMetrics;

// Auto fetch setup on start
window.addEventListener('DOMContentLoaded', () => {
  fetchMetrics();
  ui.updateVariablesUI();
  setInterval(() => fetchMetrics(false), 5000);
});
