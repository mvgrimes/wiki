import test from "node:test";
import assert from "node:assert/strict";

global.document = {
  getElementById() {
    return null;
  },
};

const mod = await import("./wiki_edit.js");

test("previewMarkdown returns rendered HTML", async () => {
  global.fetch = async () => ({
    ok: true,
    async json() {
      return { html: "<h1>Title</h1>" };
    },
  });

  const html = await mod.previewMarkdown("# Title");
  assert.equal(html, "<h1>Title</h1>");
});

test("buildAttachmentMarkdown handles image payload", () => {
  const snippet = mod.buildAttachmentMarkdown({
    file_name: "photo.png",
    storage_key: "attachments/page/file.png",
    content_type: "image/png",
  });
  assert.equal(snippet, "![photo.png](attachments/page/file.png)");
});

test("buildAttachmentMarkdown handles non-image payload", () => {
  const snippet = mod.buildAttachmentMarkdown({
    file_name: "guide.pdf",
    storage_key: "attachments/page/guide.pdf",
    content_type: "application/pdf",
  });
  assert.equal(snippet, "[guide.pdf](attachments/page/guide.pdf)");
});
