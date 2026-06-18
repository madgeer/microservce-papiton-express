/* PAPITON Express — Main Coordinator Module */

import * as api from './api.js';
import * as ui from './ui.js';

let mapInstance = null;
let markerInstance = null;

// Customer Desk Map Variables
let customerMap = null;
let senderMarker = null;
let recipientMarker = null;
let routeLine = null;
let currentSenderCoords = [-6.9175, 107.6191]; // Default Bandung
let currentRecipientCoords = [-6.2088, 106.8456]; // Default Jakarta

const LocationCoords = {
  'WH-BDG': [-6.9175, 107.6191],
  'WH-BDO': [-6.9175, 107.6191],
  'BANDUNG': [-6.9175, 107.6191],
  'WH-JKT': [-6.2088, 106.8456],
  'JAKARTA': [-6.2088, 106.8456],
  'WH-SUB': [-7.2575, 112.7521],
  'SURABAYA': [-7.2575, 112.7521],
  'WH-UPI': [-6.8619, 107.5944] // Kampus UPI Bandung
};

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

// Helper to Geocode address using OpenStreetMap Nominatim API
async function geocodeAddress(address, fallbackCoords) {
  if (!address || address.trim() === "") return fallbackCoords;
  try {
    const encodedAddress = encodeURIComponent(address + ", Indonesia");
    const response = await fetch(`https://nominatim.openstreetmap.org/search?q=${encodedAddress}&format=json&limit=1`, {
      headers: {
        'Accept': 'application/json'
      }
    });
    if (!response.ok) throw new Error("Geocoding network error");
    const results = await response.json();
    if (results && results.length > 0) {
      const lat = parseFloat(results[0].lat);
      const lon = parseFloat(results[0].lon);
      console.log(`Geocoded address "${address}" to: [${lat}, ${lon}]`);
      return [lat, lon];
    }
  } catch (e) {
    console.warn("Geocoding failed, falling back to default city coordinates:", e);
  }
  return fallbackCoords;
}

// Reverse Geocoding helper
async function reverseGeocode(lat, lon, inputId, cityId) {
  try {
    const response = await fetch(`https://nominatim.openstreetmap.org/reverse?lat=${lat}&lon=${lon}&format=json&addressdetails=1`);
    if (!response.ok) return null;
    const data = await response.json();
    if (data && data.display_name) {
      document.getElementById(inputId).value = data.display_name;
      
      const citySelect = document.getElementById(cityId);
      if (citySelect) {
        const addr = data.address || {};
        const regionText = JSON.stringify(addr).toLowerCase();
        
        if (regionText.includes('jakarta')) {
          citySelect.value = 'Jakarta';
        } else if (regionText.includes('surabaya') || regionText.includes('jawa timur')) {
          citySelect.value = 'Surabaya';
        } else if (regionText.includes('bandung') || regionText.includes('jawa barat') || regionText.includes('banten')) {
          citySelect.value = 'Bandung';
        }
      }
      return data.display_name;
    }
  } catch (e) {
    console.error("Reverse geocoding error:", e);
  }
  return null;
}

// Initialize Customer Desk Map
function initCustomerMap() {
  if (customerMap) return;
  const container = document.getElementById('customerMap');
  if (!container) return;

  const blueIcon = new L.Icon({
    iconUrl: 'https://raw.githubusercontent.com/pointhi/leaflet-color-markers/master/img/marker-icon-2x-blue.png',
    shadowUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/0.7.7/images/marker-shadow.png',
    iconSize: [25, 41],
    iconAnchor: [12, 41],
    popupAnchor: [1, -34],
    shadowSize: [41, 41]
  });

  const redIcon = new L.Icon({
    iconUrl: 'https://raw.githubusercontent.com/pointhi/leaflet-color-markers/master/img/marker-icon-2x-red.png',
    shadowUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/0.7.7/images/marker-shadow.png',
    iconSize: [25, 41],
    iconAnchor: [12, 41],
    popupAnchor: [1, -34],
    shadowSize: [41, 41]
  });

  try {
    customerMap = L.map('customerMap').setView([-6.5, 107.2], 8);
    L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
      maxZoom: 19,
      attribution: '© OpenStreetMap contributors'
    }).addTo(customerMap);

    senderMarker = L.marker(currentSenderCoords, { icon: blueIcon, draggable: true }).addTo(customerMap);
    senderMarker.bindPopup("<b>Asal Pengiriman</b><br>Seret untuk menyesuaikan.");

    recipientMarker = L.marker(currentRecipientCoords, { icon: redIcon, draggable: true }).addTo(customerMap);
    recipientMarker.bindPopup("<b>Tujuan Pengiriman</b><br>Seret untuk menyesuaikan.");

    senderMarker.on('dragend', async () => {
      const latLng = senderMarker.getLatLng();
      currentSenderCoords = [latLng.lat, latLng.lng];
      document.getElementById('valSenderCoords').innerText = `${latLng.lat.toFixed(6)}, ${latLng.lng.toFixed(6)}`;
      updateRoute();
      const resolved = await reverseGeocode(latLng.lat, latLng.lng, 'senderAddress', 'senderCity');
      if (resolved) {
        lastSenderResolvedAddress = resolved;
      }
    });

    recipientMarker.on('dragend', async () => {
      const latLng = recipientMarker.getLatLng();
      currentRecipientCoords = [latLng.lat, latLng.lng];
      document.getElementById('valRecipientCoords').innerText = `${latLng.lat.toFixed(6)}, ${latLng.lng.toFixed(6)}`;
      updateRoute();
      const resolved = await reverseGeocode(latLng.lat, latLng.lng, 'recipientAddress', 'recipientCity');
      if (resolved) {
        lastRecipientResolvedAddress = resolved;
      }
    });

    updateRoute();
  } catch (err) {
    console.error("Gagal inisialisasi customer map:", err);
  }
}

