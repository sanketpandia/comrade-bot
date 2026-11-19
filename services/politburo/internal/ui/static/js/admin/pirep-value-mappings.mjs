import { parseErrorMessage, readJSON, etagRevision } from './config-http.mjs';

const API = '/api/v1/va/pirep-value-mappings';

const statusEl = document.getElementById('pirep-value-mappings-status');
const defaultsForm = document.getElementById('pirep-value-mappings-defaults-form');
const listHost = document.getElementById('pirep-value-mappings-list');
const previewHost = document.getElementById('pirep-value-mappings-preview');
const importInput = document.getElementById('pirep-value-mappings-import');
const tabAircraft = document.getElementById('pirep-vm-tab-aircraft');
const tabAirline = document.getElementById('pirep-vm-tab-airline');
const searchInput = document.getElementById('pirep-value-mappings-search');

let revision = '0';
let kind = 'aircraft';
let draft = emptyDocument();
let pendingImport = null;

function emptyDocument() {
  return {
    version: 1,
    defaults: { aircraft: null, airline: null },
    AircraftMappings: {},
    AirlineMappings: {},
  };
}

function showStatus(message, isError) {
  if (!statusEl) return;
  statusEl.textContent = message;
  statusEl.hidden = !message;
  statusEl.classList.toggle('panel-message--error', Boolean(isError));
}

function activeMap() {
  return kind === 'airline' ? draft.AirlineMappings : draft.AircraftMappings;
}

function renderDefaults() {
  if (!defaultsForm) return;
  const aircraft = defaultsForm.querySelector('[name="default_aircraft"]');
  const airline = defaultsForm.querySelector('[name="default_airline"]');
  if (aircraft) aircraft.value = draft.defaults?.aircraft ?? '';
  if (airline) airline.value = draft.defaults?.airline ?? '';
}

async function loadSaved() {
  const res = await fetch(API, { credentials: 'same-origin' });
  const body = await readJSON(res);
  if (!res.ok) {
    showStatus(parseErrorMessage(body, 'Failed to load value mappings.'), true);
    return;
  }
  const data = body.data || {};
  revision = String(data.revision ?? 0);
  draft = data.document || emptyDocument();
  renderDefaults();
  await refreshList();
  showStatus(data.configured ? 'Value mappings loaded.' : 'Configure defaults and rules, then save.', false);
}

async function refreshList() {
  if (!listHost) return;
  const q = encodeURIComponent(searchInput?.value?.trim() || '');
  const res = await fetch(`${API}/catalog?kind=${kind}&search=${q}&limit=100`, { credentials: 'same-origin' });
  const body = await readJSON(res);
  if (!res.ok) {
    listHost.textContent = parseErrorMessage(body, 'Failed to load catalog.');
    return;
  }
  const items = body.data?.items || [];
  listHost.replaceChildren();
  const map = activeMap();
  for (const item of items) {
    const row = document.createElement('div');
    row.className = 'stack';
    const label = document.createElement('label');
    label.className = 'field__label';
    const input = document.createElement('input');
    input.className = 'input';
    input.type = 'text';
    input.value = map[item.name] ?? item.filingValue ?? '';
    input.dataset.name = item.name;
    input.placeholder = item.status === 'needs_value' ? 'Needs value' : '';
    label.append(document.createTextNode(`${item.name} (${item.status}) `), input);
    row.append(label);
    listHost.append(row);
  }
}

function collectDraftFromUI() {
  const aircraft = defaultsForm?.querySelector('[name="default_aircraft"]')?.value?.trim() || null;
  const airline = defaultsForm?.querySelector('[name="default_airline"]')?.value?.trim() || null;
  draft.defaults = { aircraft: aircraft || null, airline: airline || null };
  if (listHost) {
    for (const input of listHost.querySelectorAll('input[data-name]')) {
      const name = input.dataset.name;
      const value = input.value.trim();
      const map = activeMap();
      if (value) map[name] = value;
      else delete map[name];
    }
  }
  return draft;
}

defaultsForm?.addEventListener('submit', async (event) => {
  event.preventDefault();
  collectDraftFromUI();
  const res = await fetch(API, {
    method: 'PUT',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json', 'If-Match': revision },
    body: JSON.stringify(draft),
  });
  const body = await readJSON(res);
  if (res.status === 412) {
    showStatus('Someone else saved first. Reload the page and review your draft.', true);
    return;
  }
  if (!res.ok) {
    showStatus(parseErrorMessage(body, 'Save failed.'), true);
    return;
  }
  revision = String(body.data?.revision ?? revision);
  showStatus('Saved aircraft and airline values.', false);
});

document.getElementById('pirep-value-mappings-save-rules')?.addEventListener('click', () => {
  defaultsForm?.requestSubmit();
});

tabAircraft?.addEventListener('click', async () => {
  kind = 'aircraft';
  tabAircraft.setAttribute('aria-current', 'true');
  tabAirline?.removeAttribute('aria-current');
  await refreshList();
});

tabAirline?.addEventListener('click', async () => {
  kind = 'airline';
  tabAirline.setAttribute('aria-current', 'true');
  tabAircraft?.removeAttribute('aria-current');
  await refreshList();
});

searchInput?.addEventListener('change', () => refreshList());

document.getElementById('pirep-value-mappings-export')?.addEventListener('click', () => {
  window.location.href = `${API}/export`;
});

document.getElementById('pirep-value-mappings-catalog-csv')?.addEventListener('click', () => {
  window.location.href = `${API}/catalog.csv`;
});

importInput?.addEventListener('change', async () => {
  const file = importInput.files?.[0];
  if (!file) return;
  const text = await file.text();
  let doc;
  try {
    doc = JSON.parse(text);
  } catch {
    showStatus('Invalid JSON file.', true);
    return;
  }
  const res = await fetch(`${API}/preview`, {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ document: doc }),
  });
  const body = await readJSON(res);
  if (!res.ok) {
    showStatus(parseErrorMessage(body, 'Import preview failed.'), true);
    return;
  }
  pendingImport = body.data?.normalized;
  if (previewHost) {
    previewHost.hidden = false;
    previewHost.textContent = `Import preview: ${JSON.stringify(body.data?.diff ?? {}, null, 2)}`;
  }
  showStatus('Review the import preview, then apply.', false);
});

document.getElementById('pirep-value-mappings-apply-import')?.addEventListener('click', async () => {
  if (!pendingImport) {
    showStatus('Choose a JSON file first.', true);
    return;
  }
  const res = await fetch(API, {
    method: 'PUT',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json', 'If-Match': revision },
    body: JSON.stringify(pendingImport),
  });
  const body = await readJSON(res);
  if (!res.ok) {
    showStatus(parseErrorMessage(body, 'Apply import failed.'), true);
    return;
  }
  pendingImport = null;
  draft = body.data?.document || pendingImport;
  revision = String(body.data?.revision ?? revision);
  renderDefaults();
  await refreshList();
  showStatus('Import applied.', false);
});

loadSaved();