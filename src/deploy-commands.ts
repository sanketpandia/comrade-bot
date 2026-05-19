import * as dotenv from "dotenv";
import { getDeployableCommandNames, validateCommands } from "./commands/registry";
import { CommandDeploymentService, DeploymentScope } from "./services/deploymentService";

dotenv.config();

/**
 * Deploy slash commands to Discord
 * Supports both guild-specific (dev) and global (prod) deployment
 *
 * Usage:
 *   npm run deploy              - Deploy to guild if GUILD_ID is set, otherwise global
 *   npm run deploy:local        - Deploy to guild (requires GUILD_ID)
 *   npm run deploy:global       - Deploy globally to all servers
 */
async function deployCommands() {
    // Validate environment variables
    const clientId = process.env.DISCORD_BOT_CLIENT_ID;
    const token = process.env.DISCORD_BOT_TOKEN;
    const guildId = process.env.GUILD_ID;

    if (!clientId) {
        console.error("❌ DISCORD_BOT_CLIENT_ID is not set");
        process.exit(1);
    }

    if (!token) {
        console.error("❌ DISCORD_BOT_TOKEN is not set");
        process.exit(1);
    }

    // Check for deployment mode from command line arguments
    const args = process.argv.slice(2);
    const mode = args[0]; // 'local' or 'global'

    try {
        validateCommands();
        const commandNames = getDeployableCommandNames();

        console.log(`\n📦 Deploying ${commandNames.length} slash commands...`);
        console.log(`📋 Commands: ${commandNames.join(", ")}`);

        // Determine deployment scope
        let scope: DeploymentScope;
        let scopeLabel: string;

        if (mode === "global") {
            scope = { type: "global" };
            scopeLabel = "Global (all servers)";
            console.log("🌍 Mode: GLOBAL deployment");
        } else if (mode === "local" || guildId) {
            if (!guildId) {
                console.error("❌ GUILD_ID environment variable is required for local deployment");
                process.exit(1);
            }
            scope = { type: "guild", guildId };
            scopeLabel = `Guild: ${guildId}`;
            console.log("🏠 Mode: LOCAL (guild-specific) deployment");
        } else {
            scope = { type: "global" };
            scopeLabel = "Global (all servers)";
            console.log("🌍 Mode: GLOBAL deployment (default)");
        }

        console.log(`🎯 Scope: ${scopeLabel}`);

        const service = new CommandDeploymentService({ clientId, token });
        const result = await service.deploy(scope);

        if (!result.success) {
            throw new Error(result.message);
        }

        console.log(`\n✅ Successfully deployed ${result.commandCount} commands!`);

        if (mode === "global" || !guildId) {
            console.log("⏳ Note: Global commands may take up to 1 hour to update across all servers");
        } else {
            console.log("⚡ Guild commands are available immediately!");
        }

    } catch (error) {
        console.error("\n❌ Failed to deploy commands:", error);
        process.exit(1);
    }
}

// Run deployment
deployCommands();
