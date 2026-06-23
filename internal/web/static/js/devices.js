// Emulator/device list: inventory polling, rendering, and row actions.

import { $, el } from "./dom.js";
import { state, LAUNCH_TIMEOUT } from "./state.js";
import { api } from "./api.js";
import { openLogcat, stopLogcat } from "./logcat.js";

export async function refresh() {
  let data;
  try { data = await api.get("/api/inventory"); }
  catch (e) { return; }
  $("#backend").textContent = "backend: " + (data.backend || "?");
  state.entries = data.entries || [];
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
    meta.append(el("div", "state", e.running ? e.serial : starting ? "starting…" : "stopped"));
    li.append(meta);

    const actions = el("div", "actions");
    if (e.running) {
      actions.append(btn("Logcat", () => { state.selected = e.serial; openLogcat(e); renderDevices(); }));
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
