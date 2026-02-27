import { Interaction } from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { CUSTOM_IDS } from "../configs/constants";

// Import handlers
import RegisterHandler from "../commands/registerModalHandler";
import InitServerHandler from "../commands/initServerModalHandler";
import PirepModalHandler from "../commands/pirepModalHandler";
import { handleInitServerProceed } from "../commands/initServerButtonHandler";
import { handleRegisterNew, handleRegisterLink } from "../commands/registerButtonHandler";
import { ConfigurePilotRoleHandler } from "../commands/ConfigurePilotRoleHandler";
import { SyncUserModalHandler } from "../commands/SyncUserHandler";
import { handleFlightHistory } from "../commands/logbookHandler";
import { handleLiveFlights } from "../commands/liveHandler";
import { logModeSelectionHandler } from "../commands/logModeSelectionHandler";
import { handleTourFilePirep } from "../commands/tourButtonHandler";
import { commandMap } from "../configs/commandMap";

/**
 * Centralized interaction router
 * Handles all Discord interactions: commands, modals, buttons, select menus
 */
export class InteractionRouter {
    /**
     * Route incoming interaction to appropriate handler
     */
    static async route(rawInteraction: Interaction): Promise<void> {
        try {
            // Route based on interaction type
            if (rawInteraction.isStringSelectMenu()) {
                await this.handleSelectMenu(rawInteraction);
            } else if (rawInteraction.isModalSubmit()) {
                await this.handleModalSubmit(rawInteraction);
            } else if (rawInteraction.isButton()) {
                await this.handleButton(rawInteraction);
            } else if (rawInteraction.isChatInputCommand()) {
                await this.handleCommand(rawInteraction);
            }
        } catch (error) {
            console.error("[InteractionRouter] Error:", error);
            await this.handleError(rawInteraction, error);
        }
    }

    /**
     * Handle select menu interactions
     */
    private static async handleSelectMenu(interaction: Interaction): Promise<void> {
        if (!interaction.isStringSelectMenu()) return;

        const [prefix, section, tag, userId] = interaction.customId.split("_");

        // Route to pilot role configuration
        if (prefix === CUSTOM_IDS.SET_PILOT_ROLE_MODAL) {
            const selectedRole = interaction.values[0];
            await ConfigurePilotRoleHandler.execute(
                new DiscordInteraction(interaction),
                section,
                selectedRole
            );
        }
    }

    /**
     * Handle modal submit interactions
     */
    private static async handleModalSubmit(interaction: Interaction): Promise<void> {
        if (!interaction.isModalSubmit()) return;

        const wrapped = new DiscordInteraction(interaction);

        // Static modal handlers
        switch (interaction.customId) {
            case RegisterHandler.data.name:
                await RegisterHandler.execute(wrapped);
                break;

            case InitServerHandler.data.name:
                await InitServerHandler.execute(wrapped);
                break;

            case CUSTOM_IDS.LINK_PILOT_CONFIRM:
                await SyncUserModalHandler.execute(wrapped);
                break;

            case "register_link_modal":
                // Handle link-only registration (callsign-only modal)
                await RegisterHandler.execute(wrapped);
                break;

            case CUSTOM_IDS.SYNC_USER_MODAL:
                await SyncUserModalHandler.execute(wrapped);
                break;

            case CUSTOM_IDS.MEMBERSHIP_JOIN_MODAL:
                const MembershipJoinHandler = await import("../commands/membershipJoinModalHandler");
                await MembershipJoinHandler.execute(wrapped);
                break;

            default:
                // Check if it's a PIREP modal with encoded mode_id (format: pirepModal_modeId)
                if (interaction.customId.startsWith(CUSTOM_IDS.PIREP_MODAL)) {
                    await PirepModalHandler.execute(wrapped);
                } else {
                    // Dynamic modal handlers (with IDs)
                    await this.handleDynamicModal(interaction, wrapped);
                }
        }
    }

    /**
     * Handle dynamic modals with custom IDs containing parameters
     */
    private static async handleDynamicModal(
        interaction: Interaction,
        wrapped: DiscordInteraction
    ): Promise<void> {
        if (!interaction.isModalSubmit()) return;

        const [prefix, section, tag] = interaction.customId.split("_");
        const customId = `${prefix}_${section}_${tag}`;

        // Sync user modal
        if (customId === CUSTOM_IDS.SYNC_USER_MODAL) {
            await SyncUserModalHandler.execute(wrapped);
        }
    }

    /**
     * Handle button interactions
     */
    private static async handleButton(interaction: Interaction): Promise<void> {
        if (!interaction.isButton()) return;

        const wrapped = new DiscordInteraction(interaction);

        // Handle initserver proceed button
        if (interaction.customId === "initserver_proceed") {
            await handleInitServerProceed(wrapped);
            return;
        }

        // Handle register new user button
        if (interaction.customId === "register_new") {
            await handleRegisterNew(wrapped);
            return;
        }

        // Handle register link button (for users who need to link to VA)
        if (interaction.customId === "register_link") {
            await handleRegisterLink(wrapped);
            return;
        }

        // Handle membership join proceed button
        if (interaction.customId === CUSTOM_IDS.MEMBERSHIP_JOIN_BUTTON) {
            const { handleMembershipJoinProceed } = await import("../commands/membershipJoinButtonHandler");
            await handleMembershipJoinProceed(wrapped);
            return;
        }

        // Handle PIREP mode selection buttons
        if (interaction.customId.startsWith(CUSTOM_IDS.PIREP_MODE_PREFIX)) {
            await logModeSelectionHandler(wrapped);
            return;
        }

        // Handle tour File PIREP button
        if (interaction.customId === "tour_file_pirep") {
            await handleTourFilePirep(wrapped);
            return;
        }

        // Parse button custom ID: {prefix}_{action}_{param1}_{param2}
        const [prefix, action, ...params] = interaction.customId.split("_");

        // Live flights pagination
        if (prefix === "live" && (action === "prev" || action === "next")) {
            const [pageStr] = params;
            const page = parseInt(pageStr, 10);
            await handleLiveFlights(wrapped, page);
            return;
        }

        // Flight history pagination
        if (prefix === "flights" && (action === "prev" || action === "next")) {
            const [ifcId, pageStr] = params;
            const page = parseInt(pageStr, 10);
            await handleFlightHistory(wrapped, page, ifcId);
        }
    }

    /**
     * Handle slash command interactions
     */
    private static async handleCommand(interaction: Interaction): Promise<void> {
        if (!interaction.isChatInputCommand()) return;

        console.log(`[Command] ${interaction.commandName} (Guild: ${interaction.guildId || 'DM'})`);

        const command = commandMap[interaction.commandName];
        if (!command) {
            console.warn(`[Command] Unknown command: ${interaction.commandName}`);
            return;
        }

        const wrapped = new DiscordInteraction(interaction);
        await command.execute(wrapped);
    }


    /**
     * Handle errors gracefully
     */
    private static async handleError(interaction: Interaction, error: unknown): Promise<void> {
        console.error("[InteractionRouter] Error details:", error);

        const errorMessage = {
            content: "⚠️ An unexpected error occurred. Please try again later.",
            ephemeral: true
        };

        try {
            if (interaction.isRepliable()) {
                if (interaction.replied || interaction.deferred) {
                    await interaction.followUp(errorMessage);
                } else {
                    await interaction.reply(errorMessage);
                }
            }
        } catch (replyError) {
            console.error("[InteractionRouter] Failed to send error message:", replyError);
        }
    }
}
