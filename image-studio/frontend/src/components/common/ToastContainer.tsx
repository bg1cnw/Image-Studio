import { useStudioStore } from "../../state/studioStore";
import type { Toast } from "../../types/domain";

export function ToastContainer() {
  const toasts = useStudioStore((s) => s.toasts);
  const dismiss = useStudioStore((s) => s.dismissToast);

  if (toasts.length === 0) return null;

  return (
    <div className="toast-stack">
      {toasts.map((t) => (
        <ToastItem key={t.id} t={t} onClose={() => dismiss(t.id)} />
      ))}
    </div>
  );
}

function ToastItem({ t, onClose }: { t: Toast; onClose: () => void }) {
  const icon = t.kind === "success" ? "✓" : t.kind === "error" ? "✕" : t.kind === "warn" ? "!" : "i";
  return (
    <div className={`toast toast-${t.kind}`} onClick={onClose}>
      <span className="toast-icon">{icon}</span>
      <span className="toast-text">{t.text}</span>
    </div>
  );
}
