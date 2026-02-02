import { SlashCommandBuilder, SlashCommandOptionsOnlyBuilder, SlashCommandSubcommandsOnlyBuilder } from "discord.js";

// Import all command data
import { data as statusCmd } from "../commands/status";
import { data as logbookCmd } from "../commands/logbook";
import { data as registerCmd } from "../commands/register";
import { data as initServerCmd } from "../commands/initServer";
import { data as pilotCmd } from "../commands/pilot";
import { data as liveCmd } from "../commands/live";
import { data as rolloutCmd } from "../commands/rollout";
import { data as statsCmd } from "../commands/stats";
import { data as logCmd } from "../commands/log";
import { data as helpCmd } from "../commands/help";
import { data as dashboardCmd } from "../commands/dashboard";
import { data as membershipCmd } from "../commands/membership";

/**
 * Command registry
 * Centralized list of all bot commands
 * Supports SlashCommandBuilder, SlashCommandOptionsOnlyBuilder, and SlashCommandSubcommandsOnlyBuilder
 */
export const COMMANDS: (SlashCommandBuilder | SlashCommandOptionsOnlyBuilder | SlashCommandSubcommandsOnlyBuilder)[] = [
    statusCmd,
    registerCmd,
    logbookCmd,
    initServerCmd,
    liveCmd,
    pilotCmd,
    rolloutCmd,
    statsCmd,
    logCmd,
    helpCmd,
    dashboardCmd,
    membershipCmd
];

/**
 * Get all commands in JSON format for Discord API
 */
export function getCommandsJSON() {
    return COMMANDS.map(cmd => cmd.toJSON());
}

/**
 * Get command names
 */
export function getCommandNames(): string[] {
    return COMMANDS.map(cmd => cmd.name);
}

/**
 * Validate all commands
 */
export function validateCommands(): void {
    const names = new Set<string>();

    for (const cmd of COMMANDS) {
        // Check for duplicate names
        if (names.has(cmd.name)) {
            throw new Error(`Duplicate command name: ${cmd.name}`);
        }
        names.add(cmd.name);

        // Validate command structure
        if (!cmd.description) {
            throw new Error(`Command ${cmd.name} missing description`);
        }

        if (cmd.name.length < 1 || cmd.name.length > 32) {
            throw new Error(`Command ${cmd.name} has invalid name length`);
        }
    }

    console.log(`✅ Validated ${COMMANDS.length} commands`);
}
