import { DiscordInteraction } from "../types/DiscordInteraction";
import { parseLiveOrgSelectCustomId } from "../helpers/liveCommands";
import { handleAircraftsOrgSelect } from "./aircrafts";
import { handleSchedulesOrgSelect } from "./schedules";
import { handleHistoryOrgSelect } from "./history";

export async function handleLiveOrgSelect(interaction: DiscordInteraction): Promise<void> {
    const select = interaction.getSelectMenuInteraction();
    if (!select) return;

    const parsed = parseLiveOrgSelectCustomId(select.customId);
    if (!parsed) {
        await select.reply({ content: "Invalid selection context.", ephemeral: true });
        return;
    }

    switch (parsed.command) {
        case "aircrafts":
            await handleAircraftsOrgSelect(interaction, parsed.payload);
            return;
        case "schedules":
            await handleSchedulesOrgSelect(interaction, parsed.payload);
            return;
        case "history":
            await handleHistoryOrgSelect(interaction, parsed.payload);
            return;
        default:
            await select.reply({ content: "Unknown command context.", ephemeral: true });
    }
}
