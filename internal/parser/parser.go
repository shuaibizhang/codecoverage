package parser

type LanguageType string

const (
	GoLanguage LanguageType = "go"
)

const (
	NotInstrLine int32 = -1
)

// Parser 定义解析器，屏蔽各语言底层覆盖率数据差异，统一成归一化覆盖率数据
type Parser interface {
	Parse(string) (*CovNormalInfo, error)
	ParseMultiFiles(map[string]string) (*CovNormalInfo, error)
	ScanCoverageFiles(rootDir string) (map[string]string, error)
}
