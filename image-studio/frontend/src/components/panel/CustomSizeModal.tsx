import { useEffect, useMemo, useState, type FormEvent } from "react";
import { Ruler } from "lucide-react";
import { useStudioStore } from "../../state/studioStore";
import {
  formatSizeValue,
  MAX_EXACT_SIZE,
  MIN_EXACT_SIZE,
  parseSizeValue,
  reduceAspectRatioLabel,
} from "./sizeCapabilities";
import { Modal } from "../common/Modal";

const DEFAULT_EXACT_SIZE = { width: 1024, height: 1024 };

export function CustomSizeModal() {
  const open = useStudioStore((state) => state.customSizeModalOpen);
  const size = useStudioStore((state) => state.size);
  const applyCustomSize = useStudioStore((state) => state.applyCustomSize);
  const close = useStudioStore((state) => state.closeCustomSizeModal);
  const [widthInput, setWidthInput] = useState("");
  const [heightInput, setHeightInput] = useState("");

  useEffect(() => {
    if (!open) return;
    const parsed = size === "auto" ? DEFAULT_EXACT_SIZE : (parseSizeValue(size) ?? DEFAULT_EXACT_SIZE);
    setWidthInput(String(parsed.width));
    setHeightInput(String(parsed.height));
  }, [open, size]);

  const ratioHint = useMemo(() => {
    const width = Number(widthInput);
    const height = Number(heightInput);
    return reduceAspectRatioLabel(width, height);
  }, [heightInput, widthInput]);

  const multipleHint = useMemo(() => {
    const width = Number(widthInput);
    const height = Number(heightInput);
    if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) return "";
    return width % 8 === 0 && height % 8 === 0
      ? "当前尺寸已对齐到 8 的倍数。"
      : "当前尺寸不是 8 的倍数，部分上游可能会拒绝。";
  }, [heightInput, widthInput]);

  const handleApply = (event?: FormEvent<HTMLFormElement>) => {
    event?.preventDefault();
    applyCustomSize(Number(widthInput), Number(heightInput));
  };

  return (
    <Modal open={open} onClose={close} title="精确尺寸" width={680}>
      <div className="flex flex-col gap-4">
        <div className="rounded-[18px] border border-black/[0.06] bg-[var(--surface)] px-4 py-3 text-[12px] leading-6 text-zinc-600 dark:border-white/[0.08] dark:text-zinc-300">
          直接指定要传给上游的 <span className="font-semibold text-zinc-900 dark:text-zinc-100">size</span>。适合比例按钮不够精确的场景。
          点击比例或分辨率预设后，会自动退出精确尺寸模式。
        </div>

        <div className="rounded-[20px] border border-black/[0.06] bg-[var(--surface)] px-4 py-3 dark:border-white/[0.08]">
          <div className="flex items-center gap-2 text-[13px] font-semibold text-zinc-900 dark:text-zinc-100">
            <Ruler className="h-4 w-4 text-[var(--accent)]" />
            当前工作区尺寸
          </div>
          <div className="mt-2 text-[12px] text-zinc-500 dark:text-zinc-400">
            {size === "auto" ? "Auto（由上游决定）" : formatSizeValue(size)}
          </div>
        </div>

        <form onSubmit={handleApply} className="rounded-[20px] border border-black/[0.06] bg-[var(--surface)] p-4 dark:border-white/[0.08]">
          <div className="flex items-center justify-between gap-3">
            <div>
              <div className="text-[13px] font-semibold text-zinc-900 dark:text-zinc-100">设置精确尺寸</div>
              <div className="mt-1 text-[11px] text-zinc-500 dark:text-zinc-400">
                请输入 {MIN_EXACT_SIZE} 到 {MAX_EXACT_SIZE} 之间的整数像素值。
              </div>
            </div>
            {ratioHint ? (
              <div className="rounded-full bg-black/[0.04] px-2.5 py-1 text-[11px] text-zinc-500 dark:bg-white/[0.06] dark:text-zinc-300">
                比例 {ratioHint}
              </div>
            ) : null}
          </div>

          <div className="mt-4 grid gap-3 md:grid-cols-[1fr_auto_1fr_auto]">
            <label className="flex flex-col gap-1.5 text-[11px] text-zinc-500 dark:text-zinc-400">
              宽度(px)
              <input
                type="number"
                min={MIN_EXACT_SIZE}
                max={MAX_EXACT_SIZE}
                step={1}
                inputMode="numeric"
                value={widthInput}
                onChange={(event) => setWidthInput(event.currentTarget.value)}
                placeholder="例如 1536"
                className="focus-ring rounded-[14px] border border-black/[0.08] bg-white/80 px-3 py-2 text-[14px] text-zinc-900 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-zinc-100"
              />
            </label>
            <div className="flex items-end justify-center pb-2 text-[18px] font-semibold text-zinc-400">×</div>
            <label className="flex flex-col gap-1.5 text-[11px] text-zinc-500 dark:text-zinc-400">
              高度(px)
              <input
                type="number"
                min={MIN_EXACT_SIZE}
                max={MAX_EXACT_SIZE}
                step={1}
                inputMode="numeric"
                value={heightInput}
                onChange={(event) => setHeightInput(event.currentTarget.value)}
                placeholder="例如 1024"
                className="focus-ring rounded-[14px] border border-black/[0.08] bg-white/80 px-3 py-2 text-[14px] text-zinc-900 dark:border-white/[0.08] dark:bg-white/[0.04] dark:text-zinc-100"
              />
            </label>
            <button
              type="submit"
              className="inline-flex min-h-[44px] items-center justify-center rounded-[14px] bg-[var(--accent)] px-4 py-2 text-[13px] font-semibold text-white transition-opacity hover:opacity-90"
            >
              应用尺寸
            </button>
          </div>

          {multipleHint ? (
            <div className="mt-3 text-[11px] text-zinc-500 dark:text-zinc-400">
              {multipleHint}
            </div>
          ) : null}
        </form>
      </div>
    </Modal>
  );
}
