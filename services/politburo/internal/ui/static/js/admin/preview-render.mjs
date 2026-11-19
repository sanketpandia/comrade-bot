export function formatDurationSeconds(sec) {
  const n = Number(sec);
  if (!Number.isFinite(n)) return String(sec ?? '');
  const h = Math.floor(n / 3600);
  const m = Math.floor((n % 3600) / 60);
  return `${h}h ${String(m).padStart(2, '0')}m`;
}

export function renderPreviewTable(container, data) {
  if (!container) return;
  container.replaceChildren();
  const sections = data?.sections || [];
  if (sections.length === 0 && data?.filing?.length) {
    const table = document.createElement('table');
    table.className = 'data-table';
    const head = document.createElement('thead');
    head.innerHTML = '<tr><th>Field</th><th>Sample</th><th>Value</th></tr>';
    table.append(head);
    const body = document.createElement('tbody');
    for (const row of data.filing) {
      const tr = document.createElement('tr');
      tr.innerHTML = `<td>${row.label || row.key}</td><td>${row.destinationColumn || row.destination || ''}</td><td>${row.display || ''}</td>`;
      body.append(tr);
    }
    table.append(body);
    container.append(table);
    return;
  }
  for (const sec of sections) {
    const heading = document.createElement('h4');
    heading.className = 'section__title';
    heading.textContent = sec.name;
    container.append(heading);
    const table = document.createElement('table');
    table.className = 'data-table';
    table.innerHTML = '<thead><tr><th>Field</th><th>Display</th></tr></thead>';
    const body = document.createElement('tbody');
    for (const f of sec.fields || []) {
      const tr = document.createElement('tr');
      const display = f.display || (f.value != null ? String(f.value) : '—');
      tr.innerHTML = `<td>${f.label || f.key}</td><td>${display}${f.warning ? ` (${f.warning})` : ''}</td>`;
      body.append(tr);
    }
    table.append(body);
    container.append(table);
  }
  for (const w of data?.warnings || []) {
    const p = document.createElement('p');
    p.className = 'panel-message panel-message--error';
    p.textContent = w;
    container.append(p);
  }
}