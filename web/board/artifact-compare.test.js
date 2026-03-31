const test = require('node:test');
const assert = require('node:assert/strict');
const path = require('node:path');

const compareModulePath = path.resolve(__dirname, 'artifact-compare.js');

const loadCompareModule = () => {
  const previousDocument = global.document;
  const previousWindow = global.window;
  const previousLocation = global.location;
  const previousHistory = global.history;

  global.document = { getElementById: () => null };
  global.window = {
    location: { search: '', pathname: '/board/artifacts/compare' },
    history: { replaceState() {} },
    addEventListener() {},
  };
  global.location = global.window.location;
  global.history = global.window.history;

  delete require.cache[compareModulePath];

  try {
    return require(compareModulePath);
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

const { buildArtifactCompareURL, composeArtifactCompareMarkup } = loadCompareModule();

test('ready state renders unified diff content and navigation links', () => {
  const markup = composeArtifactCompareMarkup({
    status: 'ready',
    current: {
      artifact_id: 'artifact-current',
      run_id: 'run-2',
      task_id: 'task-1',
      kind: 'assistant_summary',
      content_type: 'text/plain; charset=utf-8',
      created_at: '2026-04-01T10:00:00Z',
    },
    previous: {
      artifact_id: 'artifact-previous',
      run_id: 'run-1',
      task_id: 'task-1',
      kind: 'assistant_summary',
      content_type: 'text/plain; charset=utf-8',
      created_at: '2026-04-01T09:00:00Z',
    },
    diff: {
      format: 'text/unified-diff',
      content: '--- previous:artifact-previous\n+++ current:artifact-current\n',
    },
    limits: { max_compare_bytes: 65536 },
    messages: {
      title: 'Compare ready',
      detail: 'Showing a unified diff between the current artifact and the previous artifact.',
    },
    navigation: {
      current_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-current',
      previous_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-previous',
      back_to_run_url: '/board/runs/workbench?run_id=run-2',
    },
    history: [
      {
        artifact_id: 'artifact-previous',
        run_id: 'run-1',
        created_at: '2026-04-01T09:00:00Z',
        summary: 'Selected previous summary',
        selected: true,
        compare_url: '/board/artifacts/compare?artifact_id=artifact-current&previous_artifact_id=artifact-previous',
      },
      {
        artifact_id: 'artifact-older',
        run_id: 'run-0',
        created_at: '2026-04-01T08:00:00Z',
        summary: 'Older summary',
        selected: false,
        compare_url: '/board/artifacts/compare?artifact_id=artifact-current&previous_artifact_id=artifact-older',
      },
    ],
  });

  assert.match(markup, /Artifact Compare/);
  assert.match(markup, /previous:artifact-previous/);
  assert.match(markup, /current:artifact-current/);
  assert.match(markup, /Back to current artifact/);
  assert.match(markup, /Back to run workbench/);
  assert.match(markup, /Recent History/);
  assert.match(markup, /artifact-older/);
  assert.match(markup, /previous_artifact_id=artifact-older/);
  assert.match(markup, /is-selected/);
});

test('no_previous state renders an empty-state compare panel', () => {
  const markup = composeArtifactCompareMarkup({
    status: 'no_previous',
    current: {
      artifact_id: 'artifact-current',
      run_id: 'run-2',
      task_id: 'task-1',
      kind: 'assistant_summary',
      content_type: 'text/plain; charset=utf-8',
      created_at: '2026-04-01T10:00:00Z',
    },
    previous: null,
    diff: null,
    limits: { max_compare_bytes: 65536 },
    messages: {
      title: 'No previous artifact',
      detail: 'No earlier artifact with the same task and kind is available for compare.',
    },
    navigation: {
      current_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-current',
      back_to_run_url: '/board/runs/workbench?run_id=run-2',
    },
  });

  assert.match(markup, /No previous artifact/);
  assert.doesNotMatch(markup, /artifact-previous/);
});

test('unsupported state keeps previous metadata but no diff', () => {
  const markup = composeArtifactCompareMarkup({
    status: 'unsupported',
    current: {
      artifact_id: 'artifact-current',
      run_id: 'run-2',
      task_id: 'task-1',
      kind: 'screenshot',
      content_type: 'image\/png',
      created_at: '2026-04-01T10:00:00Z',
    },
    previous: {
      artifact_id: 'artifact-previous',
      run_id: 'run-1',
      task_id: 'task-1',
      kind: 'screenshot',
      content_type: 'image\/png',
      created_at: '2026-04-01T09:00:00Z',
    },
    diff: null,
    limits: { max_compare_bytes: 65536 },
    messages: {
      title: 'Compare unavailable',
      detail: 'Artifact compare currently supports text and structured-text content only.',
    },
    navigation: {
      current_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-current',
      previous_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-previous',
      back_to_run_url: '/board/runs/workbench?run_id=run-2',
    },
  });

  assert.match(markup, /Compare unavailable/);
  assert.match(markup, /artifact-previous/);
  assert.doesNotMatch(markup, /text\/unified-diff/);
});

test('too_large state renders limit-aware messaging', () => {
  const markup = composeArtifactCompareMarkup({
    status: 'too_large',
    current: {
      artifact_id: 'artifact-current',
      run_id: 'run-2',
      task_id: 'task-1',
      kind: 'assistant_summary',
      content_type: 'text\/plain; charset=utf-8',
      created_at: '2026-04-01T10:00:00Z',
    },
    previous: {
      artifact_id: 'artifact-previous',
      run_id: 'run-1',
      task_id: 'task-1',
      kind: 'assistant_summary',
      content_type: 'text\/plain; charset=utf-8',
      created_at: '2026-04-01T09:00:00Z',
    },
    diff: null,
    limits: { max_compare_bytes: 65536 },
    messages: {
      title: 'Compare too large',
      detail: 'One or both artifacts exceed the 65536 byte compare limit.',
    },
    navigation: {
      current_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-current',
      previous_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-previous',
      back_to_run_url: '/board/runs/workbench?run_id=run-2',
    },
  });

  assert.match(markup, /Compare too large/);
  assert.match(markup, /65536/);
});

test('refresh URLs stay keyed by the current artifact_id', () => {
  assert.equal(
    buildArtifactCompareURL('artifact-current'),
    '/board/artifacts/compare?artifact_id=artifact-current',
  );
});

test('boot path clears stale compare content during popstate-driven reload', async () => {
  const listeners = new Map();
  const nodes = new Map();
  const makeNode = () => ({
    innerHTML: '',
    textContent: '',
    dataset: {},
    addEventListener() {},
  });
  [
    'artifact-compare-artifact-id',
    'artifact-compare-refresh',
    'artifact-compare-status',
    'artifact-compare-current',
    'artifact-compare-result',
    'artifact-compare-previous',
  ].forEach((id) => nodes.set(id, makeNode()));

  const previousDocument = global.document;
  const previousWindow = global.window;
  const previousLocation = global.location;
  const previousHistory = global.history;
  const previousFetch = global.fetch;

  let resolveFirst;
  let resolveSecond;
  const firstResponse = new Promise((resolve) => {
    resolveFirst = resolve;
  });
  const secondResponse = new Promise((resolve) => {
    resolveSecond = resolve;
  });
  let callCount = 0;

  global.document = {
    getElementById(id) {
      return nodes.get(id) || null;
    },
  };
  global.window = {
    location: { search: '?artifact_id=artifact-current', pathname: '/board/artifacts/compare' },
    history: { replaceState() {} },
    addEventListener(type, listener) {
      listeners.set(type, listener);
    },
  };
  global.location = global.window.location;
  global.history = global.window.history;
  global.fetch = () => {
    callCount += 1;
    const payload =
      callCount === 1
        ? firstResponse
        : secondResponse;
    return payload;
  };

  delete require.cache[compareModulePath];

  try {
    require(compareModulePath);

    resolveFirst({
      ok: true,
      json: async () => ({
        status: 'ready',
        current: { artifact_id: 'artifact-current', run_id: 'run-2', task_id: 'task-1', kind: 'assistant_summary', content_type: 'text/plain', created_at: '2026-04-01T10:00:00Z' },
        previous: { artifact_id: 'artifact-previous', run_id: 'run-1', task_id: 'task-1', kind: 'assistant_summary', content_type: 'text/plain', created_at: '2026-04-01T09:00:00Z' },
        diff: { format: 'text/unified-diff', content: 'current artifact compare' },
        limits: { max_compare_bytes: 65536 },
        messages: { title: 'Compare ready', detail: 'Showing a unified diff between the current artifact and the previous artifact.' },
        navigation: { current_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-current', previous_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-previous', back_to_run_url: '/board/runs/workbench?run_id=run-2' },
        history: [
          { artifact_id: 'artifact-previous', run_id: 'run-1', created_at: '2026-04-01T09:00:00Z', summary: 'Previous', selected: true, compare_url: '/board/artifacts/compare?artifact_id=artifact-current&previous_artifact_id=artifact-previous' },
          { artifact_id: 'artifact-older', run_id: 'run-0', created_at: '2026-04-01T08:00:00Z', summary: 'Older', selected: false, compare_url: '/board/artifacts/compare?artifact_id=artifact-current&previous_artifact_id=artifact-older' },
        ],
      }),
    });

    await new Promise((resolve) => setImmediate(resolve));
    await new Promise((resolve) => setImmediate(resolve));

    assert.match(nodes.get('artifact-compare-current').innerHTML, /artifact-current/);

    global.window.location.search = '?artifact_id=artifact-next&previous_artifact_id=artifact-older';
    listeners.get('popstate')();

    assert.doesNotMatch(nodes.get('artifact-compare-current').innerHTML, /artifact-current/);
    assert.match(nodes.get('artifact-compare-result').innerHTML, /Loading artifact compare/);

    resolveSecond({
      ok: true,
      json: async () => ({
        status: 'ready',
        current: { artifact_id: 'artifact-next', run_id: 'run-3', task_id: 'task-1', kind: 'assistant_summary', content_type: 'text/plain', created_at: '2026-04-01T11:00:00Z' },
        previous: { artifact_id: 'artifact-current', run_id: 'run-2', task_id: 'task-1', kind: 'assistant_summary', content_type: 'text/plain', created_at: '2026-04-01T10:00:00Z' },
        diff: { format: 'text/unified-diff', content: 'next artifact compare' },
        limits: { max_compare_bytes: 65536 },
        messages: { title: 'Compare ready', detail: 'Showing a unified diff between the current artifact and the previous artifact.' },
        navigation: { current_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-next', previous_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-current', back_to_run_url: '/board/runs/workbench?run_id=run-3' },
        history: [
          { artifact_id: 'artifact-current', run_id: 'run-2', created_at: '2026-04-01T10:00:00Z', summary: 'Current previous', selected: false, compare_url: '/board/artifacts/compare?artifact_id=artifact-next&previous_artifact_id=artifact-current' },
          { artifact_id: 'artifact-older', run_id: 'run-1', created_at: '2026-04-01T09:00:00Z', summary: 'Selected older', selected: true, compare_url: '/board/artifacts/compare?artifact_id=artifact-next&previous_artifact_id=artifact-older' },
        ],
      }),
    });

    await new Promise((resolve) => setImmediate(resolve));
    await new Promise((resolve) => setImmediate(resolve));

    assert.match(nodes.get('artifact-compare-current').innerHTML, /artifact-next/);
    assert.doesNotMatch(nodes.get('artifact-compare-current').innerHTML, /artifact-current<\/p>/);
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
    if (previousFetch === undefined) {
      delete global.fetch;
    } else {
      global.fetch = previousFetch;
    }
    delete require.cache[compareModulePath];
  }
});

test('boot path fetches compare using previous_artifact_id from URL', async () => {
  const nodes = new Map();
  const requests = [];
  const makeNode = () => ({
    innerHTML: '',
    textContent: '',
    dataset: {},
    addEventListener() {},
  });
  [
    'artifact-compare-artifact-id',
    'artifact-compare-refresh',
    'artifact-compare-status',
    'artifact-compare-current',
    'artifact-compare-result',
    'artifact-compare-previous',
  ].forEach((id) => nodes.set(id, makeNode()));

  const previousDocument = global.document;
  const previousWindow = global.window;
  const previousLocation = global.location;
  const previousHistory = global.history;
  const previousFetch = global.fetch;

  global.document = {
    getElementById(id) {
      return nodes.get(id) || null;
    },
  };
  global.window = {
    location: { search: '?artifact_id=artifact-current&previous_artifact_id=artifact-older', pathname: '/board/artifacts/compare' },
    history: { replaceState() {} },
    addEventListener() {},
  };
  global.location = global.window.location;
  global.history = global.window.history;
  global.fetch = async (url) => {
    requests.push(url);
    return {
      ok: true,
      json: async () => ({
        status: 'ready',
        current: { artifact_id: 'artifact-current', run_id: 'run-2', task_id: 'task-1', kind: 'assistant_summary', content_type: 'text/plain', created_at: '2026-04-01T10:00:00Z' },
        previous: { artifact_id: 'artifact-older', run_id: 'run-0', task_id: 'task-1', kind: 'assistant_summary', content_type: 'text/plain', created_at: '2026-04-01T08:00:00Z' },
        diff: { format: 'text/unified-diff', content: 'older compare' },
        limits: { max_compare_bytes: 65536 },
        messages: { title: 'Compare ready', detail: 'Showing a unified diff between the current artifact and the previous artifact.' },
        navigation: { current_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-current', previous_workbench_url: '/board/artifacts/workbench?artifact_id=artifact-older', back_to_run_url: '/board/runs/workbench?run_id=run-2' },
        history: [
          { artifact_id: 'artifact-older', run_id: 'run-0', created_at: '2026-04-01T08:00:00Z', summary: 'Older', selected: true, compare_url: '/board/artifacts/compare?artifact_id=artifact-current&previous_artifact_id=artifact-older' },
        ],
      }),
    };
  };

  delete require.cache[compareModulePath];

  try {
    require(compareModulePath);
    await new Promise((resolve) => setImmediate(resolve));
    await new Promise((resolve) => setImmediate(resolve));
    assert.equal(
      requests[0],
      '/api/manager/artifacts/artifact-current/compare?previous_artifact_id=artifact-older',
    );
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
    if (previousFetch === undefined) {
      delete global.fetch;
    } else {
      global.fetch = previousFetch;
    }
    delete require.cache[compareModulePath];
  }
});
