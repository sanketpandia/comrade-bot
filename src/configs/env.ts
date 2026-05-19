export interface BotConfig {
    discordBotToken: string;
    discordClientId: string;
    apiUrl: string;
    apiKey?: string;
    guildId?: string;
}

function requiredEnv(name: string): string {
    const value = process.env[name];
    if (!value) {
        throw new Error(`${name} is not set in environment variables`);
    }
    return value;
}

export function loadBotConfig(): BotConfig {
    return {
        discordBotToken: requiredEnv("DISCORD_BOT_TOKEN"),
        discordClientId: requiredEnv("DISCORD_BOT_CLIENT_ID"),
        apiUrl: process.env.API_URL ?? "http://localhost:8080",
        apiKey: process.env.API_KEY,
        guildId: process.env.GUILD_ID,
    };
}
