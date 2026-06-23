// Tiny DOM helpers shared across modules.

export const $ = (sel) => document.querySelector(sel);

export function el(tag, cls, text) {
  const e = document.createElement(tag);
  if (cls) e.className = cls;
  if (text != null) e.textContent = text;
  return e;
}

// Collapse/expand the devices panel and keep the header toggle's pressed state
// in sync. Collapsed = logcat takes the full width.
export function setDevicesCollapsed(collapsed) {
  $("main").classList.toggle("devices-collapsed", collapsed);
  $("#toggle-devices").setAttribute("aria-pressed", String(collapsed));
}

export const devicesCollapsed = () => $("main").classList.contains("devices-collapsed");
