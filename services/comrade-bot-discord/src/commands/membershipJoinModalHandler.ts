import { ApiService } from "../services/apiService";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { CUSTOM_IDS } from "../configs/constants";
import { CommandErrorHandler } from "../helpers/commandErrorHandler";

export const data = {
    name: CUSTOM_IDS.MEMBERSHIP_JOIN_MODAL
}

/**
 * Handles membership join modal submission
 * Submits callsign to API and processes response
 */
export async function execute(interaction: DiscordInteraction) {
    const _interaction = interaction.getModalInputInteraction();
    if (!_interaction) return;

    // Extract callsign
    const callsign = _interaction.fields.getTextInputValue('callsign').trim();

    // Basic validation - ensure not empty
    if (!callsign || callsign.length === 0) {
        await interaction.reply({
            content: "❌ **Validation Error**\nCallsign cannot be empty.",
            ephemeral: true
        });
        return;
    }

    // Log execution
    CommandErrorHandler.logExecution(
        "Membership Join",
        _interaction.user.id,
        _interaction.guildId,
        { callsign }
    );

    try {
        const response = await ApiService.joinMembership(
            interaction.getMetaInfo(),
            callsign
        );

        // Validate response
        if (!response) {
            await CommandErrorHandler.handleEmptyResponse(interaction);
            return;
        }

        // Success response
        if (response.success) {
            const roleDisplay = response.role.charAt(0).toUpperCase() + response.role.slice(1);

            await interaction.reply({
                content: `✅ **Successfully Joined Virtual Airline!**\n\n${response.message}\n\n**Your Details:**\n📋 Callsign: \`${response.callsign}\`\n🎖️ Role: ${roleDisplay}\n\nUse \`/status\` to view your full profile.`,
                ephemeral: true
            });
        } else {
            // Failure response from API
            await interaction.reply({
                content: `❌ **Join Failed**\n\n${response.message}`,
                ephemeral: true
            });
        }

    } catch (error) {
        // Handle specific error codes
        const errorMessage = error instanceof Error ? error.message : '';

        if (errorMessage.startsWith("ALREADY_MEMBER:")) {
            await interaction.reply({
                content: `❌ **Already a Member**\n\nYou are already a member of this Virtual Airline. Use \`/status\` to view your membership details.`,
                ephemeral: true
            });
            return;
        }

        if (errorMessage.startsWith("CALLSIGN_TAKEN:")) {
            await interaction.reply({
                content: `❌ **Callsign Unavailable**\n\nThis callsign is already taken by another member. Please verify your callsign with VA staff.`,
                ephemeral: true
            });
            return;
        }

        if (errorMessage.startsWith("VA_NOT_FOUND:")) {
            await interaction.reply({
                content: `❌ **Virtual Airline Not Found**\n\nThis server's Virtual Airline configuration was not found. Please contact server administrators.`,
                ephemeral: true
            });
            return;
        }

        if (errorMessage.startsWith("USER_NOT_FOUND:")) {
            await interaction.reply({
                content: `❌ **Not Registered**\n\nYou must register with Comrade Bot first before joining a Virtual Airline.\n\nUse \`/register\` to get started.`,
                ephemeral: true
            });
            return;
        }

        // Generic error handling
        await CommandErrorHandler.handleApiError(interaction, error, "Membership Join");
    }
}

export default {
    data,
    execute
};
