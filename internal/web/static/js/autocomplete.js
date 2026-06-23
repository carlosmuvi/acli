// Filter input + Android-Studio-style autocomplete dropdown.

import { $, el } from "./dom.js";
import { state } from "./state.js";
import { FILTER_KEYS, knownTags, procSet, mineSet, parseQuery } from "./filter.js";
import { renderLog } from "./logcat.js";

let filterEl, acEl;
let acItems = [];
let acIndex = 0;

export function initFilter() {
  filterEl = $("#filter");
  acEl = $("#ac");

  filterEl.addEventListener("input", () => { applyFilter(); updateAutocomplete(); });
  filterEl.addEventListener("focus", updateAutocomplete);
  filterEl.addEventListener("blur", () => setTimeout(hideAC, 150));
  filterEl.addEventListener("keydown", onKeydown);
}

function applyFilter() {
  state.log.filter = filterEl.value;
  state.log.terms = parseQuery(filterEl.value);
  renderLog();
}

// tokenAtCaret returns the whitespace-delimited token the caret sits in.
function tokenAtCaret() {
  const v = filterEl.value;
  const pos = filterEl.selectionStart;
  let start = pos;
  while (start > 0 && !/\s/.test(v[start - 1])) start--;
  let end = pos;
  while (end < v.length && !/\s/.test(v[end])) end++;
  return { token: v.slice(start, end), start, end };
}

function suggestionsFor(token) {
  const ci = token.indexOf(":");
  if (ci < 0) {
    const t = token.toLowerCase();
    return FILTER_KEYS.filter((k) => k.startsWith(t))
      .map((k) => ({ insert: k, label: k, hint: "filter", complete: false }));
  }
  const key = token.slice(0, ci).toLowerCase();
  const partial = token.slice(ci + 1).toLowerCase();
  let values = [];
  if (key === "tag") values = [...knownTags];
  else if (key === "package" || key === "process") values = ["mine", ...mineSet, ...procSet];
  else if (key === "level") values = ["VERBOSE", "DEBUG", "INFO", "WARN", "ERROR", "FATAL"];
  else if (key === "is") values = ["crash", "stacktrace"];
  else return [];
  const seen = new Set();
  return values
    .filter((v) => v && v.toLowerCase().includes(partial) && !seen.has(v) && seen.add(v))
    .slice(0, 12)
    .map((v) => ({ insert: key + ":" + v + " ", label: key + ":" + v, hint: "", complete: true }));
}

function updateAutocomplete() {
  const { token } = tokenAtCaret();
  acItems = suggestionsFor(token);
  if (acItems.length === 0) { hideAC(); return; }
  acIndex = 0;
  renderAC();
}

function renderAC() {
  acEl.innerHTML = "";
  acItems.forEach((s, i) => {
    const li = el("li", i === acIndex ? "active" : null);
    li.append(el("span", "val", s.label));
    if (s.hint) li.append(el("span", "hint", s.hint));
    li.addEventListener("mousedown", (ev) => { ev.preventDefault(); acceptAC(i); });
    acEl.append(li);
  });
  acEl.hidden = false;
}

function hideAC() { acEl.hidden = true; acItems = []; }

function acceptAC(i) {
  const s = acItems[i];
  if (!s) return;
  const { start, end } = tokenAtCaret();
  const v = filterEl.value;
  filterEl.value = v.slice(0, start) + s.insert + v.slice(end);
  const caret = start + s.insert.length;
  filterEl.setSelectionRange(caret, caret);
  filterEl.focus();
  applyFilter();
  updateAutocomplete(); // after choosing a key, offer its values
}

function onKeydown(e) {
  if (acEl.hidden || acItems.length === 0) return;
  if (e.key === "ArrowDown") { e.preventDefault(); acIndex = (acIndex + 1) % acItems.length; renderAC(); }
  else if (e.key === "ArrowUp") { e.preventDefault(); acIndex = (acIndex - 1 + acItems.length) % acItems.length; renderAC(); }
  else if (e.key === "Enter" || e.key === "Tab") { e.preventDefault(); acceptAC(acIndex); }
  else if (e.key === "Escape") { hideAC(); }
}
