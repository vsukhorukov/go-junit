// Copyright Josh Komoroske. All rights reserved.
// Use of this source code is governed by the MIT license,
// a copy of which can be found in the LICENSE.txt file.
// SPDX-License-Identifier: MIT

package junit

import (
	"strconv"
	"strings"
	"time"
)

const timestampLayout = "2006-01-02T15:04:05" // ISO8601

// findSuites performs a depth-first search through the XML document, and
// attempts to ingest any "testsuite" tags that are encountered.
func findSuites(nodes []xmlNode, suites chan Suite) {
	for _, node := range nodes {
		switch node.XMLName.Local {
		case "testsuite":
			suites <- ingestSuite(node)
		default:
			findSuites(node.Nodes, suites)
		}
	}
}

func ingestSuite(root xmlNode) Suite {
	suite := Suite{
		Name:       root.Attr("name"),
		Package:    root.Attr("package"),
		Properties: root.Attrs,
	}
	if root.Attr("timestamp") != "" {
		if timestamp, err := time.Parse(timestampLayout, root.Attr("timestamp")); err == nil {
			suite.Timestamp = timestamp
		}
	}

	for _, node := range root.Nodes {
		switch node.XMLName.Local {
		case "testsuite":
			testsuite := ingestSuite(node)
			suite.Suites = append(suite.Suites, testsuite)
		case "testcase":
			testcase := ingestTestcase(node)
			suite.Tests = append(suite.Tests, testcase)
		case "properties":
			props := ingestProperties(node)
			for k, v := range props {
				suite.Properties[k] = v
			}
		case "system-out":
			suite.SystemOut = string(node.Content)
		case "system-err":
			suite.SystemErr = string(node.Content)
		}
	}

	suite.Aggregate()

	return suite
}

func ingestProperties(root xmlNode) map[string]string {
	props := make(map[string]string, len(root.Nodes))

	for _, node := range root.Nodes {
		if node.XMLName.Local == "property" {
			name := node.Attr("name")
			value := node.Attr("value")
			props[name] = value
		}
	}

	return props
}

func ingestTestcase(root xmlNode) Test {
	test := Test{
		Name:       root.Attr("name"),
		Classname:  root.Attr("classname"),
		Duration:   duration(root.Attr("time")),
		Status:     StatusPassed,
		Properties: root.Attrs,
	}

	for _, node := range root.Nodes {
		switch node.XMLName.Local {
		case "skipped":
			test.Status = StatusSkipped
			test.Message = node.Attr("message")
		case "failure":
			test.Status = StatusFailed
			test.Message = node.Attr("message")
			test.Error = ingestError(node)
		case "error":
			test.Status = StatusError
			test.Message = node.Attr("message")
			test.Error = ingestError(node)
		case "system-out":
			test.SystemOut = string(node.Content)
		case "system-err":
			test.SystemErr = string(node.Content)
		case "properties":
			props := ingestProperties(node)
			for k, v := range props {
				test.Properties[k] = v
			}
		}
	}

	return test
}

func ingestError(root xmlNode) Error {
	return Error{
		Body:    string(root.Content),
		Type:    root.Attr("type"),
		Message: root.Attr("message"),
	}
}

func duration(timespec string) time.Duration {
	// Remove commas for larger durations
	timespec = strings.ReplaceAll(timespec, ",", "")

	// Check if there was a valid decimal value
	if s, err := strconv.ParseFloat(timespec, 64); err == nil {
		return time.Duration(s * float64(time.Second))
	}

	// Check if there was a valid duration string
	if d, err := time.ParseDuration(timespec); err == nil {
		return d
	}

	return 0
}
