import { SlashCommandBuilder } from 'discord.js';
import { ApiService } from '../services/apiService';
import { DiscordInteraction } from '../types/DiscordInteraction';
import { CommandErrorHandler } from '../helpers/commandErrorHandler';

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
        const expiresInMinutes = Math.floor(expiresIn / 60);

        // Send response with dashboard link
        // Note: Discord doesn't support markdown links in message content, use angle brackets for clickable URLs
        await interaction.editReply({
            content: `🚀 **Vizburo Dashboard Access**\n\n` +
                    `Click the link below to access your flight dashboard:\n` +
                    `<${url}>\n\n` +
                    `⏱️ This link expires in **${expiresInMinutes} minutes**\n` +
                    `🔒 For security, this link can only be used once`,
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
