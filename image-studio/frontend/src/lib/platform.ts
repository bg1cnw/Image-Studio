import { useEffect, useState } from "react";

export type UIPlatform = "macos" | "windows" | "linux" | "ios" | "android" | "web";
export type UITargetPlatform = UIPlatform | "android-pad";
export type UIFamily = "apple" | "fluent" | "android" | "generic";

const ANDROID_MEDIUM_WIDTH_DP = 600;
const ANDROID_EXPANDED_WIDTH_DP = 840;

function fromOverride(raw?: string): UITargetPlatform | null {
  switch ((raw ?? "").trim().toLowerCase()) {
    case "mac":
    case "macos":
    case "darwin":
      return "macos";
    case "windows":
    case "win":
    case "win32":
      return "windows";
    case "linux":
      return "linux";
    case "ios":
      return "ios";
    case "android":
      return "android";
    case "android-pad":
    case "android_tablet":
    case "android-tablet":
    case "tablet":
    case "pad":
      return "android-pad";
    case "web":
      return "web";
    default:
      return null;
  }
}

function detectRawTargetPlatform(): UITargetPlatform {
  const env = (import.meta as ImportMeta & { env?: Record<string, string | undefined> }).env;
  const override = fromOverride(env?.VITE_TARGET_PLATFORM);
  if (override && override !== "android") return override;
  if (typeof navigator === "undefined") return override ?? "web";

  const uaDataPlatform = (navigator as Navigator & { userAgentData?: { platform?: string } }).userAgentData?.platform ?? "";
  const platform = navigator.platform ?? "";
  const userAgent = navigator.userAgent ?? "";
  const source = `${uaDataPlatform} ${platform} ${userAgent}`.toLowerCase();

  if (/iphone|ipad|ipod|ios/.test(source)) return "ios";
  if (/android/.test(source)) return "android";
  if (/mac/.test(source)) return "macos";
  if (/win/.test(source)) return "windows";
  if (/linux|x11/.test(source)) return "linux";
  return "web";
}

function detectAndroidAdaptiveTarget(): UITargetPlatform {
  if (typeof window === "undefined") return "android";
  const width = Math.max(0, window.innerWidth || 0);
  const height = Math.max(0, window.innerHeight || 0);
  const landscape = width >= height;

  if (width >= ANDROID_EXPANDED_WIDTH_DP) return "android-pad";
  if (width >= ANDROID_MEDIUM_WIDTH_DP && landscape) return "android-pad";
  return "android";
}

function normalizeRuntimePlatform(value: UITargetPlatform): UIPlatform {
  if (value === "android-pad") return "android";
  return value;
}

function familyForTarget(value: UITargetPlatform): UIFamily {
  switch (value) {
    case "macos":
    case "ios":
      return "apple";
    case "android":
    case "android-pad":
      return "android";
    case "windows":
      return "fluent";
    default:
      return "generic";
  }
}

const rawTargetPlatform = detectRawTargetPlatform();

export function targetPlatformForViewport(): UITargetPlatform {
  if (rawTargetPlatform !== "android") return rawTargetPlatform;
  return detectAndroidAdaptiveTarget();
}

export function readRuntimePlatformState() {
  const target = targetPlatformForViewport();
  const platform = normalizeRuntimePlatform(target);
  const uiFamily = familyForTarget(target);
  return {
    targetPlatform: target,
    platform,
    uiFamily,
    isAndroid: platform === "android",
    isAndroidPad: target === "android-pad",
    isAndroidPhone: target === "android",
    isMac: platform === "macos" || platform === "ios",
    isWindows: platform === "windows",
    usesAppleUI: uiFamily === "apple",
    usesAndroidUI: uiFamily === "android",
  };
}

const initialState = readRuntimePlatformState();

export const targetPlatform = initialState.targetPlatform;
export const platform = initialState.platform;
export const uiFamily = initialState.uiFamily;
export const isAndroid = initialState.isAndroid;
export const isAndroidPad = initialState.isAndroidPad;
export const isAndroidPhone = initialState.isAndroidPhone;
export const usesAppleUI = initialState.usesAppleUI;
export const usesAndroidUI = initialState.usesAndroidUI;
export const isMac = initialState.isMac;
export const isWindows = initialState.isWindows;

export function applyPlatformAttributes(root: HTMLElement = document.documentElement) {
  if (!root) return;
  const state = readRuntimePlatformState();
  root.dataset.platform = state.platform;
  root.dataset.targetPlatform = state.targetPlatform;
  root.dataset.uiFamily = state.uiFamily;
}

export function useRuntimePlatform() {
  const [state, setState] = useState(() => readRuntimePlatformState());
  useEffect(() => {
    if (rawTargetPlatform !== "android") return;
    const update = () => {
      applyPlatformAttributes();
      setState(readRuntimePlatformState());
    };
    const viewport = window.visualViewport;
    update();
    window.addEventListener("resize", update);
    window.addEventListener("orientationchange", update);
    viewport?.addEventListener("resize", update);
    return () => {
      window.removeEventListener("resize", update);
      window.removeEventListener("orientationchange", update);
      viewport?.removeEventListener("resize", update);
    };
  }, []);
  return state;
}

export const primaryModifierLabel = isMac ? "⌘" : "Ctrl";
export const redoShortcutLabel = isMac ? "⇧⌘Z" : "Ctrl+Shift+Z";
export const newTabShortcutLabel = isMac ? "⌘N" : "Ctrl+N";
export const closeTabShortcutLabel = isMac ? "⌘W" : "Ctrl+W";
export const submitShortcutLabel = isMac ? "⌘Enter" : "Ctrl+Enter";
export const copyShortcutLabel = isMac ? "⌘C" : "Ctrl+C";
export const pasteShortcutLabel = isMac ? "⌘V" : "Ctrl+V";
export const undoShortcutLabel = isMac ? "⌘Z" : "Ctrl+Z";
export const fullscreenShortcutLabel = isMac ? "⌃⌘F" : "F11";

export function platformOutputRootLabel() {
  const state = readRuntimePlatformState();
  if (state.isAndroidPad) return "应用图片目录 / MediaStore Pictures";
  if (state.isAndroidPhone) return "系统下载 / 分享面板";
  if (state.isMac) return "~/Pictures/Image Studio";
  if (state.isWindows) return "%APPDATA%\\image-studio";
  return "~/Pictures/Image Studio";
}

export function platformRuntimeLabel() {
  const state = readRuntimePlatformState();
  if (state.isAndroidPad) return "Android Pad WebView / Material 3 adaptive frontend";
  if (state.isAndroidPhone) return "Android WebView / Material 3 phone frontend";
  if (state.isMac) return "Wails v2 / WKWebView";
  if (state.isWindows) return "Wails v2 / WebView2";
  return "Wails v2 / WebKitGTK";
}
