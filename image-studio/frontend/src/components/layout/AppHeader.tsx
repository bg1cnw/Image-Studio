import { useStudioStore } from "../../state/studioStore";
import { OpenExternalURL } from "../../../wailsjs/go/backend/Service";

const REPO_URL = "https://github.com/RoseKhlifa/Image-Studio";

export function AppHeader() {
  const { fullscreen, theme, setTheme, pushToast, workspaces, newWorkspace } = useStudioStore();
  if (fullscreen) return null;

  return (
    <header className="app-header">
      <div className="app-header-left">
        <div className="brand">
          <span className="brand-logo-dot" />
          <span className="brand-name">Image Studio</span>
        </div>
      </div>
      <div className="app-header-right">
        <button
          className="icon-btn"
          title={workspaces.length > 1 ? `${workspaces.length} 个标签 · 新建` : "新建标签"}
          onClick={() => newWorkspace()}
        >
          +{workspaces.length > 1 && <span className="badge">{workspaces.length}</span>}
        </button>
        <button
          className="icon-btn"
          title={theme === "dark" ? "浅色主题" : "深色主题"}
          onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
        >
          {theme === "dark" ? "☀" : "🌙"}
        </button>
        <button
          className="icon-btn"
          title="GitHub"
          onClick={() => OpenExternalURL(REPO_URL).catch(() => pushToast("无法打开浏览器", "error"))}
        >
          ⌭
        </button>
      </div>
    </header>
  );
}
