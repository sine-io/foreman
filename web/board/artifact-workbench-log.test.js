const test = require('node:test');
const assert = require('node:assert/strict');
const path = require('node:path');

const renderers = require('./artifact-renderers.js');
const ergonomics = require('./artifact-log-ergonomics.js');

const workbenchModulePath = path.resolve(__dirname, 'artifact-workbench.js');

const flushPromises = async () => {
  await new Promise((resolve) => setImmediate(resolve));
  await new Promise((resolve) => setImmediate(resolve));
};

const createElement = (id) => {
  const listeners = new Map();

  return {
    id,
    innerHTML: '',
    textContent: '',
    value: '',
    dataset: {},
    addEventListener(type, handler) {
      listeners.set(type, handler);
    },
    dispatch(type, eventInit = {}) {
      const handler = listeners.get(type);
      if (!handler) {
        return undefined;
      }
      return handler({
        preventDefault() {},
        stopPropagation() {},
        currentTarget: this,
        target: eventInit.target || this,
        key: eventInit.key,
      });
    },
  };
};

const buildLongPreview = (artifactLabel, lineCount = 16) =>
  Array.from({ length: lineCount }, (_, index) => {
    const lineNumber = index + 1;
    if (lineNumber === 1) {
      return '$ npm test';
    }
    if (lineNumber === 2) {
      return `${artifactLabel} bootstrap complete`;
    }
    if (lineNumber === 12) {
      return `${artifactLabel} failed to connect to database`;
    }
    return `${artifactLabel} line ${lineNumber}`;
  }).join('\n');

const buildArtifactDetail = (artifactID, options = {}) => ({
  artifact_id: artifactID,
  kind: options.kind || 'artifact',
  summary: options.summary || 'bootstrap complete; failed to connect to database',
  preview: options.preview || buildLongPreview(artifactID),
  preview_truncated: Boolean(options.preview_truncated),
  content_type: options.content_type || 'text/plain',
  path: options.path || `logs/${artifactID}.log`,
  raw_content_url: `/artifacts/${artifactID}`,
  run_workbench_url: '/board/runs/workbench?run_id=run-1',
  run_id: 'run-1',
  task_id: 'task-1',
  project_id: 'project-1',
  module_id: 'module-1',
  siblings: options.siblings || [
    { artifact_id: 'artifact-1', kind: 'artifact', summary: 'artifact-1', selected: artifactID === 'artifact-1' },
    { artifact_id: 'artifact-2', kind: 'artifact', summary: 'artifact-2', selected: artifactID === 'artifact-2' },
  ],
});

const createActionTarget = (action, dataset = {}) => ({
  closest(selector) {
    if (selector !== '[data-artifact-preview-action]') {
      return null;
    }
    return {
      dataset: {
        artifactPreviewAction: action,
        ...dataset,
      },
    };
  },
});

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

const withWorkbenchHarness = async (detailsByID, runTest) => {
  const previousDocument = global.document;
  const previousWindow = global.window;
  const previousLocation = global.location;
  const previousHistory = global.history;
  const previousFetch = global.fetch;
  const previousRequestAnimationFrame = global.requestAnimationFrame;
  const previousRenderers = global.ForemanArtifactRenderers;
  const previousErgonomics = global.ForemanArtifactLogErgonomics;

  const nodes = new Map();
  [
    'artifact-workbench-artifact-id',
    'artifact-workbench-refresh',
    'artifact-workbench-status',
    'artifact-workbench-siblings',
    'artifact-workbench-detail',
    'artifact-workbench-metadata',
  ].forEach((id) => {
    nodes.set(id, createElement(id));
  });

  const documentRef = {
    getElementById(id) {
      return nodes.get(id) || null;
    },
  };

  const windowRef = {
    document: documentRef,
    location: { search: '?artifact_id=artifact-1', pathname: '/board/artifacts/workbench' },
    history: { replaceState() {} },
    ForemanArtifactRenderers: renderers,
    ForemanArtifactLogErgonomics: ergonomics,
    addEventListener() {},
  };

  global.document = documentRef;
  global.window = windowRef;
  global.location = windowRef.location;
  global.history = windowRef.history;
  global.fetch = async (url) => {
    const artifactID = String(url).match(/artifacts\/([^/]+)\/workbench/)?.[1];
    const detail = artifactID ? detailsByID[decodeURIComponent(artifactID)] : null;
    if (!detail) {
      return {
        ok: false,
        status: 404,
        async json() {
          return {};
        },
      };
    }
    return {
      ok: true,
      status: 200,
      async json() {
        return detail;
      },
    };
  };
  global.requestAnimationFrame = (callback) => callback();
  global.ForemanArtifactRenderers = renderers;
  global.ForemanArtifactLogErgonomics = ergonomics;

  delete require.cache[workbenchModulePath];

  try {
    const workbench = require(workbenchModulePath);
    await flushPromises();
    await runTest({
      workbench,
      artifactInput: nodes.get('artifact-workbench-artifact-id'),
      refreshButton: nodes.get('artifact-workbench-refresh'),
      detailRoot: nodes.get('artifact-workbench-detail'),
      statusNode: nodes.get('artifact-workbench-status'),
    });
  } finally {
    delete require.cache[workbenchModulePath];

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

    if (previousFetch === undefined) {
      delete global.fetch;
    } else {
      global.fetch = previousFetch;
    }

    if (previousRequestAnimationFrame === undefined) {
      delete global.requestAnimationFrame;
    } else {
      global.requestAnimationFrame = previousRequestAnimationFrame;
    }

    if (previousRenderers === undefined) {
      delete global.ForemanArtifactRenderers;
    } else {
      global.ForemanArtifactRenderers = previousRenderers;
    }

    if (previousErgonomics === undefined) {
      delete global.ForemanArtifactLogErgonomics;
    } else {
      global.ForemanArtifactLogErgonomics = previousErgonomics;
    }
  }
};

