/*
 * FishDB
 *
// Copyright 2025 Fisch-labs
 *
*/

/*
Package interpreter contains the GraphQL interpreter for FishDB.
*/
package interpreter

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/Fisch-Labs/Toolkit/lang/graphql/parser"
)

// Not Implemented Runtime
// =======================

/*
Special runtime for not implemented constructs.
*/
type invalidRuntime struct {
	rtp  *GraphQLRuntimeProvider
	node *parser.ASTNode
}

/*
invalidRuntimeInst returns a new runtime component instance.
*/
func invalidRuntimeInst(rtp *GraphQLRuntimeProvider, node *parser.ASTNode) parser.Runtime {
	return &invalidRuntime{rtp, node}
}

/*
Validate this node and all its child nodes.
*/
func (rt *invalidRuntime) Validate() error {
	return rt.rtp.newFatalRuntimeError(ErrInvalidConstruct, rt.node.Name, rt.node)
}

/*
Eval evaluate this runtime component.
*/
func (rt *invalidRuntime) Eval() (map[string]interface{}, error) {
	return nil, rt.rtp.newFatalRuntimeError(ErrInvalidConstruct, rt.node.Name, rt.node)
}

// Value Runtime
// =============

/*
Special runtime for values.
*/
type valueRuntime struct {
	*invalidRuntime
	rtp  *GraphQLRuntimeProvider
	node *parser.ASTNode
}

/*
valueRuntimeInst returns a new runtime component instance.
*/
func valueRuntimeInst(rtp *GraphQLRuntimeProvider, node *parser.ASTNode) parser.Runtime {
	return &valueRuntime{&invalidRuntime{rtp, node}, rtp, node}
}

/*
Value returns the calculated value of the expression.
*/
func (rt *valueRuntime) Value() interface{} {
	switch rt.node.Name {
	case parser.NodeVariable:
		val, ok := rt.rtp.VariableValues[rt.node.Token.Val]
		if !ok {
			rt.rtp.handleRuntimeError(fmt.Errorf(
				"Variable %s was used but not declared", rt.node.Token.Val),
				[]string{}, rt.node)
		}
		return val

	case parser.NodeValue, parser.NodeDefaultValue:
		val := rt.node.Token.Val
		switch rt.node.Token.ID {
		case parser.TokenIntValue:
			i, err := strconv.ParseInt(val, 10, 64)
			if err != nil {
				rt.rtp.handleRuntimeError(fmt.Errorf("invalid integer value %s", val), []string{}, rt.node)
			}
			return i
		case parser.TokenFloatValue:
			f, err := strconv.ParseFloat(val, 64)
			if err != nil {
				rt.rtp.handleRuntimeError(fmt.Errorf("invalid float value %s", val), []string{}, rt.node)
			}
			return f
		case parser.TokenStringValue:
			return val
		}
		if val == "true" {
			return true
		} else if val == "false" {
			return false
		} else if val == "null" {
			return nil
		}
		// default: enum fallback
		return val

	case parser.NodeObjectValue:
		res := make(map[string]interface{})
		for _, c := range rt.node.Children {
			res[c.Token.Val] = c.Children[0].Runtime.(*valueRuntime).Value()
		}
		return res

	case parser.NodeListValue:
		res := make([]interface{}, 0)
		for _, c := range rt.node.Children {
			res = append(res, c.Runtime.(*valueRuntime).Value())
		}
		return res
	}

	// fallback
	return rt.node.Token.Val
}

// Data sorting
// ============

/*
dataSort sorts a list of maps.
*/
func dataSort(list []map[string]interface{}, attr string, ascending bool) {
	sort.Sort(&DataSlice{list, attr, ascending})
}

/*
DataSlice attaches the methods of sort.Interface to []map[string]interface{},
sorting in ascending or descending order by a given attribute.
*/
type DataSlice struct {
	data      []map[string]interface{}
	attr      string
	ascending bool
}

/*
Len belongs to the sort.Interface.
*/
func (d DataSlice) Len() int { return len(d.data) }

/*
Less belongs to the sort.Interface.
*/
func (d DataSlice) Less(i, j int) bool {
	ia, ok1 := d.data[i][d.attr]
	ja, ok2 := d.data[j][d.attr]
	if !ok1 || !ok2 {
		return false
	}

	// Try numbers first
	is := fmt.Sprint(ia)
	js := fmt.Sprint(ja)
	in, err1 := strconv.Atoi(is)
	jn, err2 := strconv.Atoi(js)

	if err1 == nil && err2 == nil {
		if d.ascending {
			return in < jn
		}
		return in > jn
	}

	// Compare as strings
	if d.ascending {
		return is < js
	}
	return is > js
}

/*
Swap belongs to the sort.Interface.
*/
func (d DataSlice) Swap(i, j int) {
	d.data[i], d.data[j] = d.data[j], d.data[i]
}
