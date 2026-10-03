import { test, expect, Page } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

async function login(page: Page, role = "senior") {
  await page.goto("/login");
  await page
    .getByLabel("Email", { exact: true })
    .fill(`${role}@northwind.demo`);
  await page.getByRole("button", { name: "Sign in to workspace" }).click();
  await expect(page).toHaveURL(/\/workspace$/);
  await expect(page.locator(".case-title").first()).toBeVisible();
}
async function accessible(page: Page) {
  const result = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21aa"])
    .analyze();
  expect(
    result.violations.map((v) => ({
      id: v.id,
      description: v.description,
      nodes: v.nodes.map((n) => n.target),
    })),
  ).toEqual([]);
}
async function fits(page: Page) {
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBeTruthy();
}

test("landing navigation, photography, disclosure, and responsive accessibility", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (err) => errors.push(err.message));
  await page.goto("/");
  await expect(page.getByRole("heading", { level: 1 })).toContainText(
    "Every signal",
  );
  expect(
    await page
      .locator(".hero-photo img")
      .evaluate(
        (img: HTMLImageElement) => img.complete && img.naturalWidth > 0,
      ),
  ).toBeTruthy();
  for (let index = 0; index < 3; index++) {
    await page.locator(".preview-switcher button").nth(index).click();
    await expect(
      page.locator(".preview-switcher button").nth(index),
    ).toHaveAttribute("aria-pressed", "true");
    await page
      .locator(".product-frame img")
      .evaluate((img: HTMLImageElement) => img.decode());
  }
  await page.locator(".preview-switcher button").first().click();
  await page.locator('nav a[href="#platform"]').click();
  await expect(page).toHaveURL(/#platform$/);
  await page.getByText("AI has a defined role", { exact: false }).click();
  await expect(
    page.getByText("The assistant helps review case evidence", {
      exact: false,
    }),
  ).toBeVisible();
  await accessible(page);
  for (const width of [390, 768, 1440]) {
    await page.setViewportSize({ width, height: 900 });
    await fits(page);
  }
  await page.setViewportSize({ width: 390, height: 844 });
  await page.getByRole("button", { name: "Menu" }).click();
  await page.getByRole("link", { name: "How it works", exact: true }).click();
  await expect(page).toHaveURL(/#workflow$/);
  await expect(page.getByRole("button", { name: "Menu" })).toHaveAttribute(
    "aria-expanded",
    "false",
  );
  await page
    .getByRole("link", { name: "Explore the workspace", exact: true })
    .click();
  await expect(page).toHaveURL(/\/login$/);
  await accessible(page);
  await fits(page);
  expect(errors).toEqual([]);
});

