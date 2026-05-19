import { useStudioStore } from "../../state/studioStore";

// Minimal empty state — no decorative cards, no example grid. Just a quiet
// hint that nothing has been generated yet and how to proceed.
export function EmptyState() {
  const importImageFile = useStudioStore((s) => s.importImageFile);

  function onFilePick(e: React.ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0];
    if (f) importImageFile(f);
    e.target.value = "";
  }

  return (
    <div className="empty-stage">
      <div className="empty-stage-inner">
        <div className="empty-stage-icon">🖼</div>
        <h2 className="empty-stage-title">还没有图片</h2>
        <p className="empty-stage-sub">
          在左侧填好 prompt 后点「生成」, 或者拖入一张本地图片来编辑。
        </p>
        <label className="empty-stage-link">
          <input type="file" accept="image/png,image/jpeg,image/webp" onChange={onFilePick} style={{ display: "none" }} />
          选择本地图片
        </label>
      </div>
    </div>
  );
}
