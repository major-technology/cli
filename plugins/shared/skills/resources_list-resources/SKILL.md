---
name: list-resources
description: Use this skill whenever you need to discover or work with resources (databases, APIs, storage, etc.) available to the application. Load this skill before doing any work that involves resources.
---

# Discovering and Using Resources

> For the complete resource interaction model (MCP tools, TypeScript clients, write safety, security rules), load the `resources` skill.

The application has been granted access to resources — external services (databases, APIs, storage, etc.) that your application can interact with through Major's secure clients.

## Step 1: List available resources

Call `mcp__resources__list_resources` to get all resources the application has access to.

## Step 2: Check for context documents before doing any work

**Always call `mcp__resources__list_resource_context` for every resource before doing any other work with it.** Resources often have context documents attached (API docs, schema references, usage guides) that tell you exactly how to use them.

For each resource you plan to use:

1. Call `mcp__resources__list_resource_context` with the `resourceId` immediately after listing resources
2. **If documents exist, you MUST read them before doing anything else with the resource.** Do NOT skip this step. Do NOT query the resource directly until you have read all relevant context documents. The user uploaded these documents specifically to guide how you use the resource.
3. For each relevant document, spawn the `file-reader` agent to download and read it:
   ```
   Task tool with subagent_type: "file-reader"
   Prompt: "Download and read this resource context document.
     resourceId: <id>, documentId: <id>, filename: <name from list>
     Extract: <what information you need for your current task>"
   ```
   The agent will download the file, read it, and return a summary plus the local file path.
4. If the context document contains schema or API information, use it directly — do not make redundant queries (e.g. do not run `\d` table commands if the schema is already in the context doc)
5. Tell the user which context documents you read and what you learned, so they know their context is being used
