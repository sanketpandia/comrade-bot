import { FlightMap } from '/static/js/map/flight-map.mjs';
import { initMapsChrome, onChromeTransitionEnd } from '/static/js/maps/maps-chrome.mjs';
import { initMultiselect } from '/static/js/ui/multiselect.mjs';

const DEFAULT_SERVER = 'casual';
const MARKER_COLOR = '#d5c4a1';

const form = document.getElementById('maps-filters');
const serverSelect = document.getElementById('maps-server');
const shell = document.getElementById('maps-shell');
const detailPanel = document.getElementById('maps-detail-panel');
const detailBody = document.getElementById('maps-detail-body');
const detailTabs = document.getElementById('maps-detail-tabs');
const detailClose = document.getElementById('maps-detail-close');
const toggleCount = document.getElementById('maps-filters-toggle-count');

const map = new FlightMap('leaflet-map', { center: [20, 0], zoom: 2 });
let fittedOnce = false;
let selectedFlightId = '';

function detailEnabled() {
  return shell?.dataset.detailEnabled !== '0';
}

function readShowCallsigns() {
  const box = document.getElementById('maps-show-callsigns');
  return box ? box.checked : true;
}

function syncShowCallsignsField() {
  const box = document.getElementById('maps-show-callsigns');
  if (!box || !form) return;
  if (!box.checked) {
    const hidden = document.createElement('input');
    hidden.type = 'hidden';
    hidden.name = 'showCallsigns';
    hidden.value = '0';
    hidden.dataset.ephemeral = '1';
    form.querySelectorAll('input[data-ephemeral="1"]').forEach((n) => n.remove());
    form.appendChild(hidden);
  } else {
    form.querySelectorAll('input[data-ephemeral="1"]').forEach((n) => n.remove());
  }
}

function readTracksFromDOM() {
  const node = document.getElementById('maps-tracks');
  if (!node?.textContent) return [];
  try {
    const parsed = JSON.parse(node.textContent);
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

export function getMapPadding() {
  const filtersOpen = shell && !shell.classList.contains('maps-shell--filters-collapsed');
  const detailOpen = shell?.classList.contains('maps-shell--detail-open');
  const top = 72;
  let left = filtersOpen ? 380 : 56;
  let right = detailOpen ? 360 : 56;
  let bottom = 56;
  if (shell?.classList.contains('maps-shell--map-only')) {
    left = 56;
    right = 56;
    bottom = 56;
  }
  if (window.matchMedia('(max-width: 768px)').matches && detailOpen) {
    bottom = Math.floor(window.innerHeight * 0.42) + 24;
  }
  return { paddingTopLeft: [left, top], paddingBottomRight: [right, bottom] };
}

function applyTracksFromPartial() {
  syncShowCallsignsField();
  const countEl = document.getElementById('maps-count');
  if (toggleCount && countEl) {
    toggleCount.textContent = countEl.textContent.trim();
  }
  const flights = readTracksFromDOM();
  const padding = getMapPadding();
  map.addFlights(flights, {
    fitBounds: !fittedOnce && flights.length > 0,
    showCallsigns: readShowCallsigns(),
    markerColor: MARKER_COLOR,
    fitBoundsPadding: padding,
  });
  if (flights.length > 0) fittedOnce = true;
  if (selectedFlightId) {
    map.focusFlight(selectedFlightId, { markerColor: MARKER_COLOR });
  }
}

function openDetailPanel() {
  if (!detailPanel || !shell) return;
  detailPanel.hidden = false;
  shell.classList.add('maps-shell--detail-open');
  onChromeTransitionEnd(() => map.invalidateSize());
}

function closeDetailPanel() {
  if (!detailPanel || !shell) return;
  shell.classList.remove('maps-shell--detail-open');
  detailPanel.hidden = true;
  selectedFlightId = '';
  onChromeTransitionEnd(() => map.invalidateSize());
}

function setActiveTab(tab) {
  if (!detailTabs) return;
  for (const link of detailTabs.querySelectorAll('.tabs__tab')) {
    const active = link.dataset.tab === tab;
    link.classList.toggle('tabs__tab--active', active);
    link.setAttribute('aria-selected', active ? 'true' : 'false');
  }
}

async function loadFlightDetail(flightId, tab = 'flight') {
  if (!flightId || !detailEnabled()) return;
  selectedFlightId = flightId;
  openDetailPanel();
  setActiveTab(tab);
  const url = tab === 'flight'
    ? `/maps/flights/active/flights/${encodeURIComponent(flightId)}`
    : `/maps/flights/active/flights/${encodeURIComponent(flightId)}?tab=${encodeURIComponent(tab)}`;
  try {
    const response = await fetch(url, { credentials: 'same-origin' });
    const html = await response.text();
    if (detailBody) detailBody.innerHTML = html;
    if (response.ok) {
      map.focusFlight(flightId, { markerColor: MARKER_COLOR });
    }
  } catch {
    if (detailBody) {
      detailBody.innerHTML = '<p class="panel-message panel-message--error">Failed to load flight details.</p>';
    }
  }
}

function bindDetailTabs() {
  detailTabs?.addEventListener('click', (event) => {
    const link = event.target.closest('.tabs__tab');
    if (!link || !selectedFlightId) return;
    event.preventDefault();
    loadFlightDetail(selectedFlightId, link.dataset.tab || 'flight');
  });
  detailClose?.addEventListener('click', () => closeDetailPanel());
}

function ensureServerOption(serverId) {
  if (!serverSelect || !serverId) return;
  for (const option of serverSelect.options) {
    if (option.value === serverId) {
      serverSelect.value = serverId;
      return;
    }
  }
  const option = document.createElement('option');
  option.value = serverId;
  option.textContent = serverId;
  option.selected = true;
  serverSelect.appendChild(option);
}

function readServerFromURL() {
  return new URLSearchParams(window.location.search).get('serverId')?.trim() || '';
}

function whenServersReady() {
  return new Promise((resolve) => {
    if (!serverSelect) {
      resolve();
      return;
    }
    const done = () => {
      serverSelect.removeEventListener('htmx:afterSettle', done);
      resolve();
    };
    serverSelect.addEventListener('htmx:afterSettle', done);
    window.setTimeout(resolve, 2000);
  });
}

document.body.addEventListener('htmx:afterSettle', (event) => {
  if (event.target === form || event.detail?.target?.id === 'maps-flights-swap') {
    applyTracksFromPartial();
  }
});

form?.addEventListener('htmx:beforeRequest', (event) => {
  if (event.target !== form) return;
  syncShowCallsignsField();
});
form?.addEventListener('submit', (event) => {
  event.preventDefault();
});

function start() {
  initMapsChrome({ shell, map, getMapPadding });
  initMultiselect(document.getElementById('maps-pilot-state'));
  bindDetailTabs();
  map.onFlightClick((flight) => {
    const flightId = flight?.flightId ?? flight?.flight_id ?? '';
    if (!flightId) return;
    loadFlightDetail(flightId, 'flight');
  });
  whenServersReady().then(() => {
    const fromURL = readServerFromURL();
    ensureServerOption(fromURL || DEFAULT_SERVER);
    if (!serverSelect.value) {
      serverSelect.value = DEFAULT_SERVER;
    }
    syncShowCallsignsField();
  });
}

start();

export { map, closeDetailPanel, loadFlightDetail };
