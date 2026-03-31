const artifactInput = document.getElementById("artifact-workbench-artifact-id");
const refreshButton = document.getElementById("artifact-workbench-refresh");
const statusNode = document.getElementById("artifact-workbench-status");
const siblingsRoot = document.getElementById("artifact-workbench-siblings");
const detailRoot = document.getElementById("artifact-workbench-detail");
const metadataRoot = document.getElementById("artifact-workbench-metadata");

if (artifactInput && refreshButton && statusNode && siblingsRoot && detailRoot && metadataRoot) {
  const state = {
    artifactId: "",
    detail: null,
    detailState: "idle",
    notice: "",
    noticeTone: "info",
    requestToken: 0,
  };

  const escapeHTML = (value) =>
    String(value ?? "")
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");

  const readArtifactID = () => {
    const searchParams = new URLSearchParams(window.location.search);
    return searchParams.get("artifact_id") || "";
  };

  const updateURLState = (artifactId) => {
    const searchParams = new URLSearchParams(window.location.search);
    if (artifactId) {
      searchParams.set("artifact_id", artifactId);
    } else {
      searchParams.delete("artifact_id");
    }

    const query = searchParams.toString();
    const nextURL = query ? `${window.location.pathname}?${query}` : window.location.pathname;
    window.history.replaceState({}, "", nextURL);
  };

  const setStatus = (message, tone = "info") => {
    statusNode.textContent = message;
    statusNode.dataset.tone = tone;
  };

  const isPreviewableArtifact = (detail) => {
    const contentType = detail.content_type || "";
    return (
      (detail.content_type && detail.content_type.startsWith("text/")) ||
      contentType === "application/json" ||
      contentType === "application/xml" ||
      contentType === "application/x-yaml"
    );
  };

  const siblingWorkbenchURL = (sibling) =>
    sibling.artifact_id
      ? `/board/artifacts/workbench?artifact_id=${encodeURIComponent(sibling.artifact_id)}`
      : "";

  const renderSiblings = () => {
    if (state.detailState === "idle") {
      siblingsRoot.innerHTML = '<p class="empty-state">Enter an artifact_id to load same-run sibling artifacts.</p>';
      return;
    }

    if (state.detailState === "loading") {
      siblingsRoot.innerHTML = '<p class="empty-state">Loading sibling artifacts...</p>';
      return;
    }

    if (state.detailState === "not_found") {
      siblingsRoot.innerHTML = '<p class="empty-state">Sibling artifacts unavailable.</p>';
      return;
    }

    if (state.detailState === "conflict") {
      siblingsRoot.innerHTML = '<p class="empty-state">Legacy artifacts do not expose same-run sibling navigation yet.</p>';
      return;
    }

    if (state.detailState === "error") {
      siblingsRoot.innerHTML = '<p class="empty-state">Sibling artifacts unavailable.</p>';
      return;
    }

    const detail = state.detail;
    if (!detail.siblings || !detail.siblings.length) {
      siblingsRoot.innerHTML = '<p class="empty-state">No same-run sibling artifacts are available for this artifact.</p>';
      return;
    }

    siblingsRoot.innerHTML = `
      <div class="artifact-sibling-list">
        ${detail.siblings
          .map((sibling) => {
            const siblingURL = siblingWorkbenchURL(sibling);
            const selectedClass = sibling.selected ? " is-selected" : "";
            const title = sibling.summary || sibling.artifact_id || "Artifact";
            return `
              <a class="artifact-sibling-item${selectedClass}" href="${escapeHTML(siblingURL)}">
                <p class="artifact-kind">${escapeHTML(sibling.kind || sibling.artifact_id || "artifact")}</p>
                <strong>${escapeHTML(title)}</strong>
                <p class="detail-copy">${escapeHTML(sibling.artifact_id)}</p>
              </a>
            `;
          })
          .join("")}
      </div>
    `;
  };

  const renderDetail = () => {
    if (state.detailState === "idle") {
      detailRoot.innerHTML = '<p class="empty-state">Enter an artifact_id to load artifact detail.</p>';
      metadataRoot.innerHTML = '<p class="empty-state">Artifact metadata will appear here.</p>';
      return;
    }

    if (state.detailState === "loading") {
      detailRoot.innerHTML = '<p class="empty-state">Loading artifact detail...</p>';
      metadataRoot.innerHTML = '<p class="empty-state">Loading artifact metadata...</p>';
      return;
    }

    if (state.detailState === "not_found") {
      detailRoot.innerHTML = `
        <article class="approval-detail-card">
          <p class="detail-title">Artifact not found</p>
          <p class="detail-copy">No artifact with ID <code>${escapeHTML(state.artifactId)}</code> exists.</p>
        </article>
      `;
      metadataRoot.innerHTML = '<p class="empty-state">Artifact metadata unavailable.</p>';
      return;
    }

    if (state.detailState === "conflict") {
      detailRoot.innerHTML = `
        <article class="approval-detail-card">
          <p class="detail-title">Artifact is not linked to one exact run</p>
          <p class="detail-copy">${escapeHTML(state.notice || "Refresh from the run workbench and choose a newer linked artifact.")}</p>
        </article>
      `;
      metadataRoot.innerHTML = '<p class="empty-state">Artifact metadata unavailable.</p>';
      return;
    }

    if (state.detailState === "error") {
      detailRoot.innerHTML = `
        <article class="approval-detail-card">
          <p class="detail-title">Unable to load artifact detail</p>
          <p class="detail-copy">${escapeHTML(state.notice || "Refresh and try again.")}</p>
        </article>
      `;
      metadataRoot.innerHTML = '<p class="empty-state">Artifact metadata unavailable.</p>';
      return;
    }

    const detail = state.detail;
    const rawContentLink = detail.raw_content_url
      ? `<a class="board-link" href="${escapeHTML(detail.raw_content_url)}" target="_blank" rel="noopener noreferrer">Open raw artifact</a>`
      : '<span class="board-link board-link-disabled" aria-disabled="true">Raw artifact unavailable</span>';
    const previewContent = detail.preview ?? "";
    const previewMarkup = isPreviewableArtifact(detail)
      ? `
        <section class="detail-block detail-block-wide">
          <p class="detail-label">Bounded Preview</p>
          <pre class="artifact-preview">${escapeHTML(previewContent)}</pre>
          ${detail.preview_truncated ? '<p class="detail-copy">Preview truncated to the workbench preview limit.</p>' : ""}
        </section>
      `
      : `
        <section class="detail-block detail-block-wide">
          <p class="detail-label">Preview</p>
          <p class="detail-copy">Inline preview is unavailable for this artifact type. Use the raw artifact link for the original content.</p>
          ${rawContentLink}
        </section>
      `;

    detailRoot.innerHTML = `
      <article class="approval-detail-card">
        <header class="approval-detail-header">
          <div>
            <p class="panel-kicker">${escapeHTML(detail.kind || "artifact")}</p>
            <h3>${escapeHTML(detail.summary || detail.path || detail.artifact_id)}</h3>
          </div>
          <div class="approval-detail-badges">
            <span class="detail-pill">${escapeHTML(detail.run_id || "run unknown")}</span>
          </div>
        </header>

        <section class="detail-block detail-block-wide">
          <p class="detail-label">Artifact Summary</p>
          <p class="detail-copy">${escapeHTML(detail.summary || "No summary recorded.")}</p>
        </section>

        ${previewMarkup}
      </article>
    `;

    const runWorkbenchLink = detail.run_workbench_url
      ? `<a class="board-link" href="${escapeHTML(detail.run_workbench_url)}">Back to run workbench</a>`
      : '<span class="board-link board-link-disabled" aria-disabled="true">Run workbench unavailable</span>';

    metadataRoot.innerHTML = `
      <article class="approval-detail-card">
        <section class="detail-grid detail-grid-secondary artifact-metadata-grid">
          <article class="detail-block">
            <p class="detail-label">Artifact ID</p>
            <p class="detail-copy">${escapeHTML(detail.artifact_id)}</p>
          </article>

          <article class="detail-block">
            <p class="detail-label">Run ID</p>
            <p class="detail-copy">${escapeHTML(detail.run_id || "Not recorded")}</p>
          </article>

          <article class="detail-block">
            <p class="detail-label">Task ID</p>
            <p class="detail-copy">${escapeHTML(detail.task_id || "Not recorded")}</p>
          </article>

          <article class="detail-block">
            <p class="detail-label">Project ID</p>
            <p class="detail-copy">${escapeHTML(detail.project_id || "Not recorded")}</p>
          </article>

          <article class="detail-block">
            <p class="detail-label">Module ID</p>
            <p class="detail-copy">${escapeHTML(detail.module_id || "Not recorded")}</p>
          </article>

          <article class="detail-block">
            <p class="detail-label">Content Type</p>
            <p class="detail-copy">${escapeHTML(detail.content_type || "Unknown")}</p>
          </article>

          <article class="detail-block detail-block-wide artifact-metadata-path">
            <p class="detail-label">Path</p>
            <p class="detail-copy">${escapeHTML(detail.path || "Artifact path not recorded")}</p>
          </article>
        </section>

        <section class="artifact-metadata-actions">
          ${runWorkbenchLink}
          ${rawContentLink}
        </section>
      </article>
    `;
  };

  const renderWorkbench = () => {
    renderSiblings();
    renderDetail();
  };

  const fetchJSON = async (url, options) => {
    const response = await fetch(url, options);
    if (response.status === 404) {
      return { notFound: true };
    }
    if (response.status === 409) {
      const payload = await response.json();
      return { conflict: true, message: payload.error || "Artifact is not linked to one exact run." };
    }
    if (!response.ok) {
      const payload = await response.json();
      throw new Error(payload.error || `Request failed with status ${response.status}`);
    }
    return response.json();
  };

  const loadArtifactDetail = async () => {
    const requestedArtifactId = state.artifactId;
    const requestToken = ++state.requestToken;

    if (!requestedArtifactId) {
      state.detail = null;
      state.detailState = "idle";
      state.notice = "";
      updateURLState("");
      renderWorkbench();
      setStatus("Enter an artifact_id to load artifact detail.");
      return;
    }

    state.detailState = "loading";
    state.notice = "";
    updateURLState(requestedArtifactId);
    renderWorkbench();
    setStatus(`Loading ${requestedArtifactId}...`);

    try {
      const detail = await fetchJSON(`/api/manager/artifacts/${encodeURIComponent(requestedArtifactId)}/workbench`, {
        method: "GET",
      });

      if (requestToken !== state.requestToken) {
        return;
      }

      if (detail.notFound) {
        state.detail = null;
        state.detailState = "not_found";
        renderWorkbench();
        setStatus(`Artifact ${requestedArtifactId} not found.`, "danger");
        return;
      }

      if (detail.conflict) {
        state.detail = null;
        state.detailState = "conflict";
        state.notice = detail.message || "Artifact is not linked to one exact run.";
        renderWorkbench();
        setStatus(`Artifact ${requestedArtifactId} is not linked to one exact run.`, "warning");
        return;
      }

      state.detail = detail;
      state.detailState = "ready";
      renderWorkbench();
      setStatus(`Loaded ${requestedArtifactId}.`);
    } catch (error) {
      if (requestToken !== state.requestToken) {
        return;
      }

      console.error(error);
      state.detail = null;
      state.detailState = "error";
      state.notice = error.message || "Failed to load artifact workbench.";
      state.noticeTone = "danger";
      renderWorkbench();
      setStatus(`Failed to load ${requestedArtifactId}.`, "danger");
    }
  };

  const refreshWorkbench = async () => {
    state.artifactId = artifactInput.value.trim();
    artifactInput.value = state.artifactId;
    state.noticeTone = "info";
    await loadArtifactDetail();
  };

  refreshButton.addEventListener("click", refreshWorkbench);
  artifactInput.addEventListener("keydown", (event) => {
    if (event.key === "Enter") {
      refreshWorkbench();
    }
  });
  window.addEventListener("popstate", () => {
    artifactInput.value = readArtifactID();
    refreshWorkbench();
  });

  state.artifactId = readArtifactID();
  artifactInput.value = state.artifactId;
  refreshWorkbench();
}
