import { DiscordInteraction } from "../types/DiscordInteraction";

/**
 * Handle the "File PIREP" button click from the /tour command
 * This is a placeholder that shows a message directing users to use /log
 */
export async function handleTourFilePirep(interaction: DiscordInteraction): Promise<void> {
    const button = interaction.getButtonInteraction();
    if (!button) return;

    try {
        // Show placeholder message directing to /log command
        await button.reply({
            embeds: [{
                title: "File PIREP",
                description: "To file a PIREP for the tour, use the `/log` command.\n\nThis will allow you to file a PIREP for your current flight.",
                color: 0x0099ff,
                timestamp: new Date().toISOString()
            }],
            ephemeral: true
        });
    } catch (err) {
        console.error("[tourButtonHandler] Error handling File PIREP button:", err);
        try {
            await button.reply({
                content: "⚠️ An error occurred. Please try using `/log` to file your PIREP.",
                ephemeral: true
            });
        } catch (replyErr) {
            console.error("[tourButtonHandler] Failed to send error message:", replyErr);
        }
    }
}
