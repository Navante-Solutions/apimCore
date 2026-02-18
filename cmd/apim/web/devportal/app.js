const base = '/devportal/api';

async function get(path) {
  const r = await fetch(base + path);
  if (!r.ok) throw new Error(r.statusText);
  return r.json();
}

function el(id) { return document.getElementById(id); }

function renderProducts(container, data) {
  if (!Array.isArray(data) || data.length === 0) {
    container.innerHTML = '<p class="muted">Nenhum produto publicado.</p>';
    return;
  }
  container.innerHTML = data.map(p =>
    `<div class="item"><strong>${escape(p.name)}</strong> (${escape(p.slug)})</div>`
  ).join('');
}

function renderApis(container, data) {
  if (!Array.isArray(data) || data.length === 0) {
    container.innerHTML = '<p class="muted">Nenhuma API listada.</p>';
    return;
  }
  container.innerHTML = data.map(a =>
    `<div class="item"><strong>${escape(a.name)}</strong> ${escape(a.pathPrefix)}</div>`
  ).join('');
}

function renderUsage(container, data) {
  if (!data) {
    container.innerHTML = '<p class="loading">Carregando...</p>';
    return;
  }
  const byApi = data.by_api || {};
  const total = data.total || 0;
  let html = '<p>Total: <strong>' + total + '</strong></p>';
  if (Object.keys(byApi).length) {
    html += '<pre>' + JSON.stringify(byApi, null, 2) + '</pre>';
  }
  container.innerHTML = html;
}

function escape(s) {
  if (s == null) return '';
  const div = document.createElement('div');
  div.textContent = s;
  return div.innerHTML;
}

async function load() {
  var productsEl = el('products');
  var apisEl = el('apis');
  var usageEl = el('usage');
  if (!productsEl) return;
  productsEl.innerHTML = '<p class="loading">Carregando...</p>';
  apisEl.innerHTML = '<p class="loading">Carregando...</p>';
  usageEl.innerHTML = '<p class="loading">Carregando...</p>';
  try {
    var products = await get('/products');
    var apis = await get('/apis');
    var usage = await get('/usage');
    renderProducts(productsEl, products);
    renderApis(apisEl, apis);
    renderUsage(usageEl, usage);
  } catch (e) {
    productsEl.innerHTML = '<p class="err">Erro: ' + escape(e.message) + '</p>';
  }
}

load();
