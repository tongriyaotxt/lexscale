// Package main 律衡 CLI — 一键合同审查
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"lexscale/internal/graph"
)

// Finding 风险发现（CLI 输出格式）
type Finding struct {
	Level      string     `json:"level"`
	Title      string     `json:"title"`
	Severity   int        `json:"severity"`
	Likelihood int        `json:"likelihood"`
	Tier       string     `json:"tier"`
	Citations  []Citation `json:"citations,omitempty"`
	Reasoning  string     `json:"reasoning,omitempty"`
	Clause     string     `json:"clause,omitempty"`
}

// Citation 法条引用
type Citation struct {
	Source string `json:"source"`
	Excerpt string `json:"excerpt"`
}

// AnalysisResult 分析结果
type AnalysisResult struct {
	ContractName string    `json:"contract_name"`
	TotalFindings int      `json:"total_findings"`
	RedCount     int       `json:"red_count"`
	OrangeCount  int       `json:"orange_count"`
	YellowCount  int       `json:"yellow_count"`
	GreenCount   int       `json:"green_count"`
	Findings     []Finding `json:"findings"`
}

// 内置规则引擎（基于关键词匹配 + 法条库）
var ruleEngine = []Rule{
	// ===== 保密协议相关 =====
	{
		ID:       "nda-perpetual-confidentiality",
		Name:     "保密期限永久",
		Severity: 4,
		Likelihood: 3,
		Keywords: []string{"永久", "无限期", "perpetual"},
		Context:  []string{"保密", "confidentiality"},
		Statute:  "劳动合同法第23条",
		Excerpt:  "竞业限制期限不得超过二年",
		Advice:   "保密期限约定为'永久'可能因显失公平被法院调整，建议约定合理期限（通常2-5年）",
	},
	{
		ID:       "nda-no-exclusions",
		Name:     "保密信息定义过宽",
		Severity: 2,
		Likelihood: 4,
		Keywords: []string{"一切信息", "所有信息", "any information"},
		Context:  []string{"保密", "confidential"},
		Statute:  "反不正当竞争法第9条",
		Excerpt:  "商业秘密是指不为公众所知悉、具有商业价值并经权利人采取相应保密措施的技术信息、经营信息等商业信息",
		Advice:   "保密信息定义过于宽泛可能难以执行，建议明确列举具体类型",
	},
	
	// ===== 违约金相关 =====
	{
		ID:       "excessive-penalty",
		Name:     "违约金过高",
		Severity: 5,
		Likelihood: 4,
		Keywords: []string{"500%", "10倍", "5000%", "十倍", "500％"},
		Context:  []string{"违约金", "违约", "penalty"},
		Statute:  "民法典第585条",
		Excerpt:  "约定的违约金过分高于造成的损失的，人民法院或者仲裁机构可以根据当事人的请求予以适当减少",
		Advice:   "违约金远超实际损失（通常超过30%），法院可能调低，建议约定合理比例",
	},
	{
		ID:       "penalty-no-cap",
		Name:     "违约金无上限",
		Severity: 3,
		Likelihood: 3,
		Keywords: []string{"全部损失", "一切损失", "所有损失"},
		Context:  []string{"违约金", "赔偿", "损失"},
		Statute:  "民法典第584条",
		Excerpt:  "当事人一方不履行合同义务或者履行合同义务不符合约定，造成对方损失的，损失赔偿额应当相当于因违约所造成的损失",
		Advice:   "建议设定违约金上限，避免承担不可预见的巨额赔偿",
	},
	
	// ===== 竞业限制相关 =====
	{
		ID:       "non-compete-5years",
		Name:     "竞业限制期限过长",
		Severity: 5,
		Likelihood: 5,
		Keywords: []string{"5年", "五年", "全球"},
		Context:  []string{"竞业", "竞争", "non-compete"},
		Statute:  "劳动合同法第23条",
		Excerpt:  "在解除或者终止劳动合同后，前款规定的人员到与本单位生产或者经营同类产品、从事同类业务的有竞争关系的其他用人单位，或者自己开业生产或者经营同类产品、从事同类业务的竞业限制期限，不得超过二年",
		Advice:   "竞业限制期限超过2年，超出部分无效！建议调整为2年以内",
	},
	{
		ID:       "non-compete-no-compensation",
		Name:     "竞业限制无补偿",
		Severity: 5,
		Likelihood: 4,
		Keywords: []string{"无需补偿", "无补偿", "不给补偿"},
		Context:  []string{"竞业", "限制"},
		Statute:  "劳动合同法第23条",
		Excerpt:  "对负有保密义务的劳动者，用人单位可以在劳动合同或者保密协议中与劳动者约定竞业限制条款，并约定在解除或者终止劳动合同后，在竞业限制期限内按月给予劳动者经济补偿",
		Advice:   "竞业限制必须支付经济补偿，否则条款无效",
	},
	{
		ID:       "non-compete-low-compensation",
		Name:     "竞业限制补偿过低",
		Severity: 4,
		Likelihood: 4,
		Keywords: []string{"10%", "5%", "20%"},
		Context:  []string{"补偿", "经济补偿"},
		Statute:  "最高人民法院关于审理劳动争议案件适用法律问题的解释（一）第36条",
		Excerpt:  "当事人在劳动合同或者保密协议中约定了竞业限制，但未约定解除或者终止劳动合同后给予劳动者经济补偿，劳动者履行了竞业限制义务，要求用人单位按照劳动者在劳动合同解除或者终止前十二个月平均工资的30%按月支付经济补偿的，人民法院应予支持",
		Advice:   "竞业限制补偿低于法定标准（前12个月平均工资的30%），可能无效",
	},
	
	// ===== 劳动合同相关 =====
	{
		ID:       "probation-6months",
		Name:     "试用期过长",
		Severity: 4,
		Likelihood: 5,
		Keywords: []string{"6个月", "六个月"},
		Context:  []string{"试用期"},
		Statute:  "劳动合同法第19条",
		Excerpt:  "劳动合同期限一年以上不满三年的，试用期不得超过二个月；三年以上固定期限和无固定期限的劳动合同，试用期不得超过六个月",
		Advice:   "试用期超过法定上限，超出部分无效，且可能需支付赔偿金",
	},
	{
		ID:       "probation-no-salary",
		Name:     "试用期无工资",
		Severity: 5,
		Likelihood: 3,
		Keywords: []string{"无工资", "不支付工资", "试用期间无"},
		Context:  []string{"试用期", "工资"},
		Statute:  "劳动合同法第20条",
		Excerpt:  "劳动者在试用期的工资不得低于本单位相同岗位最低档工资或者劳动合同约定工资的百分之八十，并不得低于用人单位所在地的最低工资标准",
		Advice:   "试用期必须支付工资，且不得低于约定工资的80%和当地最低工资",
	},
	{
		ID:       "overtime-voluntary",
		Name:     "自愿加班条款",
		Severity: 4,
		Likelihood: 4,
		Keywords: []string{"自愿加班", "自愿加班的除外"},
		Context:  []string{"加班", "工作时间"},
		Statute:  "劳动法第41条",
		Excerpt:  "用人单位由于生产经营需要，经与工会和劳动者协商后可以延长工作时间，一般每日不得超过一小时；因特殊原因需要延长工作时间的，在保障劳动者身体健康的条件下延长工作时间每日不得超过三小时，但是每月不得超过三十六小时",
		Advice:   "'自愿加班除外'条款可能规避法定加班费支付义务，存在被认定无效的风险",
	},
	{
		ID:       "no-social-insurance",
		Name:     "不缴纳社保",
		Severity: 5,
		Likelihood: 2,
		Keywords: []string{"不缴纳社保", "自愿放弃社保", "放弃社会保险"},
		Context:  []string{"社保", "社会保险"},
		Statute:  "社会保险法第58条",
		Excerpt:  "用人单位应当自用工之日起三十日内为其职工向社会保险经办机构申请办理社会保险登记",
		Advice:   "约定不缴社保无效！用人单位必须为员工缴纳社会保险",
	},
	
	// ===== 知识产权相关 =====
	{
		ID:       "ip-co-ownership",
		Name:     "知识产权归属不明确",
		Severity: 3,
		Likelihood: 4,
		Keywords: []string{"共同所有", "共有", "co-own"},
		Context:  []string{"知识产权", "专利", "著作权"},
		Statute:  "民法典第843条",
		Excerpt:  "委托开发完成的发明创造，除法律另有规定或者当事人另有约定外，申请专利的权利属于研究开发人",
		Advice:   "'共同所有'未明确使用范围和收益分配，建议详细约定各方权利和收益比例",
	},
	{
		ID:       "ip-transfer-restriction",
		Name:     "技术资料知识产权限制",
		Severity: 2,
		Likelihood: 3,
		Keywords: []string{"归乙方所有", "甲方仅有使用权"},
		Context:  []string{"知识产权", "技术资料"},
		Statute:  "民法典第843条",
		Excerpt:  "委托开发完成的发明创造，除法律另有规定或者当事人另有约定外，申请专利的权利属于研究开发人",
		Advice:   "买方仅有使用权可能限制后续改进和升级，建议争取共同所有或买断",
	},
	
	// ===== 管辖与争议解决 =====
	{
		ID:       "jurisdiction-unilateral",
		Name:     "管辖条款偏向一方",
		Severity: 3,
		Likelihood: 3,
		Keywords: []string{"甲方所在地", "乙方所在地"},
		Context:  []string{"管辖", "诉讼", "争议"},
		Statute:  "民事诉讼法第35条",
		Excerpt:  "合同或者其他财产权益纠纷的当事人可以书面协议选择被告住所地、合同履行地、合同签订地、原告住所地、标的物所在地等与争议有实际联系的地点的人民法院管辖",
		Advice:   "管辖条款约定单方所在地，可能增加另一方维权成本，建议约定中立地点或双方均可接受的地点",
	},
	{
		ID:       "arbitration-unspecified",
		Name:     "仲裁条款不明确",
		Severity: 3,
		Likelihood: 3,
		Keywords: []string{"仲裁委员会", "仲裁机构"},
		Context:  []string{"仲裁", "争议"},
		Statute:  "仲裁法第16条",
		Excerpt:  "仲裁协议应当具有下列内容：（一）请求仲裁的意思表示；（二）仲裁事项；（三）选定的仲裁委员会",
		Advice:   "仲裁条款未明确约定仲裁机构，可能导致仲裁条款无效",
	},
	
	// ===== 合同解除相关 =====
	{
		ID:       "termination-unilateral",
		Name:     "单方解除权过宽",
		Severity: 3,
		Likelihood: 3,
		Keywords: []string{"甲方有权单方面", "乙方有权单方面", "甲方有权单方"},
		Context:  []string{"解除", "终止"},
		Statute:  "民法典第563条",
		Excerpt:  "有下列情形之一的，当事人可以解除合同：（一）因不可抗力致使不能实现合同目的；（二）在履行期限届满前，当事人一方明确表示或者以自己的行为表明不履行主要债务；（三）当事人一方迟延履行主要债务，经催告后在合理期限内仍未履行；（四）当事人一方迟延履行债务或者有其他违约行为致使不能实现合同目的",
		Advice:   "过宽的单方解除权可能导致合同不稳定，建议明确约定具体解除条件",
	},
	
	// ===== 格式条款相关 =====
	{
		ID:       "unfair-format-clause",
		Name:     "格式条款显失公平",
		Severity: 4,
		Likelihood: 3,
		Keywords: []string{"最终解释权", "概不负责", "一切后果"},
		Context:  []string{"解释", "责任", "条款"},
		Statute:  "民法典第496条",
		Excerpt:  "格式条款是当事人为了重复使用而预先拟定，并在订立合同时未与对方协商的条款。采用格式条款订立合同的，提供格式条款的一方应当遵循公平原则确定当事人之间的权利和义务",
		Advice:   "'最终解释权'等格式条款可能因显失公平被认定无效",
	},
	
	// ===== 付款与交付 =====
	{
		ID:       "delivery-penalty-daily",
		Name:     "逾期交付违约金比例过高",
		Severity: 3,
		Likelihood: 3,
		Keywords: []string{"0.5%", "1%", "每日"},
		Context:  []string{"逾期", "违约金"},
		Statute:  "民法典第585条",
		Excerpt:  "约定的违约金过分高于造成的损失的，人民法院或者仲裁机构可以根据当事人的请求予以适当减少",
		Advice:   "日违约金比例过高（通常不超过万分之五），建议调整",
	},
	{
		ID:       "payment-no-security",
		Name:     "预付款无保障",
		Severity: 2,
		Likelihood: 3,
		Keywords: []string{"预付款", "定金"},
		Context:  []string{"支付", "付款"},
		Statute:  "民法典第586条",
		Excerpt:  "当事人可以约定一方向对方给付定金作为债权的担保。定金合同自实际交付定金时成立",
		Advice:   "大额预付款建议约定担保措施或分期支付，降低风险",
	},
	
	// ===== 其他高风险条款 =====
	{
		ID:       "unlimited-liability",
		Name:     "无限责任条款",
		Severity: 4,
		Likelihood: 3,
		Keywords: []string{"无限责任", "全部责任", "一切责任"},
		Context:  []string{"责任", "赔偿", "损失"},
		Statute:  "民法典第584条",
		Excerpt:  "当事人一方不履行合同义务或者履行合同义务不符合约定，造成对方损失的，损失赔偿额应当相当于因违约所造成的损失，包括合同履行后可以获得的利益；但是，不得超过违约一方订立合同时预见到或者应当预见到的因违约可能造成的损失",
		Advice:   "无限责任条款可能承担不可预见损失，建议设定责任上限",
	},
	{
		ID:       "warranty-unlimited",
		Name:     "无限期保证",
		Severity: 3,
		Likelihood: 2,
		Keywords: []string{"终身质保", "永久保修", "无限期保证"},
		Context:  []string{"保修", "质保", "保证"},
		Statute:  "民法典第620条",
		Excerpt:  "买受人收到标的物时应当在约定的检验期限内检验。没有约定检验期限的，应当及时检验",
		Advice:   "无限期保修成本难以控制，建议设定合理保修期",
	},
	{
		ID:       "automatic-renewal",
		Name:     "自动续约条款",
		Severity: 2,
		Likelihood: 4,
		Keywords: []string{"自动续约", "自动续期", "期满自动"},
		Context:  []string{"续约", "续期", "期限"},
		Statute:  "民法典第543条",
		Excerpt:  "当事人协商一致，可以变更合同",
		Advice:   "自动续约条款可能限制合同自由，建议约定提前通知解除机制",
	},
	{
		ID:       "non-refundable",
		Name:     "不退款条款",
		Severity: 3,
		Likelihood: 3,
		Keywords: []string{"不予退还", "概不退款", "不退还"},
		Context:  []string{"退款", "退还", "费用"},
		Statute:  "消费者权益保护法第53条",
		Excerpt:  "经营者以预收款方式提供商品或者服务的，应当按照约定提供。未按照约定提供的，应当按照消费者的要求履行约定或者退回预付款",
		Advice:   "'概不退款'条款可能因显失公平被认定无效",
	},
}

