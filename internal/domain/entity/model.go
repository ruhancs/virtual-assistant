package entity

// modelos do chat pt
type Model struct {
	Name      string
	MaxTokens int
}

func NewModel(name string, maxTokens int) *Model {
	return &Model{
		Name:      name,
		MaxTokens: maxTokens,
	}
}

func (m *Model) GetMaxToken() int {
	return m.MaxTokens
}

func (m *Model) GetModelName() string {
	return m.Name
}
