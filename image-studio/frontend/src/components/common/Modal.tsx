import { ReactNode, useEffect } from "react";

// Lightweight modal: click backdrop or press Esc to close. Children render
// inside a centred card. The modal is intentionally uncontrolled — callers
// hold the `open` state.
export function Modal({
  open, onClose, title, children, width = 480,
}: {
  open: boolean;
  onClose: () => void;
  title?: string;
  children: ReactNode;
  width?: number;
}) {
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;
  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div
        className="modal-card"
        style={{ width }}
        onClick={(e) => e.stopPropagation()}
      >
        {title && (
          <div className="modal-header">
            <h3 style={{ margin: 0, fontSize: 14, color: "var(--text)" }}>{title}</h3>
            <button className="modal-close" onClick={onClose} title="关闭 (Esc)">×</button>
          </div>
        )}
        <div className="modal-body">{children}</div>
      </div>
    </div>
  );
}
