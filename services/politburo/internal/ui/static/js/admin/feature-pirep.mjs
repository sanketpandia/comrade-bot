import { parseErrorMessage, readJSON } from './config-http.mjs';
import { renderPreviewTable } from './preview-render.mjs';

const API_URL = '/api/v1/va/feature-configs/pirep';
const CONNECTIONS_URL = '/api/v1/va/data-connections';

const statusEl = document.getElementById('admin-feature-pirep-status');
const form = document.getElementById('feature-pirep-form');
const previewEl = document.getElementById('feature-pirep-preview');
const previewBtn = document.getElementById('feature-pirep-preview-btn');
const deleteBtn = document.getElementById('feature-pirep-delete-btn');
const fieldsHost = document.getElementById('feature-pirep-fields');

let revision = '0';
let connectionsRevision = 0;
let catalog = [];
let draft = { version: 1, revision: 0, fields: [] };

function showStatus(message, isError) {
  if (!statusEl) return;
  statusEl.textContent = message;
  statusEl.hidden = !message;
  statusEl.classList.toggle('panel-message--error', Boolean(isError));
}

function renderFieldToggles() {
  if (!fieldsHost) return;
  fieldsHost.replaceChildren();
  for (const f of catalog) {
    if (f.usage === 'read') continue;
    const row = document.createElement('label');
    row.className = 'field__label';
    const cb = document.createElement('input');
    cb.type = 'checkbox';
    cb.value = f.key;
    cb.checked = draft.fields?.includes(f.key);
    row.append(cb, document.createTextNode(` ${f.label || f.key}`));
    fieldsHost.append(row);
  }
}

function collectDraft() {
  draft.fields = [...(fieldsHost?.querySelectorAll('input[type="checkbox"]:checked') || [])].map((el) => el.value);
  draft.enabled = draft.fields.length > 0;
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
    showStatus('Feature mappings require DATA_ENCRYPTION_KEY.', true);
    form?.setAttribute('hidden', '');
    return;
  }
  connectionsRevision = connBody.data?.revision ?? 0;
  if (!connBody.data?.mappings?.pireps?.tableId) {
    showStatus('Configure PIREPs mapping under Data connections first.', true);
  }
  if (!featRes.ok) {
    showStatus(parseErrorMessage(body, 'Failed to load PIREP mapping.'), true);
    return;
  }
  const data = body.data || {};
  revision = String(data.revision ?? 0);
  catalog = data.catalog || [];
  draft = data.pirep || { version: 1, revision: 0, fields: [] };
  renderFieldToggles();
  showStatus(data.configured ? 'Configuration loaded.' : 'Select filing fields, then save.', false);
}

form?.addEventListener('submit', async (event) => {
  event.preventDefault();
  const res = await fetch(API_URL, {
    method: 'PUT',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json', 'If-Match': revision },
    body: JSON.stringify(collectDraft()),
  });
  const body = await readJSON(res);
  if (res.status === 412) {
    showStatus('Someone else saved changes. Reload to continue.', true);
    return;
  }
  if (!res.ok) {
    showStatus(parseErrorMessage(body, 'Could not save PIREP mapping.'), true);
    return;
  }
  await load();
  showStatus('Saved.', false);
});

previewBtn?.addEventListener('click', async () => {
  const res = await fetch(`${API_URL}/preview`, {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ pirep: collectDraft() }),
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