---
name: using-connectors
description: Use for ANY operation against a connected resource or connector — databases, CRMs, email, chat, storage, ads platforms, custom REST/GraphQL APIs, remote MCP servers — and for Major platform resources (managed database, managed file storage, app auth, AI proxy, HTTP proxy). Covers discovering resources, reading their context docs, and the per-connector reference that documents each connector's tools, clients, and API. Load before doing any work that involves a resource.
---

# Using connectors

The organization's connectors are _resources_ — external services (databases, APIs, storage, etc.) the agent and its apps reach through Major's secure clients and MCP tools. Every connector is used the same way: find the resource, read its context docs, then follow that connector's reference in this skill.

## Step 1: List available resources

Call `mcp__resources__list_resources` to get the resources you have access to, with their `resourceId`, type, and `subtype`.

## Step 2: Check for context documents before doing any work

**Always call `mcp__resources__list_resource_context` for every resource before doing any other work with it.** Resources often have context documents attached (API docs, schema references, usage guides) that tell you exactly how to use them.

For each resource you plan to use:

1. Call `mcp__resources__list_resource_context` with the `resourceId` immediately after listing resources.
2. **If documents exist, you MUST read them before doing anything else with the resource.** Do NOT query the resource directly until you have read the relevant context documents — the user attached them specifically to guide how you use it. `mcp__resources__read_resource_context` returns a download link for a document; fetch and read it.
3. If a context document contains schema or API information, use it directly — do not make redundant queries (e.g. do not run `\d` table commands if the schema is already in the context doc).
4. Tell the user which context documents you read and what you learned, so they know their context is being used.

## Step 3: Read the connector's reference

Find the connector's row below and read its reference file (in this skill's `references/` directory) **before** writing any query, tool call, or client code. Match on the resource's `subtype` (or, for Pipedream-backed connectors, its app slug), comparing case-insensitively and ignoring `-` and `_`. Each reference documents the connector's MCP tools (called through `mcp__major__execute_resource_tool`), its generated clients, and the underlying API.

### Connectors

