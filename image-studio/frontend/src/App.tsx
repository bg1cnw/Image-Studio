import { lazy, Suspense, useEffect, useState } from "react";
import { AppHeader } from "./components/layout/AppHeader";
import { WorkspaceBar } from "./components/layout/WorkspaceBar";
import { FooterBar } from "./components/layout/FooterBar";
import { ToastContainer } from "./components/common/ToastContainer";
import { useStudioStore } from "./state/studioStore";
import { usePlatform } from "./lib/platformContext";
import { AndroidShell } from "./platform-shells/android/AndroidShell";
import { DesktopShell } from "./platform-shells/desktop/DesktopShell";

const UpstreamConfigModal = lazy(() => import("./components/panel/UpstreamConfigModal").then((m) => ({ default: m.UpstreamConfigModal })));
const ResultDetailDrawer = lazy(() => import("./components/panel/ResultDetailDrawer").then((m) => ({ default: m.ResultDetailDrawer })));
const SettingsPanel = lazy(() => import("./components/panel/SettingsPanel").then((m) => ({ default: m.SettingsPanel })));
// StarPromptModal 只在「第一次成功生图」后挂载一次,长尾用户根本不会触发 ——
// lazy 进一步避免它进入 critical bundle。
const StarPromptModal = lazy(() => import("./components/common/StarPromptModal").then((m) => ({ default: m.StarPromptModal })));

function App() {
  const bootstrap = useStudioStore((s) => s.bootstrap);
  const importImageFile = useStudioStore((s) => s.importImageFile);
  const fullscreen = useStudioStore((s) => s.fullscreen);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const { isAndroid, isAndroidPhone, isAndroidPad, isMac } = usePlatform();
  const [androidView, setAndroidView] = useState<"compose" | "canvas" | "history">(isAndroidPhone ? "compose" : "canvas");
  useEffect(() => { bootstrap(); }, [bootstrap]);

  useEffect(() => {
    if (!isAndroid) return;
    setAndroidView((cur) => {
      if (isAndroidPad) return cur === "compose" ? "compose" : "canvas";
      return cur === "history" ? "history" : "compose";
    });
  }, [isAndroid, isAndroidPad]);

  // Global app-level shortcuts. Canvas-scoped shortcuts (undo/redo, tool
  // switching, Esc) stay in CanvasStage so they don't fire when no image is up.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const meta = e.ctrlKey || e.metaKey;
      if (!meta) return;
      const target = e.target as HTMLElement | null;
      const inField = !!target && (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable);
      const k = e.key.toLowerCase();
      const st = useStudioStore.getState();

      // Primary-modifier + Enter: submit. Works inside the prompt textarea too.
      if (k === "enter") {
        e.preventDefault();
        st.submit();
        return;
      }
      // The rest only fire when NOT typing in a field.
      if (inField) return;
      if (k === "n") {
        e.preventDefault();
        st.newWorkspace();
      } else if (k === "w") {
        e.preventDefault();
        if (st.workspaces.length > 1) st.closeWorkspace(st.activeWorkspaceId);
      } else if (isMac && e.ctrlKey && e.metaKey && k === "f") {
        e.preventDefault();
        st.setField("fullscreen", !st.fullscreen);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  // Drop / paste → import to canvas.
  const [dragHover, setDragHover] = useState(false);
  useEffect(() => {
    let depth = 0;
    const onDragEnter = (e: DragEvent) => {
      if (!e.dataTransfer?.types.includes("Files")) return;
      e.preventDefault();
      depth++;
      setDragHover(true);
    };
    const onDragOver = (e: DragEvent) => {
      if (!e.dataTransfer?.types.includes("Files")) return;
      e.preventDefault();
    };
    const onDragLeave = (e: DragEvent) => {
      e.preventDefault();
      depth = Math.max(0, depth - 1);
      if (depth === 0) setDragHover(false);
    };
    const onDrop = (e: DragEvent) => {
      e.preventDefault();
      depth = 0;
      setDragHover(false);
      const files = e.dataTransfer?.files;
      if (!files || files.length === 0) return;
      void (async () => {
        for (const file of Array.from(files)) {
          await importImageFile(file);
        }
      })();
    };
    const onPaste = (e: ClipboardEvent) => {
      const target = e.target as HTMLElement | null;
      if (target && (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable)) {
        return; // let the user paste text into form fields normally
      }
      const items = e.clipboardData?.items;
      if (!items) return;
      for (const it of items) {
        if (it.kind === "file" && it.type.startsWith("image/")) {
          const file = it.getAsFile();
          if (file) {
            e.preventDefault();
            importImageFile(file);
            return;
          }
        }
      }
    };

    window.addEventListener("dragenter", onDragEnter);
    window.addEventListener("dragover", onDragOver);
    window.addEventListener("dragleave", onDragLeave);
    window.addEventListener("drop", onDrop);
    document.addEventListener("paste", onPaste);
    return () => {
      window.removeEventListener("dragenter", onDragEnter);
      window.removeEventListener("dragover", onDragOver);
      window.removeEventListener("dragleave", onDragLeave);
      window.removeEventListener("drop", onDrop);
      document.removeEventListener("paste", onPaste);
    };
  }, [importImageFile]);

  return (
    <div className="app-root relative">
      <div className="liquid-ambient" aria-hidden="true" />

      <AppHeader onOpenSettings={() => setSettingsOpen(true)} />
      <WorkspaceBar />
      {isAndroid ? (
        <AndroidShell
          fullscreen={fullscreen}
          isPad={isAndroidPad}
          androidView={androidView}
          onChangeView={setAndroidView}
        />
      ) : (
        <DesktopShell fullscreen={fullscreen} />
      )}
      <ToastContainer />
      {dragHover && (
        <div className="drop-overlay">
          <div className="drop-message">
            <div style={{ fontSize: 48, marginBottom: 12 }}>📥</div>
            松开鼠标导入图片到画板
            <div style={{ fontSize: 12, opacity: 0.6, marginTop: 8 }}>支持 PNG / JPG / WebP,最大 50MB</div>
          </div>
        </div>
      )}
      <FooterBar />
      <UpstreamConfigGate />
      <Suspense fallback={null}>
        <SettingsPanel open={settingsOpen} onClose={() => setSettingsOpen(false)} />
      </Suspense>
      <ResultDetailGate />
      <StarPromptGate />
    </div>
  );
}

// Star prompt 弹窗只在 store.starPromptOpen=true 时才挂载 Suspense + lazy 模块。
// 拆出来跟 UpstreamConfigGate 一样,避免顶层 App 因为一个布尔 state 重渲整棵树。
function StarPromptGate() {
  const { isMac } = usePlatform();
  if (isMac) return null;
  const open = useStudioStore((s) => s.starPromptOpen);
  if (!open) return null;
  return (
    <Suspense fallback={null}>
      <StarPromptModal open={open} />
    </Suspense>
  );
}

function ResultDetailGate() {
  const item = useStudioStore((s) => s.resultDetail);
  if (!item) return null;
  return (
    <Suspense fallback={null}>
      <ResultDetailDrawer />
    </Suspense>
  );
}

// Render the upstream-config modal driven by store state.
// Split out so the read of `upstreamModalOpen` only re-renders this subtree,
// not the whole App.
function UpstreamConfigGate() {
  const open = useStudioStore((s) => s.upstreamModalOpen);
  const close = useStudioStore((s) => s.closeUpstreamConfig);
  if (!open) return null;
  return (
    <Suspense fallback={null}>
      <UpstreamConfigModal open={open} onClose={close} />
    </Suspense>
  );
}

export default App;
