/* PAPITON Express — Public Page Coordinator Module */

import * as api from './api.js';

let mapInstance = null;
let markerInstance = null;

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

// Toggle Tabs
export function switchPublicTab(tabName) {
  document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
  document.querySelectorAll('.public-tab-content').forEach(content => content.classList.remove('active'));
  
  const activeBtn = document.querySelector(`.tab-btn[data-tab="${tabName}"]`);
  if (activeBtn) activeBtn.classList.add('active');
  
  const activeContent = document.getElementById(`tab-${tabName}`);
  if (activeContent) activeContent.classList.add('active');
}

// Calculate Tariff
async function calculateTariff() {
  const resultDiv = document.getElementById('tariffResult');
  resultDiv.style.display = 'block';
  resultDiv.className = 'info-result';
  resultDiv.innerHTML = '<em>Sedang menghitung tarif pengiriman...</em>';
  
  const payload = {
    sender: {
      city: document.getElementById('senderCity').value,
      coordinate: { latitude: -6.8915, longitude: 107.6106 }
    },
    recipient: {
      city: document.getElementById('recipientCity').value,
      coordinate: { latitude: -6.2088, longitude: 106.8456 }
    },
    package: {
      length: 10, width: 10, height: 10,
      actual_weight: parseFloat(document.getElementById('pkgWeight').value || 1.0)
    },
    service_type: document.getElementById('pkgService').value,
    has_insurance: document.getElementById('hasInsurance').checked,
    has_packing: false
  };

  try {
    const data = await api.apiCalculateTariff(payload);
    if (data.status === 'error' || !data.tarif_total) {
      resultDiv.className = 'info-result error';
      resultDiv.innerText = 'Gagal menghitung tarif: Layanan tidak tersedia.';
      return;
    }

    resultDiv.className = 'info-result success';
    const formattedPrice = new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(data.tarif_total);
    resultDiv.innerHTML = `
      <div style="font-size: 1.1rem; font-weight: 700; margin-bottom: 0.25rem;">Estimasi Ongkos Kirim: ${formattedPrice}</div>
      <div style="font-size: 0.85rem; color: #64748b;">Jarak: ${data.distance_km || 0} Km | Estimasi Waktu: ${data.eta || 'N/A'}</div>
    `;
  } catch (e) {
    resultDiv.className = 'info-result error';
    resultDiv.innerText = 'Koneksi API Gagal: ' + e.message;
  }
}

// Render Timeline
function renderTrackingTimeline(historyLogs) {
  const timeline = document.getElementById('trackingTimeline');
  timeline.innerHTML = '';
  
  historyLogs.forEach((step, index) => {
    const stepDiv = document.createElement('div');
    stepDiv.className = `timeline-step ${index === historyLogs.length - 1 ? 'active' : 'completed'}`;
    
    const timeFormatted = new Date(step.timestamp).toLocaleString('id-ID');
    
    stepDiv.innerHTML = `
      <div class="timeline-circle"></div>
      <div class="timeline-header-info">
        <span class="timeline-status">${step.activity_code}</span>
        <span class="timeline-time">${timeFormatted}</span>
      </div>
      <div class="timeline-desc">Lokasi Gudang: ${step.location_code}</div>
    `;
    timeline.appendChild(stepDiv);
  });
}

// Update Map
async function updateTrackingMap(historyLogs) {
  if (!historyLogs || historyLogs.length === 0) return;

  const latestStep = historyLogs[historyLogs.length - 1];
  const locCode = (latestStep.location_code || '').toUpperCase();
  
  let coords = LocationCoords[locCode] || [-6.9175, 107.6191];
  let popupText = `Lokasi Paket: ${latestStep.location_code} (Status: ${latestStep.activity_code})`;

  if (latestStep.activity_code === 'PICKED_UP' || latestStep.activity_code === 'OUT_FOR_DELIVERY') {
    try {
      const courierLoc = await api.apiGetCourierLocation('C-001');
      if (courierLoc && courierLoc.latitude && courierLoc.longitude) {
        coords = [courierLoc.latitude, courierLoc.longitude];
        popupText = `<b>Kurir C-001 (Live GPS)</b><br>Sedang membawa paket.<br>Koordinat: ${courierLoc.latitude.toFixed(4)}, ${courierLoc.longitude.toFixed(4)}`;
      }
    } catch (e) {
      console.warn("Gagal polling koordinat live kurir:", e);
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

    setTimeout(() => {
      mapInstance.invalidateSize();
    }, 200);
  } catch (err) {
    console.error("Leaflet error:", err);
  }
}

// Track AWB
async function trackAwb() {
  const resi = document.getElementById('searchAwb').value;
  const resultArea = document.getElementById('trackingResultArea');
  const errArea = document.getElementById('trackingErrorArea');
  
  resultArea.style.display = 'none';
  errArea.style.display = 'none';

  if (!resi) return;

  try {
    const data = await api.apiGetTrackingHistory(resi);
    if (!data.history || data.history.length === 0) {
      errArea.style.display = 'block';
      errArea.innerText = 'Nomor resi tidak ditemukan atau belum diperbarui.';
      return;
    }
    
    resultArea.style.display = 'block';
    renderTrackingTimeline(data.history);
    updateTrackingMap(data.history);
  } catch (e) {
    errArea.style.display = 'block';
    errArea.innerText = 'Koneksi gagal atau resi tidak valid.';
  }
}

// Global scope binding
window.switchPublicTab = switchPublicTab;
window.calculateTariff = calculateTariff;
window.trackAwb = trackAwb;
