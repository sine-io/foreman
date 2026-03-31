(function (globalScope) {
  const documentRef = globalScope.document || globalThis.document;
  const previewTruncationFallback = "Preview truncated to the workbench preview limit.";

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
  const currentRenderers = () => globalScope.ForemanArtifactRenderers || globalThis.ForemanArtifactRenderers;
  const currentLogErgonomics = () =>
    globalScope.ForemanArtifactLogErgonomics || globalThis.ForemanArtifactLogErgonomics;

  const isPreviewableArtifact = (detail) => {
    const contentType = normalizeText(detail && detail.content_type);
    return (
      contentType.startsWith("text/") ||
      contentType === "application/json" ||
      contentType === "application/xml" ||
      contentType === "application/x-yaml"
    );
  };

  const resolveTruncationNotice = (detail, renderers) => {
    if (!detail || !detail.preview_truncated) {
      return "";
    }

    if (renderers && typeof renderers.truncatedNotice === "string" && renderers.truncatedNotice) {
      return renderers.truncatedNotice;
    }

    return previewTruncationFallback;
  };

  const buildRawContentLinkMarkup = (detail) =>
    detail && detail.raw_content_url
      ? `<a class="board-link" href="${escapeHTML(detail.raw_content_url)}" target="_blank" rel="noopener noreferrer">Open raw artifact</a>`
      : '<span class="board-link board-link-disabled" aria-disabled="true">Raw artifact unavailable</span>';

  const wrapPreviewSection = (label, bodyMarkup) => `
    <section class="detail-block detail-block-wide">
      <p class="detail-label">${escapeHTML(label)}</p>
      ${bodyMarkup}
    </section>
  `;

  const buildPreviewNoticeMarkup = (notice) =>
    notice ? `<p class="detail-copy artifact-preview-notice">${escapeHTML(notice)}</p>` : "";

  const buildTextPreviewMarkup = (previewText, options = {}) => {
    const extraClasses = options.extraClasses || "";
    const classSuffix = extraClasses ? ` ${extraClasses}` : "";
    return `
      <pre class="artifact-preview artifact-preview-text${classSuffix}">${escapeHTML(previewText)}</pre>
      ${buildPreviewNoticeMarkup(options.truncatedNotice || "")}
    `;
  };

  const buildMarkdownPreviewMarkup = (html, truncatedNotice) => `
    <div class="artifact-preview artifact-preview-rendered artifact-preview-markdown">${html}</div>
    ${buildPreviewNoticeMarkup(truncatedNotice)}
  `;

  const buildDiffPreviewMarkup = (lines, truncatedNotice) => `
    <div class="artifact-preview artifact-preview-rendered artifact-preview-diff">
      ${lines
        .map((line) => {
          const lineType = normalizeText((line && line.type) || "context");
          const lineText = normalizeText(line && line.text);
          return `<span class="artifact-preview-diff-line" data-diff-type="${escapeHTML(lineType)}">${escapeHTML(lineText)}</span>`;
        })
        .join("")}
    </div>
    ${buildPreviewNoticeMarkup(truncatedNotice)}
  `;

  const isStructuredRendererSuccess = (previewResult) => {
    if (!previewResult || typeof previewResult !== "object") {
      return false;
    }

    if (previewResult.output !== "text") {
      if (previewResult.output === "html" && typeof previewResult.html === "string") {
        return true;
      }
      return previewResult.output === "lines" && Array.isArray(previewResult.lines);
    }

    if (typeof previewResult.text !== "string") {
      return false;
    }

    return previewResult.renderer !== "text";
  };

  const normalizePreviewExpansion = (previewExpansion, previewModel) => {
    const canExpand = Boolean(
      (previewModel && previewModel.expansion && previewModel.expansion.canExpand) ||
        (previewModel && previewModel.teaser && previewModel.teaser.collapsed),
    );

    if (!previewExpansion || typeof previewExpansion !== "object") {
      return {
        canExpand,
        expanded: canExpand
          ? Boolean(previewModel && previewModel.expansion && previewModel.expansion.expanded)
          : true,
      };
    }

    return {
      canExpand,
      expanded: canExpand ? Boolean(previewExpansion.expanded) : true,
    };
  };

  const previewLineElementID = (lineNumber) => `artifact-preview-line-${lineNumber}`;

  const buildCollapsedTeaserCopy = (previewModel) => {
    const visibleLineCount =
      previewModel && previewModel.teaser && Array.isArray(previewModel.teaser.visibleLines)
        ? previewModel.teaser.visibleLines.length
        : 0;
    const totalLineCount = Array.isArray(previewModel && previewModel.lines) ? previewModel.lines.length : 0;
    const hiddenLineCount = Number(previewModel && previewModel.teaser && previewModel.teaser.hiddenLineCount) || 0;
    const hiddenCharacterCount =
      Number(previewModel && previewModel.teaser && previewModel.teaser.hiddenCharacterCount) || 0;

    if (hiddenLineCount > 0) {
      return `Showing first ${visibleLineCount} of ${totalLineCount} lines. Expand all to reveal ${hiddenLineCount} more lines from the bounded preview.`;
    }

    if (hiddenCharacterCount > 0) {
      return `Showing the first ${visibleLineCount} line. Expand all to reveal ${hiddenCharacterCount} more characters from the bounded preview.`;
    }

    return "Expand all to inspect the full bounded preview.";
  };

  const buildSummaryNavigationMarkup = (previewModel) => {
    const anchors =
      previewModel &&
      previewModel.navigation &&
      Array.isArray(previewModel.navigation.anchors)
        ? previewModel.navigation.anchors
        : [];

    if (!anchors.length) {
      return "";
    }

    return `
      <nav class="artifact-preview-summary-nav" aria-label="Preview navigation">
        ${previewModel.navigation.anchors
          .map((anchor) => {
            const lineNumber = escapeHTML(String(anchor.lineNumber || ""));
            return `
              <button
                type="button"
                class="board-button board-button-secondary artifact-preview-nav-button"
                data-artifact-preview-action="jump-to-line"
                data-line-number="${lineNumber}"
              >
                Line ${lineNumber} · ${escapeHTML(anchor.label || "")}
              </button>
            `;
          })
          .join("")}
      </nav>
    `;
  };

  const buildLogLineMarkup = (line, targetLineNumber) => {
    const lineNumber = Number.parseInt(line && line.lineNumber, 10);
    const lineText = normalizeText(line && line.text);
    const isTargeted = Number.isFinite(targetLineNumber) && targetLineNumber > 0 && lineNumber === targetLineNumber;
    const targetedClass = isTargeted ? " is-targeted" : "";
    return `
      <div class="artifact-preview-log-line${targetedClass}" id="${escapeHTML(previewLineElementID(lineNumber))}">
        <span class="artifact-preview-line-number">${escapeHTML(String(lineNumber))}</span>
        <span class="artifact-preview-line-text">${lineText ? escapeHTML(lineText) : "&nbsp;"}</span>
      </div>
    `;
  };

  const buildLongTextPreviewMarkup = (previewModel, options = {}) => {
    const visibleLines = previewModel.expansion.expanded ? previewModel.lines : previewModel.teaser.visibleLines;
    const teaserMarkup =
      previewModel.expansion.canExpand && !previewModel.expansion.expanded
        ? `<p class="detail-copy artifact-preview-teaser-note">${escapeHTML(buildCollapsedTeaserCopy(previewModel))}</p>`
        : "";
    const expandControlMarkup =
      previewModel.expansion.canExpand && !previewModel.expansion.expanded
        ? `
          <div class="artifact-preview-controls">
            <button
              type="button"
              class="board-button board-button-secondary artifact-preview-expand-button"
              data-artifact-preview-action="expand-all"
            >
              Expand all
            </button>
          </div>
        `
        : "";

    return `
      ${buildSummaryNavigationMarkup(previewModel)}
      ${teaserMarkup}
      ${expandControlMarkup}
      <div class="artifact-preview artifact-preview-log" data-expanded="${previewModel.expansion.expanded ? "true" : "false"}">
        ${visibleLines.map((line) => buildLogLineMarkup(line, options.targetLineNumber)).join("")}
      </div>
      ${buildPreviewNoticeMarkup(options.truncatedNotice || "")}
    `;
  };

  const buildArtifactPreviewViewModel = (detail, options = {}) => {
    const normalizedDetail = detail && typeof detail === "object" ? detail : {};
    const previewContent = normalizedDetail.preview ?? "";
    const rawContentLink = options.rawContentLinkMarkup || buildRawContentLinkMarkup(normalizedDetail);
    const renderers = options.renderers || currentRenderers();
    const logErgonomics = options.logErgonomics || currentLogErgonomics();
    const genericTruncationNotice = resolveTruncationNotice(normalizedDetail, renderers);
    const renderGenericPreview = (previewText = previewContent, previewResult = {}) => {
      const truncatedNotice =
        typeof previewResult.truncated_notice === "string" && previewResult.truncated_notice
          ? previewResult.truncated_notice
          : genericTruncationNotice;
      const extraClasses = previewResult.renderer === "json" ? "artifact-preview-json" : "";

      if (
        previewResult.output === "text" &&
        typeof previewResult.text === "string" &&
        previewResult.renderer === "text"
      ) {
        try {
          if (logErgonomics && typeof logErgonomics.buildLogErgonomicsModel === "function") {
            const previewModel = logErgonomics.buildLogErgonomicsModel(normalizedDetail, previewResult);
            if (previewModel && typeof previewModel === "object" && previewModel.eligible) {
              const numberedLines =
                Array.isArray(previewModel.lines) && previewModel.lines.length
                  ? previewModel.lines
                  : typeof logErgonomics.renderLineNumberedText === "function"
                    ? logErgonomics.renderLineNumberedText(previewText)
                    : [];
              const teaser =
                previewModel.teaser &&
                typeof previewModel.teaser === "object" &&
                Array.isArray(previewModel.teaser.visibleLines)
                  ? previewModel.teaser
                  : typeof logErgonomics.sliceCollapsedTeaser === "function"
                    ? logErgonomics.sliceCollapsedTeaser(numberedLines)
                    : {
                        collapsed: false,
                        hiddenCharacterCount: 0,
                        hiddenLineCount: 0,
                        visibleLines: numberedLines,
                      };
              const longTextPreviewModel = {
                ...previewModel,
                expansion: normalizePreviewExpansion(options.previewExpansion, {
                  ...previewModel,
                  teaser,
                }),
                lines: numberedLines,
                teaser,
              };

              return {
                markup: wrapPreviewSection(
                  "Bounded Preview",
                  buildLongTextPreviewMarkup(longTextPreviewModel, {
                    targetLineNumber: options.targetLineNumber,
                    truncatedNotice,
                  }),
                ),
                previewModel: longTextPreviewModel,
                previewResult,
              };
            }
          }
        } catch (_error) {
          // Fall through to the existing generic text rendering when ergonomics fail.
        }
      }

      return {
        markup: wrapPreviewSection(
          "Bounded Preview",
          buildTextPreviewMarkup(previewText, {
            extraClasses,
            truncatedNotice,
          }),
        ),
        previewModel: null,
        previewResult,
      };
    };

    if (!isPreviewableArtifact(normalizedDetail)) {
      return {
        markup: wrapPreviewSection(
          "Preview",
          `
            <p class="detail-copy">Inline preview is unavailable for this artifact type. Use the raw artifact link for the original content.</p>
            ${rawContentLink}
          `,
        ),
        previewModel: null,
        previewResult: null,
      };
    }

    try {
      if (!renderers || typeof renderers.renderPreview !== "function") {
        return renderGenericPreview();
      }

      const previewResult = renderers.renderPreview(normalizedDetail, previewContent);
      if (!previewResult || typeof previewResult !== "object") {
        return renderGenericPreview();
      }

      const truncatedNotice =
        typeof previewResult.truncated_notice === "string" && previewResult.truncated_notice
          ? previewResult.truncated_notice
          : genericTruncationNotice;

      if (isStructuredRendererSuccess(previewResult)) {
        if (previewResult.output === "html" && typeof previewResult.html === "string") {
          return {
            markup: wrapPreviewSection(
              "Bounded Preview",
              buildMarkdownPreviewMarkup(previewResult.html, truncatedNotice),
            ),
            previewModel: null,
            previewResult,
          };
        }

        if (previewResult.output === "lines" && Array.isArray(previewResult.lines)) {
          return {
            markup: wrapPreviewSection(
              "Bounded Preview",
              buildDiffPreviewMarkup(previewResult.lines, truncatedNotice),
            ),
            previewModel: null,
            previewResult,
          };
        }

        return {
          markup: wrapPreviewSection(
            "Bounded Preview",
            buildTextPreviewMarkup(previewResult.text, {
              extraClasses: previewResult.renderer === "json" ? "artifact-preview-json" : "",
              truncatedNotice,
            }),
          ),
          previewModel: null,
          previewResult,
        };
      }

      return renderGenericPreview(
        typeof previewResult.text === "string" ? previewResult.text : previewContent,
        previewResult,
      );
    } catch (_error) {
      return renderGenericPreview();
    }
  };

  const composeArtifactPreviewMarkup = (detail, options = {}) =>
    buildArtifactPreviewViewModel(detail, options).markup;

  const api = {
    composeArtifactPreviewMarkup,
  };

  globalScope.ForemanArtifactWorkbench = api;

  if (typeof module !== "undefined" && module.exports) {
    module.exports = api;
  }

  if (!documentRef) {
    return;
  }

  const artifactInput = documentRef.getElementById("artifact-workbench-artifact-id");
  const refreshButton = documentRef.getElementById("artifact-workbench-refresh");
  const statusNode = documentRef.getElementById("artifact-workbench-status");
  const siblingsRoot = documentRef.getElementById("artifact-workbench-siblings");
  const detailRoot = documentRef.getElementById("artifact-workbench-detail");
  const metadataRoot = documentRef.getElementById("artifact-workbench-metadata");

  if (!artifactInput || !refreshButton || !statusNode || !siblingsRoot || !detailRoot || !metadataRoot) {
    return;
  }

  const state = {
    artifactId: "",
    detail: null,
    detailState: "idle",
    notice: "",
    noticeTone: "info",
    previewExpansion: null,
    previewTargetLineNumber: 0,
    requestToken: 0,
  };

  const readArtifactID = () => {
    const searchParams = new URLSearchParams(currentLocation().search);
    return searchParams.get("artifact_id") || "";
  };

  const updateURLState = (artifactId) => {
    const searchParams = new URLSearchParams(currentLocation().search);
    if (artifactId) {
      searchParams.set("artifact_id", artifactId);
    } else {
      searchParams.delete("artifact_id");
    }

    const query = searchParams.toString();
    const nextURL = query ? `${currentLocation().pathname}?${query}` : currentLocation().pathname;
    const historyRef = currentHistory();
    if (historyRef && typeof historyRef.replaceState === "function") {
      historyRef.replaceState({}, "", nextURL);
    }
  };

  const setStatus = (message, tone = "info") => {
    statusNode.textContent = message;
    statusNode.dataset.tone = tone;
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
    const rawContentLink = buildRawContentLinkMarkup(detail);
    const previewViewModel = buildArtifactPreviewViewModel(detail, {
      rawContentLinkMarkup: rawContentLink,
      renderers: currentRenderers(),
      logErgonomics: currentLogErgonomics(),
      previewExpansion: state.previewExpansion,
      targetLineNumber: state.previewTargetLineNumber,
    });
    const previewMarkup = previewViewModel.markup;
    state.previewExpansion = previewViewModel.previewModel ? previewViewModel.previewModel.expansion : null;
    if (!previewViewModel.previewModel) {
      state.previewTargetLineNumber = 0;
    }

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

  const parseLineNumber = (value) => {
    const parsed = Number.parseInt(value, 10);
    return Number.isFinite(parsed) && parsed > 0 ? parsed : 0;
  };

  const scrollToPreviewLine = (lineNumber) => {
    const parsedLineNumber = parseLineNumber(lineNumber);
    if (!parsedLineNumber) {
      return;
    }

    const targetNode = documentRef.getElementById(previewLineElementID(parsedLineNumber));
    if (!targetNode || typeof targetNode.scrollIntoView !== "function") {
      return;
    }

    const requestAnimationFrameRef =
      globalScope.requestAnimationFrame || globalThis.requestAnimationFrame || null;
    if (typeof requestAnimationFrameRef === "function") {
      requestAnimationFrameRef(() => targetNode.scrollIntoView({ block: "center" }));
      return;
    }

    targetNode.scrollIntoView({ block: "center" });
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
        state.previewExpansion = null;
        state.previewTargetLineNumber = 0;
        renderWorkbench();
        setStatus(`Artifact ${requestedArtifactId} not found.`, "danger");
        return;
      }

      if (detail.conflict) {
        state.detail = null;
        state.detailState = "conflict";
        state.notice = detail.message || "Artifact is not linked to one exact run.";
        state.previewExpansion = null;
        state.previewTargetLineNumber = 0;
        renderWorkbench();
        setStatus(`Artifact ${requestedArtifactId} is not linked to one exact run.`, "warning");
        return;
      }

      state.detail = detail;
      state.detailState = "ready";
      state.previewExpansion = null;
      state.previewTargetLineNumber = 0;
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
      state.previewExpansion = null;
      state.previewTargetLineNumber = 0;
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
  detailRoot.addEventListener("click", (event) => {
    const target = event && event.target;
    const actionNode =
      target && typeof target.closest === "function"
        ? target.closest("[data-artifact-preview-action]")
        : null;

    if (!actionNode || !actionNode.dataset) {
      return;
    }

    if (typeof event.preventDefault === "function") {
      event.preventDefault();
    }

    const action = actionNode.dataset.artifactPreviewAction;
    if (action === "expand-all" && state.previewExpansion && state.previewExpansion.canExpand) {
      const logErgonomics = currentLogErgonomics();
      state.previewExpansion =
        logErgonomics && typeof logErgonomics.expandAllState === "function"
          ? logErgonomics.expandAllState(state.previewExpansion)
          : {
              ...state.previewExpansion,
              expanded: true,
            };
      renderDetail();
      return;
    }

    if (action === "jump-to-line") {
      const lineNumber = parseLineNumber(actionNode.dataset.lineNumber);
      if (!lineNumber) {
        return;
      }

      state.previewTargetLineNumber = lineNumber;
      if (state.previewExpansion && state.previewExpansion.canExpand && !state.previewExpansion.expanded) {
        const logErgonomics = currentLogErgonomics();
        state.previewExpansion =
          logErgonomics && typeof logErgonomics.expandAllState === "function"
            ? logErgonomics.expandAllState(state.previewExpansion)
            : {
                ...state.previewExpansion,
                expanded: true,
              };
      }
      renderDetail();
      scrollToPreviewLine(lineNumber);
    }
  });

  if (typeof globalScope.addEventListener === "function") {
    globalScope.addEventListener("popstate", () => {
      artifactInput.value = readArtifactID();
      refreshWorkbench();
    });
  }

  state.artifactId = readArtifactID();
  artifactInput.value = state.artifactId;
  refreshWorkbench();
})(typeof window !== "undefined" ? window : globalThis);
