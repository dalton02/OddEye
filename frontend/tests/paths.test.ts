import { createResolver, parseRichText } from "../src/lib/paths";

const resolve = createResolver([
  "src/infra/repositories/prisma/withdrawal.ts",
  "src/use-cases/withdrawal/approve/index.ts",
  "src/use-cases/payment/approve/index.ts",
  "prisma/schema.prisma"
]);

const cases: [string, string | null][] = [
  ["src/infra/repositories/prisma/withdrawal.ts", "src/infra/repositories/prisma/withdrawal.ts"],
  ["@infra/repositories/prisma/withdrawal", "src/infra/repositories/prisma/withdrawal.ts"],
  ["@use-cases/withdrawal/approve", "src/use-cases/withdrawal/approve/index.ts"],
  ["withdrawal/approve/index.ts", "src/use-cases/withdrawal/approve/index.ts"],
  ["approve/index.ts", null],
  ["schema.prisma", "prisma/schema.prisma"],
  ["index.ts", null],
  ["ApproveWithdrawalUseCase", null],
  ["src/nao/existe.ts", null]
];

let failed = 0;
for (const [input, want] of cases) {
  const got = resolve(input);
  if (got !== want) {
    failed++;
    console.error(`resolve(${input}) = ${got}; want ${want}`);
  }
}
const parts = parseRichText("Chama `@infra/repositories/prisma/withdrawal` e `execute()`.", resolve);
if (parts[1].path !== "src/infra/repositories/prisma/withdrawal.ts" || parts[3].path !== null || !parts[3].code) {
  failed++;
  console.error("parseRichText", parts);
}
if (failed) process.exit(1);
console.log(`ok ${cases.length + 1} casos`);
