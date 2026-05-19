export interface BotConfig {
    discordBotToken: string;
    discordClientId: string;
    apiUrl: string;
    apiKey?: string;
    guildId?: string;
    appEnv: string;
    logLevel: string;
    metrics: {
        enabled: boolean;
        host: string;
        port: number;
    };
}

function requiredEnv(name: string): string {
    const value = process.env[name];
    if (!value) {
        throw new Error(`${name} is not set in environment variables`);
    }
    return value;
}

export function loadBotConfig(): BotConfig {
    const metricsPort = Number.parseInt(process.env.METRICS_PORT || "9091", 10);

    return {
        discordBotToken: requiredEnv("DISCORD_BOT_TOKEN"),
        discordClientId: requiredEnv("DISCORD_BOT_CLIENT_ID"),
        apiUrl: process.env.API_URL ?? "http://localhost:8080",
        apiKey: process.env.API_KEY,
        guildId: process.env.GUILD_ID,
        appEnv: process.env.APP_ENV || process.env.NODE_ENV || "development",
        logLevel: process.env.LOG_LEVEL || "info",
        metrics: {
            enabled: (process.env.METRICS_ENABLED || "true").toLowerCase() !== "false",
            host: process.env.METRICS_HOST || "0.0.0.0",
            port: Number.isFinite(metricsPort) ? metricsPort : 9091,
        },
    };
}
