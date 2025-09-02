/*
 * FishDB
 *
// Copyright 2025 Fisch-labs
 *
*/

package interpreter

import (
	"fmt"
	"strings"

	"github.com/Fisch-Labs/Toolkit/lang/graphql/parser"
)

/*
ProcessIntrospection filters the full introspection down to the required fields.
*/
func (rt *selectionSetRuntime) ProcessIntrospection() map[string]interface{} {
	return rt.FilterIntrospectionResponse(rt.ProcessFullIntrospection())
}

/*
ProcessFullIntrospection returns the full introspection with all known fields.
*/
func (rt *selectionSetRuntime) ProcessFullIntrospection() map[string]interface{} {
	res := make(map[string]interface{})

	fieldMap := rt.GetFields()

	for symbol := range fieldMap {

		// General types

		if symbol == "queryType" {
			res["queryType"] = map[string]interface{}{
				"name": "Query",
			}

			if !rt.rtp.readOnly {
				res["mutationType"] = map[string]interface{}{
					"name": "Mutation",
				}
			} else {
				res["mutationType"] = nil
			}

			res["subscriptionType"] = map[string]interface{}{
				"name": "Subscription",
			}
		}

		if symbol == "types" {
			res["types"] = rt.GetTypesIntrospection()
		}

		if symbol == "directives" {
			res["directives"] = rt.GetDirectivesIntrospection()
		}
	}

	return res
}

func (rt *selectionSetRuntime) FilterIntrospectionResponse(res map[string]interface{}) map[string]interface{} {
	filteredRes := make(map[string]interface{})

	fieldMap := rt.GetFields()

	for symbol, field := range fieldMap {
		reschild := res[symbol]

		if srt := field.SelectionSetRuntime(); srt != nil {

			// Check for list

			if reschildList, ok := reschild.([]interface{}); ok {
				filterResList := []interface{}{}

				for _, reschild := range reschildList {
					filterResList = append(filterResList, srt.FilterIntrospectionResponse(reschild.(map[string]interface{})))
				}

				filteredRes[symbol] = filterResList

			} else if reschildMap, ok := reschild.(map[string]interface{}); ok {

				filteredRes[symbol] = srt.FilterIntrospectionResponse(reschildMap)

			} else {

				filteredRes[symbol] = reschild
			}

		} else {

			filteredRes[symbol] = reschild
		}
	}

	return filteredRes
}

/*
GetTypesIntrospection returns the introspection for all available types.
*/
func (rt *selectionSetRuntime) GetTypesIntrospection() interface{} {
	res := make([]interface{}, 0)

	queryType := map[string]interface{}{
		"kind":          "OBJECT",
		"name":          "Query",
		"description":   "Entry point for single read queryies.",
		"fields":        rt.GetFieldTypesIntrospection("Lookup", true),
		"inputFields":   nil,
		"interfaces":    []interface{}{},
		"enumValues":    nil,
		"possibleTypes": nil,
	}
	res = append(res, queryType)

	if !rt.rtp.readOnly {
		mutationType := map[string]interface{}{
			"kind":          "OBJECT",
			"name":          "Mutation",
			"description":   "Entry point for writing queryies.",
			"fields":        rt.GetFieldTypesIntrospection("Insert or modify", false),
			"inputFields":   nil,
			"interfaces":    []interface{}{},
			"enumValues":    nil,
			"possibleTypes": nil,
		}
		res = append(res, mutationType)
	}

	subscriptionType := map[string]interface{}{
		"kind":          "OBJECT",
		"name":          "Subscription",
		"description":   "Entry point for subscriptions.",
		"fields":        rt.GetFieldTypesIntrospection("Subscribe to", true),
		"inputFields":   nil,
		"interfaces":    []interface{}{},
		"enumValues":    nil,
		"possibleTypes": nil,
	}
	res = append(res, subscriptionType)

	// Add FishDB specific types

	res = append(res, rt.GetFishDBTypesIntrospection().([]interface{})...)

	// Add all the default GraphQL types like __Schema, __Type, etc.

	res = append(res, rt.GetStandardTypesIntrospection().([]interface{})...)

	return res
}

