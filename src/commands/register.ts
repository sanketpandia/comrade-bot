import {
    ActionRowBuilder,
    SlashCommandBuilder,
    ButtonBuilder,
    ButtonStyle,
    EmbedBuilder,
    AttachmentBuilder,
} from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { ApiService } from "../services/apiService";
import * as path from "path";

export const data = new SlashCommandBuilder()
    .setName("register")
    .setDescription("Register to Comrade Bot and link to your Virtual Airline");

/**
 * Shows registration info screen with proceed button
 * Checks user status first to show contextual message and appropriate action
 */
export async function execute(interaction: DiscordInteraction) {
    const chatInput = interaction.getChatInputInteraction();
    if (!chatInput) return;

    // Check user's current status
    let userDetails;
    try {
        userDetails = await ApiService.getUserDetails(interaction.getMetaInfo());
    } catch (error) {
        // User not registered or API error - continue with new user registration flow
        console.log("[Register] User details fetch failed, assuming new user:", error);
    }

    const isRegistered = userDetails?.is_active || false;
    const isLinkedToVA = userDetails?.current_va?.is_member || false;

    // SCENARIO 1: User is already registered AND linked to this VA
    if (isRegistered && isLinkedToVA && userDetails) {
        const successEmbed = new EmbedBuilder()
            .setColor(0x00FF00)
            .setTitle("✅ Already Registered & Linked!")
            .setDescription(`You're all set up!`)
            .addFields(
                {
                    name: "📝 IFC Username",
                    value: userDetails.if_community_id,
                    inline: true
                },
                {
                    name: "🎖️ Role",
                    value: userDetails.current_va.role.charAt(0).toUpperCase() + userDetails.current_va.role.slice(1),
                    inline: true
                }
            )
            .setFooter({ text: "Use /help for more info • Contact staff to change callsign" });

        await chatInput.reply({
            embeds: [successEmbed],
            ephemeral: true
        });
        return;
    }

    // SCENARIO 2: User is registered but NOT linked to this VA
    if (isRegistered && !isLinkedToVA && userDetails) {
        const linkEmbed = new EmbedBuilder()
            .setColor(0xFFA500)
            .setTitle("🔗 Link to Virtual Airline")
            .setDescription(`You're registered as **${userDetails.if_community_id}**, but not linked to this VA.\n\nEnter your callsign number (1-5 digits) to link.`)
            .addFields(
                {
                    name: "📌 Note",
                    value: "Callsign is locked after you set it. Contact staff if you need to change it.",
                    inline: false
                }
            )
            .setFooter({ text: "Click 'Link to VA' to proceed" });

        // Create link button
        const linkButton = new ButtonBuilder()
            .setCustomId("register_link")
            .setLabel("Link to VA")
            .setStyle(ButtonStyle.Primary)
            .setEmoji("🔗");

        const row = new ActionRowBuilder<ButtonBuilder>()
            .addComponents(linkButton);

        await chatInput.reply({
            embeds: [linkEmbed],
            components: [row],
            ephemeral: true
        });
        return;
    }

    // SCENARIO 3: User is NOT registered - show new registration flow
    await showNewUserRegistration(chatInput);
}

/**
 * Shows new user registration info and button
 */
async function showNewUserRegistration(chatInput: any) {
    const infoEmbed = new EmbedBuilder()
        .setColor(0x0099FF)
        .setTitle("✈️ User Registration")
        .setDescription("Register with Comrade Bot using your Infinite Flight Community account. If this server is a Virtual Airline, callsign linking will be available after registration.")
        .addFields(
            {
                name: "📝 IFC Username",
                value: "Your Infinite Flight Community login username.\nExample: `john_doe123`",
                inline: false
            },
            {
                name: "🛫 Last Flight",
                value: "Your most recent flight (origin-destination).\nFormat: `EGLL-KSEA` (4-letter ICAO codes)\n\nPick the highlighted flight from your Infinite Flight logbook (see image below).",
                inline: false
            }
        )
        .setImage("attachment://register_logbook.png")
        .setFooter({ text: "Click 'Proceed' to continue" });

    // Create proceed button for new registration
    const proceedButton = new ButtonBuilder()
        .setCustomId("register_new")
        .setLabel("Proceed")
        .setStyle(ButtonStyle.Success)
        .setEmoji("✅");

    const row = new ActionRowBuilder<ButtonBuilder>()
        .addComponents(proceedButton);

    // Load the logbook reference image
    const imagePath = path.join(__dirname, "../../docs/images/register_logbook.png");
    const attachment = new AttachmentBuilder(imagePath, { name: "register_logbook.png" });

    await chatInput.reply({
        embeds: [infoEmbed],
        files: [attachment],
        components: [row],
        ephemeral: true
    });
}