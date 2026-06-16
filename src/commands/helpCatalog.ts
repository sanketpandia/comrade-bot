export type HelpCategory = "onboarding" | "status" | "flight" | "admin" | "events";

export interface HelpEntry {
    name: string;
    category: HelpCategory;
    summary: string;
    details: string[];
    examples?: string[];
    visible: boolean;
}

export const helpCatalog: HelpEntry[] = [
    {
        name: "register",
        category: "onboarding",
        summary: "Create your global Comrade Bot account and link the current VA when this server is configured as one.",
        details: [
            "Account creation collects only your IF Community username and last-flight route for ownership validation.",
            "Callsign is collected only after the current Discord server is confirmed as a configured VA.",
            "Use only your own IF account; the validation step helps prevent impersonation.",
        ],
        examples: ["/register"],
        visible: true,
    },
    {
        name: "status",
        category: "status",
        summary: "Show your account and current-server VA membership status.",
        details: ["Does not include backend health. Use /botstatus for operational status."],
        examples: ["/status"],
        visible: true,
    },
    {
        name: "botstatus",
        category: "status",
        summary: "Show sanitized bot/backend operational status.",
        details: ["Shows only high-level availability and never raw dependency details."],
        examples: ["/botstatus"],
        visible: true,
    },
    {
        name: "log",
        category: "flight",
        summary: "File a PIREP for your current flight when your VA has enabled flight logging.",
        details: ["Requires a registered account and current-server VA membership."],
        examples: ["/log"],
        visible: true,
    },
    {
        name: "logbook",
        category: "flight",
        summary: "View pilot flight history. Staff/Admin only.",
        details: ["Use the pilot's IF Community username."],
        examples: ["/logbook ifc_id:john_doe123"],
        visible: true,
    },
    {
        name: "live",
        category: "flight",
        summary: "View cached active flights for the current server's VA.",
        details: [
            "Run it in a configured VA server; the bot sends your Discord user/server context automatically.",
            "Returns an ephemeral summary/table and an Open Live Map button when a signed map link is available.",
        ],
        examples: ["/live"],
        visible: true,
    },
    {
        name: "stats",
        category: "flight",
        summary: "View your pilot statistics and synced VA activity.",
        details: ["Stats may be cached and can lag behind live activity."],
        examples: ["/stats"],
        visible: true,
    },
    {
        name: "dashboard",
        category: "flight",
        summary: "Generate a signed link to the Vizburo dashboard.",
        details: ["Links are time-limited."],
        examples: ["/dashboard"],
        visible: true,
    },
    {
        name: "initserver",
        category: "admin",
        summary: "Bootstrap this Discord server with a VA Code / ID. Admin only.",
        details: [
            "Collects only your VA Code / ID in Discord; setup continues in Vizburo.",
            "Admins must run /register first.",
            "Use a desktop browser or desktop view for Basic Setup, including callsign matching.",
        ],
        examples: ["/initserver"],
        visible: true,
    },
    {
        name: "events",
        category: "events",
        summary: "View VA events and tours where enabled.",
        details: ["Availability depends on current VA configuration."],
        visible: true,
    },
];

export function visibleHelpEntries(): HelpEntry[] {
    return helpCatalog.filter(entry => entry.visible);
}

export function findHelpEntry(name: string): HelpEntry | undefined {
    return visibleHelpEntries().find(entry => entry.name === name);
}
