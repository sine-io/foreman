const test = require('node:test');
const assert = require('node:assert/strict');

const renderers = require('./artifact-renderers.js');
const ergonomics = require('./artifact-log-ergonomics.js');

const {
  buildLogErgonomicsModel,
  createExpansionState,
  expandAllState,
  extractSummaryAnchors,
  isLongTextPreview,
  renderLineNumberedText,
  sliceCollapsedTeaser,
} = ergonomics;

const buildLongPreview = (lineCount, linePrefix = 'line') =>
  Array.from({ length: lineCount }, (_, index) => `${linePrefix} ${index + 1}`).join('\n');

test('isLongTextPreview only enables ergonomics for sufficiently long generic text', () => {
  const shortDetail = { content_type: 'text/plain', kind: 'artifact', path: 'notes.txt', preview: 'short preview' };
  const shortPreviewResult = renderers.renderPreview(shortDetail, shortDetail.preview);
  assert.equal(isLongTextPreview(shortDetail, shortPreviewResult), false);

  const longDetail = {
    content_type: 'text/plain',
    kind: 'artifact',
    path: 'server.log',
    preview: buildLongPreview(18, 'log line'),
  };
  const longPreviewResult = renderers.renderPreview(longDetail, longDetail.preview);
  assert.equal(isLongTextPreview(longDetail, longPreviewResult), true);
});

test('renderLineNumberedText returns numbered lines for bounded preview text', () => {
  const lines = renderLineNumberedText('alpha\nbeta\n');

  assert.deepEqual(lines, [
    { lineNumber: 1, text: 'alpha' },
    { lineNumber: 2, text: 'beta' },
    { lineNumber: 3, text: '' },
  ]);
});

test('sliceCollapsedTeaser returns a first-screen teaser and hidden count', () => {
  const lines = renderLineNumberedText(buildLongPreview(8, 'preview'));
  const teaser = sliceCollapsedTeaser(lines, { teaserLineCount: 3 });

  assert.equal(teaser.collapsed, true);
  assert.equal(teaser.hiddenLineCount, 5);
  assert.deepEqual(
    teaser.visibleLines.map((line) => line.lineNumber),
    [1, 2, 3],
  );
});

test('buildLogErgonomicsModel collapses long single-line previews with a character teaser', () => {
  const preview = '0123456789'.repeat(90);
  const detail = {
    content_type: 'text/plain',
    kind: 'artifact',
    path: 'stdout.log',
    preview,
    preview_truncated: false,
  };
  const previewResult = renderers.renderPreview(detail, detail.preview);
  const model = buildLogErgonomicsModel(detail, previewResult, { teaserCharacters: 120 });

  assert.equal(model.eligible, true);
  assert.equal(model.teaser.collapsed, true);
  assert.equal(model.teaser.hiddenLineCount, 0);
  assert.equal(model.teaser.hiddenCharacterCount > 0, true);
  assert.equal(model.expansion.canExpand, true);
  assert.equal(model.expansion.expanded, false);
  assert.equal(model.teaser.visibleLines.length, 1);
  assert.equal(model.teaser.visibleLines[0].text.length, 120);
  assert.equal(model.teaser.visibleLines[0].text, preview.slice(0, 120));
});

test('expand-all state helpers transition from collapsed to expanded without mutation', () => {
  const initialState = createExpansionState({ eligible: true });
  const expandedState = expandAllState(initialState);

  assert.deepEqual(initialState, { canExpand: true, expanded: false });
  assert.deepEqual(expandedState, { canExpand: true, expanded: true });
  assert.notStrictEqual(expandedState, initialState);
});

test('extractSummaryAnchors derives navigation from current summary and bounded preview only', () => {
  const navigation = extractSummaryAnchors(
    'Bootstrap completed; failed to connect; hidden footer',
    [
      '$ npm test',
      'Bootstrap completed in 2s',
      'ERROR failed to connect to database',
      'ready for retry',
    ].join('\n'),
  );

  assert.equal(Array.isArray(navigation.anchors), true);
  assert.deepEqual(
    navigation.anchors.map((anchor) => anchor.lineNumber),
    [1, 2, 3],
  );
  assert.match(navigation.anchors[1].label, /Bootstrap completed/i);
  assert.match(navigation.anchors[2].label, /failed to connect/i);
  assert.equal(navigation.anchors.some((anchor) => /hidden footer/i.test(anchor.label)), false);
});

test('buildLogErgonomicsModel keeps JSON, Markdown, and diff success paths out of generic log ergonomics', () => {
  const jsonDetail = {
    content_type: 'application/json',
    kind: 'report',
    path: 'report.json',
    preview: '{"alpha":1,"nested":{"beta":true}}',
    preview_truncated: false,
  };
  const jsonModel = buildLogErgonomicsModel(jsonDetail, renderers.renderPreview(jsonDetail, jsonDetail.preview));
  assert.equal(jsonModel.eligible, false);

  const markdownDetail = {
    content_type: 'text/markdown',
    kind: 'notes',
    path: 'README.md',
    preview: '# Title\n\nParagraph',
    preview_truncated: false,
  };
  const markdownModel = buildLogErgonomicsModel(
    markdownDetail,
    renderers.renderPreview(markdownDetail, markdownDetail.preview),
  );
  assert.equal(markdownModel.eligible, false);

  const diffDetail = {
    content_type: 'text/x-diff',
    kind: 'patch',
    path: 'changes.patch',
    preview: 'diff --git a/app.txt b/app.txt\n@@ -1 +1 @@\n-old\n+new',
    preview_truncated: false,
  };
  const diffModel = buildLogErgonomicsModel(diffDetail, renderers.renderPreview(diffDetail, diffDetail.preview));
  assert.equal(diffModel.eligible, false);
});

test('buildLogErgonomicsModel allows structured artifacts that already fell back to generic text', () => {
  const detail = {
    content_type: 'application/json',
    kind: 'report',
    path: 'report.json',
    preview: `${buildLongPreview(16, '{"alpha"')}`,
    preview_truncated: false,
    summary: 'alpha parse failure',
  };
  const previewResult = renderers.renderPreview(detail, detail.preview);
  const model = buildLogErgonomicsModel(detail, previewResult);

  assert.equal(previewResult.renderer, 'text');
  assert.equal(previewResult.fallback, true);
  assert.equal(model.eligible, true);
  assert.equal(model.lines.length, 16);
  assert.equal(model.teaser.collapsed, true);
});

test('buildLogErgonomicsModel requires a valid generic previewResult contract before applying ergonomics', () => {
  const detail = {
    content_type: 'application/json',
    kind: 'report',
    path: 'report.json',
    preview: 'x'.repeat(900),
    preview_truncated: false,
  };

  const missingPreviewResultModel = buildLogErgonomicsModel(detail);
  assert.equal(missingPreviewResultModel.eligible, false);
  assert.equal(missingPreviewResultModel.expansion.canExpand, false);

  const invalidPreviewResultModel = buildLogErgonomicsModel(detail, { renderer: 'text' });
  assert.equal(invalidPreviewResultModel.eligible, false);
  assert.equal(invalidPreviewResultModel.expansion.canExpand, false);

  const fallbackPreviewResult = renderers.renderPreview(detail, detail.preview);
  assert.equal(fallbackPreviewResult.fallback, true);
  const fallbackModel = buildLogErgonomicsModel(detail, fallbackPreviewResult);
  assert.equal(fallbackModel.eligible, true);
});
