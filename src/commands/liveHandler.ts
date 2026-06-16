/**
 * liveHandler.ts – wrapper-aware handler for live flights with pagination
 */

import {
  ActionRowBuilder,
  ButtonBuilder,
  ButtonStyle,
} from "discord.js";

import { ApiService } from "../services/apiService";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { LiveFlightRecord, LiveFlightsResult } from "../types/Responses";
import { UnauthorizedError } from "../helpers/UnauthorizedException";

const MAX_DISPLAYED_FLIGHTS = 30;
const DISCORD_MESSAGE_LIMIT = 2000;
const MESSAGE_TARGET_LIMIT = 1800;
const MESSAGE_APPEND_LIMIT = 1700;
const CODE_BLOCK_OVERHEAD = 8;
const USERNAME_LIMIT = 24;
const EQUIPMENT_LIMIT = 56;
const ROUTE_LIMIT = 64;

type ReplyTarget = NonNullable<ReturnType<DiscordInteraction["getChatInputInteraction"]>> | NonNullable<ReturnType<DiscordInteraction["getButtonInteraction"]>>;

function truncate(value: string | undefined | null, maxLength: number, fallback: string): string {
  const normalized = value?.trim() || fallback;
  if (normalized.length <= maxLength) return normalized;
  return `${normalized.slice(0, Math.max(0, maxLength - 3))}...`;
}

function formatNumber(value: number | undefined | null): string | undefined {
  if (typeof value !== "number" || Number.isNaN(value)) return undefined;
  return Math.round(value).toLocaleString("en-US");
}

function formatAltitude(value: number | undefined | null): string {
  const altitude = formatNumber(value);
  return altitude ? `${altitude} ft` : "alt unknown";
}

function formatSpeed(value: number | undefined | null): string {
  const speed = formatNumber(value);
  return speed ? `${speed} kt` : "speed unknown";
}

function formatRoute(flight: LiveFlightRecord): string {
  if (flight.route?.trim()) return truncate(flight.route, ROUTE_LIMIT, "Route unknown");
  if (flight.origin?.trim() || flight.destination?.trim()) {
    return truncate(`${flight.origin || "?"} -> ${flight.destination || "?"}`, ROUTE_LIMIT, "Route unknown");
  }
  return "Route unknown";
}

function formatFlightBlock(flight: LiveFlightRecord, displayIndex: number): string {
  const callsign = truncate(flight.callsign, 42, "Unknown callsign");
  const username = truncate(flight.username, USERNAME_LIMIT, "Unknown pilot");
  const aircraft = truncate(flight.aircraft_name, 28, "Unknown aircraft");
  const livery = truncate(flight.livery_name, 24, "Unknown livery");
  const equipment = truncate(`${aircraft} / ${livery}`, EQUIPMENT_LIMIT, "Unknown equipment");
  const statusParts = [
    formatAltitude(flight.altitude),
    formatSpeed(flight.speed),
    truncate(flight.session_name, 18, "",),
    truncate(flight.phase, 18, "",),
  ].filter(Boolean);

  return [
    `${displayIndex}. ${callsign} - ${username}`,
    equipment,
    statusParts.join(" | "),
    formatRoute(flight),
  ].join("\n");
}

function buildHeader(result: LiveFlightsResult | undefined, displayedCount: number, totalFlights: number): string {
  const detected = result?.summary?.total_detected_flights ?? totalFlights;
  const lines = [`Live flights - showing ${displayedCount} of ${detected}`];

  if (result?.message) lines.push(result.message);
  if (result?.summary?.top_route) {
    lines.push(`Top route: ${result.summary.top_route.route} x${result.summary.top_route.count}`);
  }
  if (totalFlights > MAX_DISPLAYED_FLIGHTS) {
    lines.push(`Showing first ${MAX_DISPLAYED_FLIGHTS} flights.`);
  }
  if (result?.code === "SIGNED_LINK_UNAVAILABLE") {
    lines.push("Live map link is temporarily unavailable.");
  }

  return lines.join("\n");
}

function buildContinuationHeader(batchNumber: number): string {
  return `Live flights continued (${batchNumber})`;
}

