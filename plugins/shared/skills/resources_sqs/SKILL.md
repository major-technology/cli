---

name: using-sqs-connector

description: Implements AWS SQS message queue operations for sending, receiving, and managing messages using generated clients and MCP tools. Use when doing ANYTHING that touches SQS in any way, load this skill.

---

# Major Platform Resource: AWS SQS

## Common: Interacting with Resources

**Security**: Never connect directly to databases/APIs. Never use credentials in code. Always use generated clients or MCP tools.

**Two ways to interact with resources:**

1. **MCP tools** (direct, no code needed): Tools follow the pattern `mcp__resources__<resourcetype>_<toolname>`. Use `mcp__resources__list_resources` to discover available resources and their IDs.

2. **Generated TypeScript clients** (for app code): Call `mcp__resource-tools__add-resource-client` with a `resourceId` to generate a typed client. Clients are created in `/clients/` (Next.js) or `/src/clients/` (Vite).

**CRITICAL: Do NOT guess client method names or signatures.** The TypeScript clients in `@major-tech/resource-client` have strongly typed inputs and outputs. ALWAYS read the actual client source code in the generated `/clients/` directory (or the package itself) to verify available methods and their exact signatures before writing any client code.

**Framework note**: Next.js = resource clients must be used in server-side code only (Server Components, Server Actions, API Routes). Vite = call directly from frontend.

**Error handling**: Always check `result.ok` before accessing `result.result`.

**Invocation keys must be static strings** -- use descriptive literals like `"send-order-notification"`, never dynamic values like `` `${queueName}-send` ``.

---

## MCP Tools

- `mcp__resources__sqs_list_queues` -- List all SQS queues. Args: `resourceId`, `queueNamePrefix?`, `maxResults?`

- `mcp__resources__sqs_get_queue_attributes` -- Get queue attributes (message count, delay, etc). Args: `resourceId`, `queueUrl`

- `mcp__resources__sqs_send_message` -- Send a message to a queue. Args: `resourceId`, `queueUrl`, `messageBody`, `delaySeconds?`, `messageGroupId?`, `messageDeduplicationId?`

- `mcp__resources__sqs_receive_message` -- Receive messages from a queue. Args: `resourceId`, `queueUrl`, `maxNumberOfMessages?`, `waitTimeSeconds?`, `visibilityTimeout?`

- `mcp__resources__sqs_delete_message` -- Delete a processed message. Args: `resourceId`, `queueUrl`, `receiptHandle`

- `mcp__resources__sqs_invoke` -- Generic command execution. Args: `resourceId`, `command`, `queueUrl?`, `params?`

## TypeScript Client

```typescript

import { sqsClient } from "./clients";

// invoke(command, params, invocationKey, options?)

const result = await sqsClient.invoke(

	"SendMessage",

	{

		queueUrl: "[https://sqs.us-east-1.amazonaws.com/123456789012/my-queue](https://sqs.us-east-1.amazonaws.com/123456789012/my-queue)",

		messageBody: JSON.stringify({ orderId: "12345" }),

	},

	"send-order-notification",

);

if (result.ok) {

	console.log("Message ID:", [result.result.data](http://result.result.data).MessageId);

}

```

## Tips

- **Queue URLs, not names**: Most SQS operations require the full queue URL, not just the queue name. Use `sqs_list_queues` first to get URLs.

- **Receipt handles for deletion**: After receiving a message, use the `receiptHandle` from the response to delete it. Receipt handles expire after the visibility timeout.

- **FIFO queues**: If the queue URL ends in `.fifo`, you must provide `messageGroupId` and `messageDeduplicationId` when sending messages.

- **Visibility timeout**: When you receive a message, it becomes invisible to other consumers for the visibility timeout period. Process and delete it within this window, or it will reappear.

- **Long polling**: Set `waitTimeSeconds` (up to 20) on receive to reduce empty responses and API costs.

- **Message size limit**: SQS messages can be up to 256 KB. For larger payloads, store the data in S3 and send a reference.

- **Batch operations**: Use `sqs_invoke` with `DeleteMessageBatch` command to delete up to 10 messages at once.

- **PurgeQueue is not available**: For safety, bulk queue purging is not exposed through the platform. Delete messages individually or in batches instead.