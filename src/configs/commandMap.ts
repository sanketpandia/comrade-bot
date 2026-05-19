import { DiscordInteraction } from "../types/DiscordInteraction";
import { commandMap as registryCommandMap } from "../commands/registry";

export type CommandHandler = {
    execute: (interaction: DiscordInteraction) => Promise<void>;
}

export const commandMap: Record<string, CommandHandler> = registryCommandMap;
