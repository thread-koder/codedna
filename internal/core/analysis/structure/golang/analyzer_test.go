package gostructure_test

import (
	"path/filepath"
	"testing"

	gostructure "codedna/internal/core/analysis/structure/golang"
	goparser "codedna/internal/core/parser/golang"
)

func TestAnalyzer_SampleFile(t *testing.T) {
	// Create parser and analyzer
	parser := goparser.New()
	analyzer := gostructure.NewAnalyzer()

	// Parse test file
	testFile := filepath.Join("testdata", "sample.go")
	astNode, err := parser.ParseFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// Create Go node and analyze
	node := gostructure.NewNode(astNode)
	analysis, err := analyzer.Analyze(node)
	if err != nil {
		t.Fatalf("Failed to analyze file: %v", err)
	}

	// Type assert to Go analysis
	goAnalysis, ok := analysis.(*gostructure.Analysis)
	if !ok {
		t.Fatalf("Expected Go analysis, got %T", analysis)
	}

	// Verify elements
	t.Run("Elements", func(t *testing.T) {
		expectedElements := map[string]struct {
			typ        gostructure.ElementType
			isExported bool
		}{
			"Writer": {
				typ:        gostructure.ElementInterface,
				isExported: true,
			},
			"Document": {
				typ:        gostructure.ElementTypeDecl,
				isExported: true,
			},
			"JSONDocument": {
				typ:        gostructure.ElementTypeDecl,
				isExported: true,
			},
			"Validator": {
				typ:        gostructure.ElementInterface,
				isExported: true,
			},
			"ValidatingDocument": {
				typ:        gostructure.ElementTypeDecl,
				isExported: true,
			},
			"NewDocument": {
				typ:        gostructure.ElementFunction,
				isExported: true,
			},
			"Write": {
				typ:        gostructure.ElementMethod,
				isExported: true,
			},
			"GetContent": {
				typ:        gostructure.ElementMethod,
				isExported: true,
			},
		}

		for _, elem := range goAnalysis.Structure.Elements {
			if expected, ok := expectedElements[elem.Name]; ok {
				if elem.Type != expected.typ {
					t.Errorf("Element %s: expected type %s, got %s", elem.Name, expected.typ, elem.Type)
				}
				if isExported := elem.Attributes["is_exported"].(bool); isExported != expected.isExported {
					t.Errorf("Element %s: expected exported=%v, got %v", elem.Name, expected.isExported, isExported)
				}
			}
		}
	})

	// Verify relationships
	t.Run("Relationships", func(t *testing.T) {
		expectedRelationships := []struct {
			sourceType gostructure.ElementType
			sourceName string
			relType    gostructure.RelationType
			targetType gostructure.ElementType
			targetName string
		}{
			{gostructure.ElementTypeDecl, "Document", gostructure.RelationImplements, gostructure.ElementInterface, "Writer"},
			{gostructure.ElementTypeDecl, "JSONDocument", gostructure.RelationImplements, gostructure.ElementInterface, "Writer"},
			{gostructure.ElementTypeDecl, "ValidatingDocument", gostructure.RelationImplements, gostructure.ElementInterface, "Writer"},
			{gostructure.ElementTypeDecl, "JSONDocument", gostructure.RelationEmbeds, gostructure.ElementTypeDecl, "Document"},
			{gostructure.ElementTypeDecl, "ValidatingDocument", gostructure.RelationEmbeds, gostructure.ElementTypeDecl, "Document"},
			{gostructure.ElementMethod, "Write", gostructure.RelationMethodReceiver, gostructure.ElementTypeDecl, "Document"},
			{gostructure.ElementMethod, "GetContent", gostructure.RelationMethodReceiver, gostructure.ElementTypeDecl, "Document"},
		}

		for _, expected := range expectedRelationships {
			found := false
			for _, rel := range goAnalysis.Structure.Relationships {
				if rel.Source.Type == expected.sourceType &&
					rel.Source.Name == expected.sourceName &&
					rel.Type == expected.relType &&
					rel.Target.Type == expected.targetType &&
					rel.Target.Name == expected.targetName {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Missing relationship: %s.%s -%s-> %s.%s",
					expected.sourceType, expected.sourceName,
					expected.relType,
					expected.targetType, expected.targetName)
			}
		}
	})

	// Verify metrics
	t.Run("Metrics", func(t *testing.T) {
		collector := gostructure.NewMetricsCollector()
		collector.CollectMetrics(goAnalysis.Structure)

		expectedMetrics := map[gostructure.MetricType]int{
			gostructure.MetricTotalElements:   11, // All top-level declarations
			gostructure.MetricPackages:        1,  // testdata package
			gostructure.MetricTypes:           3,  // Document, JSONDocument, ValidatingDocument
			gostructure.MetricFunctions:       1,  // NewDocument
			gostructure.MetricMethods:         2,  // Write, GetContent
			gostructure.MetricInterfaces:      2,  // Writer, Validator
			gostructure.MetricVariables:       2,  // TypeText, TypeJSON
			gostructure.MetricContains:        10, // Package contains all declarations
			gostructure.MetricImplements:      3,  // Document->Writer, JSONDocument->Writer, ValidatingDocument->Writer
			gostructure.MetricEmbeds:          2,  // JSONDocument->Document, ValidatingDocument->Document
			gostructure.MetricInterfaceEmbeds: 0,  // No interface embeddings
			gostructure.MetricMethodReceiver:  2,  // Write->Document, GetContent->Document
			gostructure.MetricCalls:           0,  // No function calls analyzed
			gostructure.MetricReferences:      6,  // Various type references
			gostructure.MetricMaxDepth:        1,  // All declarations at package level
			gostructure.MetricAvgDepth:        1,  // All declarations at same depth
			gostructure.MetricMaxChildren:     10, // Package has 10 top-level declarations
			gostructure.MetricAvgChildren:     10, // Only one parent with all children
		}

		for metricType, expectedValue := range expectedMetrics {
			if actual := collector.Metric(metricType); actual != expectedValue {
				t.Errorf("Metric %v: expected %d, got %d", metricType, expectedValue, actual)
			}
		}
	})
}

