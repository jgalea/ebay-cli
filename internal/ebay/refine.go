package ebay

type CategoryDistribution struct {
	CategoryID   string `json:"categoryId"`
	CategoryName string `json:"categoryName"`
	MatchCount   int    `json:"matchCount"`
}

type ConditionDistribution struct {
	Condition   string `json:"condition"`
	ConditionID string `json:"conditionId"`
	MatchCount  int    `json:"matchCount"`
}

type AspectValueDistribution struct {
	Value      string `json:"localizedAspectValue"`
	MatchCount int    `json:"matchCount"`
}

type AspectDistribution struct {
	Name   string                    `json:"localizedAspectName"`
	Values []AspectValueDistribution `json:"aspectValueDistributions"`
}

type Refinement struct {
	DominantCategoryID     string                  `json:"dominantCategoryId"`
	CategoryDistributions  []CategoryDistribution  `json:"categoryDistributions"`
	ConditionDistributions []ConditionDistribution `json:"conditionDistributions"`
	AspectDistributions    []AspectDistribution    `json:"aspectDistributions"`
}
