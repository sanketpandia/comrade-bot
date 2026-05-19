import { REST, Routes } from "discord.js";
import { getDeployableCommandNames, getDeployableCommandsJSON, validateCommands } from "../commands/registry";

export type DeploymentScope =
    | { type: "guild"; guildId: string }
    | { type: "global" };

export interface DeploymentCredentials {
    clientId: string;
    token: string;
}

/**
 * Service for deploying Discord slash commands
 */
export class CommandDeploymentService {
    private clientId: string;
    private token: string;

    constructor(credentials: DeploymentCredentials) {
        this.clientId = credentials.clientId;
        this.token = credentials.token;
    }

    /**
     * Deploy commands to Discord for the requested scope.
     */
    async deploy(scope: DeploymentScope): Promise<DeploymentResult> {
        try {
            validateCommands();

            const commands = getDeployableCommandsJSON();
            const commandNames = getDeployableCommandNames();

            const rest = new REST().setToken(this.token);
            const route = scope.type === "guild"
                ? Routes.applicationGuildCommands(this.clientId, scope.guildId)
                : Routes.applicationCommands(this.clientId);

            console.log(`[CommandDeploymentService] Deploying ${commands.length} commands (${scope.type})`);

            const deployedCommands = await rest.put(route, { body: commands }) as any[];

            console.log(`[CommandDeploymentService] Successfully deployed ${deployedCommands.length} commands (${scope.type})`);

            return {
                success: true,
                commandCount: deployedCommands.length,
                commandNames,
                guildId: scope.type === "guild" ? scope.guildId : undefined,
                scope: scope.type,
                message: scope.type === "guild"
                    ? `Successfully deployed ${deployedCommands.length} commands`
                    : `Successfully deployed ${deployedCommands.length} commands globally (may take up to 1 hour to propagate)`
            };

        } catch (error: any) {
            const message = error?.message || "Unknown deployment error";
            console.error(`[CommandDeploymentService] Deployment failed (${scope.type}):`, message);

            return {
                success: false,
                commandCount: 0,
                commandNames: [],
                guildId: scope.type === "guild" ? scope.guildId : undefined,
                scope: scope.type,
                message,
                error
            };
        }
    }

    /**
     * Deploy commands to a specific guild.
     */
    async deployToGuild(guildId: string): Promise<DeploymentResult> {
        return this.deploy({ type: "guild", guildId });
    }

    /**
     * Deploy commands globally (takes ~1 hour to propagate).
     */
    async deployGlobally(): Promise<DeploymentResult> {
        return this.deploy({ type: "global" });
    }
}

export interface DeploymentResult {
    success: boolean;
    commandCount: number;
    commandNames: string[];
    guildId?: string;
    scope: DeploymentScope["type"];
    message: string;
    error?: any;
}
