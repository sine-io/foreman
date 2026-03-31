(function (globalScope) {
  const DEFAULT_MAX_ANCHORS = 5;
  const DEFAULT_MIN_LONG_TEXT_CHARACTERS = 600;
  const DEFAULT_MIN_LONG_TEXT_LINES = 12;
  const DEFAULT_TEASER_CHARACTERS = 400;
  const DEFAULT_TEASER_LINE_COUNT = 8;
  const ERROR_KEYWORDS = [
    "error",
    "failed",
    "failed to connect",
    "failure",
    "fatal",
    "exception",
    "timeout",
  ];

  const normalizeText = (value) => String(value ?? "");
  const normalizeDetail = (detail) => (detail && typeof detail === "object" ? detail : {});
  const normalizePreviewResult = (previewResult) =>
    previewResult && typeof previewResult === "object" ? previewResult : {};

  const normalizeNewlines = (value) => normalizeText(value).replace(/\r\n?/g, "\n");

  const safeInteger = (value, fallback) => {
    const parsed = Number.parseInt(value, 10);
    return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
  };

  const previewTextFor = (detail, previewResult) => {
    const normalizedDetail = normalizeDetail(detail);
    const normalizedPreviewResult = normalizePreviewResult(previewResult);
    if (typeof normalizedPreviewResult.text === "string") {
      return normalizeNewlines(normalizedPreviewResult.text);
    }
    return normalizeNewlines(normalizedDetail.preview);
  };

  const hasValidPreviewResultContract = (previewResult) => {
    if (!previewResult || typeof previewResult !== "object") {
      return false;
    }
    if (typeof previewResult.renderer !== "string" || typeof previewResult.output !== "string") {
      return false;
    }
    switch (previewResult.output) {
      case "text":
        return typeof previewResult.text === "string";
      case "html":
        return typeof previewResult.html === "string";
      case "lines":
        return Array.isArray(previewResult.lines);
      default:
        return false;
    }
  };

  const isGenericTextPath = (previewResult) => {
    if (!hasValidPreviewResultContract(previewResult)) {
      return false;
    }
    if (previewResult.renderer !== "text") {
      return false;
    }
    if (previewResult.output !== "text") {
      return false;
    }
    return true;
  };

  const isTextLikeArtifact = (detail) => {
    const contentType = normalizeText(normalizeDetail(detail).content_type).toLowerCase();
    if (contentType.startsWith("text/")) {
      return true;
    }
    if (
      contentType === "application/json" ||
      contentType === "application/xml" ||
      contentType === "application/x-yaml"
    ) {
      return true;
    }
    return false;
  };

  const renderLineNumberedTextUnsafe = (previewText) =>
    normalizeNewlines(previewText).split("\n").map((line, index) => ({
      lineNumber: index + 1,
      text: line,
    }));

  const renderLineNumberedText = (previewText) => {
    try {
      return renderLineNumberedTextUnsafe(previewText);
    } catch (_error) {
      return [];
    }
  };

  const isLongTextPreviewUnsafe = (detail, previewResult, options = {}) => {
    if (!isGenericTextPath(previewResult)) {
      return false;
    }

    const previewText = previewTextFor(detail, previewResult);
    if (!isTextLikeArtifact(detail, previewResult, previewText)) {
      return false;
    }

    const lines = renderLineNumberedTextUnsafe(previewText);
    const minLines = safeInteger(options.minimumLongTextLines, DEFAULT_MIN_LONG_TEXT_LINES);
    const minCharacters = safeInteger(
      options.minimumLongTextCharacters,
      DEFAULT_MIN_LONG_TEXT_CHARACTERS,
    );
    return lines.length >= minLines || previewText.length >= minCharacters;
  };

  const isLongTextPreview = (detail, previewResult, options = {}) => {
    try {
      return isLongTextPreviewUnsafe(detail, previewResult, options);
    } catch (_error) {
      return false;
    }
  };

  const sliceCollapsedTeaserUnsafe = (lines, options = {}) => {
    const normalizedLines = Array.isArray(lines) ? lines.slice() : renderLineNumberedTextUnsafe(lines);
    const teaserLineCount = safeInteger(options.teaserLineCount, DEFAULT_TEASER_LINE_COUNT);
    const teaserCharacters = safeInteger(options.teaserCharacters, DEFAULT_TEASER_CHARACTERS);
    let visibleLines = normalizedLines.slice(0, teaserLineCount);
    let hiddenLineCount = Math.max(normalizedLines.length - visibleLines.length, 0);
    let hiddenCharacterCount = 0;

    if (hiddenLineCount === 0) {
      const totalCharacterCount = normalizedLines.reduce(
        (sum, line) => sum + normalizeText(line && line.text).length,
        0,
      );

      if (totalCharacterCount > teaserCharacters) {
        let remainingCharacters = teaserCharacters;
        const characterVisibleLines = [];

        normalizedLines.forEach((line) => {
          const lineText = normalizeText(line && line.text);
          if (remainingCharacters <= 0) {
            hiddenCharacterCount += lineText.length;
            return;
          }

          if (lineText.length <= remainingCharacters) {
            characterVisibleLines.push({
              ...line,
              text: lineText,
            });
            remainingCharacters -= lineText.length;
            return;
          }

          characterVisibleLines.push({
            ...line,
            text: lineText.slice(0, remainingCharacters),
          });
          hiddenCharacterCount += lineText.length - remainingCharacters;
          remainingCharacters = 0;
        });

        if (hiddenCharacterCount > 0) {
          visibleLines = characterVisibleLines;
          hiddenLineCount = Math.max(normalizedLines.length - visibleLines.length, 0);
        }
      }
    }

    return {
      collapsed: hiddenLineCount > 0 || hiddenCharacterCount > 0,
      hiddenCharacterCount,
      hiddenLineCount,
      visibleLines,
    };
  };

  const sliceCollapsedTeaser = (lines, options = {}) => {
    try {
      return sliceCollapsedTeaserUnsafe(lines, options);
    } catch (_error) {
      return {
        collapsed: false,
        hiddenCharacterCount: 0,
        hiddenLineCount: 0,
        visibleLines: [],
      };
    }
  };

  const createExpansionStateUnsafe = (options = {}) => {
    const canExpand = Boolean(options.canExpand ?? options.eligible);
    return {
      canExpand,
      expanded: !canExpand,
    };
  };

  const createExpansionState = (options = {}) => {
    try {
      return createExpansionStateUnsafe(options);
    } catch (_error) {
      return {
        canExpand: false,
        expanded: true,
      };
    }
  };

  const expandAllState = (state) => {
    try {
      const normalizedState = state && typeof state === "object" ? state : {};
      const canExpand = Boolean(normalizedState.canExpand);
      return {
        canExpand,
        expanded: canExpand ? true : Boolean(normalizedState.expanded),
      };
    } catch (_error) {
      return {
        canExpand: false,
        expanded: true,
      };
    }
  };

  const truncateLabel = (value) => {
    const label = normalizeText(value).trim().replace(/\s+/g, " ");
    return label.length > 80 ? `${label.slice(0, 77)}...` : label;
  };

  const summaryCandidatesFor = (summaryText) =>
    normalizeText(summaryText)
      .split(/\n|;|\|/)
      .map((segment) => segment.trim())
      .filter((segment) => segment.length >= 6);

  const matchesErrorKeyword = (lineText) => {
    const normalizedLine = normalizeText(lineText).toLowerCase();
    return ERROR_KEYWORDS.some((keyword) => normalizedLine.includes(keyword));
  };

  const extractSummaryAnchorsUnsafe = (summaryText, previewText, options = {}) => {
    const lines = renderLineNumberedTextUnsafe(previewText);
    const maxAnchors = safeInteger(options.maxAnchors, DEFAULT_MAX_ANCHORS);
    const anchors = [];
    const seenLineNumbers = new Set();

    const addAnchor = (lineNumber, label, source) => {
      if (!Number.isFinite(lineNumber) || lineNumber <= 0 || seenLineNumbers.has(lineNumber)) {
        return;
      }
      const normalizedLabel = truncateLabel(label);
      if (!normalizedLabel) {
        return;
      }
      seenLineNumbers.add(lineNumber);
      anchors.push({ label: normalizedLabel, lineNumber, source });
    };

    const summaryCandidates = summaryCandidatesFor(summaryText).map((candidate) => candidate.toLowerCase());
    summaryCandidates.forEach((candidate) => {
      const line = lines.find((entry) => normalizeText(entry.text).toLowerCase().includes(candidate));
      if (line) {
        addAnchor(line.lineNumber, line.text, "summary");
      }
    });

    lines.forEach((line) => {
      const trimmedText = normalizeText(line.text).trim();
      if (!trimmedText) {
        return;
      }
      if (/^\$ /.test(trimmedText)) {
        addAnchor(line.lineNumber, trimmedText, "command");
        return;
      }
      if (/^@@/.test(trimmedText)) {
        addAnchor(line.lineNumber, trimmedText, "hunk");
        return;
      }
      if (/^#{1,6}\s+/.test(trimmedText)) {
        addAnchor(line.lineNumber, trimmedText, "heading");
        return;
      }
      if (matchesErrorKeyword(trimmedText)) {
        addAnchor(line.lineNumber, trimmedText, "error");
      }
    });

    anchors.sort((left, right) => left.lineNumber - right.lineNumber);
    return {
      anchors: anchors.slice(0, maxAnchors),
    };
  };

  const extractSummaryAnchors = (summaryText, previewText, options = {}) => {
    try {
      return extractSummaryAnchorsUnsafe(summaryText, previewText, options);
    } catch (_error) {
      return { anchors: [] };
    }
  };

  const buildLogErgonomicsModelUnsafe = (detail, previewResult, options = {}) => {
    const text = previewTextFor(detail, previewResult);
    const lines = renderLineNumberedTextUnsafe(text);
    const eligible = isLongTextPreviewUnsafe(detail, previewResult, options);
    const teaser = eligible
      ? sliceCollapsedTeaserUnsafe(lines, options)
      : {
          collapsed: false,
          hiddenCharacterCount: 0,
          hiddenLineCount: 0,
          visibleLines: lines,
        };
    const expansion = createExpansionStateUnsafe({ canExpand: teaser.collapsed });
    const summary = normalizeDetail(detail).summary;
    const navigation = eligible ? extractSummaryAnchorsUnsafe(summary, text, options) : { anchors: [] };

    return {
      eligible,
      expansion,
      lines,
      navigation,
      teaser,
      text,
    };
  };

  const buildLogErgonomicsModel = (detail, previewResult, options = {}) => {
    try {
      return buildLogErgonomicsModelUnsafe(detail, previewResult, options);
    } catch (_error) {
      return {
        eligible: false,
        expansion: {
          canExpand: false,
          expanded: true,
        },
        lines: [],
        navigation: { anchors: [] },
        teaser: {
          collapsed: false,
          hiddenCharacterCount: 0,
          hiddenLineCount: 0,
          visibleLines: [],
        },
        text: "",
      };
    }
  };

  const api = {
    buildLogErgonomicsModel,
    createExpansionState,
    defaultTeaserLineCount: DEFAULT_TEASER_LINE_COUNT,
    expandAllState,
    extractSummaryAnchors,
    isLongTextPreview,
    minimumLongTextLines: DEFAULT_MIN_LONG_TEXT_LINES,
    renderLineNumberedText,
    sliceCollapsedTeaser,
  };

  globalScope.ForemanArtifactLogErgonomics = api;

  if (typeof module !== "undefined" && module.exports) {
    module.exports = api;
  }
})(typeof window !== "undefined" ? window : globalThis);
