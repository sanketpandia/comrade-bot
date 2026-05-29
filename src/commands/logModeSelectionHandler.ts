import {
    ModalBuilder,
} from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { ApiService } from "../services/apiService";
import { CUSTOM_IDS } from "../configs/constants";
import { buildSingleModalForMode } from "./discordPirepFlowOrchestrator";

export async function logModeSelectionHandler(interaction: DiscordInteraction): Promise<void> {
    const buttonInteraction = interaction.getButtonInteraction();
    if (!buttonInteraction) return;

    try {
        // Extract mode_id from customId (e.g., "mode_classic" -> "classic")
        const modeId = buttonInteraction.customId.replace(new RegExp(`^${CUSTOM_IDS.PIREP_MODE_PREFIX}`), "");

        const metaInfo = interaction.getMetaInfo();

        // Fetch PIREP config to get field definitions for this mode
        const pirepConfig = await ApiService.getPirepConfig(metaInfo);

        // Validate we have data
        if (!pirepConfig.result) {
            console.error("[logModeSelectionHandler] No data in PIREP config response");
            await buttonInteraction.reply({
                content: "❌ Failed to load PIREP configuration",
                ephemeral: true
            });
            return;
        }

        // Extract user info from config
        const userInfo = pirepConfig.result.user_info;

        // Find the selected mode
        const selectedMode = pirepConfig.result.available_modes.find((m: { mode_id: string }) => m.mode_id === modeId);

        if (!selectedMode) {
            console.error(`[logModeSelectionHandler] Selected mode ${modeId} not found`);
            await buttonInteraction.reply({
                content: "❌ Selected mode not found",
                ephemeral: true
            });
            return;
        }

        if (selectedMode.status !== "valid") {
            console.error(`[logModeSelectionHandler] Mode ${modeId} not valid: ${selectedMode.error_reason}`);
            await buttonInteraction.reply({
                content: `❌ Mode not valid: ${selectedMode.error_reason}`,
                ephemeral: true
            });
            return;
        }

        let modal: ModalBuilder;
        try {
            modal = buildSingleModalForMode(selectedMode);
        } catch (orchestratorErr: any) {
            console.warn(`[logModeSelectionHandler] Invalid mode ${modeId}: ${orchestratorErr?.message}`);
            await buttonInteraction.reply({
                content: "❌ This flight mode configuration is invalid (more than 5 modal fields). Please contact a VA admin.",
                ephemeral: true
            });
            return;
        }

        // Remove the button row from the original message before showing modal
        try {
            if (buttonInteraction.message) {
                await buttonInteraction.message.edit({
                    components: [] // Remove all button components
                });
                console.log(`[logModeSelectionHandler] Removed button row from original message`);
            }
        } catch (editErr: any) {
            console.error("[logModeSelectionHandler] Failed to remove buttons from original message:", editErr?.message);
            // Continue anyway - not a critical failure
        }

        // Show the modal (no followup possible after modal is shown)
        try {
            await buttonInteraction.showModal(modal);
        } catch (modalErr: any) {
            console.error("[logModeSelectionHandler] Failed to show modal:", modalErr?.message);
            // If modal failed (e.g., interaction already replied), try to reply with error
            try {
                if (!buttonInteraction.replied && !buttonInteraction.deferred) {
                    await buttonInteraction.reply({
                        content: "❌ Failed to open PIREP form. Please try again.",
                        ephemeral: true
                    });
                }
            } catch (replyErr) {
                console.error("[logModeSelectionHandler] Failed to send error reply:", replyErr);
            }
        }
    } catch (err) {
        console.error("[logModeSelectionHandler] Handler error:", err);
        // Try to reply if we haven't already
        try {
            if (!buttonInteraction.replied && !buttonInteraction.deferred) {
                await buttonInteraction.reply({
                    content: "❌ An error occurred while processing your request",
                    ephemeral: true
                });
            }
        } catch (replyErr) {
            console.error("[logModeSelectionHandler] Failed to send error reply:", replyErr);
        }
    }
}
