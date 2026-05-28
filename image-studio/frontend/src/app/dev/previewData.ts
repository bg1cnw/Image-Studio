import type {
  HistoryItem,
  SourceImage,
  UpstreamProfile,
  Workspace,
} from "../../types/domain";

export type PreviewScenario = "mac-workspace";

const PREVIEW_PNG_B64 =
  "iVBORw0KGgoAAAANSUhEUgAAAEAAAABACAIAAAAlC+aJAAAAbUlEQVR4nO3PQQ3AIADAQMD2/hdwwZE8SBR0ztn3jJ9Zd7wD8E1gTWBNYE1gTWBNYE1gTWBNYE1gTWBNYE1gTWBNYE1gTWBNYE1gTWBNYE1gTWBNYE1gTWBNYE1gTWBNYE1gTWBNYE1gTWBNYF0X2AGCb5Q0aAAAAAElFTkSuQmCC";

export function readPreviewScenario(): PreviewScenario | null {
  if (typeof window === "undefined") return null;
  try {
    const params = new URLSearchParams(window.location.search);
    const preview = (params.get("preview") ?? "").trim().toLowerCase();
    return preview === "mac-workspace" ? "mac-workspace" : null;
  } catch {
    return null;
  }
}

function buildHistory(now: number): HistoryItem[] {
  const defs = [
    {
      id: "preview-history-1",
      prompt: "赛博雨夜角色海报，湿地街道反光，红青霓虹边缘光，35mm，电影感，超细节",
      revisedPrompt: "高对比、冷暖霓虹、边缘轮廓光、海报构图、主体占中",
      mode: "edit" as const,
      size: "2880x2880" as const,
      quality: "high" as const,
      negativePrompt: "模糊, 低清晰度, 脏污噪点",
      styleTag: "电影海报",
    },
    {
      id: "preview-history-2",
      prompt: "产品棚拍样张，银色耳机置于磨砂台面，柔光箱高光干净，商业摄影",
      revisedPrompt: "极简背景、金属反射控制、轻微俯拍、留白构图",
      mode: "generate" as const,
      size: "2048x2048" as const,
      quality: "medium" as const,
      negativePrompt: "畸变, 文字, 水印",
      styleTag: "商业摄影",
    },
    {
      id: "preview-history-3",
      prompt: "复古室内肖像，暖调台灯，胶片颗粒，人物半身，背景略虚化",
      revisedPrompt: "暖黄侧光、低饱和、胶片肤色、细节保留",
      mode: "edit" as const,
      size: "2048x2048" as const,
      quality: "medium" as const,
      negativePrompt: "过曝, 手部畸形",
      styleTag: "胶片人像",
    },
    {
      id: "preview-history-4",
      prompt: "建筑外观黄昏蓝调时刻，玻璃幕墙反射天光，广角透视校正",
      revisedPrompt: "蓝金时刻、垂直线控制、通透玻璃、干净天空",
      mode: "generate" as const,
      size: "1024x1024" as const,
      quality: "high" as const,
      negativePrompt: "低清, 透视变形",
      styleTag: "建筑表现",
    },
    {
      id: "preview-history-5",
      prompt: "桌面静物，黑胶唱片与咖啡杯，晨光从百叶窗斜切进来，生活方式摄影",
      revisedPrompt: "浅景深、微尘漂浮、木质纹理、低噪点",
      mode: "edit" as const,
      size: "1024x1024" as const,
      quality: "medium" as const,
      negativePrompt: "重影, 杂乱背景",
      styleTag: "静物生活",
    },
    {
      id: "preview-history-6",
      prompt: "未来概念载具三视图展示，白底信息板风格，结构线清晰，工业设计稿",
      revisedPrompt: "信息图式布局、边框精简、材质标注感、线稿利落",
      mode: "generate" as const,
      size: "1024x1024" as const,
      quality: "medium" as const,
      negativePrompt: "糊边, 文字乱码",
      styleTag: "工业设计",
    },
  ];

  return defs.map((item, index) => ({
    ...item,
    imageB64: PREVIEW_PNG_B64,
    outputFormat: "png",
    createdAt: now - index * 55 * 60 * 1000,
    savedPath: `/tmp/${item.id}.png`,
    rawPath: `/tmp/${item.id}.json`,
    seed: 3200 + index,
    elapsedSec: 7 + index,
  }));
}

function buildSources(): SourceImage[] {
  return [
    {
      path: "/tmp/preview-source-a.png",
      name: "原图-A.png",
      size: 16384,
      imageB64: PREVIEW_PNG_B64,
    },
    {
      path: "/tmp/preview-source-b.png",
      name: "构图参考-B.png",
      size: 16384,
      imageB64: PREVIEW_PNG_B64,
    },
  ];
}

function buildPreviewProfile(now: number): UpstreamProfile {
  return {
    id: "preview-profile",
    name: "Preview Responses",
    apiMode: "responses",
    requestPolicy: "openai",
    baseURL: "https://code1.linzefeng.top",
    textModelID: "gpt-4.1-mini",
    imageModelID: "gpt-image-1",
    concurrencyLimit: 1,
    createdAt: now,
    lastUsedAt: now,
  };
}

function buildWorkspace(
  workspaceId: string,
  currentImage: HistoryItem,
  sources: SourceImage[],
): Workspace {
  return {
    id: workspaceId,
    name: "联调样例",
    prompt: currentImage.prompt,
    negativePrompt: currentImage.negativePrompt ?? "",
    mode: "edit",
    size: "2880x2880",
    quality: "high",
    outputFormat: "png",
    seed: 3200,
    batchCount: 1,
    sources,
    currentImageId: currentImage.id,
    batchResultIds: [],
    resultGridOpen: false,
    runningJobIds: [],
    jobsTotal: 0,
    jobsCompleted: 0,
    progress: null,
    lastLogLine: "",
    errorMessage: null,
    errorRawPath: null,
    lastPayload: null,
  };
}

export function buildMacWorkspacePreview(workspaceId: string) {
  const now = Date.now();
  const history = buildHistory(now);
  const currentImage = history[0];
  const sources = buildSources();
  const profile = buildPreviewProfile(now);
  const workspace = buildWorkspace(workspaceId, currentImage, sources);

  return {
    profile,
    history,
    currentImage,
    sources,
    workspace,
  };
}
