import { useState } from "react";
import { FolderOpen, Save } from "lucide-react";
import { useStudioStore } from "../../state/studioStore";
import { historyPreviewSrc, useBlobURL } from "../../lib/images";
import { saveHistoryItemAs } from "../../lib/saveResultImage";
import { androidSaveHint, androidTarget } from "../../platform/android/bridge";
import { usePlatform } from "../../platform/context";
import { Modal } from "./Modal";

export function SavePromptModal() {
  const item = useStudioStore((s) => s.savePromptItem);
  const closeSavePrompt = useStudioStore((s) => s.closeSavePrompt);
  const suppressed = useStudioStore((s) => s.savePromptSuppressed);
  const setSuppressed = useStudioStore((s) => s.setSavePromptSuppressed);
  const pushToast = useStudioStore((s) => s.pushToast);
  const { usesFluentUI, isAndroidPhone } = usePlatform();
  const [saving, setSaving] = useState(false);
  const previewURL = useBlobURL(item?.previewBlob ?? item?.imageBlob ?? null, item?.imageB64 ?? null);

  if (!item) return null;
  const imageSrc = historyPreviewSrc(item, previewURL);

  async function saveAs() {
    if (saving || !item) return;
    setSaving(true);
    try {
      const saved = await saveHistoryItemAs(item);
      if (saved) pushToast(`已保存:${saved.split(/[\\/]/).pop()}`, "success");
      closeSavePrompt();
    } catch (error: any) {
      pushToast(`保存失败:${error?.message ?? error}`, "error", 6000);
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal open onClose={closeSavePrompt} title="是否另存这张图片?" width={isAndroidPhone ? 420 : 520}>
      <div className="space-y-4">
        <div className="grid gap-3 sm:grid-cols-[132px_minmax(0,1fr)]">
          <div className={`grid min-h-[132px] place-items-center overflow-hidden border border-black/[0.08] bg-[var(--surface)] p-2 dark:border-white/[0.06] ${usesFluentUI ? "rounded-[10px]" : "rounded-[16px]"}`}>
            {imageSrc ? (
              <img
                src={imageSrc}
                alt="生成结果预览"
                decoding="async"
                className={`max-h-[120px] max-w-full object-contain ${usesFluentUI ? "rounded-[8px]" : "rounded-[12px]"}`}
              />
            ) : (
              <Save className="h-8 w-8 text-zinc-400" />
            )}
          </div>
          <div className="min-w-0 space-y-2">
            <p className="text-sm font-medium text-zinc-900 dark:text-zinc-100">
              图片已生成并保存在默认输出目录。
            </p>
            <p className="text-xs leading-relaxed text-zinc-600 dark:text-zinc-300">
              需要放到项目、相册或其他目录时，可以现在选择目标位置另存一份。
            </p>
            {item.savedPath ? (
              <p className={`font-mono-token break-all border border-black/[0.06] bg-[var(--surface)] px-2.5 py-2 text-[11px] text-zinc-500 dark:border-white/[0.04] ${usesFluentUI ? "rounded-[8px]" : "rounded-[12px]"}`}>
                {item.savedPath}
              </p>
            ) : null}
          </div>
        </div>

        {androidTarget.isAndroid ? (
          <p className="text-[11px] leading-relaxed text-zinc-500">{androidSaveHint()}</p>
        ) : null}

        <label className="flex cursor-pointer items-center gap-2 text-xs text-zinc-600 dark:text-zinc-300">
          <input
            type="checkbox"
            checked={suppressed}
            onChange={(event) => setSuppressed(event.currentTarget.checked)}
            className="h-4 w-4 accent-[var(--accent)]"
          />
          以后不再提示
        </label>

        <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
          <button
            type="button"
            onClick={closeSavePrompt}
            className={`platform-action-btn border border-black/[0.08] px-4 py-2 text-sm text-zinc-700 transition-colors hover:bg-black/[0.04] dark:border-white/[0.08] dark:text-zinc-300 dark:hover:bg-white/[0.06] ${usesFluentUI ? "rounded-[8px]" : "rounded-full"}`}
          >
            稍后
          </button>
          <button
            type="button"
            onClick={saveAs}
            disabled={saving}
            className={`liquid-primary-button inline-flex items-center justify-center gap-1.5 bg-[var(--accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--accent-2)] disabled:cursor-not-allowed disabled:opacity-60 ${usesFluentUI ? "rounded-[8px]" : "rounded-full"}`}
          >
            <FolderOpen className="h-4 w-4" />
            {saving ? "保存中..." : "保存到指定位置"}
          </button>
        </div>
      </div>
    </Modal>
  );
}
