import { EmbedBuilder } from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";

const AUTHOR = "infinite-experiment";

export class DiscordResponses {

    static async replyWithEmbed(
        interaction: DiscordInteraction,
        title: string,
        description: string,
        color: number = 0x0099ff
    ) {
        const embed = new EmbedBuilder()
            .setTitle(title)
            .setDescription(description)
            .setColor(color);
        
        await interaction.reply({
            embeds: [embed]
        });
    }

    /**
     * Formats a signed link message for Discord
     * @param title - Title of the message (e.g., "📋 Logbook Viewer" or "🚀 Vizburo Dashboard Access")
     * @param url - The signed link URL
     * @param expiresIn - Expiration time in seconds
     * @param description - Optional additional description (e.g., "Click the link below to view the logbook for **username**")
     * @returns Formatted message content string
     */
    static formatSignedLinkMessage(
        title: string,
        url: string,
        expiresIn: number,
        description?: string
    ): string {
        const expiresInMinutes = Math.floor(expiresIn / 60);
        let content = `${title}\n\n`;
        
        if (description) {
            content += `${description}\n`;
        }
        
        content += `Click the link below to access:\n` +
                  `<${url}>\n\n` +
                  `⏱️ This link expires in **${expiresInMinutes} minutes**\n` +
                  `🔒 For security, this link can only be used once`;
        
        return content;
    }
}