import { FlightMap } from '/static/js/map/flight-map.mjs';

const POLL_MS = 60_000;
const FILTER_DEBOUNCE_MS = 300;
const DEFAULT_SERVER = 'casual';
const PAGE_LENGTH = 5000;

const form = document.getElementById('maps-filters');
const serverSelect = document.getElementById('maps-server');
const countEl = document.getElementById('maps-count');
const cachedEl = document.getElementById('maps-cached');
const errorEl = document.getElementById('maps-error');

const map = new FlightMap('leaflet-map', { center: [20, 0], zoom: 2 });
let fittedOnce = false;
let pollTimer = 0;
let debounceTimer = 0;

function readFiltersFromForm() {
  const data = new FormData(form);
  return {
    serverId: String(data.get('serverId') || '').trim(),
    callSign: String(data.get('callSign') || '').trim(),
    userName: String(data.get('userName') || '').trim(),
    pilotState: data.getAll('pilotState').map(String),
    showCallsigns: data.get('showCallsigns') != null,
  };
}

function applyFiltersToForm(filters) {
  if (filters.serverId) {
    serverSelect.value = filters.serverId;
  }
  form.elements.callSign.value = filters.callSign || '';
  form.elements.userName.value = filters.userName || '';
  for (const input of form.querySelectorAll('input[name="pilotState"]')) {
    input.checked = filters.pilotState.includes(input.value);
  }
  form.elements.showCallsigns.checked = filters.showCallsigns !== false;
}

function readFiltersFromURL() {
  const params = new URLSearchParams(window.location.search);
  const showCallsignsParam = params.get('showCallsigns');
  return {
    serverId: (params.get('serverId') || '').trim(),
    callSign: (params.get('callSign') || '').trim(),
    userName: (params.get('userName') || '').trim(),
    pilotState: params.getAll('pilotState'),
    showCallsigns: showCallsignsParam === null ? true : showCallsignsParam !== '0',
  };
}

function writeFiltersToURL(filters) {
  const params = new URLSearchParams();
  if (filters.serverId) params.set('serverId', filters.serverId);
  for (const state of filters.pilotState) {
    params.append('pilotState', state);
  }
  if (filters.callSign) params.set('callSign', filters.callSign);
  if (filters.userName) params.set('userName', filters.userName);
  if (!filters.showCallsigns) params.set('showCallsigns', '0');
  const query = params.toString();
  const next = query ? `${window.location.pathname}?${query}` : window.location.pathname;
  window.history.replaceState(null, '', next);
}

async function fetchJSON(url) {
  const response = await fetch(url, { credentials: 'same-origin' });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    const message = body?.error?.message || `Request failed (${response.status})`;
    const error = new Error(message);
    error.status = response.status;
    throw error;
  }
  return body;
}

function setError(message) {
  if (!message) {
    errorEl.hidden = true;
    errorEl.textContent = '';
    return;
  }
  errorEl.hidden = false;
  errorEl.textContent = message;
}

function setStatus(count, lastCached) {
  countEl.textContent = `${count} ${count === 1 ? 'flight' : 'flights'}`;
  if (!lastCached) {
    cachedEl.textContent = '';
    return;
  }
  const at = new Date(lastCached);
  cachedEl.textContent = Number.isNaN(at.getTime()) ? '' : `Cached ${at.toISOString()}`;
}

async function loadSessions() {
  const body = await fetchJSON('/api/v1/game/sessions/active');
  const sessions = Array.isArray(body?.data?.result) ? body.data.result : [];
  const urlFilters = readFiltersFromURL();
  const preferred = urlFilters.serverId || DEFAULT_SERVER;
  serverSelect.replaceChildren();
  for (const session of sessions) {
    const value = session.normalizedName || '';
    if (!value) continue;
    const option = document.createElement('option');
    option.value = value;
    option.textContent = value.replaceAll('_', ' ');
    serverSelect.append(option);
  }
  if (serverSelect.options.length === 0) {
    const option = document.createElement('option');
    option.value = DEFAULT_SERVER;
    option.textContent = DEFAULT_SERVER;
    serverSelect.append(option);
  }
  if ([...serverSelect.options].some((option) => option.value === preferred)) {
    serverSelect.value = preferred;
  } else {
    serverSelect.selectedIndex = 0;
  }
  applyFiltersToForm({
    ...urlFilters,
    serverId: serverSelect.value,
  });
}

function flightsURL(filters, pageNumber) {
  const params = new URLSearchParams();
  params.set('pageNumber', String(pageNumber));
  params.set('pageLength', String(PAGE_LENGTH));
  for (const state of filters.pilotState) {
    params.append('pilotState', state);
  }
  if (filters.callSign) params.set('callSign', filters.callSign);
  if (filters.userName) params.set('userName', filters.userName);
  const query = params.toString();
  return `/api/v1/game/flights/active/${encodeURIComponent(filters.serverId)}?${query}`;
}

async function fetchAllFlights(filters) {
  const all = [];
  let pageNumber = 1;
  let totalLength = 0;
  let lastCached = '';
  for (;;) {
    const body = await fetchJSON(flightsURL(filters, pageNumber));
    const page = Array.isArray(body?.data?.result) ? body.data.result : [];
    all.push(...page);
    totalLength = body?.data?.pagination?.totalLength ?? all.length;
    lastCached = body?.data?.meta?.lastCached ?? lastCached;
    if (all.length >= totalLength || page.length === 0) {
      return { flights: all, totalLength, lastCached };
    }
    pageNumber += 1;
  }
}

async function loadFlights() {
  const filters = readFiltersFromForm();
  if (!filters.serverId) {
    map.addFlights([], { fitBounds: false });
    setStatus(0, '');
    setError('Select a server');
    return;
  }
  writeFiltersToURL(filters);
  try {
    const { flights, totalLength, lastCached } = await fetchAllFlights(filters);
    map.addFlights(flights, {
      fitBounds: !fittedOnce && flights.length > 0,
      showCallsigns: filters.showCallsigns,
    });
    if (flights.length > 0) fittedOnce = true;
    setStatus(totalLength, lastCached);
    setError('');
  } catch (err) {
    map.addFlights([], { fitBounds: false });
    setStatus(0, '');
    setError(err.message || 'Failed to load flights');
  }
}

function scheduleLoad() {
  window.clearTimeout(debounceTimer);
  debounceTimer = window.setTimeout(() => {
    loadFlights();
  }, FILTER_DEBOUNCE_MS);
}

form.addEventListener('change', () => {
  loadFlights();
});
form.addEventListener('input', (event) => {
  if (event.target instanceof HTMLInputElement && event.target.type === 'search') {
    scheduleLoad();
  }
});
form.addEventListener('submit', (event) => {
  event.preventDefault();
  loadFlights();
});

async function start() {
  try {
    await loadSessions();
    setError('');
  } catch (err) {
    applyFiltersToForm(readFiltersFromURL());
    if (!serverSelect.value) {
      const option = document.createElement('option');
      option.value = DEFAULT_SERVER;
      option.textContent = DEFAULT_SERVER;
      serverSelect.append(option);
      serverSelect.value = DEFAULT_SERVER;
    }
    setError(err.message || 'Failed to load servers');
  }
  await loadFlights();
  pollTimer = window.setInterval(loadFlights, POLL_MS);
}

window.addEventListener('beforeunload', () => {
  window.clearInterval(pollTimer);
  window.clearTimeout(debounceTimer);
});

start();
