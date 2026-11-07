const table = document.getElementById('admin-members-table');
const tbody = document.getElementById('admin-members-body');
const statusEl = document.getElementById('admin-members-status');

const canEditRoles = table?.dataset.canEditRoles === 'true';
const canRemoveStaff = table?.dataset.canRemoveStaff === 'true';
const membersMode = table?.dataset.membersMode === 'deleted' ? 'deleted' : 'active';
const isDeletedView = membersMode === 'deleted';

const roles = ['proletariat', 'bourgeoisie', 'administrator'];

function showStatus(message, isError) {
  if (!statusEl) {
    return;
  }
  statusEl.textContent = message;
  statusEl.hidden = !message;
  statusEl.classList.toggle('panel-message--error', Boolean(isError));
}

function formatDate(value) {
  if (!value) {
    return '—';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return '—';
  }
  return date.toLocaleString();
}

function displayUser(row) {
  return row.username || row.userId || '—';
}

function canRemoveRow(row) {
  if (isDeletedView) {
    return false;
  }
  if (row.role === 'administrator') {
    return false;
  }
  if (row.role === 'bourgeoisie' && !canRemoveStaff) {
    return false;
  }
  return true;
}

function roleSelect(row) {
  const select = document.createElement('select');
  select.className = 'select';
  select.dataset.userId = row.userId;
  for (const role of roles) {
    const option = document.createElement('option');
    option.value = role;
    option.textContent = role;
    if (role === row.role) {
      option.selected = true;
    }
    select.appendChild(option);
  }
  select.addEventListener('change', () => void patchRole(row.userId, select));
  return select;
}

async function patchRole(userId, select) {
  const role = select.value;
  const prior = select.dataset.priorRole || role;
  select.disabled = true;
  try {
    const response = await fetch(`/api/v1/va/members/${encodeURIComponent(userId)}`, {
      method: 'PATCH',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ role }),
    });
    if (!response.ok) {
      const body = await response.json().catch(() => ({}));
      const message = body?.error?.message || `Request failed (${response.status})`;
      showStatus(message, true);
      select.value = prior;
      return;
    }
    select.dataset.priorRole = role;
    showStatus('Role updated.', false);
    await loadMembers();
  } catch {
    showStatus('Could not update role.', true);
    select.value = prior;
  } finally {
    select.disabled = false;
  }
}

async function removeMember(userId) {
  try {
    const response = await fetch(`/api/v1/va/members/${encodeURIComponent(userId)}`, {
      method: 'DELETE',
      credentials: 'same-origin',
    });
    if (!response.ok) {
      const body = await response.json().catch(() => ({}));
      const message = body?.error?.message || `Request failed (${response.status})`;
      showStatus(message, true);
      return;
    }
    showStatus('Member removed.', false);
    await loadMembers();
  } catch {
    showStatus('Could not remove member.', true);
  }
}

function renderRows(rows) {
  if (!tbody) {
    return;
  }
  tbody.replaceChildren();
  if (!rows.length) {
    const tr = document.createElement('tr');
    const td = document.createElement('td');
    td.colSpan = 7;
    td.textContent = isDeletedView ? 'No removed members.' : 'No members yet.';
    tr.appendChild(td);
    tbody.appendChild(tr);
    return;
  }

  for (const row of rows) {
    const tr = document.createElement('tr');

    const cells = [
      displayUser(row),
      row.ifCommunityId || '—',
      null,
      row.callsign || '—',
      formatDate(row.joinedAt),
      formatDate(row.updatedAt),
      null,
    ];

    for (let i = 0; i < cells.length; i += 1) {
      const td = document.createElement('td');
      if (i === 2) {
        if (canEditRoles && !isDeletedView) {
          td.appendChild(roleSelect(row));
        } else {
          td.textContent = row.role || '—';
        }
      } else if (i === 6) {
        if (canRemoveRow(row)) {
          const button = document.createElement('button');
          button.type = 'button';
          button.className = 'btn';
          button.textContent = 'Remove';
          button.addEventListener('click', () => void removeMember(row.userId));
          td.appendChild(button);
        } else {
          td.textContent = '—';
        }
      } else {
        td.textContent = cells[i];
      }
      tr.appendChild(td);
    }
    tbody.appendChild(tr);
  }
}

async function loadMembers() {
  const status = isDeletedView ? 'deleted' : 'active';
  try {
    const response = await fetch(`/api/v1/va/members?status=${encodeURIComponent(status)}`, {
      credentials: 'same-origin',
    });
    const body = await response.json().catch(() => ({}));
    if (!response.ok) {
      const message = body?.error?.message || `Request failed (${response.status})`;
      showStatus(message, true);
      renderRows([]);
      return;
    }
    showStatus('', false);
    renderRows(Array.isArray(body?.data) ? body.data : []);
  } catch {
    showStatus('Could not load members.', true);
    renderRows([]);
  }
}

if (table && tbody) {
  loadMembers();
}
