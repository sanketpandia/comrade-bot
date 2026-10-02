import "./loadEnv";
import { BotClient } from "./bot/BotClient";
import { loadBotConfig } from "./configs/env";
import { logger, errorFields } from "./infra/logger";
import { MetricsServer } from "./infra/metricsServer";

/**
 * Main entry point for Comrade Bot
 */
async function main() {
    const config = loadBotConfig();
    if (!config.apiKey) {
        logger.warn("api_key_missing", {
            hint: "Set API_KEY to an active UUID from Politburo api_keys (see services/politburo/migrations/README.md)",
        });
    }
    const metricsServer = new MetricsServer(config.metrics);

    // Initialize and start bot
    const bot = new BotClient(config);

    await metricsServer.start();

    // Handle graceful shutdown
    process.on("SIGINT", async () => {
        logger.info("shutdown_signal_received", { signal: "SIGINT" });
        await bot.stop();
        await metricsServer.stop();
        process.exit(0);
    });

    process.on("SIGTERM", async () => {
        logger.info("shutdown_signal_received", { signal: "SIGTERM" });
        await bot.stop();
        await metricsServer.stop();
        process.exit(0);
    });

    // Handle uncaught errors
    process.on("uncaughtException", (error) => {
        logger.error("uncaught_exception", errorFields(error));
        process.exit(1);
    });

    process.on("unhandledRejection", (reason, promise) => {
        logger.error("unhandled_rejection", {
            ...errorFields(reason),
            promise: String(promise),
        });
    });

    // Start the bot
    await bot.start();
}

// Run the bot
main().catch((error) => {
    logger.error("fatal_error", errorFields(error));
    process.exit(1);
});
 
