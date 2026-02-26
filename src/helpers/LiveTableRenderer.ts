import { createCanvas, GlobalFonts } from "@napi-rs/canvas";
import { LiveFlightRecord } from "../types/Responses";

// ──────────────────────────────────────────────────────────────
// Constants / styles - Grid Layout Design
// ──────────────────────────────────────────────────────────────

const FONT_FAMILY = "DejaVu Sans Mono";

// Slate color scheme (matching the React design)
const BG_CONTAINER = "#161925";
const BG_HEADER = "#1e293b"; // slate-900
const BG_ROW_EVEN = "#1e293b"; // slate-900
const BG_ROW_ODD = "#1e293b80"; // slate-900/50
const BG_COLUMN_HEADER = "#1e293b80"; // slate-900/50
const BG_FOOTER = "#1e293b"; // slate-900
const FG_TEXT_PRIMARY = "#f1f5f9"; // slate-100
const FG_TEXT_SECONDARY = "#cbd5e1"; // slate-300
const FG_TEXT_TERTIARY = "#64748b"; // slate-500
const FG_ROUTE_ORIGIN = "#ffffff"; // white
const FG_ROUTE_DEST = "#34d399"; // emerald-400
const BORDER_COLOR = "#334155"; // slate-700
const BORDER_DARK = "#1e293b"; // slate-800

// Phase colors (matching React design)
const PHASE_COLORS: Record<string, { bg: string; text: string; border: string }> = {
    cruise: { bg: "#3b82f633", text: "#60a5fa", border: "#3b82f64d" }, // blue-500/20, blue-400, blue-500/30
    climb: { bg: "#10b98133", text: "#34d399", border: "#10b9814d" }, // emerald-500/20, emerald-400, emerald-500/30
    descent: { bg: "#a855f733", text: "#c084fc", border: "#a855f74d" }, // purple-500/20, purple-400, purple-500/30
    landing: { bg: "#f59e0b33", text: "#fbbf24", border: "#f59e0b4d" }, // amber-500/20, amber-400, amber-500/30
    approach: { bg: "#f59e0b33", text: "#fbbf24", border: "#f59e0b4d" }, // amber-500/20, amber-400, amber-500/30
    takeoff: { bg: "#10b98133", text: "#34d399", border: "#10b9814d" }, // emerald-500/20, emerald-400, emerald-500/30
    ground: { bg: "#64748b33", text: "#94a3b8", border: "#64748b4d" }, // slate-500/20, slate-400, slate-500/30
    default: { bg: "#3b82f633", text: "#60a5fa", border: "#3b82f64d" } // blue-500/20, blue-400, blue-500/30
};

// Layout constants
const CANVAS_WIDTH = 1000; // Wider for grid layout
const HEADER_HEIGHT = 80;
const COLUMN_HEADER_HEIGHT = 40;
const ROW_HEIGHT = 90; // More space for full names
const FOOTER_HEIGHT = 40;
const PADDING = 20;
const COLUMN_PADDING = 16;

// Column widths (as percentages, then calculated)
const COL_IDENTITY_WIDTH = 0.33; // 33%
const COL_ROUTE_WIDTH = 0.25; // 25%
const COL_TELEMETRY_WIDTH = 0.25; // 25%
const COL_STATUS_WIDTH = 0.17; // 17% (remaining)

// Pre‑register font (safe‑fail) ------------------------------------------------
try {
    GlobalFonts.registerFromPath(
        "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
        FONT_FAMILY,
    );
} catch { }

// ──────────────────────────────────────────────────────────────
// Helper functions
// ──────────────────────────────────────────────────────────────

function fmtTime(iso: string): string {
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return iso;

    const diff = Date.now() - d.getTime();
    const mins = Math.floor(diff / 60000);

    if (mins < 1) return "Just now";
    if (mins < 60) return `${mins}m ago`;
    const hours = Math.floor(mins / 60);
    if (hours < 24) return `${hours}h ago`;
    return `${Math.floor(hours / 24)}d ago`;
}

function getPhaseStyle(phase: string): { bg: string; text: string; border: string } {
    const p = phase.toLowerCase();
    if (p.includes('land') || p.includes('approach')) return PHASE_COLORS.landing;
    if (p.includes('takeoff') || p.includes('climb')) return PHASE_COLORS.climb;
    if (p.includes('descent')) return PHASE_COLORS.descent;
    if (p.includes('ground')) return PHASE_COLORS.ground;
    return PHASE_COLORS.cruise; // Default to cruise
}

