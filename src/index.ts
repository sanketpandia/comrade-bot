import * as dotenv from "dotenv";
import { BotClient } from "./bot/BotClient";
import { loadBotConfig } from "./configs/env";

// Load environment variables
dotenv.config();

/**
 * Main entry point for Comrade Bot
 */
async function main() {
    const config = loadBotConfig();

    // Initialize and start bot
    const bot = new BotClient(config);

    // Handle graceful shutdown
    process.on("SIGINT", async () => {
        console.log("\n Received SIGINT signal");
        await bot.stop();
        process.exit(0);
    });

    process.on("SIGTERM", async () => {
        console.log("\n Received SIGTERM signal");
        await bot.stop();
        process.exit(0);
    });

    // Handle uncaught errors
    process.on("uncaughtException", (error) => {
        console.error("Uncaught Exception:", error);
        process.exit(1);
    });

    process.on("unhandledRejection", (reason, promise) => {
        console.error("Unhandled Rejection at:", promise, "reason:", reason);
    });

    // Start the bot
    await bot.start();
}

// Run the bot
main().catch((error) => {
    console.error("Fatal error:", error);
    process.exit(1);
});
 
