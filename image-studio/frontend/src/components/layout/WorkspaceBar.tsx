import { useState } from "react";
import { useStudioStore } from "../../state/studioStore";

// Browser-tab style bar across the very top of the window. Each tab is a
// fully independent workspace: prompt, sources, currentImage, params. History
// is shared across tabs.
export function WorkspaceBar() {
  const { workspaces, activeWorkspaceId, newWorkspace, switchWorkspace, closeWorkspace, renameWorkspace, fullscreen } = useStudioStore();
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editingName, setEditingName] = useState("");

  if (fullscreen) return null;
  // Single-workspace looks cleaner without a tab bar. The "+" lives on top
  // of the AppHeader instead — but here we just hide the whole strip.
  if (workspaces.length <= 1) return null;

  function startRename(id: string, currentName: string) {
    setEditingId(id);
    setEditingName(currentName);
  }
  function commitRename() {
    if (editingId) {
      renameWorkspace(editingId, editingName.trim() || "未命名");
    }
    setEditingId(null);
  }

  return (
    <div className="workspace-bar">
      {workspaces.map((w) => {
        const active = w.id === activeWorkspaceId;
        const isEditing = editingId === w.id;
        return (
          <div
            key={w.id}
            className={`workspace-tab ${active ? "active" : ""}`}
            onClick={() => !isEditing && switchWorkspace(w.id)}
            onDoubleClick={() => startRename(w.id, w.name)}
            title="双击重命名"
          >
            {isEditing ? (
              <input
                className="workspace-tab-input"
                value={editingName}
                autoFocus
                onChange={(e) => setEditingName(e.target.value)}
                onBlur={commitRename}
                onKeyDown={(e) => {
                  if (e.key === "Enter") commitRename();
                  if (e.key === "Escape") setEditingId(null);
                }}
              />
            ) : (
              <span className="workspace-tab-name">{w.name}</span>
            )}
            {workspaces.length > 1 && !isEditing && (
              <button
                className="workspace-tab-close"
                onClick={(e) => {
                  e.stopPropagation();
                  closeWorkspace(w.id);
                }}
                title="关闭(至少保留 1 个)"
              >
                ×
              </button>
            )}
          </div>
        );
      })}
      <button
        className="workspace-tab-add"
        onClick={() => newWorkspace()}
        title="新建标签页"
      >
        +
      </button>
    </div>
  );
}
