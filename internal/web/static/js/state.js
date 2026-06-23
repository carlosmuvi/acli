// Shared mutable UI state. Modules import and mutate this single object.

export const LAUNCH_TIMEOUT = 120000; // ms a row shows "starting…" before giving up

export const state = {
  entries: [],        // unified emulator/device list from /api/inventory
  selected: null,     // serial of the device whose logcat is open
  starting: new Map(), // avd -> timestamp of a launch we kicked off
  log: {
    es: null,         // EventSource for the live stream
    serial: null,
    name: null,
    lines: [],        // ring buffer of recent parsed lines
    paused: false,
    minLevel: 0,      // from the level dropdown
    filter: "",       // raw filter query string
    terms: [],        // parsed query terms
  },
};
