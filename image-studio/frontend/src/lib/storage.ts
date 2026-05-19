// localStorage helpers for the API key, plus idb-keyval persistence for history.
// Key intentionally kept in localStorage (not IndexedDB) for sync read on boot.

import { get, set, del, keys } from "idb-keyval";
import type { HistoryItem } from "../types/domain";

const API_KEY = "gptcodex.apiKey";
const HISTORY_PREFIX = "history:";

export function loadAPIKey(): string {
  try {
    return localStorage.getItem(API_KEY) ?? "";
  } catch {
    return "";
  }
}

export function saveAPIKey(value: string): void {
  try {
    if (value) localStorage.setItem(API_KEY, value);
    else localStorage.removeItem(API_KEY);
  } catch {
    // ignore
  }
}

export async function persistHistoryItem(item: HistoryItem): Promise<void> {
  await set(HISTORY_PREFIX + item.id, item);
}

export async function removeHistoryItem(id: string): Promise<void> {
  await del(HISTORY_PREFIX + id);
}

export async function loadAllHistory(): Promise<HistoryItem[]> {
  const all = await keys();
  const items: HistoryItem[] = [];
  for (const k of all) {
    if (typeof k !== "string" || !k.startsWith(HISTORY_PREFIX)) continue;
    const v = await get<HistoryItem>(k);
    if (v) items.push(v);
  }
  items.sort((a, b) => b.createdAt - a.createdAt);
  return items;
}
