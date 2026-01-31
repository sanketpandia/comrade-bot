import { SlashCommandBuilder, ChatInputCommandInteraction } from "discord.js";
import fetch from "node-fetch";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { ApiService } from "../services/apiService";
import { MessageFormatters } from "../helpers/messageFormatter";
import { DiscordResponses } from "../helpers/discordResponses";
import { UnauthorizedError } from "../helpers/UnauthorizedException";
import { NotFoundError } from "../helpers/NotFoundException";

export const data = new SlashCommandBuilder()
    .setName("status")
    .setDescription("Check your registration and VA membership status")

export async function execute(interaction: DiscordInteraction) {
    const chat = interaction.getChatInputInteraction();
    if (!chat) return;

    // Defer the reply since we're making API calls
    await chat.deferReply();

    let title: string;
    let description: string;
    let color: number;
    let timestamp: string;

    try {
        const metaInfo = interaction.getMetaInfo();

        // Get user details and health data
        const userData = await ApiService.getUserDetails(metaInfo);
        const healthData = await ApiService.getHealth(metaInfo);

        // Format messages
        const healthMessage = MessageFormatters.generateHealthString(healthData);
        const formattedMessage = MessageFormatters.generateUserDetailsString(
            userData,
            metaInfo.discordId
        );

        // Set success response
        title = "Status Report";
        description = healthMessage + "\n" + formattedMessage;
        color = userData.current_va.is_member ? 0x00ff00 : 0xff9900;
        timestamp = new Date().toISOString();

    } catch (err: any) {
        console.error("[status command]", err);

        // Set error response based on error type
        if (err instanceof NotFoundError) {
            title = "Not found";
            description = `❌ You are not registered to the bot: \n${String(err)}`;
        } else {
            title = "Error";
            description = `❌ An error occurred: \n${String(err)}`;
        }
        color = 0xff0000;
        timestamp = new Date().toISOString();
    }

    // Single editReply with computed values
    try {
        await chat.editReply({
            embeds: [{
                title,
                description,
                color,
                timestamp
            }]
        });
    } catch (replyErr) {
        console.error("[status command] Failed to send message:", replyErr);
    }
}