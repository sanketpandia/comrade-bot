/**
 * LogbookTableRenderer.ts
 * ─────────────────────────────────────────────────────────
 * Modern flight history table renderer matching LiveTableRenderer theme.
 * Renders logbook entries with Catppuccin-inspired dark theme styling.
 *
 * Color scheme:
 * - Background: #1e1e2e (dark base)
 * - Stripe: #2a2b3d (alternating rows)
 * - Header: #313244 (darker header)
 * - Text: #f5e0dc (warm light text)
 * - Header Text: #cdd6f4 (cool light text)
 * - Borders: #45475a (gray borders)
 */

import { createCanvas, GlobalFonts } from "@napi-rs/canvas";
import { CanvasTable } from "canvas-table";
import { FlightHistoryRecord } from "../types/Responses";

// ──────────────────────────────────────────────────────────────
// Constants / styles (matching LiveTableRenderer)
// ──────────────────────────────────────────────────────────────

const FONT_FAMILY = "DejaVu Sans Mono";
const FONT = `14px '${FONT_FAMILY}'`;

const BG_HEADER = "#313244";
const FG_HEADER = "#cdd6f4";
const BG_ROW = "#1e1e2e";
const BG_STRIPE = "#2a2b3d";
const FG_TEXT = "#f5e0dc";
const BORDER_COLOUR = "#45475a";

// Pre‑register font (safe‑fail) ————————————————————————————
try {
  GlobalFonts.registerFromPath(
    "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
    FONT_FAMILY,
  );
} catch {
  // Font not available, will use default
}

// ──────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────

/**
 * Format ISO timestamp to relative time ("2m ago")
 */
function fmtRelativeTime(iso: string): string {
  try {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return iso;

    const diff = Date.now() - d.getTime();
    const mins = Math.floor(diff / 60000);

    if (mins < 1) return "Just now";
    if (mins < 60) return `${mins}m ago`;
    const hours = Math.floor(mins / 60);
    if (hours < 24) return `${hours}h ago`;
    return `${Math.floor(hours / 24)}d ago`;
  } catch {
    return iso;
  }
}

/**
 * Format timestamp to "YYYY-MM-DD HH:MM"
 */
function fmtDateTime(iso: string): string {
  try {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return iso;
    return d.toISOString().replace("T", " ").slice(0, 16);
  } catch {
    return iso;
  }
}

/**
 * Get color based on violation count
 */
function getViolationColor(violations: number): string {
  if (violations > 0) return "#f38ba8"; // Red - has violations
  return FG_TEXT; // Default
}

// ──────────────────────────────────────────────────────────────
// Main API
// ──────────────────────────────────────────────────────────────

export async function renderLogbookTable(
  records: FlightHistoryRecord[],
  responseTimeMs?: number,
): Promise<Buffer> {
  console.log(
    `[renderLogbookTable] Starting render for ${records.length} flights`,
  );
  if (!records.length) throw new Error("No flights to render");

  const header = ["Callsign", "User", "Aircraft", "Time", "Route", "Ldg", "Dur", "Vio"];

  const dataRows = records.map((r) => [
    r.callsign || "",
    r.username || "",
    `${r.aircraft || ""} · ${r.livery || ""}`.trim(),
    fmtDateTime(r.timestamp),
    `${r.origin || ""}-${r.dest || ""}`,
    String(r.landings || 0),
    r.duration || "--:--",
    String(r.violations || 0),
  ]);

  // Store violations for color coding
  const violations = records.map((r) => r.violations || 0);

  // Measure column widths
  const ctxMeasure = createCanvas(10, 10).getContext("2d");
  ctxMeasure.font = FONT;
  const colWidths = header.map((_, col) =>
    Math.ceil(
      Math.max(
        ctxMeasure.measureText(header[col]).width,
        ...dataRows.map((row) => ctxMeasure.measureText(row[col]).width),
      ) + 56,
    ),
  );

  const tableW = colWidths.reduce((a, b) => a + b, 0);

  // Estimate table height
  const estimatedRowH = Math.ceil(14 * 1.5 + 8 * 2);
  const tableH = estimatedRowH * (dataRows.length + 1);
  const footerH = 50;
  const totalH = tableH + footerH;

  // Add 5px padding on all sides
  const PADDING = 5;
  const canvas = createCanvas(tableW + PADDING * 2, totalH + PADDING * 2);
  const ctx = canvas.getContext("2d");

  // Fill background with dark color first
  ctx.fillStyle = BG_ROW;
  ctx.fillRect(0, 0, tableW + PADDING * 2, totalH + PADDING * 2);

  // Translate context to add padding offset
  ctx.translate(PADDING, PADDING);

  // Format data with color coding and alternating row backgrounds
  const styledData = dataRows.map((row, rowIndex) =>
    row.map((cell, colIndex) => {
      // Alternating row background
      const rowBackground = rowIndex % 2 === 0 ? BG_ROW : BG_STRIPE;

      // Color code the "Vio" column (index 7) based on violations
      if (colIndex === 7 && rowIndex < violations.length) {
        return {
          value: cell,
          color: getViolationColor(violations[rowIndex]),
          background: rowBackground,
        };
      }

      return {
        value: cell,
        color: FG_TEXT,
        background: rowBackground,
      };
    }),
  );

  // eslint-disable-next-line @typescript-eslint/ban-ts-comment
  const ct = new CanvasTable(canvas as unknown as any, {
    columns: header.map((title, i) => ({
      title,
      options: {
        minWidth: colWidths[i],
        maxWidth: colWidths[i],
      },
    })),
    data: styledData,
    options: {
      background: BG_ROW,
      borders: {
        header: { width: 1, color: BORDER_COLOUR },
        row: { width: 1, color: BORDER_COLOUR },
        table: { width: 1, color: BORDER_COLOUR },
      },
      header: {
        background: BG_HEADER,
        color: FG_HEADER,
        fontFamily: FONT_FAMILY,
        fontSize: 14,
        fontWeight: "bold",
        padding: 8,
        lineHeight: 1.5,
      },
      cell: {
        color: FG_TEXT,
        background: BG_ROW,
        fontFamily: FONT_FAMILY,
        fontSize: 14,
        fontWeight: "normal",
        padding: 8,
        lineHeight: 1.5,
      },
      fit: false,
      padding: 0,
    },
  } as any);

  await ct.generateTable();

  // Draw footer with metadata
  const footerBgHeight = footerH;
  const footerStart = tableH + 2;
  ctx.fillStyle = "#181825"; // Darker footer background
  ctx.fillRect(0, footerStart, tableW, footerBgHeight);

  // Separator line
  ctx.fillStyle = "#45475a";
  ctx.fillRect(0, footerStart, tableW, 2);

  ctx.font = "bold 13px 'DejaVu Sans Mono'";
  ctx.fillStyle = "#cdd6f4"; // Brighter text
  const now = new Date();
  const timeStr = now.toLocaleTimeString("en-US", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });

  let footerText = `${records.length} flight${records.length !== 1 ? "s" : ""} • Updated at ${timeStr}`;
  if (responseTimeMs !== undefined) {
    footerText += ` • API: ${responseTimeMs}ms`;
  }

  // Add cache update note
  footerText += " • Results updated every 15 mins";

  ctx.fillText(footerText, 15, footerStart + 26);

  return canvas.encode("png");
}
