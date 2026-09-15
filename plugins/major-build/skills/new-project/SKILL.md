---
name: new-project
description: Use this skill when the user is building a brand new app for the first time. Guides connector setup, data layer understanding, and initial planning.
---

You are helping a user build the first iteration of an app. You are given a starter template Next.js application that you will flush out to build the app.
The template's structure and conventions — what you need to know to build on it — are provided to you as context.

You will need to perform these steps before building:

1. Figure out the connectors that the user wants to use:

- There are MCPs available for you to list their available connectors. IF, their connector is not yet connected, use `AskUserQuestion` to figure out which connectors
  they want to integrate. Then use the MCPs to connect the connectors that. Do not proceed until all connectors are connected or they explicitly say they want to use mock data.

2. Deeply understand how the user wants to use those connectors:

- Use the memories and MCPs available to explore and understand the connectors
- Use the `AskUserQuestion` tool to clarify anything that is unclear or ambiguous. Do not assume — always confirm with the user. Examples of good questions:
  - Is the start date for a cohort based on `created_at` on the User's table in the "Main DB"?
  - Which records should be pulled, leads or contacts?
  - Which fields do you want on the onboarding table?
    - Just the basics (name, title, company)
    - Everything (name, title, company, headcount, industry, size)
- Do not proceed until the user is satisfied with the data layer. There should be no surprises about where data comes from in the finished app.

3. Put together a plan for the app:

- Before writing the plan, use `AskUserQuestion` to confirm key product decisions. Examples:
  - What layout style? (sidebar nav, top nav, single page, etc.)
  - What are the key pages or views?
  - What features are highest priority for the first version?
- The plan should look more like a product spec than a technical plan. It should be readable by a non-technical user.
