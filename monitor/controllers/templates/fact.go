package controller

import (
	"context"
	"encoding/json"
	"fmt"
	apis "klustercost/monitor/controllers/apis"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	model "klustercost/monitor/pkg/model"
	signals "klustercost/monitor/pkg/signals"

	"github.com/jsonata-go/jsonata"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

var re *regexp.Regexp = regexp.MustCompile(`\$(.+?)\$`)

type factColumn struct {
	Column            string `json:"column"`
	Query             string `json:"query"`
	Transform         string `json:"transform"`
	expansionKeys     []string
	jsonataExpression jsonata.Expression
}

func NewFactsFromFile(path string, jsonataInstance jsonata.JSONataInstance) ([]factColumn, error) {
	var facts []factColumn
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &facts)
	if err != nil {
		return nil, err
	}

	for idx := range facts {
		err := facts[idx].Compile(jsonataInstance)
		if err != nil {
			return nil, err
		}
	}

	return facts, nil
}

func (c *factColumn) computeKeySet() {
	c.expansionKeys = re.FindAllString(c.Query, -1)
	slices.Sort(c.expansionKeys)
	c.expansionKeys = slices.Compact(c.expansionKeys)
}

func (c *factColumn) compileTransform(jsonataInstance jsonata.JSONataInstance) error {
	var err error
	c.jsonataExpression, err = jsonataInstance.Compile(c.Transform, false)
	if err != nil {
		return err
	}
	return nil
}

func (c *factColumn) Compile(jsonataInstance jsonata.JSONataInstance) error {
	c.computeKeySet()
	return c.compileTransform(jsonataInstance)
}

func (c *factColumn) queryWithContext(from model.DataExchange) (string, error) {
	result := c.Query
	for _, key := range c.expansionKeys {
		if value, exists := from[key[1:len(key)-1]]; exists {
			result = strings.Replace(result, key, fmt.Sprintf("%v", value), -1)
		} else {
			return "", fmt.Errorf("Unable to expand key %s for fact column %s", key, c.Column)
		}
	}
	return result, nil
}

func (c *factColumn) AddFact(ctx context.Context, object model.DataExchange) (model.DataExchange, error) {
	expandedQuery, err := c.queryWithContext(object)
	if err != nil {
		signals.Logger.Error(err, "Unable to generate query for fact transformation")
		return object, err
	}

	metric, _, err := apis.GetPrometheusAPI().Query(
		ctx,
		expandedQuery,
		time.Now(),
		prometheusv1.WithTimeout(5*time.Second))
	if err != nil {
		signals.Logger.Error(err, "Unable to query for fact transformation")
		return object, err
	}

	metricJSON, err := json.Marshal(metric)
	if err != nil {
		signals.Logger.Error(err, "Unable to marshal metric")
		return object, err
	}

	test, err := c.jsonataExpression.Evaluate(metricJSON, nil)
	if err != nil {
		signals.Logger.Error(err, "Unable to evaluate template")
		return object, err
	}

	metricJSON = test

	var extraKeys model.DataExchange
	json.Unmarshal(metricJSON, &extraKeys)
	for k, v := range extraKeys {
		object[k] = v
	}

	return object, nil
}
