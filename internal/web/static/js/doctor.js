// Doctor modal: shows the preflight health checks.

import { $, el } from "./dom.js";
import { api } from "./api.js";

export async function showDoctor() {
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
