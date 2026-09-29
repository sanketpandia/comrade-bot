import { SlashCommandBuilder, ActionRowBuilder, ButtonBuilder, ButtonStyle } from 'discord.js';
import { ApiService } from '../services/apiService';
import { DiscordInteraction } from '../types/DiscordInteraction';
import { UnauthorizedError } from '../helpers/UnauthorizedException';
import { PermissionDeniedError } from '../helpers/PermissionDeniedException';

const data = new SlashCommandBuilder()
    .setName('dashboard')
    .setDescription('Get a secure link to access the Vizburo flight dashboard');

async function execute(interaction: DiscordInteraction): Promise<void> {
    const chat = interaction.getChatInputInteraction();
    if (!chat) return;

    try {
        // Defer the reply since we're making API calls (ephemeral)
        await chat.deferReply({ ephemeral: true });

        const metaInfo = interaction.getMetaInfo();

        try {
            // Generate signed link for dashboard (same as /tour command)
            const signedLinkResponse = await ApiService.generateSignedLink(metaInfo, "/dashboard");
            
            if (!signedLinkResponse?.result?.url) {
                await chat.editReply({
                    embeds: [{
                        title: "❌ Error",
                        description: "Could not generate dashboard link.\nPlease ensure you are registered with the Virtual Airline.",
                        color: 0xff0000,
                        timestamp: new Date().toISOString()
                    }]
                });
                return;
            }

            const dashboardLink = signedLinkResponse.result.url;
            const expiresIn = signedLinkResponse.result.expires_in || 900; // seconds

            // Create "Open Dashboard" button
            const dashboardButton = new ButtonBuilder()
                .setLabel("Open Dashboard")
                .setStyle(ButtonStyle.Link)
                .setURL(dashboardLink);

            const buttonRow = new ActionRowBuilder<ButtonBuilder>()
                .addComponents(dashboardButton);

            await chat.editReply({
                embeds: [{
                    title: "🚀 Vizburo Dashboard Access",
                    description: `Click the button below to open your dashboard.\n\n*Link expires in ${Math.floor(expiresIn / 60)} minutes*`,
                    color: 0x0099ff,
                    timestamp: new Date().toISOString()
                }],
                components: [buttonRow]
            });
        } catch (dashboardErr: any) {
            if (dashboardErr instanceof UnauthorizedError) {
                await chat.editReply({
                    embeds: [{
                        title: "🔒 Not Authorized",
                        description: `❌ ${dashboardErr.message}`,
                        color: 0xff0000,
                        timestamp: new Date().toISOString()
                    }]
                });
                return;
            }

            if (dashboardErr instanceof PermissionDeniedError) {
                await chat.editReply({
                    embeds: [{
                        title: "⚠️ Registration Required",
                        description: `❌ ${dashboardErr.message}\n\nPlease register your account using the \`/register\` command before accessing the dashboard.`,
                        color: 0xff9900,
                        timestamp: new Date().toISOString()
                    }]
                });
                return;
            }

            console.error("[dashboard command] Error generating dashboard link:", dashboardErr);
            await chat.editReply({
                embeds: [{
                    title: "❌ Error",
                    description: `Unable to generate dashboard link: ${dashboardErr.message || 'Unknown error'}`,
                    color: 0xff0000,
                    timestamp: new Date().toISOString()
                }]
            });
        }
    } catch (err) {
        console.error("[dashboard command]", err);
        try {
            const chat = interaction.getChatInputInteraction();
            if (chat) {
                await chat.editReply({
                    embeds: [{
                        title: "Error",
                        description: `❌ An error occurred: ${String(err)}`,
                        color: 0xff0000
                    }]
                });
            }
        } catch (replyErr) {
            console.error("[dashboard command] Failed to send error message:", replyErr);
        }
    }
}

export { data, execute };
