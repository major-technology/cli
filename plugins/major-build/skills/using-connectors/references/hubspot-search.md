# HubSpot CRM Search API Reference

**Endpoint:** `POST /crm/v3/objects/{objectType}/search`

## Request Body

All fields are optional. Send only what you need.

```json
{
	"filterGroups": [],
	"properties": ["prop1", "prop2"],
	"sorts": [{ "propertyName": "createdate", "direction": "DESCENDING" }],
	"limit": 20,
	"after": "20",
	"query": "search text"
}
```

---

## Filter Operators

| Operator             | Value field           | Behavior                                                     |
| -------------------- | --------------------- | ------------------------------------------------------------ |
| `EQ`                 | `value`               | Exact match (case-insensitive except enums)                  |
| `NEQ`                | `value`               | Not equal (case-insensitive except enums)                    |
| `LT`                 | `value`               | Less than                                                    |
| `LTE`                | `value`               | Less than or equal                                           |
| `GT`                 | `value`               | Greater than                                                 |
| `GTE`                | `value`               | Greater than or equal                                        |
| `BETWEEN`            | `value` + `highValue` | Range (inclusive)                                            |
| `IN`                 | `values` (array)      | Matches any in list. **String values must be lowercase.**    |
| `NOT_IN`             | `values` (array)      | Excludes all in list. **String values must be lowercase.**   |
| `HAS_PROPERTY`       | (none)                | Property has any value                                       |
| `NOT_HAS_PROPERTY`   | (none)                | Property is empty/null                                       |
| `CONTAINS_TOKEN`     | `value`               | Token match. Supports `*` wildcards for partial matching.    |
| `NOT_CONTAINS_TOKEN` | `value`               | Excludes token. Supports `*` wildcards for partial matching. |

**Common mistakes:**

- Using `value` instead of `values` (array) for `IN`/`NOT_IN`
- Forgetting `highValue` for `BETWEEN`
- Passing a `value` for `HAS_PROPERTY`/`NOT_HAS_PROPERTY` (they take none)
- Not lowercasing string values for `IN`/`NOT_IN`
- Using `CONTAINS_TOKEN` without wildcards for partial matching — `"smith"` matches the whole token `"smith"` only, NOT `"Blacksmith"`. Use `"*smith*"` for partial matches.

---

## FilterGroups Logic

- Filters **within** a filterGroup = **AND**
- Multiple **filterGroups** = **OR**

**Limits:**

- Max **5** filterGroups
- Max **6** filters per group
- Max **18** filters total across all groups
- Exceeding any limit returns `VALIDATION_ERROR`

```json
{
	"filterGroups": [
		{
			"filters": [
				{ "propertyName": "firstname", "operator": "EQ", "value": "Alice" },
				{ "propertyName": "city", "operator": "EQ", "value": "Boston" }
			]
		},
		{
			"filters": [{ "propertyName": "email", "operator": "CONTAINS_TOKEN", "value": "*@example.com" }]
		}
	]
}
```

This matches: (firstname=Alice AND city=Boston) OR (email contains @example.com)

---

## Case Sensitivity

| Context                              | Behavior                                    |
| ------------------------------------ | ------------------------------------------- |
| Enumeration (dropdown) properties    | **Always case-sensitive** for all operators |
| String properties with `IN`/`NOT_IN` | Values **must be lowercase**                |
| All other string filters             | Case-insensitive                            |

---

## Date/Timestamp Values

**CRITICAL: Use Unix epoch milliseconds as strings, NOT date strings.**

| Property Type                                            | Format                                             | Notes                                                                    |
| -------------------------------------------------------- | -------------------------------------------------- | ------------------------------------------------------------------------ |
| **datetime** (e.g., `createdate`, `hs_lastmodifieddate`) | Unix ms as string: `"1642672800000"`               | Any valid ms timestamp                                                   |
| **date-only** (no time component)                        | `YYYY-MM-DD` string OR Unix ms at **midnight UTC** | If using epoch ms, must be exactly midnight UTC or the date may be wrong |

```json
{
	"filterGroups": [
		{
			"filters": [
				{
					"propertyName": "hs_lastmodifieddate",
					"operator": "BETWEEN",
					"value": "1579514400000",
					"highValue": "1642672800000"
				}
			]
		}
	]
}
```

**In TypeScript (via Major resource client):**

```typescript
const value = new Date("2024-01-01").getTime().toString(); // "1704067200000"

const result = await hubspotClient.invoke("POST", "/crm/v3/objects/contacts/search", "search-contacts", {
	body: {
		filterGroups: [
			{
				filters: [
					{
						propertyName: "createdate",
						operator: "GTE",
						value: value,
					},
				],
			},
		],
		properties: ["firstname", "lastname", "email"],
	},
});
```

---

## Sorting

- Only **1** sort rule per request
- `direction`: `ASCENDING` or `DESCENDING`
- Default (no sort): ordered by creation date, oldest first

```json
{ "sorts": [{ "propertyName": "createdate", "direction": "DESCENDING" }] }
```

---

## Pagination

- Default page size: **10**
- Max page size: **200**
- Max total results: **10,000** (paging beyond this returns 400)
- Use `paging.next.after` from the response as the `after` parameter for the next page
- When `paging.next.after` is absent, there are no more results

---

## Searching by Associations

Use the pseudo-property `associations.{objectType}`:

```json
{ "propertyName": "associations.contact", "operator": "EQ", "value": "123" }
```

**Limitation:** Association searching is NOT supported for custom objects via search endpoints.

---

## Searchable Objects

Contacts, companies, deals, tickets, products, quotes, line items, orders, invoices, carts, leads, discounts, fees, taxes, deal splits, feedback submissions, payments, subscriptions, and custom objects.

**Engagement objects:** calls, emails, meetings, notes, tasks.

### Default Searchable Properties (for `query` parameter)

| Object    | Searchable Properties                                                                              |
| --------- | -------------------------------------------------------------------------------------------------- |
| Contacts  | `firstname`, `lastname`, `email`, `phone`, `hs_additional_emails`, `fax`, `mobilephone`, `company` |
| Companies | `website`, `phone`, `name`, `domain`                                                               |
| Deals     | `dealname`, `pipeline`, `dealstage`, `description`, `dealtype`                                     |
| Tickets   | `subject`, `content`, `hs_pipeline_stage`, `hs_ticket_category`, `hs_ticket_id`                    |
| Products  | `name`, `description`, `price`, `hs_sku`                                                           |

---

## Limits & Gotchas

- **Rate limit**: 5 requests/second per account for search (stricter than general API)
- **Request body max**: 3,000 characters
- **Newly created/updated objects** may take a few seconds to appear in search results
- **Archived objects** never appear in results
- **Cannot filter** engagement objects by `hs_body_preview_html`; emails also cannot filter by `hs_email_html` or `hs_body_preview`
- **Phone numbers** are normalized — omit country code when searching
- **Property names are case-sensitive** — use exact internal names from the CRM schema