test('long text previews start collapsed with line numbers and expand-all reveals the full bounded preview', async () => {
  await withWorkbenchHarness(
    {
      'artifact-1': buildArtifactDetail('artifact-1'),
    },
    async ({ detailRoot }) => {
      assert.match(detailRoot.innerHTML, /artifact-preview-line-number/);
      assert.match(detailRoot.innerHTML, /Expand all/);
      assert.doesNotMatch(detailRoot.innerHTML, /id="artifact-preview-line-12"/);

      detailRoot.dispatch('click', { target: createActionTarget('expand-all') });
      await flushPromises();

      assert.match(detailRoot.innerHTML, /id="artifact-preview-line-12"/);
      assert.match(detailRoot.innerHTML, /artifact-1 failed to connect to database/);
      assert.match(detailRoot.innerHTML, /artifact-preview-line-number">12</);
    },
  );
});

test('summary navigation renders anchors for long text and auto-expands hidden targets', async () => {
  await withWorkbenchHarness(
    {
      'artifact-1': buildArtifactDetail('artifact-1'),
    },
    async ({ detailRoot }) => {
      assert.match(detailRoot.innerHTML, /artifact-preview-summary-nav/);
      assert.match(detailRoot.innerHTML, /data-line-number="12"/);
      assert.doesNotMatch(detailRoot.innerHTML, /id="artifact-preview-line-12"/);

      detailRoot.dispatch('click', {
        target: createActionTarget('jump-to-line', { lineNumber: '12' }),
      });
      await flushPromises();

      assert.match(detailRoot.innerHTML, /id="artifact-preview-line-12"/);
      assert.match(detailRoot.innerHTML, /artifact-1 failed to connect to database/);
    },
  );
});

test('summary navigation keeps collapsed teaser when the requested line is already visible', async () => {
  await withWorkbenchHarness(
    {
      'artifact-1': buildArtifactDetail('artifact-1'),
    },
    async ({ detailRoot }) => {
      assert.match(detailRoot.innerHTML, /data-line-number="2"/);
      assert.match(detailRoot.innerHTML, /data-expanded="false"/);
      assert.doesNotMatch(detailRoot.innerHTML, /id="artifact-preview-line-12"/);

      detailRoot.dispatch('click', {
        target: createActionTarget('jump-to-line', { lineNumber: '2' }),
      });
      await flushPromises();

      assert.match(detailRoot.innerHTML, /data-expanded="false"/);
      assert.match(detailRoot.innerHTML, /Expand all/);
      assert.doesNotMatch(detailRoot.innerHTML, /id="artifact-preview-line-12"/);
      assert.match(detailRoot.innerHTML, /artifact-preview-log-line is-targeted" id="artifact-preview-line-2"/);
    },
  );
});

test('summary navigation expands single-line previews when the anchor target sits beyond the teaser character budget', async () => {
  const summaryTarget = 'late marker';
  const hiddenPreviewTail = 'preview hidden tail marker';
  const preview = `${'0123456789'.repeat(45)} ${hiddenPreviewTail} ${summaryTarget} ${'abcdef'.repeat(35)}`;

  await withWorkbenchHarness(
    {
      'artifact-1': buildArtifactDetail('artifact-1', {
        summary: summaryTarget,
        preview,
      }),
    },
    async ({ detailRoot }) => {
      assert.match(detailRoot.innerHTML, /data-line-number="1"/);
      assert.match(detailRoot.innerHTML, /data-expanded="false"/);
      assert.doesNotMatch(detailRoot.innerHTML, /preview hidden tail marker/);

      detailRoot.dispatch('click', {
        target: createActionTarget('jump-to-line', { lineNumber: '1' }),
      });
      await flushPromises();

      assert.match(detailRoot.innerHTML, /data-expanded="true"/);
      assert.match(detailRoot.innerHTML, /preview hidden tail marker/);
      assert.match(detailRoot.innerHTML, /artifact-preview-log-line is-targeted" id="artifact-preview-line-1"/);
    },
  );
});

test('summary navigation expands when a visible teaser line is itself character-truncated', async () => {
  const summaryTarget = 'line two anchor marker';
  const hiddenPreviewTail = 'preview hidden multi tail marker';
  const preview = [
    'a'.repeat(260),
    `${'b'.repeat(165)} ${hiddenPreviewTail} ${summaryTarget} ${'c'.repeat(40)}`,
    'd'.repeat(260),
  ].join('\n');

  await withWorkbenchHarness(
    {
      'artifact-1': buildArtifactDetail('artifact-1', {
        summary: summaryTarget,
        preview,
      }),
    },
    async ({ detailRoot }) => {
      assert.match(detailRoot.innerHTML, /data-line-number="2"/);
      assert.match(detailRoot.innerHTML, /data-expanded="false"/);
      assert.doesNotMatch(detailRoot.innerHTML, /preview hidden multi tail marker/);

      detailRoot.dispatch('click', {
        target: createActionTarget('jump-to-line', { lineNumber: '2' }),
      });
      await flushPromises();

      assert.match(detailRoot.innerHTML, /data-expanded="true"/);
      assert.match(detailRoot.innerHTML, /preview hidden multi tail marker/);
      assert.match(detailRoot.innerHTML, /artifact-preview-log-line is-targeted" id="artifact-preview-line-2"/);
    },
  );
});

test('switching to a different artifact resets long-text previews to collapsed mode', async () => {
  await withWorkbenchHarness(
    {
      'artifact-1': buildArtifactDetail('artifact-1'),
      'artifact-2': buildArtifactDetail('artifact-2'),
    },
    async ({ artifactInput, refreshButton, detailRoot }) => {
      detailRoot.dispatch('click', { target: createActionTarget('expand-all') });
      await flushPromises();
      assert.match(detailRoot.innerHTML, /artifact-1 failed to connect to database/);

      artifactInput.value = 'artifact-2';
      refreshButton.dispatch('click');
      await flushPromises();

      assert.match(detailRoot.innerHTML, /artifact-2 bootstrap complete/);
      assert.doesNotMatch(detailRoot.innerHTML, /id="artifact-preview-line-12"/);
      assert.match(detailRoot.innerHTML, /Expand all/);
    },
  );
});

test('structured renderer success paths stay untouched even when they output text', () => {
  const { composeArtifactPreviewMarkup } = loadWorkbenchModule();

  const markup = composeArtifactPreviewMarkup(
    {
      content_type: 'application/json',
      kind: 'report',
      path: 'report.json',
      preview: JSON.stringify({ lines: Array.from({ length: 40 }, (_, index) => `row ${index + 1}`) }),
      preview_truncated: false,
      raw_content_url: '/artifacts/report.json',
    },
    {
      renderers,
      logErgonomics: ergonomics,
    },
  );

  assert.match(markup, /artifact-preview-json/);
  assert.doesNotMatch(markup, /artifact-preview-line-number/);
  assert.doesNotMatch(markup, /Expand all/);
  assert.doesNotMatch(markup, /artifact-preview-summary-nav/);
});

test('generic long-text rendering falls back to the existing text preview when ergonomics fail', () => {
  const { composeArtifactPreviewMarkup } = loadWorkbenchModule();
  const preview = buildLongPreview('artifact-1');

  const markup = composeArtifactPreviewMarkup(
    {
      content_type: 'text/plain',
      kind: 'artifact',
      path: 'logs/artifact-1.log',
      preview,
      preview_truncated: false,
      raw_content_url: '/artifacts/artifact-1',
    },
    {
      renderers,
      logErgonomics: {
        buildLogErgonomicsModel() {
          throw new Error('boom');
        },
      },
    },
  );

  assert.match(markup, /<pre class="artifact-preview artifact-preview-text">/);
  assert.match(markup, /artifact-1 failed to connect to database/);
  assert.doesNotMatch(markup, /Expand all/);
  assert.doesNotMatch(markup, /artifact-preview-line-number/);
});

test('truncation warnings stay visible while long text is collapsed and after expansion', async () => {
  await withWorkbenchHarness(
    {
      'artifact-1': buildArtifactDetail('artifact-1', { preview_truncated: true }),
    },
    async ({ detailRoot }) => {
      assert.match(detailRoot.innerHTML, /Preview truncated to the workbench preview limit\./);
      assert.doesNotMatch(detailRoot.innerHTML, /id="artifact-preview-line-12"/);

      detailRoot.dispatch('click', { target: createActionTarget('expand-all') });
      await flushPromises();

      assert.match(detailRoot.innerHTML, /id="artifact-preview-line-12"/);
      assert.match(detailRoot.innerHTML, /artifact-1 failed to connect to database/);
      assert.match(detailRoot.innerHTML, /Preview truncated to the workbench preview limit\./);
    },
  );
});
