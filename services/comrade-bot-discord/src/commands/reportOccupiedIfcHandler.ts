import { ApiService } from "../services/apiService";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { CUSTOM_IDS } from "../configs/constants";
import { CommandErrorHandler } from "../helpers/commandErrorHandler";

export async function handleReportOccupiedIFC(interaction: DiscordInteraction, customId: string) {
    const ifc = customId.slice(CUSTOM_IDS.REPORT_OCCUPIED_IFC_PREFIX.length);
    if (!ifc) {
        await interaction.reply({ content: "❌ Missing IFC username for report.", ephemeral: true });
        return;
    }
    try {
        const report = await ApiService.reportOccupiedIFC(interaction.getMetaInfo(), ifc);
        await interaction.reply({
            content: `✅ **Report submitted**\nYour report for IFC username **${ifc}** was recorded (id: \`${report.id}\`). Platform operators will review it.`,
            ephemeral: true,
        });
    } catch (error) {
        await CommandErrorHandler.handleApiError(interaction, error, "ReportOccupiedIFC");
    }
}
