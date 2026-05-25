import { ReactNode, useEffect } from "react";
import { X } from "lucide-react";
import { usePlatform } from "../../lib/platformContext";

// 居中 modal:点击背景 / Esc 关闭。
export function Modal({
  open, onClose, title, children, width = 480,
}: {
  open: boolean;
  onClose: () => void;
  title?: string;
  children: ReactNode;
  width?: number;
}) {
  const { isAndroidPhone, isWindows, usesAppleUI } = usePlatform();
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
    <div
      className={`fixed inset-0 z-[9100] flex bg-black/18 backdrop-blur-md ${isAndroidPhone ? "items-end justify-stretch p-0" : "items-center justify-center p-5"}`}
      onClick={onClose}
    >
      <div
        style={{ width }}
        onClick={(e) => e.stopPropagation()}
        className={`flex flex-col overflow-hidden border border-black/[0.08] bg-white/92 shadow-[0_30px_80px_rgb(15_23_42_/_0.18)] backdrop-blur-2xl dark:border-white/[0.08] dark:bg-zinc-900/92 ${usesAppleUI ? "liquid-glass-panel" : ""} ${isAndroidPhone ? "max-h-[92vh] min-h-[70vh] w-full rounded-t-[28px] border-b-0" : `max-h-[86vh] ${isWindows ? "rounded-[14px]" : "rounded-[22px]"}`}`}
      >
        {title && (
          <div className={`flex items-center justify-between border-b border-black/[0.06] dark:border-white/[0.04] ${isAndroidPhone ? "px-4 py-3" : "px-5 py-3.5"}`}>
            <h3 className="m-0 text-[15px] font-semibold tracking-[-0.01em] text-zinc-900 dark:text-zinc-100">{title}</h3>
            <button
              onClick={onClose}
              title="关闭 (Esc)"
              className={`-mr-1 p-1.5 text-zinc-500 hover:bg-black/[0.05] hover:text-zinc-900 dark:hover:bg-white/[0.06] dark:hover:text-zinc-100 ${isWindows ? "rounded-[8px]" : "rounded-full"}`}
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        )}
        <div className={`modal-scroll-body overflow-y-auto ${isAndroidPhone ? "p-4" : "p-5"}`}>{children}</div>
      </div>
    </div>
  );
}
