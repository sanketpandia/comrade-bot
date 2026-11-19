import { etagRevision, parseErrorMessage, readJSON } from './config-http.mjs';
import { fetchSchema, fillFieldSelect, fillTableSelect, compatibleFieldsForKey } from './schema-picker.mjs';

const API_URL = '/api/v1/va/data-connections';
const ENTITIES = ['pilots', 'careerPilots', 'routes', 'pireps'];

const statusEl = document.getElementById('admin-data-connections-status');
const listEl = document.getElementById('data-connections-list');
const mappingsEl = document.getElementById('data-connections-mappings');
const mappingForm = document.getElementById('data-connections-mapping-form');
const createForm = document.getElementById('data-connections-create-form');

let revision = '0';
let state = { connections: [], mappings: {}, catalog: [] };
let schemaTables = [];

function showStatus(message, isError) {
  if (!statusEl) return;
  statusEl.textContent = message;
  statusEl.hidden = !message;
  statusEl.classList.toggle('panel-message--error', Boolean(isError));
}

async function load() {
  const res = await fetch(API_URL, { credentials: 'same-origin' });
  const body = await readJSON(res);
  if (res.status === 503) {
    showStatus('Data connections are not configured on this server. Set DATA_ENCRYPTION_KEY.', true);
    if (createForm) createForm.hidden = true;
    return;
  }
  if (!res.ok) {
    showStatus(parseErrorMessage(body, 'Failed to load configuration.'), true);
    return;
  }
  const data = body.data || {};
  revision = String(data.revision ?? 0);
  state.connections = data.connections || [];
  state.mappings = data.mappings || {};
  state.catalog = data.catalog || [];
  renderConnections(state.connections);
  renderMappings(state.mappings);
  renderMappingForm();
  showStatus('', false);
}

function renderConnections(connections) {
  if (!listEl) return;
  listEl.replaceChildren();
  if (connections.length === 0) {
    const li = document.createElement('li');
    li.textContent = 'No connections yet.';
    listEl.append(li);
    return;
  }
  for (const conn of connections) {
    const li = document.createElement('li');
    const bases = (conn.bases || []).map((b) => `${b.name} (${b.baseId})`).join(', ');
    li.textContent = `${conn.name} · ${conn.provider}${conn.hasCredential ? '' : ' · missing token'}${bases ? ` · ${bases}` : ''}`;
    listEl.append(li);
  }
}

function renderMappings(mappings) {
  if (!mappingsEl) return;
  mappingsEl.replaceChildren();
  for (const entity of ENTITIES) {
    const dt = document.createElement('dt');
    dt.textContent = entity;
    const dd = document.createElement('dd');
    const mapping = mappings[entity];
    dd.textContent = mapping?.tableId ? `Table ${mapping.tableId} (${Object.keys(mapping.fields || {}).length} fields)` : 'Not configured';
    mappingsEl.append(dt, dd);
  }
}

function catalogForEntity(entity) {
  return (state.catalog || []).find((c) => c.entity === entity)?.fields || [];
}

