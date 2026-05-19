import { Client, GatewayIntentBits, Events } from "discord.js";
import { InteractionRouter } from "../handlers/InteractionRouter";
import { BotConfig } from "../configs/env";
import { logger, errorFields } from "../infra/logger";
import { metrics } from "../infra/metrics";

/**
 * Bot client manager
 * Handles Discord client initialization and lifecycle
 */
export class BotClient {
    private client: Client;
    private token: string;

    constructor(config: BotConfig) {
        this.token = config.discordBotToken;
        this.client = new Client({
            intents: [GatewayIntentBits.Guilds]
        });

        this.setupEventHandlers();
    }

    /**
     * Setup Discord event handlers
     */
    private setupEventHandlers(): void {
        // Bot ready event
        this.client.once(Events.ClientReady, async (client) => {
            metrics.recordDiscordEvent("ready", "success");
            logger.info("discord_client_ready", { bot_tag: client.user.tag });

            // Wait a bit for guild cache to populate
            await new Promise(resolve => setTimeout(resolve, 1000));

            const guildCount = client.guilds.cache.size;
            logger.info("discord_guild_cache_ready", { guild_count: guildCount });

            if (guildCount === 0) {
                logger.warn("discord_no_guilds_found");
            }
        });

        // Interaction events
        this.client.on(Events.InteractionCreate, async (interaction) => {
            await InteractionRouter.route(interaction);
        });

        // Error handling
        this.client.on(Events.Error, (error) => {
            metrics.recordDiscordEvent("error", "error");
            logger.error("discord_client_error", errorFields(error));
        });

        this.client.on(Events.Warn, (warning) => {
            metrics.recordDiscordEvent("warn", "warning");
            logger.warn("discord_client_warning", { warning });
        });

        // Debug events (optional - remove in production)
        if (process.env.DEBUG === "true") {
            this.client.on(Events.Debug, (info) => {
                logger.debug("discord_client_debug", { info });
            });
        }
    }

    /**
     * Start the bot
     */
    async start(): Promise<void> {
        try {
            logger.info("bot_starting");
            await this.client.login(this.token);
        } catch (error) {
            logger.error("bot_start_failed", errorFields(error));
            throw error;
        }
    }

    /**
     * Stop the bot gracefully
     */
    async stop(): Promise<void> {
        logger.info("bot_stopping");
        this.client.destroy();
    }

    /**
     * Get the Discord client instance
     */
    getClient(): Client {
        return this.client;
    }
}
