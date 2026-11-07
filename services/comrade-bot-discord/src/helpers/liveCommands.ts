import {
    ActionRowBuilder,
    APIEmbed,
    InteractionEditReplyOptions,
    StringSelectMenuBuilder,
    StringSelectMenuOptionBuilder,
} from "discord.js";
import { ApiService } from "../services/apiService";
import { CUSTOM_IDS } from "../configs/constants";
import { UnauthorizedError } from "./UnauthorizedException";
import { PermissionDeniedError } from "./PermissionDeniedException";
import { PolitburoApiError } from "./PolitburoApiError";
import type { MetaInfo } from "../types/DiscordInteraction";
import type { LiveScheduleItem, VALiveConnection } from "../generated/politburoClient";

export const LIVE_PAGE_SIZE = 10;

export type LiveOrgCommand = "aircrafts" | "schedules" | "history";

export type LiveScheduleCrewMemberLoose = NonNullable<LiveScheduleItem["crew"]>[number];

export function liveOrgSelectCustomId(command: LiveOrgCommand, payload: string): string {
    return `${CUSTOM_IDS.LIVE_ORG_SELECT_PREFIX}${command}|${encodeURIComponent(payload)}`;
}

export function parseLiveOrgSelectCustomId(
    customId: string,
): { command: LiveOrgCommand; payload: string } | null {
    if (!customId.startsWith(CUSTOM_IDS.LIVE_ORG_SELECT_PREFIX)) {
        return null;
    }
    const rest = customId.slice(CUSTOM_IDS.LIVE_ORG_SELECT_PREFIX.length);
    const pipe = rest.indexOf("|");
    if (pipe < 0) {
        return null;
    }
    const command = rest.slice(0, pipe) as LiveOrgCommand;
    if (command !== "aircrafts" && command !== "schedules" && command !== "history") {
        return null;
    }
    return { command, payload: decodeURIComponent(rest.slice(pipe + 1)) };
}

export function buildOrgSelectRow(
    command: LiveOrgCommand,
    payload: string,
    connections: VALiveConnection[],
    placeholder: string,
): ActionRowBuilder<StringSelectMenuBuilder> {
    const options = connections.slice(0, 25).map(conn =>
        new StringSelectMenuOptionBuilder().setLabel(conn.name).setValue(conn.id),
    );
    const menu = new StringSelectMenuBuilder()
        .setCustomId(liveOrgSelectCustomId(command, payload))
        .setPlaceholder(placeholder)
        .addOptions(options);
    return new ActionRowBuilder<StringSelectMenuBuilder>().addComponents(menu);
}

export function formatUtcShort(iso: string | undefined | null): string {
    if (!iso) {
        return "—";
    }
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) {
        return "—";
    }
    return `${d.toISOString().slice(0, 19).replace("T", " ")} UTC`;
}

export function formatCrewLine(crew: LiveScheduleCrewMemberLoose[] | undefined): string {
    if (!crew?.length) {
        return "—";
    }
    const parts = crew.map(member => {
        const username = member.username?.trim();
        const role = member.role?.trim() || "Crew";
        if (username) {
            return `${username} (${role})`;
        }
        return role;
    });
    return parts.join(", ");
}

export function formatFlightTimeMinutes(minutes: number | undefined): string {
    const total = Math.max(0, Math.round(Number(minutes) || 0));
    const h = Math.floor(total / 60);
    const m = total % 60;
    if (h > 0 && m > 0) {
        return `${h}h ${m}m`;
    }
    if (h > 0) {
        return `${h}h`;
    }
    return `${m}m`;
}

export function flightStatusLabel(status: number | undefined): string {
    switch (status) {
        case 0:
            return "Active";
        case 1:
            return "Completed";
        case 2:
            return "Cancelled";
        default:
            return "Unknown";
    }
}

export function paginationFooter(page: number, totalPages: number, commandName: string): string {
    const safeTotal = Math.max(1, totalPages);
    if (safeTotal <= 1) {
        return `Page ${page} of ${safeTotal}`;
    }
    const next = page < safeTotal ? page + 1 : undefined;
    const hint = next ? ` Use \`/${commandName} ... page:${next}\` for more.` : "";
    return `Page ${page} of ${safeTotal}.${hint}`;
}

export async function handleLiveMemberError(
    editReply: (content: { embeds?: APIEmbed[] }) => Promise<unknown>,
    err: unknown,
    title: string,
): Promise<void> {
    if (err instanceof UnauthorizedError) {
        await editReply({
            embeds: [{
                title: "Not Authorized",
                description: err.message,
                color: 0xff0000,
                timestamp: new Date().toISOString(),
            }],
        });
        return;
    }
    if (err instanceof PermissionDeniedError) {
        await editReply({
            embeds: [{
                title: "Membership required",
                description: `${err.message}\n\nUse \`/register\` first.`,
                color: 0xff9900,
                timestamp: new Date().toISOString(),
            }],
        });
        return;
    }
    if (PolitburoApiError.isPolitburoApiError(err)) {
        await editReply({
            embeds: [{
                title,
                description: err.message,
                color: 0xff0000,
                timestamp: new Date().toISOString(),
            }],
        });
        return;
    }
    const message = err instanceof Error ? err.message : String(err);
    await editReply({
        embeds: [{
            title: "Error",
            description: message,
            color: 0xff0000,
            timestamp: new Date().toISOString(),
        }],
    });
}

export function noLiveOrgEmbed(): APIEmbed {
    return {
        title: "No Live org linked",
        description: "This VA has no Infinite Flight Live organisation connected yet.",
        color: 0xff9900,
        timestamp: new Date().toISOString(),
    };
}

export async function withVaLiveOrg<T>(
    meta: MetaInfo,
    command: LiveOrgCommand,
    payload: string,
    chooseDescription: string,
    runForOrg: (orgId: string, orgName: string) => Promise<T>,
    deferEdit: (content: InteractionEditReplyOptions) => Promise<unknown>,
): Promise<void> {
    const connections = await ApiService.listVALiveConnections(meta);
    if (!connections.length) {
        await deferEdit({ embeds: [noLiveOrgEmbed()] });
        return;
    }
    if (connections.length === 1) {
        await runForOrg(connections[0].id, connections[0].name);
        return;
    }
    await deferEdit({
        embeds: [{
            title: "Choose organisation",
            description: chooseDescription,
            color: 0x0099ff,
            timestamp: new Date().toISOString(),
        }],
        components: [buildOrgSelectRow(command, payload, connections, "Select organisation")],
    });
}
