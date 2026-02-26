import {
    ModalBuilder,
    TextInputBuilder,
    ActionRowBuilder,
    TextInputStyle,
} from "discord.js";
import { CUSTOM_IDS } from "../configs/constants";
import { DiscordInteraction } from "../types/DiscordInteraction";

/**
 * Handles the "Proceed" button for membership join
 * Shows modal with callsign input field
 */
export async function handleMembershipJoinProceed(interaction: DiscordInteraction) {
    const buttonInteraction = interaction.getButtonInteraction();
    if (!buttonInteraction) return;

    const callsignInput = new TextInputBuilder()
        .setCustomId("callsign")
        .setLabel("Your Callsign")
        .setStyle(TextInputStyle.Short)
        .setPlaceholder("Enter your callsign exactly as registered")
        .setRequired(true)
        .setMinLength(1)
        .setMaxLength(50);

    const modal = new ModalBuilder()
        .setCustomId(CUSTOM_IDS.MEMBERSHIP_JOIN_MODAL)
        .setTitle("Join Virtual Airline");

    modal.addComponents(
        new ActionRowBuilder<TextInputBuilder>().addComponents(callsignInput)
    );

    await buttonInteraction.showModal(modal);
}

export default {
    handleMembershipJoinProceed
};
