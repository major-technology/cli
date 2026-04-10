---
name: resources
description: >
  How resources work on the Major platform -- two access modes (MCP tools and resource clients),
  security rules, write operation safety, and a directory of all resource-specific skills.
  Load this skill when working with any resource or when you need to understand the resource model.
---

# Resources

Resources are external services (databases, APIs, storage) connected to Major apps through the platform's secure proxy. They are managed at the organization level and attached to individual apps.

## Discovery

Before working with any resource, discover what's available. Load the `resources_list-resources` skill for the full discovery workflow, including reading context documents the user has attached.

Quick start: call `mcp__resources__list_resources` to list all resources the app has access to.

## Two Access Modes

### 1. MCP Tools (direct, no code needed)

Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use these for ad-hoc exploration, queries, and operations during your session.

Example: `mcp__resources__postgresql_query`, `mcp__resources__s3_list_objects`

### 2. Resource Clients (for app code)

Each resource attached to an app has a typed client that your application code uses to communicate with it. These clients handle authentication and connection details automatically -- you never use credentials directly in code. Load the resource-specific skill (see directory below) for client usage details, method signatures, and examples.

## Rules

**Security**:
- Never connect directly to databases or APIs. Never use credentials in code.
- Always use generated clients or MCP tools.

**Client usage**:
- **Do NOT guess client method names or signatures.** Always read the client source to verify available methods and their exact signatures.
- Always check `result.ok` before accessing `result.result`.
- Invocation keys must be static strings (e.g., `"fetch-user-orders"`), never dynamic values.

**Write operation safety**:
- Write operations (create, update, delete records; send messages; run mutations) can have real-world consequences.
- Always confirm with the user before performing write operations on production resources.
- For destructive operations (DELETE, DROP, truncate), require explicit user confirmation and state what will be affected.
- Prefer read-only exploration first (list, describe, query) before any writes.

## Resource Skills Directory

Load the specific skill for the resource type you're working with:

| Skill | Resource | Description |
|-------|----------|-------------|
| `resources_postgresql` | PostgreSQL | SQL queries, migrations, psql |
| `resources_mssql` | SQL Server | T-SQL queries, schema exploration |
| `resources_snowflake` | Snowflake | Warehouse queries, schema exploration |
| `resources_bigquery` | BigQuery | SQL queries, dataset/table operations |
| `resources_neo4j` | Neo4j | Cypher queries, graph traversal |
| `resources_dynamodb` | DynamoDB | Queries, scans, CRUD operations |
| `resources_cosmosdb` | CosmosDB | Container queries, CRUD, patch operations |
| `resources_managed-database` | Managed DB | Major-hosted PostgreSQL setup |
| `resources_s3` | Amazon S3 | Object operations, presigned URLs, uploads |
| `resources_salesforce` | Salesforce | SOQL queries, sObject CRUD |
| `resources_hubspot` | HubSpot | Contacts, companies, deals |
| `resources_quickbooks` | QuickBooks | Invoices, customers, accounts |
| `resources_outreach` | Outreach | Prospects, sequences |
| `resources_slack` | Slack | Messaging, channel operations |
| `resources_googlesheets` | Google Sheets | Reading, writing, formatting |
| `resources_google-analytics` | Google Analytics | GA4 reports, account management |
| `resources_googlecalendar` | Google Calendar | Events, calendar management |
| `resources_lambda` | AWS Lambda | Function invocation, management |
| `resources_graphql` | GraphQL | Queries, mutations, introspection |
| `resources_custom-api` | Custom API | HTTP requests with auto auth |
| `resources_ai-proxy` | AI Proxy | Built-in LLM proxy (Anthropic/OpenAI) |
| `resources_majorauth` | Major Auth | Share/revoke app access by email |
| `resources_list-resources` | Discovery | List resources, read context docs |
