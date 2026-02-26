import { ApiService } from "../services/apiService";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { CUSTOM_IDS } from "../configs/constants";
import { CommandErrorHandler, ValidationPatterns } from "../helpers/commandErrorHandler";

export const data = {
    name: CUSTOM_IDS.REGISTER_MODAL
}

/**
 * Handles user registration modal submission
 * Supports two scenarios:
 * 1. Full registration (REGISTER_MODAL): IFC ID + Last Flight + optional Callsign
 * 2. Link-only (register_link_modal): Callsign only
 */
export async function execute(interaction: DiscordInteraction) {
    const _interaction = interaction.getModalInputInteraction();
    if (!_interaction) return;

    const modalId = _interaction.customId;

    // SCENARIO 1: Link-only modal (registered user linking to VA)
    if (modalId === "register_link_modal") {
        await handleLinkOnlyRegistration(interaction);
        return;
    }

    // SCENARIO 2: Full registration modal (new user registration)
    await handleFullRegistration(interaction);
}

/**
 * Handles full user registration with IFC ID, last flight, and optional callsign
 */
async function handleFullRegistration(interaction: DiscordInteraction) {
    const _interaction = interaction.getModalInputInteraction();
    if (!_interaction) return;

    // Extract inputs
    const ifcId = _interaction.fields.getTextInputValue('ifcId').trim();
    const lastFlight = _interaction.fields.getTextInputValue('lastFlight').trim().toUpperCase();

    // Validate IFC ID
    if (!await CommandErrorHandler.validateInput(
        interaction, ifcId, "IFC ID", ValidationPatterns.IFC_USERNAME, 3, 30
    )) return;

    // Validate flight route
    if (!await CommandErrorHandler.validateInput(
        interaction, lastFlight, "flight route", ValidationPatterns.FLIGHT_ROUTE
    )) return;

    // Log execution
    CommandErrorHandler.logExecution("Registration", _interaction.user.id, _interaction.guildId, {
        ifcId,
        lastFlight
    });

    try {
        const response = await ApiService.initiateRegistration(
            interaction.getMetaInfo(),
            ifcId,
            lastFlight
        );

        // Validate response
        if (!response) {
            await CommandErrorHandler.handleEmptyResponse(interaction);
            return;
        }

        // Send success response
        if (response.success) {
            let message = `✅ **Registration Successful!**\n\n${response.message}\n\n`;

            if (response.is_va_registered) {
                // User is registered to this VA server
                message += "📌 **Next Step:**\nYou're registered to Comrade Bot. Use `/membership join` to link yourself to this Virtual Airline.";
            } else {
                // User registered to bot but not a VA member
                message += "You are successfully setup for using this bot!";
            }

            await interaction.reply({
                content: message,
                ephemeral: true
            });
        } else {
            // Registration failed
            await interaction.reply({
                content: `❌ **Registration Failed**\n\n${response.message}`,
                ephemeral: true
            });
        }

    } catch (error) {
        await CommandErrorHandler.handleApiError(interaction, error, "Registration");
    }
}

/**
 * Handles link-only registration (registered user linking to VA with callsign)
 */
async function handleLinkOnlyRegistration(interaction: DiscordInteraction) {
    const _interaction = interaction.getModalInputInteraction();
    if (!_interaction) return;

    // Extract callsign
    const callsign = _interaction.fields.getTextInputValue('callsign').trim();

    // Validate callsign format (1-5 digits)
    if (!/^\d{1,5}$/.test(callsign)) {
        await interaction.reply({
            content: "❌ **Validation Error**\nCallsign must be 1-5 digits only.",
            ephemeral: true
        });
        return;
    }

    // Log execution
    CommandErrorHandler.logExecution("VA Link", _interaction.user.id, _interaction.guildId, {
        callsign
    });

    try {
        // Call registration API with only callsign (backend will detect existing user and link to VA)
        const response = await ApiService.linkUserToVA(
            interaction.getMetaInfo(),
            callsign
        );

        // Validate response
        if (!response?.result) {
            await CommandErrorHandler.handleEmptyResponse(interaction);
            return;
        }

        // Send success response
        await interaction.reply({
            content: `✅ **Successfully Linked to Virtual Airline!**\n\nYou're now linked with callsign **${callsign}**.\n\nUse \`/help\` to learn how to use Comrade Bot.`,
            ephemeral: true
        });

    } catch (error) {
        await CommandErrorHandler.handleApiError(interaction, error, "VA Link");
    }
}

export default {
    data,
    execute
};
