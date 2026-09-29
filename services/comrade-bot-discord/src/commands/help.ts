import { EmbedBuilder, SlashCommandBuilder } from "discord.js";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { findHelpEntry, HelpCategory, HelpEntry, visibleHelpEntries } from "./helpCatalog";

const entries = visibleHelpEntries();

export const data = new SlashCommandBuilder()
    .setName("help")
    .setDescription("Get help with Comrade Bot commands")
    .addStringOption(option => {
        const commandOption = option
            .setName("command")
            .setDescription("Specific command to get help with")
            .setRequired(false);
        for (const entry of entries.slice(0, 25)) {
            commandOption.addChoices({ name: entry.name, value: entry.name });
        }
        return commandOption;
    });

export async function execute(interaction: DiscordInteraction) {
    const chat = interaction.getChatInputInteraction();
    if (!chat) return;

    const commandOption = chat.options.getString("command");
    if (!commandOption) {
        await chat.reply({ embeds: [getGeneralHelpEmbed()], ephemeral: true });
        return;
    }

    const entry = findHelpEntry(commandOption);
    if (!entry) {
        await chat.reply({ content: `❌ No help found for command: \`/${commandOption}\``, ephemeral: true });
        return;
    }

    await chat.reply({ embeds: [getCommandHelpEmbed(entry)], ephemeral: true });
}

function getGeneralHelpEmbed(): EmbedBuilder {
    const grouped = entries.reduce<Record<HelpCategory, HelpEntry[]>>((acc, entry) => {
        acc[entry.category] = acc[entry.category] || [];
        acc[entry.category].push(entry);
        return acc;
    }, {} as Record<HelpCategory, HelpEntry[]>);

    const embed = new EmbedBuilder()
        .setTitle("🤖 Comrade Bot Help")
        .setDescription("Use `/help command:<name>` for details. `/register` handles account creation and current-server VA linking when available.");

    for (const [category, categoryEntries] of Object.entries(grouped)) {
        embed.addFields({
            name: formatCategory(category as HelpCategory),
            value: categoryEntries.map(entry => `• \`/${entry.name}\` — ${entry.summary}`).join("\n"),
            inline: false,
        });
    }

    return embed.setFooter({ text: "Need operational status? Use /botstatus" });
}

function getCommandHelpEmbed(entry: HelpEntry): EmbedBuilder {
    const embed = new EmbedBuilder()
        .setTitle(`/${entry.name}`)
        .setDescription(entry.summary)
        .addFields({ name: "Details", value: entry.details.map(detail => `• ${detail}`).join("\n"), inline: false });

    if (entry.examples?.length) {
        embed.addFields({ name: "Examples", value: entry.examples.map(example => `\`${example}\``).join("\n"), inline: false });
    }

    return embed;
}

function formatCategory(category: HelpCategory): string {
    const labels: Record<HelpCategory, string> = {
        onboarding: "📝 Onboarding",
        status: "✅ Status",
        flight: "✈️ Flight tools",
        admin: "🏢 Admin",
        events: "🎫 Events",
    };
    return labels[category];
}
