import { useStudioStore } from "../../state/studioStore";
import { OpenExternalURL, OpenOutputDir } from "../../../wailsjs/go/backend/Service";

const REPO_URL = "https://github.com/RoseKhlifa/Image-Studio";
const ISSUES_URL = "https://github.com/RoseKhlifa/Image-Studio/issues";
const VERSION = "0.1.0";

export function FooterBar() {
  const { fullscreen, history, runningJobs, isRunning, pushToast } = useStudioStore();
  if (fullscreen) return null;

  const monthCount = history.filter(
    (h) => Date.now() - h.createdAt < 30 * 24 * 3600 * 1000,
  ).length;

  function open(url: string) {
    OpenExternalURL(url).catch(() => pushToast("无法打开浏览器", "error"));
  }

  return (
    <footer className="footer-bar">
      <div className="footer-left">
        <button className="footer-link" onClick={() => OpenOutputDir().catch(() => undefined)}>
          📂 打开输出目录
        </button>
        <button className="footer-link" onClick={() => open(REPO_URL)}>
          ⌭ GitHub
        </button>
        <button className="footer-link" onClick={() => open(ISSUES_URL)}>
          💬 反馈 / Issues
        </button>
      </div>
      <div className="footer-mid">
        <span className="footer-stat">
          <span className="footer-stat-label">本月生成</span>
          <span className="footer-stat-val">{monthCount} 张</span>
        </span>
        <span className="footer-sep">·</span>
        <span className="footer-stat">
          <span className="footer-stat-label">历史总数</span>
          <span className="footer-stat-val">{history.length}</span>
        </span>
        {isRunning && (
          <>
            <span className="footer-sep">·</span>
            <span className="footer-stat">
              <span className="footer-stat-label">并发</span>
              <span className="footer-stat-val" style={{ color: "var(--accent)" }}>{runningJobs.length}</span>
            </span>
          </>
        )}
      </div>
      <div className="footer-right">
        <span className="footer-stat-label">{isRunning ? "运行中" : "就绪"}</span>
        <span className={`status-dot ${isRunning ? "busy" : ""}`} />
        <span className="footer-version">v{VERSION}</span>
      </div>
    </footer>
  );
}