func TestAnalyzer_WholeDirectory(t *testing.T) {
	// Create parser and analyzer
	parser := goparser.New()
	analyzer := gostructure.NewAnalyzer()

	// Parse test directory containing both files
	testDir := filepath.Join("testdata")
	astNodes, err := parser.ParseDir(testDir)
	if err != nil {
		t.Fatalf("Failed to parse directory: %v", err)
	}

	// Analyze each file
	var goAnalysis *gostructure.Analysis
	for _, astNode := range astNodes {
		// Create Go node and analyze
		node := gostructure.NewNode(astNode)
		analysis, err := analyzer.Analyze(node)
		if err != nil {
			t.Fatalf("Failed to analyze file: %v", err)
		}

		// Type assert and merge analyses
		if ga, ok := analysis.(*gostructure.Analysis); ok {
			if goAnalysis == nil {
				goAnalysis = ga
			} else {
				goAnalysis.Structure.Elements = append(goAnalysis.Structure.Elements, ga.Structure.Elements...)
				goAnalysis.Structure.Relationships = append(goAnalysis.Structure.Relationships, ga.Structure.Relationships...)
			}
		} else {
			t.Fatalf("Expected Go analysis, got %T", analysis)
		}
	}

	// Verify combined analysis
	t.Run("Combined Elements", func(t *testing.T) {
		expectedInterfaces := []string{
			"Reader", "ReadWriter", "Storage", "CacheStats",
			"DocumentProcessor", "Writer", "Validator",
		}
		expectedTypes := []string{
			"BaseStorage", "MemoryDocument", "SimpleProcessor",
			"Document", "JSONDocument", "ValidatingDocument",
		}

		// Verify interfaces
		foundInterfaces := make(map[string]bool)
		for _, elem := range goAnalysis.Structure.Elements {
			if elem.Type == gostructure.ElementInterface {
				foundInterfaces[elem.Name] = true
			}
		}
		for _, expected := range expectedInterfaces {
			if !foundInterfaces[expected] {
				t.Errorf("Missing interface: %s", expected)
			}
		}

		// Verify types
		foundTypes := make(map[string]bool)
		for _, elem := range goAnalysis.Structure.Elements {
			if elem.Type == gostructure.ElementTypeDecl {
				foundTypes[elem.Name] = true
			}
		}
		for _, expected := range expectedTypes {
			if !foundTypes[expected] {
				t.Errorf("Missing type: %s", expected)
			}
		}
	})

	// Verify combined relationships
	t.Run("Combined Relationships", func(t *testing.T) {
		expectedRelationships := []struct {
			sourceType gostructure.ElementType
			sourceName string
			relType    gostructure.RelationType
			targetType gostructure.ElementType
			targetName string
		}{
			{gostructure.ElementInterface, "ReadWriter", gostructure.RelationInterfaceEmbeds, gostructure.ElementInterface, "Reader"},
			{gostructure.ElementTypeDecl, "MemoryDocument", gostructure.RelationImplements, gostructure.ElementInterface, "Reader"},
			{gostructure.ElementTypeDecl, "BaseStorage", gostructure.RelationImplements, gostructure.ElementInterface, "Storage"},
			{gostructure.ElementTypeDecl, "MemoryDocument", gostructure.RelationImplements, gostructure.ElementInterface, "CacheStats"},
			{gostructure.ElementTypeDecl, "SimpleProcessor", gostructure.RelationImplements, gostructure.ElementInterface, "DocumentProcessor"},
			{gostructure.ElementTypeDecl, "Document", gostructure.RelationImplements, gostructure.ElementInterface, "Writer"},
			{gostructure.ElementTypeDecl, "MemoryDocument", gostructure.RelationEmbeds, gostructure.ElementTypeDecl, "BaseStorage"},
			{gostructure.ElementTypeDecl, "JSONDocument", gostructure.RelationEmbeds, gostructure.ElementTypeDecl, "Document"},
			{gostructure.ElementTypeDecl, "ValidatingDocument", gostructure.RelationEmbeds, gostructure.ElementTypeDecl, "Document"},
		}

		for _, expected := range expectedRelationships {
			found := false
			for _, rel := range goAnalysis.Structure.Relationships {
				if rel.Source.Type == expected.sourceType &&
					rel.Source.Name == expected.sourceName &&
					rel.Type == expected.relType &&
					rel.Target.Type == expected.targetType &&
					rel.Target.Name == expected.targetName {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Missing relationship: %s.%s -%s-> %s.%s",
					expected.sourceType, expected.sourceName,
					expected.relType,
					expected.targetType, expected.targetName)
			}
		}
	})

	// Verify combined metrics
	t.Run("Combined Metrics", func(t *testing.T) {
		collector := gostructure.NewMetricsCollector()
		collector.CollectMetrics(goAnalysis.Structure)

		expectedMetrics := map[gostructure.MetricType]int{
			gostructure.MetricTotalElements:   28, // All elements from both files
			gostructure.MetricPackages:        2,  // testdata package appears twice
			gostructure.MetricTypes:           6,  // All struct types
			gostructure.MetricFunctions:       2,  // NewMemoryDocument, NewDocument
			gostructure.MetricMethods:         9,  // All methods
			gostructure.MetricInterfaces:      7,  // All interfaces
			gostructure.MetricVariables:       2,  // TypeText, TypeJSON
			gostructure.MetricContains:        26, // Package contains all declarations
			gostructure.MetricImplements:      8,  // All interface implementations
			gostructure.MetricEmbeds:          3,  // All struct embeddings
			gostructure.MetricInterfaceEmbeds: 1,  // ReadWriter embeds Reader
			gostructure.MetricMethodReceiver:  9,  // All method receivers
			gostructure.MetricCalls:           0,  // No function calls analyzed
			gostructure.MetricReferences:      11, // All type references
			gostructure.MetricMaxDepth:        1,  // All declarations at package level
			gostructure.MetricAvgDepth:        1,  // All declarations at same depth
			gostructure.MetricMaxChildren:     16, // Max declarations in a package
			gostructure.MetricAvgChildren:     13, // Average declarations per package
		}

		for metricType, expectedValue := range expectedMetrics {
			if actual := collector.Metric(metricType); actual != expectedValue {
				t.Errorf("Metric %v: expected %d, got %d", metricType, expectedValue, actual)
			}
		}
	})
}

