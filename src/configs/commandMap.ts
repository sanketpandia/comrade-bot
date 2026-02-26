import * as status from "../commands/status";
import * as register from "../commands/register";
import * as logbook from "../commands/logbook";
import * as initserver from "../commands/initServer";
import * as live from "../commands/live";
import * as pilotmanage from "../commands/pilot";
import * as rollout from "../commands/rollout";
import * as stats from "../commands/stats";
import * as log from "../commands/log";
import * as help from "../commands/help";
import * as dashboard from "../commands/dashboard";
import * as membership from "../commands/membership";
import { DiscordInteraction } from "../types/DiscordInteraction";

export type CommandHandler = {
    execute: (interaction: DiscordInteraction) => Promise<void>;
}

export const commandMap: Record<string, CommandHandler> = {
    status,
    register,
    logbook,
    initserver,
	live,
    pilotmanage,
    rollout,
    stats,
    log,
    help,
    dashboard,
    membership
}
