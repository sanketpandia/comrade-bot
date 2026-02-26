import {
    ModalBuilder,
    TextInputBuilder,
    TextInputStyle,
    ActionRowBuilder,
    ModalActionRowComponentBuilder,
} from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { ApiService } from "../services/apiService";
import { CUSTOM_IDS } from "../configs/constants";

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

        // Create modal with dynamic fields based on mode configuration
        // Encode mode_id in custom ID so we can extract it on submission
        const modal = new ModalBuilder()
            .setCustomId(`${CUSTOM_IDS.PIREP_MODAL}_${modeId}`)
            .setTitle(`${selectedMode.display_name} - PIREP`);

        // No context metadata field needed - backend derives everything from claims + request

        // Add route field if mode requires route selection
        if (selectedMode.requires_route_selection) {
            const routeField = new TextInputBuilder()
                .setCustomId("route_id")
                .setLabel("Route (e.g., LFPG-EGLL)")
                .setPlaceholder("Enter the route code")
                .setStyle(TextInputStyle.Short)
                .setRequired(true);

            // Prepopulate with autofill_route (takes priority)
            // OR use current route if it's valid for this mode
            if (selectedMode.autofill_route) {
                routeField.setValue(selectedMode.autofill_route);
            } else if (selectedMode.status === "valid" && userInfo.current_route) {
                // Current route is valid for this mode, prefill it
                routeField.setValue(userInfo.current_route);
            }

            const routeRow = new ActionRowBuilder<ModalActionRowComponentBuilder>()
                .addComponents(routeField);
            modal.addComponents(routeRow);
        }

        // Add mode-specific fields
        for (const field of selectedMode.fields) {
            const inputField = createInputField(field);
            const row = new ActionRowBuilder<ModalActionRowComponentBuilder>()
                .addComponents(inputField);
            modal.addComponents(row);
        }

        // Discord modals have a max of 5 rows total (5 input components)
        // Count: 1 context (hidden) + 1 route (if required) + N mode fields
        const totalComponents = 1 + (selectedMode.requires_route_selection ? 1 : 0) + selectedMode.fields.length;
        if (totalComponents > 5) {
            console.warn(`[logModeSelectionHandler] Mode ${modeId} has ${totalComponents} fields, exceeds Discord limit of 5`);
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

/**
 * Create a TextInputBuilder from a field configuration
 */
function createInputField(field: any): TextInputBuilder {
    const input = new TextInputBuilder()
        .setCustomId(field.name)
        .setLabel(field.label)
        .setRequired(field.required);

    // Set input style based on field type
    if (field.type === "textarea") {
        input.setStyle(TextInputStyle.Paragraph);
    } else if (field.type === "number") {
        input.setStyle(TextInputStyle.Short);
        input.setPlaceholder("Enter a number");
    } else {
        // text or default
        input.setStyle(TextInputStyle.Short);
    }

    // Add placeholders for common fields
    if (field.name === "flight_time") {
        input.setPlaceholder("HH:MM");
    } else if (field.name === "fuel_kg") {
        input.setPlaceholder("e.g., 15000");
    } else if (field.name === "cargo_kg") {
        input.setPlaceholder("e.g., 5000");
    } else if (field.name === "passengers") {
        input.setPlaceholder("e.g., 200");
    }

    return input;
}
