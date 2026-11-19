import { etagRevision, parseErrorMessage, readJSON } from './config-http.mjs';
import { renderPreviewTable } from './preview-render.mjs';

const API_URL = '/api/v1/va/feature-configs/stats';
const CONNECTIONS_URL = '/api/v1/va/data-connections';

const statusEl = document.getElementById('admin-feature-stats-status');
const form = document.getElementById('feature-stats-form');
const previewEl = document.getElementById('feature-stats-preview');
const previewBtn = document.getElementById('feature-stats-preview-btn');
const deleteBtn = document.getElementById('feature-stats-delete-btn');
const fieldsHost = document.getElementById('feature-stats-fields');

let revision = '0';
let connectionsRevision = 0;
let catalog = [];
let draft = { version: 1, revision: 0, pilot: null, career: null };

function showStatus(message, isError) {
  if (!statusEl) return;
  statusEl.textContent = message;
  statusEl.hidden = !message;
  statusEl.classList.toggle('panel-message--error', Boolean(isError));
}

function renderFieldToggles() {
  if (!fieldsHost) return;
  fieldsHost.replaceChildren();
  const pilotBox = document.createElement('fieldset');
  pilotBox.className = 'stack';
  pilotBox.innerHTML = '<legend class="section__title">Pilot section</legend>';
  const matchPilot = document.createElement('select');
  matchPilot.className = 'select';
  matchPilot.id = 'feature-stats-pilot-match';
  matchPilot.innerHTML = '<option value="callsign">Callsign</option><option value="ifcUsername">IFC username</option>';
  pilotBox.append(matchPilot);
  for (const f of catalog.filter((c) => c.key !== 'lastFlownRoute')) {
    const row = document.createElement('label');
    row.className = 'field__label';
    const cb = document.createElement('input');
    cb.type = 'checkbox';
    cb.dataset.section = 'pilot';
    cb.value = f.key;
    cb.checked = Boolean(draft.pilot?.fields?.includes(f.key));
    row.append(cb, document.createTextNode(` ${f.label || f.key}`));
    pilotBox.append(row);
  }
  const careerBox = document.createElement('fieldset');
  careerBox.className = 'stack';
  careerBox.innerHTML = '<legend class="section__title">Career section</legend>';
  const matchCareer = document.createElement('select');
  matchCareer.className = 'select';
  matchCareer.id = 'feature-stats-career-match';
  matchCareer.innerHTML = '<option value="callsign">Callsign</option><option value="ifcUsername">IFC username</option>';
  careerBox.append(matchCareer);
  for (const f of catalog) {
    const row = document.createElement('label');
    row.className = 'field__label';
    const cb = document.createElement('input');
    cb.type = 'checkbox';
    cb.dataset.section = 'career';
    cb.value = f.key;
    cb.checked = Boolean(draft.career?.fields?.includes(f.key));
    row.append(cb, document.createTextNode(` ${f.label || f.key}`));
    careerBox.append(row);
  }
  fieldsHost.append(pilotBox, careerBox);
  if (draft.pilot?.matchBy) matchPilot.value = draft.pilot.matchBy;
  if (draft.career?.matchBy) matchCareer.value = draft.career.matchBy;
  matchPilot.addEventListener('change', syncTestIdentityFields);
  matchCareer.addEventListener('change', syncTestIdentityFields);
  syncTestIdentityFields();
}

function syncTestIdentityFields() {
  const pilotMatch = document.getElementById('feature-stats-pilot-match')?.value;
  const careerMatch = document.getElementById('feature-stats-career-match')?.value;
  const needsCs = pilotMatch === 'callsign' || careerMatch === 'callsign';
  const needsIfc = pilotMatch === 'ifcUsername' || careerMatch === 'ifcUsername';
  const csWrap = document.getElementById('feature-stats-callsign')?.closest('.field');
  const ifcWrap = document.getElementById('feature-stats-ifc-username')?.closest('.field');
  if (csWrap) csWrap.hidden = !needsCs;
  if (ifcWrap) ifcWrap.hidden = !needsIfc;
}

