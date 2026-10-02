import {
    ActionRowBuilder,
    SlashCommandBuilder,
    ButtonBuilder,
    ButtonStyle,
    EmbedBuilder,
    AttachmentBuilder,
    ChatInputCommandInteraction,
} from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { CUSTOM_IDS } from "../configs/constants";
import * as path from "path";

export const data = new SlashCommandBuilder()
    .setName("register")
    .setDescription("Register to Comrade Bot and link to your Virtual Airline");

/**
 * Opens the registration flow. Politburo validates IFC, flight proof, and duplicate accounts on submit.
 */
export async function execute(interaction: DiscordInteraction) {
    const chatInput = interaction.getChatInputInteraction();
    if (!chatInput) return;

    await showNewUserRegistration(chatInput);
}

async function showNewUserRegistration(chatInput: ChatInputCommandInteraction) {
    const infoEmbed = new EmbedBuilder()
        .setColor(0x0099FF)
        .setTitle("✈️ User Registration")
        .setDescription(
            "Create your global Comrade Bot account with your Infinite Flight Community identity. " +
            "If this Discord server is a VA, you can link with a callsign after your account is created."
        )
        .addFields(
            {
                name: "📝 IFC Username",
                value: "Your Infinite Flight Community username. We do **not** ask for your IF password, email, credentials, or full logbook.",
                inline: false,
            },
            {
                name: "🛫 Last Flight",
                value: "Your most recent complete flight (with origin-destination).\nFormat: `VIDP-VIDP` (4-letter ICAO codes)\n\nPick the highlighted flight from your Infinite Flight logbook (see image below).",
                inline: false,
            },
            {
                name: "⚠️ Privacy & impersonation warning",
                value: "Use only your own IF Community account. The last-flight check helps prevent impersonation.",
                inline: false,
            }
        )
        .setImage("attachment://register_logbook.png")
        .setFooter({ text: "Click 'Proceed' to continue • Already registered? Use 'Link to VA' or /status" });

    const proceedButton = new ButtonBuilder()
        .setCustomId(CUSTOM_IDS.REGISTER_NEW_BUTTON)
        .setLabel("Proceed")
        .setStyle(ButtonStyle.Success)
        .setEmoji("✅");

    const linkButton = new ButtonBuilder()
        .setCustomId(CUSTOM_IDS.REGISTER_LINK_BUTTON)
        .setLabel("Link to VA")
        .setStyle(ButtonStyle.Primary)
        .setEmoji("🔗");

    const row = new ActionRowBuilder<ButtonBuilder>().addComponents(proceedButton, linkButton);

    const imagePath = path.join(__dirname, "../../assets/images/register_logbook.png");
    const attachment = new AttachmentBuilder(imagePath, { name: "register_logbook.png" });

    await chatInput.reply({
        embeds: [infoEmbed],
        files: [attachment],
        components: [row],
        ephemeral: true,
    });
}