// Rule 内置规则
type Rule struct {
	ID         string
	Name       string
	Severity   int
	Likelihood int
	Keywords   []string
	Context    []string
	Statute    string
	Excerpt    string
	Advice     string
}

func main() {
	var (
		file   = flag.String("f", "", "合同文件路径")
		format = flag.String("o", "text", "输出格式: text / json / markdown")
		lang   = flag.String("l", "zh", "语言: zh / en")
		verbose = flag.Bool("v", false, "详细输出")
		ruleID = flag.String("rule", "", "只运行指定规则")
	)
	flag.Parse()

	if *file == "" {
		printUsage()
		os.Exit(1)
	}

	content, err := os.ReadFile(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取文件失败: %v\n", err)
		os.Exit(1)
	}

	rules := ruleEngine
	if *ruleID != "" {
		rules = filterRules(rules, *ruleID)
	}

	result := analyzeContract(string(content), *file, rules)
	
	switch *format {
	case "json":
		outputJSON(result)
	case "markdown", "md":
		outputMarkdown(result, *lang)
	default:
		outputText(result, *lang, *verbose)
	}
}

// printUsage 打印帮助
func printUsage() {
	fmt.Println("⚖️ 律衡 LexScale — 合同审查 CLI")
	fmt.Println("")
	fmt.Println("用法:")
	fmt.Println("  lexscale -f <合同文件> [选项]")
	fmt.Println("")
	fmt.Println("选项:")
	fmt.Println("  -o    输出格式: text / json / markdown (默认 text)")
	fmt.Println("  -l    语言: zh / en (默认 zh)")
	fmt.Println("  -v    详细输出")
	fmt.Println("  -rule 只运行指定规则")
	fmt.Println("")
	fmt.Println("示例:")
	fmt.Println("  lexscale -f examples/nda.txt")
	fmt.Println("  lexscale -f examples/labor_contract.txt -o json")
	fmt.Println("  lexscale -f examples/nda.txt -o markdown")
	fmt.Println("  lexscale -f examples/nda.txt -rule excessive-penalty")
	fmt.Println("")
	fmt.Println("可用规则:")
	for _, r := range ruleEngine {
		fmt.Printf("  %s: %s\n", r.ID, r.Name)
	}
}

