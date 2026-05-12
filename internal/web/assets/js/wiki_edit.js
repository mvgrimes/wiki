const editorApp = document.getElementById("editor-app");

export function buildAttachmentMarkdown(payload) {
  const name = payload.file_name || "upload";
  const target = payload.storage_key || "";
  if (payload.content_type && payload.content_type.startsWith("image/")) {
    return `![${name}](${target})`;
  }
  return `[${name}](${target})`;
}

export async function previewMarkdown(markdown) {
  const response = await fetch("/api/markdown/preview", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ markdown }),
  });

  if (!response.ok) {
    throw new Error(`preview failed (${response.status})`);
  }

  const payload = await response.json();
  return payload.html || "";
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
  const uploadInput = document.getElementById("upload-input");
  const dropzone = document.getElementById("upload-dropzone");
  const progress = document.getElementById("upload-progress");
  const status = document.getElementById("upload-status");
  const pageSlug = editorApp.dataset.pageSlug || "";

  if (!textarea || !uploadInput || !dropzone || !progress || !status) {
    return;
  }

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
        textarea.dispatchEvent(new Event("input", { bubbles: true }));
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
}

if (typeof window !== "undefined") {
  window.wikiEditor = function wikiEditor(el) {
    return {
      markdown: el?.dataset?.initialMarkdown || "",
      previewHTML: "",
      async init() {
        await this.refreshPreview();
      },
      async refreshPreview() {
        try {
          this.previewHTML = await previewMarkdown(this.markdown);
        } catch {
          this.previewHTML = "<p>Preview unavailable.</p>";
        }
      },
      pageSlug: el?.dataset?.pageSlug || "",
    };
  };
}

initWikiEditor();
