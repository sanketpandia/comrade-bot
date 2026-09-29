import { SlashCommandBuilder } from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { handleLiveFlights } from "./liveHandler";

export const data = new SlashCommandBuilder()
    .setName("live")
    .setDescription("Fetch VA Live flights");

export async function execute(interaction: DiscordInteraction) {
    const page = 1;
    await handleLiveFlights(interaction, page);
}