func BenchmarkAnalyzer_SampleFile(b *testing.B) {
	parser := goparser.New()
	analyzer := gostructure.NewAnalyzer()
	testFile := filepath.Join("testdata", "sample.go")

	// Parse file once outside the benchmark loop
	astNode, err := parser.ParseFile(testFile)
	if err != nil {
		b.Fatalf("Failed to parse file: %v", err)
	}

	// Create Go node once
	node := gostructure.NewNode(astNode)

	// Reset timer to exclude setup time
	for b.Loop() {
		analysis, err := analyzer.Analyze(node)
		if err != nil {
			b.Fatalf("Failed to analyze file: %v", err)
		}
		if _, ok := analysis.(*gostructure.Analysis); !ok {
			b.Fatal("Expected Go analysis")
		}
	}
}

func BenchmarkAnalyzer_WholeDirectory(b *testing.B) {
	parser := goparser.New()
	analyzer := gostructure.NewAnalyzer()
	testDir := filepath.Join("testdata")

	// Parse directory once outside the benchmark loop
	astNodes, err := parser.ParseDir(testDir)
	if err != nil {
		b.Fatalf("Failed to parse directory: %v", err)
	}

	// Reset timer to exclude setup time
	for b.Loop() {
		var goAnalysis *gostructure.Analysis

		// Analyze each file
		for _, astNode := range astNodes {
			node := gostructure.NewNode(astNode)
			analysis, err := analyzer.Analyze(node)
			if err != nil {
				b.Fatalf("Failed to analyze file: %v", err)
			}

			// Merge analyses
			if ga, ok := analysis.(*gostructure.Analysis); ok {
				if goAnalysis == nil {
					goAnalysis = ga
				} else {
					goAnalysis.Structure.Elements = append(goAnalysis.Structure.Elements, ga.Structure.Elements...)
					goAnalysis.Structure.Relationships = append(goAnalysis.Structure.Relationships, ga.Structure.Relationships...)
				}
			} else {
				b.Fatal("Expected Go analysis")
			}
		}
	}
}
