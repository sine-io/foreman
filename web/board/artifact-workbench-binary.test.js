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

const listenerUsesCapture = (options) => options === true || Boolean(options && options.capture);

const createElement = (id) => {
  const listeners = new Map();
  const registeredListeners = (type) => listeners.get(type) || [];

  const element = {
    id,
    innerHTML: '',
    textContent: '',
    value: '',
    dataset: {},
    addEventListener(type, handler, options) {
      listeners.set(type, [
        ...registeredListeners(type),
        {
          capture: listenerUsesCapture(options),
          handler,
        },
      ]);
    },
    getEventListeners(type) {
      return [...registeredListeners(type)];
    },
    dispatch(type, eventInit = {}) {
      const target = eventInit.target || element;
      const bubbles = eventInit.bubbles !== undefined ? Boolean(eventInit.bubbles) : true;
      const baseEvent = {
        preventDefault() {},
        stopPropagation() {},
        target,
        key: eventInit.key,
      };
      const registrations = registeredListeners(type);
      const targetIsCurrent = target === element;

      for (const listener of registrations) {
        if (!listener.capture) {
          continue;
        }

        listener.handler({
          ...baseEvent,
          currentTarget: element,
        });
      }

      if (targetIsCurrent || bubbles) {
        for (const listener of registrations) {
          if (listener.capture) {
            continue;
          }

          listener.handler({
            ...baseEvent,
            currentTarget: element,
          });
        }
      }

      return undefined;
    },
  };

  return element;
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
  summary: options.summary || 'artifact summary',
  preview: options.preview || '',
  preview_truncated: Boolean(options.preview_truncated),
  content_type: options.content_type || 'application/octet-stream',
  path: options.path || `artifacts/${artifactID}`,
  raw_content_url: options.raw_content_url || `/artifacts/${artifactID}`,
  run_workbench_url: '/board/runs/workbench?run_id=run-1',
  run_id: 'run-1',
  task_id: 'task-1',
  project_id: 'project-1',
  module_id: 'module-1',
  siblings: options.siblings || [
    { artifact_id: artifactID, kind: options.kind || 'artifact', summary: options.summary || artifactID, selected: true },
  ],
});

