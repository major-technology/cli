---
name: using-ai-proxy
description: Use when the user asks to add AI features, LLM calls, or chat functionality to their app. Covers Major's built-in AI proxy for Anthropic and OpenAI APIs.
---

## Major AI Proxy

Major provides a built-in AI proxy that lets apps call Anthropic and OpenAI APIs without configuring API keys. Usage is billed at cost against the user's credits.

## Workflow

1. Call `check_ai_proxy_status` to see if the proxy is enabled for this app
2. If enabled: use it directly with the env vars below
3. If not enabled: ask the user if they want to enable it (recommended) or use their own API keys
4. If user wants to enable it: call `enable_ai_proxy`

## MCP Tools

- `mcp__major-platform__check_ai_proxy_status` — Check if proxy is enabled and current month spend
- `mcp__major-platform__enable_ai_proxy` — Enable the proxy for the current app

## Environment Variables

These env vars are already available in the app's runtime environment:

- `MAJOR_AI_PROXY_URL` — Base URL for the AI proxy (set in `.env`)
- `MAJOR_JWT_TOKEN` — Authentication token (set as pod env var). **You MUST pass this as the `apiKey` when initializing the SDK client.** The proxy authenticates every request using this token — requests without it will be rejected.

They might not be available in your Bash, that's normal.

## Code Examples

### Anthropic

```typescript
import Anthropic from "@anthropic-ai/sdk";

const client = new Anthropic({
	baseURL: process.env.MAJOR_AI_PROXY_URL + "/anthropic",
	apiKey: process.env.MAJOR_JWT_TOKEN,
});

const message = await client.messages.create({
	model: "claude-sonnet-4-6",
	max_tokens: 1024,
	messages: [{ role: "user", content: "Hello!" }],
});
```

### OpenAI — Chat

```typescript
import OpenAI from "openai";

const client = new OpenAI({
	baseURL: process.env.MAJOR_AI_PROXY_URL + "/openai",
	apiKey: process.env.MAJOR_JWT_TOKEN,
});

const completion = await client.chat.completions.create({
	model: "gpt-4.1",
	messages: [{ role: "user", content: "Hello!" }],
});
```

### OpenAI — Text-to-Speech

```typescript
const speech = await client.audio.speech.create({
	model: "tts-1",
	voice: "alloy",
	input: "Hello, world!",
});
```

### OpenAI — Speech-to-Text

```typescript
const transcription = await client.audio.transcriptions.create({
	model: "whisper-1",
	file: audioFile,
});
```

## Available Models

**Anthropic:**

- `claude-opus-4-7` (flagship)
- `claude-sonnet-4-6` (mid-tier)
- `claude-haiku-4-5-20251001` (fast/cheap)

**OpenAI:**

- `gpt-5.4` (flagship)
- `gpt-4.1` (mid-tier, 1M context)
- `gpt-4.1-mini` (good value, 1M context)
- `gpt-4.1-nano` (cheapest)
- `o3` (reasoning)
- `o4-mini` (fast reasoning)

## Rules

- Always use the native provider SDK (Anthropic or OpenAI) — never a unified SDK
- Set `baseURL` to `process.env.MAJOR_AI_PROXY_URL + "/<provider>"` (e.g. `/anthropic` or `/openai`)
- **Set `apiKey` to `process.env.MAJOR_JWT_TOKEN`** — this is required for authentication. Without it, all requests will be rejected.
- Never hardcode API keys or proxy URLs
- Only use models from the allowlist above

## Available Endpoints

Only these endpoints are available through the proxy:

- **Anthropic:** `/v1/messages` (chat)
- **OpenAI:** `/v1/chat/completions` (chat), `/v1/responses` (responses API), `/v1/audio/speech` (text-to-speech), `/v1/audio/transcriptions` (speech-to-text)

## Limitations

- No embeddings or image generation
- Anthropic and OpenAI only
- Monthly spending limits apply — requests are rejected if limit exceeded or wallet empty
- Specific models from the allowlist only
