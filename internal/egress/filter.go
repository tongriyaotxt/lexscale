// Package egress 保密审查——法律版"出口过滤"。
//
// 仿 ARTEX 的 egress 滑窗哈希设计：
//   - 注册案件敏感信息（当事人姓名、身份证号、银行账号、商业秘密关键词）的 SHA256 指纹
//   - Worker 产出物（文书/分析结论）外发前过指纹匹配
//   - 命中即 deny——防止 A 案信息被写进 B 案文书（防串案）
//   - 只存哈希不存原文，常量时间比对防时序侧信道
package egress

import (
	"crypto/sha256"
	"crypto/subtle"
	"sync"
)

// Filter 保密过滤器
type Filter struct {
	mu          sync.RWMutex
	fingerprints map[string][]fingerprint // matterID → 指纹列表（按案件隔离）
}

// fingerprint 敏感信息指纹
type fingerprint struct {
	hash   [8]byte // SHA256 前 8 字节
	length int     // 原文长度（辅助匹配）
	label  string  // 标签（如"身份证号-张三"），便于审计日志
}

// New 创建过滤器
func New() *Filter {
	return &Filter{
		fingerprints: make(map[string][]fingerprint),
	}
}

// RegisterSensitive 为案件注册敏感信息（只存指纹不存原文）
func (f *Filter) RegisterSensitive(matterID, plaintext, label string) {
	h := sha256.Sum256([]byte(plaintext))
	fp := fingerprint{
		hash:   [8]byte(h[:8]),
		length: len(plaintext),
		label:  label,
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.fingerprints[matterID] = append(f.fingerprints[matterID], fp)
}

// Scan 扫描文本是否包含当前案件的敏感信息
// 返回命中的标签列表（空 = 干净）
func (f *Filter) Scan(matterID, text string) []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	fps, ok := f.fingerprints[matterID]
	if !ok {
		return nil
	}

	var hits []string
	// 滑窗：对每个指纹检查文本中所有同长度窗口的哈希
	// 成本 O(文本长 × 指纹数)，文书通常 KB 级，开销可忽略
	for _, fp := range fps {
		if len(text) < fp.length {
			continue
		}
		for i := 0; i <= len(text)-fp.length; i++ {
			window := text[i : i+fp.length]
			h := sha256.Sum256([]byte(window))
			if subtle.ConstantTimeCompare(h[:8], fp.hash[:]) == 1 {
				hits = append(hits, fp.label)
				break
			}
		}
	}
	return hits
}

// ScanCrossMatter 串案检测：检查文本是否包含【其他案件】的敏感信息
// 这是防串案的核心——A 案文书绝不能出现 B 案当事人的信息
func (f *Filter) ScanCrossMatter(currentMatterID, text string) map[string][]string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	violations := make(map[string][]string)
	for matterID, fps := range f.fingerprints {
		if matterID == currentMatterID {
			continue // 本案件的敏感信息出现在本案件文书里是正常的
		}
		var hits []string
		for _, fp := range fps {
			if len(text) < fp.length {
				continue
			}
			for i := 0; i <= len(text)-fp.length; i++ {
				window := text[i : i+fp.length]
				h := sha256.Sum256([]byte(window))
				if subtle.ConstantTimeCompare(h[:8], fp.hash[:]) == 1 {
					hits = append(hits, fp.label)
					break
				}
			}
		}
		if len(hits) > 0 {
			violations[matterID] = hits
		}
	}
	return violations
}

// MatterCount 返回已注册指纹的案件数
func (f *Filter) MatterCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.fingerprints)
}
