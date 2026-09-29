// Small helpers shared by the uPlot charts.

export function cssVar(name: string, fallback: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim() || fallback;
}

/** Appends an 8-bit alpha to a #rrggbb colour (a in 0..1); other values pass through. */
export function withAlpha(hex: string, a: number): string {
  if (!/^#[0-9a-fA-F]{6}$/.test(hex)) return hex;
  return hex + Math.round(Math.max(0, Math.min(1, a)) * 255).toString(16).padStart(2, "0");
}

/** Keeps a chart sized to its host element. Returns a cleanup function. */
export function autosize(host: HTMLElement, resize: (width: number, height: number) => void): () => void {
  const obs = new ResizeObserver(() => resize(Math.max(200, host.clientWidth), host.clientHeight));
  obs.observe(host);
  return () => obs.disconnect();
}
