import { SlashCommandBuilder, ActionRowBuilder, ButtonBuilder, ButtonStyle } from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { ApiService } from "../services/apiService";
import { UnauthorizedError } from "../helpers/UnauthorizedException";

export const data = new SlashCommandBuilder()
    .setName("tour")
    .setDescription("View active tour details");

export async function execute(interaction: DiscordInteraction) {
    const chat = interaction.getChatInputInteraction();
    if (!chat) return;

    try {
        // Defer the reply since we're making API calls
        await chat.deferReply();

        const metaInfo = interaction.getMetaInfo();

        try {
            // Fetch active events
            const eventsResponse = await ApiService.getActiveEvents(metaInfo);
            const events = eventsResponse.result || [];

            // Find active multi-leg event (tour)
            const tourEvent = events.find(event => (event.legs?.length || 0) > 1);

            if (!tourEvent) {
                await chat.editReply({
                    embeds: [{
                        title: "🗺️ No Active Tour",
                        description: "No active tour found. A tour is a multi-leg event with more than one leg.",
                        color: 0xff9900,
                        timestamp: new Date().toISOString(),
                        footer: {
                            text: "Use /events to view all active events"
                        }
                    }]
                });
                return;
            }

            // Build tour summary embed
            const fields: Array<{ name: string; value: string; inline?: boolean }> = [];

            // Add description if available
            if (tourEvent.description) {
                fields.push({
                    name: "📝 Description",
                    value: tourEvent.description,
                    inline: false
                });
            }

            const legCount = tourEvent.legs?.length || 0;

            // Add total legs and status
            fields.push({
                name: "🗺️ Total Legs",
                value: legCount.toString(),
                inline: true
            });

            // Add status with emoji
            const statusEmoji = tourEvent.status === "active" ? "🟢" : tourEvent.status === "completed" ? "✅" : tourEvent.status === "cancelled" ? "❌" : "⚪";
            fields.push({
                name: `${statusEmoji} Status`,
                value: tourEvent.status.charAt(0).toUpperCase() + tourEvent.status.slice(1),
                inline: true
            });

            // Add dates if available
            if (tourEvent.start_date || tourEvent.end_date) {
                const dateInfo: string[] = [];
                if (tourEvent.start_date) {
                    const startDate = new Date(tourEvent.start_date);
                    dateInfo.push(`📅 **Start:** ${startDate.toLocaleDateString("en-US", {
                        month: "short",
                        day: "numeric",
                        year: "numeric",
                        hour: "2-digit",
                        minute: "2-digit"
                    })}`);
                }
                if (tourEvent.end_date) {
                    const endDate = new Date(tourEvent.end_date);
                    dateInfo.push(`📅 **End:** ${endDate.toLocaleDateString("en-US", {
                        month: "short",
                        day: "numeric",
                        year: "numeric",
                        hour: "2-digit",
                        minute: "2-digit"
                    })}`);
                }
                fields.push({
                    name: "📅 Dates",
                    value: dateInfo.join("\n"),
                    inline: false
                });
            }

            // Add leg preview (first 5 legs)
            if (tourEvent.legs && tourEvent.legs.length > 0) {
                // Sort legs by leg_number to ensure proper ordering
                const sortedLegs = [...tourEvent.legs].sort((a, b) => a.leg_number - b.leg_number);
                const previewLegs = sortedLegs.slice(0, 5);
                
                // Use position-based numbering (1-indexed) instead of leg_number
                const legPreview = previewLegs.map((leg, index) => {
                    const position = index + 1;
                    return `**Leg #${position}:** ${leg.origin} → ${leg.destination}`;
                }).join("\n");

                const moreLegs = legCount > 5 ? `\n*...and ${legCount - 5} more leg${legCount - 5 !== 1 ? 's' : ''}*` : "";
                
                fields.push({
                    name: "📍 Leg Preview",
                    value: legPreview + moreLegs,
                    inline: false
                });
            }

            // Create "File PIREP" button
            const filePirepButton = new ButtonBuilder()
                .setCustomId("tour_file_pirep")
                .setLabel("File PIREP")
                .setStyle(ButtonStyle.Primary);

            const buttonRow = new ActionRowBuilder<ButtonBuilder>()
                .addComponents(filePirepButton);

            await chat.editReply({
                embeds: [{
                    title: `🗺️ ${tourEvent.name}`,
                    description: `Active tour with **${legCount}** leg${legCount !== 1 ? 's' : ''}`,
                    fields: fields,
                    color: 0x0099ff,
                    timestamp: new Date().toISOString(),
                    footer: {
                        text: "Use /event_leg <number> to view detailed leg information"
                    }
                }],
                components: [buttonRow]
            });
        } catch (tourErr: any) {
            if (tourErr instanceof UnauthorizedError) {
                await chat.editReply({
                    embeds: [{
                        title: "🔒 Not Authorized",
                        description: `❌ ${tourErr.message}`,
                        color: 0xff0000,
                        timestamp: new Date().toISOString()
                    }]
                });
                return;
            }

            console.error("[tour command] Error fetching tour:", tourErr);
            await chat.editReply({
                embeds: [{
                    title: "❌ Error",
                    description: `Unable to fetch tour: ${tourErr.message || 'Unknown error'}`,
                    color: 0xff0000,
                    timestamp: new Date().toISOString()
                }]
            });
        }
    } catch (err) {
        console.error("[tour command]", err);
        try {
            const chat = interaction.getChatInputInteraction();
            if (chat) {
                await chat.editReply({
                    embeds: [{
                        title: "Error",
                        description: `❌ An error occurred: ${String(err)}`,
                        color: 0xff0000
                    }]
                });
            }
        } catch (replyErr) {
            console.error("[tour command] Failed to send error message:", replyErr);
        }
    }
}
