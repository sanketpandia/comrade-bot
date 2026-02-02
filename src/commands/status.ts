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

    const metaInfo = interaction.getMetaInfo();
    let userData: any = null;
    let healthData: any = null;
    let userError: string | null = null;
    let healthError: string | null = null;

    // Try to get user details
    try {
        userData = await ApiService.getUserDetails(metaInfo);
    } catch (err: any) {
        console.error("[status command] getUserDetails error:", err);
        if (err instanceof NotFoundError) {
            userError = `❌ You are not registered to the bot: \n${String(err)}`;
        } else {
            userError = `❌ Error fetching user details: \n${String(err)}`;
        }
    }

    // Try to get health data
    try {
        healthData = await ApiService.getHealth(metaInfo);
    } catch (err: any) {
        console.error("[status command] getHealth error:", err);
        healthError = `❌ Error fetching health data: \n${String(err)}`;
    }

    // Build response based on what data we have
    if (!userData && !healthData) {
        // Both failed
        title = "Error";
        description = userError || healthError || "❌ An error occurred";
        color = 0xff0000;
    } else if (!userData) {
        // Only user data failed
        title = "Partial Status Report";
        description = (userError || "") + "\n\n" + MessageFormatters.generateHealthString(healthData);
        color = 0xff9900;
    } else if (!healthData) {
        // Only health data failed
        title = "Partial Status Report";
        const formattedMessage = MessageFormatters.generateUserDetailsString(
            userData,
            metaInfo.discordId
        );
        description = (healthError || "") + "\n\n" + formattedMessage;
        color = userData.current_va.is_member ? 0x00ff00 : 0xff9900;
    } else {
        // Both succeeded
        const healthMessage = MessageFormatters.generateHealthString(healthData);
        const formattedMessage = MessageFormatters.generateUserDetailsString(
            userData,
            metaInfo.discordId
        );
        title = "Status Report";
        description = healthMessage + "\n" + formattedMessage;
        color = userData.current_va.is_member ? 0x00ff00 : 0xff9900;
    }

    timestamp = new Date().toISOString();

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