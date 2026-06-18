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

// Calculate Tariff
async function calculateTariff() {
  const resultDiv = document.getElementById('tariffResult');
  resultDiv.style.display = 'block';
  resultDiv.className = 'info-result';
  resultDiv.innerHTML = '<em>Sedang menghitung tarif pengiriman...</em>';
  
  const senderCity = document.getElementById('senderCity').value;
  const recipientCity = document.getElementById('recipientCity').value;

  const senderAddress = document.getElementById('senderAddress').value;
  const recipientAddress = document.getElementById('recipientAddress').value;

  const defaultSenderCoords = LocationCoords[senderCity.toUpperCase()] || [-6.9175, 107.6191];
  const defaultRecipientCoords = LocationCoords[recipientCity.toUpperCase()] || [-6.2088, 106.8456];

  // Geocode optional addresses
  const [senderCoords, recipientCoords] = await Promise.all([
    geocodeAddress(senderAddress, defaultSenderCoords),
    geocodeAddress(recipientAddress, defaultRecipientCoords)
  ]);

  const payload = {
    sender: {
      name: "Pengirim Publik",
      phone: "08123456789",
      email: "pengirim@gmail.com",
      full_address: senderAddress || "Alamat Pengirim",
      city: senderCity,
      coordinate: { latitude: senderCoords[0], longitude: senderCoords[1] }
    },
    recipient: {
      name: "Penerima Publik",
      phone: "08987654321",
      email: "penerima@gmail.com",
      full_address: recipientAddress || "Alamat Penerima",
      city: recipientCity,
      coordinate: { latitude: recipientCoords[0], longitude: recipientCoords[1] }
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
    if (data.status === 'error' || data.total === undefined) {
      resultDiv.className = 'info-result error';
      resultDiv.innerText = 'Gagal menghitung tarif: Layanan tidak tersedia.';
      return;
    }

    resultDiv.className = 'info-result success';
    const formattedPrice = new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(data.total);
    resultDiv.innerHTML = `
      <div style="font-size: 1.1rem; font-weight: 700; margin-bottom: 0.25rem;">Estimasi Ongkos Kirim: ${formattedPrice}</div>
      <div style="font-size: 0.85rem; color: #64748b;">Jarak: ${data.distance !== undefined ? data.distance.toFixed(1) : 0} Km | Estimasi Waktu: ${data.eta || 'N/A'}</div>
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

function setupAddressAutocomplete(inputId, suggestionsId, cityId) {
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

// Global scope binding
window.switchPublicTab = switchPublicTab;
window.calculateTariff = calculateTariff;
window.trackAwb = trackAwb;

window.addEventListener('DOMContentLoaded', () => {
  setupAddressAutocomplete('senderAddress', 'senderSuggestions', 'senderCity');
  setupAddressAutocomplete('recipientAddress', 'recipientSuggestions', 'recipientCity');
});
