import {SlashCommandBuilder} from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { ApiService } from "../services/apiService";
import { UnauthorizedError } from "../helpers/UnauthorizedException";

// Helper: Format HH:MM time string (already formatted from backend)
function formatTimeString(timeValue: any): string {
    if (typeof timeValue === 'string') {
        // Already formatted as HH:MM from backend
        return timeValue;
    }
    if (typeof timeValue === 'number') {
        // Fallback: convert seconds to HH:MM
        const hours = Math.floor(timeValue / 3600);
        const mins = Math.floor((timeValue % 3600) / 60);
        return `${hours.toString().padStart(2, '0')}:${mins.toString().padStart(2, '0')}`;
    }
    return String(timeValue);
}

// Helper: Format array to show top 6 items
function formatArray(arr: any[]): string {
    if (!Array.isArray(arr) || arr.length === 0) {
        return '';
    }
    const top6 = arr.slice(0, 6);
    return top6.join(', ');
}

// Helper: Format a value with proper unit
function formatValue(value: any, key?: string): string {
    // Handle arrays - show top 6
    if (Array.isArray(value)) {
        return formatArray(value);
    }
    
    // Handle time fields - check if it's a number (seconds) or already formatted string
    if (key && (key.includes('hours') || key.includes('time') || key.includes('duration'))) {
        // If it's a number, assume it's seconds and format it
        if (typeof value === 'number') {
            return formatTimeString(value);
        }
        // If it's already a string (HH:MM format), return as is
        return String(value);
    }
    
    // Handle numbers
    if (typeof value === 'number') {
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

                // Flight time is in seconds, convert to HH:MM format
                if (gs.flight_time) {
                    const hours = Math.floor(gs.flight_time / 3600);
                    const mins = Math.floor((gs.flight_time % 3600) / 60);
                    gameStatsFields.push(`**Flight Time:** ${hours}h ${mins}m`);
                }
                if (gs.online_flights) gameStatsFields.push(`**Online Flights:** ${gs.online_flights.toLocaleString()}`);
                if (gs.landing_count) gameStatsFields.push(`**Landings:** ${gs.landing_count.toLocaleString()}`);
                if (gs.xp) gameStatsFields.push(`**XP:** ${gs.xp.toLocaleString()}`);
                if (gs.grade) gameStatsFields.push(`**Grade:** ${gs.grade}`);
                if (gs.violations) gameStatsFields.push(`**Violations:** ${gs.violations}`);

                if (gameStatsFields.length > 0) {
                    fields.push({
                        name: '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━',
                        value: '**🎮 GAME STATISTICS**\n' + gameStatsFields.join('\n'),
                        inline: false
                    });
                }
            }

            // ===== CAREER MODE SECTION =====
            if (statsData.career_mode_data) {
                const cmFields: string[] = [];
                const cm = statsData.career_mode_data;

                // Show aircraft and airline separately if available
                if (cm.aircraft) cmFields.push(`**Aircraft:** ${cm.aircraft}`);
                if (cm.airline) cmFields.push(`**Airline:** ${cm.airline}`);
                
                // Time fields (already formatted as HH:MM from backend)
                if (cm.total_cm_hours) {
                    cmFields.push(`**Total Hours:** ${formatValue(cm.total_cm_hours, 'total_cm_hours')}`);
                }
                if (cm.required_hours_to_next) {
                    cmFields.push(`**To Next Level:** ${formatValue(cm.required_hours_to_next, 'required_hours_to_next')}`);
                }
                
                // Last flown route - show prominently
                if (cm.last_career_mode_flight) {
                    cmFields.push(`**Last Flown Route:** ${cm.last_career_mode_flight}`);
                }
                
                // Assigned routes - show top 6
                if (cm.assigned_routes) {
                    if (Array.isArray(cm.assigned_routes) && cm.assigned_routes.length > 0) {
                        const routesDisplay = formatArray(cm.assigned_routes);
                        cmFields.push(`**Routes Assigned:** ${routesDisplay}${cm.assigned_routes.length > 6 ? ` (${cm.assigned_routes.length} total)` : ''}`);
                    }
                }
                
                // Last activity
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
                
                // Additional fields from career mode
                if (cm.additional_fields && typeof cm.additional_fields === 'object') {
                    const af = cm.additional_fields;
                    // Show any other relevant fields
                    Object.keys(af).forEach(key => {
                        if (key !== 'callsign' && af[key] !== null && af[key] !== undefined) {
                            const val = Array.isArray(af[key]) ? formatArray(af[key]) : af[key];
                            cmFields.push(`**${key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}:** ${val}`);
                        }
                    });
                }

                if (cmFields.length > 0) {
                    fields.push({
                        name: '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━',
                        value: '**✈️ CAREER MODE STATISTICS**\n' + cmFields.join('\n'),
                        inline: false
                    });
                }
            }

            // ===== PROVIDER DATA SECTION =====
            if (statsData.provider_data) {
                const providerFields: string[] = [];
                const pd = statsData.provider_data;

                // Standard fields - check for existence explicitly
                if (pd.join_date !== undefined && pd.join_date !== null) {
                    providerFields.push(`**Join Date:** ${pd.join_date}`);
                }
                if (pd.last_activity !== undefined && pd.last_activity !== null) {
                    providerFields.push(`**Last Activity:** ${pd.last_activity}`);
                }
                if (pd.region !== undefined && pd.region !== null) {
                    providerFields.push(`**Region:** ${pd.region}`);
                }
                if (pd.rank !== undefined && pd.rank !== null) {
                    providerFields.push(`**Rank:** ${pd.rank}`);
                }
                if (pd.status !== undefined && pd.status !== null) {
                    providerFields.push(`**Status:** ${pd.status}`);
                }
                
                // Flight hours - handle both formatted strings and raw numbers (seconds)
                if (pd.flight_hours !== undefined && pd.flight_hours !== null) {
                    providerFields.push(`**Flight Hours:** ${formatValue(pd.flight_hours, 'flight_hours')}`);
                }
                
                // Total flights
                if (pd.total_flights !== undefined && pd.total_flights !== null) {
                    providerFields.push(`**Total Flights:** ${pd.total_flights.toLocaleString()}`);
                }

                // Handle additional fields from provider_data
                if (pd.additional_fields && typeof pd.additional_fields === 'object') {
                    const af = pd.additional_fields;
                    // Show all additional fields (excluding ones already shown)
                    // Use case-insensitive comparison for shownKeys
                    const shownKeysLower = ['callsign', 'category', 'cm_status', 'rank', 'status', 'region', 'join_date', 'last_activity', 'flight_hours', 'total_flights', 'if community id', 'ifc name'];
                    Object.keys(af).forEach(key => {
                        const keyLower = key.toLowerCase();
                        if (!shownKeysLower.includes(keyLower) && af[key] !== null && af[key] !== undefined && af[key] !== '') {
                            const val = Array.isArray(af[key]) ? formatArray(af[key]) : formatValue(af[key], key);
                            // Format the key nicely for display
                            const displayKey = key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
                            providerFields.push(`**${displayKey}:** ${val}`);
                        }
                    });
                }

                if (providerFields.length > 0) {
                    fields.push({
                        name: '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━',
                        value: '**📋 VA STATISTICS**\n' + providerFields.join('\n'),
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
