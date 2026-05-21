---
name: using-ai-proxy
description: Use when the user asks to add AI features, LLM calls, or chat functionality to their app. Covers Major's built-in AI proxy for Anthropic, OpenAI, and Gemini APIs.
---

## Major AI Proxy

Major provides a built-in AI proxy that lets apps call Anthropic, OpenAI, and Gemini APIs without configuring API keys. Usage is billed at cost against the user's credits.

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

### OpenAI — Image Generation

Three stateless endpoints. None persist images server-side — input image bytes only live in the request body.

```typescript
// Text → image
const image = await client.images.generate({
	model: "gpt-image-1",
	prompt: "A white siamese cat",
	n: 1,
	size: "1024x1024",
});

// Image + prompt → edited image
const edited = await client.images.edit({
	model: "gpt-image-1",
	image: fs.createReadStream("input.png"),
	prompt: "Add a top hat",
});

// Image → variations (DALL-E 2 only)
const variations = await client.images.createVariation({
	image: fs.createReadStream("input.png"),
	n: 2,
});
```

### Gemini

Use the official `@google/genai` SDK. Point its `baseUrl` at the proxy's `/genai` prefix — the SDK appends `/v1beta/models/...` itself.

```typescript
import { GoogleGenAI } from "@google/genai";

const ai = new GoogleGenAI({
	apiKey: process.env.MAJOR_JWT_TOKEN,
	httpOptions: {
		baseUrl: process.env.MAJOR_AI_PROXY_URL + "/genai",
	},
});

const response = await ai.models.generateContent({
	model: "gemini-2.5-pro",
	contents: "Hello!",
});
```

Streaming uses `ai.models.generateContentStream(...)` and counting tokens uses `ai.models.countTokens(...)` with the same client.

### Gemini — Image Generation

Two paths: an image-capable model via `generateContent` (output comes back as `inlineData` parts), or the dedicated `generateImages` endpoint for Imagen.

```typescript
// Gemini image models via generateContent
const response = await ai.models.generateContent({
	model: "gemini-2.5-flash-image",
	contents: "A picture of a banana dish in a fancy restaurant",
});

for (const part of response.candidates[0].content.parts) {
	if (part.inlineData) {
		const dataUrl = `data:image/png;base64,${part.inlineData.data}`;
	}
}

// Imagen via generateImages
const imagen = await ai.models.generateImages({
	model: "imagen-3.0-generate-002",
	prompt: "Robot holding a red skateboard",
	config: { numberOfImages: 1 },
});

const imageBytes = imagen.generatedImages?.[0]?.image?.imageBytes;
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

**Gemini:**

- `gemini-2.5-pro` (flagship, reasoning)
- `gemini-2.5-flash` (fast, mid-tier)
- `gemini-2.5-flash-lite` (cheapest)

**Image generation:**

- `gpt-image-1`, `gpt-image-2` (OpenAI, via `images.generate` / `images.edit`)
- `dall-e-2` (OpenAI, the only model that supports `images.createVariation`)
- `gemini-2.5-flash-image`, `gemini-3-pro-image-preview` (Gemini, via `generateContent`)
- `imagen-3.0-generate-002` (Gemini, via `generateImages`)

## Rules

- Always use the native provider SDK (`@anthropic-ai/sdk`, `openai`, or `@google/genai`) — never a unified SDK
- Set the base URL to `process.env.MAJOR_AI_PROXY_URL + "/<provider>"` (`/anthropic`, `/openai`, or `/genai`). For Gemini, this is the SDK's `httpOptions.baseUrl`; for Anthropic and OpenAI, it's `baseURL`.
- **Set `apiKey` to `process.env.MAJOR_JWT_TOKEN`** — this is required for authentication. Without it, all requests will be rejected.
- Never hardcode API keys or proxy URLs
- Only use models from the allowlist above

## Available Endpoints

Only these endpoints are available through the proxy:

- **Anthropic:** `/v1/messages` (chat)
- **OpenAI:** `/v1/chat/completions` (chat), `/v1/responses` (responses API), `/v1/audio/speech` (text-to-speech), `/v1/audio/transcriptions` (speech-to-text), `/v1/images/generations`, `/v1/images/edits`, `/v1/images/variations`
- **Gemini:** `/v1beta/models/{model}:generateContent`, `/v1beta/models/{model}:streamGenerateContent`, `/v1beta/models/{model}:countTokens`, `/v1beta/models/{model}:generateImages`

All image endpoints are stateless — input image bytes only live in the request body for that one call. There's no Files API, no Assistants/threads, and no server-side persistence.

## Limitations

- No embeddings or video generation
- Anthropic, OpenAI, and Gemini only
- Gemini: chat, streaming, token counting, and image generation only — embeddings, files, video, cached contents, tuning, and the Live API are not supported
- Monthly spending limits apply — requests are rejected if limit exceeded or wallet empty
- Specific models from the allowlist only