// Truncate text with ellipsis
function truncate(text: string, maxWidth: number, ctx: any): string {
    const width = ctx.measureText(text).width;
    if (width <= maxWidth) return text;

    let truncated = text;
    while (ctx.measureText(truncated + "...").width > maxWidth && truncated.length > 0) {
        truncated = truncated.slice(0, -1);
    }
    return truncated + "...";
}

// ──────────────────────────────────────────────────────────────
// Main API - Grid Layout Design
// ──────────────────────────────────────────────────────────────

export async function renderLiveFlights(records: LiveFlightRecord[], responseTimeMs?: number, page = 1, totalFlights = 0): Promise<Buffer> {
    console.log(`[renderLiveFlights] Starting render for ${records.length} flights`);
    if (!records.length) throw new Error("No flights to render");

    // Calculate column positions
    const colIdentityX = PADDING;
    const colIdentityW = CANVAS_WIDTH * COL_IDENTITY_WIDTH;
    const colRouteX = colIdentityX + colIdentityW;
    const colRouteW = CANVAS_WIDTH * COL_ROUTE_WIDTH;
    const colTelemetryX = colRouteX + colRouteW;
    const colTelemetryW = CANVAS_WIDTH * COL_TELEMETRY_WIDTH;
    const colStatusX = colTelemetryX + colTelemetryW;
    const colStatusW = CANVAS_WIDTH - colStatusX - PADDING;

    // Calculate total height
    const contentHeight = HEADER_HEIGHT + COLUMN_HEADER_HEIGHT + (records.length * ROW_HEIGHT) + FOOTER_HEIGHT;
    const canvas = createCanvas(CANVAS_WIDTH, contentHeight);
    const ctx = canvas.getContext("2d");

    // Fill background
    ctx.fillStyle = BG_CONTAINER;
    ctx.fillRect(0, 0, CANVAS_WIDTH, contentHeight);

    let yPos = 0;

    // ──── HEADER ────────────────────────────────────────────────
    ctx.fillStyle = BG_HEADER;
    ctx.fillRect(0, yPos, CANVAS_WIDTH, HEADER_HEIGHT);

    // Border at bottom of header
    ctx.fillStyle = BORDER_COLOR;
    ctx.fillRect(0, HEADER_HEIGHT - 1, CANVAS_WIDTH, 1);

    // Header left side - "LIVE FLIGHTS"
    ctx.font = `bold 20px '${FONT_FAMILY}'`;
    ctx.fillStyle = FG_TEXT_PRIMARY;
    const headerTextY = yPos + 30;
    ctx.fillText("LIVE FLIGHTS", PADDING, headerTextY);

    // Active pilots count
    ctx.font = `11px '${FONT_FAMILY}'`;
    ctx.fillStyle = FG_TEXT_TERTIARY;
    const displayCount = totalFlights > 0 ? totalFlights : records.length;
    ctx.fillText(`${displayCount} Active Pilots Tracking`, PADDING, headerTextY + 20);

    // "EXPERT SERVER" badge (below active pilots count)
    ctx.font = `bold 11px '${FONT_FAMILY}'`;
    const badgeText = "EXPERT SERVER";
    const badgeTextWidth = ctx.measureText(badgeText).width;
    const badgePadding = 8;
    const badgeWidth = badgeTextWidth + (badgePadding * 2);
    const badgeX = PADDING;
    const badgeY = headerTextY + 32;
    const badgeHeight = 20;

    // Badge background
    ctx.fillStyle = "#3b82f61a"; // blue-500/10
    ctx.fillRect(badgeX, badgeY, badgeWidth, badgeHeight);
    // Badge border
    ctx.strokeStyle = "#3b82f633"; // blue-500/20
    ctx.lineWidth = 1;
    ctx.strokeRect(badgeX, badgeY, badgeWidth, badgeHeight);
    // Badge text
    ctx.fillStyle = "#60a5fa"; // blue-400
    ctx.fillText(badgeText, badgeX + badgePadding, badgeY + 14);

    // Header right side - System time
    ctx.font = `bold 24px '${FONT_FAMILY}'`;
    ctx.fillStyle = FG_TEXT_SECONDARY;
    const now = new Date();
    const timeStr = now.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false });
    const timeWidth = ctx.measureText(timeStr).width;
    ctx.fillText(timeStr, CANVAS_WIDTH - timeWidth - PADDING, headerTextY);
    
    ctx.font = `9px '${FONT_FAMILY}'`;
    ctx.fillStyle = FG_TEXT_TERTIARY;
    ctx.fillText("UTC", CANVAS_WIDTH - timeWidth - PADDING, headerTextY + 12);
    
    ctx.font = `10px '${FONT_FAMILY}'`;
    ctx.fillText("SYSTEM TIME", CANVAS_WIDTH - timeWidth - PADDING, headerTextY + 25);

    yPos = HEADER_HEIGHT;

    // ──── COLUMN HEADERS ────────────────────────────────────────
    ctx.fillStyle = BG_COLUMN_HEADER;
    ctx.fillRect(0, yPos, CANVAS_WIDTH, COLUMN_HEADER_HEIGHT);

    // Border at bottom
    ctx.fillStyle = BORDER_COLOR;
    ctx.fillRect(0, yPos + COLUMN_HEADER_HEIGHT - 1, CANVAS_WIDTH, 1);

    ctx.font = `bold 10px '${FONT_FAMILY}'`;
    ctx.fillStyle = FG_TEXT_TERTIARY;
    const colHeaderY = yPos + 25;

    ctx.fillText("IDENTIFICATION", colIdentityX + COLUMN_PADDING, colHeaderY);
    ctx.fillText("ROUTE", colRouteX + COLUMN_PADDING, colHeaderY);
    ctx.fillText("TELEMETRY", colTelemetryX + COLUMN_PADDING, colHeaderY);
    ctx.fillText("STATUS", colStatusX + COLUMN_PADDING, colHeaderY);

    yPos += COLUMN_HEADER_HEIGHT;

    // ──── FLIGHT ROWS ────────────────────────────────────────────
    records.forEach((flight, idx) => {
        const bgColor = idx % 2 === 0 ? BG_ROW_EVEN : BG_ROW_ODD;

        // Row background
        ctx.fillStyle = bgColor;
        ctx.fillRect(0, yPos, CANVAS_WIDTH, ROW_HEIGHT);

        // Border at bottom
        ctx.fillStyle = BORDER_COLOR;
        ctx.fillRect(0, yPos + ROW_HEIGHT - 1, CANVAS_WIDTH, 1);

        // Left border (color-coded by phase) - subtle
        const phaseStyle = getPhaseStyle(flight.phase);
        ctx.fillStyle = phaseStyle.text;
        ctx.globalAlpha = 0.3;
        ctx.fillRect(0, yPos, 3, ROW_HEIGHT);
        ctx.globalAlpha = 1.0;

        const rowCenterY = yPos + Math.floor(ROW_HEIGHT / 2);

        // ──── COLUMN 1: IDENTITY ─────────────────────────────────
        const identityX = colIdentityX + COLUMN_PADDING;
        let identityY = yPos + 18;

        // Livery name (small, uppercase)
        ctx.font = `bold 10px '${FONT_FAMILY}'`;
        ctx.fillStyle = FG_TEXT_TERTIARY;
        const liveryName = flight.livery_name.toUpperCase();
        ctx.fillText(liveryName, identityX, identityY);

        // Aircraft name (below livery, same font style)
        identityY += 14;
        const aircraftMaxWidth = colIdentityW - (COLUMN_PADDING * 2) - 20;
        const aircraft = truncate(flight.aircraft_name, aircraftMaxWidth, ctx);
        ctx.fillText(aircraft, identityX, identityY);

        // Callsign (large, bold)
        identityY += 22;
        ctx.font = `bold 18px '${FONT_FAMILY}'`;
        ctx.fillStyle = FG_TEXT_PRIMARY;
        const callsignMaxWidth = colIdentityW - (COLUMN_PADDING * 2) - 20;
        const callsign = truncate(flight.callsign, callsignMaxWidth, ctx);
        ctx.fillText(callsign, identityX, identityY);

        // Username (small)
        identityY += 18;
        ctx.font = `11px '${FONT_FAMILY}'`;
        ctx.fillStyle = FG_TEXT_SECONDARY;
        ctx.fillText(`@${flight.username}`, identityX, identityY);

        // ──── COLUMN 2: ROUTE ────────────────────────────────────
        const routeX = colRouteX + COLUMN_PADDING;
        const routeY = rowCenterY;

        // Handle missing origin/destination gracefully
        const origin = flight.origin || "N/A";
        const destination = flight.destination || "N/A";
        const hasRoute = flight.origin && flight.destination;

        // Origin (large, monospace, white)
        ctx.font = `bold 20px '${FONT_FAMILY}'`;
        ctx.fillStyle = hasRoute ? FG_ROUTE_ORIGIN : FG_TEXT_TERTIARY;
        const originWidth = ctx.measureText(origin).width;
        ctx.fillText(origin, routeX, routeY);

        // Arrow/separator (only show if both origin and destination exist)
        if (hasRoute) {
            const arrowX = routeX + originWidth + 12;
            ctx.font = `14px '${FONT_FAMILY}'`;
            ctx.fillStyle = FG_TEXT_TERTIARY;
            ctx.fillText("→", arrowX, routeY);

            // Destination (large, monospace, green)
            const destX = arrowX + 20;
            ctx.font = `bold 20px '${FONT_FAMILY}'`;
            ctx.fillStyle = FG_ROUTE_DEST;
            ctx.fillText(destination, destX, routeY);
        } else if (flight.origin && !flight.destination) {
            // Only origin available
            const arrowX = routeX + originWidth + 12;
            ctx.font = `14px '${FONT_FAMILY}'`;
            ctx.fillStyle = FG_TEXT_TERTIARY;
            ctx.fillText("→", arrowX, routeY);
            
            const destX = arrowX + 20;
            ctx.font = `bold 20px '${FONT_FAMILY}'`;
            ctx.fillStyle = FG_TEXT_TERTIARY;
            ctx.fillText("N/A", destX, routeY);
        } else if (!flight.origin && flight.destination) {
            // Only destination available
            const arrowX = routeX + 12;
            ctx.font = `14px '${FONT_FAMILY}'`;
            ctx.fillStyle = FG_TEXT_TERTIARY;
            ctx.fillText("→", arrowX, routeY);
            
            const destX = arrowX + 20;
            ctx.font = `bold 20px '${FONT_FAMILY}'`;
            ctx.fillStyle = FG_ROUTE_DEST;
            ctx.fillText(destination, destX, routeY);
        }

        // ──── COLUMN 3: TELEMETRY ────────────────────────────────
        const telemetryX = colTelemetryX + COLUMN_PADDING;
        let telemetryY = yPos + 18;

        // Altitude and Speed (highlighted)
        ctx.font = `bold 12px '${FONT_FAMILY}'`;
        ctx.fillStyle = FG_TEXT_PRIMARY; // Highlighted color

        const altText = `${flight.altitude.toLocaleString()}ft`;
        ctx.fillText(altText, telemetryX, telemetryY);

        const altWidth = ctx.measureText(altText).width;
        const spdText = `${flight.speed}kts`;
        ctx.fillText(spdText, telemetryX + altWidth + 16, telemetryY);

        // Vertical Speed (below altitude/speed)
        telemetryY += 18;
        ctx.font = `11px '${FONT_FAMILY}'`;
        ctx.fillStyle = FG_TEXT_TERTIARY;
        const verticalSpeed = flight.vertical_speed || 0;
        const vsFpm = Math.round(verticalSpeed); // Already in fpm from API
        const vsArrow = verticalSpeed > 0 ? "↑" : verticalSpeed < 0 ? "↓" : "→";
        const vsText = `${vsArrow} ${Math.abs(vsFpm).toLocaleString()}fpm`;
        ctx.fillText(vsText, telemetryX, telemetryY);

        // FPL Fetch
        telemetryY += 16;
        ctx.font = `10px '${FONT_FAMILY}'`;
        ctx.fillStyle = FG_TEXT_TERTIARY;
        const fplFetchLabel = "FPL Fetch: ";
        ctx.fillText(fplFetchLabel, telemetryX, telemetryY);
        
        const fplFetchLabelWidth = ctx.measureText(fplFetchLabel).width;
        if (flight.last_flight_plan_fetch && flight.last_flight_plan_fetch !== "0001-01-01T00:00:00Z") {
            const fplTime = fmtTime(flight.last_flight_plan_fetch);
            ctx.fillText(fplTime, telemetryX + fplFetchLabelWidth, telemetryY);
        } else {
            ctx.fillText("-", telemetryX + fplFetchLabelWidth, telemetryY);
        }

        // T/O time (est.)
        telemetryY += 14;
        const toTimeLabel = "T/O time (est.): ";
        ctx.fillText(toTimeLabel, telemetryX, telemetryY);
        
        const toTimeLabelWidth = ctx.measureText(toTimeLabel).width;
        if (flight.takeoff_time && flight.takeoff_time !== "0001-01-01T00:00:00Z") {
            const toTime = fmtTime(flight.takeoff_time);
            ctx.fillText(toTime, telemetryX + toTimeLabelWidth, telemetryY);
        } else {
            ctx.fillText("-", telemetryX + toTimeLabelWidth, telemetryY);
        }

        // ──── COLUMN 4: STATUS ────────────────────────────────────
        const statusX = colStatusX + COLUMN_PADDING;
        const statusY = yPos + 18;

        // Phase badge
        const phaseText = flight.phase.toUpperCase();
        ctx.font = `bold 10px '${FONT_FAMILY}'`;
        const phaseTextWidth = ctx.measureText(phaseText).width;
        const phaseBadgePadding = 10;
        const phaseBadgeWidth = phaseTextWidth + (phaseBadgePadding * 2);
        const phaseBadgeX = statusX;
        const phaseBadgeY = statusY;
        const phaseBadgeHeight = 20;

        // Badge background
        ctx.fillStyle = phaseStyle.bg;
        ctx.fillRect(phaseBadgeX, phaseBadgeY, phaseBadgeWidth, phaseBadgeHeight);
        // Badge border
        ctx.strokeStyle = phaseStyle.border;
        ctx.lineWidth = 1;
        ctx.strokeRect(phaseBadgeX, phaseBadgeY, phaseBadgeWidth, phaseBadgeHeight);
        // Badge text
        ctx.fillStyle = phaseStyle.text;
        ctx.fillText(phaseText, phaseBadgeX + phaseBadgePadding, phaseBadgeY + 14);

        // Last updated time
        let statusTextY = statusY + 28;
        ctx.font = `10px '${FONT_FAMILY}'`;
        ctx.fillStyle = FG_TEXT_TERTIARY;
        const lastSeen = fmtTime(flight.last_report);
        ctx.fillText(lastSeen, statusX, statusTextY);

        // Max Altitude
        statusTextY += 16;
        const maxAltLabel = "Max Alt: ";
        ctx.fillText(maxAltLabel, statusX, statusTextY);
        
        const maxAltLabelWidth = ctx.measureText(maxAltLabel).width;
        if (flight.max_altitude !== undefined && flight.max_altitude !== null) {
            const maxAltText = `${flight.max_altitude.toLocaleString()}ft`;
            ctx.fillText(maxAltText, statusX + maxAltLabelWidth, statusTextY);
        } else {
            ctx.fillText("-", statusX + maxAltLabelWidth, statusTextY);
        }

        // Max Speed
        statusTextY += 14;
        const maxSpdLabel = "Max Spd: ";
        ctx.fillText(maxSpdLabel, statusX, statusTextY);
        
        const maxSpdLabelWidth = ctx.measureText(maxSpdLabel).width;
        if (flight.max_speed !== undefined && flight.max_speed !== null) {
            const maxSpdText = `${flight.max_speed}kts`;
            ctx.fillText(maxSpdText, statusX + maxSpdLabelWidth, statusTextY);
        } else {
            ctx.fillText("-", statusX + maxSpdLabelWidth, statusTextY);
        }

        yPos += ROW_HEIGHT;
    });

    // ──── FOOTER ────────────────────────────────────────────────
    // Top border
    ctx.fillStyle = BORDER_COLOR;
    ctx.fillRect(0, yPos, CANVAS_WIDTH, 1);

    // Footer background
    ctx.fillStyle = BG_FOOTER;
    ctx.fillRect(0, yPos, CANVAS_WIDTH, FOOTER_HEIGHT);

    // Footer text (left and right)
    ctx.font = `10px '${FONT_FAMILY}'`;
    ctx.fillStyle = FG_TEXT_TERTIARY;
    const footerY = yPos + Math.floor(FOOTER_HEIGHT / 2) + 3;

    // Left side - Bot ID
    ctx.fillText("BOT-ID: 8829-AFX", PADDING, footerY);

    // Right side - API latency
    let rightText = "";
    if (responseTimeMs !== undefined) {
        rightText = `API LATENCY: ${responseTimeMs}ms`;
    } else {
        rightText = "API LATENCY: --ms";
    }
    if (totalFlights > records.length) {
        rightText += ` • Page ${page}`;
    }
    const rightTextWidth = ctx.measureText(rightText).width;
    ctx.fillText(rightText, CANVAS_WIDTH - rightTextWidth - PADDING, footerY);

    return canvas.encode("png");
}
