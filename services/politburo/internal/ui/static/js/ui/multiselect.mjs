export function initMultiselect(container) {
  if (!container) return;

  const trigger = container.querySelector('.multiselect__trigger');
  const panel = container.querySelector('.multiselect__panel');
  const labelEl = container.querySelector('.multiselect__value');
  const inputs = container.querySelectorAll('input[type="checkbox"]');
  if (!trigger || !panel || !labelEl || inputs.length === 0) return;

  const emptyLabel = container.dataset.empty || 'Any selected';
  const allLabel = container.dataset.all || 'All selected';

  function optionLabel(input) {
    const option = input.closest('.multiselect__option');
    if (option) {
      const text = option.textContent.replace(/\s+/g, ' ').trim();
      if (text) return text;
    }
    return input.value;
  }

  function updateLabel() {
    const selected = [...inputs].filter((input) => input.checked).map(optionLabel);
    if (selected.length === 0) {
      labelEl.textContent = emptyLabel;
    } else if (selected.length === 1) {
      labelEl.textContent = selected[0];
    } else if (selected.length === inputs.length) {
      labelEl.textContent = allLabel;
    } else {
      labelEl.textContent = `${selected.length} selected`;
    }
  }

  function setOpen(open) {
    panel.hidden = !open;
    trigger.setAttribute('aria-expanded', open ? 'true' : 'false');
  }

  trigger.addEventListener('click', (event) => {
    event.stopPropagation();
    setOpen(panel.hidden);
  });

  panel.addEventListener('click', (event) => {
    event.stopPropagation();
  });

  for (const input of inputs) {
    input.addEventListener('change', updateLabel);
  }

  document.addEventListener('click', () => setOpen(false));
  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') setOpen(false);
  });

  updateLabel();
}
