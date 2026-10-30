import { createRequire } from "node:module";
import { readFileSync } from "node:fs";
import { homedir } from "node:os";

const fe = `${homedir()}/work/cleverfarm/repos/cf-farm-frontend`;
const { chromium } = createRequire(`${fe}/package.json`)("@playwright/test");

const read = (file) => {
  try {
    return readFileSync(`${fe}/${file}`, "utf8");
  } catch {
    return "";
  }
};
const pick = (text, key) => text.match(new RegExp(`^${key}=["']?([^"'\\n]*)`, "m"))?.[1];

const target = pick(read("env.example"), "TARGET") ?? "nightly";
const appUrl = {
  nightly: "https://nightly.dev.cleverfarm.cz",
  staging: "https://staging.dev.cleverfarm.cz",
  prod: "https://app.cleverfarm.cz",
}[target] ?? target;
const username = pick(read(".env"), "FE_AUTOMATION_EMAIL") ?? pick(read("env.example"), "FE_AUTOMATION_EMAIL");
const password = pick(read(".env"), "FE_AUTOMATION_PSW");
if (!username || !password) {
  console.error("FE_AUTOMATION_EMAIL (env.example) or FE_AUTOMATION_PSW (.env) missing in cf-farm-frontend");
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
await page.goto(`${appUrl}/login`);
await page.fill('input[name="username"]', username);
await page.fill('input[name="password"]', password);
await page.getByTestId("login-submit").click();
const token = await Promise.race([bearer, new Promise((_, rej) => setTimeout(() => rej(new Error("no Bearer request in 60s")), 60000))]);
await browser.close();
console.log(token);
