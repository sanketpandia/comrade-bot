import { APIEmbed, InteractionEditReplyOptions, SlashCommandBuilder } from "discord.js";
import { ApiService } from "../services/apiService";
import { DiscordInteraction } from "../types/DiscordInteraction";
import {
    flightStatusLabel,
    formatFlightTimeMinutes,
    formatUtcShort,
    handleLiveMemberError,
    LIVE_PAGE_SIZE,
    liveModeRedirectPath,
    paginationFooter,
    withLiveModeLink,
    withVaLiveOrg,
} from "../helpers/liveCommands";
import { parseSchedulesPayload, schedulesPayload } from "./schedules";

export const data = new SlashCommandBuilder()
    .setName("history")
    .setDescription("List flight history for an aircraft registration")
    .addStringOption(option =>
        option
            .setName("registration")
            .setDescription("Aircraft registration (e.g. PH-ZNB)")
            .setRequired(true),
    )
    .addIntegerOption(option =>
        option.setName("page").setDescription("Page number").setMinValue(1).setRequired(false),
    );

export async function replyHistoryPage(
    editReply: (content: InteractionEditReplyOptions) => Promise<unknown>,
    meta: ReturnType<DiscordInteraction["getMetaInfo"]>,
    organizationId: string,
    orgName: string,
    registration: string,
    page: number,
): Promise<void> {
    const result = await ApiService.listVALiveOrganizationFlights(meta, organizationId, {
        registration,
        page,
        pageSize: LIVE_PAGE_SIZE,
    });

    if (!result.data.length) {
        await editReply({
            embeds: [{
                title: `History · ${registration}`,
                description: `No flights logged for **${registration}** in **${orgName}**.`,
                color: 0x0099ff,
                timestamp: new Date().toISOString(),
            }],
        });
        return;
    }

    const fields = result.data.map(row => {
        const route =
            row.originIcao && row.destinationIcao
                ? `${row.originIcao}-${row.destinationIcao}`
                : "—";
        const crew = row.crew?.map(m => (m.username?.trim() ? m.username.trim() : "")).filter(Boolean).join(", ") || "—";
        return {
            name: `${row.callsign} · ${route}`.slice(0, 256),
            value: [
                `Status: **${flightStatusLabel(row.status)}**`,
                `Departed: **${formatUtcShort(row.actualDepartureUtc ?? undefined)}**`,
                `Arrived: **${formatUtcShort(row.actualArrivalUtc ?? undefined)}**`,
                `Flight time: **${formatFlightTimeMinutes(row.flightTimeMinutes)}**`,
                `Crew: ${crew}`,
            ].join("\n"),
            inline: false,
        };
    });

    const baseEmbed: APIEmbed = {
        title: `History · ${registration}`,
        description: `Organisation: **${orgName}**`,
        color: 0x0099ff,
        fields,
        footer: { text: paginationFooter(result.page, result.totalPages, "history") },
        timestamp: new Date().toISOString(),
    };

    const linked = await withLiveModeLink(meta, [baseEmbed], liveModeRedirectPath("flights", organizationId));
    await editReply(linked);
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
            "history",
            schedulesPayload(registration, page),
            `Registration **${registration}** is linked to more than one org. Pick the correct one.`,
            async (orgId, orgName) => {
                await replyHistoryPage(chat.editReply.bind(chat), meta, orgId, orgName, registration, page);
            },
            chat.editReply.bind(chat),
        );
    } catch (err: unknown) {
        await handleLiveMemberError(chat.editReply.bind(chat), err, "Could not list flight history");
    }
}

export async function handleHistoryOrgSelect(interaction: DiscordInteraction, payload: string): Promise<void> {
    const select = interaction.getSelectMenuInteraction();
    if (!select) return;

    await select.deferReply();
    const { registration, page } = parseSchedulesPayload(payload);
    const orgId = select.values[0];
    const meta = interaction.getMetaInfo();

    try {
        const connections = await ApiService.listVALiveConnections(meta);
        const orgName = connections.find(c => c.id === orgId)?.name ?? "Organisation";
        await replyHistoryPage(select.editReply.bind(select), meta, orgId, orgName, registration, page);
    } catch (err: unknown) {
        await handleLiveMemberError(select.editReply.bind(select), err, "Could not list flight history");
    }
}
