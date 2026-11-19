export async function fetchSchema(connectionId, baseSetupId) {
  const res = await fetch(
    `/api/v1/va/data-connections/connections/${encodeURIComponent(connectionId)}/bases/${encodeURIComponent(baseSetupId)}/schema`,
    { credentials: 'same-origin' },
  );
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(body?.error?.message || 'Schema request failed');
  return body.data?.tables || [];
}

export function fillTableSelect(select, tables, selectedId) {
  if (!select) return;
  select.replaceChildren();
  const empty = document.createElement('option');
  empty.value = '';
  empty.textContent = 'Choose a table';
  select.append(empty);
  for (const tbl of tables) {
    const opt = document.createElement('option');
    opt.value = tbl.id;
    opt.textContent = tbl.name || tbl.id;
    if (tbl.id === selectedId) opt.selected = true;
    select.append(opt);
  }
}

export function compatibleFieldsForKey(table, logicalKey) {
  const fields = table?.fields || [];
  return fields.filter((f) => (f.compatibleKeys || []).includes(logicalKey));
}

export function fillFieldSelect(select, fields, selectedId) {
  if (!select) return;
  select.replaceChildren();
  const empty = document.createElement('option');
  empty.value = '';
  empty.textContent = '—';
  select.append(empty);
  for (const f of fields) {
    const opt = document.createElement('option');
    opt.value = f.id;
    opt.textContent = `${f.name} (${f.type})`;
    if (f.id === selectedId) opt.selected = true;
    select.append(opt);
  }
}