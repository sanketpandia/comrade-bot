import {
    ModalBuilder,
    TextInputBuilder,
    ActionRowBuilder,
    TextInputStyle,
} from "discord.js";
import { CUSTOM_IDS } from "../configs/constants";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { ApiService } from "../services/apiService";
import { CommandErrorHandler } from "../helpers/commandErrorHandler";

/**
 * Handles the "Proceed with Registration" button for new users
 * Shows modal with IFC ID and last flight.
 */
export async function handleRegisterNew(interaction: DiscordInteraction) {
    const buttonInteraction = interaction.getButtonInteraction();
    if (!buttonInteraction) return;

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
 * Links a registered user to the current guild VA (no callsign prompt).
 */
export async function handleRegisterLink(interaction: DiscordInteraction) {
    const buttonInteraction = interaction.getButtonInteraction();
    if (!buttonInteraction) return;

    await buttonInteraction.deferReply({ ephemeral: true });

    CommandErrorHandler.logExecution("VA Link", buttonInteraction.user.id, buttonInteraction.guildId, {});

    try {
        const response = await ApiService.linkMembership(interaction.getMetaInfo());
        if (!response?.success) {
            await CommandErrorHandler.handleEmptyResponse(interaction);
            return;
        }

        await interaction.editReply({
            content: "✅ **Successfully Linked to Virtual Airline!**\n\nYou're now a member of this server's VA.\n\nUse `/help` to learn how to use Comrade Bot.",
        });
    } catch (error) {
        await CommandErrorHandler.handleApiError(interaction, error, "VA Link");
    }
}

export default {
    handleRegisterNew,
    handleRegisterLink
};
