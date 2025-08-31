/*
 * FishDB
 *
// Copyright 2025 Fisch-labs
 *
*/

/*
Package graphql contains the main API for GraphQL.

Example GraphQL query:

	{
		Person @withValue(name : "Marvin") {
			key
			kind
			name
		}
	}
*/
package graphql

import (
	"fmt"

	"github.com/Fisch-Labs/FishDB/graph"
	"github.com/Fisch-Labs/FishDB/graphql/interpreter"
	"github.com/Fisch-Labs/Toolkit/lang/graphql/parser"
)

/*
RunQuery runs a GraphQL query against a given graph database. The query parameter
needs to have the following fields:

	operationName - Operation to Execute (string)
	query         - Query document (string)
	variables     - Variables map (map[string]interface{})

Set the readOnly flag if the query should only be allowed to do read operations.
*/
func RunQuery(name string, part string, query map[string]interface{},
	gm *graph.Manager, callbackHandler interpreter.SubscriptionCallbackHandler,
	readOnly bool) (map[string]interface{}, error) {

	var ok bool
	var vars map[string]interface{}
	const (
		KeyOperationName = "operationName"
		KeyQuery         = "query"
		KeyVariables     = "variables"
	)
	// Make sure all info is present on the query object

	for _, op := range []string{KeyOperationName, KeyQuery, KeyVariables} {
		if _, ok := query[op]; !ok {
			return nil, fmt.Errorf("Mandatory field '%s' missing from query object", op)
		}
	}

	// Nil pointer become empty strings

	if query["operationName"] == nil {
		query["operationName"] = ""
	}
	if query["query"] == nil {
		query["query"] = ""
	}

	if vars, ok = query["variables"].(map[string]interface{}); !ok {
		vars = make(map[string]interface{})
	}

	// Create runtime provider

	rtp := interpreter.NewGraphQLRuntimeProvider(name, part, gm,
		fmt.Sprint(query["operationName"]), vars, callbackHandler, readOnly)

	// Parse the query and annotate the AST with runtime components

	ast, err := parser.ParseWithRuntime(name, fmt.Sprint(query["query"]), rtp)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare GraphQL query :%w", err)
	}
	err = ast.Runtime.Validate()
	if err != nil {
		return nil, fmt.Errorf("GraphQL query validation failed :%w", err)
	}
	result, err := ast.Runtime.Eval()
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate GraphQL query : %w", err)
	}
	return result, nil
}

/*
ParseQuery parses a GraphQL query and return its Abstract Syntax Tree.
*/
func ParseQuery(name string, query string) (*parser.ASTNode, error) {
	ast, err := parser.ParseWithRuntime(name, query, nil)

	if err != nil {
		return nil, err
	}

	return ast, nil
}
