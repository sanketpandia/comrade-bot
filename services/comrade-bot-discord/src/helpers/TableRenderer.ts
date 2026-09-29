/*
 * flightHistoryRenderer.ts
 * --------------------------------------
 * Purpose‑built image renderer for **Flight History** tables.
 * Generates a PNG that you can attach in Discord instead of a monospace code‑block.
 *
 * Build deps (install once in your bot project):
 *   npm i @napi-rs/canvas canvas-table
 *
 * This version suppresses all TypeScript type‑mismatch noise that stems from
 * the fact that `canvas-table` typings are written for the old `canvas` package
 * and expose only a subset of runtime options.  We cast the config object to
 * `any` and annotate hook parameters as `any`, so `tsc` will compile without
 * `--noImplicitAny` complaints while runtime remains unchanged.
 */

import { createCanvas, GlobalFonts } from "@napi-rs/canvas";
import { CanvasTable } from "canvas-table";
import { FlightHistoryRecord } from "../types/Responses";


// ──────────────────────────────────────────────────────────────
// Constants / styles
// ──────────────────────────────────────────────────────────────

const ROW_H = 28;
const FONT_FAMILY = "DejaVu Sans Mono";
const FONT = `14px '${FONT_FAMILY}'`;
const BG_HEADER = "#1d3557";
const FG_HEADER = "#ffffff";
const BG_STRIPE = "#f5f5f5";
const BG_VIOLATION = "#ffe5e5";
const BORDER_COLOUR = "#a9a9a9";

// Pre‑register font (safe‑fail) ------------------------------------------------
try {
  GlobalFonts.registerFromPath(
    "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
    FONT_FAMILY,
  );
} catch {/* ignore if font missing */ }

// ──────────────────────────────────────────────────────────────
// Helper: timestamp → "yyyy‑mm‑dd HH:MM"
// ──────────────────────────────────────────────────────────────
function fmtTime(iso: string) {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toISOString().replace("T", " ").slice(0, 16);
}

// ──────────────────────────────────────────────────────────────
// Main API
// ──────────────────────────────────────────────────────────────

export async function renderFlightHistory(records: FlightHistoryRecord[]): Promise<Buffer> {
  if (records.length === 0) throw new Error("No flight records to render");

  // Column order expected by the user
  const header = ["Callsign", "Srv", "Aircraft", "Time", "Ldg", "Dur", "Route"];

  const dataRows: string[][] = [];
  const violationIdx: number[] = [];  // 1‑based idx of rows with >0 violations


  records.forEach((r, i) => {
    const aircraft = r.aircraft ?? r.equipment?.split(" ")[0] ?? "";
    const compound = `${aircraft} · ${r.livery ?? ""}`.trim();

    const origin = r.origin ?? "";
    const dest = r.dest ?? "";
    const route = origin && dest
      ? `${origin}-${dest}`
      : origin + "-" || "-" + dest; // one of them, if only one is set


    dataRows.push([
      r.callsign,
      r.server[0],               // E / T / C
      compound,
      fmtTime(r.timestamp),
      String(r.landings),
      r.duration,
      `${r.origin}-${r.dest}`
    ]);

    if (r.violations && r.violations > 0) violationIdx.push(i + 1);
  });

  // Measure & layout
  const ctxMeasure = createCanvas(10, 10).getContext("2d");
  ctxMeasure.font = FONT;
  const colWidths = header.map((_, col) => {
    const widest = Math.max(
      ctxMeasure.measureText(header[col]).width,
      ...dataRows.map(r => ctxMeasure.measureText(r[col]).width),
    );
    return Math.ceil(widest + 16);
  });
  const tableW = colWidths.reduce((a, b) => a + b, 0);
  const tableH = ROW_H * (dataRows.length + 1);

  const legendNeeded = violationIdx.length > 0;
  const legendH = legendNeeded ? 34 : 0;

  const canvas = createCanvas(tableW, tableH + legendH);

  // The second arg config object is cast to `any` so TS won't bark about
  // unknown props like fontFamily, headerBackground, etc.
  // eslint-disable-next-line @typescript-eslint/ban-ts-comment
  const ct = new CanvasTable(canvas as unknown as any, {
    fontFamily: FONT_FAMILY,
    fontSize: 14,
    rowHeight: ROW_H,
    lineWidth: 1,
    borderColor: BORDER_COLOUR,
    headerBackground: BG_HEADER,
    headerColor: FG_HEADER,
    stripBackground1: BG_STRIPE,
    stripBackground2: "#ffffff",
    columns: header.map((t, i) => ({ title: t, width: colWidths[i] })),
    data: dataRows,
    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
    drawCellHook: ({ row, ctx, bounds }: any) => {
      if (row.isHeader) return;
      const rel = row.rowIndex + 1; // 1‑based relative to dataRows
      if (violationIdx.includes(rel) && bounds.colIndex === 0) {
        ctx.fillStyle = BG_VIOLATION;
        ctx.fillRect(0, bounds.y, tableW, ROW_H);
      }
    },
  } as any);
  await ct.generateTable();

  // Legend -----------------------------------------------------
  if (legendNeeded) {
    const ctx = canvas.getContext("2d");
    ctx.font = FONT;
    ctx.textBaseline = "middle";

    const sq = 18;
    const y = tableH + 10;
    ctx.fillStyle = BG_VIOLATION;
    ctx.fillRect(0, y, sq, sq);
    ctx.strokeStyle = BORDER_COLOUR;
    ctx.strokeRect(0, y, sq, sq);
    ctx.fillStyle = "#000";
    ctx.fillText("Flights with violations", sq + 8, y + sq / 2);
  }

  return canvas.encode("png");
}

// Example usage (replace old ASCII table):
/*
import { AttachmentBuilder, MessageFlags } from "discord.js";

const img = await renderFlightHistory(apiResp.data.records);
await interaction.reply({
  files: [new AttachmentBuilder(img, { name: "logbook.png" })],
  components,          // your pagination buttons
  flags: MessageFlags.Ephemeral,
});
*/
