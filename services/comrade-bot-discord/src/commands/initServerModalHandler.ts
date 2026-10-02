import { CUSTOM_IDS } from "../configs/constants";
import { ApiService } from "../services/apiService";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { CommandErrorHandler, ValidationPatterns } from "../helpers/commandErrorHandler";
import { PolitburoApiError } from "../helpers/PolitburoApiError";
import { replyToCreateServerError } from "../helpers/serverRegistrationErrors";

export const data = {
    name: CUSTOM_IDS.INIT_SERVER_MODAL
}

/**
 * Handles server initialization modal submission
 * Validates VA code, then registers server via API
 */
export async function execute(interaction: DiscordInteraction) {
    const _interaction = interaction.getModalInputInteraction();
    if (!_interaction) return;

    if (_interaction.customId !== CUSTOM_IDS.INIT_SERVER_MODAL) return;

    const vaCode = _interaction.fields.getTextInputValue("vaCode").trim().toUpperCase();

    if (!await CommandErrorHandler.validateInput(
        interaction, vaCode, "VA code", ValidationPatterns.VA_CODE, 3, 5
    )) return;

    CommandErrorHandler.logExecution("Server Init", _interaction.user.id, _interaction.guildId, {
        vaCode,
    });

    try {
        const response = await ApiService.initiateServerRegistration(
            interaction.getMetaInfo(),
            vaCode,
        );

        if (!response) {
            await CommandErrorHandler.handleEmptyResponse(interaction);
            return;
        }

        if (response.success) {
            await interaction.reply({
                content: `✅ **VA setup started!**\n\n${response.message}\n\n**VA Code / ID:** ${response.va_code}\n\nNext: use \`/dashboard\` to open Vizburo and finish Basic Setup. A desktop browser or desktop view is recommended. Live-flight matching starts after staff add a callsign start or end.`,
                ephemeral: true
            });
        } else {
            await interaction.reply({
                content: `❌ **Server Initialization Failed**\n\n${response.message}`,
                ephemeral: true
            });
        }

    } catch (error) {
        if (PolitburoApiError.isPolitburoApiError(error)) {
            await replyToCreateServerError(interaction, error);
            return;
        }
        await CommandErrorHandler.handleApiError(interaction, error, "Server Init");
    }
}

export default {
    execute,
    data
}
