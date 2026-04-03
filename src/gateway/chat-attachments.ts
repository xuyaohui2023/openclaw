import { estimateBase64DecodedBytes } from "../media/base64.js";
import {
  extractFileContentFromSource,
  resolveInputFileLimits,
  type InputFileSource,
} from "../media/input-files.js";
import { sniffMimeFromBase64 } from "../media/sniff-mime-from-base64.js";

export type ChatAttachment = {
  type?: string;
  mimeType?: string;
  fileName?: string;
  content?: unknown;
};

export type ChatImageContent = {
  type: "image";
  data: string;
  mimeType: string;
};

export type ParsedMessageWithImages = {
  message: string;
  images: ChatImageContent[];
};

type AttachmentLog = {
  warn: (message: string) => void;
};

type NormalizedAttachment = {
  label: string;
  mime: string;
  base64: string;
};

function normalizeMime(mime?: string): string | undefined {
  if (!mime) {
    return undefined;
  }
  const cleaned = mime.split(";")[0]?.trim().toLowerCase();
  return cleaned || undefined;
}

const DOCUMENT_MIMES = new Set([
  "application/pdf",
  "text/plain",
  "text/markdown",
  "text/html",
  "text/csv",
  "application/json",
]);

function isImageMime(mime?: string): boolean {
  return typeof mime === "string" && mime.startsWith("image/");
}

function isDocumentMime(mime?: string): boolean {
  return typeof mime === "string" && DOCUMENT_MIMES.has(mime);
}

function isValidBase64(value: string): boolean {
  if (value.length === 0 || value.length % 4 !== 0) {
    return false;
  }
  // Skip full regex scan for large payloads to avoid O(n) cost at 100MB scale.
  if (value.length > 4096) {
    return true;
  }
  return /^[A-Za-z0-9+/]+={0,2}$/.test(value);
}

function normalizeAttachment(
  att: ChatAttachment,
  idx: number,
  opts: { stripDataUrlPrefix: boolean; requireImageMime: boolean },
): NormalizedAttachment {
  const mime = att.mimeType ?? "";
  const content = att.content;
  const label = att.fileName || att.type || `attachment-${idx + 1}`;

  if (typeof content !== "string") {
    throw new Error(`attachment ${label}: content must be base64 string`);
  }
  if (opts.requireImageMime && !mime.startsWith("image/")) {
    throw new Error(`attachment ${label}: only image/* supported`);
  }

  let base64 = content.trim();
  if (opts.stripDataUrlPrefix) {
    // Strip data URL prefix if present (e.g., "data:image/jpeg;base64,...").
    const dataUrlMatch = /^data:[^;]+;base64,(.*)$/.exec(base64);
    if (dataUrlMatch) {
      base64 = dataUrlMatch[1];
    }
  }
  return { label, mime, base64 };
}

function validateAttachmentBase64OrThrow(
  normalized: NormalizedAttachment,
  opts: { maxBytes: number },
): number {
  if (!isValidBase64(normalized.base64)) {
    throw new Error(`attachment ${normalized.label}: invalid base64 content`);
  }
  const sizeBytes = estimateBase64DecodedBytes(normalized.base64);
  if (sizeBytes <= 0 || sizeBytes > opts.maxBytes) {
    throw new Error(
      `attachment ${normalized.label}: exceeds size limit (${sizeBytes} > ${opts.maxBytes} bytes)`,
    );
  }
  return sizeBytes;
}

/**
 * Parse attachments and extract images and documents as structured content.
 * Images are returned as structured content blocks compatible with Claude API.
 * Documents (PDF, TXT, CSV, JSON, Markdown) are extracted and appended to the message text.
 * Max file size: 2MB per attachment, 50MB total.
 */
