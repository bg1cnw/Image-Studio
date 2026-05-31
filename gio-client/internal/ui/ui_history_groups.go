package ui

import (
	"strings"

	sharedCompat "image-studio/shared/compat"
)

type historyPromptGroup struct {
	Key            string
	Prompt         string
	Representative sharedCompat.HistoryItem
	Items          []sharedCompat.HistoryItem
}

type historyPromptEntry struct {
	Kind  string
	Key   string
	Item  sharedCompat.HistoryItem
	Group historyPromptGroup
}

func normalizeHistoryPrompt(prompt string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(prompt)), " "))
}

func compactHistoryPrompt(prompt string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(prompt)), " ")
}

func buildHistoryPromptEntries(items []sharedCompat.HistoryItem) []historyPromptEntry {
	groups := make([]historyPromptGroup, 0, len(items))
	indexByKey := map[string]int{}
	for _, item := range items {
		normalized := normalizeHistoryPrompt(item.Prompt)
		key := "prompt:" + normalized
		if idx, ok := indexByKey[key]; ok {
			groups[idx].Items = append(groups[idx].Items, item)
			continue
		}
		indexByKey[key] = len(groups)
		groups = append(groups, historyPromptGroup{
			Key:            key,
			Prompt:         compactHistoryPrompt(item.Prompt),
			Representative: item,
			Items:          []sharedCompat.HistoryItem{item},
		})
	}

	entries := make([]historyPromptEntry, 0, len(groups))
	for _, group := range groups {
		if len(group.Items) > 1 {
			entries = append(entries, historyPromptEntry{
				Kind:  "group",
				Key:   group.Key,
				Group: group,
			})
			continue
		}
		entries = append(entries, historyPromptEntry{
			Kind:  "item",
			Key:   group.Representative.ID,
			Item:  group.Representative,
			Group: group,
		})
	}
	return entries
}

func historyPromptGroupContains(group historyPromptGroup, itemID string) bool {
	if strings.TrimSpace(itemID) == "" {
		return false
	}
	for _, item := range group.Items {
		if item.ID == itemID {
			return true
		}
	}
	return false
}

func findPromptGroupForItem(items []sharedCompat.HistoryItem, itemID string) (historyPromptGroup, bool) {
	if strings.TrimSpace(itemID) == "" {
		return historyPromptGroup{}, false
	}
	for _, entry := range buildHistoryPromptEntries(items) {
		group := entry.Group
		if group.Key == "" && entry.Item.ID != "" {
			group = historyPromptGroup{
				Key:            "prompt:" + normalizeHistoryPrompt(entry.Item.Prompt),
				Prompt:         compactHistoryPrompt(entry.Item.Prompt),
				Representative: entry.Item,
				Items:          []sharedCompat.HistoryItem{entry.Item},
			}
		}
		if historyPromptGroupContains(group, itemID) {
			return group, len(group.Items) > 0
		}
	}
	return historyPromptGroup{}, false
}