function collectDraft() {
  const pilotFields = [...(fieldsHost?.querySelectorAll('input[data-section="pilot"]:checked') || [])].map((el) => el.value);
  const careerFields = [...(fieldsHost?.querySelectorAll('input[data-section="career"]:checked') || [])].map((el) => el.value);
  const pilotMatch = document.getElementById('feature-stats-pilot-match')?.value || 'callsign';
  const careerMatch = document.getElementById('feature-stats-career-match')?.value || 'callsign';
  draft.pilot = pilotFields.length ? { matchBy: pilotMatch, fields: pilotFields } : null;
  draft.career = careerFields.length ? { matchBy: careerMatch, fields: careerFields } : null;
  draft.enabled = Boolean(draft.pilot || draft.career);
  draft.revision = Number(revision);
  draft.connectionsRevision = connectionsRevision;
  return draft;
}

async function load() {
  const [connRes, featRes] = await Promise.all([
    fetch(CONNECTIONS_URL, { credentials: 'same-origin' }),
    fetch(API_URL, { credentials: 'same-origin' }),
  ]);
  const connBody = await readJSON(connRes);
  const body = await readJSON(featRes);
  if (featRes.status === 503) {
    showStatus('Feature mappings require data connections and DATA_ENCRYPTION_KEY.', true);
    form?.setAttribute('hidden', '');
    return;
  }
  connectionsRevision = connBody.data?.revision ?? 0;
  const missing = ['pilots'].filter((e) => !connBody.data?.mappings?.[e]?.tableId);
  if (missing.length) {
    showStatus(`Configure ${missing.join(', ')} under Data connections first.`, true);
  }
  if (!featRes.ok) {
    showStatus(parseErrorMessage(body, 'Failed to load stats mapping.'), true);
    return;
  }
  const data = body.data || {};
  revision = String(data.revision ?? 0);
  catalog = data.catalog || [];
  draft = data.stats || { version: 1, revision: 0, pilot: null, career: null };
  renderFieldToggles();
  showStatus(data.configured ? 'Configuration loaded.' : 'Choose pilot and career fields, then save.', false);
}

form?.addEventListener('submit', async (event) => {
  event.preventDefault();
  const payload = collectDraft();
  const res = await fetch(API_URL, {
    method: 'PUT',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json', 'If-Match': revision },
    body: JSON.stringify(payload),
  });
  const body = await readJSON(res);
  if (res.status === 412) {
    showStatus('Someone else saved changes. Reload to continue.', true);
    return;
  }
  if (!res.ok) {
    showStatus(parseErrorMessage(body, 'Could not save stats mapping.'), true);
    return;
  }
  await load();
  showStatus('Saved.', false);
});

function previewIdentityHint(d) {
  const needs = new Set();
  if (d.pilot?.matchBy) needs.add(d.pilot.matchBy);
  if (d.career?.matchBy) needs.add(d.career.matchBy);
  if (needs.has('ifcUsername') && needs.has('callsign')) {
    return 'Enter both a test callsign and IFC username because Pilot and Career use different lookup keys.';
  }
  if (needs.has('ifcUsername')) {
    return 'Enter a test IFC username to match your configured lookup column.';
  }
  return 'Enter a test callsign to match your configured lookup column.';
}

function previewIdentityReady(d, callsign, ifcUsername) {
  const needs = new Set();
  if (d.pilot?.matchBy) needs.add(d.pilot.matchBy);
  if (d.career?.matchBy) needs.add(d.career.matchBy);
  if (needs.has('callsign') && !callsign) return false;
  if (needs.has('ifcUsername') && !ifcUsername) return false;
  return needs.size > 0;
}

previewBtn?.addEventListener('click', async () => {
  const stats = collectDraft();
  const callsign = String(document.getElementById('feature-stats-callsign')?.value || '').trim();
  const ifcUsername = String(document.getElementById('feature-stats-ifc-username')?.value || '').trim();
  if (!previewIdentityReady(stats, callsign, ifcUsername)) {
    showStatus(previewIdentityHint(stats), true);
    return;
  }
  const payload = { callsign, ifcUsername, stats };
  const res = await fetch(`${API_URL}/preview`, {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  const body = await readJSON(res);
  if (!res.ok) {
    showStatus(parseErrorMessage(body, 'Preview failed.'), true);
    return;
  }
  if (previewEl) {
    previewEl.hidden = false;
    previewEl.replaceChildren();
    renderPreviewTable(previewEl, body.data);
  }
});

deleteBtn?.addEventListener('click', async () => {
  const res = await fetch(API_URL, {
    method: 'DELETE',
    credentials: 'same-origin',
    headers: { 'If-Match': revision },
  });
  const body = await readJSON(res);
  if (!res.ok) {
    showStatus(parseErrorMessage(body, 'Could not remove configuration.'), true);
    return;
  }
  await load();
  showStatus('Removed.', false);
});

void load();