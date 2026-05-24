import { lazy, Suspense, useEffect, useState } from "react";
import type { ReactElement } from "react";
import { History, Image as ImageIcon, SlidersHorizontal } from "lucide-react";
import { AppHeader } from "./components/layout/AppHeader";
import { WorkspaceBar } from "./components/layout/WorkspaceBar";
import { FooterBar } from "./components/layout/FooterBar";
import { ControlPanel } from "./components/panel/ControlPanel";
import { CanvasStage } from "./components/canvas/CanvasStage";
import { Toolbar } from "./components/canvas/Toolbar";
import { SourceStrip } from "./components/canvas/SourceStrip";
import { StatusBar } from "./components/canvas/StatusBar";
import { HistoryRail } from "./components/history/HistoryRail";
import { ToastContainer } from "./components/common/ToastContainer";
import { useStudioStore } from "./state/studioStore";
import { isAndroidPhone, isAndroidPad, isMac } from "./lib/platform";

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
  const [androidView, setAndroidView] = useState<"compose" | "canvas" | "history">("canvas");
  useEffect(() => { bootstrap(); }, [bootstrap]);

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
      // ★ 之前只取 files[0],拖多张时只读第一张。改成遍历所有文件,
      //   importImageFile 内部会逐张去重 + 追加到 sources 末尾。
      //   串行 await 而不是 Promise.all 是为了避免后端文件名冲突,
      //   也保证 sources 顺序跟用户拖入顺序一致。
      const files = e.dataTransfer?.files;
      if (!files || files.length === 0) return;
      void (async () => {
        for (const f of Array.from(files)) {
          await importImageFile(f);
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
      <div
        className={`studio ${fullscreen ? "fullscreen" : ""} ${isAndroidPhone ? "android-phone" : ""} ${isAndroidPad ? "android-pad" : ""}`}
        data-android-view={androidView}
      >
        {isAndroidPad && !fullscreen && <AndroidRail active={androidView} onChange={setAndroidView} />}
        <ControlPanel />
        <div className="canvas-shell">
          <Toolbar />
          <SourceStrip />
          <CanvasStage />
          <StatusBar />
        </div>
        <HistoryRail />
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
      </div>
      <FooterBar />
      {isAndroidPhone && !fullscreen && <AndroidBottomNav active={androidView} onChange={setAndroidView} />}
      <UpstreamConfigGate />
      <Suspense fallback={null}>
        <SettingsPanel open={settingsOpen} onClose={() => setSettingsOpen(false)} />
      </Suspense>
      <Suspense fallback={null}>
        <ResultDetailDrawer />
      </Suspense>
      <StarPromptGate />
    </div>
  );
}

// Star prompt 弹窗只在 store.starPromptOpen=true 时才挂载 Suspense + lazy 模块。
// 拆出来跟 UpstreamConfigGate 一样,避免顶层 App 因为一个布尔 state 重渲整棵树。
function StarPromptGate() {
  const open = useStudioStore((s) => s.starPromptOpen);
  if (!open) return null;
  return (
    <Suspense fallback={null}>
      <StarPromptModal open={open} />
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

function AndroidRail({
  active,
  onChange,
}: {
  active: "compose" | "canvas" | "history";
  onChange: (value: "compose" | "canvas" | "history") => void;
}) {
  return (
    <nav className="android-rail" aria-label="Android Pad navigation">
      <AndroidNavButton icon={<SlidersHorizontal />} label="参数" active={active === "compose"} onClick={() => onChange("compose")} />
      <AndroidNavButton icon={<ImageIcon />} label="画布" active={active === "canvas"} onClick={() => onChange("canvas")} />
      <AndroidNavButton icon={<History />} label="历史" active={active === "history"} onClick={() => onChange("history")} />
    </nav>
  );
}

function AndroidBottomNav({
  active,
  onChange,
}: {
  active: "compose" | "canvas" | "history";
  onChange: (value: "compose" | "canvas" | "history") => void;
}) {
  return (
    <nav className="android-bottom-nav" aria-label="Android navigation">
      <AndroidNavButton icon={<SlidersHorizontal />} label="参数" active={active === "compose"} onClick={() => onChange("compose")} />
      <AndroidNavButton icon={<ImageIcon />} label="画布" active={active === "canvas"} onClick={() => onChange("canvas")} />
      <AndroidNavButton icon={<History />} label="历史" active={active === "history"} onClick={() => onChange("history")} />
    </nav>
  );
}

function AndroidNavButton({
  icon,
  label,
  active,
  onClick,
}: {
  icon: ReactElement;
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button type="button" className={`android-nav-button ${active ? "active" : ""}`} onClick={onClick} aria-current={active ? "page" : undefined}>
      <span className="android-nav-icon">{icon}</span>
      <span className="android-nav-label">{label}</span>
    </button>
  );
}

export default App;
