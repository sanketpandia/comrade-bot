import test from "node:test";
import assert from "node:assert/strict";
import { metrics } from "../src/infra/metrics";

test("command metrics use known command labels and duration buckets", async () => {
    metrics.recordCommand("status", "success", "chat_input_command", 125);

    const rendered = await metrics.register.metrics();

    assert.match(rendered, /comrade_bot_command_executions_total\{command="status",result="success",interaction_type="chat_input_command"\}\s+1/);
    assert.match(rendered, /comrade_bot_command_duration_seconds_bucket\{le="0\.25",command="status",result="success",interaction_type="chat_input_command"\}\s+1/);
});

test("interaction and discord lifecycle metrics are emitted", async () => {
    metrics.recordInteraction("button", "success");
    metrics.recordDiscordEvent("ready", "success");

    const rendered = await metrics.register.metrics();

    assert.match(rendered, /comrade_bot_interactions_total\{interaction_type="button",result="success"\}\s+1/);
    assert.match(rendered, /comrade_bot_discord_events_total\{event="ready",result="success"\}\s+1/);
});