/*
GetFieldTypesIntrospection returns the introspection for all available field types.
*/
func (rt *selectionSetRuntime) GetFieldTypesIntrospection(action string, lookupArgs bool) interface{} {
	var args []interface{}

	res := make([]interface{}, 0)

	if lookupArgs {
		args = []interface{}{
			rt.newArg("key", "Lookup a particular node by key.", rt.newType("SCALAR", "String", nil), nil),
			rt.newArg("matches", "Lookup nodes matching this template.", rt.newType("OBJECT", "NodeTemplate", nil), nil),
			rt.newArg("storeNode", "Store a node according to this template.", rt.newType("OBJECT", "NodeTemplate", nil), nil),
			rt.newArg("removeNode", "Remove a node according to this template (only kind is needed).", rt.newType("OBJECT", "NodeTemplate", nil), nil),
			rt.newArg("storeEdge", "Store an edge according to this template.", rt.newType("OBJECT", "NodeTemplate", nil), nil),
			rt.newArg("removeEdge", "Remove an edge according to this template (only key and kind are needed).", rt.newType("OBJECT", "NodeTemplate", nil), nil),
			rt.newArg("ascending", "Sort resuting data ascending using the values of the specified key.", rt.newType("SCALAR", "String", nil), nil),
			rt.newArg("descending", "Sort resuting data descending using the values of the specified key.", rt.newType("SCALAR", "String", nil), nil),
			rt.newArg("from", "Retrieve data after the first n entries.", rt.newType("SCALAR", "Int", nil), nil),
			rt.newArg("items", "Retrieve n entries.", rt.newType("SCALAR", "Int", nil), nil),
			rt.newArg("last", "Only return last n entries.", rt.newType("SCALAR", "Int", nil), nil),
		}
	} else {

		args = []interface{}{}
	}

	for _, kind := range rt.rtp.gm.NodeKinds() {

		res = append(res, rt.newField(kind, fmt.Sprintf("%s %s nodes in the datastore.", action, kind), args, rt.newType("LIST", nil, rt.newType("OBJECT", fmt.Sprintf("%sNode", strings.Title(kind)), nil))))
	}

	return res
}

/*
newArg is a helper for creating an argument object in the introspection response.
*/
func (rt *selectionSetRuntime) newArg(name string, description string, typeValue interface{}, defaultValue interface{}) map[string]interface{} {
	return map[string]interface{}{
		"name":         name,
		"description":  description,
		"type":         typeValue,
		"defaultValue": defaultValue,
	}
}

/*
GetFishDBTypesIntrospection returns FishDB types.
*/
func (rt *selectionSetRuntime) GetFishDBTypesIntrospection() interface{} {
	res := make([]interface{}, 0)

	for _, kind := range rt.rtp.gm.NodeKinds() {

		fields := make([]interface{}, 0)

		for _, attr := range rt.rtp.gm.NodeAttrs(kind) {

			fields = append(fields, rt.newField(attr, fmt.Sprintf("The %s attribute of a %s node.", attr, kind), []interface{}{}, rt.newType("SCALAR", "String", nil)))
		}

		for _, edge := range rt.rtp.gm.NodeEdges(kind) {
			edgeName := strings.Replace(edge, ":", "_", -1)
			edgeTargetKind := strings.Split(edge, ":")[3]

			fields = append(fields, rt.newField(edgeName, fmt.Sprintf("The %s edge of a %s node to a %s node.", edge, kind, edgeTargetKind), []interface{}{
				rt.newArg("traverse", fmt.Sprintf("Use %s to traverse from %s to %s.", edge, kind, edgeTargetKind), rt.newType("NON_NULL", nil, rt.newType("SCALAR", "String", nil)), nil),
				rt.newArg("matches", "Lookup nodes matching this template.", rt.newType("OBJECT", "NodeTemplate", nil), nil),
				rt.newArg("ascending", "Sort resuting data ascending using the values of the specified key.", rt.newType("SCALAR", "String", nil), nil),
				rt.newArg("descending", "Sort resuting data descending using the values of the specified key.", rt.newType("SCALAR", "String", nil), nil),
				rt.newArg("from", "Retrieve data after the first n entries.", rt.newType("SCALAR", "Int", nil), nil),
				rt.newArg("items", "Retrieve n entries.", rt.newType("SCALAR", "Int", nil), nil),
				rt.newArg("last", "Only return last n entries.", rt.newType("SCALAR", "Int", nil), nil),
			}, rt.newType("LIST", nil, rt.newType("OBJECT", fmt.Sprintf("%sNode", strings.Title(edgeTargetKind)), nil))))
		}

		res = append(res, map[string]interface{}{
			"kind":          "OBJECT",
			"name":          fmt.Sprintf("%sNode", strings.Title(kind)),
			"description":   fmt.Sprintf("Represents a %s node.", kind),
			"fields":        fields,
			"inputFields":   nil,
			"interfaces":    []interface{}{},
			"enumValues":    nil,
			"possibleTypes": nil,
		})
	}

	res = append(res, map[string]interface{}{
		"kind":          "INPUT_OBJECT",
		"name":          "NodeTemplate",
		"description":   "Template of a node. Fields of this object can either be regular expressions or direct matches. A `not_` prefix negates the condition (e.g. `not_key`).",
		"fields":        []interface{}{},
		"inputFields":   []interface{}{},
		"interfaces":    []interface{}{},
		"enumValues":    nil,
		"possibleTypes": nil,
	})

	return res
}

