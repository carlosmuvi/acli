// Unit tests for the logcat filter query language (static/js/filter.js).
// Run with: node --test  (from internal/web)

import test from "node:test";
import assert from "node:assert/strict";

import { parseQuery, passesFilter, mineSet, knownTags } from "../static/js/filter.js";
import { state } from "../static/js/state.js";

function line(o) {
  return Object.assign(
    { time: "", level: "I", prio: 2, tag: "", msg: "", process: "" },
    o
  );
}

function withFilter(query, minLevel = 0) {
  state.log.minLevel = minLevel;
  state.log.terms = parseQuery(query);
}

test("parseQuery splits terms, keys, and negation", () => {
  const terms = parseQuery("hello tag:Foo -package:mine");
  assert.deepEqual(terms, [
    { neg: false, key: "", val: "hello" },
    { neg: false, key: "tag", val: "foo" },
    { neg: true, key: "package", val: "mine" },
  ]);
});

test("level: filters by minimum priority", () => {
  withFilter("level:W");
  assert.equal(passesFilter(line({ prio: 4, level: "E" })), true);
  assert.equal(passesFilter(line({ prio: 3, level: "W" })), true);
  assert.equal(passesFilter(line({ prio: 2, level: "I" })), false);
});

test("tag: matches substring, case-insensitive", () => {
  withFilter("tag:activity");
  assert.equal(passesFilter(line({ tag: "ActivityManager" })), true);
  assert.equal(passesFilter(line({ tag: "Choreographer" })), false);
});

test("negation excludes matches", () => {
  withFilter("-tag:Choreographer");
  assert.equal(passesFilter(line({ tag: "Choreographer" })), false);
  assert.equal(passesFilter(line({ tag: "ActivityManager" })), true);
});

test("package:mine matches third-party packages only", () => {
  mineSet.clear();
  mineSet.add("com.example.app");
  withFilter("package:mine");
  assert.equal(passesFilter(line({ process: "com.example.app" })), true);
  assert.equal(passesFilter(line({ process: "com.example.app:push" })), true);
  assert.equal(passesFilter(line({ process: "com.android.systemui" })), false);
});

test("package:<name> matches process substring", () => {
  withFilter("process:systemui");
  assert.equal(passesFilter(line({ process: "com.android.systemui" })), true);
  assert.equal(passesFilter(line({ process: "com.example.app" })), false);
});

test("is:crash flags fatal lines", () => {
  withFilter("is:crash");
  assert.equal(passesFilter(line({ level: "F", prio: 5 })), true);
  assert.equal(
    passesFilter(line({ tag: "AndroidRuntime", msg: "FATAL EXCEPTION: main" })),
    true
  );
  assert.equal(passesFilter(line({ tag: "Foo", msg: "all good" })), false);
});

test("multiple terms AND together", () => {
  withFilter("tag:App message:boom");
  assert.equal(passesFilter(line({ tag: "MyApp", msg: "boom happened" })), true);
  assert.equal(passesFilter(line({ tag: "MyApp", msg: "all fine" })), false);
});

test("bare term matches tag, message, or process", () => {
  knownTags.clear();
  withFilter("boom");
  assert.equal(passesFilter(line({ msg: "kaboom" })), true);
  assert.equal(passesFilter(line({ tag: "Boombox" })), true);
  assert.equal(passesFilter(line({ msg: "quiet" })), false);
});
