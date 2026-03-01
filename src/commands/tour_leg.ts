import { SlashCommandBuilder, ActionRowBuilder, ButtonBuilder, ButtonStyle } from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { ApiService } from "../services/apiService";
import { UnauthorizedError } from "../helpers/UnauthorizedException";
import { PermissionDeniedError } from "../helpers/PermissionDeniedException";

export const data = new SlashCommandBuilder()
    .setName("tour_leg")
    .setDescription("View details of a specific tour leg")
    .addIntegerOption(option =>
        option
            .setName("number")
            .setDescription("The leg number to view")
            .setRequired(true)
            .setMinValue(1)
    );

export async function execute(interaction: DiscordInteraction) {
    const chat = interaction.getChatInputInteraction();
    if (!chat) return;

    try {
        // Defer the reply since we're making API calls
        await chat.deferReply();

        const metaInfo = interaction.getMetaInfo();
        const legNumber = chat.options.getInteger("number", true);

        try {
            // Fetch active events to find the tour
            const eventsResponse = await ApiService.getActiveEvents(metaInfo);
            const events = eventsResponse.result || [];

            // Find active multi-leg event (tour)
            const tourEvent = events.find((event: any) => (event.legs?.length || 0) > 1);

            if (!tourEvent) {
                await chat.editReply({
                    embeds: [{
                        title: "🗺️ No Active Tour",
                        description: "No active tour found. A tour is a multi-leg event with more than one leg.",
                        color: 0xff9900,
                        timestamp: new Date().toISOString()
                    }]
                });
                return;
            }

            // Ensure we have legs
            if (!tourEvent.legs || tourEvent.legs.length === 0) {
                await chat.editReply({
                    embeds: [{
                        title: "⚠️ No Legs Found",
                        description: `The tour "${tourEvent.name}" has no legs configured.`,
                        color: 0xff9900,
                        timestamp: new Date().toISOString()
                    }]
                });
                return;
            }

            // Sort legs by leg_number to ensure proper ordering
            const sortedLegs = [...tourEvent.legs].sort((a, b) => a.leg_number - b.leg_number);

            // Find the leg by position (1-indexed) instead of leg_number
            const leg = sortedLegs[legNumber - 1];

            if (!leg) {
                const availableLegs = sortedLegs.length;
                await chat.editReply({
                    embeds: [{
                        title: "⚠️ Leg Not Found",
                        description: `Leg #${legNumber} not found in the active tour "${tourEvent.name}".\n\nAvailable legs: 1-${availableLegs}`,
                        color: 0xff9900,
                        timestamp: new Date().toISOString()
                    }]
                });
                return;
            }

            // The position is the same as the requested legNumber (1-indexed)
            const legPosition = legNumber;

            // Build embed fields
            const fields: Array<{ name: string; value: string; inline?: boolean }> = [];

            // Basic Details Section
            fields.push({
                name: "📍 Route",
                value: `${leg.origin} → ${leg.destination}`,
                inline: false
            });
            
            fields.push({
                name: "🔢 Leg Number",
                value: legPosition.toString(),
                inline: true
            });

            if (leg.route_at_id) {
                fields.push({
                    name: "🛣️ Route ID",
                    value: leg.route_at_id,
                    inline: true
                });
            }

            // Description if available
            if (leg.description) {
                fields.push({
                    name: "📝 Description",
                    value: leg.description,
                    inline: false
                });
            }

            // Add event dates from parent event
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
                    name: "📅 Event Dates",
                    value: dateInfo.join("\n"),
                    inline: false
                });
            }

            // Additional Data Section
            // Check for Flight Plan to send separately after embed
            let flightPlanText: string | null = null;
            let flightPlanKey: string | null = null;
            
            if (leg.additional_data && Object.keys(leg.additional_data).length > 0) {
                // Check for Flight Plan or Plan in additional_data (case-insensitive, with variations)
                const flightPlanKeyPatterns = ["flight plan", "plan", "flightplan", "flight_plan"];

                for (const [key, value] of Object.entries(leg.additional_data)) {
                    const keyLower = key.toLowerCase().trim();
                    // Check if key matches any flight plan pattern
                    const matchesPattern = flightPlanKeyPatterns.some(pattern => 
                        keyLower === pattern || keyLower.includes(pattern)
                    );
                    
                    if (matchesPattern && value != null && value !== "") {
                        // Accept any truthy value (string, array, etc.)
                        const planText = Array.isArray(value) 
                            ? value.join("\n") 
                            : String(value).trim();
                        
                        if (planText) {
                            flightPlanText = planText;
                            flightPlanKey = key;
                            break;
                        }
                    }
                }

                // Filter out Airtable record IDs (but keep other useful IDs)
                const filteredAdditionalData = Object.entries(leg.additional_data).filter(([key]) => {
                    const keyLower = key.toLowerCase();
                    // Filter out Airtable-specific ID patterns
                    return !keyLower.includes('airtable') && 
                           !(keyLower === 'record_id') && 
                           !(keyLower === 'at_id') &&
                           !(keyLower === 'airtable_id') &&
                           !(keyLower === 'airtable_record_id');
                });

                if (filteredAdditionalData.length > 0) {
                    // Add a separator field
                    fields.push({
                        name: "━━━━━━━━━━━━━━━━━━━━",
                        value: "**Additional Information**",
                        inline: false
                    });

                    // Format additional_data fields
                    for (const [key, value] of filteredAdditionalData) {
                    let displayValue: string;
                    
                    if (value === null || value === undefined) {
                        displayValue = "N/A";
                    } else if (typeof value === "object") {
                        displayValue = JSON.stringify(value, null, 2);
                    } else {
                        displayValue = String(value);
                    }

                    // Truncate long values
                    if (displayValue.length > 1024) {
                        displayValue = displayValue.substring(0, 1021) + "...";
                    }

                    // Format key name (convert snake_case to Title Case)
                    const formattedKey = key
                        .split("_")
                        .map(word => word.charAt(0).toUpperCase() + word.slice(1))
                        .join(" ");

                        fields.push({
                            name: formattedKey,
                            value: displayValue,
                            inline: key.length < 20 // Use inline for shorter keys
                        });
                    }
                } else {
                    // All additional data was filtered out (only IDs)
                    fields.push({
                        name: "ℹ️ Additional Information",
                        value: "No additional data available for this leg.",
                        inline: false
                    });
                }
            } else {
                fields.push({
                    name: "ℹ️ Additional Information",
                    value: "No additional data available for this leg.",
                    inline: false
                });
            }

            // Build embed
            const embed: any = {
                title: `Leg #${legPosition}: ${leg.origin} → ${leg.destination}`,
                fields: fields,
                color: 0x0099ff,
                timestamp: new Date().toISOString(),
                footer: {
                    text: `Tour: ${tourEvent.name}`
                }
            };

            // Add thumbnail if available
            if (leg.thumbnail_url) {
                embed.thumbnail = {
                    url: leg.thumbnail_url
                };
            }

            // Create "File PIREP" button
            const filePirepButton = new ButtonBuilder()
                .setCustomId("tour_leg_file_pirep")
                .setLabel("File PIREP")
                .setStyle(ButtonStyle.Primary);

            const buttonRow = new ActionRowBuilder<ButtonBuilder>()
                .addComponents(filePirepButton);

            await chat.editReply({
                embeds: [embed],
                components: [buttonRow]
            });

            // Send flight plan after embed (try DM first, then channel message)
            if (flightPlanText && flightPlanKey) {
                // Try to DM first
                try {
                    const user = chat.user;
                    await user.send(flightPlanText);
                } catch (dmErr) {
                    console.error("[tour_leg command] Failed to send DM:", dmErr);
                    // If DM failed, send as follow-up message in channel
                    try {
                        await chat.followUp({
                            content: flightPlanText,
                            ephemeral: false
                        });
                    } catch (followUpErr) {
                        console.error("[tour_leg command] Failed to send follow-up message:", followUpErr);
                    }
                }
            }
        } catch (legErr: any) {
            if (legErr instanceof UnauthorizedError) {
                await chat.editReply({
                    embeds: [{
                        title: "🔒 Not Authorized",
                        description: `❌ ${legErr.message}`,
                        color: 0xff0000,
                        timestamp: new Date().toISOString()
                    }]
                });
                return;
            }

            if (legErr instanceof PermissionDeniedError) {
                await chat.editReply({
                    embeds: [{
                        title: "⚠️ Registration Required",
                        description: `❌ ${legErr.message}\n\nPlease register your account using the \`/register\` command before accessing tour leg information.`,
                        color: 0xff9900,
                        timestamp: new Date().toISOString()
                    }]
                });
                return;
            }

            console.error("[tour_leg command] Error fetching leg:", legErr);
            await chat.editReply({
                embeds: [{
                    title: "❌ Error",
                    description: `Unable to fetch leg details: ${legErr.message || 'Unknown error'}`,
                    color: 0xff0000,
                    timestamp: new Date().toISOString()
                }]
            });
        }
    } catch (err) {
        console.error("[tour_leg command]", err);
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
            console.error("[tour_leg command] Failed to send error message:", replyErr);
        }
    }
}
