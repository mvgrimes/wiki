import test from "node:test";
import assert from "node:assert/strict";

global.document = {
  getElementById() {
    return null;
  },
};

const mod = await import("./wiki_edit.js");

test("renderMarkdown renders headings and list", () => {
  const html = mod.renderMarkdown("# Title\n\n- one\n- two");
  assert.match(html, /<h1>Title<\/h1>/);
  assert.match(html, /<ul>/);
  assert.match(html, /<li>one<\/li>/);
});

test("renderMarkdown escapes scripts", () => {
  const html = mod.renderMarkdown("<script>alert(1)</script>");
  assert.doesNotMatch(html, /<script>/);
  assert.match(html, /&lt;script&gt;alert\(1\)&lt;\/script&gt;/);
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
