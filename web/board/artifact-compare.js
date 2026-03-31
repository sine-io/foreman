(function (globalScope) {
  const documentRef = globalScope.document || globalThis.document;

  const normalizeText = (value) => String(value ?? "");

  const escapeHTML = (value) =>
    normalizeText(value)
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");

  const currentLocation = () => globalScope.location || globalThis.location || { search: "", pathname: "" };
  const currentHistory = () => globalScope.history || globalThis.history;

  const buildArtifactCompareURL = (artifactId) => {
    if (!artifactId) {
      return "/board/artifacts/compare";
    }
    return `/board/artifacts/compare?artifact_id=${encodeURIComponent(artifactId)}`;
  };

  const compareFetchURL = (artifactId, previousArtifactId) => {
    const searchParams = new URLSearchParams();
    if (previousArtifactId) {
      searchParams.set("previous_artifact_id", previousArtifactId);
    }
    const query = searchParams.toString();
    return query
      ? `/api/manager/artifacts/${encodeURIComponent(artifactId)}/compare?${query}`
      : `/api/manager/artifacts/${encodeURIComponent(artifactId)}/compare`;
  };

  const metadataBlock = (label, value) => `
    <article class="detail-block">
      <p class="detail-label">${escapeHTML(label)}</p>
      <p class="detail-copy">${escapeHTML(value || "Not recorded")}</p>
    </article>
  `;

  const composeArtifactCompareView = (detail) => {
    const current = detail && detail.current ? detail.current : {};
    const previous = detail && detail.previous ? detail.previous : null;
    const history = detail && Array.isArray(detail.history) ? detail.history : [];
    const navigation = detail && detail.navigation ? detail.navigation : {};
    const messages = detail && detail.messages ? detail.messages : {};
    const diff = detail && detail.diff ? detail.diff : null;

    const currentMarkup = `
      <article class="approval-detail-card">
        <section class="detail-grid detail-grid-secondary artifact-compare-metadata-grid">
          ${metadataBlock("Artifact ID", current.artifact_id)}
          ${metadataBlock("Run ID", current.run_id)}
          ${metadataBlock("Task ID", current.task_id)}
          ${metadataBlock("Kind", current.kind)}
          ${metadataBlock("Content Type", current.content_type)}
          ${metadataBlock("Created At", current.created_at)}
        </section>
      </article>
    `;

    const historyMarkup = history.length
      ? `
        <section class="detail-block detail-block-wide artifact-compare-history-block">
          <p class="detail-label">Recent History</p>
          <div class="artifact-compare-history-list">
            ${history
              .map((item) => {
                const selectedClass = item.selected ? " is-selected" : "";
                return `
                  <a class="artifact-compare-history-item${selectedClass}" href="${escapeHTML(item.compare_url || "")}">
                    <strong>${escapeHTML(item.summary || item.artifact_id || "Artifact")}</strong>
                    <p class="detail-copy">${escapeHTML(item.artifact_id || "")}</p>
                    <p class="detail-copy">${escapeHTML(item.run_id || "")}</p>
                    <p class="detail-copy">${escapeHTML(item.created_at || "")}</p>
                  </a>
                `;
              })
              .join("")}
          </div>
        </section>
      `
      : `
        <section class="detail-block detail-block-wide artifact-compare-history-block">
          <p class="detail-label">Recent History</p>
          <p class="empty-state">No recent history is available for this compare view.</p>
        </section>
      `;

    const previousMetadataMarkup = previous
      ? `
        <section class="detail-grid detail-grid-secondary artifact-compare-metadata-grid">
          ${metadataBlock("Artifact ID", previous.artifact_id)}
          ${metadataBlock("Run ID", previous.run_id)}
          ${metadataBlock("Task ID", previous.task_id)}
          ${metadataBlock("Kind", previous.kind)}
          ${metadataBlock("Content Type", previous.content_type)}
          ${metadataBlock("Created At", previous.created_at)}
        </section>
      `
      : '<p class="empty-state">No previous artifact is available for this compare view.</p>';

    const previousMarkup = `
      <article class="approval-detail-card">
        ${previousMetadataMarkup}
        ${historyMarkup}
      </article>
    `;

    const currentWorkbenchLink = navigation.current_workbench_url
      ? `<a class="board-link" href="${escapeHTML(navigation.current_workbench_url)}">Back to current artifact</a>`
      : '<span class="board-link board-link-disabled" aria-disabled="true">Current artifact unavailable</span>';
    const runWorkbenchLink = navigation.back_to_run_url
      ? `<a class="board-link" href="${escapeHTML(navigation.back_to_run_url)}">Back to run workbench</a>`
      : '<span class="board-link board-link-disabled" aria-disabled="true">Run workbench unavailable</span>';

    const resultMarkup = `
      <article class="approval-detail-card">
        <header class="approval-detail-header">
          <div>
            <p class="panel-kicker">${escapeHTML(detail && detail.status ? detail.status : "compare")}</p>
            <h3>Artifact Compare</h3>
          </div>
        </header>

        <section class="artifact-compare-actions">
          ${currentWorkbenchLink}
          ${runWorkbenchLink}
        </section>

        <section class="detail-block detail-block-wide">
          <p class="detail-label">${escapeHTML(messages.title || "Compare unavailable")}</p>
          <p class="detail-copy">${escapeHTML(messages.detail || "Artifact compare is unavailable.")}</p>
          ${
            detail && detail.status === "ready" && diff
              ? `<pre class="artifact-compare-diff">${escapeHTML(diff.content || "")}</pre>`
              : ""
          }
        </section>
      </article>
    `;

    return {
      currentMarkup,
      resultMarkup,
      previousMarkup,
    };
  };

  const composeArtifactCompareMarkup = (detail) => {
    const view = composeArtifactCompareView(detail);
    return `${view.currentMarkup}${view.resultMarkup}${view.previousMarkup}`;
  };

  const api = {
    buildArtifactCompareURL,
    composeArtifactCompareMarkup,
    composeArtifactCompareView,
  };

  globalScope.ForemanArtifactCompare = api;

  if (typeof module !== "undefined" && module.exports) {
    module.exports = api;
  }

  if (!documentRef) {
    return;
  }

  const artifactInput = documentRef.getElementById("artifact-compare-artifact-id");
  const refreshButton = documentRef.getElementById("artifact-compare-refresh");
  const statusNode = documentRef.getElementById("artifact-compare-status");
  const currentRoot = documentRef.getElementById("artifact-compare-current");
  const resultRoot = documentRef.getElementById("artifact-compare-result");
  const previousRoot = documentRef.getElementById("artifact-compare-previous");

  if (!artifactInput || !refreshButton || !statusNode || !currentRoot || !resultRoot || !previousRoot) {
    return;
  }

  const state = {
    artifactId: "",
    detail: null,
    requestToken: 0,
    notice: "",
    loading: false,
  };

  const readArtifactID = () => {
    const searchParams = new URLSearchParams(currentLocation().search);
    return searchParams.get("artifact_id") || "";
  };

  const readPreviousArtifactID = () => {
    const searchParams = new URLSearchParams(currentLocation().search);
    return searchParams.get("previous_artifact_id") || "";
  };

  const setStatus = (message, tone = "info") => {
    statusNode.textContent = message;
    statusNode.dataset.tone = tone;
  };

  const renderEmpty = (message) => {
    currentRoot.innerHTML = '<p class="empty-state">Artifact metadata will appear here.</p>';
    previousRoot.innerHTML = '<p class="empty-state">Previous artifact metadata will appear here.</p>';
    resultRoot.innerHTML = `<p class="empty-state">${escapeHTML(message)}</p>`;
  };

  const renderCompare = () => {
    artifactInput.textContent = state.artifactId || "Missing artifact_id";

    if (state.loading && !state.detail) {
      renderEmpty("Loading artifact compare...");
      return;
    }

    if (state.notice && !state.detail) {
      renderEmpty(state.notice);
      return;
    }

    const view = composeArtifactCompareView(state.detail);
    currentRoot.innerHTML = view.currentMarkup;
    resultRoot.innerHTML = view.resultMarkup;
    previousRoot.innerHTML = view.previousMarkup;
  };

  const fetchJSON = async (url) => {
    const response = await fetch(url, { method: "GET" });
    if (response.status === 404) {
      return { notFound: true };
    }
    if (response.status === 400) {
      const payload = await response.json();
      return { clientError: true, message: payload.error || "Invalid compare selection." };
    }
    if (!response.ok) {
      const payload = await response.json();
      throw new Error(payload.error || `Request failed with status ${response.status}`);
    }
    return response.json();
  };

  const loadCompare = async () => {
    const requestedArtifactId = readArtifactID();
    const requestedPreviousArtifactId = readPreviousArtifactID();
    const requestToken = ++state.requestToken;
    state.artifactId = requestedArtifactId;

    if (!requestedArtifactId) {
      state.detail = null;
      state.loading = false;
      state.notice = "Artifact compare requires artifact_id in the page URL.";
      renderCompare();
      setStatus("Artifact compare requires artifact_id.", "danger");
      return;
    }

    state.loading = true;
    state.notice = "";
    state.detail = null;
    renderCompare();
    setStatus(`Loading compare for ${requestedArtifactId}...`);

    try {
      const detail = await fetchJSON(compareFetchURL(requestedArtifactId, requestedPreviousArtifactId));
      if (requestToken !== state.requestToken) {
        return;
      }
      if (detail.clientError) {
        state.detail = null;
        state.loading = false;
        state.notice = detail.message || "Selected compare target is no longer available.";
        renderCompare();
        setStatus(`Failed to load ${requestedArtifactId}.`, "danger");
        return;
      }
      if (detail.notFound) {
        state.detail = null;
        state.loading = false;
        state.notice = `No artifact with ID ${requestedArtifactId} exists.`;
        renderCompare();
        setStatus(`Artifact ${requestedArtifactId} not found.`, "danger");
        return;
      }

      state.detail = detail;
      state.loading = false;
      renderCompare();
      setStatus(`Loaded compare for ${requestedArtifactId}.`);
    } catch (error) {
      if (requestToken !== state.requestToken) {
        return;
      }
      state.detail = null;
      state.loading = false;
      state.notice = error.message || "Failed to load artifact compare.";
      renderCompare();
      setStatus(`Failed to load ${requestedArtifactId}.`, "danger");
    }
  };

  const refreshCompare = async () => {
    await loadCompare();
  };

  refreshButton.addEventListener("click", refreshCompare);

  if (typeof globalScope.addEventListener === "function") {
    globalScope.addEventListener("popstate", () => {
      refreshCompare();
    });
  }

  state.artifactId = readArtifactID();
  refreshCompare();
})(typeof window !== "undefined" ? window : globalThis);
