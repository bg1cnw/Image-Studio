import { useEffect, useState } from "react";
import { Modal } from "../common/Modal";
import { ReadTextFile } from "../../../wailsjs/go/backend/Service";
import { useStudioStore } from "../../state/studioStore";

const MAX_PREVIEW = 200_000; // chars

export function RawResponseModal({ path, onClose }: { path: string; onClose: () => void }) {
  const [text, setText] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const pushToast = useStudioStore((s) => s.pushToast);

  useEffect(() => {
    setLoading(true);
    ReadTextFile(path)
      .then((t) => {
        if (t.length > MAX_PREVIEW) setText(t.slice(0, MAX_PREVIEW) + `\n\n... [截断,完整 ${(t.length / 1024).toFixed(1)} KB 在文件里]`);
        else setText(t);
      })
      .catch((e: any) => setError(e?.message ?? String(e)))
      .finally(() => setLoading(false));
  }, [path]);

  async function copyAll() {
    try {
      await navigator.clipboard.writeText(text);
      pushToast("已复制到剪贴板", "success");
    } catch (e: any) {
      pushToast(`复制失败:${e?.message ?? e}`, "error");
    }
  }

  return (
    <Modal open onClose={onClose} title="原始 SSE 响应" width={760}>
      <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 8, fontSize: 11, color: "var(--text-muted)" }}>
        <code style={{ wordBreak: "break-all" }}>{path}</code>
        <div style={{ display: "flex", gap: 6 }}>
          <button className="tool-btn" onClick={copyAll}>复制全文</button>
        </div>
      </div>
      {loading && <div style={{ color: "var(--text-muted)", padding: 12 }}>读取中...</div>}
      {error && <div className="error-banner">{error}</div>}
      {!loading && !error && (
        <pre style={{
          background: "var(--bg)",
          color: "var(--text-muted)",
          padding: 12,
          borderRadius: 6,
          maxHeight: "55vh",
          overflow: "auto",
          fontSize: 11,
          lineHeight: 1.5,
          whiteSpace: "pre-wrap",
          wordBreak: "break-all",
          border: "1px solid var(--border)",
        }}>{text}</pre>
      )}
    </Modal>
  );
}
