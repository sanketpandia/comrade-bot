import { SlashCommandBuilder } from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { handleFlightHistory } from "./logbookHandler";
import { ApiService } from "../services/apiService";
import { CommandErrorHandler } from "../helpers/commandErrorHandler";
import { DiscordResponses } from "../helpers/discordResponses";

export const data = new SlashCommandBuilder()
    .setName("logbook")
    .setDescription("Fetch flights from logbook of user")
    .addStringOption(option =>
        option.setName("ifc_id")
          .setDescription("The user's Infinite Flight Community ID")
          .setRequired(true)
      );

export async function execute(interaction: DiscordInteraction) {
    try {
        // Defer reply to keep interaction token alive (ephemeral for privacy)
        await interaction.deferReply(true);

        const ifcId = interaction.getStringParam("ifc_id", true);
        if (!ifcId) {
            await interaction.editReply({
                content: "❌ **Error**\nIFC ID is required."
            });
            return;
        }

        // Get meta info for API call
        const metaInfo = interaction.getMetaInfo();

        // Generate signed link with logbook URL and user_id query parameter
        const redirectTo = `/dashboard/logbook?user_id=${encodeURIComponent(ifcId)}`;
        const signedLinkResponse = await ApiService.generateSignedLink(metaInfo, redirectTo, 15);

        if (!signedLinkResponse || !signedLinkResponse.result?.url) {
            // Fallback to regular logbook handler if signed link generation fails
            const page = 1;
            await handleFlightHistory(interaction, page);
            return;
        }

        const url = signedLinkResponse.result.url;
        const expiresIn = signedLinkResponse.result.expires_in || 900; // seconds

        // Send response with logbook link using reusable helper
        await interaction.editReply({
            content: DiscordResponses.formatSignedLinkMessage(
                "📋 **Logbook Viewer**",
                url,
                expiresIn,
                `Click the link below to view the logbook for **${ifcId}**:`
            )
        });

        // Log execution
        CommandErrorHandler.logExecution("logbook", interaction.userId, interaction.guildId, {
            ifc_id: ifcId,
            signed_link: true,
            url: url.substring(0, 50) + '...',
            expiresIn
        });

    } catch (error) {
        await CommandErrorHandler.handleApiError(interaction, error, "logbook");
    }
}