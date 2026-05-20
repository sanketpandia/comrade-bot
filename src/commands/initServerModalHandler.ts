import { CUSTOM_IDS } from "../configs/constants";
import { ApiService } from "../services/apiService";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { MessageFormatters } from "../helpers/messageFormatter";
import { CommandErrorHandler, ValidationPatterns } from "../helpers/commandErrorHandler";

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

    // Extract inputs
    const vaCode = _interaction.fields.getTextInputValue("vaCode").trim().toUpperCase();

    // Validate VA code
    if (!await CommandErrorHandler.validateInput(
        interaction, vaCode, "VA code", ValidationPatterns.VA_CODE, 3, 5
    )) return;

    // Log execution
    CommandErrorHandler.logExecution("Server Init", _interaction.user.id, _interaction.guildId, {
        vaCode,
    });
 
    try {
        const response = await ApiService.initiateServerRegistration(
            interaction.getMetaInfo(),
            vaCode,
        );

        // Validate response
        if (!response) {
            await CommandErrorHandler.handleEmptyResponse(interaction);
            return;
        }

        // Send success response
        if (response.success) {
            await interaction.reply({
                content: `✅ **VA setup started!**\n\n${response.message}\n\n**VA Code / ID:** ${response.va_code}\n\nNext: use \`/dashboard\` to open Vizburo and finish Basic Setup. A desktop browser or desktop view is recommended. Live-flight matching starts after staff add a callsign start or end.`,
                ephemeral: true
            });
        } else {
            // Initialization failed
            await interaction.reply({
                content: `❌ **Server Initialization Failed**\n\n${response.message}`,
                ephemeral: true
            });
        }

    } catch (error) {
        const errorMessage = error instanceof Error ? error.message : "";
        if (errorMessage.includes("USER_NOT_REGISTERED")) {
            await interaction.reply({ content: "👋 Please run `/register` first, then try `/initserver` again.", ephemeral: true });
            return;
        }
        if (errorMessage.includes("SERVER_ALREADY_REGISTERED")) {
            await interaction.reply({ content: "✅ This Discord server is already initialized. Use `/dashboard` to continue setup in Vizburo.", ephemeral: true });
            return;
        }
        if (errorMessage.includes("VA_CODE_ALREADY_EXISTS")) {
            await interaction.reply({ content: "❌ That VA Code / ID is already in use. Please choose the official unique code for your VA or contact support.", ephemeral: true });
            return;
        }
        if (errorMessage.includes("MISSING_DISCORD_CONTEXT")) {
            await interaction.reply({ content: "🔒 Discord context was missing. Please try `/initserver` again inside the server you want to set up.", ephemeral: true });
            return;
        }
        await CommandErrorHandler.handleApiError(interaction, error, "Server Init");
    }
}

export default {
    execute,
    data
}