| Connector | Subtype | Reference | Covers |
| --- | --- | --- | --- |
| Attio | `attio` | [attio.md](references/attio.md) | Implements Attio CRM data access for people, companies, lists, notes, and tasks using generated clients and MCP tools. |
| BigQuery | `bigquery` | [bigquery.md](references/bigquery.md) | Implements BigQuery dataset exploration, SQL queries, and table operations using generated clients and MCP tools. |
| Clerk Backend API | `clerk` | [clerk.md](references/clerk.md) | Implements Clerk Backend API requests with automatic Bearer Token auth using generated clients and MCP tools. |
| ClickHouse | `clickhouse` | [clickhouse.md](references/clickhouse.md) | Implements ClickHouse database connections, SQL queries, and data operations using generated clients and MCP tools. |
| Azure CosmosDB | `cosmosdb` | [cosmosdb.md](references/cosmosdb.md) | Implements Azure CosmosDB container queries, CRUD, and patch operations using generated clients and MCP tools. |
| Custom REST API | `custom` | [custom-api.md](references/custom-api.md) | Implements custom REST API HTTP requests with automatic auth header injection using generated clients and MCP tools. |
| DynamoDB | `dynamodb` | [dynamodb.md](references/dynamodb.md) | Implements DynamoDB queries, scans, and CRUD operations using generated clients and MCP tools. |
| Fireflies | `fireflies` | [fireflies.md](references/fireflies.md) | Implements Fireflies AI meeting transcription API access for transcripts, users, summaries, and audio upload using generated clients and MCP tools. |
| GitHub | `github` | [github.md](references/github.md) | Implements GitHub repository, issue, pull request, release, content, branch, and authenticated git operations using the GitHub connector, mounted MCP tools, generated clients, and HTTP proxy. |
| Gmail | `gmail` | [gmail.md](references/gmail.md) | Implements Gmail email reading, searching, and sending using generated clients and MCP tools. |
| Google Analytics | `google-analytics` | [google-analytics.md](references/google-analytics.md) | Implements Google Analytics (GA4) reporting, metadata exploration, and account management using generated clients and MCP tools. |
| Google Calendar | `googlecalendar` | [googlecalendar.md](references/googlecalendar.md) | Implements Google Calendar event management and scheduling using generated clients and MCP tools. |
| Google Drive | `googledrive` | [googledrive.md](references/googledrive.md) | Implements Google Drive file listing, reading, and management using generated clients and MCP tools. |
| Google Search Console | `googlesearchconsole` | [googlesearchconsole.md](references/googlesearchconsole.md) | Implements Google Search Console data access for search analytics, sitemaps, sites, and URL inspection using generated clients and MCP tools. |
| Google Sheets | `googlesheets` | [googlesheets.md](references/googlesheets.md) | Implements Google Sheets reading, writing, formatting, and batch operations using generated clients and MCP tools. |
| GraphQL API | `graphql` | [graphql.md](references/graphql.md) | Executes GraphQL queries and mutations against a configured endpoint using generated clients and MCP tools. |
| HubSpot | `hubspot` | [hubspot.md](references/hubspot.md) | Implements HubSpot CRM data access for contacts, companies, and deals using generated clients and MCP tools. |
| AWS Lambda | `lambda` | [lambda.md](references/lambda.md) | Implements AWS Lambda function invocation and management using generated clients and MCP tools. |
| LinkedIn Marketing API | `linkedin` | [linkedin.md](references/linkedin.md) | Implements LinkedIn Marketing API access for ad accounts, campaigns, creatives, and ad analytics using generated clients and MCP tools. |
| LinkedIn Marketing API | `linkedin` | [linkedinads.md](references/linkedinads.md) | Implements LinkedIn Marketing API data access for ad accounts, campaigns, creatives, and analytics using generated clients and MCP tools. |
| Custom MCP Connector (BYO remote MCP server) | `mcp_custom` | [mcp_custom.md](references/mcp_custom.md) | Implements runtime tool calls to a custom (bring-your-own) remote MCP server connector, using the in-session MCP tools for exploration and the generic createMcpClient for app code. |
| Meta Marketing | `metamarketing` | [metamarketing.md](references/metamarketing.md) | Implements Meta (Facebook) Marketing API access for campaigns, ads, insights, and lead forms using generated clients and MCP tools. |
| Microsoft SQL Server | `mssql` | [mssql.md](references/mssql.md) | Implements Microsoft SQL Server connections, queries, and schema exploration using generated clients and MCP tools. |
| MySQL | `mysql` | [mysql.md](references/mysql.md) | Implements MySQL database connections, SQL queries, and data operations using generated clients and MCP tools. |
| Neo4j | `neo4j` | [neo4j.md](references/neo4j.md) | Implements Neo4j Cypher queries, graph traversal, and node/relationship operations using generated clients and MCP tools. |
| Notion | `notion` | [notion.md](references/notion.md) | Implements Notion API interactions for pages, databases, blocks, users, and search using generated clients and MCP tools. |
| Outreach | `outreach` | [outreach.md](references/outreach.md) | Implements Outreach prospect and sequence management using generated clients and MCP tools. |
| PostgreSQL | `postgresql` | [postgresql.md](references/postgresql.md) | Implements PostgreSQL connections, SQL queries, and migration patterns using generated clients and MCP tools. |
| QuickBooks Online | `quickbooks` | [quickbooks.md](references/quickbooks.md) | Implements QuickBooks Online accounting data access for customers, invoices, items, accounts, vendors, bills, and payments using generated clients and MCP tools. |
| RingCentral | `ringcentral` | [ringcentral.md](references/ringcentral.md) | Implements RingCentral API access (call logs, messages, SMS, extensions) through the Major HTTP proxy. |
| Amazon S3 | `s3` | [s3.md](references/s3.md) | Implements Amazon S3 object operations, presigned URLs, and file uploads/downloads using generated clients and MCP tools. |
| Salesforce | `salesforce` | [salesforce.md](references/salesforce.md) | Implements Salesforce SOQL queries, sObject CRUD, and metadata exploration using generated clients and MCP tools. |
| SharePoint | `sharepoint` | [sharepoint.md](references/sharepoint.md) | Implements Microsoft SharePoint access — sites, lists, document libraries, and file operations — using generated clients and MCP tools. |
| Slack | `slack` | [slack.md](references/slack.md) | Implements Slack messaging, channel operations, and Web API calls using generated clients and MCP tools. |
| Snowflake | `snowflake` | [snowflake.md](references/snowflake.md) | Implements Snowflake warehouse queries, schema exploration, and data operations using generated clients and MCP tools. |
| AWS SQS | `sqs` | [sqs.md](references/sqs.md) | Implements AWS SQS message queue operations for sending, receiving, and managing messages using generated clients and MCP tools. |
| Stripe | `stripe` | [stripe.md](references/stripe.md) | Implements Stripe payment API access for customers, payments, subscriptions, invoices, and balance using generated clients and MCP tools. |
| TikTok Marketing API | `tiktokads` | [tiktokads.md](references/tiktokads.md) | Implements TikTok Marketing API access for ad accounts, campaigns, and ad reports using generated clients and MCP tools. |
| Zendesk | `zendesk` | [zendesk.md](references/zendesk.md) | Implements Zendesk Support API access (tickets, search, users, comments) through the Major HTTP proxy. |

### Platform resources

| Resource | Subtype | Reference | Covers |
| --- | --- | --- | --- |
| Major AI Proxy | `ai-proxy` | [ai-proxy.md](references/ai-proxy.md) | Use when the user asks to add AI features, LLM calls, or chat functionality to their app. |
| HTTP Proxy | `http-proxy` | [http-proxy.md](references/http-proxy.md) | Implements drop-in HTTP proxy access to any connected resource (Stripe, HubSpot, Slack, Gmail, etc.) via the SDK fetch wrapper or generic MCP tools. |
| Major Auth Connector | `majorauth` | [majorauth.md](references/majorauth.md) | Share or revoke application access for users by email. |
| Managed Databases | `managed-database` | [managed-database.md](references/managed-database.md) | Set up and use Major-managed PostgreSQL databases. |
| Managed File Storage | `managed-file-storage` | [managed-file-storage.md](references/managed-file-storage.md) | Set up and use Major-managed file storage (object/blob storage) for app uploads, downloads, images, documents, and attachments. |

Anything with a plain HTTP API (`proxy.compatible: true` in `list_resources`) is also reachable through the generic HTTP proxy — see [http-proxy.md](references/http-proxy.md).
