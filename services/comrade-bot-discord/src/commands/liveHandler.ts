/**
 * liveHandler.ts – wrapper-aware handler for live flights with pagination
 */

import {
  ActionRowBuilder,
  AttachmentBuilder,
  ButtonBuilder,
  ButtonInteraction,
  ButtonStyle,
  ChatInputCommandInteraction,
} from "discord.js";

import { ApiService } from "../services/apiService";
import { renderLiveFlights } from "../helpers/LiveTableRenderer";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { LiveFlightRecord } from "../types/Responses";
import { UnauthorizedError } from "../helpers/UnauthorizedException";

const FLIGHTS_PER_PAGE = 15;

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

  // ── 1) ACK once ────────────────────────────────
  if (fromSlash) {
    await chat!.deferReply({ ephemeral: true });
  } else {
    await btn!.deferReply({ ephemeral: true });
  }

  // ── 2) Fetch data ──────────────────────────────
  let flights: LiveFlightRecord[] = [];
  let responseTimeMs: number | undefined;
  let apiResponseTime: string | undefined;
  let signedLink: string | undefined;

  try {
    const startTime = Date.now();
    const result = await ApiService.getLiveFlights(di.getMetaInfo());
    responseTimeMs = Date.now() - startTime;
    flights = result.flights;
    apiResponseTime = result.responseTime;
    signedLink = result.signedLink;
  } catch (err: any) {
    console.error("[handleLiveFlights] Error fetching flights:", err);

    if (err instanceof UnauthorizedError) {
      const errorEmbed = {
        title: "Not Registered",
        description: "❌ You must be registered to view live flights.\n\nUse `/register` to get started.",
        color: 0xff0000,
        timestamp: new Date().toISOString()
      };
      if (fromSlash) await chat!.editReply({ embeds: [errorEmbed] });
      else await btn!.followUp({ embeds: [errorEmbed], ephemeral: true });
      return;
    }

    // Check for 403 Forbidden (not registered/authorized)
    if (err.message?.includes("403") || err.message?.includes("Forbidden")) {
      const errorEmbed = {
        title: "Not Registered",
        description: "❌ You must be registered to view live flights.\n\nUse `/register` to get started.",
        color: 0xff0000,
        timestamp: new Date().toISOString()
      };
      if (fromSlash) await chat!.editReply({ embeds: [errorEmbed] });
      else await btn!.followUp({ embeds: [errorEmbed], ephemeral: true });
      return;
    }

    // Generic error
    const errorEmbed = {
      title: "Error",
      description: "❌ Failed to fetch live flights.\n\nPlease try again later or contact support.",
      color: 0xff0000,
      timestamp: new Date().toISOString()
    };
    if (fromSlash) await chat!.editReply({ embeds: [errorEmbed] });
    else await btn!.followUp({ embeds: [errorEmbed], ephemeral: true });
    return;
  }

  if (!flights || flights.length === 0) {
    const errorEmbed = {
      title: "No Live Flights",
      description: "No live flights currently active for this VA.",
      color: 0xff9900,
      timestamp: new Date().toISOString()
    };
    if (fromSlash) await chat!.editReply({ embeds: [errorEmbed] });
    else await btn!.editReply({ embeds: [errorEmbed] });
    return;
  }

  // ── 3) Paginate client-side ────────────────────
  const totalFlights = flights.length;
  const totalPages = Math.ceil(totalFlights / FLIGHTS_PER_PAGE);
  const currentPage = Math.max(1, Math.min(page, totalPages));
  
  const startIdx = (currentPage - 1) * FLIGHTS_PER_PAGE;
  const endIdx = startIdx + FLIGHTS_PER_PAGE;
  const pageFlights = flights.slice(startIdx, endIdx);

  // ── 4) Render PNG ──────────────────────────────
  // Parse response_time from API if available (format: "3ms" → 3)
  let parsedResponseTime: number | undefined = responseTimeMs;
  if (apiResponseTime) {
    const match = apiResponseTime.match(/(\d+)ms?/);
    if (match) {
      parsedResponseTime = parseInt(match[1], 10);
    }
  }

  console.log(`[handleLiveFlights] Rendering page ${currentPage}/${totalPages} (${pageFlights.length} flights)`);
  const png = await renderLiveFlights(pageFlights, parsedResponseTime, currentPage, totalFlights);
  const file = new AttachmentBuilder(png, { name: "live-flights.png" });

  // ── 5) Pagination buttons ──────────────────────
  const row = new ActionRowBuilder<ButtonBuilder>();
  if (currentPage > 1) {
    row.addComponents(
      new ButtonBuilder()
        .setCustomId(`live_prev_${currentPage - 1}`)
        .setLabel("Previous")
        .setStyle(ButtonStyle.Primary),
    );
  }
  if (currentPage < totalPages) {
    row.addComponents(
      new ButtonBuilder()
        .setCustomId(`live_next_${currentPage + 1}`)
        .setLabel("Next")
        .setStyle(ButtonStyle.Primary),
    );
  }
  // Add "See Map" button if signed link is available
  if (signedLink) {
    row.addComponents(
      new ButtonBuilder()
        .setLabel("See Map")
        .setStyle(ButtonStyle.Link)
        .setURL(signedLink),
    );
  }

  const editPayload = {
    files: [file],
    components: row.components.length ? [row] : [],
  } as const;

  // ── 6) Edit / update exactly once ───────────────
  if (fromSlash) {
    await chat!.editReply(editPayload);
  } else {
    await btn!.editReply(editPayload);
  }
}
