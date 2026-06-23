// Android-Studio-style logcat filter query language.
//
// Space-separated terms are ANDed. Each term may be negated with a leading "-"
// and may carry a key: tag, package/process, level, message, is. Bare terms
// match tag + message + process.

import { state } from "./state.js";

// Autocomplete value sources, populated by the logcat stream / meta endpoint.
export const knownTags = new Set();
export const procSet = new Set();
export const mineSet = new Set();

export const FILTER_KEYS = ["tag:", "package:", "process:", "level:", "message:", "is:"];

const LEVELS = {
  v: 0, d: 1, i: 2, w: 3, e: 4, f: 5,
  verbose: 0, debug: 1, info: 2, warn: 3, warning: 3, error: 4, fatal: 5, assert: 5,
};

export function parseQuery(s) {
  const terms = [];
  for (let raw of s.trim().split(/\s+/)) {
    if (!raw) continue;
    let neg = false;
    if (raw[0] === "-") { neg = true; raw = raw.slice(1); }
    if (!raw) continue;
    const ci = raw.indexOf(":");
    let key = "", val = raw;
    if (ci > 0) { key = raw.slice(0, ci).toLowerCase(); val = raw.slice(ci + 1); }
    terms.push({ neg, key, val: val.toLowerCase() });
  }
  return terms;
}

function matchTerm(line, t) {
  const tag = (line.tag || "").toLowerCase();
  const msg = (line.msg || "").toLowerCase();
  const proc = (line.process || "").toLowerCase();
  const procBase = proc.split(":")[0];
  switch (t.key) {
    case "tag": return tag.includes(t.val);
    case "message": case "msg": return msg.includes(t.val);
    case "package": case "process":
      if (t.val === "mine") return mineSet.has(procBase) || mineSet.has(proc);
      return proc.includes(t.val);
    case "level": {
      const lv = LEVELS[t.val];
      return lv == null ? true : line.prio >= lv;
    }
    case "is":
      if (t.val === "crash") return line.level === "F" || (tag.includes("androidruntime") && msg.includes("fatal"));
      if (t.val === "stacktrace") return /^\s*(at\s+\S+\(|caused by:|\.\.\.\s+\d+\s+more)/i.test(line.msg || "");
      return true;
    case "": return tag.includes(t.val) || msg.includes(t.val) || procBase.includes(t.val);
    default: return (tag + " " + msg).includes(t.key + ":" + t.val);
  }
}

// passesFilter applies the level dropdown plus every parsed query term.
export function passesFilter(line) {
  if (line.prio < state.log.minLevel && line.level !== "?") return false;
  for (const t of state.log.terms) {
    const m = matchTerm(line, t);
    if (t.neg ? m : !m) return false;
  }
  return true;
}
