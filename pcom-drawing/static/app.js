'use strict';
(() => {
  const $ = (id) => document.getElementById(id);
  const storageKey = 'pcom-drawing-studio-v1';
  const selectFields = ['role', 'user_serial', 'maker', 'filter_current', 'io_connector', 'maint_connector', 'user_connector', 'board_connector', 'belt_connector', 'board_pin_order'];
  const lengthFields = ['io', 'maint', 'user', 'plc', 'belt', 'board'];
  let catalog;
  let currentConfig;
  let latestModel;
  let previewTimer;
  let previewAbort;
  let sequence = 0;
  let pending = false;
  let valid = false;
  let toastTimer;
  const clone = (value) => JSON.parse(JSON.stringify(value));
  const fieldLabels = {bit_code:'비트 구분', role:'역할', function_code:'기능',user_serial:'USER 통신',maker:'적용 장비사',filter_current:'필터 전류',io_connector:'I/O 커넥터',maint_connector:'MAINT 커넥터',user_connector:'USER 커넥터',board_connector:'BOARD 커넥터',belt_connector:'BELT 커넥터',board_pin_order:'BOARD 전원 핀 순서','lengths.io':'I/O 길이','lengths.maint':'MAINT 길이','lengths.user':'USER 길이','lengths.plc':'PLC 길이','lengths.belt':'BELT 측 길이','lengths.board':'BOARD 측 길이',revision:'개정',date:'작성일',notes:'도면 비고'};

  function toast(message) {
    clearTimeout(toastTimer);
    $('toast').textContent = message;
    $('toast').classList.add('visible');
    toastTimer = setTimeout(() => $('toast').classList.remove('visible'), 3200);
  }
  function status(kind, message) {
    $('readiness').className = `readiness ${kind || ''}`;
    $('readiness').querySelector('b').textContent = message;
  }
  function updateButtons() {
    for (const id of ['export-pptx', 'export-svg', 'save-button', 'copy-code']) $(id).disabled = !valid || pending;
  }
  function setBusy(value) {
    pending = value;
    updateButtons();
  }
  function clearErrors() {
    $('form-errors').hidden = true;
    $('form-errors').replaceChildren();
    $('spec-form').querySelectorAll('.invalid').forEach((el) => { el.classList.remove('invalid'); el.removeAttribute('aria-invalid'); });
  }
  function errorMessage(error) {
    if (error.detail && Array.isArray(error.detail)) return error.detail.map((item) => typeof item === 'string' ? item : `${fieldLabels[item.field] || item.field || '사양'}: ${item.message || item.msg || '값을 확인하세요.'}`).join('\n');
    if (typeof error.detail === 'string') return error.detail;
    return error.message || '요청을 처리하지 못했습니다.';
  }
  function showErrors(error) {
    clearErrors();
    const banner = $('form-errors');
    const title = document.createElement('p');
    title.textContent = '사양을 확인해주세요. 아래 항목을 수정하면 도면이 다시 생성됩니다.';
    banner.append(title);
    const list = document.createElement('ul');
    const details = Array.isArray(error.detail) ? error.detail : [{message: errorMessage(error)}];
    for (const detail of details) {
      const li = document.createElement('li');
      const field = detail.field;
      li.textContent = typeof detail === 'string' ? detail : `${field ? `${fieldLabels[field] || field}: ` : ''}${detail.message || detail.msg || '값을 확인하세요.'}`;
      list.append(li);
      if (field) {
        const el = [...$('spec-form').elements].find((input) => input.name === field);
        if (el) { el.classList.add('invalid'); el.setAttribute('aria-invalid', 'true'); }
      }
    }
    banner.append(list);
    banner.hidden = false;
  }
  async function api(path, data, signal) {
    const res = await fetch(path, {method: data === undefined ? 'GET' : 'POST', headers: data === undefined ? {} : {'Content-Type': 'application/json'}, body: data === undefined ? undefined : JSON.stringify(data), signal});
    if (!res.ok) {
      let problem;
      try { problem = await res.json(); } catch { problem = {message: `요청 실패 (HTTP ${res.status})`}; }
      throw problem;
    }
    return res;
  }
  function fillOptions(field) {
    const options = catalog.options[field] || [];
    const select = $(field);
    select.replaceChildren();
    for (const item of options) {
      const option = document.createElement('option');
      option.value = item.value;
      option.textContent = item.label;
      if (item.description) option.title = item.description;
      select.append(option);
    }
  }
  function ensureOption(select, value) {
    if (![...select.options].some((opt) => opt.value === value)) {
      const option = document.createElement('option');
      option.value = value;
      option.textContent = value === '' ? '직접 입력 사양' : `${value} · 기존 도면 코드`;
      select.append(option);
    }
    select.value = value;
  }
  function selectedPreset() { return catalog.presets.find((item) => item.id === currentConfig?.preset_id); }
  function updatePresetInfo() {
    const preset = selectedPreset();
    $('preset-description').textContent = preset?.source ? `${preset.source.filename} · ${preset.source.slide}쪽` : '기본 사양 또는 기존 도면에서 시작하세요. 선택한 값을 자유롭게 수정할 수 있습니다.';
  }
  function applyConfig(input, {save = false} = {}) {
    currentConfig = {...clone(catalog.defaults), ...clone(input), lengths: {...catalog.defaults.lengths, ...(input.lengths || {})}};
    if (currentConfig.bit_code !== 'A8') { /* keep unsupported code so validation can explain; never silently convert it */ }
    for (const field of selectFields) ensureOption($(field), String(currentConfig[field] ?? ''));
    for (const field of lengthFields) $(`length-${field}`).value = currentConfig.lengths[field] ?? '';
    for (const field of ['revision', 'date', 'notes']) $(field).value = currentConfig[field] || '';
    const presetID = currentConfig.preset_id || '';
    ensureOption($('preset'), presetID);
    updatePresetInfo();
    if (save) saveLocal();
  }
  function readConfig() {
    const next = clone(currentConfig);
    for (const field of selectFields) next[field] = $(field).value;
    for (const field of lengthFields) {
      const raw = $(`length-${field}`).value;
      next.lengths[field] = raw === '' ? null : Number(raw);
    }
    for (const field of ['revision', 'date', 'notes']) next[field] = $(field).value;
    next.preset_id = $('preset').value;
    return next;
  }
  function saveLocal() {
    try {
      localStorage.setItem(storageKey, JSON.stringify(currentConfig));
      $('saved-status').textContent = '이 기기에 현재 사양이 저장되었습니다.';
    } catch {
      $('saved-status').textContent = '기기 저장 공간을 사용할 수 없습니다.';
    }
  }
  function safeSVG(markup) {
    const doc = new DOMParser().parseFromString(markup, 'image/svg+xml');
    const root = doc.documentElement;
    if (root.nodeName !== 'svg' || doc.querySelector('parsererror')) throw new Error('도면 SVG를 표시하지 못했습니다.');
    root.querySelectorAll('script,foreignObject,iframe,object,embed').forEach((el) => el.remove());
    for (const el of [root, ...root.querySelectorAll('*')]) {
      for (const attribute of [...el.attributes]) {
        if (/^on/i.test(attribute.name) || ((/^(href|xlink:href)$/i.test(attribute.name)) && !attribute.value.startsWith('#'))) el.removeAttribute(attribute.name);
      }
    }
    root.setAttribute('role', 'img');
    root.setAttribute('aria-label', '현재 선택 사양으로 생성한 pCOM 8비트 케이블 연결 도면');
    return document.importNode(root, true);
  }
  function makeCell(text, className) {
    const td = document.createElement('td');
    td.textContent = text ?? '—';
    if (className) td.className = className;
    return td;
  }
  function renderTables(model) {
    const pins = document.createDocumentFragment();
    for (const item of model.pins || []) {
      const row = document.createElement('tr');
      row.append(makeCell(item.connector),makeCell(item.pin,'numeric'),makeCell(item.signal,'signal-cell'),makeCell(item.color,'dim'),makeCell(item.target,'target-cell'));
      pins.append(row);
    }
    $('pins-body').replaceChildren(pins);
    $('pins-count').textContent = (model.pins || []).length;
    const bom = document.createDocumentFragment();
    for (const item of model.bom || []) {
      const row = document.createElement('tr');
      row.append(makeCell(item.ref,'dim'),makeCell(item.category),makeCell(item.model,'model-cell'),makeCell(item.maker,'dim'),makeCell(item.quantity,'numeric'),makeCell(item.note,'dim'));
      bom.append(row);
    }
    $('bom-body').replaceChildren(bom);
    $('bom-count').textContent = (model.bom || []).length;
  }
  function renderSummary(model) {
    const fragment = document.createDocumentFragment();
    for (const item of model.summary || []) {
      const box = document.createElement('div'); box.className = 'summary-item';
      const label = document.createElement('span'); label.textContent = item.label;
      const value = document.createElement('strong'); value.textContent = item.value;
      box.append(label, value); fragment.append(box);
    }
    $('summary-strip').replaceChildren(fragment);
    $('summary-strip').hidden = !model.summary?.length;
  }
  function renderNotes(model, extraWarnings = []) {
    const warnings = [...(model.warnings || []), ...extraWarnings];
    $('review-section').hidden = warnings.length === 0;
    $('review-count').textContent = `${warnings.length}개`;
    $('warning-list').replaceChildren(...warnings.map((item) => {
      const li = document.createElement('li'); li.textContent = typeof item === 'string' ? item : item.message; return li;
    }));
    const preset = selectedPreset();
    const source = model.source || preset?.source;
    $('source-section').hidden = !source;
    $('source-name').textContent = source ? `${source.filename} · ${source.slide}쪽${source.revision ? ` · ${source.revision}` : ''}` : '';
    $('source-code').textContent = source?.drawing_number || '';
    const notes = preset?.notes || [];
    $('source-notes').replaceChildren(...notes.map((text) => { const li = document.createElement('li'); li.textContent = text; return li; }));
  }
  function renderModel(model) {
    latestModel = model;
    $('drawing-title').textContent = `pCOM ${model.role_label || '8BIT'} 케이블 도면`;
    $('profile-label').textContent = model.profile_label || '8비트 케이블 사양';
    $('drawing-code').textContent = model.code;
    $('svg-container').replaceChildren(safeSVG(model.svg));
    renderTables(model);
    renderSummary(model);
    renderNotes(model);
  }
  async function previewNow() {
    clearTimeout(previewTimer);
    previewAbort?.abort();
    previewAbort = new AbortController();
    const requestID = ++sequence;
    const sentConfig = readConfig();
    currentConfig = sentConfig;
    valid = false;
    setBusy(true);
    status('busy', '도면 생성 중');
    clearErrors();
    try {
      const res = await api('/api/preview', sentConfig, previewAbort.signal);
      const model = await res.json();
      if (requestID !== sequence) return;
      currentConfig = clone(model.config || sentConfig);
      renderModel(model);
      valid = true;
      status('', '미리보기 준비됨');
      saveLocal();
    } catch (error) {
      if (error.name === 'AbortError' || requestID !== sequence) return;
      valid = false;
      status('error', '입력 확인 필요');
      showErrors(error);
    } finally {
      if (requestID === sequence) setBusy(false);
    }
  }
  function schedulePreview() {
    previewAbort?.abort();
    ++sequence;
    pending = false;
    valid = false;
    updateButtons();
    status('busy', '사양 변경 중');
    clearTimeout(previewTimer);
    previewTimer = setTimeout(previewNow, 280);
  }
  async function download(kind) {
    if (!valid || pending) return;
    setBusy(true);
    const button = kind === 'pptx' ? $('export-pptx') : kind === 'svg' ? $('export-svg') : $('save-button');
    button.setAttribute('aria-busy', 'true');
    try {
      const res = await api(`/api/export/${kind}`, readConfig());
      const blob = await res.blob();
      const safeCode = String(latestModel.code).replace(/[^A-Za-z0-9_-]/g, '_');
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a'); link.href = url; link.download = `${safeCode}.${kind === 'json' ? 'json' : kind}`;
      document.body.append(link); link.click(); link.remove();
      setTimeout(() => URL.revokeObjectURL(url), 10000);
      toast(kind === 'pptx' ? '편집 가능한 PPTX 도면을 저장했습니다.' : kind === 'svg' ? 'SVG 도면을 저장했습니다.' : '현재 사양을 JSON 파일로 저장했습니다.');
    } catch (error) { showErrors(error); toast(errorMessage(error)); }
    finally { button.removeAttribute('aria-busy'); setBusy(false); }
  }
  function activateTab(name) {
    for (const tab of document.querySelectorAll('.tab')) {
      const selected = tab.dataset.tab === name;
      tab.classList.toggle('active', selected);
      tab.setAttribute('aria-selected', String(selected));
      tab.tabIndex = selected ? 0 : -1;
      $(`panel-${tab.dataset.tab}`).hidden = !selected;
    }
  }
  async function decodeCode() {
    const text = $('decode-input').value.trim();
    $('decode-error').textContent = '';
    if (!text) { $('decode-error').textContent = '도면번호를 입력하세요.'; return; }
    $('decode-button').disabled = true;
    try {
      const res = await api('/api/decode', {code: text});
      const result = await res.json();
      applyConfig({...result.config, preset_id: result.config.preset_id || ''});
      await previewNow();
      if (valid && result.warnings?.length) renderNotes(latestModel, result.warnings);
      toast('도면번호의 사양을 불러왔습니다.');
    } catch (error) { $('decode-error').textContent = errorMessage(error); }
    finally { $('decode-button').disabled = false; }
  }
  async function importConfig(file) {
    if (!file) return;
    if (file.size > 1024 * 1024) { toast('설정 파일은 1MB 이하의 JSON 파일을 선택하세요.'); return; }
    try {
      const value = JSON.parse(await file.text());
      const config = value.config || value;
      if (!config || typeof config !== 'object' || Array.isArray(config) || !config.lengths) throw new Error('pCOM 설정 JSON 파일을 선택하세요.');
      // Preserve 16-bit/unsupported inputs for explicit backend rejection instead of silently converting them.
      applyConfig(config);
      await previewNow();
      if (valid) toast('저장한 사양을 불러왔습니다.');
    } catch (error) { toast(errorMessage(error)); }
    finally { $('import-file').value = ''; }
  }
  function bindEvents() {
    $('spec-form').addEventListener('submit', (event) => event.preventDefault());
    $('spec-form').addEventListener('input', (event) => { if (event.target.id !== 'preset') schedulePreview(); });
    $('preset').addEventListener('change', () => {
      const preset = catalog.presets.find((item) => item.id === $('preset').value);
      if (!preset) return;
      applyConfig({...clone(preset.config), preset_id: preset.id});
      previewNow();
    });
    $('reset-button').addEventListener('click', () => { if (!confirm('현재 사양을 기본 사양으로 초기화할까요?')) return; applyConfig(catalog.defaults); previewNow(); toast('기본 사양으로 초기화했습니다.'); });
    $('export-pptx').addEventListener('click', () => download('pptx'));
    $('export-svg').addEventListener('click', () => download('svg'));
    $('save-button').addEventListener('click', () => download('json'));
    $('import-button').addEventListener('click', () => $('import-file').click());
    $('import-file').addEventListener('change', (event) => importConfig(event.target.files[0]));
    $('copy-code').addEventListener('click', async () => {
      if (!latestModel) return;
      try { await navigator.clipboard.writeText(latestModel.code); toast('도면번호를 복사했습니다.'); }
      catch { const input = document.createElement('textarea'); input.value = latestModel.code; input.style.position='fixed'; input.style.opacity='0'; document.body.append(input); input.select(); const copied = document.execCommand('copy'); input.remove(); toast(copied ? '도면번호를 복사했습니다.' : '도면번호를 선택하여 복사해주세요.'); }
    });
    $('decode-button').addEventListener('click', decodeCode);
    $('decode-input').addEventListener('keydown', (event) => { if (event.key === 'Enter') { event.preventDefault(); decodeCode(); } });
    for (const tab of document.querySelectorAll('.tab')) {
      tab.addEventListener('click', () => activateTab(tab.dataset.tab));
      tab.addEventListener('keydown', (event) => {
        if (!['ArrowLeft','ArrowRight'].includes(event.key)) return;
        const tabs = [...document.querySelectorAll('.tab')];
        const next = tabs[(tabs.indexOf(tab) + (event.key === 'ArrowRight' ? 1 : tabs.length - 1)) % tabs.length];
        next.click(); next.focus(); event.preventDefault();
      });
    }
    $('exit-button').addEventListener('click', async () => {
      if (!confirm('pCOM 도면 작성 프로그램을 종료할까요? 현재 사양은 이 기기에 저장됩니다.')) return;
      try {
        if (valid) saveLocal();
        await api('/api/shutdown', {});
        $('boot-error').hidden = false;
        $('boot-error').querySelector('h2').textContent = '프로그램을 종료했습니다.';
        $('boot-error-text').textContent = '이 창을 닫으세요. 다시 작업하려면 pCOM 실행 파일을 여세요.';
        $('reload-button').hidden = true;
        window.close();
      } catch (error) { toast(`종료하지 못했습니다. ${errorMessage(error)}`); }
    });
    $('reload-button').addEventListener('click', () => location.reload());
  }
  async function boot() {
    try {
      const res = await api('/api/catalog'); catalog = await res.json();
      for (const field of selectFields) fillOptions(field);
      const customOption = document.createElement('option'); customOption.value = ''; customOption.textContent = '직접 입력 사양';
      $('preset').replaceChildren(customOption, ...catalog.presets.map((item) => {
        const option = document.createElement('option'); option.value = item.id; option.textContent = item.label; return option;
      }));
      let restored;
      try { const value = localStorage.getItem(storageKey); if (value) { const parsed = JSON.parse(value); if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) restored = parsed; } } catch { /* startup continues with defaults */ }
      applyConfig(restored || catalog.defaults);
      bindEvents();
      setInterval(() => fetch('/api/ping', {cache: 'no-store'}).catch(() => {}), 30000);
      await previewNow();
    } catch (error) {
      $('boot-error').hidden = false;
      $('boot-error-text').textContent = `${errorMessage(error)}\n실행 파일을 다시 열거나 연결 상태를 확인하세요.`;
      $('reload-button').addEventListener('click', () => location.reload());
    }
  }
  boot();
})();
