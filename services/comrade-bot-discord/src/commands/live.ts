import { SlashCommandBuilder, ActionRowBuilder, ButtonBuilder, ButtonStyle } from "discord.js";
import { ApiService } from "../services/apiService";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { UnauthorizedError } from "../helpers/UnauthorizedException";
import { PermissionDeniedError } from "../helpers/PermissionDeniedException";

export const data = new SlashCommandBuilder()
    .setName("live")
    .setDescription("Get a secure link to the live flights map");

export async function execute(interaction: DiscordInteraction): Promise<void> {
    const chat = interaction.getChatInputInteraction();
    if (!chat) return;

    await chat.deferReply({ ephemeral: true });

    const metaInfo = interaction.getMetaInfo();

    try {
        const signedLinkResponse = await ApiService.generateSignedLink(metaInfo, "/maps/flights/active");

        if (!signedLinkResponse?.result?.url) {
            await chat.editReply({
                embeds: [{
                    title: "Error",
                    description: "Could not generate map link.\nPlease ensure you are registered with Politburo.",
                    color: 0xff0000,
                    timestamp: new Date().toISOString(),
                }],
            });
            return;
        }

        const mapLink = signedLinkResponse.result.url;
        const expiresIn = signedLinkResponse.result.expires_in || 900;

        const mapButton = new ButtonBuilder()
            .setLabel("Open Live Map")
            .setStyle(ButtonStyle.Link)
            .setURL(mapLink);

        const buttonRow = new ActionRowBuilder<ButtonBuilder>().addComponents(mapButton);

        await chat.editReply({
            embeds: [{
                title: "Live flights map",
                description: `Click the button below to open the Infinite Flight live map.\n\n*Link expires in ${Math.floor(expiresIn / 60)} minutes*`,
                color: 0x0099ff,
                timestamp: new Date().toISOString(),
            }],
            components: [buttonRow],
        });
    } catch (err: unknown) {
        if (err instanceof UnauthorizedError) {
            await chat.editReply({
                embeds: [{
                    title: "Not Authorized",
                    description: err.message,
                    color: 0xff0000,
                    timestamp: new Date().toISOString(),
                }],
            });
            return;
        }

        if (err instanceof PermissionDeniedError) {
            await chat.editReply({
                embeds: [{
                    title: "Registration Required",
                    description: `${err.message}\n\nUse \`/register\` before opening the live map.`,
                    color: 0xff9900,
                    timestamp: new Date().toISOString(),
                }],
            });
            return;
        }

        console.error("[live command]", err);
        const message = err instanceof Error ? err.message : String(err);
        await chat.editReply({
            embeds: [{
                title: "Error",
                description: `Unable to generate map link: ${message}`,
                color: 0xff0000,
                timestamp: new Date().toISOString(),
            }],
        });
    }
}