function renderMappingForm() {
  if (!mappingForm) return;
  mappingForm.replaceChildren();
  const entityField = document.createElement('div');
  entityField.className = 'field';
  entityField.innerHTML = '<label class="field__label" for="dc-map-entity">Entity</label>';
  const entitySelect = document.createElement('select');
  entitySelect.className = 'select';
  entitySelect.id = 'dc-map-entity';
  for (const e of ENTITIES) {
    const opt = document.createElement('option');
    opt.value = e;
    opt.textContent = e;
    entitySelect.append(opt);
  }
  entityField.append(entitySelect);

  const connField = document.createElement('div');
  connField.className = 'field';
  connField.innerHTML = '<label class="field__label" for="dc-map-connection">Connection</label>';
  const connSelect = document.createElement('select');
  connSelect.className = 'select';
  connSelect.id = 'dc-map-connection';
  for (const c of state.connections) {
    const opt = document.createElement('option');
    opt.value = c.id;
    opt.textContent = c.name;
    connSelect.append(opt);
  }
  connField.append(connSelect);

  const baseField = document.createElement('div');
  baseField.className = 'field';
  baseField.innerHTML = '<label class="field__label" for="dc-map-base">Base</label>';
  const baseSelect = document.createElement('select');
  baseSelect.className = 'select';
  baseSelect.id = 'dc-map-base';
  baseField.append(baseSelect);

  const tableField = document.createElement('div');
  tableField.className = 'field';
  tableField.innerHTML = '<label class="field__label" for="dc-map-table">Table</label>';
  const tableSelect = document.createElement('select');
  tableSelect.className = 'select';
  tableSelect.id = 'dc-map-table';
  tableField.append(tableSelect);

  const fieldsWrap = document.createElement('div');
  fieldsWrap.id = 'dc-map-fields';
  fieldsWrap.className = 'stack';

  const actions = document.createElement('div');
  actions.className = 'stack';
  const loadSchemaBtn = document.createElement('button');
  loadSchemaBtn.type = 'button';
  loadSchemaBtn.className = 'btn btn--secondary';
  loadSchemaBtn.textContent = 'Load schema';
  const previewBtn = document.createElement('button');
  previewBtn.type = 'button';
  previewBtn.className = 'btn btn--secondary';
  previewBtn.textContent = 'Preview rows';
  const saveBtn = document.createElement('button');
  saveBtn.type = 'submit';
  saveBtn.className = 'btn btn--primary';
  saveBtn.textContent = 'Save mapping';
  actions.append(loadSchemaBtn, previewBtn, saveBtn);

  const previewOut = document.createElement('pre');
  previewOut.id = 'dc-map-preview';
  previewOut.hidden = true;
  previewOut.className = 'panel-message';

  mappingForm.append(entityField, connField, baseField, tableField, fieldsWrap, actions, previewOut);

  function refreshBases() {
    const conn = state.connections.find((c) => c.id === connSelect.value);
    baseSelect.replaceChildren();
    for (const b of conn?.bases || []) {
      const opt = document.createElement('option');
      opt.value = b.id;
      opt.textContent = b.name;
      baseSelect.append(opt);
    }
  }

  function loadExistingMapping() {
    const entity = entitySelect.value;
    const m = state.mappings[entity];
    if (!m) return;
    connSelect.value = m.connectionId || connSelect.value;
    refreshBases();
    baseSelect.value = m.baseSetupId || baseSelect.value;
  }

  async function loadSchema() {
    try {
      schemaTables = await fetchSchema(connSelect.value, baseSelect.value);
      const entity = entitySelect.value;
      const tableId = state.mappings[entity]?.tableId || '';
      fillTableSelect(tableSelect, schemaTables, tableId);
      renderFieldPickers();
      showStatus('Schema loaded.', false);
    } catch (e) {
      showStatus(e.message, true);
    }
  }

  function renderFieldPickers() {
    const entity = entitySelect.value;
    const table = schemaTables.find((t) => t.id === tableSelect.value);
    const catalog = catalogForEntity(entity);
    const existing = state.mappings[entity]?.fields || {};
    fieldsWrap.replaceChildren();
    for (const cf of catalog) {
      const row = document.createElement('div');
      row.className = 'field';
      const label = document.createElement('label');
      label.className = 'field__label';
      label.textContent = cf.label || cf.key;
      const select = document.createElement('select');
      select.className = 'select';
      select.dataset.logicalKey = cf.key;
      fillFieldSelect(select, compatibleFieldsForKey(table, cf.key), existing[cf.key]);
      row.append(label, select);
      fieldsWrap.append(row);
    }
    tableSelect.onchange = renderFieldPickers;
  }

  async function previewMapping() {
    const mapping = collectMapping();
    const res = await fetch(`${API_URL}/mappings/${encodeURIComponent(entitySelect.value)}/preview`, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(mapping),
    });
    const body = await readJSON(res);
    if (!res.ok) {
      showStatus(parseErrorMessage(body, 'Preview failed.'), true);
      return;
    }
    previewOut.hidden = false;
    previewOut.textContent = JSON.stringify(body.data?.records || [], null, 2);
  }

  function collectMapping() {
    const fields = {};
    for (const sel of fieldsWrap.querySelectorAll('select[data-logical-key]')) {
      if (sel.value) fields[sel.dataset.logicalKey] = sel.value;
    }
    return {
      connectionId: connSelect.value,
      baseSetupId: baseSelect.value,
      tableId: tableSelect.value,
      fields,
    };
  }

  connSelect.addEventListener('change', refreshBases);
  entitySelect.addEventListener('change', () => {
    loadExistingMapping();
    renderFieldPickers();
  });
  loadSchemaBtn.addEventListener('click', () => void loadSchema());
  previewBtn.addEventListener('click', () => void previewMapping());
  mappingForm.addEventListener('submit', async (event) => {
    event.preventDefault();
    const res = await fetch(`${API_URL}/mappings/${encodeURIComponent(entitySelect.value)}`, {
      method: 'PUT',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json', 'If-Match': revision },
      body: JSON.stringify(collectMapping()),
    });
    const body = await readJSON(res);
    if (res.status === 412) {
      showStatus('Someone else saved changes. Reload to continue.', true);
      return;
    }
    if (!res.ok) {
      showStatus(parseErrorMessage(body, 'Could not save mapping.'), true);
      return;
    }
    await load();
    showStatus('Mapping saved.', false);
  });

  refreshBases();
  loadExistingMapping();
}

createForm?.addEventListener('submit', async (event) => {
  event.preventDefault();
  const data = new FormData(createForm);
  const payload = {
    name: String(data.get('name') || '').trim(),
    provider: 'airtable',
    token: String(data.get('token') || '').trim(),
    baseName: String(data.get('baseName') || '').trim(),
    baseId: String(data.get('baseId') || '').trim(),
  };
  const res = await fetch(`${API_URL}/connections`, {
    method: 'POST',
    credentials: 'same-origin',
    headers: {
      'Content-Type': 'application/json',
      'If-Match': revision,
    },
    body: JSON.stringify(payload),
  });
  const body = await readJSON(res);
  if (res.status === 412) {
    showStatus('Someone else saved changes. Reload to continue.', true);
    return;
  }
  if (!res.ok) {
    showStatus(parseErrorMessage(body, 'Could not create connection.'), true);
    return;
  }
  createForm.reset();
  await load();
  showStatus('Connection saved.', false);
});

void load();