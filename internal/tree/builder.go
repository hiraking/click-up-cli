// internal/tree/builder.go
package tree

import "github.com/hiraking/click-up-cli/internal/models"

// Build はフラットな TaskSummary スライスを受け取り、
// 親子関係を解決したツリー構造として返す。
// ParentID が nil のタスク、または ParentID に対応するタスクが
// スライス内に存在しないタスクをルートとして扱う。
func Build(tasks []models.TaskSummary) []models.TaskSummary {
	if len(tasks) == 0 {
		return []models.TaskSummary{}
	}

	// children マップ: parentID → []TaskSummary
	children := make(map[string][]models.TaskSummary)
	rootIDs := []string{}
	taskMap := make(map[string]models.TaskSummary, len(tasks))

	for _, t := range tasks {
		cp := t
		cp.Subtasks = []models.TaskSummary{}
		taskMap[cp.ID] = cp

		if cp.ParentID == nil {
			rootIDs = append(rootIDs, cp.ID)
		} else {
			children[*cp.ParentID] = append(children[*cp.ParentID], cp)
		}
	}

	// 孤立タスク（親が存在しない）もルートに追加（スライス順序で確定的に処理）
	for _, t := range tasks {
		if t.ParentID != nil {
			if _, exists := taskMap[*t.ParentID]; !exists {
				rootIDs = append(rootIDs, t.ID)
			}
		}
	}

	var buildTree func(id string) models.TaskSummary
	buildTree = func(id string) models.TaskSummary {
		node := taskMap[id]
		for _, child := range children[id] {
			node.Subtasks = append(node.Subtasks, buildTree(child.ID))
		}
		if node.Subtasks == nil {
			node.Subtasks = []models.TaskSummary{}
		}
		return node
	}

	result := make([]models.TaskSummary, 0, len(rootIDs))
	for _, id := range rootIDs {
		result = append(result, buildTree(id))
	}
	return result
}
