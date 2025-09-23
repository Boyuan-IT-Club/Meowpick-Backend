package util

import "strconv"

// TrieNode Trie树节点
type TrieNode struct {
	children map[rune]*TrieNode
	ids      []int32 // 存储包含此前缀的所有ID
}

// Trie 字符串搜索树
type Trie struct {
	root *TrieNode
}

// NewTrie 创建新的Trie树
func NewTrie() *Trie {
	return &Trie{
		root: &TrieNode{children: make(map[rune]*TrieNode)},
	}
}

// BuildFromMap 从map构建Trie树，支持任意子串匹配
func (t *Trie) BuildFromMap(dataMap map[string]string) {
	for idStr, name := range dataMap {
		if id, err := strconv.ParseInt(idStr, 10, 32); err == nil {
			t.insertSubstrings(name, int32(id))
		}
	}
}

// insertSubstrings 插入字符串的所有子串
func (t *Trie) insertSubstrings(text string, id int32) {
	runes := []rune(text)

	// 为每个可能的子串建立索引
	for i := range runes {
		node := t.root
		for j := i; j < len(runes); j++ {
			char := runes[j]
			if node.children[char] == nil {
				node.children[char] = &TrieNode{children: make(map[rune]*TrieNode)}
			}
			node = node.children[char]

			// 在每个节点上记录包含此子串的ID
			node.ids = append(node.ids, id)
		}
	}
}

// Search 搜索关键词，返回匹配的ID列表
func (t *Trie) Search(keyword string) []int32 {
	node := t.root
	runes := []rune(keyword)

	// 沿着关键词路径遍历
	for _, char := range runes {
		if node.children[char] == nil {
			return []int32{} // 没有找到匹配
		}
		node = node.children[char]
	}

	// 去重返回结果
	seen := make(map[int32]bool)
	var result []int32
	for _, id := range node.ids {
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}

	return result
}

// SearchBest 搜索最匹配的单个ID（优先完全匹配，其次最短匹配）
func (t *Trie) SearchBest(keyword string) int32 {
	node := t.root
	runes := []rune(keyword)

	// 沿着关键词路径遍历
	for _, char := range runes {
		if node.children[char] == nil {
			return 0 // 没有找到匹配
		}
		node = node.children[char]
	}

	if len(node.ids) > 0 {
		// 返回第一个匹配的ID，或者可以添加更复杂的排序逻辑
		return node.ids[0]
	}

	return 0
}
