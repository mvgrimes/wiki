const editorApp = document.getElementById("editor-app");

function escapeHTML(input) {
  return input
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

export function renderMarkdown(raw) {
  const escaped = escapeHTML(raw);
  const lines = escaped.split("\n");
  const out = [];
  let inList = false;

  for (const line of lines) {
    if (line.startsWith("- ")) {
      if (!inList) {
        out.push("<ul>");
        inList = true;
      }
      out.push(`<li>${inlineMarkdown(line.slice(2))}</li>`);
      continue;
    }

    if (inList) {
      out.push("</ul>");
      inList = false;
    }

    if (line.startsWith("### ")) {
      out.push(`<h3>${inlineMarkdown(line.slice(4))}</h3>`);
    } else if (line.startsWith("## ")) {
      out.push(`<h2>${inlineMarkdown(line.slice(3))}</h2>`);
    } else if (line.startsWith("# ")) {
      out.push(`<h1>${inlineMarkdown(line.slice(2))}</h1>`);
    } else if (line.trim() === "") {
      out.push("<br />");
    } else {
      out.push(`<p>${inlineMarkdown(line)}</p>`);
    }
  }

  if (inList) {
    out.push("</ul>");
  }

  return out.join("\n");
}

function inlineMarkdown(text) {
  return text
    .replace(/!\[([^\]]*)\]\(([^)]+)\)/g, '<img alt="$1" src="$2" />')
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>')
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/\*([^*]+)\*/g, "<em>$1</em>");
}

export function buildAttachmentMarkdown(payload) {
  const name = payload.file_name || "upload";
  const target = payload.storage_key || "";
  if (payload.content_type && payload.content_type.startsWith("image/")) {
    return `![${name}](${target})`;
  }
  return `[${name}](${target})`;
}

export async function uploadFile(file, pageSlug, onProgress) {
  const body = new FormData();
  body.append("page_slug", pageSlug);
  body.append("file", file);

  if (onProgress) {
    onProgress(25);
  }

  const response = await fetch("/api/uploads", {
    method: "POST",
    body,
  });

  if (onProgress) {
    onProgress(85);
  }

  if (!response.ok) {
    let message = `upload failed (${response.status})`;
    try {
      const payload = await response.json();
      if (payload && payload.error) {
        message = payload.error;
      }
    } catch {
      // Keep default status message.
    }
    throw new Error(message);
  }

  if (onProgress) {
    onProgress(100);
  }

  return response.json();
}

function insertAtCursor(textarea, snippet) {
  const start = textarea.selectionStart;
  const end = textarea.selectionEnd;
  const before = textarea.value.slice(0, start);
  const after = textarea.value.slice(end);
  const separator = before.endsWith("\n") || before.length === 0 ? "" : "\n";
  textarea.value = `${before}${separator}${snippet}\n${after}`;
  const next = before.length + separator.length + snippet.length + 1;
  textarea.selectionStart = next;
  textarea.selectionEnd = next;
}

function initWikiEditor() {
  if (!editorApp) {
    return;
  }

  const textarea = document.getElementById("wiki-body");
  const preview = document.getElementById("markdown-preview");
  const uploadInput = document.getElementById("upload-input");
  const dropzone = document.getElementById("upload-dropzone");
  const progress = document.getElementById("upload-progress");
  const status = document.getElementById("upload-status");
  const pageSlug = editorApp.dataset.pageSlug || "";

  if (!textarea || !preview || !uploadInput || !dropzone || !progress || !status) {
    return;
  }

  const updatePreview = () => {
    preview.innerHTML = renderMarkdown(textarea.value);
  };

  const runUpload = async (files) => {
    if (!files || files.length === 0) {
      return;
    }

    progress.hidden = false;
    for (const file of files) {
      status.textContent = `Uploading ${file.name}...`;
      progress.value = 0;
      try {
        const payload = await uploadFile(file, pageSlug, (value) => {
          progress.value = value;
        });
        const snippet = buildAttachmentMarkdown(payload);
        insertAtCursor(textarea, snippet);
        updatePreview();
        status.textContent = `Uploaded ${file.name}`;
      } catch (err) {
        status.textContent = `Error uploading ${file.name}: ${err.message}`;
      }
    }
    setTimeout(() => {
      progress.hidden = true;
      progress.value = 0;
    }, 300);
  };

  textarea.addEventListener("input", updatePreview);
  uploadInput.addEventListener("change", () => runUpload(uploadInput.files));

  const preventDefaults = (event) => {
    event.preventDefault();
    event.stopPropagation();
  };

  ["dragenter", "dragover", "dragleave", "drop"].forEach((eventName) => {
    dropzone.addEventListener(eventName, preventDefaults);
  });

  ["dragenter", "dragover"].forEach((eventName) => {
    dropzone.addEventListener(eventName, () => {
      dropzone.dataset.active = "true";
    });
  });

  ["dragleave", "drop"].forEach((eventName) => {
    dropzone.addEventListener(eventName, () => {
      delete dropzone.dataset.active;
    });
  });

  dropzone.addEventListener("drop", (event) => {
    runUpload(event.dataTransfer.files);
  });

  updatePreview();
}

initWikiEditor();
