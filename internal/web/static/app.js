"use strict";

const $ = (sel) => document.querySelector(sel);
const el = (tag, cls, text) => {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  if (text != null) e.textContent = text;
  return e;
};

const api = {
  async get(path) { const r = await fetch(path); return r.json(); },
  async post(path, body) {
    const r = await fetch(path, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body || {}),
    });
    const data = await r.json().catch(() => ({}));
    if (!r.ok) throw new Error(data.error || r.statusText);
    return data;
  },
};

const state = {
  entries: [],
  selected: null,        // serial of selected device
  starting: new Map(),   // avd -> timestamp
  log: {
    es: null,
    serial: null,
    name: null,
    lines: [],           // {time, level, prio, tag, msg}
    paused: false,
    minLevel: 0,
    filter: "",
  },
};

const LAUNCH_TIMEOUT = 120000;

// ---- inventory ----

async function refresh() {
  let data;
  try { data = await api.get("/api/inventory"); }
  catch (e) { return; }
  $("#backend").textContent = "backend: " + (data.backend || "?");
  state.entries = data.entries || [];
  // clear "starting" once an avd is running
  for (const e of state.entries) {
    if (e.running && e.avd) state.starting.delete(e.avd);
  }
  renderDevices();
}

function isStarting(avd) {
  const t = state.starting.get(avd);
  return t != null && Date.now() - t < LAUNCH_TIMEOUT;
}

function renderDevices() {
  const list = $("#devices");
  list.innerHTML = "";
  $("#devices-empty").hidden = state.entries.length > 0;

  for (const e of state.entries) {
    const li = el("li", "device");
    if (e.serial && e.serial === state.selected) li.classList.add("selected");

    const starting = !e.running && isStarting(e.avd);
    const status = e.running ? "running" : starting ? "starting" : "stopped";
    li.append(el("span", "dot " + status));

    const meta = el("div");
    meta.append(el("span", "name", e.name));
    const st = el("div", "state", e.running ? e.serial : starting ? "starting…" : "stopped");
    meta.append(st);
    li.append(meta);

    const actions = el("div", "actions");
    if (e.running) {
      actions.append(btn("Logcat", () => openLogcat(e)));
      actions.append(btn("Shot", () => screenshot(e)));
      actions.append(btn("Kill", () => kill(e), "danger"));
    } else if (e.avd) {
      actions.append(btn("Launch", () => launch(e.avd, {})));
      actions.append(btn("Cold", () => launch(e.avd, { cold: true })));
      actions.append(btn("Wipe", () => launch(e.avd, { wipe: true })));
    }
    li.append(actions);
    list.append(li);
  }
}

function btn(label, onClick, cls) {
  const b = el("button", "btn" + (cls ? " " + cls : ""), label);
  b.addEventListener("click", onClick);
  return b;
}

// ---- actions ----

async function launch(avd, opts) {
  state.starting.set(avd, Date.now());
  renderDevices();
  try { await api.post("/api/launch", { avd, cold: !!opts.cold, wipe: !!opts.wipe }); }
  catch (e) { alert("launch failed: " + e.message); state.starting.delete(avd); }
  setTimeout(refresh, 800);
}

async function kill(e) {
  try { await api.post("/api/kill", { serial: e.serial }); }
  catch (err) { alert("kill failed: " + err.message); }
  if (state.log.serial === e.serial) stopLogcat();
  setTimeout(refresh, 500);
}

async function screenshot(e) {
  try {
    const r = await api.post("/api/screenshot", { serial: e.serial, name: e.name });
    $("#shot-img").src = r.url + "?t=" + Date.now();
    $("#shot-path").textContent = "saved to " + r.path;
    $("#shot-modal").hidden = false;
  } catch (err) { alert("screenshot failed: " + err.message); }
}

// ---- logcat ----

function openLogcat(e) {
  stopLogcat();
  state.selected = e.serial;
  state.log.serial = e.serial;
  state.log.name = e.name;
  state.log.lines = [];
  state.log.paused = false;
  $("#logcat-title").textContent = "Logcat ▸ " + e.name;
  $("#logcat-hint").hidden = true;
  $("#pause").disabled = false;
  $("#clear").disabled = false;
  $("#pause").textContent = "Pause";
  renderDevices();
  renderLog();

  const q = new URLSearchParams({ serial: e.serial, name: e.name });
  const es = new EventSource("/api/logcat?" + q.toString());
  es.onmessage = (ev) => {
    const line = JSON.parse(ev.data);
    state.log.lines.push(line);
    if (state.log.lines.length > 5000) state.log.lines.splice(0, state.log.lines.length - 5000);
    if (!state.log.paused) appendLine(line);
  };
  es.addEventListener("end", () => es.close());
  es.onerror = () => {};
  state.log.es = es;
}

function stopLogcat() {
  if (state.log.es) { state.log.es.close(); state.log.es = null; }
  state.log.serial = null;
}

function passesFilter(line) {
  if (line.prio < state.log.minLevel && line.level !== "?") return false;
  const f = state.log.filter.toLowerCase();
  if (f && !((line.tag + " " + line.msg).toLowerCase().includes(f))) return false;
  return true;
}

function lineNode(line) {
  const n = el("div", "line lvl-" + line.level);
  n.append(el("span", "t", line.time));
  n.append(el("span", "lv", line.level));
  if (line.tag) n.append(el("span", "tag", line.tag + ":"));
  n.append(el("span", "msg", line.msg));
  return n;
}

function appendLine(line) {
  if (!passesFilter(line)) return;
  const log = $("#log");
  const atBottom = log.scrollHeight - log.scrollTop - log.clientHeight < 40;
  log.append(lineNode(line));
  if (atBottom) log.scrollTop = log.scrollHeight;
}

function renderLog() {
  const log = $("#log");
  log.innerHTML = "";
  for (const line of state.log.lines) {
    if (passesFilter(line)) log.append(lineNode(line));
  }
  log.scrollTop = log.scrollHeight;
}

// ---- doctor ----

async function showDoctor() {
  const checks = await api.get("/api/doctor");
  const ul = $("#doctor-list");
  ul.innerHTML = "";
  for (const c of checks) {
    const li = el("li");
    li.append(el("span", null, (c.OK ? "✓ " : "✗ ") + c.Name + "  "));
    li.append(el("span", "muted", c.Detail || ""));
    if (!c.OK && c.Fix) li.append(el("div", "fix", "fix: " + c.Fix));
    ul.append(li);
  }
  $("#doctor-modal").hidden = false;
}

// ---- wiring ----

$("#refresh").addEventListener("click", refresh);
$("#doctor-btn").addEventListener("click", showDoctor);
$("#doctor-close").addEventListener("click", () => ($("#doctor-modal").hidden = true));
$("#shot-close").addEventListener("click", () => ($("#shot-modal").hidden = true));

$("#pause").addEventListener("click", () => {
  state.log.paused = !state.log.paused;
  $("#pause").textContent = state.log.paused ? "Resume" : "Pause";
  if (!state.log.paused) renderLog();
});
$("#clear").addEventListener("click", () => { state.log.lines = []; renderLog(); });
$("#level").addEventListener("change", (e) => { state.log.minLevel = +e.target.value; renderLog(); });
$("#filter").addEventListener("input", (e) => { state.log.filter = e.target.value; renderLog(); });

refresh();
setInterval(refresh, 3000);
