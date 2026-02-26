import { ApiService } from "../services/apiService";
import { DiscordInteraction } from "../types/DiscordInteraction";
import { CUSTOM_IDS } from "../configs/constants";

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

        // Build PIREP submission data
        const pirepData = {
            mode: modeId,
            route_id: routeId || undefined,
            flight_time: flightTime,
            pilot_remarks: pilotRemarks,
            fuel_kg: fuelKg,
            cargo_kg: cargoKg,
            passengers: passengers,
        };

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

            // Check if submission was successful (result.success flag)
            const responseData = submitResponse.result;
            if (!responseData || !responseData.success) {
                console.error("[handlePirepModal] Submit failed:", submitResponse);

                // Extract error message from response, with multiple fallback options
                const errorMessage =
                    responseData?.error_message ||
                    submitResponse.message ||
                    "Failed to process PIREP submission. Please try again.";

                await modalInteraction.editReply({
                    embeds: [{
                        title: "❌ PIREP Submission Failed",
                        description: `Error: ${errorMessage}`,
                        color: 0xff0000,
                        timestamp: new Date().toISOString(),
                    }]
                });
                return;
            }

            // Show success response
            console.log("[handlePirepModal] PIREP submitted successfully:", submitResponse);

            await modalInteraction.editReply({
                embeds: [{
                    title: "✅ PIREP Submitted Successfully",
                    description: summaryLines.join("\n"),
                    color: 0x00ff00,
                    timestamp: new Date().toISOString(),
                    fields: [
                        {
                            name: "PIREP ID",
                            value: responseData.pirep_id || "N/A",
                            inline: true
                        },
                        {
                            name: "Processing Time",
                            value: submitResponse.responseTimeMs ? `${submitResponse.responseTimeMs}ms` : "N/A",
                            inline: true
                        }
                    ]
                }]
            });
        } catch (submitErr) {
            console.error("[handlePirepModal] Submit API call failed:", submitErr);
            try {
                await modalInteraction.editReply({
                    embeds: [{
                        title: "❌ PIREP Submission Error",
                        description: "Failed to submit PIREP to backend. Please try again later.",
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
