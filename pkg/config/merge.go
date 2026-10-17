package config

import (
	"fmt"
	"reflect"
)

// MergeModules 合并多个模块配置，检测 key 冲突
func MergeModules(modules []map[string]interface{}) (map[string]interface{}, []Collision, error) {
	merged := make(map[string]interface{})
	var collisions []Collision

	for i, mod := range modules {
		c, err := mergeMap(merged, mod, fmt.Sprintf("module[%d]", i))
		if err != nil {
			return nil, nil, err
		}
		collisions = append(collisions, c...)
	}
	return merged, collisions, nil
}

type Collision struct {
	Path   string      `json:"path"`
	Values []interface{} `json:"values"`
	Source string      `json:"source"`
}

func mergeMap(dst, src map[string]interface{}, srcPath string) ([]Collision, error) {
	var collisions []Collision
	for k, v := range src {
		path := srcPath + "." + k
		if existing, ok := dst[k]; ok {
			// Check if both are maps -> recursive merge
			em, eok := existing.(map[string]interface{})
			vm, vok := v.(map[string]interface{})
			if eok && vok {
				c, err := mergeMap(em, vm, path)
				if err != nil {
					return nil, err
				}
				collisions = append(collisions, c...)
				continue
			}
			// Collision detected
			collisions = append(collisions, Collision{
				Path:   path,
				Values: []interface{}{existing, v},
				Source: srcPath,
			})
			// In v0.1.0, later value wins with warning
			dst[k] = v
		} else {
			dst[k] = v
		}
	}
	return collisions, nil
}

// ValidateNoOverlap 验证两组配置无重叠 key
func ValidateNoOverlap(a, b map[string]interface{}) []string {
	var overlaps []string
	for k := range a {
		if _, ok := b[k]; ok {
			overlaps = append(overlaps, k)
		}
	}
	return overlaps
}

// DeepEqualKeys 返回两个 map 的共同 key
func DeepEqualKeys(a, b map[string]interface{}) []string {
	var common []string
	for k := range a {
		if _, ok := b[k]; ok {
			if reflect.DeepEqual(a[k], b[k]) {
				common = append(common, k)
			}
		}
	}
	return common
}
