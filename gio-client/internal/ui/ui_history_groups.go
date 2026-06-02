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

type historyDayGroup struct {
	Key     string
	Label   string
	Entries []historyPromptEntry
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

func buildHistoryDayGroups(items []sharedCompat.HistoryItem) []historyDayGroup {
	entries := buildHistoryPromptEntries(items)
	groups := make([]historyDayGroup, 0, len(entries))
	indexByKey := map[string]int{}
	for _, entry := range entries {
		representative := historyEntryRepresentative(entry)
		key := formatHistoryDay(representative.CreatedAt)
		if idx, ok := indexByKey[key]; ok {
			groups[idx].Entries = append(groups[idx].Entries, entry)
			continue
		}
		indexByKey[key] = len(groups)
		groups = append(groups, historyDayGroup{
			Key:     key,
			Label:   key,
			Entries: []historyPromptEntry{entry},
		})
	}
	return groups
}

func historyEntryRepresentative(entry historyPromptEntry) sharedCompat.HistoryItem {
	if entry.Kind == "group" {
		return entry.Group.Representative
	}
	return entry.Item
}

func historyItemsByIDs(items []sharedCompat.HistoryItem, ids []string) []sharedCompat.HistoryItem {
	if len(ids) == 0 {
		return nil
	}
	index := make(map[string]sharedCompat.HistoryItem, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		index[item.ID] = item
	}
	out := make([]sharedCompat.HistoryItem, 0, len(ids))
	for _, id := range ids {
		if item, ok := index[strings.TrimSpace(id)]; ok {
			out = append(out, item)
		}
	}
	return out
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
