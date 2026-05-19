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
import { logger, errorFields } from "../infra/logger";
import { CommandResult, InteractionResult, metrics } from "../infra/metrics";

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
            } else {
                metrics.recordInteraction("unknown", "ignored");
            }
        } catch (error) {
            logger.error("interaction_route_failed", errorFields(error));
            await this.handleError(rawInteraction, error);
        }
    }

    /**
     * Handle select menu interactions
     */
    private static async handleSelectMenu(_interaction: Interaction): Promise<void> {
        metrics.recordInteraction("select_menu", "ignored");
        // No active select menu handlers — pilot role configuration removed.
    }

    /**
     * Handle modal submit interactions
     */
    private static async handleModalSubmit(interaction: Interaction): Promise<void> {
        if (!interaction.isModalSubmit()) return;

        const start = Date.now();
        const wrapped = new DiscordInteraction(interaction);
        let result: InteractionResult = "success";

        try {
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

            result = "ignored";
            await this.handleDynamicModal(interaction, wrapped);
        } catch (error) {
            result = "error";
            throw error;
        } finally {
            metrics.recordInteraction("modal_submit", result);
            logger.info("interaction_completed", {
                interaction_type: "modal_submit",
                result,
                guild_id: interaction.guildId || "DM",
                duration_ms: Date.now() - start,
            });
        }
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

        const start = Date.now();
        const wrapped = new DiscordInteraction(interaction);
        let result: InteractionResult = "success";

        try {
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
                return;
            }

            result = "ignored";
        } catch (error) {
            result = "error";
            throw error;
        } finally {
            metrics.recordInteraction("button", result);
            logger.info("interaction_completed", {
                interaction_type: "button",
                result,
                guild_id: interaction.guildId || "DM",
                duration_ms: Date.now() - start,
            });
        }
    }

    /**
     * Handle slash command interactions
     */
    private static async handleCommand(interaction: Interaction): Promise<void> {
        if (!interaction.isChatInputCommand()) return;

        const start = Date.now();
        const interactionType = "chat_input_command";
        let result: CommandResult = "success";

        const command = commandMap[interaction.commandName];
        if (!command) {
            result = "unknown_command";
            metrics.recordCommand("unknown", result, interactionType, Date.now() - start);
            metrics.recordInteraction(interactionType, result);
            logger.warn("unknown_command", {
                command: interaction.commandName,
                interaction_type: interactionType,
                guild_id: interaction.guildId || "DM",
                result,
            });
            return;
        }

        const wrapped = new DiscordInteraction(interaction);
        logger.info("command_started", {
            command: interaction.commandName,
            interaction_type: interactionType,
            guild_id: interaction.guildId || "DM",
        });

        try {
            await command.execute(wrapped);
        } catch (error) {
            result = "error";
            logger.error("command_failed", {
                command: interaction.commandName,
                interaction_type: interactionType,
                guild_id: interaction.guildId || "DM",
                result,
                ...errorFields(error),
            });
            throw error;
        } finally {
            const durationMs = Date.now() - start;
            metrics.recordCommand(interaction.commandName, result, interactionType, durationMs);
            metrics.recordInteraction(interactionType, result === "success" ? "success" : "error");
            logger.info("command_completed", {
                command: interaction.commandName,
                interaction_type: interactionType,
                guild_id: interaction.guildId || "DM",
                result,
                duration_ms: durationMs,
            });
        }
    }


    /**
     * Handle errors gracefully
     */
    private static async handleError(interaction: Interaction, error: unknown): Promise<void> {
        logger.error("interaction_error_reply", errorFields(error));

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
            logger.error("interaction_error_reply_failed", errorFields(replyError));
        }
    }
}