test("login error, role selection, filters, refresh, logout, and route protection", async ({
  page,
}) => {
  await page.goto("/workspace");
  await expect(page).toHaveURL(/\/login$/);
  await page.getByLabel("Password", { exact: true }).fill("incorrect-password");
  await page.getByRole("button", { name: "Sign in to workspace" }).click();
  await expect(page.getByRole("alert")).toBeVisible();
  await page
    .getByRole("button", { name: "Senior investigator", exact: true })
    .click();
  await expect(page.getByLabel("Email", { exact: true })).toHaveValue(
    "senior@northwind.demo",
  );
  await page.getByRole("button", { name: "Sign in to workspace" }).click();
  await expect(page.locator(".case-title").first()).toBeVisible();
  await page
    .getByRole("button", { name: "Filter open cases", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Open", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  await page
    .getByRole("button", { name: "Filter open cases", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "All cases", exact: true }),
  ).toHaveAttribute("aria-pressed", "true");
  const count = await page.locator(".case-title").count();
  expect(count).toBeGreaterThan(0);
  await page
    .getByRole("textbox", { name: "Search cases" })
    .fill("no-matching-account");
  await expect(
    page.getByRole("heading", { name: "No matching cases" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Clear filters" }).click();
  await expect(page.locator(".case-title")).toHaveCount(count);
  await page
    .getByRole("combobox", { name: "Filter by priority" })
    .selectOption("high");
  for (const priority of await page
    .locator(".case-table tbody .priority")
    .allTextContents())
    expect(priority).toBe("high");
  await page
    .getByRole("combobox", { name: "Filter by priority" })
    .selectOption("all");
  await page.getByRole("button", { name: "Open", exact: true }).click();
  for (const status of await page
    .locator(".case-table tbody .tag")
    .allTextContents())
    expect(status).toBe("open");
  await page.getByRole("button", { name: "All cases", exact: true }).click();
  await page.getByRole("button", { name: "Refresh queue" }).click();
  await expect(
    page.getByRole("button", { name: "Refresh queue" }),
  ).toBeEnabled();
  await accessible(page);
  for (const width of [390, 768, 1440]) {
    await page.setViewportSize({ width, height: 900 });
    await fits(page);
  }
  await page.getByRole("button", { name: "Log out", exact: true }).click();
  await expect(page).toHaveURL(/\/login$/);
  await page.goto("/workspace");
  await expect(page).toHaveURL(/\/login$/);
});

test("evidence, transactions, graph, grounded report, assistant, notes, lifecycle, and audit", async ({
  page,
}) => {
  await login(page);
  await page
    .getByRole("link", { name: "Review activity on acct_demo", exact: false })
    .first()
    .click();
  await expect(
    page.getByRole("heading", { name: "Detected signals" }),
  ).toBeVisible();
  await expect(page.locator("#transactions tbody tr").first()).toBeVisible();
  const citation = page.locator("#signals .chip").first();
  const evidenceID = await citation.innerText();
  await citation.click();
  await expect(page.locator(".evidence article.target")).toHaveAttribute(
    "id",
    evidenceID,
  );
  const graph = await (
    await page.request.get(`/api/v1/cases/${page.url().split("/").pop()}/graph`)
  ).json();
  expect(await page.locator(".graph circle").count()).toBe(graph.nodes.length);
  await page.getByRole("button", { name: "Draft from evidence" }).click();
  await expect(page.locator("#report")).toContainText("Not a live model call");
  await expect(page.locator("#report .chip").first()).toBeVisible();
  await page
    .getByLabel("Question", { exact: true })
    .fill("What evidence is missing?");
  await page.getByRole("button", { name: "Ask assistant" }).click();
  await expect(page.locator("#assistant .claim").last()).toContainText(
    /evidence|missing|review/i,
  );
  await page
    .getByLabel("Question", { exact: true })
    .fill("Please freeze the account");
  await page.getByRole("button", { name: "Ask assistant" }).click();
  await expect(page.locator("#assistant .claim").last()).toContainText(
    "refused",
  );
  const note = `Browser verification ${Date.now()}: review supporting source records.`;
  await page.getByLabel("Add a note", { exact: true }).fill(note);
  await page.getByRole("button", { name: "Save note" }).click();
  await expect(page.locator("#notes")).toContainText(note);
  await page.getByRole("button", { name: "Assign to me", exact: true }).click();
  await expect(page.locator("main > .row .tag").first()).toHaveText(
    "in review",
  );
  await expect(
    page.getByRole("button", { name: "Start review", exact: true }),
  ).toHaveCount(0);
  await page
    .getByRole("button", { name: "Escalate case", exact: true })
    .click();
  await expect(page.getByRole("alert")).toContainText("20 characters");
  await page
    .getByLabel("Rationale", { exact: true })
    .fill("Escalating for senior review of the observed evidence.");
  await page
    .getByRole("button", { name: "Escalate case", exact: true })
    .click();
  await expect(page.locator("main > .row .tag").first()).toHaveText(
    "escalated",
  );
  await page
    .getByLabel("Rationale", { exact: true })
    .fill("Resolved after reviewing source records; no adverse action.");
  await page.getByRole("button", { name: "Resolve case", exact: true }).click();
  await expect(page.locator("main > .row .tag").first()).toHaveText("resolved");
  await expect(page.locator("#decision")).toContainText("no_adverse_action");
  await page
    .getByLabel("Rationale", { exact: true })
    .fill("Closing after the completed and documented human review.");
  await page.getByRole("button", { name: "Close case", exact: true }).click();
  await expect(page.locator("main > .row .tag").first()).toHaveText("closed");
  for (const action of [
    "case_assign",
    "case_escalate",
    "case_resolve",
    "case_close",
    "note_added",
  ])
    await expect(page.locator("#audit")).toContainText(action);
  await accessible(page);
  for (const width of [390, 768, 1440]) {
    await page.setViewportSize({ width, height: 900 });
    await fits(page);
  }
});

test("viewer is read-only and investigator cannot make senior decisions", async ({
  page,
}) => {
  await login(page, "viewer");
  await page.locator(".case-title").first().click();
  await expect(
    page.getByRole("heading", { name: "Evidence library" }),
  ).toBeVisible();
  for (const name of [
    "Draft from evidence",
    "Ask assistant",
    "Save note",
    "Escalate case",
    "Resolve case",
  ])
    await expect(page.getByRole("button", { name, exact: true })).toHaveCount(
      0,
    );
  const id = page.url().split("/").pop();
  const forbidden = await page.request.post(`/api/v1/cases/${id}/notes`, {
    headers: { "X-FinGuard-Request": "1" },
    data: { body: "Denied viewer note" },
  });
  expect(forbidden.status()).toBe(403);
  await page.getByRole("button", { name: "Log out", exact: true }).click();
  await login(page, "investigator");
  await page.getByRole("button", { name: "Open", exact: true }).click();
  await page.locator(".case-title").first().click();
  await expect(
    page.getByRole("button", { name: "Start review", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Escalate case", exact: true }),
  ).toHaveCount(0);
  await page.getByRole("button", { name: "Start review", exact: true }).click();
  await expect(page.locator("main > .row .tag").first()).toHaveText(
    "in review",
  );
});

test("queue API failures show a recoverable error rather than a false empty queue", async ({
  page,
}) => {
  await login(page);
  await page.route("**/api/v1/dashboard", (route) =>
    route.fulfill({
      status: 503,
      contentType: "application/json",
      body: JSON.stringify({ error: { message: "Temporarily unavailable" } }),
    }),
  );
  await page.reload();
  await expect(page.getByRole("alert")).toContainText(
    "Temporarily unavailable",
  );
  await expect(
    page.getByRole("heading", { name: "Your queue is clear" }),
  ).toHaveCount(0);
  await page.unroute("**/api/v1/dashboard");
  await page.getByRole("button", { name: "Try again" }).click();
  await expect(page.locator(".case-title").first()).toBeVisible();
});

test("additional queue pages append without duplicate cases", async ({
  page,
}) => {
  await login(page);
  const all = (await (await page.request.get("/api/v1/cases?limit=25")).json())
    .data;
  expect(all.length).toBeGreaterThan(3);
  await page.route("**/api/v1/cases?*", (route) => {
    const second = new URL(route.request().url()).searchParams.has("cursor");
    return route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        data: second ? [all[1], all[2], all[3]] : all.slice(0, 2),
        page: { next_cursor: second ? "" : "test-next-page", limit: 2 },
      }),
    });
  });
  await page.reload();
  await expect(page.locator(".case-title")).toHaveCount(2);
  await page
    .getByRole("button", { name: "Load more cases", exact: true })
    .click();
  await expect(page.locator(".case-title")).toHaveCount(4);
  await expect(
    page.getByRole("button", { name: "Load more cases", exact: true }),
  ).toHaveCount(0);
});

test("failed report generation stays explicit and leaves evidence usable", async ({
  page,
}) => {
  await login(page);
  await page.locator(".case-title").first().click();
  await expect(
    page.getByRole("heading", { name: "Evidence library" }),
  ).toBeVisible();
  await page.route("**/api/v1/cases/*/reports", async (route) => {
    if (route.request().method() !== "POST") return route.continue();
    return route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        status: "failed",
        provider: "local-stub",
        live: false,
      }),
    });
  });
  await page.getByRole("button", { name: "Draft from evidence" }).click();
  await expect(page.getByRole("alert")).toContainText("Report failed");
  await expect(
    page.getByText("Report draft updated.", { exact: false }),
  ).toHaveCount(0);
  await expect(page.locator(".evidence article").first()).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Draft from evidence" }),
  ).toBeEnabled();
});
