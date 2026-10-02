import { MetaInfo } from "../types/DiscordInteraction";

/** Read at request time so .env is loaded before first Politburo call. */
export function getPolitburoApiKey(): string {
    return process.env.API_KEY?.trim() ?? "";
}

export function getPolitburoApiUrl(): string {
    return process.env.API_URL?.trim() || "http://localhost:8080";
}

export function generateMetaHeaders (metainfo: MetaInfo) {
    return {
        "X-Discord-Id": metainfo.userId,
        "X-Server-Id": metainfo.discordId,
        "X-API-Key": getPolitburoApiKey(),
        "Content-Type": "application/json"
    }
}

export function generateRegistrationMetaHeaders (metainfo: MetaInfo) {
    return {
        "X-Discord-User-Id": metainfo.userId,
        "X-Discord-Server-Id": metainfo.discordId,
        "X-API-Key": getPolitburoApiKey(),
        "Content-Type": "application/json"
    }
}
