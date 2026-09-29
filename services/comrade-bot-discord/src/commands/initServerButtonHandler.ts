import { ModalBuilder, TextInputBuilder, ActionRowBuilder, TextInputStyle } from "discord.js";
import { CUSTOM_IDS } from "../configs/constants";
import { DiscordInteraction } from "../types/DiscordInteraction";

/**
 * Handles the "Proceed" button click for server initialization
 * Shows the modal with VA details form
 */
export async function handleInitServerProceed(interaction: DiscordInteraction) {
  const buttonInteraction = interaction.getButtonInteraction();
  if (!buttonInteraction) return;

  // Create modal with the single Discord bootstrap field.
  const vaCodeInput = new TextInputBuilder()
    .setCustomId("vaCode")
    .setLabel("VA Code / ID")
    .setStyle(TextInputStyle.Short)
    .setMinLength(3)
    .setMaxLength(5)
    .setRequired(true)
    .setPlaceholder("IFE");

  const modal = new ModalBuilder()
    .setCustomId(CUSTOM_IDS.INIT_SERVER_MODAL)
    .setTitle("Start VA Setup");

  modal.addComponents(
    new ActionRowBuilder<TextInputBuilder>().addComponents(vaCodeInput),
  );

  await buttonInteraction.showModal(modal);
}

export default {
  handleInitServerProceed
};
