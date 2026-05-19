import { Interaction } from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { CUSTOM_IDS } from "../configs/constants";

// Import handlers
import RegisterHandler from "../commands/registerModalHandler";
import InitServerHandler from "../commands/initServerModalHandler";
import PirepModalHandler from "../commands/pirepModalHandler";
import { handleInitServerProceed } from "../commands/initServerButtonHandler";
import { handleRegisterNew, handleRegisterLink } from "../commands/registerButtonHandler";
import { handleFlightHistory } from "../commands/logbookHandler";
import { handleLiveFlights } from "../commands/liveHandler";
import { logModeSelectionHandler } from "../commands/logModeSelectionHandler";
import { handleTourFilePirep } from "../commands/tourButtonHandler";
import { commandMap } from "../configs/commandMap";

type WrappedHandler = (interaction: DiscordInteraction) => Promise<void>;

/**
 * Centralized interaction router
 * Handles all Discord interactions: commands, modals, buttons, select menus
 */
export class InteractionRouter {
    private static readonly modalHandlers: Record<string, WrappedHandler> = {
        [RegisterHandler.data.name]: RegisterHandler.execute,
        [InitServerHandler.data.name]: InitServerHandler.execute,
        [CUSTOM_IDS.REGISTER_LINK_MODAL]: RegisterHandler.execute,
    };

    private static readonly buttonHandlers: Record<string, WrappedHandler> = {
        [CUSTOM_IDS.INIT_SERVER_PROCEED_BUTTON]: handleInitServerProceed,
        [CUSTOM_IDS.REGISTER_NEW_BUTTON]: handleRegisterNew,
        [CUSTOM_IDS.REGISTER_LINK_BUTTON]: handleRegisterLink,
        [CUSTOM_IDS.TOUR_FILE_PIREP_BUTTON]: handleTourFilePirep,
        [CUSTOM_IDS.TOUR_LEG_FILE_PIREP_BUTTON]: handleTourFilePirep,
    };

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
    private static async handleSelectMenu(_interaction: Interaction): Promise<void> {
        // No active select menu handlers — pilot role configuration removed.
    }

    /**
     * Handle modal submit interactions
     */
    private static async handleModalSubmit(interaction: Interaction): Promise<void> {
        if (!interaction.isModalSubmit()) return;

        const wrapped = new DiscordInteraction(interaction);

        const handler = this.modalHandlers[interaction.customId];
        if (handler) {
            await handler(wrapped);
            return;
        }

        if (interaction.customId === CUSTOM_IDS.MEMBERSHIP_JOIN_MODAL) {
            const MembershipJoinHandler = await import("../commands/membershipJoinModalHandler");
            await MembershipJoinHandler.execute(wrapped);
            return;
        }

        // Check if it's a PIREP modal with encoded mode_id (format: pirepModal_modeId)
        if (interaction.customId.startsWith(CUSTOM_IDS.PIREP_MODAL)) {
            await PirepModalHandler.execute(wrapped);
            return;
        }

        await this.handleDynamicModal(interaction, wrapped);
    }

    /**
     * Handle dynamic modals with custom IDs containing parameters.
     * Previously routed SyncUserModal; removed along with the syncUserToVA endpoint.
     */
    private static async handleDynamicModal(
        _interaction: Interaction,
        _wrapped: DiscordInteraction
    ): Promise<void> {
        // No active dynamic modal handlers.
    }

    /**
     * Handle button interactions
     */
    private static async handleButton(interaction: Interaction): Promise<void> {
        if (!interaction.isButton()) return;

        const wrapped = new DiscordInteraction(interaction);

        const handler = this.buttonHandlers[interaction.customId];
        if (handler) {
            await handler(wrapped);
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
