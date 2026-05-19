import {
    deployableCommandRegistry,
    getDeployableCommandNames,
    getDeployableCommandsJSON,
    validateCommands,
} from "../commands/registry";

/**
 * Command registry
 * Centralized list of all bot commands
 * Supports SlashCommandBuilder, SlashCommandOptionsOnlyBuilder, and SlashCommandSubcommandsOnlyBuilder
 */
export const COMMANDS = deployableCommandRegistry.map(command => command.data);

/**
 * Get all commands in JSON format for Discord API
 */
export function getCommandsJSON() {
    return getDeployableCommandsJSON();
}

/**
 * Get command names
 */
export function getCommandNames(): string[] {
    return getDeployableCommandNames();
}

/**
 * Validate all commands
 */
export { validateCommands };
