import { EmbedBuilder, SlashCommandBuilder } from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { ApiService } from "../services/apiService";
import { UserDetailsData } from "../types/Responses";

export const data = new SlashCommandBuilder()
    .setName("status")
    .setDescription("Check your account and current-server VA membership status");

export async function execute(interaction: DiscordInteraction) {
    const chat = interaction.getChatInputInteraction();
    if (!chat) return;

    await chat.deferReply({ ephemeral: true });

    try {
        const status = await ApiService.getUserDetails(interaction.getMetaInfo());
        await chat.editReply({ embeds: [buildStatusEmbed(status)] });
    } catch (_err) {
        await chat.editReply({
            embeds: [new EmbedBuilder()
                .setTitle("Unable to check status")
                .setDescription("I couldn't fetch your account status right now. Please try again later.")
                .addFields({ name: "Need to onboard?", value: "Run `/register` to create your account or link this VA." })],
        });
    }
}

function buildStatusEmbed(status: UserDetailsData): EmbedBuilder {
    const currentServer = status.current_server;
    const currentVA = status.current_va;
    const lines: string[] = [];

    if (!status.is_registered) {
        lines.push("❌ You do not have a Comrade Bot account yet.");
        lines.push("Run `/register` to create one. Registration collects only your IF Community username and last-flight route for account verification.");
    } else {
        lines.push(`✅ Registered${status.if_community_id ? ` as **${status.if_community_id}**` : ""}.`);
    }

    if (!currentServer.is_configured_va) {
        lines.push("ℹ️ This Discord server is not configured as a Virtual Airline, so no callsign is needed here.");
    } else if (currentVA?.is_member) {
        lines.push(`✅ Linked to **${currentVA.va_name ?? currentServer.va_name}**${currentVA.callsign ? ` with callsign **${currentVA.callsign}**` : ""}.`);
    } else {
        lines.push(`🔗 This server is configured as **${currentServer.va_name}**. Run \`/register\` to link your VA callsign.`);
    }

    const embed = new EmbedBuilder()
        .setTitle("Your Comrade Bot status")
        .setDescription(lines.join("\n\n"))
        .addFields(
            {
                name: "Current server",
                value: currentServer.is_configured_va
                    ? `${currentServer.va_name} (${currentServer.va_code})`
                    : "Not configured as a VA",
                inline: false,
            },
            {
                name: "VA memberships",
                value: `${status.memberships_summary?.active_count ?? 0} active / ${status.memberships_summary?.total_count ?? 0} total`,
                inline: true,
            },
        )
        .setTimestamp(new Date());

    if (currentVA?.is_member) {
        embed.addFields({
            name: "Current VA role",
            value: [currentVA.role ? formatRole(currentVA.role) : undefined, currentVA.is_active === false ? "Inactive" : "Active"].filter(Boolean).join(" • "),
            inline: true,
        });
    }

    if ((status.other_memberships_count ?? 0) > 0 && status.other_memberships?.length) {
        embed.addFields({
            name: "Other memberships",
            value: status.other_memberships.slice(0, 5).map(m => `${m.va_name} (${m.va_code}) — ${formatRole(m.role)}`).join("\n"),
            inline: false,
        });
    }

    return embed;
}

function formatRole(role: string): string {
    return role.charAt(0).toUpperCase() + role.slice(1);
}
