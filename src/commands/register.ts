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
import { CUSTOM_IDS } from "../configs/constants";
import * as path from "path";
import { UserDetailsData } from "../types/Responses";

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

    // Check user's current status. The backend represents normal onboarding
    // states explicitly, so do not infer an unregistered user from 404s.
    let userDetails: UserDetailsData;
    try {
        userDetails = await ApiService.getUserDetails(interaction.getMetaInfo());
    } catch (error) {
        await chatInput.reply({
            content: "⚠️ I couldn't check your registration status right now. Please try again later.",
            ephemeral: true,
        });
        return;
    }

    const isRegistered = userDetails.is_registered;
    const isLinkedToVA = userDetails.current_va?.is_member || false;
    const currentServerIsVA = userDetails.current_server?.is_configured_va || false;

    // SCENARIO 1: User is already registered AND linked to this VA
    if (isRegistered && currentServerIsVA && isLinkedToVA) {
        const successEmbed = new EmbedBuilder()
            .setColor(0x00FF00)
            .setTitle("✅ Already Registered & Linked!")
            .setDescription(`You're all set up for **${userDetails.current_server.va_name}**.`)
            .addFields(
                {
                    name: "📝 IFC Username",
                    value: userDetails.if_community_id ?? "Registered",
                    inline: true
                },
                ...(userDetails.current_va.callsign ? [{ name: "📻 Callsign", value: userDetails.current_va.callsign, inline: true }] : []),
                {
                    name: "🎖️ Role",
                    value: userDetails.current_va.role ? userDetails.current_va.role.charAt(0).toUpperCase() + userDetails.current_va.role.slice(1) : "Member",
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

    // SCENARIO 2: User is registered, current server is a VA, but user is not linked to it.
    if (isRegistered && currentServerIsVA && !isLinkedToVA) {
        const linkEmbed = new EmbedBuilder()
            .setColor(0xFFA500)
            .setTitle("🔗 Link to Virtual Airline")
            .setDescription(`You're registered as **${userDetails.if_community_id ?? "a Comrade Bot user"}**. This flow will only link your account to **${userDetails.current_server.va_name}**.`)
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
            .setCustomId(CUSTOM_IDS.REGISTER_LINK_BUTTON)
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

    // SCENARIO 3: User is registered, but this server is not configured as a VA.
    if (isRegistered && !currentServerIsVA) {
        await chatInput.reply({
            embeds: [new EmbedBuilder()
                .setColor(0x0099FF)
                .setTitle("✅ Globally Registered")
                .setDescription(`You're registered${userDetails.if_community_id ? ` as **${userDetails.if_community_id}**` : ""}. This Discord server is not configured as a VA, so I won't collect a callsign here.`)
                .setFooter({ text: "Run /status anytime to review your setup" })],
            ephemeral: true,
        });
        return;
    }

    // SCENARIO 4: User is NOT registered - show new registration flow.
    await showNewUserRegistration(chatInput, currentServerIsVA, userDetails.current_server?.va_name);
}

/**
 * Shows new user registration info and button
 */
async function showNewUserRegistration(chatInput: any, currentServerIsVA: boolean, vaName?: string) {
    const infoEmbed = new EmbedBuilder()
        .setColor(0x0099FF)
        .setTitle("✈️ User Registration")
        .setDescription(`Create your global Comrade Bot account with your Infinite Flight Community identity.${currentServerIsVA ? ` After account creation, I'll offer to link you to **${vaName ?? "this VA"}** with a callsign.` : " This server is not configured as a VA, so no callsign is needed here."}`)
        .addFields(
            {
                name: "📝 IFC Username",
                value: "Your Infinite Flight Community username. We do **not** ask for your IF password, email, credentials, or full logbook.",
                inline: false
            },
            {
                name: "🛫 Last Flight",
                value: "Your most recent flight (origin-destination).\nFormat: `EGLL-KSEA` (4-letter ICAO codes)\n\nPick the highlighted flight from your Infinite Flight logbook (see image below).",
                inline: false
            }
            ,
            {
                name: "⚠️ Privacy & impersonation warning",
                value: "Use only your own IF Community account. The last-flight check helps prevent impersonation.",
                inline: false
            }
        )
        .setImage("attachment://register_logbook.png")
        .setFooter({ text: "Click 'Proceed' to continue" });

    // Create proceed button for new registration
    const proceedButton = new ButtonBuilder()
        .setCustomId(CUSTOM_IDS.REGISTER_NEW_BUTTON)
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