// Draw polyline and adjust boundaries
function updateRoute() {
  if (!customerMap) return;

  if (routeLine) {
    customerMap.removeLayer(routeLine);
  }

  routeLine = L.polyline([currentSenderCoords, currentRecipientCoords], {
    color: '#3b82f6',
    weight: 4,
    dashArray: '8, 8',
    opacity: 0.85
  }).addTo(customerMap);

  const bounds = L.latLngBounds([currentSenderCoords, currentRecipientCoords]);
  customerMap.fitBounds(bounds, { padding: [40, 40] });
}

// 2. Calculate Tariff
async function calculateTariff() {
  const respBox = document.getElementById('customerResponse');
  respBox.classList.add('active');
  respBox.innerText = "Mengajukan estimasi tarif...";
  
  try {
    const payload = await getOrderFormPayload();
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
  
  try {
    const payload = await getOrderFormPayload();
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

// Tracking the last resolved address to prevent concurrent requests and rate limiting
let lastSenderResolvedAddress = "Jl. Ganesha No.10, Kel. Lebak Siliwangi, Kec. Coblong, Kota Bandung";
let lastRecipientResolvedAddress = "Jl. Jenderal Sudirman Kav. 21, Senayan, Jakarta Selatan";

async function getOrderFormPayload() {
  const senderCity = document.getElementById('senderCity').value;
  const recipientCity = document.getElementById('recipientCity').value;

  const senderAddress = document.getElementById('senderAddress').value;
  const recipientAddress = document.getElementById('recipientAddress').value;

  const defaultSenderCoords = LocationCoords[senderCity.toUpperCase()] || [-6.9175, 107.6191];
  const defaultRecipientCoords = LocationCoords[recipientCity.toUpperCase()] || [-6.2088, 106.8456];

  let senderCoords = currentSenderCoords;
  let recipientCoords = currentRecipientCoords;

  let geocodedSender = false;

  // Only geocode sender if address has been modified
  if (senderAddress !== lastSenderResolvedAddress) {
    console.log(`Sender address changed, geocoding: "${senderAddress}"`);
    senderCoords = await geocodeAddress(senderAddress, defaultSenderCoords);
    currentSenderCoords = senderCoords;
    lastSenderResolvedAddress = senderAddress;
    document.getElementById('valSenderCoords').innerText = `${senderCoords[0].toFixed(6)}, ${senderCoords[1].toFixed(6)}`;
    if (senderMarker) senderMarker.setLatLng(currentSenderCoords);
    geocodedSender = true;
  }

  // Only geocode recipient if address has been modified
  if (recipientAddress !== lastRecipientResolvedAddress) {
    // Sleep 500ms to avoid Nominatim rate limiting if we just queried the sender
    if (geocodedSender) {
      await new Promise(resolve => setTimeout(resolve, 500));
    }
    console.log(`Recipient address changed, geocoding: "${recipientAddress}"`);
    recipientCoords = await geocodeAddress(recipientAddress, defaultRecipientCoords);
    currentRecipientCoords = recipientCoords;
    lastRecipientResolvedAddress = recipientAddress;
    document.getElementById('valRecipientCoords').innerText = `${recipientCoords[0].toFixed(6)}, ${recipientCoords[1].toFixed(6)}`;
    if (recipientMarker) recipientMarker.setLatLng(currentRecipientCoords);
  }

  updateRoute();

  return {
    sender: {
      name: document.getElementById('senderName').value,
      phone: document.getElementById('senderPhone').value,
      email: document.getElementById('senderEmail').value,
      full_address: senderAddress,
      city: senderCity,
      coordinate: { latitude: senderCoords[0], longitude: senderCoords[1] }
    },
    recipient: {
      name: document.getElementById('recipientName').value,
      phone: document.getElementById('recipientPhone').value,
      email: document.getElementById('recipientEmail').value,
      full_address: recipientAddress,
      city: recipientCity,
      coordinate: { latitude: recipientCoords[0], longitude: recipientCoords[1] }
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

// 7b. Create Manifest Outbound
async function createManifest() {
  const resBox = document.getElementById('manifestResponse');
  resBox.classList.add('active');
  resBox.innerText = "Membuat manifest truk baru...";

  const payload = {
    truck_id: document.getElementById('manifestTruck').value,
    driver_name: document.getElementById('manifestDriver').value
  };

  try {
    const data = await api.apiCreateManifest(payload);
    resBox.innerText = JSON.stringify(data, null, 2);
    if (data.manifest_id) {
      ui.updateState('activeMan', data.manifest_id);
      ui.triggerAlert('manifestAlert');
    }
  } catch (e) {
    resBox.innerText = "Error Manifest Create: " + e.message;
  }
}

// 7c. Add package to Manifest
async function addToManifest() {
  const resBox = document.getElementById('manifestResponse');
  resBox.classList.add('active');
  resBox.innerText = "Memasukkan paket ke manifest truk...";

  const payload = {
    manifest_id: document.getElementById('manifestIdInput').value,
    resi: document.getElementById('manifestAwbInput').value
  };

  try {
    const data = await api.apiAddToManifest(payload);
    resBox.innerText = JSON.stringify(data, null, 2);
    ui.triggerAlert('manifestAlert');
  } catch (e) {
    resBox.innerText = "Error Add to Manifest: " + e.message;
  }
}

// 7d. Update Manifest (Depart)
async function updateManifest() {
  const resBox = document.getElementById('manifestResponse');
  resBox.classList.add('active');
  resBox.innerText = "Mengirim update status manifest (Depart)...";

  const payload = {
    manifest_id: document.getElementById('manifestIdUpdate').value
  };

  try {
    const data = await api.apiUpdateManifest(payload);
    resBox.innerText = JSON.stringify(data, null, 2);
    ui.triggerAlert('manifestAlert');
  } catch (e) {
    resBox.innerText = "Error Update Manifest: " + e.message;
  }
}

async function updateTrackingMap(historyLogs) {
  if (!historyLogs || historyLogs.length === 0) return;

  const latestStep = historyLogs[historyLogs.length - 1];
  const locCode = (latestStep.location_code || '').toUpperCase();
  
  // Resolve base coordinate
  let coords = LocationCoords[locCode] || [-6.9175, 107.6191]; // Fallback to Bandung
  let popupText = `Lokasi Paket: ${latestStep.location_code} (Status: ${latestStep.activity_code})`;

  // If status is active delivery/pickup, poll live GPS courier from MongoDB database
  if (latestStep.activity_code === 'PICKED_UP' || latestStep.activity_code === 'OUT_FOR_DELIVERY') {
    try {
      // Default courier ID used in system
      const courierLoc = await api.apiGetCourierLocation('C-001');
      if (courierLoc && courierLoc.latitude && courierLoc.longitude) {
        coords = [courierLoc.latitude, courierLoc.longitude];
        popupText = `<b>Kurir C-001 (Live GPS)</b><br>Sedang mengantarkan paket.<br>Koordinat: ${courierLoc.latitude.toFixed(4)}, ${courierLoc.longitude.toFixed(4)}`;
      }
    } catch (e) {
      console.warn("Gagal mendapatkan koordinat GPS live kurir, menggunakan fallback lokasi kota:", e);
    }
  }

  try {
    if (!mapInstance) {
      mapInstance = L.map('trackingMap').setView(coords, 13);
      L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
        maxZoom: 19,
        attribution: '© OpenStreetMap contributors'
      }).addTo(mapInstance);
    } else {
      mapInstance.setView(coords, 13);
    }

    if (markerInstance) {
      mapInstance.removeLayer(markerInstance);
    }

    markerInstance = L.marker(coords).addTo(mapInstance)
      .bindPopup(popupText)
      .openPopup();

    // Invalidate size in case Leaflet container was hidden during initialization
    setTimeout(() => {
      mapInstance.invalidateSize();
    }, 200);
  } catch (err) {
    console.error("Leaflet map initialization error:", err);
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
    updateTrackingMap(data.history);
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

function setupAddressAutocomplete(inputId, suggestionsId, cityId, onSelectCoords) {
  const input = document.getElementById(inputId);
  const suggestionsContainer = document.getElementById(suggestionsId);
  const citySelect = cityId ? document.getElementById(cityId) : null;
  
  if (!input || !suggestionsContainer) return;
  
  let debounceTimeout = null;
  
  input.addEventListener('input', () => {
    clearTimeout(debounceTimeout);
    const query = input.value.trim();
    
    if (query.length < 3) {
      suggestionsContainer.style.display = 'none';
      suggestionsContainer.innerHTML = '';
      return;
    }
    
    debounceTimeout = setTimeout(async () => {
      try {
        const response = await fetch(`https://nominatim.openstreetmap.org/search?q=${encodeURIComponent(query)}&countrycodes=id&limit=5&format=json&addressdetails=1`, {
          headers: { 'Accept': 'application/json' }
        });
        if (!response.ok) return;
        const results = await response.json();
        
        if (results.length === 0) {
          suggestionsContainer.style.display = 'none';
          suggestionsContainer.innerHTML = '';
          return;
        }
        
        suggestionsContainer.innerHTML = '';
        results.forEach(item => {
          const div = document.createElement('div');
          div.className = 'suggestion-item';
          
          const name = item.name || item.display_name.split(',')[0];
          const sub = item.display_name.replace(name + ', ', '');
          
          div.innerHTML = `
            <div class="suggestion-item-main">${name}</div>
            <div class="suggestion-item-sub">${sub}</div>
          `;
          
          div.addEventListener('click', () => {
            input.value = item.display_name;
            suggestionsContainer.style.display = 'none';
            
            const lat = parseFloat(item.lat);
            const lon = parseFloat(item.lon);
            if (onSelectCoords) {
              onSelectCoords(lat, lon);
            }
            
            if (citySelect) {
              const addr = item.address || {};
              const regionText = JSON.stringify(addr).toLowerCase();
              if (regionText.includes('jakarta')) {
                citySelect.value = 'Jakarta';
              } else if (regionText.includes('surabaya') || regionText.includes('jawa timur')) {
                citySelect.value = 'Surabaya';
              } else if (regionText.includes('bandung') || regionText.includes('jawa barat') || regionText.includes('banten')) {
                citySelect.value = 'Bandung';
              }
            }
          });
          suggestionsContainer.appendChild(div);
        });
        suggestionsContainer.style.display = 'block';
      } catch (e) {
        console.error('Error fetching autocomplete suggestions:', e);
      }
    }, 400);
  });
  
  document.addEventListener('click', (e) => {
    if (e.target !== input && e.target !== suggestionsContainer && !suggestionsContainer.contains(e.target)) {
      suggestionsContainer.style.display = 'none';
    }
  });
}

// Bind module functions to the global window scope
window.switchTab = (tabName) => {
  ui.switchTab(tabName);
  if (tabName === 'customer') {
    setTimeout(() => {
      initCustomerMap();
      if (customerMap) {
        customerMap.invalidateSize();
      }
    }, 150);
  }
};
window.calculateTariff = calculateTariff;
window.createOrder = createOrder;
window.registerDriver = registerDriver;
window.autoDispatch = autoDispatch;
window.confirmPickUp = confirmPickUp;
window.processInbound = processInbound;
window.createManifest = createManifest;
window.addToManifest = addToManifest;
window.updateManifest = updateManifest;
window.trackAwb = trackAwb;
window.sendManualScan = sendManualScan;
window.fetchMetrics = fetchMetrics;

// Auto fetch setup on start
window.addEventListener('DOMContentLoaded', () => {
  fetchMetrics();
  ui.updateVariablesUI();
  
  setupAddressAutocomplete('senderAddress', 'senderSuggestions', 'senderCity', (lat, lon) => {
    currentSenderCoords = [lat, lon];
    document.getElementById('valSenderCoords').innerText = `${lat.toFixed(6)}, ${lon.toFixed(6)}`;
    if (senderMarker) {
      senderMarker.setLatLng(currentSenderCoords);
      updateRoute();
    }
  });
  
  setupAddressAutocomplete('recipientAddress', 'recipientSuggestions', 'recipientCity', (lat, lon) => {
    currentRecipientCoords = [lat, lon];
    document.getElementById('valRecipientCoords').innerText = `${lat.toFixed(6)}, ${lon.toFixed(6)}`;
    if (recipientMarker) {
      recipientMarker.setLatLng(currentRecipientCoords);
      updateRoute();
    }
  });

  setInterval(() => fetchMetrics(false), 5000);
});