// filterRules 过滤规则
func filterRules(rules []Rule, id string) []Rule {
	for _, r := range rules {
		if r.ID == id {
			return []Rule{r}
		}
	}
	return []Rule{}
}

// analyzeContract 分析合同
func analyzeContract(content, filename string, rules []Rule) *AnalysisResult {
	result := &AnalysisResult{
		ContractName: filename,
		Findings:     []Finding{},
	}

	text := strings.ToLower(content)

	for _, rule := range rules {
		matched := false
		
		// 检查关键词
		for _, kw := range rule.Keywords {
			if strings.Contains(text, strings.ToLower(kw)) {
				// 检查上下文
				for _, ctx := range rule.Context {
					if strings.Contains(text, strings.ToLower(ctx)) {
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
		}

		if matched {
			level := calculateLevel(rule.Severity, rule.Likelihood)
			finding := Finding{
				Level:      level,
				Title:      rule.Name,
				Severity:   rule.Severity,
				Likelihood: rule.Likelihood,
				Tier:       "INFERENCE",
				Citations: []Citation{
					{Source: rule.Statute, Excerpt: rule.Excerpt},
				},
				Reasoning: rule.Advice,
				Clause:    extractClause(text, rule.Keywords),
			}
			result.Findings = append(result.Findings, finding)
			
			switch level {
			case "RED":
				result.RedCount++
			case "ORANGE":
				result.OrangeCount++
			case "YELLOW":
				result.YellowCount++
			case "GREEN":
				result.GreenCount++
			}
		}
	}

	result.TotalFindings = len(result.Findings)
	return result
}

// extractClause 提取相关条款
func extractClause(text string, keywords []string) string {
	for _, kw := range keywords {
		idx := strings.Index(text, strings.ToLower(kw))
		if idx >= 0 {
			start := idx - 50
			if start < 0 {
				start = 0
			}
			end := idx + len(kw) + 100
			if end > len(text) {
				end = len(text)
			}
			return "..." + text[start:end] + "..."
		}
	}
	return ""
}

// calculateLevel 计算风险等级
func calculateLevel(severity, likelihood int) string {
	score := severity * likelihood
	switch {
	case score >= 16:
		return "RED"
	case score >= 10:
		return "ORANGE"
	case score >= 5:
		return "YELLOW"
	default:
		return "GREEN"
	}
}

// outputText 文本输出
func outputText(result *AnalysisResult, lang string, verbose bool) {
	if lang == "en" {
		outputEnglish(result, verbose)
		return
	}

	fmt.Printf("⚖️ 律衡 LexScale — 合同审查报告\n")
	fmt.Printf("═══════════════════════════════════\n")
	fmt.Printf("📄 合同: %s\n", result.ContractName)
	fmt.Printf("🔍 发现: %d 个风险\n\n", result.TotalFindings)

	emoji := map[string]string{"RED": "🔴", "ORANGE": "🟠", "YELLOW": "🟡", "GREEN": "🟢"}
	
	for i, f := range result.Findings {
		fmt.Printf("%d. %s [%s] %s\n", i+1, emoji[f.Level], f.Level, f.Title)
		fmt.Printf("   严重度: %d | 可能性: %d\n", f.Severity, f.Likelihood)
		for _, c := range f.Citations {
			fmt.Printf("   📎 %s\n", c.Source)
			if verbose {
				fmt.Printf("      \"%s\"\n", truncate(c.Excerpt, 80))
			}
		}
		if verbose && f.Clause != "" {
			fmt.Printf("   📍 相关条款: %s\n", truncate(f.Clause, 100))
		}
		fmt.Printf("   💡 %s\n\n", f.Reasoning)
	}

	fmt.Printf("═══════════════════════════════════\n")
	fmt.Printf("统计: 🔴%d 🟠%d 🟡%d 🟢%d\n", result.RedCount, result.OrangeCount, result.YellowCount, result.GreenCount)
	
	// 输出建议
	if result.RedCount > 0 {
		fmt.Printf("\n⚠️  建议: 发现 %d 个高风险问题，建议咨询专业律师\n", result.RedCount)
	}
}

// outputEnglish 英文输出
func outputEnglish(result *AnalysisResult, verbose bool) {
	fmt.Printf("⚖️ LexScale — Contract Review Report\n")
	fmt.Printf("═══════════════════════════════════\n")
	fmt.Printf("📄 Contract: %s\n", result.ContractName)
	fmt.Printf("🔍 Found: %d risks\n\n", result.TotalFindings)

	emoji := map[string]string{"RED": "🔴", "ORANGE": "🟠", "YELLOW": "🟡", "GREEN": "🟢"}
	
	for i, f := range result.Findings {
		fmt.Printf("%d. %s [%s] %s\n", i+1, emoji[f.Level], f.Level, f.Title)
		fmt.Printf("   Severity: %d | Likelihood: %d\n", f.Severity, f.Likelihood)
		for _, c := range f.Citations {
			fmt.Printf("   📎 %s\n", c.Source)
		}
		fmt.Printf("   💡 %s\n\n", f.Reasoning)
	}

	fmt.Printf("═══════════════════════════════════\n")
	fmt.Printf("Summary: 🔴%d 🟠%d 🟡%d 🟢%d\n", result.RedCount, result.OrangeCount, result.YellowCount, result.GreenCount)
}

// outputMarkdown Markdown 输出
func outputMarkdown(result *AnalysisResult, lang string) {
	if lang == "en" {
		outputMarkdownEnglish(result)
		return
	}

	fmt.Printf("# 合同审查报告\n\n")
	fmt.Printf("**合同**: %s  \n", result.ContractName)
	fmt.Printf("**发现**: %d 个风险  \n\n", result.TotalFindings)
	
	fmt.Printf("## 风险统计\n\n")
	fmt.Printf("| 等级 | 数量 |\n")
	fmt.Printf("|------|------|\n")
	fmt.Printf("| 🔴 RED | %d |\n", result.RedCount)
	fmt.Printf("| 🟠 ORANGE | %d |\n", result.OrangeCount)
	fmt.Printf("| 🟡 YELLOW | %d |\n", result.YellowCount)
	fmt.Printf("| 🟢 GREEN | %d |\n\n", result.GreenCount)
	
	fmt.Printf("## 详细发现\n\n")
	
	for i, f := range result.Findings {
		fmt.Printf("### %d. %s\n\n", i+1, f.Title)
		fmt.Printf("- **等级**: %s\n", f.Level)
		fmt.Printf("- **严重度**: %d\n", f.Severity)
		fmt.Printf("- **可能性**: %d\n\n", f.Likelihood)
		
		fmt.Printf("#### 法律依据\n\n")
		for _, c := range f.Citations {
			fmt.Printf("> %s\n", c.Excerpt)
			fmt.Printf("> —— %s\n\n", c.Source)
		}
		
		fmt.Printf("#### 建议\n\n")
		fmt.Printf("%s\n\n", f.Reasoning)
		fmt.Printf("---\n\n")
	}
}

// outputMarkdownEnglish 英文 Markdown
func outputMarkdownEnglish(result *AnalysisResult) {
	fmt.Printf("# Contract Review Report\n\n")
	fmt.Printf("**Contract**: %s  \n", result.ContractName)
	fmt.Printf("**Found**: %d risks  \n\n", result.TotalFindings)
	
	fmt.Printf("## Risk Summary\n\n")
	fmt.Printf("| Level | Count |\n")
	fmt.Printf("|-------|-------|\n")
	fmt.Printf("| 🔴 RED | %d |\n", result.RedCount)
	fmt.Printf("| 🟠 ORANGE | %d |\n", result.OrangeCount)
	fmt.Printf("| 🟡 YELLOW | %d |\n", result.YellowCount)
	fmt.Printf("| 🟢 GREEN | %d |\n\n", result.GreenCount)
	
	fmt.Printf("## Findings\n\n")
	
	for i, f := range result.Findings {
		fmt.Printf("### %d. %s\n\n", i+1, f.Title)
		fmt.Printf("- **Level**: %s\n", f.Level)
		fmt.Printf("- **Severity**: %d\n", f.Severity)
		fmt.Printf("- **Likelihood**: %d\n\n", f.Likelihood)
		
		fmt.Printf("#### Legal Basis\n\n")
		for _, c := range f.Citations {
			fmt.Printf("> %s\n", c.Excerpt)
			fmt.Printf("> —— %s\n\n", c.Source)
		}
		
		fmt.Printf("#### Recommendation\n\n")
		fmt.Printf("%s\n\n", f.Reasoning)
		fmt.Printf("---\n\n")
	}
}

// outputJSON JSON 输出
func outputJSON(result *AnalysisResult) {
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}

// truncate 截断字符串
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// 引用 graph 包避免 unused
var _ = graph.TierFact
