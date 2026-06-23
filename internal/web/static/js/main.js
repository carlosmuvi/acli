// Composition root: wire top-level controls and start the inventory poll.

import { $ } from "./dom.js";
import { refresh } from "./devices.js";
import { wireControls } from "./logcat.js";
import { initFilter } from "./autocomplete.js";
import { showDoctor } from "./doctor.js";

$("#refresh").addEventListener("click", refresh);
$("#doctor-btn").addEventListener("click", showDoctor);
$("#doctor-close").addEventListener("click", () => ($("#doctor-modal").hidden = true));
$("#shot-close").addEventListener("click", () => ($("#shot-modal").hidden = true));

wireControls();
initFilter();

refresh();
setInterval(refresh, 3000);
