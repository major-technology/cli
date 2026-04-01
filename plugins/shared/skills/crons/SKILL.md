---
name: using-crons
description: Use when the user needs to run something on a cadence or do anything that interacts with crons. Creates, modifies and deletes cron jobs that run on a cadence.
---

# Cron Jobs

Scheduled jobs are configured via a `cron.json` file at the project root and handled by route endpoints in the application.

## cron.json

Create `cron.json` at the project root:

```json
{
	"version": "1",
	"crons": [
		{
			"name": "daily-cleanup",
			"description": "Removes expired sessions",
			"url": "/api/crons/cleanup",
			"schedule": "0 2 * * *"
		}
	]
}
```

Fields:

- `name` — unique identifier for the job
- `description` — human-readable explanation (used for documentation)
- `url` — the route path the scheduler will POST to
- `schedule` — 5-field cron expression (see below)

## Schedule Format

Use a 5-field cron expression. Do **not** include a seconds field.

```
┌─ minute (0–59)
│  ┌─ hour (0–23)
│  │  ┌─ day of month (1–31)
│  │  │  ┌─ month (1–12)
│  │  │  │  ┌─ day of week (0–7, 0 and 7 = Sunday)
│  │  │  │  │
*  *  *  *  *
```

Common schedules:

| Schedule               | Expression    |
| ---------------------- | ------------- |
| Every minute           | `* * * * *`   |
| Every 5 minutes        | `*/5 * * * *` |
| Every hour             | `0 * * * *`   |
| Every day at 2 AM      | `0 2 * * *`   |
| Every Monday 9 AM      | `0 9 * * 1`   |
| 1st of month, midnight | `0 0 1 * *`   |

## Notes

- You should tell the user after you've build the scheduled job that they need to deploy for the cron to take affect
- The user can see a list of all crons they have on the left hand side bar
