import { DiscordInteraction } from "../types/DiscordInteraction";
import { UnauthorizedError } from "./UnauthorizedException";
import { PermissionDeniedError } from "./PermissionDeniedException";

/**
 * Standard error messages for common scenarios
 */
export const ErrorMessages = {
    EMPTY_RESPONSE: "❌ Empty response from API. Please try again later.",
    GENERIC_ERROR: "⚠️ **Operation Failed**\nSomething went wrong. Please try again later or contact support.",
    UNAUTHORIZED: (message: string) => `❌ **Authorization Failed**\n${message}`,
    PERMISSION_DENIED: (message: string) => `🔒 **Permission Denied**\n${message}`,
    VALIDATION_ERROR: (field: string, requirement: string) => `❌ Invalid ${field}. ${requirement}`,
    USER_ALREADY_REGISTERED: "❌ **Already Registered**\nThis IFC account is already registered. Use `/status` to view your details.",
    IFC_USER_NOT_FOUND: "❌ **IFC User Not Found**\nThe provided IFC username was not found. Please check your spelling and try again.",
    FLIGHT_MISMATCH: "❌ **Flight Verification Failed**\nThe flight route you provided doesn't match your most recent flight. Please verify your last flight in the Infinite Flight app and try again.",
} as const;

/**
 * Standard input validation patterns
 */
export const ValidationPatterns = {
    FLIGHT_ROUTE: /^[A-Z]{4}-[A-Z]{4}$/,
    VA_CODE: /^[A-Z0-9]{3,5}$/,
    IFC_USERNAME: /^[a-zA-Z0-9_-]{3,30}$/,
} as const;

/**
 * Centralized error handler for modal interactions
 * Provides consistent error formatting and logging
 */
export class CommandErrorHandler {
    /**
     * Handle API errors with consistent formatting
     */
    static async handleApiError(
        interaction: DiscordInteraction,
        error: unknown,
        operation: string
    ): Promise<void> {
        console.error(`[${operation} Error]`, error);

        // Handle unauthorized errors (401)
        if (error instanceof UnauthorizedError) {
            await interaction.reply({
                content: ErrorMessages.UNAUTHORIZED(error.message),
                ephemeral: true
            });
            return;
        }

        // Handle permission denied errors (403)
        if (error instanceof PermissionDeniedError) {
            await interaction.reply({
                content: ErrorMessages.PERMISSION_DENIED(error.message),
                ephemeral: true
            });
            return;
        }

        // Handle specific registration errors
        const errorMessage = error instanceof Error ? error.message.toLowerCase() : '';

        if (errorMessage.includes('already registered')) {
            await interaction.reply({
                content: ErrorMessages.USER_ALREADY_REGISTERED,
                ephemeral: true
            });
            return;
        }

        if (errorMessage.includes('user not found') || errorMessage.includes('ifc user not found')) {
            await interaction.reply({
                content: ErrorMessages.IFC_USER_NOT_FOUND,
                ephemeral: true
            });
            return;
        }

        if (errorMessage.includes('flight') && (errorMessage.includes('mismatch') || errorMessage.includes('verification failed'))) {
            await interaction.reply({
                content: ErrorMessages.FLIGHT_MISMATCH,
                ephemeral: true
            });
            return;
        }

        // Handle generic errors
        await interaction.reply({
            content: ErrorMessages.GENERIC_ERROR,
            ephemeral: true
        });
    }

    /**
     * Validate empty response
     */
    static async handleEmptyResponse(interaction: DiscordInteraction): Promise<boolean> {
        await interaction.reply({
            content: ErrorMessages.EMPTY_RESPONSE,
            ephemeral: true
        });
        return false;
    }

    /**
     * Validate input field
     */
    static async validateInput(
        interaction: DiscordInteraction,
        value: string,
        fieldName: string,
        pattern?: RegExp,
        minLength?: number,
        maxLength?: number
    ): Promise<boolean> {
        // Check length
        if (minLength && value.length < minLength) {
            await interaction.reply({
                content: ErrorMessages.VALIDATION_ERROR(
                    fieldName,
                    `Must be at least ${minLength} characters.`
                ),
                ephemeral: true
            });
            return false;
        }

        if (maxLength && value.length > maxLength) {
            await interaction.reply({
                content: ErrorMessages.VALIDATION_ERROR(
                    fieldName,
                    `Must be no more than ${maxLength} characters.`
                ),
                ephemeral: true
            });
            return false;
        }

        // Check pattern
        if (pattern && !pattern.test(value)) {
            await interaction.reply({
                content: ErrorMessages.VALIDATION_ERROR(
                    fieldName,
                    `Does not match required format.`
                ),
                ephemeral: true
            });
            return false;
        }

        return true;
    }

    /**
     * Log command execution
     */
    static logExecution(
        commandName: string,
        userId: string,
        guildId: string | null,
        params: Record<string, any>
    ): void {
        console.log(`[${commandName}] User: ${userId}, Guild: ${guildId || 'DM'}, Params:`, params);
    }
}
