import { describe, expect, it } from "vitest";
import {
  MAX_UPLOAD_BYTES,
  uploadErrorMessage,
  validateImageFile,
} from "../upload-errors";

describe("validateImageFile", () => {
  it("passes a normal phone photo", () => {
    expect(validateImageFile({ size: 4 * 1024 * 1024 })).toBeNull();
  });

  it("rejects a file over the cap and names both sizes", () => {
    const msg = validateImageFile({ size: MAX_UPLOAD_BYTES + 1 });
    expect(msg).toContain("слишком большой");
    expect(msg).toContain("10,0 МБ");
  });

  it("rejects an empty file", () => {
    expect(validateImageFile({ size: 0 })).toContain("пустой");
  });

  it("does not second-guess the format", () => {
    // iPhone HEIC often arrives with an empty type; the server decides.
    expect(validateImageFile({ size: 100 })).toBeNull();
  });
});

describe("uploadErrorMessage", () => {
  it("explains the CORS-masked proxy rejection instead of «Failed to fetch»", () => {
    const msg = uploadErrorMessage(new TypeError("Failed to fetch"));
    expect(msg).not.toContain("Failed to fetch");
    expect(msg).toContain("размер файла");
  });

  it("maps 413 to the size limit", () => {
    expect(uploadErrorMessage(new Error("upload failed: 413 "))).toContain("слишком большой");
  });

  it("maps 415 to the format list", () => {
    expect(uploadErrorMessage(new Error("upload failed: 415 unsupported"))).toContain("HEIC");
  });

  it("maps an expired session", () => {
    expect(uploadErrorMessage(new Error("upload failed: 401 unauthorized"))).toContain("Войдите");
  });

  it("falls back to a generic message", () => {
    expect(uploadErrorMessage(new Error("upload failed: 500"))).toContain("Попробуйте ещё раз");
  });
});
