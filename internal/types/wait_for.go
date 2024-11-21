package types

type WaitFor struct {
	Field []WaitForField
	Condition []WaitForStatusCondition
}
type WaitForField struct {
	Key       string
	Value     string
	ValueType string `mapstructure:"value_type"`
}
type WaitForStatusCondition struct {
	Type   string
	Status string
}
