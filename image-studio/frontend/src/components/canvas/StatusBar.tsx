import { useStudioStore } from "../../state/studioStore";

function fmtBytes(b: number): string {
  if (b < 1024) return `${b} B`;
  if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)} KB`;
  return `${(b / 1024 / 1024).toFixed(1)} MB`;
}

export function StatusBar() {
  const { isRunning, progress, currentImage, logLines, viewZoom, recentDurations, jobsTotal, jobsCompleted, runningJobs } = useStudioStore();
  const zoomLabel = currentImage ? `${Math.round(viewZoom * 100)}%` : "";
  const avg = recentDurations.length > 0
    ? recentDurations.reduce((a, b) => a + b, 0) / recentDurations.length
    : 0;
  const eta = isRunning && progress && avg > 0
    ? Math.max(0, Math.round(avg - progress.elapsed))
    : null;

  if (isRunning) {
    return (
      <div className="statusbar">
        <span style={{ color: "var(--text)" }}>
          {progress ? `${progress.stage} · 已等待 ${progress.elapsed}s · 已接收 ${fmtBytes(progress.bytes)}` : "正在请求..."}
        </span>
        {jobsTotal > 1 && (
          <span style={{ color: "var(--accent)", fontWeight: 500 }}>
            并发 {runningJobs.length} 个,已完成 {jobsCompleted}/{jobsTotal}
          </span>
        )}
        {eta !== null && <span style={{ color: "var(--text-muted)" }}>预计剩余 ~{eta}s</span>}
        <div className="progress-bar" />
        <span style={{ color: "var(--text-muted)", maxWidth: "30%", overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{logLines[logLines.length - 1] ?? ""}</span>
      </div>
    );
  }
  if (currentImage) {
    const created = new Date(currentImage.createdAt).toLocaleTimeString();
    const metaParts: string[] = [];
    metaParts.push(currentImage.mode === "edit" ? "编辑" : "生成");
    metaParts.push(currentImage.size);
    metaParts.push(currentImage.quality);
    if (currentImage.elapsedSec) metaParts.push(`${currentImage.elapsedSec}s`);
    if (currentImage.seed) metaParts.push(`seed ${currentImage.seed}`);
    if (currentImage.styleTag) metaParts.push(`#${currentImage.styleTag}`);
    return (
      <div className="statusbar">
        <span>✓ {metaParts.join(" · ")} · {created}</span>
        {currentImage.revisedPrompt && (
          <span style={{ color: "var(--text-muted)", flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }} title={currentImage.revisedPrompt}>
            修订: {currentImage.revisedPrompt}
          </span>
        )}
        <span style={{ color: "var(--text-muted)", marginLeft: "auto" }}>缩放 {zoomLabel}</span>
      </div>
    );
  }
  return (
    <div className="statusbar">
      <span>准备就绪</span>
    </div>
  );
}
