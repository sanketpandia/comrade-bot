import { ModalBuilder, ActionRowBuilder, ModalActionRowComponentBuilder, TextInputBuilder, TextInputStyle } from "discord.js";
import { CUSTOM_IDS } from "../configs/constants";
import { FormField, ModeResponse } from "../types/Responses";

const DISCORD_MODAL_MAX_FIELDS = 5;

export function buildSingleModalForMode(mode: ModeResponse): ModalBuilder {
    const modal = new ModalBuilder()
        .setCustomId(`${CUSTOM_IDS.PIREP_MODAL}_${mode.mode_id}`)
        .setTitle(`${mode.display_name} - PIREP`);

    const totalComponents = (mode.requires_route_selection ? 1 : 0) + mode.fields.length;
    if (totalComponents > DISCORD_MODAL_MAX_FIELDS) {
        throw new Error(`Mode config exceeds modal field limit (${totalComponents}/${DISCORD_MODAL_MAX_FIELDS})`);
    }

    if (mode.requires_route_selection) {
        const routeField = new TextInputBuilder()
            .setCustomId("route_id")
            .setLabel("Route (e.g., LFPG-EGLL)")
            .setPlaceholder("Enter the route code")
            .setStyle(TextInputStyle.Short)
            .setRequired(true);

        if (mode.autofill_route) {
            routeField.setValue(mode.autofill_route);
        }

        modal.addComponents(new ActionRowBuilder<ModalActionRowComponentBuilder>().addComponents(routeField));
    }

    for (const field of mode.fields) {
        modal.addComponents(new ActionRowBuilder<ModalActionRowComponentBuilder>().addComponents(createInputField(field)));
    }

    return modal;
}

function createInputField(field: FormField): TextInputBuilder {
    const input = new TextInputBuilder()
        .setCustomId(field.name)
        .setLabel(field.label)
        .setRequired(field.required);

    if (field.type === "textarea") {
        input.setStyle(TextInputStyle.Paragraph);
    } else {
        input.setStyle(TextInputStyle.Short);
    }

    if (field.name === "flight_time") {
        input.setPlaceholder("HH:MM");
    }

    return input;
}
