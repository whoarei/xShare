import { readFileSync, writeFileSync } from "fs";
import { resolve, dirname } from "path";
import { fileURLToPath } from "url";
import { marked } from "marked";

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = resolve(__dirname, "..");

const mdPath = resolve(root, "CHANGELOG.md");
const outPath = resolve(root, "src-tauri", "resources", "CHANGELOG.html");

const md = readFileSync(mdPath, "utf8");
const body = marked.parse(md);

const html = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<title>xShare - 更新日志</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    color: #1f2937;
    line-height: 1.6;
    padding: 24px;
    background: #fff;
  }
  h1 { font-size: 1.5rem; margin-bottom: 16px; color: #111827; }
  h2 { font-size: 1.25rem; margin: 24px 0 12px; color: #111827; border-bottom: 1px solid #e5e7eb; padding-bottom: 6px; }
  h3 { font-size: 1rem; margin: 16px 0 8px; color: #374151; }
  ul { padding-left: 20px; margin: 8px 0; }
  li { margin: 4px 0; font-size: 0.875rem; }
  code { font-family: "Cascadia Code", "Fira Code", Consolas, monospace; background: #f3f4f6; padding: 1px 4px; border-radius: 3px; font-size: 0.85em; }
  a { color: #2563eb; text-decoration: none; }
  a:hover { text-decoration: underline; }
  strong { font-weight: 600; }
  em { font-style: normal; color: #6b7280; }
</style>
</head>
<body>
${body}
</body>
</html>`;

writeFileSync(outPath, html, "utf8");
console.log(`[OK] CHANGELOG.html → ${outPath}`);
