import * as dotenv from "dotenv";
import { getDeployableCommandNames, validateCommands } from "./commands/registry";
import { CommandDeploymentService, DeploymentScope } from "./services/deploymentService";
import { logger, errorFields } from "./infra/logger";

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
        logger.error("command_deploy_config_missing", { env_var: "DISCORD_BOT_CLIENT_ID" });
        process.exit(1);
    }

    if (!token) {
        logger.error("command_deploy_config_missing", { env_var: "DISCORD_BOT_TOKEN" });
        process.exit(1);
    }

    // Check for deployment mode from command line arguments
    const args = process.argv.slice(2);
    const mode = args[0]; // 'local' or 'global'

    try {
        validateCommands();
        const commandNames = getDeployableCommandNames();

        logger.info("command_deploy_cli_started", {
            command_count: commandNames.length,
            commands: commandNames,
        });

        // Determine deployment scope
        let scope: DeploymentScope;
        let scopeLabel: string;

        if (mode === "global") {
            scope = { type: "global" };
            scopeLabel = "Global (all servers)";
            logger.info("command_deploy_cli_scope_selected", { scope: "global" });
        } else if (mode === "local" || guildId) {
            if (!guildId) {
                logger.error("command_deploy_config_missing", { env_var: "GUILD_ID", scope: "guild" });
                process.exit(1);
            }
            scope = { type: "guild", guildId };
            scopeLabel = `Guild: ${guildId}`;
            logger.info("command_deploy_cli_scope_selected", { scope: "guild", guild_id: guildId });
        } else {
            scope = { type: "global" };
            scopeLabel = "Global (all servers)";
            logger.info("command_deploy_cli_scope_selected", { scope: "global", defaulted: true });
        }

        logger.info("command_deploy_cli_scope", { scope: scopeLabel });

        const service = new CommandDeploymentService({ clientId, token });
        const result = await service.deploy(scope);

        if (!result.success) {
            throw new Error(result.message);
        }

        logger.info("command_deploy_cli_completed", {
            result: "success",
            command_count: result.commandCount,
            scope: result.scope,
        });

        if (mode === "global" || !guildId) {
            logger.info("command_deploy_global_propagation_notice");
        } else {
            logger.info("command_deploy_guild_available_notice");
        }

    } catch (error) {
        logger.error("command_deploy_cli_failed", errorFields(error));
        process.exit(1);
    }
}

// Run deployment
deployCommands();
