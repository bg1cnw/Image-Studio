import type { ReactElement } from "react";
import { History, Image as ImageIcon, SlidersHorizontal } from "lucide-react";
import { ControlPanel } from "../../components/panel/ControlPanel";
import { Toolbar } from "../../components/canvas/Toolbar";
import { SourceStrip } from "../../components/canvas/SourceStrip";
import { CanvasStage } from "../../components/canvas/CanvasStage";
import { StatusBar } from "../../components/canvas/StatusBar";
import { HistoryRail } from "../../components/history/HistoryRail";
import type { AndroidView } from "../types";

export function AndroidShell({
  fullscreen,
  isPad,
  androidView,
  onChangeView,
}: {
  fullscreen: boolean;
  isPad: boolean;
  androidView: AndroidView;
  onChangeView: (value: AndroidView) => void;
}) {
  return (
    <>
      <div
        className={`studio ${fullscreen ? "fullscreen" : ""} ${isPad ? "android-pad" : "android-phone"}`}
        data-android-view={androidView}
        data-android-target={isPad ? "android-pad" : "android"}
      >
        {isPad && !fullscreen && <AndroidRail active={androidView} onChange={onChangeView} />}
        <ControlPanel />
        <div className="canvas-shell">
          <Toolbar />
          <SourceStrip />
          <CanvasStage />
          <StatusBar />
        </div>
        <HistoryRail />
      </div>
      {!isPad && !fullscreen && <AndroidBottomNav active={androidView} onChange={onChangeView} />}
    </>
  );
}

function AndroidRail({
  active,
  onChange,
}: {
  active: AndroidView;
  onChange: (value: AndroidView) => void;
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
  active: AndroidView;
  onChange: (value: AndroidView) => void;
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
