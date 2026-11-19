const API_URL = '/api/v1/va/data-connections';

const statusEl = document.getElementById('admin-data-connections-status');
const listEl = document.getElementById('data-connections-list');
const mappingsEl = document.getElementById('data-connections-mappings');
const createForm = document.getElementById('data-connections-create-form');

let revision = '0';

function showStatus(message, isError) {
  if (!statusEl) return;
  statusEl.textContent = message;
  statusEl.hidden = !message;
  statusEl.classList.toggle('panel-message--error', Boolean(isError));
}

function errorMessage(body, fallback) {
  return body?.error?.message || fallback;
}

async function load() {
  const res = await fetch(API_URL, { credentials: 'same-origin' });
  const body = await res.json().catch(() => ({}));
  if (res.status === 503) {
    showStatus('Data connections are not configured on this server. Set DATA_CONNECTIONS_ENCRYPTION_KEY.', true);
    if (createForm) createForm.hidden = true;
    return;
  }
  if (!res.ok) {
    showStatus(errorMessage(body, 'Failed to load configuration.'), true);
    return;
  }
  const data = body.data || {};
  revision = String(data.revision ?? 0);
  renderConnections(data.connections || []);
  renderMappings(data.mappings || {});
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
  for (const entity of ['pilots', 'routes', 'pireps']) {
    const dt = document.createElement('dt');
    dt.textContent = entity;
    const dd = document.createElement('dd');
    const mapping = mappings[entity];
    dd.textContent = mapping?.tableId ? `Table ${mapping.tableId}` : 'Not configured';
    mappingsEl.append(dt, dd);
  }
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
  const body = await res.json().catch(() => ({}));
  if (res.status === 412) {
    showStatus('Someone else saved changes. Reload to continue.', true);
    return;
  }
  if (!res.ok) {
    showStatus(errorMessage(body, 'Could not create connection.'), true);
    return;
  }
  createForm.reset();
  const tokenInput = document.getElementById('dc-token');
  if (tokenInput) tokenInput.value = '';
  await load();
  showStatus('Connection saved.', false);
});

void load();