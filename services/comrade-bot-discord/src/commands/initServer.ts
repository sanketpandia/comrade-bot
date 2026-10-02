// src/commands/initServer.ts
import {
  SlashCommandBuilder,
  ActionRowBuilder,
  ButtonBuilder,
  ButtonStyle,
  EmbedBuilder,
  PermissionFlagsBits,
} from "discord.js";
import { CUSTOM_IDS } from "../configs/constants";
import { DiscordInteraction } from "../types/DiscordInteraction";

/* ──────────────────────────────────────────────────────────
   /initserver slash command metadata
   ────────────────────────────────────────────────────────── */
export const data = new SlashCommandBuilder()
  .setName("initserver")
  .setDescription("Bootstrap this Discord server with a VA Code / ID")
  .setDefaultMemberPermissions(PermissionFlagsBits.Administrator);

/* ──────────────────────────────────────────────────────────
   Slash command → show info screen with button
   ────────────────────────────────────────────────────────── */
export async function execute(interaction: DiscordInteraction) {
  const chatInput = interaction.getChatInputInteraction();
  if (!chatInput) return;

  if (!chatInput.guildId) {
    await chatInput.reply({
      content: "❌ `/initserver` must be run inside the Discord server you want to bootstrap. It cannot be used in DMs.",
      ephemeral: true,
    });
    return;
  }

  const infoEmbed = new EmbedBuilder()
    .setColor(0x0099FF)
    .setTitle("🏢 Start VA Setup")
    .setDescription("Bootstrap this Discord server with one field: your VA Code / ID. Detailed setup continues in Vizburo.")
    .addFields(
      {
        name: "📝 VA Code / ID",
        value: "A short identifier for your virtual airline. Example: `IFE`, `DAL`, `UAE`.",
        inline: false,
      },
      {
        name: "🌐 What happens next?",
        value: "After this, open `/dashboard` in a desktop browser or desktop view to set your display name and callsign matching.",
        inline: false,
      },
      {
        name: "✈️ Flight matching",
        value: "Live-flight features start working after staff add either a callsign start or callsign end in Vizburo Basic Setup.",
        inline: false,
      },
    )
    .setFooter({ text: "Click 'Start setup' to enter your VA Code / ID" });

  const proceedButton = new ButtonBuilder()
    .setCustomId(CUSTOM_IDS.INIT_SERVER_PROCEED_BUTTON)
    .setLabel("Start setup")
    .setStyle(ButtonStyle.Primary)
    .setEmoji("▶️");

  const row = new ActionRowBuilder<ButtonBuilder>()
    .addComponents(proceedButton);

  await chatInput.reply({
    embeds: [infoEmbed],
    components: [row],
    ephemeral: true,
  });
}
