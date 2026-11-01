import {
    ActionRowBuilder,
    ButtonBuilder,
    ButtonStyle,
} from "discord.js";
import { ApiService } from "../services/apiService";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { CUSTOM_IDS } from "../configs/constants";
import { CommandErrorHandler, ValidationPatterns } from "../helpers/commandErrorHandler";
import { PolitburoApiError } from "../helpers/PolitburoApiError";
import { replyToCreateUserError } from "../helpers/registrationErrors";

export const data = {
    name: CUSTOM_IDS.REGISTER_MODAL
}

/** Handles full user registration modal (IFC username + last flight). */
export async function execute(interaction: DiscordInteraction) {
    const _interaction = interaction.getModalInputInteraction();
    if (!_interaction) return;

    await handleFullRegistration(interaction);
}

async function handleFullRegistration(interaction: DiscordInteraction) {
    const _interaction = interaction.getModalInputInteraction();
    if (!_interaction) return;

    const ifcId = _interaction.fields.getTextInputValue('ifcId').trim();
    const lastFlight = _interaction.fields.getTextInputValue('lastFlight').trim().toUpperCase();

    if (!await CommandErrorHandler.validateInput(
        interaction, ifcId, "IFC ID", ValidationPatterns.IFC_USERNAME, 3, 30
    )) return;

    if (!await CommandErrorHandler.validateInput(
        interaction, lastFlight, "flight route", ValidationPatterns.FLIGHT_ROUTE
    )) return;

    CommandErrorHandler.logExecution("Registration", _interaction.user.id, _interaction.guildId, {});

    try {
        const response = await ApiService.initiateRegistration(
            interaction.getMetaInfo(),
            ifcId,
            lastFlight
        );

        if (!response) {
            console.error("[RegisterModal] Empty response received from API");
            await CommandErrorHandler.handleEmptyResponse(interaction);
            return;
        }

        if (response.success) {
            let message = `✅ **Registration Successful!**\n\n${response.message}\n\n`;

            if (response.is_va_registered) {
                message += "📌 **Next Step:**\nClick the button below to link to **this server's VA**.";

                const linkButton = new ButtonBuilder()
                    .setCustomId(CUSTOM_IDS.REGISTER_LINK_BUTTON)
                    .setLabel("Link to VA")
                    .setStyle(ButtonStyle.Primary)
                    .setEmoji("🔗");

                const row = new ActionRowBuilder<ButtonBuilder>()
                    .addComponents(linkButton);

                await interaction.reply({
                    content: message,
                    components: [row],
                    ephemeral: true
                });
            } else {
                message += "You are successfully setup for using this bot!";

                await interaction.reply({
                    content: message,
                    ephemeral: true
                });
            }
        } else {
            await interaction.reply({
                content: `❌ **Registration Failed**\n\n${response.message}`,
                ephemeral: true
            });
        }

    } catch (error) {
        if (PolitburoApiError.isPolitburoApiError(error)) {
            await replyToCreateUserError(interaction, error, { ifcUsername: ifcId });
            return;
        }
        await CommandErrorHandler.handleApiError(interaction, error, "Registration");
    }
}

export default {
    data,
    execute
};
