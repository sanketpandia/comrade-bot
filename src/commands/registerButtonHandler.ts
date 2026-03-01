import {
    ModalBuilder,
    TextInputBuilder,
    ActionRowBuilder,
    TextInputStyle,
} from "discord.js";
import { CUSTOM_IDS } from "../configs/constants";
import { DiscordInteraction } from "../types/DiscordInteraction";

/**
 * Handles the "Proceed with Registration" button for new users
 * Shows modal with IFC ID, last flight, and optional callsign
 */
export async function handleRegisterNew(interaction: DiscordInteraction) {
    const buttonInteraction = interaction.getButtonInteraction();
    if (!buttonInteraction) return;

    // Build modal fields
    const ifcIdInput = new TextInputBuilder()
        .setCustomId("ifcId")
        .setLabel("IFC Username")
        .setStyle(TextInputStyle.Short)
        .setPlaceholder("john_doe123")
        .setRequired(true)
        .setMinLength(3)
        .setMaxLength(30);

    const lastFlightInput = new TextInputBuilder()
        .setCustomId("lastFlight")
        .setLabel("Last Valid Flight (ICAO-ICAO)")
        .setStyle(TextInputStyle.Short)
        .setPlaceholder("KJFK-EGLL")
        .setRequired(true)
        .setMinLength(9)
        .setMaxLength(9);

    const modal = new ModalBuilder()
        .setCustomId(CUSTOM_IDS.REGISTER_MODAL)
        .setTitle("Register to Comrade Bot");

    modal.addComponents(
        new ActionRowBuilder<TextInputBuilder>().addComponents(ifcIdInput),
        new ActionRowBuilder<TextInputBuilder>().addComponents(lastFlightInput)
    );

    await buttonInteraction.showModal(modal);
}

/**
 * Handles the "Link to VA" button for registered users who want to link to a VA
 * Shows modal with only callsign field
 */
export async function handleRegisterLink(interaction: DiscordInteraction) {
    const buttonInteraction = interaction.getButtonInteraction();
    if (!buttonInteraction) return;

    // Build modal with only callsign field (same as membership join)
    const callsignInput = new TextInputBuilder()
        .setCustomId("callsign")
        .setLabel("Your Callsign")
        .setStyle(TextInputStyle.Short)
        .setPlaceholder("Enter your callsign exactly as registered")
        .setRequired(true)
        .setMinLength(1)
        .setMaxLength(50);

    const modal = new ModalBuilder()
        .setCustomId("register_link_modal")
        .setTitle("Link to Virtual Airline");

    modal.addComponents(
        new ActionRowBuilder<TextInputBuilder>().addComponents(callsignInput)
    );

    await buttonInteraction.showModal(modal);
}

export default {
    handleRegisterNew,
    handleRegisterLink
};
