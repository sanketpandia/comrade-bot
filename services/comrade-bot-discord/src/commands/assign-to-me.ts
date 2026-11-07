import {
    ActionRowBuilder,
    APIEmbed,
    InteractionEditReplyOptions,
    SlashCommandBuilder,
    StringSelectMenuBuilder,
    StringSelectMenuOptionBuilder,
} from "discord.js";
import { ApiService } from "../services/apiService";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { CUSTOM_IDS } from "../configs/constants";
import type { LiveScheduleAssignMeResponse } from "../generated/politburoClient";
import {
    handleLiveMemberError,
    liveModeRedirectPath,
    withLiveModeLink,
} from "../helpers/liveCommands";

export const data = new SlashCommandBuilder()
    .setName("assign-to-me")
    .setDescription("Claim a VA live schedule as captain using registration and route")
    .addStringOption(option =>
        option
            .setName("registration")
            .setDescription("Aircraft registration (e.g. PH-ZNB)")
            .setRequired(true),
    )
    .addStringOption(option =>
        option
            .setName("route")
            .setDescription("Route as ORIG-DEST ICAO pair (e.g. EHAM-CYYZ)")
            .setRequired(true),
    );

export function assignToMeSelectCustomId(registration: string, route: string): string {
    return `${CUSTOM_IDS.ASSIGN_TO_ME_SELECT_PREFIX}${encodeURIComponent(registration)}|${encodeURIComponent(route)}`;
}

export function parseAssignToMeSelectCustomId(customId: string): { registration: string; route: string } | null {
    if (!customId.startsWith(CUSTOM_IDS.ASSIGN_TO_ME_SELECT_PREFIX)) {
        return null;
    }
    const rest = customId.slice(CUSTOM_IDS.ASSIGN_TO_ME_SELECT_PREFIX.length);
    const pipe = rest.indexOf("|");
    if (pipe < 0) {
        return null;
    }
    return {
        registration: decodeURIComponent(rest.slice(0, pipe)),
        route: decodeURIComponent(rest.slice(pipe + 1)),
    };
}

async function replyAssignSuccess(
    editReply: (content: InteractionEditReplyOptions) => Promise<unknown>,
    meta: ReturnType<DiscordInteraction["getMetaInfo"]>,
    organizationId: string,
    result: LiveScheduleAssignMeResponse,
): Promise<void> {
    const linked = await withLiveModeLink(
        meta,
        [successEmbed(result)],
        liveModeRedirectPath("schedules", organizationId),
    );
    await editReply(linked);
}

export async function execute(interaction: DiscordInteraction): Promise<void> {
    const chat = interaction.getChatInputInteraction();
    if (!chat) return;

    await chat.deferReply();
    const registration = chat.options.getString("registration", true).trim();
    const route = chat.options.getString("route", true).trim();
    const meta = interaction.getMetaInfo();

    try {
        const connections = await ApiService.listVALiveConnections(meta);
        if (!connections.length) {
            await chat.editReply({
                embeds: [{
                    title: "No Live org linked",
                    description: "This VA has no Infinite Flight Live organisation connected yet.",
                    color: 0xff9900,
                    timestamp: new Date().toISOString(),
                }],
            });
            return;
        }
        if (connections.length === 1) {
            const result = await ApiService.assignLiveScheduleToMe(meta, connections[0].id, {
                registration,
                route,
            });
            await replyAssignSuccess(chat.editReply.bind(chat), meta, connections[0].id, result);
            return;
        }

        const options = connections.slice(0, 25).map(conn =>
            new StringSelectMenuOptionBuilder()
                .setLabel(conn.name)
                .setValue(conn.id),
        );
        const menu = new StringSelectMenuBuilder()
            .setCustomId(assignToMeSelectCustomId(registration, route))
            .setPlaceholder("Select which VA organisation owns this flight")
            .addOptions(options);
        const row = new ActionRowBuilder<StringSelectMenuBuilder>().addComponents(menu);

        await chat.editReply({
            embeds: [{
                title: "Choose organisation",
                description: `Registration **${registration}** on **${route}** is linked to more than one org. Pick the correct one.`,
                color: 0x0099ff,
                timestamp: new Date().toISOString(),
            }],
            components: [row],
        });
    } catch (err: unknown) {
        await handleLiveMemberError(chat.editReply.bind(chat), err, "Could not assign schedule");
    }
}

export async function handleAssignToMeOrgSelect(interaction: DiscordInteraction): Promise<void> {
    const select = interaction.getSelectMenuInteraction();
    if (!select) return;

    const parsed = parseAssignToMeSelectCustomId(select.customId);
    if (!parsed) {
        await select.reply({ content: "Invalid selection context.", ephemeral: true });
        return;
    }

    await select.deferReply();
    const orgId = select.values[0];
    const meta = interaction.getMetaInfo();

    try {
        const result = await ApiService.assignLiveScheduleToMe(meta, orgId, {
            registration: parsed.registration,
            route: parsed.route,
        });
        await replyAssignSuccess(select.editReply.bind(select), meta, orgId, result);
    } catch (err: unknown) {
        await handleLiveMemberError(select.editReply.bind(select), err, "Could not assign schedule");
    }
}

function successEmbed(result: LiveScheduleAssignMeResponse): APIEmbed {
    const route = `${result.originIcao}-${result.destinationIcao}`;
    const callsign = result.callsign ? `\nCallsign: **${result.callsign}**` : "";
    return {
        title: "Schedule claimed",
        description:
            `You are assigned as **Captain** on **${result.registration}** | **${route}**.${callsign}`,
        color: 0x57f287,
        timestamp: new Date().toISOString(),
    };
}
