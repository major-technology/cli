---
name: using-fireflies-connector
description: Implements Fireflies AI meeting transcription API access for transcripts, users, summaries, and audio upload using generated clients and MCP tools. Use when doing ANYTHING that touches Fireflies in any way, load this skill.
---

# Major Platform Resource: Fireflies

## Common: Interacting with Resources

**Security**: Never connect directly to APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Three ways to interact with Fireflies:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.
2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).
3. **HTTP proxy** (Next.js apps): Use `createProxyFetch` from `@major-tech/resource-client/next` to call the Fireflies API directly with automatic auth injection. See [using-http-proxy](../http-proxy/SKILL.md) for setup and usage — preferred when you need to hit endpoints not covered by MCP tools or the typed client, or when using an official SDK that accepts a custom `fetch`.

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `response.ok` before accessing `response.data`.

**Invocation keys must be static strings** — use descriptive literals like `"list-transcripts"`, never dynamic values like `` `${date}-transcripts` ``.

---

## MCP Tools

- `mcp__resources__fireflies_query` — Execute any GraphQL query against the Fireflies API. Supports introspection queries. Args: `resourceId`, `query`, `variables?`
- `mcp__resources__fireflies_mutate` — Execute any GraphQL mutation against the Fireflies API. Args: `resourceId`, `query`, `variables?`
- `mcp__resources__fireflies_list_transcripts` — List transcripts with optional filters. Args: `resourceId`, `keyword?`, `fromDate?`, `toDate?`, `limit?`, `skip?`, `hostEmail?`, `mine?`
- `mcp__resources__fireflies_get_transcript` — Get transcript by ID with summary, speakers, sentences, analytics. Args: `resourceId`, `transcriptId`
- `mcp__resources__fireflies_list_users` — List all team users. Args: `resourceId`
- `mcp__resources__fireflies_get_user` — Get a single user by ID. Args: `resourceId`, `userId`
- `mcp__resources__fireflies_upload_audio` — Upload audio from a public URL for transcription. Args: `resourceId`, `url`, `title?`, `attendees?`

## TypeScript Client

The client exposes `query<T>()` and `mutate<T>()` methods. The generic `T` types the parsed GraphQL data payload directly — no need to dig through result.body.kind / result.body.value / data.

### Reading data

```typescript
import { firefliesClient } from "./clients";

// List recent transcripts
const response = await firefliesClient.query<{
  transcripts: Array<{ id: string; title: string; date: number; duration: number }>;
}>(
  `query { transcripts(limit: 10) { id title date duration } }`,
  "list-transcripts"
);

if (response.ok) {
  for (const t of response.data.transcripts) {
    console.log(t.id, t.title);
  }
}

// Get a transcript with summary
const transcript = await firefliesClient.query<{
  transcript: {
    id: string; title: string; duration: number;
    summary: { overview: string; action_items: string; short_summary: string };
    speakers: Array<{ id: string; name: string }>;
  };
}>(
  `query($id: String!) {
    transcript(id: $id) {
      id title duration
      summary { overview action_items short_summary }
      speakers { id name }
    }
  }`,
  "get-transcript",
  { variables: { id: "transcript-id" } }
);

// List users
const users = await firefliesClient.query<{
  users: Array<{ user_id: string; email: string; name: string }>;
}>(
  `query { users { user_id email name } }`,
  "list-users"
);
```

### Writing data

```typescript
// Upload audio via mutation
const upload = await firefliesClient.mutate<{
  uploadAudio: { success: boolean; title: string; message: string };
}>(
  `mutation($input: AudioUploadInput) {
    uploadAudio(input: $input) { success title message }
  }`,
  "upload-audio",
  { variables: { input: { url: "https://...", title: "My Meeting" } } }
);

if (upload.ok) {
  console.log("Upload:", upload.data.uploadAudio.message);
}
```

### Search with variables

```typescript
const filtered = await firefliesClient.query<{
  transcripts: Array<{ id: string; title: string }>;
}>(
  `query($keyword: String, $limit: Int) {
    transcripts(keyword: $keyword, limit: $limit) { id title }
  }`,
  "search-transcripts",
  { variables: { keyword: "sales call", limit: 5 } }
);
```

## Tips

- **Rate limits**: Free/Pro = 50 req/day, Business/Enterprise = 60 req/min
- Only request fields you need — GraphQL precision reduces response size
- Transcripts `limit` max is 50 per query; use `skip` for pagination
- Audio upload only works with publicly accessible HTTPS URLs
- `audio_url` and `video_url` expire after 24 hours — re-query if needed
- Meeting `date` field is in epoch milliseconds (UTC)
- Common queries: `transcripts`, `transcript` (by ID), `users`, `user` (by ID)
- Common mutations: `uploadAudio`, `deleteTranscript`, `updateMeetingTitle`
- Introspection is supported — use `{ __schema { ... } }` or `{ __type(name: "Transcript") { ... } }` via the `query` tool to discover available fields

**Docs**: [Fireflies API Reference](https://docs.fireflies.ai)
