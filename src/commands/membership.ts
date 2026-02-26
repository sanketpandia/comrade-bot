import {
    ActionRowBuilder,
    SlashCommandBuilder,
    ButtonBuilder,
    ButtonStyle,
    EmbedBuilder,
} from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { CUSTOM_IDS } from "../configs/constants";

export const data = new SlashCommandBuilder()
    .setName("membership")
    .setDescription("Manage your Virtual Airline membership")
    .addSubcommand(subcommand =>
        subcommand
            .setName("join")
            .setDescription("Join this Virtual Airline with your callsign")
    );

/**
 * Handle /membership command with subcommands
 */
export async function execute(interaction: DiscordInteraction) {
    const chatInput = interaction.getChatInputInteraction();
    if (!chatInput) return;

    const subcommand = chatInput.options.getSubcommand();

    switch (subcommand) {
        case "join":
            await handleJoinSubcommand(chatInput);
            break;
        default:
            await chatInput.reply({
                content: "Unknown subcommand",
                ephemeral: true
            });
    }
}

/**
 * Handle /membership join subcommand
 * Shows info screen with callsign explanation and proceed button
 */
async function handleJoinSubcommand(chatInput: any) {
    const infoEmbed = new EmbedBuilder()
        .setColor(0x0099FF)
        .setTitle("✈️ Join Virtual Airline")
        .setDescription("Ready to join this Virtual Airline? You'll need to provide your callsign to continue.")
        .addFields(
            {
                name: "📋 What is a Callsign?",
                value: "Your callsign is your unique identifier within the Virtual Airline. It must match EXACTLY what's registered in the VA database (e.g., Airtable, spreadsheets, etc.).",
                inline: false
            },
            {
                name: "⚠️ Important",
                value: "• Callsign must match exactly (case-sensitive)\n• You must already be registered to Comrade Bot (`/register` first)\n• Contact VA staff if you're unsure of your callsign",
                inline: false
            }
        )
        .setFooter({ text: "Click 'Proceed' when ready" });

    const proceedButton = new ButtonBuilder()
        .setCustomId(CUSTOM_IDS.MEMBERSHIP_JOIN_BUTTON)
        .setLabel("Proceed")
        .setStyle(ButtonStyle.Success)
        .setEmoji("✅");

    const row = new ActionRowBuilder<ButtonBuilder>()
        .addComponents(proceedButton);

    await chatInput.reply({
        embeds: [infoEmbed],
        components: [row],
        ephemeral: true
    });
}
