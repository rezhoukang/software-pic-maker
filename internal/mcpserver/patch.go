package mcpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type PatchOperation struct {
	Op    string `json:"op" jsonschema:"add, replace, remove, or test (RFC 6902 subset)"`
	Path  string `json:"path" jsonschema:"JSON Pointer, e.g. /model/use_cases/0/name. Array add supports /-."`
	Value any    `json:"value,omitempty" jsonschema:"JSON value for add/replace/test; not used by remove"`
}
type PatchDiagramArgs struct {
	Path       string           `json:"path" jsonschema:"Workspace-relative diagram .json path"`
	Revision   string           `json:"revision" jsonschema:"Exact revision returned by read_diagram"`
	Operations []PatchOperation `json:"operations" jsonschema:"Atomic add/replace/remove/test operations. Preserve IDs and maintain affected references."`
}

func pointerTokens(pointer string) ([]string, error) {
	if pointer == "" {
		return nil, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, fmt.Errorf("JSON Pointer 必须以 / 开头")
	}
	parts := strings.Split(pointer[1:], "/")
	for i, part := range parts {
		for j := 0; j < len(part); j++ {
			if part[j] == '~' {
				j++
				if j >= len(part) || part[j] != '0' && part[j] != '1' {
					return nil, fmt.Errorf("无效 JSON Pointer 转义")
				}
			}
		}
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
	}
	return parts, nil
}
func patchAt(node any, tokens []string, op PatchOperation) (any, error) {
	if len(tokens) == 0 {
		switch op.Op {
		case "add", "replace":
			return op.Value, nil
		case "remove":
			return nil, nil
		case "test":
			if !bytes.Equal(jsonBytes(node), jsonBytes(op.Value)) {
				return nil, fmt.Errorf("test 失败")
			}
			return node, nil
		}
		return nil, fmt.Errorf("不支持的操作 %s", op.Op)
	}
	key, tail := tokens[0], tokens[1:]
	switch n := node.(type) {
	case map[string]any:
		value, exists := n[key]
		if !exists && (len(tail) > 0 || op.Op != "add") {
			return nil, fmt.Errorf("路径字段不存在：%s", key)
		}
		if len(tail) == 0 && op.Op == "remove" {
			delete(n, key)
			return n, nil
		}
		updated, err := patchAt(value, tail, op)
		if err != nil {
			return nil, err
		}
		n[key] = updated
		return n, nil
	case []any:
		index := len(n)
		if key != "-" {
			parsed, err := strconv.Atoi(key)
			if err != nil || strconv.Itoa(parsed) != key {
				return nil, fmt.Errorf("无效数组下标：%s", key)
			}
			index = parsed
		} else if len(tail) > 0 || op.Op != "add" {
			return nil, fmt.Errorf("/- 仅可用于数组追加")
		}
		if index < 0 || index > len(n) || index == len(n) && (len(tail) > 0 || op.Op != "add") {
			return nil, fmt.Errorf("数组下标越界")
		}
		if len(tail) == 0 {
			if op.Op == "add" {
				n = append(n, nil)
				copy(n[index+1:], n[index:len(n)-1])
				n[index] = op.Value
				return n, nil
			}
			if op.Op == "remove" {
				return append(n[:index], n[index+1:]...), nil
			}
		}
		updated, err := patchAt(n[index], tail, op)
		if err != nil {
			return nil, err
		}
		n[index] = updated
		return n, nil
	}
	return nil, fmt.Errorf("路径经过了非对象或非数组值")
}
func (s *projectService) patch(args PatchDiagramArgs) (any, error) {
	if len(args.Operations) == 0 || len(args.Operations) > 200 {
		return nil, fmt.Errorf("每次修改需包含 1～200 个操作")
	}
	doc, err := s.store.Read(args.Path)
	if err != nil {
		return nil, err
	}
	if args.Revision == "" || args.Revision != doc.Revision {
		return nil, fmt.Errorf("文件版本已变更，请重新 read_diagram；未应用任何修改")
	}
	var node any
	json.Unmarshal(jsonBytes(doc.Diagram), &node)
	for i, op := range args.Operations {
		tokens, tokenErr := pointerTokens(op.Path)
		if tokenErr != nil {
			return nil, tokenErr
		}
		node, err = patchAt(node, tokens, op)
		if err != nil {
			return nil, fmt.Errorf("操作 %d（%s %s）失败：%w；未写入文件", i+1, op.Op, op.Path, err)
		}
	}
	diagram, ok := node.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("修改后必须仍为完整图表对象")
	}
	return s.save(SaveDiagramArgs{Path: doc.Path, Diagram: diagram, Revision: args.Revision})
}
