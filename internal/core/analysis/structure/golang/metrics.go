package gostructure

// The type of metric
type MetricType string

const (
	// Element counts
	MetricTotalElements MetricType = "total_elements"
	MetricPackages      MetricType = "packages"
	MetricTypes         MetricType = "types"
	MetricFunctions     MetricType = "functions"
	MetricMethods       MetricType = "methods"
	MetricInterfaces    MetricType = "interfaces"
	MetricVariables     MetricType = "variables"

	// Relationship counts
	MetricContains        MetricType = "contains"
	MetricImplements      MetricType = "implements"
	MetricEmbeds          MetricType = "embeds"
	MetricInterfaceEmbeds MetricType = "interface_embeds"
	MetricMethodReceiver  MetricType = "method_receiver"
	MetricCalls           MetricType = "calls"
	MetricReferences      MetricType = "references"

	// Complexity metrics
	MetricMaxDepth    MetricType = "max_depth"
	MetricAvgDepth    MetricType = "avg_depth"
	MetricMaxChildren MetricType = "max_children"
	MetricAvgChildren MetricType = "avg_children"
)

// Collects metrics about the code structure
type MetricsCollector struct {
	metrics map[MetricType]int
}

// Creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[MetricType]int),
	}
}

// Returns the value of a metric
func (c *MetricsCollector) Metric(metric MetricType) int {
	return c.metrics[metric]
}

// CollectMetrics collects metrics from the structure
func (c *MetricsCollector) CollectMetrics(structure *Structure) {
	// Reset metrics
	c.metrics = make(map[MetricType]int)

	// Count elements by type
	for _, elem := range structure.Elements {
		c.metrics[MetricTotalElements]++
		switch elem.Type {
		case ElementPackage:
			c.metrics[MetricPackages]++
		case ElementTypeDecl:
			c.metrics[MetricTypes]++
		case ElementFunction:
			c.metrics[MetricFunctions]++
		case ElementMethod:
			c.metrics[MetricMethods]++
		case ElementInterface:
			c.metrics[MetricInterfaces]++
		case ElementVariable:
			c.metrics[MetricVariables]++
		}
	}

	// Count relationships by type
	for _, rel := range structure.Relationships {
		switch rel.Type {
		case RelationContains:
			c.metrics[MetricContains]++
		case RelationImplements:
			c.metrics[MetricImplements]++
		case RelationEmbeds:
			c.metrics[MetricEmbeds]++
		case RelationInterfaceEmbeds:
			c.metrics[MetricInterfaceEmbeds]++
		case RelationMethodReceiver:
			c.metrics[MetricMethodReceiver]++
		case RelationCalls:
			c.metrics[MetricCalls]++
		case RelationReferences:
			c.metrics[MetricReferences]++
		}
	}

	c.calculateComplexityMetrics(structure)
}

// Calculates complexity-related metrics
func (c *MetricsCollector) calculateComplexityMetrics(structure *Structure) {
	depths := make(map[*Element]int)
	children := make(map[*Element]int)

	// Calculate depths and children counts
	for _, rel := range structure.Relationships {
		if rel.Type == RelationContains {
			depths[rel.Target] = depths[rel.Source] + 1
			children[rel.Source]++
		}
	}

	// Find max depth
	maxDepth := 0
	totalDepth := 0
	for _, depth := range depths {
		if depth > maxDepth {
			maxDepth = depth
		}
		totalDepth += depth
	}

	// Find max children
	maxChildren := 0
	totalChildren := 0
	for _, count := range children {
		if count > maxChildren {
			maxChildren = count
		}
		totalChildren += count
	}

	// Store metrics
	c.metrics[MetricMaxDepth] = maxDepth
	if len(depths) > 0 {
		c.metrics[MetricAvgDepth] = totalDepth / len(depths)
	}
	c.metrics[MetricMaxChildren] = maxChildren
	if len(children) > 0 {
		c.metrics[MetricAvgChildren] = totalChildren / len(children)
	}
}
