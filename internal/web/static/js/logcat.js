// Live logcat: EventSource stream, batched rendering, and the line view.

import { $, el, setDevicesCollapsed } from "./dom.js";
import { state } from "./state.js";
import { api } from "./api.js";
import { knownTags, procSet, mineSet, passesFilter } from "./filter.js";

const MAX_DOM_LINES = 2000;
let pending = [];
let rafScheduled = false;

export function openLogcat(e) {
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
  // A device is now selected; collapse the list so logcat owns the width.
  setDevicesCollapsed(true);
  renderLog();

  // Reset and load autocomplete values for this device.
  knownTags.clear();
  procSet.clear();
  mineSet.clear();
  api.get("/api/meta?serial=" + encodeURIComponent(e.serial)).then((m) => {
    (m.mine || []).forEach((p) => mineSet.add(p));
    (m.processes || []).forEach((p) => procSet.add(p));
  }).catch(() => {});

  const q = new URLSearchParams({ serial: e.serial, name: e.name });
  const es = new EventSource("/api/logcat?" + q.toString());
  es.onmessage = (ev) => {
    const line = JSON.parse(ev.data);
    state.log.lines.push(line);
    if (state.log.lines.length > 5000) state.log.lines.splice(0, state.log.lines.length - 5000);
    if (line.tag) knownTags.add(line.tag);
    if (line.process) procSet.add(line.process.split(":")[0]);
    pending.push(line);
    scheduleFlush();
  };
  es.addEventListener("end", () => es.close());
  es.onerror = () => {};
  state.log.es = es;
}

export function stopLogcat() {
  if (state.log.es) { state.log.es.close(); state.log.es = null; }
  state.log.serial = null;
}

function lineNode(line) {
  const n = el("div", "line lvl-" + line.level);
  n.append(el("span", "t", line.time));
  n.append(el("span", "lv", line.level));
  if (line.tag) n.append(el("span", "tag", line.tag + ":"));
  n.append(el("span", "msg", line.msg));
  return n;
}

// Logcat can arrive as a firehose (thousands of lines/sec during boot). Buffer
// incoming lines and flush them to the DOM once per animation frame, capping
// how many nodes live in the DOM so the page stays responsive.
function scheduleFlush() {
  if (rafScheduled) return;
  rafScheduled = true;
  requestAnimationFrame(flushPending);
}

function flushPending() {
  rafScheduled = false;
  const lines = pending;
  pending = [];
  if (state.log.paused || lines.length === 0) return;

  const log = $("#log");
  const atBottom = log.scrollHeight - log.scrollTop - log.clientHeight < 60;
  const frag = document.createDocumentFragment();
  let added = 0;
  for (const line of lines) {
    if (passesFilter(line)) { frag.append(lineNode(line)); added++; }
  }
  if (added === 0) return;
  log.append(frag);
  trimLog(log);
  if (atBottom) log.scrollTop = log.scrollHeight;
}

function trimLog(log) {
  while (log.childElementCount > MAX_DOM_LINES) log.removeChild(log.firstChild);
}

// renderLog rebuilds the view (e.g. after a filter change), bounded to the most
// recent matching lines so a full redraw never stalls the page.
export function renderLog() {
  const log = $("#log");
  const frag = document.createDocumentFragment();
  let count = 0;
  for (let i = state.log.lines.length - 1; i >= 0 && count < MAX_DOM_LINES; i--) {
    if (passesFilter(state.log.lines[i])) { frag.prepend(lineNode(state.log.lines[i])); count++; }
  }
  log.replaceChildren(frag);
  log.scrollTop = log.scrollHeight;
}

// wireControls hooks up the pause / clear / level controls.
export function wireControls() {
  $("#pause").addEventListener("click", () => {
    state.log.paused = !state.log.paused;
    $("#pause").textContent = state.log.paused ? "Resume" : "Pause";
    if (!state.log.paused) renderLog();
  });
  $("#clear").addEventListener("click", () => { state.log.lines = []; renderLog(); });
  $("#level").addEventListener("change", (e) => { state.log.minLevel = +e.target.value; renderLog(); });
}