const createClosestTarget = (selector, dataset = {}) => ({
  closest(requestedSelector) {
    if (requestedSelector !== selector) {
      return null;
    }

    return { dataset };
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

const withWorkbenchHarness = async (detailsByID, runTest, options = {}) => {
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

  const initialArtifactID = options.initialArtifactID || 'artifact-1';
  const windowRef = {
    document: documentRef,
    location: { search: `?artifact_id=${encodeURIComponent(initialArtifactID)}`, pathname: '/board/artifacts/workbench' },
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
      metadataRoot: nodes.get('artifact-workbench-metadata'),
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

test('supported raster image artifacts render inline previews and keep raw actions visible', async () => {
  const { composeArtifactPreviewMarkup } = loadWorkbenchModule();
  const rasterContentTypes = ['image/png', 'image/jpeg', 'image/gif', 'image/webp'];

  for (const contentType of rasterContentTypes) {
    const markup = composeArtifactPreviewMarkup(
      buildArtifactDetail('image-artifact', {
        content_type: contentType,
        path: `screenshots/${contentType.split('/')[1]}`,
        raw_content_url: `/artifacts/image-artifact/${contentType.split('/')[1]}`,
      }),
      { renderers, logErgonomics: ergonomics },
    );

    assert.match(markup, /artifact-preview-image/);
    assert.match(markup, /artifact-preview-image-frame/);
    assert.match(markup, /Inline Preview/);
  }

  await withWorkbenchHarness(
    {
      'artifact-1': buildArtifactDetail('artifact-1', {
        content_type: 'image/png',
        summary: 'PNG screenshot',
        path: 'screenshots/result.png',
        raw_content_url: '/artifacts/artifact-1/raw',
      }),
    },
    async ({ detailRoot, metadataRoot }) => {
      assert.match(detailRoot.innerHTML, /artifact-preview-image/);
      assert.match(detailRoot.innerHTML, /\/artifacts\/artifact-1\/raw/);
      assert.match(metadataRoot.innerHTML, /Open raw artifact/);
      assert.match(metadataRoot.innerHTML, /href="\/artifacts\/artifact-1\/raw"/);
    },
  );
});

test('svg preview attempts the raw content url and falls back cleanly when the browser cannot display it', async () => {
  const { composeArtifactPreviewMarkup } = loadWorkbenchModule();
  const initialMarkup = composeArtifactPreviewMarkup(
    buildArtifactDetail('diagram', {
      content_type: 'image/svg+xml',
      summary: 'Architecture diagram',
      path: 'diagrams/architecture.svg',
      raw_content_url: '/artifacts/diagram.svg',
    }),
    { renderers, logErgonomics: ergonomics },
  );

  assert.match(initialMarkup, /artifact-preview-image/);
  assert.match(initialMarkup, /\/artifacts\/diagram\.svg/);

  await withWorkbenchHarness(
    {
      'artifact-1': buildArtifactDetail('artifact-1', {
        content_type: 'image/svg+xml',
        summary: 'Architecture diagram',
        path: 'diagrams/architecture.svg',
        raw_content_url: '/artifacts/diagram.svg',
      }),
    },
    async ({ detailRoot, metadataRoot }) => {
      assert.match(detailRoot.innerHTML, /artifact-preview-image/);
      assert.match(detailRoot.innerHTML, /\/artifacts\/diagram\.svg/);
      assert.deepEqual(
        detailRoot.getEventListeners('error').map((listener) => listener.capture),
        [true],
      );

      detailRoot.dispatch('error', {
        bubbles: false,
        target: createClosestTarget('[data-artifact-preview-image]', {
          artifactId: 'artifact-1',
        }),
      });
      await flushPromises();

      assert.doesNotMatch(detailRoot.innerHTML, /artifact-preview-image/);
      assert.match(detailRoot.innerHTML, /browser could not display this image preview/i);
      assert.match(detailRoot.innerHTML, /Open raw artifact/);
      assert.match(metadataRoot.innerHTML, /Open raw artifact/);
    },
  );
});

test('non-image binary artifacts stay on the metadata and download fallback path', async () => {
  const { composeArtifactPreviewMarkup } = loadWorkbenchModule();
  const markup = composeArtifactPreviewMarkup(
    buildArtifactDetail('artifact-binary', {
      content_type: 'application/pdf',
      summary: 'Quarterly PDF report',
      path: 'reports/q1.pdf',
      raw_content_url: '/artifacts/reports/q1.pdf',
    }),
    { renderers, logErgonomics: ergonomics },
  );

  assert.match(markup, /Inline preview is unavailable for this artifact type\./);
  assert.match(markup, /Open raw artifact/);
  assert.doesNotMatch(markup, /artifact-preview-image/);
  assert.doesNotMatch(markup, /artifact-preview-text/);

  await withWorkbenchHarness(
    {
      'artifact-1': buildArtifactDetail('artifact-1', {
        content_type: 'application/octet-stream',
        summary: 'Database dump',
        path: 'backups/dump.bin',
        raw_content_url: '/artifacts/backups/dump.bin',
      }),
    },
    async ({ detailRoot, metadataRoot }) => {
      assert.match(detailRoot.innerHTML, /Inline preview is unavailable for this artifact type\./);
      assert.match(detailRoot.innerHTML, /Open raw artifact/);
      assert.doesNotMatch(detailRoot.innerHTML, /artifact-preview-image/);
      assert.match(metadataRoot.innerHTML, /href="\/artifacts\/backups\/dump\.bin"/);
    },
  );
});

test('raster image load failures fall back cleanly without disrupting text and log rendering paths', async () => {
  await withWorkbenchHarness(
    {
      'artifact-1': buildArtifactDetail('artifact-1', {
        content_type: 'image/png',
        summary: 'Broken screenshot',
        path: 'screenshots/broken.png',
        raw_content_url: '/artifacts/screenshots/broken.png',
      }),
    },
    async ({ detailRoot, metadataRoot }) => {
      assert.match(detailRoot.innerHTML, /artifact-preview-image/);
      assert.deepEqual(
        detailRoot.getEventListeners('error').map((listener) => listener.capture),
        [true],
      );

      detailRoot.dispatch('error', {
        bubbles: false,
        target: createClosestTarget('[data-artifact-preview-image]', {
          artifactId: 'artifact-1',
        }),
      });
      await flushPromises();

      assert.doesNotMatch(detailRoot.innerHTML, /artifact-preview-image/);
      assert.match(detailRoot.innerHTML, /browser could not display this image preview/i);
      assert.match(detailRoot.innerHTML, /Open raw artifact/);
      assert.match(metadataRoot.innerHTML, /Open raw artifact/);
    },
  );

  const { composeArtifactPreviewMarkup } = loadWorkbenchModule();
  const logMarkup = composeArtifactPreviewMarkup(
    buildArtifactDetail('artifact-log', {
      content_type: 'text/plain',
      summary: 'bootstrap complete; failed to connect to database',
      preview: buildLongPreview('artifact-log'),
      path: 'logs/artifact-log.txt',
      raw_content_url: '/artifacts/logs/artifact-log.txt',
    }),
    {
      renderers,
      logErgonomics: ergonomics,
    },
  );

  assert.match(logMarkup, /artifact-preview-line-number/);
  assert.match(logMarkup, /Expand all/);
  assert.doesNotMatch(logMarkup, /artifact-preview-image/);
});
