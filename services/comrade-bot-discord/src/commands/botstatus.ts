import { EmbedBuilder, SlashCommandBuilder } from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { ApiService } from "../services/apiService";
import { HealthApiResponse } from "../types/Responses";

export const data = new SlashCommandBuilder()
    .setName("botstatus")
    .setDescription("Check Comrade Bot and backend operational status");

function serviceLabel(value: string | undefined): string {
    const normalized = String(value ?? "").toLowerCase();
    if (normalized === "active" || normalized === "ok") {
        return "Active";
    }
    return "Down";
}

function formatHealthFields(health: HealthApiResponse | null): { name: string; value: string; inline: boolean }[] {
    const backendUp = health !== null && health.httpStatus === 200;
    const fields = [
        { name: "comrade-bot", value: "Up", inline: true },
        { name: "backend", value: backendUp ? "Up" : "Down", inline: true },
        { name: "database", value: serviceLabel(health?.services?.database), inline: true },
        { name: "cache", value: serviceLabel(health?.services?.cache), inline: true },
        { name: "sessions", value: serviceLabel(health?.services?.["infinite-flight"]), inline: true },
    ];
    return fields;
}

export async function execute(interaction: DiscordInteraction) {
    const chat = interaction.getChatInputInteraction();
    if (!chat) return;

    await chat.deferReply({ ephemeral: true });

    const checkedAt = new Date();
    let health: HealthApiResponse | null = null;
    let description = "Comrade Bot is online. The backend API could not be reached right now.";

    try {
        health = await ApiService.getHealth(interaction.getMetaInfo());
        if (health.httpStatus === 200) {
            description = "Comrade Bot is online and the backend API is responding.";
        } else {
            description = "Comrade Bot is online, but the backend API reports degraded status.";
        }
    } catch (_err) {
        health = null;
    }

    const embed = new EmbedBuilder()
        .setTitle("Comrade Bot operational status")
        .setDescription(description)
        .addFields(formatHealthFields(health))
        .addFields({ name: "Checked", value: checkedAt.toISOString(), inline: false })
        .setTimestamp(checkedAt);

    if (health?.uptime) {
        embed.setFooter({ text: `Backend uptime: ${health.uptime}` });
    }

    await chat.editReply({ embeds: [embed] });
}
