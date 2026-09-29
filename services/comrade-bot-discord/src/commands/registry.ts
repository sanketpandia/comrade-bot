import {
    SlashCommandBuilder,
    SlashCommandOptionsOnlyBuilder,
    SlashCommandSubcommandsOnlyBuilder,
} from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { logger } from "../infra/logger";

import * as status from "./status";
import * as register from "./register";
import * as logbook from "./logbook";
import * as initserver from "./initServer";
import * as live from "./live";
import * as rollout from "./rollout";
import * as stats from "./stats";
import * as log from "./log";
import * as help from "./help";
import * as dashboard from "./dashboard";
import * as botstatus from "./botstatus";
import * as events from "./events";
import * as tour from "./tour";
import * as tourLeg from "./tour_leg";

export type SlashCommandData =
    | SlashCommandBuilder
    | SlashCommandOptionsOnlyBuilder
    | SlashCommandSubcommandsOnlyBuilder;

export interface CommandRegistryEntry {
    name: string;
    data: SlashCommandData;
    execute: (interaction: DiscordInteraction) => Promise<void>;
    deploy: boolean;
    category?: string;
}

function entry(command: { data: SlashCommandData; execute: (interaction: DiscordInteraction) => Promise<void> }, category?: string): CommandRegistryEntry {
    return {
        name: command.data.name,
        data: command.data,
        execute: command.execute,
        deploy: true,
        category,
    };
}

export const commandRegistry: CommandRegistryEntry[] = [
    entry(status, "user"),
    entry(register, "onboarding"),
    entry(logbook, "user"),
    entry(initserver, "admin"),
    entry(live, "operations"),
    entry(rollout, "admin"),
    entry(stats, "user"),
    entry(log, "operations"),
    entry(help, "user"),
    entry(botstatus, "operations"),
    entry(dashboard, "user"),
    entry(events, "events"),
    entry(tour, "events"),
    entry(tourLeg, "events"),
];

export const deployableCommandRegistry = commandRegistry.filter(command => command.deploy);

export const commandMap: Record<string, CommandRegistryEntry> = Object.fromEntries(
    commandRegistry.map(command => [command.name, command])
);

export function getDeployableCommandsJSON() {
    return deployableCommandRegistry.map(command => command.data.toJSON());
}

export function getCommandNames(): string[] {
    return commandRegistry.map(command => command.name);
}

export function getDeployableCommandNames(): string[] {
    return deployableCommandRegistry.map(command => command.name);
}

export function validateCommands(): void {
    const names = new Set<string>();

    for (const command of commandRegistry) {
        if (names.has(command.name)) {
            throw new Error(`Duplicate command name: ${command.name}`);
        }
        names.add(command.name);

        if (!command.data.description) {
            throw new Error(`Command ${command.name} missing description`);
        }

        if (command.name.length < 1 || command.name.length > 32) {
            throw new Error(`Command ${command.name} has invalid name length`);
        }
    }

    logger.info("commands_validated", {
        command_count: commandRegistry.length,
        deployable_command_count: deployableCommandRegistry.length,
    });
}
