/* PAPITON Express — UI & State Control Module */

import { renderVolumeChart, renderServiceChart, renderWarehouseChart, renderNotificationChart } from './charts.js';

// State management
export let state = {
  activeResi: "RESI-EMPTY",
  activeDsp: "DSP-EMPTY",
  activeMan: "MAN-EMPTY"
};

export function updateState(key, val) {
  state[key] = val;
  updateVariablesUI();
}

// Format utilities
export function formatRupiah(num) {
  if (num >= 1000000000) return 'Rp ' + (num / 1000000000).toFixed(2) + ' M';
  if (num >= 1000000) return 'Rp ' + (num / 1000000).toFixed(1) + ' Jt';
  return new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(num);
}

export function formatHours(hours) {
  if (hours === 0) return 'N/A';
  if (hours < 24) return hours.toFixed(1) + ' Jam';
  return (hours / 24).toFixed(1) + ' Hari';
}

// DOM Variable Sync
export function updateVariablesUI() {
  document.getElementById('activeAwb').innerText = state.activeResi;
  document.getElementById('activeDispatch').innerText = state.activeDsp;
  document.getElementById('activeManifest').innerText = state.activeMan;

  // Sync inputs
  document.getElementById('dispatchAwb').value = state.activeResi !== 'RESI-EMPTY' ? state.activeResi : '';
  document.getElementById('inboundAwb').value = state.activeResi !== 'RESI-EMPTY' ? state.activeResi : '';
  document.getElementById('searchAwb').value = state.activeResi !== 'RESI-EMPTY' ? state.activeResi : '';
  
  // Manifest inputs sync
  const manIdInput = document.getElementById('manifestIdInput');
  const manIdUpdate = document.getElementById('manifestIdUpdate');
  const manAwbInput = document.getElementById('manifestAwbInput');
  if (manIdInput) manIdInput.value = state.activeMan !== 'MAN-EMPTY' ? state.activeMan : '';
  if (manIdUpdate) manIdUpdate.value = state.activeMan !== 'MAN-EMPTY' ? state.activeMan : '';
  if (manAwbInput) manAwbInput.value = state.activeResi !== 'RESI-EMPTY' ? state.activeResi : '';
}

// Tab Swapper
export function switchTab(tabName) {
  document.querySelectorAll('.nav-item').forEach(item => item.classList.remove('active'));
  document.querySelectorAll('.tab-content').forEach(content => content.classList.remove('active'));
  
  const activeItem = Array.from(document.querySelectorAll('.nav-item')).find(item => 
    item.innerText.includes(tabName === 'dashboard' ? 'Dashboard' : tabName === 'customer' ? 'Customer' : tabName === 'driver' ? 'Driver' : tabName === 'warehouse' ? 'Warehouse' : 'Track')
  );
  if (activeItem) activeItem.classList.add('active');
  
  const contentSection = document.getElementById('tab-' + tabName);
  if (contentSection) contentSection.classList.add('active');
}

// Update Dashboard Numbers and charts
export function updateDashboardDOM(data) {
  document.getElementById('kpiTotalDelivery').innerText = data.total_delivery.toLocaleString('id-ID');
  document.getElementById('kpiRevenue').innerText = formatRupiah(data.total_revenue);
  document.getElementById('kpiAvgTime').innerText = formatHours(data.avg_duration_hours);
  document.getElementById('kpiSuccessRate').innerText = data.notification_success_rate.toFixed(1) + '%';
  
  document.getElementById('kpiDriverEarnings').innerText = formatRupiah(data.driver_avg_earnings);
  document.getElementById('kpiDriverRating').innerText = data.driver_avg_rating > 0 ? data.driver_avg_rating.toFixed(2) : 'N/A';
  document.getElementById('kpiDriverCount').innerText = data.driver_active_count.toLocaleString('id-ID');

  renderVolumeChart(data.monthly_labels, data.monthly_volumes);
  renderServiceChart(data.service_types, data.service_counts);
  renderWarehouseChart(data.warehouse_ids, data.warehouse_counts);
  renderNotificationChart(data.notif_channels, data.notif_success_rates, data.notif_failure_rates);
}

// Render stepper timeline
export function renderTrackingTimeline(historyLogs) {
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

// Alerts helpers
export function triggerAlert(elementId) {
  const alertBox = document.getElementById(elementId);
  alertBox.classList.add('active');
  setTimeout(() => {
    alertBox.classList.remove('active');
  }, 4000);
}
