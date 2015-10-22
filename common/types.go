package common

// Tool interface implemented by the various anti-plagiarism tool packages
type Tool interface {
	CreateCommands(org string, labs []LabInfo) ([]string, bool)
	SaveResults(fileNameAndPath string, baseDir string, lab LabInfo) bool
}

// The language the lab was written in
const (
	Java = iota
	Golang
	Cpp
)

// LabInfo is information about the lab: name and programming language
type LabInfo struct {
	Name     string
	Language int
}
