/**
 * Inline SVG icon set.
 * Image files are forbidden across the whole app, so every glyph is an inline
 * SVG string rendered through Lit's unsafeHTML directive — the same approach
 * the @yukino.js/lit-jsx community icon helpers use (string in, directive out).
 */

const STROKE =
  'viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"';

export const ICONS = {
  cart: `<svg ${STROKE}><circle cx="9" cy="20" r="1.4"/><circle cx="17.5" cy="20" r="1.4"/><path d="M2.5 3.5h2.6l2.5 12.2h11.1L21.5 7H6"/></svg>`,
  users: `<svg ${STROKE}><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>`,
  clock: `<svg ${STROKE}><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></svg>`,
  tag: `<svg ${STROKE}><path d="M12.6 2.9 21 11.3a2 2 0 0 1 0 2.8l-6.9 6.9a2 2 0 0 1-2.8 0L2.9 12.6A2 2 0 0 1 2.3 11V4.3a2 2 0 0 1 2-2H11a2 2 0 0 1 1.6.6Z"/><circle cx="7.5" cy="7.5" r="1.3"/></svg>`,
  check: `<svg ${STROKE}><path d="M20 6 9 17l-5-5"/></svg>`,
  close: `<svg ${STROKE}><path d="M18 6 6 18M6 6l12 12"/></svg>`,
  plus: `<svg ${STROKE}><path d="M12 5v14M5 12h14"/></svg>`,
  trash: `<svg ${STROKE}><path d="M3 6h18"/><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6"/><path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/><path d="M10 11v6M14 11v6"/></svg>`,
  pencil: `<svg ${STROKE}><path d="M17 3a2.8 2.8 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z"/></svg>`,
  info: `<svg ${STROKE}><circle cx="12" cy="12" r="9"/><path d="M12 16v-5M12 8h.01"/></svg>`,
  arrowLeft: `<svg ${STROKE}><path d="M19 12H5M12 19l-7-7 7-7"/></svg>`,
  refresh: `<svg ${STROKE}><path d="M21 12a9 9 0 1 1-2.64-6.36L21 8"/><path d="M21 3v5h-5"/></svg>`,
  sliders: `<svg ${STROKE}><path d="M4 21v-7M4 10V3M12 21v-9M12 8V3M20 21v-5M20 12V3"/><path d="M1.5 14h5M9.5 8h5M17.5 16h5"/></svg>`,
  bolt: `<svg ${STROKE}><path d="M13 2 3 14h8l-1 8 11-13h-8l1-7Z"/></svg>`,
  shield: `<svg ${STROKE}><path d="M12 22s8-3.6 8-10V5.2L12 2 4 5.2V12c0 6.4 8 10 8 10Z"/></svg>`,
  sparkles: `<svg ${STROKE}><path d="M12 3l1.9 5.1L19 10l-5.1 1.9L12 17l-1.9-5.1L5 10l5.1-1.9Z"/><path d="M19 15l.9 2.1L22 18l-2.1.9L19 21l-.9-2.1L16 18l2.1-.9Z"/></svg>`,
  alert: `<svg ${STROKE}><circle cx="12" cy="12" r="9"/><path d="M12 8v5M12 16.5h.01"/></svg>`,
  qr: `<svg ${STROKE}><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><path d="M14 14h3v3h-3zM20 14h1M14 20h1M18 18h3v3h-3z"/></svg>`,
  wallet: `<svg ${STROKE}><path d="M20 7H4a2 2 0 0 1 0-4h14v4"/><path d="M4 7v12a2 2 0 0 0 2 2h14a1 1 0 0 0 1-1V8a1 1 0 0 0-1-1"/><path d="M16.5 13.5h.01"/></svg>`,
  logout: `<svg ${STROKE}><path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/><path d="M16 17l5-5-5-5M21 12H9"/></svg>`,
} as const;

/** Injects a class attribute into the leading <svg> tag of an icon string. */
export function svgIcon(svg: string, className: string): string {
  return svg.replace("<svg ", `<svg class="${className}" `);
}
