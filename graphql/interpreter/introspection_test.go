/*
 * FishDB
 *
// Copyright 2025 Fisch-labs
 *
*/

package interpreter

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/Fisch-Labs/Toolkit/lang/graphql/parser"
)

// findNode recursively searches for a node in the AST that satisfies the predicate.
// This is a more robust way to find nodes than using hardcoded indices.
func findNode(node *parser.AST, predicate func(*parser.AST) bool) *parser.AST {
	if predicate(node) {
		return node
	}
	for _, child := range node.Children {
		if found := findNode(child, predicate); found != nil {
			return found
		}
	}
	return nil
}

func readQuery(t *testing.T, fileName string) string {
	t.Helper()
	path := filepath.Join("testdata", fileName)
	query, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read query file %s: %v", fileName, err)
	}
	return string(query)
}

func TestIntrospection(t *testing.T) {
	gm, _ := songGraphGroups()

	t.Run("Full introspection", func(t *testing.T) {
		// This is a standard, full introspection query.
		query := map[string]interface{}{
			"operationName": "IntrospectionQuery",
			"query": `
query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      ...FullType
    }
    directives {
      name
      description
      locations
      args {
        ...InputValue
      }
    }
  }
}

fragment FullType on __Type {
  kind
  name
  description
  fields(includeDeprecated: true) {
    name
    description
    args {
      ...InputValue
    }
    type {
      ...TypeRef
    }
    isDeprecated
    deprecationReason
  }
  inputFields {
    ...InputValue
  }
  interfaces {
    ...TypeRef
  }
  enumValues(includeDeprecated: true) {
    name
    description
    isDeprecated
    deprecationReason
  }
  possibleTypes {
    ...TypeRef
  }
}

fragment InputValue on __InputValue {
  name
  description
  type { ...TypeRef }
  defaultValue
}

fragment TypeRef on __Type {
  kind
  name
  ofType {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
              }
            }
          }
        }
      }
    }
  }
}
`}}

		res, err := runQuery("test", "main", query, gm, nil, false)

		data := res["data"].(map[string]interface{})
		schema := data["__schema"].(map[string]interface{})

		if _, ok := schema["types"]; !ok || err != nil {
			t.Error("Unexpected result:", schema, err)
			return
		}

		// Create runtime provider

		rtp := NewGraphQLRuntimeProvider("test", "main", gm,
			fmt.Sprint(query["operationName"]), make(map[string]interface{}), nil, true)

		// Parse the query and annotate the AST with runtime components

		ast, err := parser.ParseWithRuntime("test", fmt.Sprint(query["query"]), rtp)

		if err != nil {
			t.Error("Unexpected result", err)
		}

		err = ast.Runtime.Validate()

		if err != nil {
			t.Error("Unexpected result", err)
		}

		// Evaluate the query
		// The following is a more robust way to find the selectionSetRuntime
		// than using a hardcoded path of indices.
		schemaFieldNode := findNode(ast, func(node *parser.AST) bool {
			// This predicate identifies the __schema field.
			// You may need to adjust this if your AST structure is different.
			// This assumes the node has a 'Value' field with the name of the field.
			return node.Value == "__schema"
		})

		if schemaFieldNode == nil {
			t.Fatal("Could not find __schema field in the AST")
		}

		var sr *selectionSetRuntime
		for _, child := range schemaFieldNode.Children {
			if ssr, ok := child.Runtime.(*selectionSetRuntime); ok {
				sr = ssr
				break
			}
		}

		if sr == nil {
			t.Fatal("Could not find selectionSetRuntime for __schema field")
		}

		full := formatData(sr.ProcessFullIntrospection())
		filtered := formatData(sr.ProcessIntrospection())

		if full != filtered {

			// This needs thorough investigation - no point in outputting these
			// large datastructures during failure

			t.Error("Full and filtered introspection are different")
			return
		}
	})

	// Now try out a reduced version

	t.Run("Reduced introspection", func(t *testing.T) {
		query := map[string]interface{}{
			"operationName": nil,
			"query": `
	   query IntrospectionQuery {
	     __schema {
	       queryType { name }
	       mutationType { name }
	       subscriptionType { name }
	       directives {
	         name
	         description
	         locations
	         args {
	           ...InputValue
	           ...InputValue @skip(if: true)
				... {
					name
				}
	         }
			name1: name1
	       }
	     }
	   }

	   fragment InputValue on __InputValue {
	     name
	     description
	     type { ...TypeRef }
	     defaultValue
	   }

	   fragment TypeRef on __Type {
	     kind
	     name
	   }
	   `}}

		res, err := runQuery("test", "main", query, gm, nil, false)

		if formatData(res) != `{
  "data": {
    "__schema": {
      "directives": [
        {
          "args": [
            {
              "defaultValue": null,
              "description": "Skipped when true.",
              "name": "if",
              "type": {
                "kind": "NON_NULL",
                "name": null
              }
            }
          ],
          "description": "Directs the executor to skip this field or fragment when the `+"`if`"+` argument is true.",
          "locations": [
            "FIELD",
            "FRAGMENT_SPREAD",
            "INLINE_FRAGMENT"
          ],
          "name": "skip",
          "name1": null
        },
        {
          "args": [
            {
              "defaultValue": null,
              "description": "Included when true.",
              "name": "if",
              "type": {
                "kind": "NON_NULL",
                "name": null
              }
            }
          ],
          "description": "Directs the executor to include this field or fragment only when the `+"`if`"+` argument is true.",
          "locations": [
            "FIELD",
            "FRAGMENT_SPREAD",
            "INLINE_FRAGMENT"
          ],
          "name": "include",
          "name1": null
        }
      ],
      "mutationType": {
        "name": "Mutation"
      },
      "queryType": {
        "name": "Query"
      },,
      "subscriptionType": {
        "name": "Subscription"
      }
    }
  }
}` {
			t.Error("Unexpected result:", formatData(res), err)
			return
		}
	})

	t.Run("Deprecated fields", func(t *testing.T) {
		query := map[string]interface{}{
			"operationName": nil,
			"query":         readQuery(t, "introspection_deprecated_field_query.graphql"),
		}

		res, err := runQuery("test", "main", query, gm, nil, false)
		if err != nil {
			t.Fatalf("runQuery failed: %v", err)
		}

		data := res["data"].(map[string]interface{})
		schema := data["__schema"].(map[string]interface{})
		types := schema["types"].([]interface{})

		var songType map[string]interface{}
		for _, typ := range types {
			t := typ.(map[string]interface{})
			if t["name"] == "Song" {
				songType = t
				break
			}
		}

		if songType == nil {
			t.Fatal("Could not find type Song in introspection result")
		}

		fields := songType["fields"].([]interface{})
		var oldNameField map[string]interface{}
		for _, field := range fields {
			f := field.(map[string]interface{})
			if f["name"] == "oldName" {
				oldNameField = f
				break
			}
		}

		if oldNameField == nil {
			t.Fatal("Could not find field oldName in type Song")
		}

		if oldNameField["isDeprecated"] != true {
			t.Error("Expected oldName field to be deprecated, but it was not")
		}

		if oldNameField["deprecationReason"] == nil {
			t.Error("Expected oldName field to have a deprecation reason, but it did not")
		}
	})

	t.Run("Input types", func(t *testing.T) {
		query := map[string]interface{}{
			"operationName": nil,
			"query":         readQuery(t, "introspection_input_type_query.graphql"),
		}

		res, err := runQuery("test", "main", query, gm, nil, false)
		if err != nil {
			t.Fatalf("runQuery failed: %v", err)
		}

		data := res["data"].(map[string]interface{})
		schema := data["__schema"].(map[string]interface{})
		types := schema["types"].([]interface{})

		var songInputType map[string]interface{}
		for _, typ := range types {
			t := typ.(map[string]interface{})
			if t["name"] == "SongInput" {
				songInputType = t
				break
			}
		}

		if songInputType == nil {
			t.Fatal("Could not find type SongInput in introspection result")
		}

		if songInputType["kind"] != "INPUT_OBJECT" {
			t.Errorf("Expected kind of SongInput to be INPUT_OBJECT, but got %s", songInputType["kind"])
		}

		inputFields := songInputType["inputFields"].([]interface{})
		var nameField map[string]interface{}
		for _, field := range inputFields {
			f := field.(map[string]interface{})
			if f["name"] == "name" {
				nameField = f
				break
			}
		}

		if nameField == nil {
			t.Fatal("Could not find field name in type SongInput")
		}

		typeInfo := nameField["type"].(map[string]interface{})
		if typeInfo["kind"] != "SCALAR" || typeInfo["name"] != "String" {
			t.Errorf("Expected name field to be of type String, but got %s %s", typeInfo["kind"], typeInfo["name"])
		}
	})

	t.Run("Interfaces", func(t *testing.T) {
		query := map[string]interface{}{
			"operationName": nil,
			"query":         readQuery(t, "introspection_interface_query.graphql"),
		}

		res, err := runQuery("test", "main", query, gm, nil, false)
		if err != nil {
			t.Fatalf("runQuery failed: %v", err)
		}

		data := res["data"].(map[string]interface{})
		schema := data["__schema"].(map[string]interface{})
		types := schema["types"].([]interface{})

		var mediaInterface map[string]interface{}
		for _, typ := range types {
			t := typ.(map[string]interface{})
			if t["name"] == "Media" {
				mediaInterface = t
				break
			}
		}

		if mediaInterface == nil {
			t.Fatal("Could not find interface Media in introspection result")
		}

		if mediaInterface["kind"] != "INTERFACE" {
			t.Errorf("Expected kind of Media to be INTERFACE, but got %s", mediaInterface["kind"])
		}

		fields := mediaInterface["fields"].([]interface{})
		var titleField map[string]interface{}
		for _, field := range fields {
			f := field.(map[string]interface{})
			if f["name"] == "title" {
				titleField = f
				break
			}
		}

		if titleField == nil {
			t.Fatal("Could not find field title in interface Media")
		}

		possibleTypes := mediaInterface["possibleTypes"].([]interface{})
		var songPossibleType bool
		for _, pt := range possibleTypes {
			p := pt.(map[string]interface{})
			if p["name"] == "Song" {
				songPossibleType = true
				break
			}
		}

		if !songPossibleType {
			t.Error("Expected Song to be a possible type of Media, but it was not")
		}
	})
}