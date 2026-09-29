import { ButtonInteraction, ChatInputCommandInteraction, InteractionReplyOptions, InteractionEditReplyOptions, MessageFlags, ModalSubmitInteraction, StringSelectMenuInteraction } from "discord.js"

type AnyInteraction = ChatInputCommandInteraction | ModalSubmitInteraction | ButtonInteraction | StringSelectMenuInteraction;

export class DiscordInteraction {

    public guildId: string | null;
    public userId: string
    private _interaction: AnyInteraction
    replied: any;
    deferred: any;

    constructor(interaction: AnyInteraction) {
        this.guildId = interaction.guildId;
        this.userId = interaction.user.id;
        this._interaction = interaction;
        this.deferred = interaction.deferred;
        this.replied = interaction.replied;
    }


    public async reply(message: InteractionReplyOptions | string) {
        this._interaction.reply(message);
    }

    public async editReply(message: InteractionEditReplyOptions | string) {
        return this._interaction.editReply(message)
    }

    public isChatInputCommand(): boolean {
        return this._interaction.isChatInputCommand()
    }

    public async deferReply(ephemeral = false) {
        if (ephemeral) {
            return this._interaction.deferReply({ flags: MessageFlags.Ephemeral })
        }
        return this._interaction.deferReply()
    }


    public getStringParam(id: string, req: boolean): string {
        if (this._interaction.isChatInputCommand()) {
            const data = this._interaction.options.getString(id, req)
            if (data) return data
        }
        return ""
    }

    public getMetaInfo(): MetaInfo {
        return {
            discordId: this.guildId ?? "",
            userId: this.userId
        }
    }

    public getChatInputInteraction(): ChatInputCommandInteraction | null {
        if (this._interaction.isChatInputCommand())
            return this._interaction as ChatInputCommandInteraction;
        return null;
    }

    public getModalInputInteraction(): ModalSubmitInteraction | null {
        if (this._interaction instanceof ModalSubmitInteraction)
            return this._interaction as ModalSubmitInteraction
        return null
    }

    public getButtonInteraction(): ButtonInteraction | null {
        if (this._interaction.isButton())
            return this._interaction as ButtonInteraction
        return null
    }

    public getSelectMenuInteraction(): StringSelectMenuInteraction | null {
        if (this._interaction instanceof StringSelectMenuInteraction)
            return this._interaction as StringSelectMenuInteraction;
        return null;
    }

};

export type CommandHandler = typeof status;


export type MetaInfo = {
    discordId: string;
    userId: string;
}