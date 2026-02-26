import { AlignmentEnum, AsciiTable3 } from "ascii-table3";
import { FlightHistoryRecord, HealthApiResponse, InitRegistrationResponse, InitServerResponse, UserDetailsData, ApiResponse, PilotStatsData } from "../types/Responses";

export class MessageFormatters {
  static generateHealthString(data: HealthApiResponse): string {
    if (!data) return "No data.";
    let msg = `**Bot Status:**\n**Status:** ${data.status.toUpperCase()}\n**Uptime:** ${data.uptime}\n\n**Services:**\n`;
    for (const [svc, status] of Object.entries(data.services)) {
      msg += `- **${svc}**: ${status.status.toUpperCase()}`;
      if (status.details) msg += ` (${status.details})`;
      msg += `\n`;
    }
    return msg;
  }

  static isInitRegistration(
    r: InitRegistrationResponse | InitServerResponse
  ): r is InitRegistrationResponse {
    return "ifc_id" in r;                // ‹— key that exists only on the user variant
  }

  static makeRegistrationString(resp: InitRegistrationResponse | InitServerResponse): string {
    const key = MessageFormatters.isInitRegistration(resp) ? resp.ifc_id : resp.va_code;
    const header = resp.status
      ? `✅ Registration successful for **${key}**`
      : `❌ Registration failed for **${key}**`;

    const mainMsg = resp.message ? `${resp.message}` : '';

    const stepsMsg =
      resp.steps && resp.steps.length
        ? resp.steps.map(
          (s) =>
            `${s.status ? '✅' : '❌'} **${s.name}:** ${s.message}`
        ).join('\n')
        : '';

    return [header, mainMsg, stepsMsg].filter(Boolean).join('\n\n');

  }

  static makeFlightHistoryTable(records: FlightHistoryRecord[]): string {
    if (!records?.length) return "No flights found.";

    const table = new AsciiTable3()
      .setHeading("Time", "Route", "Equip", "L/V/S/D", "Map");

    for (const rec of records) {
      // "Jan 8 20:03" (UTC)
      const timeStr = new Intl.DateTimeFormat("en-US", {
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
        hour12: false,
        timeZone: "UTC",
      })
        .format(new Date(rec.timestamp))
        .replace(",", "");

      const route = `${rec.origin}-${rec.dest}`;
      const equip = rec.equipment;
      const lvs = `${rec.landings}/${getViolations(rec)}/${shortenServer(
        rec.server,
      )}/${rec.duration}`;

      // Show the link emoji only when a mapUrl exists
      const mapLink = rec.mapUrl && rec.mapUrl.trim() !== "" ? "[🔗]" : "";

      table.addRow(timeStr, route, equip, lvs, mapLink);
    }

    // Centre-align the “L/V/S/D” column
    table.setAlign(3, AlignmentEnum.CENTER);

    const header = "```";
    const footer =
      "```\n" +
      "L - Landings | D - Duration | V - Violations | S - Server (E - Expert, C - Casual, T - Training)\n" +
      "Data is refreshed every 15 minutes.\n" +
      "Map link will appear for flights that meet all of the following:\n" +
      "- Have both origin and destination set\n" +
      "- Duration is greater than 0 minutes\n" +
      "- Created within the last 3 days";

    const links = records
      .filter((rec) => rec.mapUrl && rec.mapUrl.trim() !== "")
      .map(
        (rec) => `🔗 ${rec.origin}-${rec.dest} (${rec.callsign}) → ${rec.mapUrl}`,
      )
      .join("\n");

    // Only add the links block if at least one exists
    const dashboardInfo = "\n\n📊 **View Flight Route Maps:**\nUse `/dashboard` in the web app to see interactive flight route maps with altitude gradients for these flights!";

    return `${header}\n${footer}${links ? `\n${links}` : ""
      }${dashboardInfo}`;
  }

  static generateUserDetailsString(data: UserDetailsData, serverId: string): string {
    if (!data) return "No user data found.";

    // Format the registration date
    const registeredDate = new Date(data.created_at).toLocaleDateString("en-US", {
      year: "numeric",
      month: "long",
      day: "numeric"
    });

    // Build the message
    let msg = "**User Status:**\n";

    // Registration status
    msg += `**Registration Status:** ${data.is_active ? "✅ Registered" : "❌ Not Active"}\n`;
    msg += `**IFC Username:** ${data.if_community_id}\n`;
    msg += `**Registered Since:** ${registeredDate}\n\n`;

    // Current VA status - find affiliation by Discord server ID
    if (data.current_va.is_member) {
      // Note: We need to match by Discord server ID, but the API returns va_id
      // The current_va object tells us if user is a member of THIS server's VA
      const currentAffiliation = data.affiliations.find(aff => aff.is_active);
      if (currentAffiliation) {
        msg += `**Current VA:** ✅ Member of **${currentAffiliation.va_name}** (${currentAffiliation.va_code})\n`;
        msg += `**Role:** ${formatRole(currentAffiliation.role)}\n`;
        msg += `**Status:** ${currentAffiliation.is_active ? "Active" : "Inactive"}\n`;

        const joinedDate = new Date(currentAffiliation.joined_at).toLocaleDateString("en-US", {
          year: "numeric",
          month: "long",
          day: "numeric"
        });
        msg += `**Joined:** ${joinedDate}\n`;
      } else {
        msg += `**Current VA:** ✅ Member\n`;
        msg += `**Role:** ${formatRole(data.current_va.role)}\n`;
      }
    } else {
      msg += `**Current VA:** ❌ Not a member of this Virtual Airline\n`;
    }

    // Other affiliations (show all affiliations since we can't reliably match by server ID)
    if (data.affiliations.length > 1) {
      msg += `\n**All Affiliations:**\n`;
      data.affiliations.forEach(aff => {
        const status = aff.is_active ? "✅" : "❌";
        msg += `${status} **${aff.va_name}** (${aff.va_code}) - ${formatRole(aff.role)}\n`;
      });
    }

    return msg;
  }


}


function shortenServer(server: string): string {
  const s = server.toLowerCase();
  if (s.includes("expert")) return "E";
  if (s.includes("casual")) return "C";
  if (s.includes("training")) return "T";
  return "?";
}

// You can enhance this later if you add violations to the data
function getViolations(rec: FlightHistoryRecord): number {
  // Placeholder — update this when violations are available in the record
  return rec.violations;
}

function formatRole(role: string): string {
  const roleMap: { [key: string]: string } = {
    "admin": "👑 Admin",
    "staff": "⭐ Staff",
    "pilot": "✈️ Pilot"
  };
  return roleMap[role.toLowerCase()] || role;
}
