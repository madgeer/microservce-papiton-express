/* PAPITON Express — Chart Visualization Module */

let chartVolumeInstance = null;
let chartServiceInstance = null;
let chartWarehouseInstance = null;
let chartNotificationInstance = null;

export function renderVolumeChart(labels, values) {
  const ctx = document.getElementById('chartVolume').getContext('2d');
  if (chartVolumeInstance) chartVolumeInstance.destroy();
  
  chartVolumeInstance = new Chart(ctx, {
    type: 'bar',
    data: {
      labels: labels,
      datasets: [{
        data: values,
        backgroundColor: '#2563eb',
        borderRadius: 4
      }]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: { legend: { display: false } },
      scales: {
        y: { grid: { color: 'rgba(255,255,255,0.06)' }, border: { dash: [5, 5] } },
        x: { grid: { display: false } }
      }
    }
  });
}

export function renderServiceChart(labels, values) {
  const ctx = document.getElementById('chartService').getContext('2d');
  if (chartServiceInstance) chartServiceInstance.destroy();

  chartServiceInstance = new Chart(ctx, {
    type: 'doughnut',
    data: {
      labels: labels,
      datasets: [{
        data: values,
        backgroundColor: ['#2563eb', '#10b981', '#8b5cf6'],
        borderWidth: 0
      }]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      plugins: {
        legend: { position: 'right', labels: { boxWidth: 12, padding: 20 } }
      },
      cutout: '65%'
    }
  });
}

export function renderWarehouseChart(labels, values) {
  const ctx = document.getElementById('chartWarehouse').getContext('2d');
  if (chartWarehouseInstance) chartWarehouseInstance.destroy();

  chartWarehouseInstance = new Chart(ctx, {
    type: 'bar',
    data: {
      labels: labels,
      datasets: [{
        data: values,
        backgroundColor: '#64748b',
        borderRadius: 4
      }]
    },
    options: {
      indexAxis: 'y',
      responsive: true,
      maintainAspectRatio: false,
      plugins: { legend: { display: false } },
      scales: {
        x: { grid: { color: 'rgba(255,255,255,0.06)' }, border: { dash: [5, 5] } },
        y: { grid: { display: false } }
      }
    }
  });
}

export function renderNotificationChart(labels, successRates, failureRates) {
  const ctx = document.getElementById('chartNotification').getContext('2d');
  if (chartNotificationInstance) chartNotificationInstance.destroy();

  chartNotificationInstance = new Chart(ctx, {
    type: 'bar',
    data: {
      labels: labels,
      datasets: [
        {
          label: 'Success (%)',
          data: successRates,
          backgroundColor: '#10b981',
          borderRadius: 4
        },
        {
          label: 'Failure (%)',
          data: failureRates,
          backgroundColor: '#ef4444',
          borderRadius: 4
        }
      ]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        x: { stacked: true, grid: { display: false } },
        y: { stacked: true, max: 100, grid: { color: 'rgba(255,255,255,0.06)' } }
      },
      plugins: {
        legend: { position: 'top', labels: { boxWidth: 12 } }
      }
    }
  });
}
