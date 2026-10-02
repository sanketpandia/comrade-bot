import type { components } from "../generated/politburo-api";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { PolitburoApiError } from "./PolitburoApiError";

export type CreateServerErrorCode = components["schemas"]["CreateServerErrorCode"];

const CREATE_SERVER_ERROR_MESSAGES: Record<CreateServerErrorCode, string> = {
    INVALID_REQUEST:
        "❌ **Invalid request**\nCheck your VA Code / ID format (3–5 letters or numbers, e.g. `IFE`).",
    INVALID_VA_CODE:
        "❌ **Invalid VA Code / ID**\nUse 3–5 uppercase letters or numbers (e.g. `IFE`, `DAL`).",
    USER_NOT_FOUND:
        "👋 Please run `/register` first, then try `/initserver` again.",
    UNAUTHORIZED:
        "❌ **Authorization failed**\nThe bot could not authenticate with Politburo. Contact server staff.",
    MISSING_DISCORD_CONTEXT:
        "🔒 Discord context was missing. Please try `/initserver` again inside the server you want to set up.",
    SERVER_ALREADY_VA:
        "✅ This Discord server is already initialized. Use `/dashboard` to continue setup in Vizburo.",
    VA_CODE_TAKEN:
        "❌ That VA Code / ID is already in use. Choose your VA's official unique code or contact support.",
    INIT_FAILED:
        "⚠️ **Server setup failed**\nPlease try again later or contact support.",
};

export async function replyToCreateServerError(
    interaction: DiscordInteraction,
    error: PolitburoApiError,
): Promise<void> {
    const code = error.code as CreateServerErrorCode;
    const content =
        CREATE_SERVER_ERROR_MESSAGES[code] ??
        `❌ **Server setup failed**\n${error.message}`;

    await interaction.reply({
        content,
        ephemeral: true,
    });
}
