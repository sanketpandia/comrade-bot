import { SlashCommandBuilder, ActionRowBuilder, ButtonBuilder, ButtonStyle } from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { ApiService } from "../services/apiService";
import { UnauthorizedError } from "../helpers/UnauthorizedException";

export const data = new SlashCommandBuilder()
    .setName("log")
    .setDescription("File a PIREP for your current flight");

export async function execute(interaction: DiscordInteraction) {
    const chat = interaction.getChatInputInteraction();
    if (!chat) return;

    try {
        // Defer the reply since we're making API calls (keep token alive)
        // ephemeral: true makes the message only visible to the user who ran /log
        await chat.deferReply({ ephemeral: true });

        const metaInfo = interaction.getMetaInfo();

        try {
            // Fetch PIREP configuration
            const pirepConfig = await ApiService.getPirepConfig(metaInfo);

            if (!pirepConfig.result) {
                await chat.editReply({
                    embeds: [{
                        title: "Error",
                        description: "❌ Failed to fetch flight configuration. No data received.",
                        color: 0xff0000,
                        timestamp: new Date().toISOString()
                    }]
                });
                return;
            }

            const { user_info, available_modes } = pirepConfig.result;

            // Filter only valid modes
            const validModes = available_modes.filter((mode: any) => mode.status === "valid");

            if (validModes.length === 0) {
                await chat.editReply({
                    embeds: [{
                        title: "No Valid Modes",
                        description: `❌ Your current flight is not eligible for any PIREP modes.\n\n**Current Flight:**\nRoute: ${user_info.current_route}\nAircraft: ${user_info.current_aircraft} (${user_info.current_livery})\n\nEnsure you're on an eligible route for at least one mode.`,
                        color: 0xff9900,
                        timestamp: new Date().toISOString()
                    }]
                });
                return;
            }

            // Create mode selection buttons
            const modeButtons = validModes.map((mode: any) =>
                new ButtonBuilder()
                    .setCustomId(`mode_${mode.mode_id}`)
                    .setLabel(mode.display_name)
                    .setStyle(ButtonStyle.Primary)
            );

            // Split buttons into rows (5 buttons per row max for Discord)
            const buttonRows: ActionRowBuilder<ButtonBuilder>[] = [];
            for (let i = 0; i < modeButtons.length; i += 5) {
                const row = new ActionRowBuilder<ButtonBuilder>()
                    .addComponents(modeButtons.slice(i, i + 5));
                buttonRows.push(row);
            }

            // Build the description with flight info and available modes
            const modeList = validModes
                .map((mode: any) => `• **${mode.display_name}**`)
                .join("\n");

            await chat.editReply({
                embeds: [{
                    title: "PIREP Filing",
                    description: `Select a flight mode to file your PIREP.\n\n**Current Flight:**\nCallsign: ${user_info.callsign}\nRoute: ${user_info.current_route}\nAircraft: ${user_info.current_aircraft} (${user_info.current_livery})\nAltitude: ${user_info.current_altitude || "N/A"} ft\nSpeed: ${user_info.current_speed || "N/A"} knots\n\n**Available Modes:**\n${modeList}`,
                    color: 0x0099ff,
                    timestamp: new Date().toISOString()
                }],
                components: buttonRows
            });
        } catch (apiErr: any) {
            if (apiErr instanceof UnauthorizedError) {
                await chat.editReply({
                    embeds: [{
                        title: "Not Authorized",
                        description: `❌ ${apiErr.message}`,
                        color: 0xff0000,
                        timestamp: new Date().toISOString()
                    }]
                });
                return;
            }

            console.error("[log command] Error fetching PIREP config:", apiErr);

            // Check if user is not in flight
            if (apiErr.message?.includes("not currently flying") ||
                apiErr.message?.includes("not in flight") ||
                apiErr.message?.includes("no live flight found")) {
                await chat.editReply({
                    embeds: [{
                        title: "Not In Flight",
                        description: "❌ You are not currently flying!\n\nPlease join a flight before filing a PIREP.",
                        color: 0xff0000,
                        timestamp: new Date().toISOString()
                    }]
                });
            } else {
                await chat.editReply({
                    embeds: [{
                        title: "Error",
                        description: `❌ Failed to fetch PIREP configuration: ${apiErr.message}`,
                        color: 0xff0000,
                        timestamp: new Date().toISOString()
                    }]
                });
            }
        }
    } catch (err) {
        console.error("[log command]", err);
        try {
            await chat.editReply({
                embeds: [{
                    title: "Error",
                    description: `❌ An error occurred: ${String(err)}`,
                    color: 0xff0000
                }]
            });
        } catch (replyErr) {
            console.error("[log command] Failed to send error message:", replyErr);
        }
    }
}
