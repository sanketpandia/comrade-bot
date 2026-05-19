import client from "prom-client";

const register = new client.Registry();

client.collectDefaultMetrics({
    register,
    prefix: "comrade_bot_",
});

const commandExecutions = new client.Counter({
    name: "comrade_bot_command_executions_total",
    help: "Total Discord slash command executions handled by Comrade Bot.",
    labelNames: ["command", "result", "interaction_type"] as const,
    registers: [register],
});

const commandDuration = new client.Histogram({
    name: "comrade_bot_command_duration_seconds",
    help: "Discord slash command execution duration in seconds.",
    labelNames: ["command", "result", "interaction_type"] as const,
    buckets: [0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30],
    registers: [register],
});

const interactions = new client.Counter({
    name: "comrade_bot_interactions_total",
    help: "Total Discord interactions routed by Comrade Bot.",
    labelNames: ["interaction_type", "result"] as const,
    registers: [register],
});

const discordEvents = new client.Counter({
    name: "comrade_bot_discord_events_total",
    help: "Total Discord client lifecycle events observed by Comrade Bot.",
    labelNames: ["event", "result"] as const,
    registers: [register],
});

export type CommandResult = "success" | "error" | "unknown_command" | "validation_error" | "permission_denied";
export type InteractionResult = "success" | "error" | "ignored" | "unknown_command";

export const metrics = {
    register,

    recordCommand(command: string, result: CommandResult, interactionType: string, durationMs: number): void {
        const safeCommand = command || "unknown";
        commandExecutions.inc({ command: safeCommand, result, interaction_type: interactionType });
        commandDuration.observe({ command: safeCommand, result, interaction_type: interactionType }, durationMs / 1000);
    },

    recordInteraction(interactionType: string, result: InteractionResult): void {
        interactions.inc({ interaction_type: interactionType, result });
    },

    recordDiscordEvent(event: string, result: "success" | "error" | "warning" = "success"): void {
        discordEvents.inc({ event, result });
    },
};
