import { lazy, Suspense } from "react";
import { useStudioStore } from "../../state/studioStore";

const SavePromptModal = lazy(() => import("../../components/common/SavePromptModal").then((module) => ({ default: module.SavePromptModal })));

export function SavePromptGate() {
  const item = useStudioStore((s) => s.savePromptItem);
  if (!item) return null;
  return (
    <Suspense fallback={null}>
      <SavePromptModal />
    </Suspense>
  );
}
