import {
  OpenImageDialog,
  ImportImageFromB64,
  SaveImageAs,
  SaveImagePathAs,
} from "../platform/runtime/host";
import { saveImageForPlatform } from "../platform/android/bridge";
import { base64ToBlob } from "../lib/images";
import { removeHistoryItem } from "../lib/storage";
import type { HistoryItem } from "../types/domain";
import type { StudioState } from "./studioStore.types";
import {
  createPreviewB64,
  ensureFullHistoryItem,
  fileToBase64,
  materializeHistoryItem,
} from "./studioStore.runtime";
import { patchWorkspaceRuntime } from "./workspaceRuntime";
import { genId } from "./studioStore.shared";

type StateAdapter = {
  getState: () => StudioState;
  setState: (patch: Partial<StudioState> | ((state: StudioState) => Partial<StudioState>)) => void;
};

export function createImageActions(store: StateAdapter) {
  return {
    async selectSourceImage() {
      try {
        const res = await OpenImageDialog();
        if (!res || !res.path) return;
        const baseName = res.path.split(/[\\/]/).pop() ?? res.path;
        const existing = store.getState().sources;
        if (existing.some((source) => source.path === res.path)) {
          store.setState({ mode: "edit", errorMessage: null, errorRawPath: null });
          return;
        }
        const imageB64 = res.imageB64 ?? "";
        const previewB64 = res.previewB64 || imageB64;
        const imageBlob = previewB64 ? base64ToBlob(previewB64) : null;
        store.setState({
          sources: [...existing, { path: res.path, name: baseName, size: res.size, imageB64: previewB64, imageBlob }],
          mode: "edit",
          errorMessage: null,
          errorRawPath: null,
        });
      } catch (error: any) {
        store.setState({ errorMessage: `选择图片失败:${error?.message ?? error}`, errorRawPath: null });
      }
    },

    removeSource(index: number) {
      const next = store.getState().sources.filter((_, i) => i !== index);
      store.setState({ sources: next, mode: next.length > 0 ? "edit" : "generate" });
    },

    clearSources() {
      store.setState({ sources: [], mode: "generate" });
    },

    reorderSources(from: number, to: number) {
      const list = [...store.getState().sources];
      if (from < 0 || from >= list.length || to < 0 || to >= list.length) return;
      const [moved] = list.splice(from, 1);
      list.splice(to, 0, moved);
      store.setState({ sources: list });
    },

    async reuseAsSource(item: HistoryItem) {
      const localItem = await materializeHistoryItem(item, {
        setState: (fn) => store.setState((state) => fn(state)),
      }).catch((e: any) => {
        store.setState({ errorMessage: `源图准备失败:${e?.message ?? e}`, errorRawPath: null });
        return null;
      });
      if (!localItem?.savedPath) return;
      const baseName = localItem.savedPath.split(/[\\/]/).pop() ?? "source.png";
      const existing = store.getState().sources;
      const alreadyIn = existing.some((source) => source.path === localItem.savedPath);
      store.setState({
        mode: "edit",
        currentImage: localItem,
        resultGridOpen: false,
        sources: alreadyIn
          ? existing
          : [...existing, {
              path: localItem.savedPath,
              name: baseName,
              size: 0,
              imageBlob: localItem.previewBlob ?? localItem.imageBlob ?? null,
              imageB64: localItem.imageB64,
            }],
      });
    },

    applyHistoryParams(item: HistoryItem) {
      const patch: Partial<StudioState> = {
        prompt: item.prompt ?? "",
        mode: item.mode,
        size: item.size,
        quality: item.quality,
      };
      if (item.seed !== undefined) patch.seed = item.seed;
      if (item.negativePrompt !== undefined) patch.negativePrompt = item.negativePrompt;
      if (item.styleTag !== undefined) patch.styleTag = item.styleTag;
      if (item.outputFormat) patch.outputFormat = item.outputFormat;
      store.setState(patch);
      store.getState().pushToast("已应用此图的参数到控制台", "success");
    },

    async regenerateFromHistory(item: HistoryItem) {
      this.applyHistoryParams(item);
      await Promise.resolve();
      await store.getState().submit();
    },

    async deleteHistoryItem(id: string) {
      await removeHistoryItem(id);
      const currentBefore = store.getState().currentImage;
      const wasCurrent = currentBefore?.id === id;
      const nextBatch = store.getState().batchResults.filter((entry) => entry.id !== id);
      const patch: Partial<StudioState> = { batchResults: nextBatch };
      if (wasCurrent) patch.currentImage = null;
      if (nextBatch.length <= 1) patch.resultGridOpen = false;
      store.setState((state) => ({
        history: state.history.filter((entry) => entry.id !== id),
        ...(patch as any),
        workspaces: patchWorkspaceRuntime(state.workspaces, state.activeWorkspaceId, {
          currentImageId: wasCurrent ? null : currentBefore?.id ?? null,
          batchResultIds: nextBatch.map((entry) => entry.id),
          resultGridOpen: nextBatch.length > 1 && (patch.resultGridOpen ?? state.resultGridOpen),
        }),
      }));
    },

    async saveCurrentImageAs() {
      const current = store.getState().currentImage;
      if (!current) return;
      const suggested = `image-${current.mode}-${current.id.slice(0, 8)}.png`;
      try {
        const saved = current.savedPath
          ? await SaveImagePathAs(current.savedPath, suggested)
          : await saveImageForPlatform((await ensureFullHistoryItem(current, {
              setState: (fn) => store.setState((state) => fn(state)),
            }))?.imageB64 ?? "", suggested, SaveImageAs);
        if (saved) store.getState().pushToast(`已保存:${saved.split(/[\\/]/).pop()}`, "success");
      } catch (e: any) {
        const msg = `保存失败:${e?.message ?? e}`;
        store.setState({ errorMessage: msg, errorRawPath: null });
        store.getState().pushToast(msg, "error");
      }
    },

    async importImageFile(file: File) {
      try {
        if (!/^image\/(png|jpe?g|webp)$/.test(file.type)) {
          store.setState({ errorMessage: `不支持的图片类型:${file.type || "(未知)"},请用 PNG/JPG/WebP`, errorRawPath: null });
          return;
        }
        const b64 = await fileToBase64(file);
        const result = await ImportImageFromB64(b64, file.name);
        const previewB64 = await createPreviewB64(b64);
        const previewBlob = base64ToBlob(previewB64);
        const fullBlob = base64ToBlob(b64);
        const transientItem: HistoryItem = {
          id: genId(),
          imageB64: b64,
          imageBlob: fullBlob,
          previewBlob,
          prompt: `(导入)${file.name}`,
          mode: "edit",
          size: "1024x1024",
          quality: "medium",
          createdAt: Date.now(),
          savedPath: result.path,
        };
        const existingSources = store.getState().sources;
        const alreadyIn = existingSources.some((source) => source.path === result.path);
        store.setState({
          currentImage: transientItem,
          batchResults: [],
          resultGridOpen: false,
          mode: "edit",
          sources: alreadyIn
            ? existingSources
            : [...existingSources, {
                path: result.path,
                name: file.name,
                size: file.size,
                imageBlob: previewBlob,
                imageB64: previewB64,
              }],
          errorMessage: null,
          errorRawPath: null,
        });
      } catch (e: any) {
        store.setState({ errorMessage: `导入失败:${e?.message ?? e}`, errorRawPath: null });
      }
    },
  };
}
