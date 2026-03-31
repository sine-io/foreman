const test = require('node:test');
const assert = require('node:assert/strict');
const path = require('node:path');

const renderers = require('./artifact-renderers.js');

const workbenchModulePath = path.resolve(__dirname, 'artifact-workbench.js');

const loadWorkbenchModule = () => {
  const previousDocument = global.document;
  const previousWindow = global.window;
  const previousLocation = global.location;
  const previousHistory = global.history;

  global.document = { getElementById: () => null };
  global.window = {
    location: { search: '', pathname: '/board/artifacts/workbench' },
    history: { replaceState() {} },
    addEventListener() {},
  };
  global.location = global.window.location;
  global.history = global.window.history;

  delete require.cache[workbenchModulePath];

  try {
    return require(workbenchModulePath);
  } finally {
    if (previousDocument === undefined) {
      delete global.document;
    } else {
      global.document = previousDocument;
    }

    if (previousWindow === undefined) {
      delete global.window;
    } else {
      global.window = previousWindow;
    }

    if (previousLocation === undefined) {
      delete global.location;
    } else {
      global.location = previousLocation;
    }

    if (previousHistory === undefined) {
      delete global.history;
    } else {
      global.history = previousHistory;
    }
  }
};

const { composeArtifactPreviewMarkup } = loadWorkbenchModule();

test('TestArtifactWorkbenchJavaScriptUsesRendererHelpersForJSON', () => {
  assert.equal(typeof composeArtifactPreviewMarkup, 'function');
  const markup = composeArtifactPreviewMarkup(
    {
      content_type: 'application/json',
      kind: 'report',
      path: 'report.json',
      preview: '{"alpha":1,"nested":{"beta":true}}',
      preview_truncated: false,
      raw_content_url: '/artifacts/report.json',
    },
    { renderers },
  );

  assert.match(markup, /artifact-preview/);
  assert.match(markup, /artifact-preview-json/);
  assert.match(markup, /\n  &quot;alpha&quot;: 1,/);
  assert.match(markup, /\n    &quot;beta&quot;: true/);
});

test('TestArtifactWorkbenchJavaScriptUsesRendererHelpersForMarkdown', () => {
  assert.equal(typeof composeArtifactPreviewMarkup, 'function');
  const markup = composeArtifactPreviewMarkup(
    {
      content_type: 'text/markdown',
      kind: 'notes',
      path: 'README.md',
      preview: '# Title\n\nParagraph with `code`.',
      preview_truncated: false,
      raw_content_url: '/artifacts/readme',
    },
    { renderers },
  );

  assert.match(markup, /artifact-preview-markdown/);
  assert.match(markup, /<h1>Title<\/h1>/);
  assert.match(markup, /<p>Paragraph with <code>code<\/code>\.<\/p>/);
});

test('TestArtifactWorkbenchJavaScriptUsesRendererHelpersForDiffArtifacts', () => {
  assert.equal(typeof composeArtifactPreviewMarkup, 'function');
  const markup = composeArtifactPreviewMarkup(
    {
      content_type: 'text/x-diff',
      kind: 'patch',
      path: 'changes.patch',
      preview: 'diff --git a/app.txt b/app.txt\n@@ -1 +1 @@\n-old\n+new\n context',
      preview_truncated: false,
      raw_content_url: '/artifacts/changes.patch',
    },
    { renderers },
  );

  assert.match(markup, /artifact-preview-diff/);
  assert.match(markup, /data-diff-type="meta"/);
  assert.match(markup, /data-diff-type="add"/);
  assert.match(markup, /\+new/);
});

test('TestArtifactWorkbenchJavaScriptKeepsGenericFallback', () => {
  assert.equal(typeof composeArtifactPreviewMarkup, 'function');
  const markup = composeArtifactPreviewMarkup(
    {
      content_type: 'text/plain',
      kind: 'artifact',
      path: 'notes.txt',
      preview: '',
      preview_truncated: false,
      raw_content_url: '/artifacts/notes.txt',
    },
    { renderers },
  );

  assert.match(markup, /<pre class="artifact-preview artifact-preview-text"><\/pre>/);
  assert.doesNotMatch(markup, /artifact-preview-markdown/);
  assert.doesNotMatch(markup, /artifact-preview-diff/);
});

test('TestArtifactWorkbenchJavaScriptKeepsTruncationNoticeVisible', () => {
  assert.equal(typeof composeArtifactPreviewMarkup, 'function');
  const markup = composeArtifactPreviewMarkup(
    {
      content_type: 'text/markdown',
      kind: 'notes',
      path: 'README.md',
      preview: '# Title\n\nStill readable',
      preview_truncated: true,
      raw_content_url: '/artifacts/readme',
    },
    { renderers },
  );

  assert.match(markup, /artifact-preview-markdown/);
  assert.match(markup, /Preview truncated to the workbench preview limit\./);
});

test('TestArtifactWorkbenchJavaScriptKeepsPageUsableWhenRendererFallsBack', () => {
  assert.equal(typeof composeArtifactPreviewMarkup, 'function');
  const markup = composeArtifactPreviewMarkup(
    {
      content_type: 'text/markdown',
      kind: 'notes',
      path: 'README.md',
      preview: '# Title\n\nStill readable',
      preview_truncated: false,
      raw_content_url: '/artifacts/readme',
    },
    {
      renderers: {
        renderPreview() {
          throw new Error('renderer exploded');
        },
      },
    },
  );

  assert.match(markup, /<pre class="artifact-preview artifact-preview-text"># Title/);
  assert.doesNotMatch(markup, /Unable to load artifact detail/);
});
