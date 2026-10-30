import { createRequire } from "node:module";
import { readFileSync } from "node:fs";
import { homedir } from "node:os";

const fe = `${homedir()}/work/cleverfarm/repos/cf-farm-frontend`;
const { chromium } = createRequire(`${fe}/package.json`)("@playwright/test");
const password = readFileSync(`${fe}/.env`, "utf8").match(/^FE_AUTOMATION_PSW=["']?([^"'\n]*)/m)?.[1];
if (!password) {
  console.error("FE_AUTOMATION_PSW missing in cf-farm-frontend/.env");
  process.exit(1);
}

const browser = await chromium.launch();
const page = await browser.newPage();
const bearer = new Promise((resolve) =>
  page.on("request", (r) => {
    const h = r.headers().authorization;
    if (h?.startsWith("Bearer ")) resolve(h.slice(7));
  }),
);
await page.goto("https://nightly.dev.cleverfarm.cz/login");
await page.fill('input[name="username"]', "fe-automation@cleverfarm.cz");
await page.fill('input[name="password"]', password);
await page.getByTestId("login-submit").click();
const token = await Promise.race([bearer, new Promise((_, rej) => setTimeout(() => rej(new Error("no Bearer request in 60s")), 60000))]);
await browser.close();
console.log(token);
