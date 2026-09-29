import {
    ModalBuilder,
    TextInputBuilder,
    TextInputStyle,
    ActionRowBuilder,
    ModalActionRowComponentBuilder,
    MessageFlags,
} from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { CUSTOM_IDS } from "../configs/constants";

/**
 * Handle the "File PIREP" button click from the /tour command
 * Shows a modal asking for flight time in hh:mm format
 */
export async function handleTourFilePirep(interaction: DiscordInteraction): Promise<void> {
    const button = interaction.getButtonInteraction();
    if (!button) return;

    try {
        // Create modal for flight time input
        const modal = new ModalBuilder()
            .setCustomId(`${CUSTOM_IDS.PIREP_MODAL}_tour`)
            .setTitle("File Tour PIREP");

        // Add flight time input field
        const flightTimeField = new TextInputBuilder()
            .setCustomId("flight_time")
            .setLabel("Flight Time")
            .setPlaceholder("HH:MM (e.g., 02:30)")
            .setStyle(TextInputStyle.Short)
            .setRequired(true)
            .setMinLength(4)
            .setMaxLength(5);

        const flightTimeRow = new ActionRowBuilder<ModalActionRowComponentBuilder>()
            .addComponents(flightTimeField);

        modal.addComponents(flightTimeRow);

        // Show the modal
        await button.showModal(modal);
    } catch (err) {
        console.error("[tourButtonHandler] Error showing modal:", err);
        try {
            if (!button.replied && !button.deferred) {
                await button.reply({
                    content: "⚠️ An error occurred while opening the PIREP form. Please try again.",
                    flags: MessageFlags.Ephemeral
                });
            }
        } catch (replyErr) {
            console.error("[tourButtonHandler] Failed to send  message:", replyErr);
        }
    }

}