export async function parseMessageWithAttachments(
  message: string,
  attachments: ChatAttachment[] | undefined,
  opts?: { maxBytes?: number; maxTotalBytes?: number; log?: AttachmentLog },
): Promise<ParsedMessageWithImages> {
  const maxBytes = opts?.maxBytes ?? 10 * 1024 * 1024; // 10MB per file
  const maxTotalBytes = opts?.maxTotalBytes ?? 50 * 1024 * 1024; // 50MB total
  const log = opts?.log;
  if (!attachments || attachments.length === 0) {
    return { message, images: [] };
  }

  const images: ChatImageContent[] = [];
  const fileBlocks: string[] = [];
  let totalBytes = 0;

  for (const [idx, att] of attachments.entries()) {
    if (!att) {
      continue;
    }
    const normalized = normalizeAttachment(att, idx, {
      stripDataUrlPrefix: true,
      requireImageMime: false,
    });
    const attachmentBytes = validateAttachmentBase64OrThrow(normalized, { maxBytes });
    totalBytes += attachmentBytes;
    if (totalBytes > maxTotalBytes) {
      throw new Error(
        `total attachment size exceeds limit (${totalBytes} > ${maxTotalBytes} bytes)`,
      );
    }
    const { base64: b64, label, mime } = normalized;

    const providedMime = normalizeMime(mime);
    const sniffedMime = normalizeMime(await sniffMimeFromBase64(b64));
    // Prefer sniffed MIME for binary formats; fall back to provided for text-family types.
    const effectiveMime = sniffedMime ?? providedMime ?? "";

    // --- Images ---
    if (isImageMime(effectiveMime) || (!sniffedMime && isImageMime(providedMime))) {
      if (sniffedMime && providedMime && sniffedMime !== providedMime) {
        log?.warn(
          `attachment ${label}: mime mismatch (${providedMime} -> ${sniffedMime}), using sniffed`,
        );
      }
      images.push({
        type: "image",
        data: b64,
        mimeType: sniffedMime ?? providedMime ?? mime,
      });
      continue;
    }

    // --- Documents (PDF, TXT, CSV, JSON, Markdown, HTML) ---
    // For text-family types, sniff often returns text/plain regardless of subtype,
    // so prefer provided MIME when it's a known document type.
    const docMime = isDocumentMime(providedMime ?? "")
      ? providedMime!
      : isDocumentMime(effectiveMime)
        ? effectiveMime
        : null;

    if (docMime) {
      try {
        const source: InputFileSource = {
          type: "base64",
          data: b64,
          mediaType: docMime,
          filename: label,
        };
        const limits = resolveInputFileLimits({
          allowedMimes: Array.from(DOCUMENT_MIMES),
          maxBytes,
          pdf: { maxPages: 20 },
        });
        const result = await extractFileContentFromSource({ source, limits });
        if (result.text) {
          fileBlocks.push(`[File: ${result.filename}]\n${result.text}`);
        }
        // PDF pages rendered as images (when text extraction yields too little content)
        if (result.images && result.images.length > 0) {
          for (const img of result.images) {
            images.push({ type: "image", data: img.data, mimeType: img.mimeType });
          }
        }
      } catch (err) {
        log?.warn(`attachment ${label}: failed to extract content (${String(err)}), dropping`);
      }
      continue;
    }

    log?.warn(
      `attachment ${label}: unsupported type (${effectiveMime || providedMime || "unknown"}), dropping`,
    );
  }

  // Append extracted file content to the message
  let finalMessage = message;
  if (fileBlocks.length > 0) {
    const separator = finalMessage.trim().length > 0 ? "\n\n" : "";
    finalMessage = `${finalMessage}${separator}${fileBlocks.join("\n\n")}`;
  }

  return { message: finalMessage, images };
}

/**
 * @deprecated Use parseMessageWithAttachments instead.
 * This function converts images to markdown data URLs which Claude API cannot process as images.
 */
export function buildMessageWithAttachments(
  message: string,
  attachments: ChatAttachment[] | undefined,
  opts?: { maxBytes?: number },
): string {
  const maxBytes = opts?.maxBytes ?? 2_000_000; // 2 MB
  if (!attachments || attachments.length === 0) {
    return message;
  }

  const blocks: string[] = [];

  for (const [idx, att] of attachments.entries()) {
    if (!att) {
      continue;
    }
    const normalized = normalizeAttachment(att, idx, {
      stripDataUrlPrefix: false,
      requireImageMime: true,
    });
    validateAttachmentBase64OrThrow(normalized, { maxBytes });
    const { base64, label, mime } = normalized;

    const safeLabel = label.replace(/\s+/g, "_");
    const dataUrl = `![${safeLabel}](data:${mime};base64,${base64})`;
    blocks.push(dataUrl);
  }

  if (blocks.length === 0) {
    return message;
  }
  const separator = message.trim().length > 0 ? "\n\n" : "";
  return `${message}${separator}${blocks.join("\n\n")}`;
}
