import { APIEmbed, InteractionEditReplyOptions, SlashCommandBuilder } from "discord.js";
import { ApiService } from "../services/apiService";
import { DiscordInteraction } from "../types/DiscordInteraction";
import {
    handleLiveMemberError,
    LIVE_PAGE_SIZE,
    paginationFooter,
    withVaLiveOrg,
} from "../helpers/liveCommands";

export const data = new SlashCommandBuilder()
    .setName("aircrafts")
    .setDescription("List aircraft in the VA Live organisation fleet")
    .addIntegerOption(option =>
        option.setName("page").setDescription("Page number").setMinValue(1).setRequired(false),
    );

function parseAircraftsPayload(payload: string): number {
    const page = Number.parseInt(payload, 10);
    return Number.isFinite(page) && page > 0 ? page : 1;
}

export async function replyAircraftsPage(
    editReply: (content: InteractionEditReplyOptions) => Promise<unknown>,
    meta: ReturnType<DiscordInteraction["getMetaInfo"]>,
    organizationId: string,
    orgName: string,
    page: number,
): Promise<void> {
    const fleet = await ApiService.listVALiveOrganizationAircraft(meta, organizationId);
    const totalPages = Math.max(1, Math.ceil(fleet.length / LIVE_PAGE_SIZE));
    const safePage = Math.min(Math.max(page, 1), totalPages);
    const slice = fleet.slice((safePage - 1) * LIVE_PAGE_SIZE, safePage * LIVE_PAGE_SIZE);

    if (!slice.length) {
        await editReply({
            embeds: [{
                title: `${orgName} fleet`,
                description: "No aircraft in this organisation.",
                color: 0x0099ff,
                timestamp: new Date().toISOString(),
            }],
        });
        return;
    }

    const fields = slice.map(row => ({
        name: row.registration,
        value: [
            row.aircraftName ? `Type: **${row.aircraftName}**` : undefined,
            `Fleet rank: **${row.fleetRank}**`,
            row.isFleetActiveSlot ? "Active fleet slot" : "Reserve slot",
        ].filter(Boolean).join("\n"),
        inline: false,
    }));

    const baseEmbed: APIEmbed = {
        title: `${orgName} fleet`,
        color: 0x0099ff,
        fields,
        footer: { text: paginationFooter(safePage, totalPages, "aircrafts") },
        timestamp: new Date().toISOString(),
    };

    await editReply({ embeds: [baseEmbed] });
}

export async function execute(interaction: DiscordInteraction): Promise<void> {
    const chat = interaction.getChatInputInteraction();
    if (!chat) return;

    await chat.deferReply();
    const page = chat.options.getInteger("page") ?? 1;
    const meta = interaction.getMetaInfo();
    const payload = String(page);

    try {
        await withVaLiveOrg(
            meta,
            "aircrafts",
            payload,
            "This VA is linked to more than one Live organisation. Pick one to list aircraft.",
            async (orgId, orgName) => {
                await replyAircraftsPage(chat.editReply.bind(chat), meta, orgId, orgName, page);
            },
            chat.editReply.bind(chat),
        );
    } catch (err: unknown) {
        await handleLiveMemberError(chat.editReply.bind(chat), err, "Could not list aircraft");
    }
}

export async function handleAircraftsOrgSelect(interaction: DiscordInteraction, payload: string): Promise<void> {
    const select = interaction.getSelectMenuInteraction();
    if (!select) return;

    await select.deferReply();
    const orgId = select.values[0];
    const meta = interaction.getMetaInfo();
    const page = parseAircraftsPayload(payload);

    try {
        const connections = await ApiService.listVALiveConnections(meta);
        const orgName = connections.find(c => c.id === orgId)?.name ?? "Organisation";
        await replyAircraftsPage(select.editReply.bind(select), meta, orgId, orgName, page);
    } catch (err: unknown) {
        await handleLiveMemberError(select.editReply.bind(select), err, "Could not list aircraft");
    }
}
