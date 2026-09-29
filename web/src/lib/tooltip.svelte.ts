// One floating tooltip for the whole page. Content is either plain text or a
// day bucket (rendered richly by Tooltip.svelte).

import type { DayBucket } from "./api-types";

export type TipContent = { text: string } | { day: DayBucket };

export const tip = $state({
  content: null as TipContent | null,
  /** Anchor in viewport coordinates; Tooltip.svelte places itself above it. */
  x: 0,
  top: 0,
  bottom: 0,
});

export function showTipAt(content: TipContent, x: number, top: number, bottom = top) {
  tip.content = content;
  tip.x = x;
  tip.top = top;
  tip.bottom = bottom;
}

export function hideTip() {
  tip.content = null;
}

/** `use:tooltip={() => content}` shows content above the element on hover. */
export function tooltip(node: HTMLElement, get: () => TipContent) {
  let current = get;
  const enter = () => {
    const r = node.getBoundingClientRect();
    showTipAt(current(), r.left + r.width / 2, r.top, r.bottom);
  };
  node.addEventListener("mouseenter", enter);
  node.addEventListener("mouseleave", hideTip);
  return {
    update(next: () => TipContent) {
      current = next;
    },
    destroy() {
      node.removeEventListener("mouseenter", enter);
      node.removeEventListener("mouseleave", hideTip);
    },
  };
}
