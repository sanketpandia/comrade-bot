import { SlashCommandBuilder } from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { ApiService } from "../services/apiService";
import { UnauthorizedError } from "../helpers/UnauthorizedException";

export const data = new SlashCommandBuilder()
    .setName("events")
    .setDescription("List all active events");

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

            if (events.length === 0) {
                await chat.editReply({
                    embeds: [{
                        title: "✈️ Active Events",
                        description: "No active events found at this time.",
                        color: 0x0099ff,
                        timestamp: new Date().toISOString(),
                        footer: {
                            text: "Use /tour to view active tours"
                        }
                    }]
                });
                return;
            }

            // Build embed fields for each event
            const fields: Array<{ name: string; value: string; inline?: boolean }> = [];

            for (const event of events) {
                const eventInfo: string[] = [];

                // Add description if available (truncate to 200 chars)
                if (event.description) {
                    const desc = event.description.length > 200 
                        ? event.description.substring(0, 197) + "..." 
                        : event.description;
                    eventInfo.push(`📝 ${desc}`);
                }

                // Add leg count with emoji
                const legCount = event.legs?.length || 0;
                const legEmoji = legCount > 1 ? "🗺️" : "📍";
                eventInfo.push(`${legEmoji} **${legCount}** leg${legCount !== 1 ? 's' : ''}`);

                // Add status with emoji
                const statusEmoji = event.status === "active" ? "🟢" : event.status === "completed" ? "✅" : event.status === "cancelled" ? "❌" : "⚪";
                eventInfo.push(`${statusEmoji} **${event.status.charAt(0).toUpperCase() + event.status.slice(1)}**`);

                // Add dates if available
                if (event.start_date || event.end_date) {
                    const dateInfo: string[] = [];
                    if (event.start_date) {
                        const startDate = new Date(event.start_date);
                        dateInfo.push(`📅 **Start:** ${startDate.toLocaleDateString("en-US", {
                            month: "short",
                            day: "numeric",
                            year: "numeric",
                            hour: "2-digit",
                            minute: "2-digit"
                        })}`);
                    }
                    if (event.end_date) {
                        const endDate = new Date(event.end_date);
                        dateInfo.push(`📅 **End:** ${endDate.toLocaleDateString("en-US", {
                            month: "short",
                            day: "numeric",
                            year: "numeric",
                            hour: "2-digit",
                            minute: "2-digit"
                        })}`);
                    }
                    eventInfo.push(dateInfo.join("\n"));
                }

                // Determine if it's a tour (multi-leg event)
                const isTour = legCount > 1;
                const eventTitle = isTour ? `🗺️ ${event.name}` : `✈️ ${event.name}`;

                fields.push({
                    name: eventTitle,
                    value: eventInfo.join("\n") || "No details available",
                    inline: false
                });
            }

            await chat.editReply({
                embeds: [{
                    title: "✈️ Active Events",
                    description: `Found **${events.length}** active event${events.length !== 1 ? 's' : ''}`,
                    fields: fields,
                    color: 0x0099ff,
                    timestamp: new Date().toISOString(),
                    footer: {
                        text: `Use /tour for tour details • Use /event_leg <number> to view leg details • Response: ${eventsResponse.responseTimeMs || 0}ms`
                    }
                }]
            });
        } catch (eventsErr: any) {
            if (eventsErr instanceof UnauthorizedError) {
                await chat.editReply({
                    embeds: [{
                        title: "🔒 Not Authorized",
                        description: `❌ ${eventsErr.message}`,
                        color: 0xff0000,
                        timestamp: new Date().toISOString()
                    }]
                });
                return;
            }

            // If no events found or other errors
            if (eventsErr.message?.includes("404") || eventsErr.message?.includes("not found")) {
                await chat.editReply({
                    embeds: [{
                        title: "⚠️ No Events Found",
                        description: "No active events found at this time.",
                        color: 0xff9900,
                        timestamp: new Date().toISOString()
                    }]
                });
            } else {
                console.error("[events command] Error fetching events:", eventsErr);
                await chat.editReply({
                    embeds: [{
                        title: "❌ Error",
                        description: `Unable to fetch events: ${eventsErr.message || 'Unknown error'}`,
                        color: 0xff0000,
                        timestamp: new Date().toISOString()
                    }]
                });
            }
        }
    } catch (err) {
        console.error("[events command]", err);
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
            console.error("[events command] Failed to send error message:", replyErr);
        }
    }
}
