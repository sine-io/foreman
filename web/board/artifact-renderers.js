(function (globalScope) {
  const TRUNCATION_NOTICE = "Preview truncated to the workbench preview limit.";

  const normalizeText = (value) => String(value ?? "");

  const escapeHTML = (value) =>
    normalizeText(value)
      .replaceAll("&", "&amp;")
      .replaceAll("<", "&lt;")
      .replaceAll(">", "&gt;")
      .replaceAll('"', "&quot;")
      .replaceAll("'", "&#39;");

  const normalizePreview = (previewText, detail) =>
    normalizeText(previewText === undefined ? detail.preview : previewText).replace(/\r\n?/g, "\n");

  const normalizeDetail = (detail) => (detail && typeof detail === "object" ? detail : {});

  const contentTypeFor = (detail) => normalizeText(detail.content_type).toLowerCase();
  const kindFor = (detail) => normalizeText(detail.kind).toLowerCase();
  const pathFor = (detail) => normalizeText(detail.path).toLowerCase();

  const isJSONArtifact = (detail) => {
    const contentType = contentTypeFor(detail);
    const kind = kindFor(detail);
    const path = pathFor(detail);
    return contentType.includes("json") || kind.includes("json") || path.endsWith(".json");
  };

  const isMarkdownArtifact = (detail) => {
    const contentType = contentTypeFor(detail);
    const kind = kindFor(detail);
    const path = pathFor(detail);
    return (
      contentType === "text/markdown" ||
      contentType.includes("markdown") ||
      kind.includes("markdown") ||
      path.endsWith(".md") ||
      path.endsWith(".markdown")
    );
  };

  const isDiffArtifact = (detail) => {
    const contentType = contentTypeFor(detail);
    const kind = kindFor(detail);
    const path = pathFor(detail);
    return (
      contentType.includes("diff") ||
      contentType.includes("patch") ||
      kind.includes("diff") ||
      kind.includes("patch") ||
      path.endsWith(".diff") ||
      path.endsWith(".patch")
    );
  };

  const detectPreviewRenderer = (detail) => {
    const normalizedDetail = normalizeDetail(detail);
    if (isJSONArtifact(normalizedDetail)) {
      return "json";
    }
    if (isMarkdownArtifact(normalizedDetail)) {
      return "markdown";
    }
    if (isDiffArtifact(normalizedDetail)) {
      return "diff";
    }
    return "text";
  };

  const createBaseResult = (detail, previewText) => {
    const normalizedDetail = normalizeDetail(detail);
    const text = normalizePreview(previewText, normalizedDetail);
    const preview_truncated = Boolean(normalizedDetail.preview_truncated);
    return {
      renderer: "text",
      output: "text",
      text,
      html: "",
      lines: [],
      fallback: false,
      fallback_reason: "",
      attempted_renderer: "",
      truncated: preview_truncated,
      preview_truncated,
      truncated_notice: preview_truncated ? TRUNCATION_NOTICE : "",
    };
  };

  const buildFallbackResult = (baseResult, attemptedRenderer, fallbackReason) => ({
    ...baseResult,
    renderer: "text",
    output: "text",
    html: "",
    lines: [],
    fallback: Boolean(attemptedRenderer),
    fallback_reason: attemptedRenderer ? fallbackReason : "",
    attempted_renderer: attemptedRenderer || "",
  });

  const renderInlineMarkdown = (value) => {
    const rawValue = normalizeText(value);
    const segments = rawValue.split(/(`[^`]+`)/g);
    return segments
      .map((segment) => {
        if (segment.startsWith("`") && segment.endsWith("`") && segment.length >= 2) {
          return `<code>${escapeHTML(segment.slice(1, -1))}</code>`;
        }
        return escapeHTML(segment);
      })
      .join("");
  };

  const renderParagraph = (lines) => `<p>${renderInlineMarkdown(lines.join(" "))}</p>`;

  const renderMarkdownPreview = (previewText) => {
    const lines = normalizeText(previewText).replace(/\r\n?/g, "\n").split("\n");
    const parts = [];
    let index = 0;

    while (index < lines.length) {
      const line = lines[index];

      if (/^```/.test(line)) {
        index += 1;
        const codeLines = [];
        while (index < lines.length && !/^```/.test(lines[index])) {
          codeLines.push(lines[index]);
          index += 1;
        }
        if (index < lines.length && /^```/.test(lines[index])) {
          index += 1;
        }
        parts.push(`<pre><code>${escapeHTML(codeLines.join("\n"))}</code></pre>`);
        continue;
      }

      if (/^\s*$/.test(line)) {
        index += 1;
        continue;
      }

      const headingMatch = line.match(/^(#{1,6})\s+(.*)$/);
      if (headingMatch) {
        const level = headingMatch[1].length;
        parts.push(`<h${level}>${renderInlineMarkdown(headingMatch[2])}</h${level}>`);
        index += 1;
        continue;
      }

      if (/^>\s?/.test(line)) {
        const quoteLines = [];
        while (index < lines.length && /^>\s?/.test(lines[index])) {
          quoteLines.push(lines[index].replace(/^>\s?/, ""));
          index += 1;
        }
        parts.push(`<blockquote>${quoteLines.map((quoteLine) => `<p>${renderInlineMarkdown(quoteLine)}</p>`).join("")}</blockquote>`);
        continue;
      }

      if (/^\s*[-*]\s+/.test(line)) {
        const items = [];
        while (index < lines.length && /^\s*[-*]\s+/.test(lines[index])) {
          items.push(lines[index].replace(/^\s*[-*]\s+/, ""));
          index += 1;
        }
        parts.push(`<ul>${items.map((item) => `<li>${renderInlineMarkdown(item)}</li>`).join("")}</ul>`);
        continue;
      }

      const paragraphLines = [];
      while (
        index < lines.length &&
        !/^\s*$/.test(lines[index]) &&
        !/^```/.test(lines[index]) &&
        !/^(#{1,6})\s+/.test(lines[index]) &&
        !/^>\s?/.test(lines[index]) &&
        !/^\s*[-*]\s+/.test(lines[index])
      ) {
        paragraphLines.push(lines[index]);
        index += 1;
      }
      parts.push(renderParagraph(paragraphLines));
    }

    return parts.join("\n");
  };

  const classifyDiffLine = (line) => {
    if (line.startsWith("@@")) {
      return "hunk";
    }
    if (
      line.startsWith("diff ") ||
      line.startsWith("index ") ||
      line.startsWith("Index: ") ||
      line.startsWith("*** ") ||
      line.startsWith("--- ") ||
      line.startsWith("+++ ")
    ) {
      return "meta";
    }
    if (line.startsWith("+") && !line.startsWith("+++ ")) {
      return "add";
    }
    if (line.startsWith("-") && !line.startsWith("--- ")) {
      return "remove";
    }
    return "context";
  };

  const renderDiffPreview = (previewText) =>
    normalizeText(previewText)
      .replace(/\r\n?/g, "\n")
      .split("\n")
      .map((line) => ({
        type: classifyDiffLine(line),
        text: line,
      }));

  const renderJSONPreview = (baseResult) => {
    if (baseResult.preview_truncated) {
      return buildFallbackResult(baseResult, "json", "truncated_preview");
    }

    try {
      const parsedPreview = JSON.parse(baseResult.text);
      return {
        ...baseResult,
        renderer: "json",
        output: "text",
        text: JSON.stringify(parsedPreview, null, 2),
        attempted_renderer: "json",
      };
    } catch (_error) {
      return buildFallbackResult(baseResult, "json", "parse_failure");
    }
  };

  const renderMarkdownResult = (baseResult) => {
    try {
      return {
        ...baseResult,
        renderer: "markdown",
        output: "html",
        html: renderMarkdownPreview(baseResult.text),
        attempted_renderer: "markdown",
      };
    } catch (_error) {
      return buildFallbackResult(baseResult, "markdown", "render_failure");
    }
  };

  const renderDiffResult = (baseResult) => {
    try {
      return {
        ...baseResult,
        renderer: "diff",
        output: "lines",
        lines: renderDiffPreview(baseResult.text),
        attempted_renderer: "diff",
      };
    } catch (_error) {
      return buildFallbackResult(baseResult, "diff", "render_failure");
    }
  };

  const renderPreview = (detail, previewText) => {
    const normalizedDetail = normalizeDetail(detail);
    const baseResult = createBaseResult(normalizedDetail, previewText);

    try {
      switch (detectPreviewRenderer(normalizedDetail)) {
        case "json":
          return renderJSONPreview(baseResult);
        case "markdown":
          return renderMarkdownResult(baseResult);
        case "diff":
          return renderDiffResult(baseResult);
        default:
          return baseResult;
      }
    } catch (_error) {
      return buildFallbackResult(baseResult, "", "");
    }
  };

  const api = {
    detectPreviewRenderer,
    escapeHTML,
    renderPreview,
    truncatedNotice: TRUNCATION_NOTICE,
  };

  globalScope.ForemanArtifactRenderers = api;

  if (typeof module !== "undefined" && module.exports) {
    module.exports = api;
  }
})(typeof window !== "undefined" ? window : globalThis);
