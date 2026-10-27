const tokenInput = document.getElementById('token');
const connectBtn = document.getElementById('connect');
const disconnectBtn = document.getElementById('disconnect');
const fullscreenBtn = document.getElementById('fullscreen');
const statusEl = document.getElementById('status');
const fpsEl = document.getElementById('fps');
const resolutionEl = document.getElementById('resolution');
const bitrateEl = document.getElementById('bitrate');
const serverStatusEl = document.getElementById('server-status');
const canvas = document.getElementById('frame');
const ctx = canvas.getContext('2d');

const STORAGE_KEY = 'screenshare_token';

let abortController = null;
let reconnectTimer = null;
let backoffMs = 1000;
let frameCount = 0;
let byteCount = 0;
let statsTimer = null;
let statusPollTimer = null;

function authHeaders() {
  const token = tokenInput.value.trim();
  return {
    Authorization: `Bearer ${token}`,
    'X-Screenshare-Token': token,
  };
}

function setStatus(text) {
  statusEl.textContent = text;
}

function loadToken() {
  const saved = sessionStorage.getItem(STORAGE_KEY);
  if (saved) tokenInput.value = saved;
}

function saveToken() {
  sessionStorage.setItem(STORAGE_KEY, tokenInput.value.trim());
}

async function apiPost(path) {
  const res = await fetch(path, { method: 'POST', headers: authHeaders() });
  if (!res.ok) throw new Error(`${path} failed: ${res.status}`);
}

function startStats() {
  frameCount = 0;
  byteCount = 0;
  if (statsTimer) clearInterval(statsTimer);
  statsTimer = setInterval(() => {
    fpsEl.textContent = String(frameCount);
    frameCount = 0;
    const kbps = Math.round((byteCount * 8) / 1000);
    bitrateEl.textContent = `${kbps} kbps`;
    byteCount = 0;
  }, 1000);
}

function stopStats() {
  if (statsTimer) clearInterval(statsTimer);
  statsTimer = null;
}

async function pollServerStatus() {
  try {
    const res = await fetch('/api/status', { headers: authHeaders() });
    if (!res.ok) return;
    const data = await res.json();
    const parts = [];
    if (data.running) parts.push('capturing');
    if (data.encode_fps) parts.push(`enc ${data.encode_fps.toFixed(1)} fps`);
    if (data.connected_clients != null) parts.push(`${data.connected_clients} viewer(s)`);
    serverStatusEl.textContent = parts.length ? parts.join(' · ') : 'idle';
  } catch {
    /* ignore */
  }
}

function startStatusPoll() {
  if (statusPollTimer) clearInterval(statusPollTimer);
  statusPollTimer = setInterval(pollServerStatus, 2000);
  pollServerStatus();
}

function stopStatusPoll() {
  if (statusPollTimer) clearInterval(statusPollTimer);
  statusPollTimer = null;
  serverStatusEl.textContent = '—';
}

async function readMJPEGStream(signal) {
  const res = await fetch('/stream', { headers: authHeaders(), signal });
  if (!res.ok) throw new Error(`stream HTTP ${res.status}`);
  const contentType = res.headers.get('Content-Type') || '';
  const boundaryMatch = contentType.match(/boundary=([^;]+)/i);
  const boundary = boundaryMatch ? boundaryMatch[1].trim() : 'screenshareframe';

  const reader = res.body.getReader();
  let carry = new Uint8Array(0);

  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    if (!value) continue;

    const merged = new Uint8Array(carry.length + value.length);
    merged.set(carry);
    merged.set(value, carry.length);
    carry = merged;

    const delimiter = `--${boundary}`;
    const delimBytes = new TextEncoder().encode(delimiter);
    let start = indexOfSubarray(carry, delimBytes);
    while (start !== -1) {
      const next = indexOfSubarray(carry, delimBytes, start + delimBytes.length);
      if (next === -1) break;
      const segment = carry.slice(start, next);
      const jpeg = extractJPEG(segment);
      if (jpeg) {
        byteCount += jpeg.byteLength;
        frameCount += 1;
        await drawJPEG(jpeg);
      }
      carry = carry.slice(next);
      start = indexOfSubarray(carry, delimBytes);
    }
  }
}

function indexOfSubarray(haystack, needle, from = 0) {
  outer: for (let i = from; i <= haystack.length - needle.length; i++) {
    for (let j = 0; j < needle.length; j++) {
      if (haystack[i + j] !== needle[j]) continue outer;
    }
    return i;
  }
  return -1;
}

function extractJPEG(partBytes) {
  const headerSep = indexOfSubarray(partBytes, new TextEncoder().encode('\r\n\r\n'));
  if (headerSep === -1) return null;
  const body = partBytes.slice(headerSep + 4);
  let end = body.length;
  if (end >= 2 && body[end - 2] === 13 && body[end - 1] === 10) end -= 2;
  if (end < 4) return null;
  if (body[0] !== 0xff || body[1] !== 0xd8) return null;
  return body.slice(0, end);
}

async function drawJPEG(jpegBytes) {
  const blob = new Blob([jpegBytes], { type: 'image/jpeg' });
  const bitmap = await createImageBitmap(blob);
  if (canvas.width !== bitmap.width || canvas.height !== bitmap.height) {
    canvas.width = bitmap.width;
    canvas.height = bitmap.height;
    resolutionEl.textContent = `${bitmap.width}×${bitmap.height}`;
  }
  ctx.drawImage(bitmap, 0, 0);
  bitmap.close();
}

async function connect() {
  saveToken();
  if (!tokenInput.value.trim()) {
    setStatus('Token required');
    return;
  }
  disconnect(false);
  abortController = new AbortController();
  connectBtn.disabled = true;
  disconnectBtn.disabled = false;
  fullscreenBtn.disabled = false;
  setStatus('Connecting…');
  startStats();
  startStatusPoll();

  try {
    await apiPost('/api/start');
    setStatus('Streaming');
    backoffMs = 1000;
    await readMJPEGStream(abortController.signal);
    setStatus('Stream ended');
  } catch (err) {
    if (abortController.signal.aborted) {
      setStatus('Disconnected');
      return;
    }
    setStatus(`Error: ${err.message}. Reconnecting…`);
    scheduleReconnect();
  } finally {
    connectBtn.disabled = false;
  }
}

function scheduleReconnect() {
  if (reconnectTimer) clearTimeout(reconnectTimer);
  reconnectTimer = setTimeout(() => {
    backoffMs = Math.min(backoffMs * 2, 15000);
    connect();
  }, backoffMs);
}

function disconnect(callStop = true) {
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (abortController) {
    abortController.abort();
    abortController = null;
  }
  stopStats();
  stopStatusPoll();
  connectBtn.disabled = false;
  disconnectBtn.disabled = true;
  fullscreenBtn.disabled = true;
  if (callStop) {
    fetch('/api/stop', { method: 'POST', headers: authHeaders() }).catch(() => {});
    setStatus('Disconnected');
  }
}

connectBtn.addEventListener('click', () => connect());
disconnectBtn.addEventListener('click', () => disconnect());
fullscreenBtn.addEventListener('click', () => canvas.requestFullscreen?.());

loadToken();
