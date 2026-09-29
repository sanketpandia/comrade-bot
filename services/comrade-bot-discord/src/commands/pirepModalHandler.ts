import { ApiService } from "../services/apiService";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { CUSTOM_IDS } from "../configs/constants";
import { UnauthorizedError } from "../helpers/UnauthorizedException";
import { PermissionDeniedError } from "../helpers/PermissionDeniedException";

export const data = {
    name: CUSTOM_IDS.PIREP_MODAL
};

/**
 * Handles PIREP modal submission
 * Extracts form data, validates, and submits to backend
 */
export async function execute(wrapped: DiscordInteraction): Promise<void> {
    const modalInteraction = wrapped.getModalInputInteraction();
    if (!modalInteraction) return;

    try {
        // Extract mode_id from modal custom ID (format: "pirepModal_modeId")
        const customIdParts = modalInteraction.customId.split('_');
        const modeId = customIdParts.slice(1).join('_'); // Handle mode IDs with underscores

        // Extract form data from modal
        const flightTime = modalInteraction.fields.getTextInputValue("flight_time");

        // Optional fields based on mode
        let routeId: string | undefined;
        let pilotRemarks: string | undefined;
        let fuelKg: number | undefined;
        let cargoKg: number | undefined;
        let passengers: number | undefined;

        try {
            routeId = modalInteraction.fields.getTextInputValue("route_id");
        } catch {
            // Route field not present in this mode
        }

        try {
            pilotRemarks = modalInteraction.fields.getTextInputValue("pilot_remarks");
        } catch {
            // Remarks not present
        }

        try {
            fuelKg = parseInt(modalInteraction.fields.getTextInputValue("fuel_kg"));
        } catch {
            // Fuel not present or invalid
        }

        try {
            cargoKg = parseInt(modalInteraction.fields.getTextInputValue("cargo_kg"));
        } catch {
            // Cargo not present or invalid
        }

        try {
            passengers = parseInt(modalInteraction.fields.getTextInputValue("passengers"));
        } catch {
            // Passengers not present or invalid
        }

        // Build summary with submitted PIREP data
        const summaryLines = [
            `**Mode:** ${modeId}`,
            ``,
            `**Flight Data:**`,
            `Flight Time: ${flightTime}`,
        ];

        if (routeId) summaryLines.push(`Route: ${routeId}`);
        if (pilotRemarks) summaryLines.push(`Remarks: ${pilotRemarks}`);
        if (fuelKg) summaryLines.push(`Fuel: ${fuelKg} kg`);
        if (cargoKg) summaryLines.push(`Cargo: ${cargoKg} kg`);
        if (passengers) summaryLines.push(`Passengers: ${passengers}`);

        // Build PIREP submission data (only include defined fields)
        const pirepData: any = {
            mode: modeId,
            flight_time: flightTime,
        };

        // Only add optional fields if they are defined
        if (routeId !== undefined && routeId !== null && routeId !== '') {
            pirepData.route_id = routeId;
        }
        if (pilotRemarks !== undefined && pilotRemarks !== null && pilotRemarks !== '') {
            pirepData.pilot_remarks = pilotRemarks;
        }
        if (fuelKg !== undefined && fuelKg !== null && !isNaN(fuelKg)) {
            pirepData.fuel_kg = fuelKg;
        }
        if (cargoKg !== undefined && cargoKg !== null && !isNaN(cargoKg)) {
            pirepData.cargo_kg = cargoKg;
        }
        if (passengers !== undefined && passengers !== null && !isNaN(passengers)) {
            pirepData.passengers = passengers;
        }

        // Log the collected data
        console.log("[handlePirepModal] Submitting PIREP Data:", pirepData);

        // Defer the reply now that we have all data extracted
        // This will extend the interaction timeout while we call the API
        try {
            await modalInteraction.deferReply();
        } catch (deferErr) {
            console.error("[handlePirepModal] Failed to defer reply:", deferErr);
            // If defer fails, try replying directly instead
            try {
                await modalInteraction.reply({
                    content: "⏳ Processing PIREP submission...",
                    flags: 64 // Ephemeral flag
                });
            } catch (quickReplyErr) {
                console.error("[handlePirepModal] Failed to send quick reply:", quickReplyErr);
                return;
            }
        }

        // Call API to submit PIREP
        try {
            const metaInfo = wrapped.getMetaInfo();
            const submitResponse = await ApiService.submitPirep(metaInfo, pirepData);

            // Check if submission was successful
            // Backend returns httpdto envelope: {status: "ok", result: {...}, responseTimeMs: 42}
            // or error: {status: "error", error: {code, message}, responseTimeMs: 42}
            const isSuccess = submitResponse.status === "ok" && submitResponse.result !== undefined;

            if (!isSuccess) {
                console.error("[handlePirepModal] Submit failed:", submitResponse);

                // Extract error message from httpdto error envelope
                const errorCode = submitResponse.error?.code;
                const backendMessage = submitResponse.error?.message || submitResponse.message;
                
                // Handle specific error types with user-friendly messages
                let errorMessage: string;
                let errorTitle = "❌ PIREP Submission Failed";

                switch (errorCode) {
                    case "FLIGHT_NOT_FOUND":
                        errorMessage = "❌ **Could not identify your flight.**\n\nPlease ensure you are currently in the game's server with your VA callsign.";
                        break;
                    case "ROUTE_NOT_MATCHED":
                        errorMessage = "❌ **Your flight plan start and end do not denote a tour route.**\n\nPlease check that your flight plan matches one of the tour leg routes.";
                        break;
                    case "NO_FLIGHTS":
                        errorMessage = "❌ **No live flights found.**\n\nPlease ensure you are currently flying in the game's server with your VA callsign.";
                        break;
                    case "FLIGHTS_FETCH_ERROR":
                        errorMessage = "❌ **Failed to fetch live flights.**\n\nThere was an error retrieving flight data. Please try again in a moment.";
                        break;
                    case "NO_ROUTE":
                        errorMessage = "❌ **Flight has no route information.**\n\nYour flight plan must include origin and destination airports.";
                        break;
                    case "VA_NOT_FOUND":
                        errorMessage = "❌ **Virtual airline not found.**\n\nPlease ensure you are registered with a virtual airline.";
                        break;
                    case "USER_NOT_FOUND":
                        errorMessage = "❌ **User not found.**\n\nPlease register your account using the `/register` command.";
                        break;
                    case "NO_COMMUNITY_ID":
                        errorMessage = "❌ **Missing Infinite Flight Community ID.**\n\nPlease ensure your account is properly linked with your Infinite Flight community profile.";
                        break;
                    case "NO_ACTIVE_TOUR":
                        errorMessage = "❌ **No active tour found.**\n\nThere is no active world tour event for your virtual airline.";
                        break;
                    case "TOUR_LOOKUP_ERROR":
                        errorMessage = "❌ **Failed to retrieve tour information.**\n\nThere was an error looking up the active tour. Please try again.";
                        break;
                    case "NO_CREDENTIALS":
                        errorMessage = "❌ **Airtable credentials not configured.**\n\nYour virtual airline administrator needs to configure Airtable credentials in the datasource settings.";
                        errorTitle = "⚠️ Configuration Required";
                        break;
                    case "NO_SCHEMA":
                        errorMessage = "❌ **PIREP schema not configured.**\n\nYour virtual airline administrator needs to configure the PIREP schema in the datasource settings.";
                        errorTitle = "⚠️ Configuration Required";
                        break;
                    case "CONFIG_ERROR":
                        errorMessage = "❌ **Configuration error.**\n\nThere was an error reading the Airtable configuration. Please contact your virtual airline administrator.";
                        break;
                    case "SCHEMA_ERROR":
                        errorMessage = "❌ **Schema processing error.**\n\nThere was an error processing the PIREP schema. Please contact your virtual airline administrator.";
                        break;
                    case "AIRTABLE_ERROR":
                        errorMessage = "❌ **Failed to submit to Airtable.**\n\nThere was an error submitting your PIREP to Airtable. Please try again or contact support.";
                        break;
                    case "BAD_REQUEST":
                        errorMessage = backendMessage || "❌ **Invalid request.**\n\nPlease check your input and try again.";
                        break;
                    case "PARSE_ERROR":
                        errorMessage = "❌ **Response parsing error.**\n\nThere was an error processing the server response. Please try again.";
                        break;
                    default:
                        // Use backend message if available, otherwise generic message
                        errorMessage = backendMessage || "❌ **Failed to process PIREP submission.**\n\nPlease try again. If the problem persists, contact support.";
                        if (errorCode) {
                            errorMessage += `\n\nError Code: ${errorCode}`;
                        }
                        break;
                }

                await modalInteraction.editReply({
                    embeds: [{
                        title: errorTitle,
                        description: errorMessage,
                        color: 0xff0000,
                        timestamp: new Date().toISOString(),
                    }]
                });
                return;
            }

            // Show success response
            console.log("[handlePirepModal] PIREP submitted successfully:", submitResponse);

            const result = submitResponse.result;
            const pirepId = result?.pirep_id || "N/A";
            const aircraft = result?.aircraft || "N/A";
            const livery = result?.livery || "N/A";
            const flightTime = result?.flight_time || "N/A";
            const routeName = result?.route_name || "N/A";

            // Build fields array with the response data
            const fields = [
                {
                    name: "PIREP ID",
                    value: pirepId,
                    inline: true
                }
            ];

            // Add tour-specific fields if available (for tour mode)
            if (aircraft !== "N/A" || livery !== "N/A" || flightTime !== "N/A" || routeName !== "N/A") {
                if (aircraft !== "N/A") {
                    fields.push({
                        name: "Aircraft",
                        value: aircraft,
                        inline: true
                    });
                }
                if (livery !== "N/A") {
                    fields.push({
                        name: "Livery",
                        value: livery,
                        inline: true
                    });
                }
                if (flightTime !== "N/A") {
                    fields.push({
                        name: "Flight Time",
                        value: flightTime,
                        inline: true
                    });
                }
                if (routeName !== "N/A") {
                    fields.push({
                        name: "Route",
                        value: routeName,
                        inline: true
                    });
                }
            }

            fields.push({
                name: "Processing Time",
                value: submitResponse.responseTimeMs ? `${submitResponse.responseTimeMs}ms` : "N/A",
                inline: true
            });

            await modalInteraction.editReply({
                embeds: [{
                    title: "✅ PIREP Submitted Successfully",
                    description: summaryLines.join("\n"),
                    color: 0x00ff00,
                    timestamp: new Date().toISOString(),
                    fields: fields
                }]
            });
        } catch (submitErr) {
            console.error("[handlePirepModal] Submit API call failed:", submitErr);
            try {
                if (submitErr instanceof UnauthorizedError) {
                    await modalInteraction.editReply({
                        embeds: [{
                            title: "🔒 Not Authorized",
                            description: `❌ ${submitErr.message}`,
                            color: 0xff0000,
                            timestamp: new Date().toISOString(),
                        }]
                    });
                    return;
                }

                if (submitErr instanceof PermissionDeniedError) {
                    await modalInteraction.editReply({
                        embeds: [{
                            title: "⚠️ Registration Required",
                            description: `❌ ${submitErr.message}\n\nPlease register your account using the \`/register\` command before submitting PIREPs.`,
                            color: 0xff9900,
                            timestamp: new Date().toISOString(),
                        }]
                    });
                    return;
                }

                // Handle network errors and other exceptions
                let errorMessage = "❌ **Failed to submit PIREP to backend.**\n\n";
                if (submitErr instanceof Error) {
                    const errMsg = submitErr.message.toLowerCase();
                    if (errMsg.includes("fetch") || errMsg.includes("network") || errMsg.includes("connection")) {
                        errorMessage += "**Network error:** Unable to connect to the server. Please check your internet connection and try again.";
                    } else if (errMsg.includes("timeout")) {
                        errorMessage += "**Request timeout:** The server took too long to respond. Please try again.";
                    } else if (errMsg.includes("json") || errMsg.includes("parse")) {
                        errorMessage += "**Response error:** The server returned an invalid response. Please try again.";
                    } else {
                        errorMessage += `**Error:** ${submitErr.message}`;
                    }
                } else {
                    errorMessage += "An unexpected error occurred. Please try again later.";
                }

                await modalInteraction.editReply({
                    embeds: [{
                        title: "❌ PIREP Submission Error",
                        description: errorMessage,
                        color: 0xff0000,
                        timestamp: new Date().toISOString(),
                    }]
                });
            } catch (editErr) {
                console.error("[handlePirepModal] Failed to edit reply:", editErr);
            }
        }
    } catch (err) {
        console.error("[handlePirepModal]", err);
        try {
            await modalInteraction?.reply({
                content: `❌ An error occurred: ${String(err)}`,
                flags: 64 // Ephemeral
            });
        } catch (replyErr) {
            console.error("[handlePirepModal] Failed to send error message:", replyErr);
        }
    }
}

export default { data, execute };
