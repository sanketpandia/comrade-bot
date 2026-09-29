import { EmbedBuilder, SlashCommandBuilder } from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { ApiService } from "../services/apiService";

export const data = new SlashCommandBuilder()
    .setName("botstatus")
    .setDescription("Check sanitized Comrade Bot operational status");

export async function execute(interaction: DiscordInteraction) {
    const chat = interaction.getChatInputInteraction();
    if (!chat) return;

    await chat.deferReply({ ephemeral: true });

    const checkedAt = new Date();
    let backendState = "Unavailable";
    let description = "Comrade Bot is online. The backend API could not be reached right now.";

    try {
        const health = await ApiService.getHealth(interaction.getMetaInfo());
        const normalized = String(health.status || "").toLowerCase();
        if (normalized === "ok" || normalized === "healthy") {
            backendState = "Operational";
            description = "Comrade Bot is online and the backend API is responding.";
        } else {
            backendState = "Degraded";
            description = "Comrade Bot is online, but the backend API reports degraded status.";
        }
    } catch (_err) {
        backendState = "Unavailable";
    }

    await chat.editReply({
        embeds: [new EmbedBuilder()
            .setTitle("Comrade Bot operational status")
            .setDescription(description)
            .addFields(
                { name: "Bot", value: "Online", inline: true },
                { name: "Backend API", value: backendState, inline: true },
                { name: "Checked", value: checkedAt.toISOString(), inline: false },
            )
            .setTimestamp(checkedAt)],
    });
}
