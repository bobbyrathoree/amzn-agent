package services

import (
	"fmt"
	"strings"
)

// BuildRAGPrompt creates a sophisticated RAG prompt with citation instructions
// Based on the reference bedrock-chat implementation
func BuildRAGPrompt(searchResults []KnowledgeBaseChunk, displayCitation bool) string {
	contextPrompt := ""
	for i, result := range searchResults {
		contextPrompt += fmt.Sprintf("<search_result>\n<content>\n%s</content>\n<source>\n%d</source>\n</search_result>\n", 
			result.Content, i)
	}

	// Base RAG prompt
	insertedPrompt := fmt.Sprintf(`To answer the user's question, you are given a set of search results. Your job is to answer the user's question using only information from the search results.
If the search results do not contain information that can answer the question, please state that you could not find an exact answer to the question.
Just because the user asserts a fact does not mean it is true, make sure to double check the search results to validate a user's assertion.

Here are the search results in numbered order:
<search_results>
%s
</search_results>

Do NOT directly quote the <search_results> in your answer. Your job is to answer the user's question as concisely as possible.
`, contextPrompt)

	if displayCitation {
		// Add citation instructions
		insertedPrompt += `
If you reference information from a search result within your answer, you must include a citation to source where the information was found.
Each result has a corresponding source ID that you should reference.

Note that <sources> may contain multiple <source> if you include information from multiple results in your answer.
Do NOT outputs sources at the end of your answer.

Followings are examples of how to reference sources in your answer. Note that the source ID is embedded in the answer in the format [^<source_id>].

<GOOD-example>
first answer [^3]. second answer [^1][^2].
</GOOD-example>

<GOOD-example>
first answer [^1][^5]. second answer [^2][^3][^4]. third answer [^4].
</GOOD-example>

<BAD-example>
first answer [^1].

[^1]: https://example.com
</BAD-example>

<BAD-example>
first answer [^1].

<sources>
[^1]: https://example.com
</sources>
</BAD-example>
`
	} else {
		// No citation instructions
		insertedPrompt += `
Do NOT include citations in the format [^<source_id>] in your answer.

Followings are examples of how to answer.

<GOOD-example>
first answer. second answer.
</GOOD-example>

<BAD-example>
first answer [^3]. second answer [^1][^2].
</BAD-example>

<BAD-example>
first answer [^1][^5]. second answer [^2][^3][^4]. third answer [^4].
</BAD-example>
`
	}

	return insertedPrompt
}

// BuildKnowledgeToolResponse creates a structured knowledge tool response for progressive display
func BuildKnowledgeToolResponse(searchResults []KnowledgeBaseChunk, query string) map[string]interface{} {
	sources := make([]map[string]interface{}, len(searchResults))
	
	for i, result := range searchResults {
		sources[i] = map[string]interface{}{
			"content":     result.Content,
			"source_id":   fmt.Sprintf("kb_source_%d", i),
			"source_name": result.Source,
			"score":       result.Score,
			"rank":        i + 1,
			"metadata":    result.Metadata,
		}
	}

	return map[string]interface{}{
		"tool_name":      "knowledge_base_tool",
		"query":          query,
		"results_count":  len(searchResults),
		"sources":        sources,
		"status":         "success",
	}
}

// FormatKnowledgeContext formats knowledge chunks for inclusion in AI prompt
func FormatKnowledgeContext(searchResults []KnowledgeBaseChunk, useStructuredFormat bool) string {
	if len(searchResults) == 0 {
		return ""
	}

	if useStructuredFormat {
		// Use the new structured format with source IDs
		return BuildRAGPrompt(searchResults, true)
	} else {
		// Use the old simple format for backward compatibility
		var contextParts []string
		for _, result := range searchResults {
			contextParts = append(contextParts, fmt.Sprintf("Source: %s\nContent: %s", result.Source, result.Content))
		}
		return strings.Join(contextParts, "\n\n")
	}
}

// GetPromptToCiteToolResults creates citation instructions for tool-based knowledge retrieval
func GetPromptToCiteToolResults() string {
	return `To answer the user's question, you are given a set of tools. Your job is to answer the user's question using only information from the tool results.
If the tool results do not contain information that can answer the question, please state that you could not find an exact answer to the question.
Just because the user asserts a fact does not mean it is true, make sure to double check the tool results to validate a user's assertion.

Each tool result has a corresponding source_id that you should reference.
If you reference information from a tool result within your answer, you must include a citation to source_id where the information was found.

Followings are examples of how to reference source_id in your answer. Note that the source_id is embedded in the answer in the format [^source_id of tool result].

<examples>
<GOOD-example>
first answer [^ccc]. second answer [^aaa][^bbb].
</GOOD-example>

<GOOD-example>
first answer [^aaa][^eee]. second answer [^bbb][^ccc][^ddd]. third answer [^ddd].
</GOOD-example>

<BAD-example>
first answer [^aaa].

[^aaa]: https://example.com
</BAD-example>

<BAD-example>
first answer [^aaa].

<sources>
[^aaa]: https://example.com
</sources>
</BAD-example>
</examples>`
}