/*
GetStandardTypesIntrospection returns the standard types.
*/
func (rt *selectionSetRuntime) GetStandardTypesIntrospection() interface{} {
	res := make([]interface{}, 0)

	// Schema type

	res = append(res, map[string]interface{}{
		"kind":        "OBJECT",
		"name":        "__Schema",
		"description": "A GraphQL Schema defines the capabilities of a GraphQL server. It exposes all available types and directives on the server, as well as the entry points for query, mutation, and subscription operations.",
		"fields": []interface{}{
			rt.newField("types", "A list of all types supported by this server.", []interface{}{}, rt.newType("NON_NULL", nil, rt.newType("LIST", nil, rt.newType("NON_NULL", nil, rt.newType("OBJECT", "__Type", nil))))),
			rt.newField("queryType", "The type that query operations will be rooted at.", []interface{}{}, rt.newType("NON_NULL", nil, rt.newType("OBJECT", "__Type", nil))),
			rt.newField("mutationType", "The type that mutation operations will be rooted at.", []interface{}{}, rt.newType("OBJECT", "__Type", nil)),
			rt.newField("subscriptionType", "The type that subscription operations will be rooted at.", []interface{}{}, rt.newType("OBJECT", "__Type", nil)),
			rt.newField("directives", "A list of all directives supported by this server.", []interface{}{}, rt.newType("NON_NULL", nil, rt.newType("LIST", nil, rt.newType("NON_NULL", nil, rt.newType("OBJECT", "__Directive", nil))))),
		},
		"inputFields":   nil,
		"interfaces":    []interface{}{},
		"enumValues":    nil,
		"possibleTypes": nil,
	})

	// Type type

	res = append(res, map[string]interface{}{
		"kind":        "OBJECT",
		"name":        "__Type",
		"description": "The fundamental unit of the GraphQL Schema.",
		"fields": []interface{}{
			rt.newField("kind", nil, []interface{}{}, rt.newType("NON_NULL", nil, rt.newType("ENUM", "__TypeKind", nil))),
			rt.newField("name", nil, []interface{}{}, rt.newType("SCALAR", "String", nil)),
			rt.newField("description", nil, []interface{}{}, rt.newType("SCALAR", "String", nil)),
			rt.newField("fields", nil, []interface{}{
				map[string]interface{}{
					"name":         "includeDeprecated",
					"description":  nil,
					"type":         rt.newType("SCALAR", "Boolean", nil),
					"defaultValue": "false",
				},
			}, rt.newType("LIST", nil, rt.newType("NON_NULL", nil, rt.newType("OBJECT", "__Field", nil)))),
			rt.newField("interfaces", nil, []interface{}{}, rt.newType("LIST", nil, rt.newType("NON_NULL", nil, rt.newType("OBJECT", "__Type", nil)))),
			rt.newField("possibleTypes", nil, []interface{}{}, rt.newType("LIST", nil, rt.newType("NON_NULL", nil, rt.newType("OBJECT", "__Type", nil)))),
			rt.newField("enumValues", nil, []interface{}{
				map[string]interface{}{
					"name":         "includeDeprecated",
					"description":  nil,
					"type":         rt.newType("SCALAR", "Boolean", nil),
					"defaultValue": "false",
				},
			}, rt.newType("LIST", nil, rt.newType("NON_NULL", nil, rt.newType("OBJECT", "__EnumValue", nil)))),
			rt.newField("inputFields", nil, []interface{}{}, rt.newType("LIST", nil, rt.newType("NON_NULL", nil, rt.newType("OBJECT", "__InputValue", nil)))),
			rt.newField("ofType", nil, []interface{}{}, rt.newType("OBJECT", "__Type", nil)),
		},
		"inputFields":   nil,
		"interfaces":    []interface{}{},
		"enumValues":    nil,
		"possibleTypes": nil,
	})

	// Default types

	res = append(res, []interface{}{
		map[string]interface{}{
			"kind":          "SCALAR",
			"name":          "String",
			"description":   "The `String` scalar type represents textual data, represented as UTF-8 character sequences.",
			"fields":        nil,
			"inputFields":   nil,
			"interfaces":    nil,
			"enumValues":    nil,
			"possibleTypes": nil,
		},
		map[string]interface{}{
			"kind":          "SCALAR",
			"name":          "Boolean",
			"description":   "The `Boolean` scalar type represents `true` or `false`.",
			"fields":        nil,
			"inputFields":   nil,
			"interfaces":    nil,
			"enumValues":    nil,
			"possibleTypes": nil,
		},

		map[string]interface{}{
			"kind":          "SCALAR",
			"name":          "Float",
			"description":   "The `Float` scalar type represents signed double-precision fractional values.",
			"fields":        nil,
			"inputFields":   nil,
			"interfaces":    nil,
			"enumValues":    nil,
			"possibleTypes": nil,
		},

		map[string]interface{}{
			"kind":          "SCALAR",
			"name":          "Int",
			"description":   "The `Int` scalar type represents non-fractional signed whole numeric values.",
			"fields":        nil,
			"inputFields":   nil,
			"interfaces":    nil,
			"enumValues":    nil,
			"possibleTypes": nil,
		},
		map[string]interface{}{
			"kind":        "OBJECT",
			"name":        "__InputValue",
			"description": "Arguments provided to Fields or Directives and the input fields of an InputObject are represented as Input Values which describe their type and optionally a default value.",
			"fields": []interface{}{
				rt.newField("name", nil, []interface{}{}, rt.newType("NON_NULL", nil, rt.newType("SCALAR", "String", nil))),
				rt.newField("description", nil, []interface{}{}, rt.newType("SCALAR", "String", nil)),
				rt.newField("type", nil, []interface{}{}, rt.newType("NON_NULL", nil, rt.newType("OBJECT", "__Type", nil))),
				rt.newField("defaultValue", "A GraphQL-formatted string representing the default value for this input value.", []interface{}{}, rt.newType("SCALAR", "String", nil)),
			},
			"inputFields":   nil,
			"interfaces":    []interface{}{},
			"enumValues":    nil,
			"possibleTypes": nil,
		},
		map[string]interface{}{
			"kind":        "OBJECT",
			"name":        "__EnumValue",
			"description": "One possible value for a given Enum. Enum values are unique values, not a placeholder for a string or numeric value. Enum values are returned in a JSON response as strings.",
			"fields": []interface{}{
				rt.newField("name", nil, []interface{}{}, rt.newType("NON_NULL", nil, rt.newType("SCALAR", "String", nil))),
				rt.newField("description", nil, []interface{}{}, rt.newType("SCALAR", "String", nil)),
				rt.newField("isDeprecated", nil, []interface{}{}, rt.newType("NON_NULL", nil, rt.newType("SCALAR", "Boolean", nil))),
				rt.newField("deprecationReason", nil, []interface{}{}, rt.newType("SCALAR", "String", nil)),
			},
			"inputFields":   nil,
			"interfaces":    []interface{}{},
			"enumValues":    nil,
			"possibleTypes": nil,
		},
		map[string]interface{}{
			"kind":        "ENUM",
			"name":        "__TypeKind",
			"description": "An enum describing what kind of type a given `__Type` is.",
			"fields":      nil,
			"inputFields": nil,
			"interfaces":  nil,
			"enumValues": []interface{}{
				map[string]interface{}{
					"name":              "SCALAR",
					"description":       "Indicates this type is a scalar.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "OBJECT",
					"description":       "Indicates this type is an object. `fields` and `interfaces` are valid fields.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "INTERFACE",
					"description":       "Indicates this type is an interface. `fields` and `possibleTypes` are valid fields.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "UNION",
					"description":       "Indicates this type is a union. `possibleTypes` is a valid field.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "ENUM",
					"description":       "Indicates this type is an enum. `enumValues` is a valid field.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "INPUT_OBJECT",
					"description":       "Indicates this type is an input object. `inputFields` is a valid field.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "LIST",
					"description":       "Indicates this type is a list. `ofType` is a valid field.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "NON_NULL",
					"description":       "Indicates this type is a non-null. `ofType` is a valid field.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
			},
			"possibleTypes": nil,
		},
		map[string]interface{}{
			"kind":        "OBJECT",
			"name":        "__Field",
			"description": "Object and Interface types are described by a list of Fields, each of which has a name, potentially a list of arguments, and a return type.",
			"fields": []interface{}{
				rt.newField("name", nil, []interface{}{}, rt.newType("NON_NULL", nil, rt.newType("SCALAR", "String", nil))),
				rt.newField("description", nil, []interface{}{}, rt.newType("SCALAR", "String", nil)),
				rt.newField("args", nil, []interface{}{}, rt.newType("NON_NULL", nil, rt.newType("LIST", nil, rt.newType("NON_NULL", nil, rt.newType("OBJECT", "__InputValue", nil))))),
				rt.newField("type", nil, []interface{}{}, rt.newType("NON_NULL", nil, rt.newType("OBJECT", "__Type", nil))),
				rt.newField("isDeprecated", nil, []interface{}{}, rt.newType("NON_NULL", nil, rt.newType("SCALAR", "Boolean", nil))),
				rt.newField("deprecationReason", nil, []interface{}{}, rt.newType("SCALAR", "String", nil)),
			},
			"inputFields":   nil,
			"interfaces":    []interface{}{},
			"enumValues":    nil,
			"possibleTypes": nil,
		},

		map[string]interface{}{
			"kind":        "OBJECT",
			"name":        "__Directive",
			"description": "A Directive provides a way to describe alternate runtime execution and type validation behavior in a GraphQL document.",
			"fields": []interface{}{
				rt.newField("name", nil, []interface{}{}, rt.newType("NON_NULL", nil, rt.newType("SCALAR", "String", nil))),
				rt.newField("description", nil, []interface{}{}, rt.newType("SCALAR", "String", nil)),
				rt.newField("locations", nil, []interface{}{}, rt.newType("NON_NULL", nil, rt.newType("LIST", nil, rt.newType("NON_NULL", nil, rt.newType("ENUM", "__DirectiveLocation", nil))))),
				rt.newField("args", nil, []interface{}{}, rt.newType("NON_NULL", nil, rt.newType("LIST", nil, rt.newType("NON_NULL", nil, rt.newType("OBJECT", "__InputValue", nil))))),
			},
			"inputFields":   nil,
			"interfaces":    []interface{}{},
			"enumValues":    nil,
			"possibleTypes": nil,
		},
		map[string]interface{}{
			"kind":        "ENUM",
			"name":        "__DirectiveLocation",
			"description": "A Directive can be adjacent to many parts of the GraphQL language, a __DirectiveLocation describes one such possible adjacencies.",
			"fields":      nil,
			"inputFields": nil,
			"interfaces":  nil,
			"enumValues": []interface{}{
				map[string]interface{}{
					"name":              "QUERY",
					"description":       "Location adjacent to a query operation.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "MUTATION",
					"description":       "Location adjacent to a mutation operation.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "SUBSCRIPTION",
					"description":       "Location adjacent to a subscription operation.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "FIELD",
					"description":       "Location adjacent to a field.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "FRAGMENT_DEFINITION",
					"description":       "Location adjacent to a fragment definition.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "FRAGMENT_SPREAD",
					"description":       "Location adjacent to a fragment spread.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "INLINE_FRAGMENT",
					"description":       "Location adjacent to an inline fragment.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "SCHEMA",
					"description":       "Location adjacent to a schema definition.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "SCALAR",
					"description":       "Location adjacent to a scalar definition.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "OBJECT",
					"description":       "Location adjacent to an object type definition.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "FIELD_DEFINITION",
					"description":       "Location adjacent to a field definition.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "ARGUMENT_DEFINITION",
					"description":       "Location adjacent to an argument definition.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "INTERFACE",
					"description":       "Location adjacent to an interface definition.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "UNION",
					"description":       "Location adjacent to a union definition.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "ENUM",
					"description":       "Location adjacent to an enum definition.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "ENUM_VALUE",
					"description":       "Location adjacent to an enum value definition.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "INPUT_OBJECT",
					"description":       "Location adjacent to an input object type definition.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
				map[string]interface{}{
					"name":              "INPUT_FIELD_DEFINITION",
					"description":       "Location adjacent to an input object field definition.",
					"isDeprecated":      false,
					"deprecationReason": nil,
				},
			},
			"possibleTypes": nil,
		},
	}...)

	return res
}

/*
newField is a helper for creating a __Field object in the introspection response.
*/
func (rt *selectionSetRuntime) newField(name string, description interface{}, args []interface{}, typeValue interface{}) map[string]interface{} {
	return map[string]interface{}{
		"name":              name,
		"description":       description,
		"args":              args,
		"type":              typeValue,
		"isDeprecated":      false,
		"deprecationReason": nil,
	}
}

/*
GetDirectivesIntrospection returns the introspection for all available directives.
*/
func (rt *selectionSetRuntime) GetDirectivesIntrospection() interface{} {

	return []interface{}{
		map[string]interface{}{
			"name":        "skip",
			"description": "Directs the executor to skip this field or fragment when the `if` argument is true.",
			"locations": []interface{}{
				"FIELD",
				"FRAGMENT_SPREAD",
				"INLINE_FRAGMENT",
			},
			"args": []interface{}{
				map[string]interface{}{
					"name":         "if",
					"description":  "Skipped when true.",
					"type":         rt.newType("NON_NULL", nil, rt.newType("SCALAR", "Boolean", nil)),
					"defaultValue": nil,
				},
			},
		},
		map[string]interface{}{
			"name":        "include",
			"description": "Directs the executor to include this field or fragment only when the `if` argument is true.",
			"locations": []interface{}{
				"FIELD",
				"FRAGMENT_SPREAD",
				"INLINE_FRAGMENT",
			},
			"args": []interface{}{
				map[string]interface{}{
					"name":         "if",
					"description":  "Included when true.",
					"type":         rt.newType("NON_NULL", nil, rt.newType("SCALAR", "Boolean", nil)),
					"defaultValue": nil,
				},
			},
		},
	}
}

/*
newType is a helper for creating a __Type object in the introspection response.
*/
func (rt *selectionSetRuntime) newType(kind string, name interface{}, ofType interface{}) map[string]interface{} {
	return map[string]interface{}{
		"kind":   kind,
		"name":   name,
		"ofType": ofType,
	}
}

/*
GetFields returns all fields of this selection set.
*/
func (rt *selectionSetRuntime) GetFields() map[string]*fieldRuntime {
	resMap := make(map[string]*fieldRuntime)
	fieldList := append(rt.node.Children[:0:0], rt.node.Children...) // Copy into new slice

	for i := 0; i < len(fieldList); i++ {
		c := fieldList[i]

		// Check for skip and include directive

		if rt.skipField([]string{}, c) {
			continue
		}

		if c.Name == parser.NodeField {

			// Handle simple fields - we ignore aliases as they will not be honored
			// when filtering the introspection data

			field := c.Runtime.(*fieldRuntime)

			resMap[field.Name()] = field

		} else if c.Name == parser.NodeFragmentSpread || c.Name == parser.NodeInlineFragment {
			var fd fragmentRuntime

			if c.Name == parser.NodeFragmentSpread {

				// Lookup fragment spreads

				fd = rt.rtp.fragments[c.Token.Val]

			} else {

				// Construct inline fragments

				fd = c.Runtime.(*inlineFragmentDefinitionRuntime)
			}

			ss := fd.SelectionSet()
			fieldList = append(fieldList, ss.Children...)
		}
	}

	return resMap
}
