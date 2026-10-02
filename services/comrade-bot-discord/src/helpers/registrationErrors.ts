import {
    ActionRowBuilder,
    ButtonBuilder,
    ButtonStyle,
} from "discord.js";
import type { components } from "../generated/politburo-api";
import { CUSTOM_IDS } from "../configs/constants";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { PolitburoApiError } from "./PolitburoApiError";

export type CreateUserErrorCode = components["schemas"]["CreateUserErrorCode"];

const CREATE_USER_ERROR_MESSAGES: Record<CreateUserErrorCode, string> = {
    INVALID_REQUEST:
        "❌ **Invalid registration**\nCheck your IFC username and flight route format (`ORIG-DEST`, four-letter ICAO codes).",
    FLIGHT_PROOF_FAILED:
        "❌ **Flight verification failed**\nThe route you provided doesn't match your most recent **complete** online logbook flight. Check Infinite Flight and try again.",
    REGISTRATION_FAILED:
        "❌ **Registration failed**\nWe couldn't complete registration. Please try again later or contact support.",
    UNAUTHORIZED:
        "❌ **Authorization failed**\nThe bot could not authenticate with Politburo. Contact server staff.",
    BANNED:
        "🚫 **Registration blocked**\nThis Discord account cannot register with Comrade Bot.",
    MISSING_DISCORD_CONTEXT:
        "❌ **Registration unavailable**\nMissing Discord context for this request. Try again from a server channel or contact support.",
    IF_USER_NOT_FOUND:
        "❌ **IFC user not found**\nThe provided IFC username was not found. Check your spelling and try again.",
    USER_ALREADY_REGISTERED:
        "❌ **Already registered**\nYour Discord account is already registered with Comrade Bot. Use `/status` to view your details.",
    IFC_ALREADY_LINKED:
        "❌ **IFC username taken**\nThis IFC username is already linked to another Discord account. If it is yours, report it for operator review.",
    AUTH_LOOKUP_FAILED:
        "⚠️ **Registration temporarily unavailable**\nPlease try again in a few minutes. If this persists, contact support.",
    IF_UNAVAILABLE:
        "⚠️ **Infinite Flight unavailable**\nRegistration needs Infinite Flight services, which are temporarily unavailable. Try again later.",
};

export type ReplyToCreateUserErrorOptions = {
    /** IFC username from the modal; used for the occupied-IFC report button. */
    ifcUsername?: string;
};

export async function replyToCreateUserError(
    interaction: DiscordInteraction,
    error: PolitburoApiError,
    options: ReplyToCreateUserErrorOptions = {},
): Promise<void> {
    const code = error.code as CreateUserErrorCode;
    const content =
        CREATE_USER_ERROR_MESSAGES[code] ??
        `❌ **Registration failed**\n${error.message}`;

    if (code === "IFC_ALREADY_LINKED" && options.ifcUsername) {
        const reportButton = new ButtonBuilder()
            .setCustomId(`${CUSTOM_IDS.REPORT_OCCUPIED_IFC_PREFIX}${options.ifcUsername}`)
            .setLabel("Report this username")
            .setStyle(ButtonStyle.Danger);
        const row = new ActionRowBuilder<ButtonBuilder>().addComponents(reportButton);
        await interaction.reply({
            content,
            components: [row],
            ephemeral: true,
        });
        return;
    }

    await interaction.reply({
        content,
        ephemeral: true,
    });
}

export function isCreateUserErrorCode(code: string): code is CreateUserErrorCode {
    return code in CREATE_USER_ERROR_MESSAGES;
}
