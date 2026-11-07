import { APIEmbed, InteractionEditReplyOptions, SlashCommandBuilder } from "discord.js";
import { ApiService } from "../services/apiService";
import { DiscordInteraction } from "../types/DiscordInteraction";
import {
    formatCrewLine,
    formatUtcShort,
    handleLiveMemberError,
    LIVE_PAGE_SIZE,
    paginationFooter,
    withVaLiveOrg,
} from "../helpers/liveCommands";

export const data = new SlashCommandBuilder()
    .setName("schedules")
    .setDescription("List open Live schedules for an aircraft registration")
    .addStringOption(option =>
        option
            .setName("registration")
            .setDescription("Aircraft registration (e.g. PH-ZNB)")
            .setRequired(true),
    )
    .addIntegerOption(option =>
        option.setName("page").setDescription("Page number").setMinValue(1).setRequired(false),
    );

export function schedulesPayload(registration: string, page: number): string {
    return `${registration}|${page}`;
}

export function parseSchedulesPayload(payload: string): { registration: string; page: number } {
    const pipe = payload.lastIndexOf("|");
    if (pipe < 0) {
        return { registration: payload, page: 1 };
    }
    const registration = payload.slice(0, pipe);
    const page = Number.parseInt(payload.slice(pipe + 1), 10);
    return { registration, page: Number.isFinite(page) && page > 0 ? page : 1 };
}

export async function replySchedulesPage(
    editReply: (content: InteractionEditReplyOptions) => Promise<unknown>,
    meta: ReturnType<DiscordInteraction["getMetaInfo"]>,
    organizationId: string,
    orgName: string,
    registration: string,
    page: number,
): Promise<void> {
    const result = await ApiService.listVALiveOrganizationSchedules(meta, organizationId, {
        registration,
        page,
        pageSize: LIVE_PAGE_SIZE,
    });

    if (!result.data.length) {
        await editReply({
            embeds: [{
                title: `Schedules · ${registration}`,
                description: `No open schedules for **${registration}** in **${orgName}**.`,
                color: 0x0099ff,
                timestamp: new Date().toISOString(),
            }],
        });
        return;
    }

    const fields = result.data.map(row => {
        const route = `${row.originIcao}-${row.destinationIcao}`;
        const departure = row.isTimeless ? "Timeless" : formatUtcShort(row.scheduledDepartureUtc);
        return {
            name: `${row.callsign} · ${route}`.slice(0, 256),
            value: [
                `Status: **${row.status}**`,
                `Type: **${row.flightType}**`,
                `Departure: **${departure}**`,
                `Crew: ${formatCrewLine(row.crew)}`,
            ].join("\n"),
            inline: false,
        };
    });

    const baseEmbed: APIEmbed = {
        title: `Schedules · ${registration}`,
        description: `Organisation: **${orgName}**`,
        color: 0x0099ff,
        fields,
        footer: { text: paginationFooter(result.page, result.totalPages, "schedules") },
        timestamp: new Date().toISOString(),
    };

    await editReply({ embeds: [baseEmbed] });
}

export async function execute(interaction: DiscordInteraction): Promise<void> {
    const chat = interaction.getChatInputInteraction();
    if (!chat) return;

    await chat.deferReply();
    const registration = chat.options.getString("registration", true).trim();
    const page = chat.options.getInteger("page") ?? 1;
    const meta = interaction.getMetaInfo();

    try {
        await withVaLiveOrg(
            meta,
            "schedules",
            schedulesPayload(registration, page),
            `Registration **${registration}** is linked to more than one org. Pick the correct one.`,
            async (orgId, orgName) => {
                await replySchedulesPage(chat.editReply.bind(chat), meta, orgId, orgName, registration, page);
            },
            chat.editReply.bind(chat),
        );
    } catch (err: unknown) {
        await handleLiveMemberError(chat.editReply.bind(chat), err, "Could not list schedules");
    }
}

export async function handleSchedulesOrgSelect(interaction: DiscordInteraction, payload: string): Promise<void> {
    const select = interaction.getSelectMenuInteraction();
    if (!select) return;

    await select.deferReply();
    const { registration, page } = parseSchedulesPayload(payload);
    const orgId = select.values[0];
    const meta = interaction.getMetaInfo();

    try {
        const connections = await ApiService.listVALiveConnections(meta);
        const orgName = connections.find(c => c.id === orgId)?.name ?? "Organisation";
        await replySchedulesPage(select.editReply.bind(select), meta, orgId, orgName, registration, page);
    } catch (err: unknown) {
        await handleLiveMemberError(select.editReply.bind(select), err, "Could not list schedules");
    }
}
