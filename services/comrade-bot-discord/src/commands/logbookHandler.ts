/**
 * flightHistoryHandler.ts  – wrapper-aware
 */

import {
  ActionRowBuilder,
  AttachmentBuilder,
  ButtonBuilder,
  ButtonInteraction,
  ButtonStyle,
  ChatInputCommandInteraction,
  Client,
  Interaction,
  MessageFlags,
} from "discord.js";

import { ApiService } from "../services/apiService";
import { renderLogbookTable } from "../helpers/LogbookTableRenderer";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { MessageFormatters } from "../helpers/messageFormatter";
import { FlightHistoryPage } from "../types/Responses";
import { UnauthorizedError } from "../helpers/UnauthorizedException";

// ────────────────────────────────────────────────
// Main worker
// ────────────────────────────────────────────────
export async function handleFlightHistory(
  di: DiscordInteraction,
  page: number,
  ifcId = "",
): Promise<void> {
  const chat = di.getChatInputInteraction();
  const btn = di.getButtonInteraction();

  if (!chat && !btn) return;                  // ignore other interactions
  const fromSlash = !!chat;

  // ── 1) IFC-ID (first page only) ─────────────────
  if (fromSlash && !ifcId) {
    ifcId = di.getStringParam("ifc_id", true);
  }

  // ── 2) ACK once ────────────────────────────────
  if (fromSlash) {
    await chat!.deferReply();
  } else {
    await btn!.deferUpdate();
  }

  // ── 3) Fetch data ──────────────────────────────
  // const apiResp = await ApiService.getUserLogbook(di.getMetaInfo(), ifcId, page);

  let apiResp: FlightHistoryPage & { response_time?: string } = { records: [], page: 0, error: "" };  // fallback shape

  try {
    apiResp = await ApiService.getUserLogbook(di.getMetaInfo(), ifcId, page);
  } catch (err: any) {
    if (err instanceof UnauthorizedError) {
      const errorEmbed = {
        title: "Permission Denied",
        description: `❌ You don't have permission to view this logbook.\n\n**Why?**\nOnly Staff and Admin members can view logbooks.\n\n**How to fix:**\nCheck your role with \`/status\`. If you should have access, ask your VA admin to grant Staff permissions.`,
        color: 0xff0000,
        timestamp: new Date().toISOString()
      };
      if (fromSlash) await chat!.editReply({ embeds: [errorEmbed] });
      else await btn!.followUp({ embeds: [errorEmbed] });
      return;
    }

    console.error("Unexpected error while fetching logbook:", err);
    const errorMsg = err?.message || "Unknown error";
    const errorEmbed = {
      title: "Error Fetching Logbook",
      description: `❌ An unexpected error occurred while fetching the logbook.\n\n**Error:** ${errorMsg}\n\n**Please try:**\n• Double-check the IFC username\n• Ensure the pilot is registered with the VA\n• Try again in a few moments\n• Contact support if the issue persists`,
      color: 0xff0000,
      timestamp: new Date().toISOString()
    };
    if (fromSlash) await chat!.editReply({ embeds: [errorEmbed] });
    else await btn!.followUp({ embeds: [errorEmbed] });
    return;
  }

  if (!apiResp?.records?.length) {
    const errorEmbed = {
      title: "No Flight History Found",
      description: `❌ No flights found for **${ifcId}**.\n\n**Possible reasons:**\n• The pilot hasn't registered with this VA yet\n• The IFC username might be incorrect (check spelling)\n• The pilot has no recorded flights yet\n\n**Next steps:**\n• Verify the pilot's IFC Community username (not their display name)\n• Ask the pilot to run \`/register\` first\n• Use \`/status\` to confirm they're linked to this VA`,
      color: 0xff9900,
      timestamp: new Date().toISOString()
    };
    if (fromSlash) await chat!.editReply({ embeds: [errorEmbed] });
    else await btn!.followUp({ embeds: [errorEmbed] });
    return;
  }

  // ── 4) Render PNG with modern theme ─────────────
  // Extract response time (format: "45ms" → 45)
  const responseTimeMs = apiResp.response_time
    ? parseInt(apiResp.response_time.replace("ms", ""), 10)
    : undefined;

  const png = await renderLogbookTable(apiResp.records, responseTimeMs);
  const file = new AttachmentBuilder(png, { name: "logbook.png" });
  const msg = MessageFormatters.makeFlightHistoryTable(apiResp.records)

  // ── 5) Pagination buttons ──────────────────────
  const row = new ActionRowBuilder<ButtonBuilder>();
  if (page > 1) {
    row.addComponents(
      new ButtonBuilder()
        .setCustomId(`flights_prev_${ifcId}_${page - 1}`)
        .setLabel("Previous")
        .setStyle(ButtonStyle.Primary),
    );
  }
  if (apiResp.records.length) {
    row.addComponents(
      new ButtonBuilder()
        .setCustomId(`flights_next_${ifcId}_${page + 1}`)
        .setLabel("Next")
        .setStyle(ButtonStyle.Primary),
    );
  }

  const editPayload = {
    files: [file],
    content: msg,
    components: row.components.length ? [row] : [],
  } as const;

  // ── 6) Edit / update exactly once ───────────────
  if (fromSlash) {
    await chat!.editReply(editPayload);
  } else {
    await btn!.editReply(editPayload);
  }
}

// ────────────────────────────────────────────────
// Router
// ────────────────────────────────────────────────
export function registerLogbookHandlers(client: Client): void {
  client.on("interactionCreate", async (raw: Interaction) => {
    const di = new DiscordInteraction(raw as any);

    try {
      // Slash
      if (raw.isChatInputCommand() && raw.commandName === "logbook") {
        return handleFlightHistory(di, 1);
      }

      // Button
      const btn = di.getButtonInteraction();
      if (btn) {
        const m = btn.customId.match(/^flights_(prev|next)_(\\d+)_(\\d+)$/);
        if (!m) return;
        const [, , ifcId, pageStr] = m;
        return handleFlightHistory(di, Number(pageStr), ifcId);
      }
    } catch (err) {
      console.error("logbook handler error", err);
      if (raw.isRepliable() && !raw.replied && !raw.deferred) {
        const errorEmbed = {
          title: "Error",
          description: "❌ An unexpected error occurred while processing your request.\n\nPlease try again or contact support if the issue persists.",
          color: 0xff0000,
          timestamp: new Date().toISOString()
        };
        await raw.reply({ embeds: [errorEmbed] });
      }
    }
  });
}
