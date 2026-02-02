import {SlashCommandBuilder} from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { ApiService } from "../services/apiService";
import { UnauthorizedError } from "../helpers/UnauthorizedException";

// Helper: Convert seconds to human-readable time
function formatSeconds(seconds: number): string {
    if (seconds < 3600) {
        const mins = Math.floor(seconds / 60);
        return `${mins}m`;
    }
    const hours = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`;
}

// Helper: Format a value with proper unit
function formatValue(value: any, key?: string): string {
    // Check if this is a time field in seconds
    if (typeof value === 'number') {
        // Keys that likely contain time in seconds
        const timeFields = ['flight_time', 'total_hours', 'total_cm_hours', 'required_hours_to_next'];
        const isTimeField = key && timeFields.some(tf => key.toLowerCase().includes(tf));

        if (isTimeField && value > 3600) {
            return formatSeconds(value);
        }
        return value.toLocaleString();
    }
    return String(value);
}

export const data = new SlashCommandBuilder()
    .setName("stats")
    .setDescription("View your pilot statistics and activity")

export async function execute(interaction: DiscordInteraction) {
    const chat = interaction.getChatInputInteraction();
    if (!chat) return;

    try {
        // Defer the reply since we're making API calls
        await chat.deferReply();

        const metaInfo = interaction.getMetaInfo();

        try {
            // Try to get pilot stats
            const statsResponse = await ApiService.getPilotStats(metaInfo);
            const statsData = statsResponse.result;

            if (!statsData) {
                throw new Error("No stats data received");
            }

            // Build embed fields for all available sections
            const fields: Array<{name: string, value: string, inline?: boolean}> = [];

            // ===== GAME STATS SECTION =====
            if (statsData.game_stats) {
                const gameStatsFields: string[] = [];
                const gs = statsData.game_stats;

                if (gs.flight_time) gameStatsFields.push(`**Flight Time:** ${formatValue(gs.flight_time, 'flight_time')}`);
                if (gs.online_flights) gameStatsFields.push(`**Online Flights:** ${gs.online_flights.toLocaleString()}`);
                if (gs.landing_count) gameStatsFields.push(`**Landings:** ${gs.landing_count.toLocaleString()}`);
                if (gs.xp) gameStatsFields.push(`**XP:** ${gs.xp.toLocaleString()}`);
                if (gs.grade) gameStatsFields.push(`**Grade:** ${gs.grade}`);
                if (gs.violations) gameStatsFields.push(`**Violations:** ${gs.violations}`);

                if (gameStatsFields.length > 0) {
                    fields.push({
                        name: '🎮 Game Statistics',
                        value: gameStatsFields.join('\n'),
                        inline: false
                    });
                }
            }

            // ===== CAREER MODE SECTION =====
            if (statsData.career_mode_data) {
                const cmFields: string[] = [];
                const cm = statsData.career_mode_data;

                if (cm.airline && cm.aircraft) {
                    cmFields.push(`**Airline:** ${cm.airline} (${cm.aircraft})`);
                }
                if (cm.total_cm_hours) cmFields.push(`**Total Hours:** ${formatValue(cm.total_cm_hours, 'total_cm_hours')}`);
                if (cm.required_hours_to_next) cmFields.push(`**To Next Level:** ${formatValue(cm.required_hours_to_next, 'required_hours_to_next')}`);
                if (cm.last_career_mode_flight) cmFields.push(`**Last Flight:** ${cm.last_career_mode_flight}`);
                if (cm.assigned_routes && Array.isArray(cm.assigned_routes) && cm.assigned_routes.length > 0) {
                    cmFields.push(`**Routes Assigned:** ${cm.assigned_routes.length}`);
                }
                if (cm.last_activity_cm) {
                    const lastActivity = new Date(cm.last_activity_cm);
                    cmFields.push(`**Last Activity:** ${lastActivity.toLocaleString("en-US", {
                        month: "short",
                        day: "numeric",
                        hour: "2-digit",
                        minute: "2-digit",
                        timeZone: "UTC"
                    })} UTC`);
                }

                if (cmFields.length > 0) {
                    fields.push({
                        name: '✈️ Career Mode',
                        value: cmFields.join('\n'),
                        inline: false
                    });
                }
            }

            // ===== PROVIDER DATA SECTION =====
            if (statsData.provider_data) {
                const providerFields: string[] = [];
                const pd = statsData.provider_data;

                if (pd.join_date) providerFields.push(`**Join Date:** ${pd.join_date}`);
                if (pd.last_activity) providerFields.push(`**Last Activity:** ${pd.last_activity}`);
                if (pd.region) providerFields.push(`**Region:** ${pd.region}`);

                // Handle additional fields from provider_data
                if (pd.additional_fields && typeof pd.additional_fields === 'object') {
                    const af = pd.additional_fields;
                    if (af.callsign) providerFields.push(`**Callsign:** ${af.callsign}`);
                    if (af.category) providerFields.push(`**Category:** ${af.category}`);
                    if (af.cm_status) providerFields.push(`**CM Status:** ${af.cm_status}`);
                }

                if (providerFields.length > 0) {
                    fields.push({
                        name: '📋 Provider Information',
                        value: providerFields.join('\n'),
                        inline: false
                    });
                }
            }

            // ===== DATA INFO SECTION =====
            const metadataValue = [
                `**Cached:** ${statsData.metadata.cached ? 'Yes' : 'No'}`,
                statsData.metadata.last_fetched
                    ? `**Last Updated:** ${new Date(statsData.metadata.last_fetched).toLocaleString("en-US", {
                        month: "short",
                        day: "numeric",
                        hour: "2-digit",
                        minute: "2-digit",
                        timeZone: "UTC"
                    })} UTC`
                    : null
            ].filter(Boolean).join('\n');

            fields.push({
                name: 'ℹ️ Data Info',
                value: metadataValue,
                inline: false
            });

            await chat.editReply({
                embeds: [{
                    title: "Your Pilot Statistics",
                    fields: fields,
                    color: 0x0099ff,
                    timestamp: new Date().toISOString(),
                    footer: {
                        text: `${statsData.metadata.va_name || "Virtual Airline"} • Response time: ${statsResponse.responseTimeMs}ms`
                    }
                }]
            });
        } catch (statsErr: any) {
            if (statsErr instanceof UnauthorizedError) {
                await chat.editReply({
                    embeds: [{
                        title: "Not Authorized",
                        description: `❌ ${statsErr.message}`,
                        color: 0xff0000,
                        timestamp: new Date().toISOString()
                    }]
                });
                return;
            }

            // If user is not registered or stats not found
            if (statsErr.message?.includes("404") || statsErr.message?.includes("not found")) {
                await chat.editReply({
                    embeds: [{
                        title: "Stats Not Available",
                        description: "❌ No pilot statistics found.\n\nMake sure you're registered with `/register` and have connected your VA account!",
                        color: 0xff0000,
                        timestamp: new Date().toISOString()
                    }]
                });
            } else {
                // For other errors, show a generic error message
                console.error("[stats command] Error fetching pilot stats:", statsErr);
                await chat.editReply({
                    embeds: [{
                        title: "Error",
                        description: `❌ Unable to fetch pilot statistics: ${statsErr.message || 'Unknown error'}`,
                        color: 0xff0000,
                        timestamp: new Date().toISOString()
                    }]
                });
            }
        }
    } catch(err) {
        console.error("[stats command]", err);
        // Try to respond with an error message
        try {
            await chat.editReply({
                embeds: [{
                    title: "Error",
                    description: `❌ An error occurred: ${String(err)}`,
                    color: 0xff0000
                }]
            });
        } catch (replyErr) {
            console.error("[stats command] Failed to send error message:", replyErr);
        }
    }
}
