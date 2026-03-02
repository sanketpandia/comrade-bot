import { SlashCommandBuilder } from 'discord.js';
import { ApiService } from '../services/apiService';
import { DiscordInteraction } from '../types/DiscordInteraction';
import { CommandErrorHandler } from '../helpers/commandErrorHandler';
import { DiscordResponses } from '../helpers/discordResponses';

const data = new SlashCommandBuilder()
    .setName('dashboard')
    .setDescription('Get a secure link to access the Vizburo flight dashboard');

async function execute(interaction: DiscordInteraction): Promise<void> {
    try {
        // Defer reply to keep interaction token alive
        await interaction.deferReply(true);

        // Get meta info for API call
        const metaInfo = interaction.getMetaInfo();

        // Call API to generate dashboard link
        const response = await ApiService.generateDashboardLink(metaInfo);

        // Check if response is valid
        if (!response || !response.result?.url) {
            await interaction.editReply({
                content: '❌ Could not generate dashboard link.\n' +
                        'Please ensure you are registered with the Virtual Airline.',
            });
            return;
        }

        const url = response.result.url;
        const expiresIn = response.result.expires_in || 900; // seconds

        // Send response with dashboard link using reusable helper
        // Note: Discord doesn't support markdown links in message content, use angle brackets for clickable URLs
        await interaction.editReply({
            content: DiscordResponses.formatSignedLinkMessage(
                "🚀 **Vizburo Dashboard Access**",
                url,
                expiresIn
            )
        });

        // Log execution
        CommandErrorHandler.logExecution("Dashboard", interaction.userId, interaction.guildId, {
            url: url.substring(0, 50) + '...',
            expiresIn,
        });

    } catch (error) {
        await CommandErrorHandler.handleApiError(interaction, error, "Dashboard");
    }
}

export { data, execute };
