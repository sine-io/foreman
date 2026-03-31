const test = require('node:test');
const assert = require('node:assert/strict');

const renderers = require('./artifact-renderers.js');

const { renderPreview } = renderers;

test('renderPreview pretty-prints JSON content', () => {
  const result = renderPreview(
    { content_type: 'application/json', kind: 'report', path: 'report.json', preview_truncated: false },
    '{"alpha":1,"nested":{"beta":true}}',
  );

  assert.equal(result.renderer, 'json');
  assert.equal(result.output, 'text');
  assert.equal(result.fallback, false);
  assert.equal(result.text, '{\n  "alpha": 1,\n  "nested": {\n    "beta": true\n  }\n}');
  assert.equal(result.truncated_notice, '');
});

test('renderPreview falls back to generic text when JSON parsing fails', () => {
  const preview = '{"alpha":';
  const result = renderPreview(
    { content_type: 'application/json', kind: 'report', path: 'report.json', preview_truncated: false },
    preview,
  );

  assert.equal(result.renderer, 'text');
  assert.equal(result.output, 'text');
  assert.equal(result.fallback, true);
  assert.equal(result.fallback_reason, 'parse_failure');
  assert.equal(result.attempted_renderer, 'json');
  assert.equal(result.text, preview);
});

test('renderPreview renders Markdown while keeping it inert', () => {
  const result = renderPreview(
    { content_type: 'text/markdown', kind: 'notes', path: 'README.md', preview_truncated: false },
    '# Title\n\nParagraph with `code`.\n\n- first\n- second\n\n<img src="https://example.com/x.png">\n\n![Alt](https://example.com/x.png)',
  );

  assert.equal(result.renderer, 'markdown');
  assert.equal(result.output, 'html');
  assert.equal(result.fallback, false);
  assert.match(result.html, /<h1>Title<\/h1>/);
  assert.match(result.html, /<p>Paragraph with <code>code<\/code>\.<\/p>/);
  assert.match(result.html, /<ul>\s*<li>first<\/li>\s*<li>second<\/li>\s*<\/ul>/);
  assert.match(result.html, /&lt;img src=&quot;https:\/\/example\.com\/x\.png&quot;&gt;/);
  assert.match(result.html, /!\[Alt\]\(https:\/\/example\.com\/x\.png\)/);
  assert.ok(!result.html.includes('<img'));
  assert.ok(!result.html.includes('<iframe'));
  assert.ok(!result.html.includes('<embed'));
});

test('renderPreview detects diff artifacts by content type or kind or path', () => {
  const diffPreview = 'diff --git a/app.txt b/app.txt\n@@ -1,2 +1,2 @@\n-old\n+new\n context';

  const byContentType = renderPreview(
    { content_type: 'text/x-diff', kind: 'artifact', path: 'app.txt', preview_truncated: false },
    diffPreview,
  );
  assert.equal(byContentType.renderer, 'diff');
  assert.equal(byContentType.output, 'lines');
  assert.deepEqual(byContentType.lines.map((line) => line.type), ['meta', 'hunk', 'remove', 'add', 'context']);

  const byKind = renderPreview(
    { content_type: 'text/plain', kind: 'patch', path: 'notes.txt', preview_truncated: false },
    diffPreview,
  );
  assert.equal(byKind.renderer, 'diff');

  const byPath = renderPreview(
    { content_type: 'text/plain', kind: 'artifact', path: 'changes.patch', preview_truncated: false },
    diffPreview,
  );
  assert.equal(byPath.renderer, 'diff');
});

test('renderPreview preserves truncation signals and falls back when partial structured content is misleading', () => {
  const truncatedJSON = renderPreview(
    { content_type: 'application/json', kind: 'report', path: 'report.json', preview_truncated: true },
    '{"alpha":1,"nested":',
  );

  assert.equal(truncatedJSON.renderer, 'text');
  assert.equal(truncatedJSON.fallback, true);
  assert.equal(truncatedJSON.fallback_reason, 'truncated_preview');
  assert.equal(truncatedJSON.attempted_renderer, 'json');
  assert.equal(truncatedJSON.truncated, true);
  assert.equal(truncatedJSON.truncated_notice, 'Preview truncated to the workbench preview limit.');

  const truncatedMarkdown = renderPreview(
    { content_type: 'text/markdown', kind: 'notes', path: 'README.md', preview_truncated: true },
    '# Title\n\nStill readable',
  );

  assert.equal(truncatedMarkdown.renderer, 'markdown');
  assert.equal(truncatedMarkdown.fallback, false);
  assert.equal(truncatedMarkdown.truncated, true);
  assert.equal(truncatedMarkdown.truncated_notice, 'Preview truncated to the workbench preview limit.');
});