function codeBlock(content: string): string {
  const safeContent = content.replace(/```/g, "'''");
  return `\`\`\`\n${safeContent.slice(0, MESSAGE_TARGET_LIMIT - CODE_BLOCK_OVERHEAD)}\n\`\`\``;
}

function splitFlightMessages(result: LiveFlightsResult | undefined, flights: LiveFlightRecord[]): string[] {
  const displayedFlights = flights.slice(0, MAX_DISPLAYED_FLIGHTS);
  const messages: string[] = [];
  let current = buildHeader(result, displayedFlights.length, flights.length);
  let batchNumber = 2;

  for (let index = 0; index < displayedFlights.length; index++) {
    const block = formatFlightBlock(displayedFlights[index], index + 1);
    const next = `${current}\n\n${block}`;

    if (next.length > MESSAGE_APPEND_LIMIT && current.length > 0) {
      messages.push(current.slice(0, MESSAGE_TARGET_LIMIT));
      current = `${buildContinuationHeader(batchNumber)}\n\n${block}`;
      batchNumber += 1;
      continue;
    }

    current = next;
  }

  if (current.length > DISCORD_MESSAGE_LIMIT) {
    messages.push(current.slice(0, MESSAGE_TARGET_LIMIT));
  } else {
    messages.push(current);
  }

  return messages;
}

function liveMapComponents(signedLink: string | undefined): ActionRowBuilder<ButtonBuilder>[] {
  if (!signedLink) return [];
  return [new ActionRowBuilder<ButtonBuilder>().addComponents(
    new ButtonBuilder()
      .setLabel("Open Live Map")
      .setStyle(ButtonStyle.Link)
      .setURL(signedLink),
  )];
}

async function editReply(target: ReplyTarget, content: string, signedLink?: string): Promise<void> {
  await target.editReply({
    content: codeBlock(content),
    components: liveMapComponents(signedLink),
  });
}

async function followUp(target: ReplyTarget, content: string): Promise<void> {
  await target.followUp({
    content: codeBlock(content),
    ephemeral: true,
  });
}

function errorMessageFor(err: any): string {
  if (err instanceof UnauthorizedError) {
    return "You need to register before viewing live flights.\nUse `/register` to get started.";
  }

  if (err.code === "MISSING_DISCORD_CONTEXT" || err.code === "VA_CONTEXT_NOT_CONFIGURED") {
    return `Live flights are not configured for this Discord server.\n${err.message || "Ask an admin to finish VA setup."}`;
  }

  if (err.code === "USER_NOT_REGISTERED" || err.code === "FORBIDDEN" || err.status === 403) {
    return `${err.message || "You need to register before viewing live flights."}\nUse \`/register\` to get started.`;
  }

  return "Live flights are temporarily unavailable.\nPlease try again later.";
}

// ────────────────────────────────────────────────
// Main worker
// ────────────────────────────────────────────────
export async function handleLiveFlights(
  di: DiscordInteraction,
  page: number = 1,
): Promise<void> {
  const chat = di.getChatInputInteraction();
  const btn = di.getButtonInteraction();

  if (!chat && !btn) return;                  // ignore other interactions
  const fromSlash = !!chat;
  const target = (chat ?? btn)!;

  // ── 1) ACK once ────────────────────────────────
  if (fromSlash) {
    await chat!.deferReply({ ephemeral: true });
  } else {
    await btn!.deferReply({ ephemeral: true });
  }

  // ── 2) Fetch data ──────────────────────────────
  let flights: LiveFlightRecord[] = [];
  let signedLink: string | undefined;
  let result: (LiveFlightsResult & { responseTimeMs?: number }) | undefined;

  try {
    result = await ApiService.getLiveFlights(di.getMetaInfo());
    flights = result.flights;
    signedLink = result.signed_link;
  } catch (err: any) {
    console.error("[handleLiveFlights] Error fetching flights:", err);
    await editReply(target, errorMessageFor(err));
    return;
  }

  if (!flights || flights.length === 0) {
    await editReply(
      target,
      [
        "No live flights right now.",
        result?.message || "No live flights currently active for this VA.",
      ].join("\n"),
      signedLink,
    );
    return;
  }

  const messages = splitFlightMessages(result, flights);
  await editReply(target, messages[0], signedLink);

  for (const message of messages.slice(1)) {
    await followUp(target, message);
  }
}
