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

	_ "klustercost/monitor/pkg/jsonata_ext"
	model "klustercost/monitor/pkg/model"
	signals "klustercost/monitor/pkg/signals"

	jsonata "github.com/blues/jsonata-go"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

var re *regexp.Regexp = regexp.MustCompile(`\$(.+?)\$`)

type metricsTransform struct {
	Query             string `json:"query"`
	Transform         string `json:"transform"`
	expansionKeys     []string
	jsonataExpression *jsonata.Expr
}

func NewMetricsTransformsFromFile(path string) ([]metricsTransform, error) {
	var transforms []metricsTransform
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &transforms)
	if err != nil {
		return nil, err
	}

	for idx := range transforms {
		err := transforms[idx].Compile()
		if err != nil {
			return nil, err
		}
	}

	return transforms, nil
}

func (c *metricsTransform) computeKeySet() {
	c.expansionKeys = re.FindAllString(c.Query, -1)
	slices.Sort(c.expansionKeys)
	c.expansionKeys = slices.Compact(c.expansionKeys)
}

func (c *metricsTransform) compileTransform() error {
	var err error
	c.jsonataExpression, err = jsonata.Compile(c.Transform)
	if err != nil {
		return err
	}
	return nil
}

func (c *metricsTransform) Compile() error {
	c.computeKeySet()
	return c.compileTransform()
}

func (c *metricsTransform) queryWithContext(from model.DataExchange) (string, error) {
	result := c.Query
	for _, key := range c.expansionKeys {
		if value, exists := from[key[1:len(key)-1]]; exists {
			result = strings.Replace(result, key, fmt.Sprintf("%v", value), -1)
		} else {
			return "", fmt.Errorf("Unable to expand key %s for metrics transform %s", key, c.Transform)
		}
	}
	return result, nil
}

func (c *metricsTransform) callAPI(ctx context.Context, query string) ([]byte, error) {
	metric, _, err := apis.GetPrometheusAPI().Query(
		ctx,
		query,
		time.Now(),
		prometheusv1.WithTimeout(5*time.Second))
	if err != nil {
		signals.Logger.Error(err, "Unable to query API for metrics transformation")
		return nil, err
	}

	metricJSON, err := json.Marshal(metric)
	if err != nil {
		signals.Logger.Error(err, "Unable to marshal metric retured from API for metrics transformation")
		return nil, err
	}
	return metricJSON, nil
}

func (c *metricsTransform) AddMetrics(ctx context.Context, transformedObject model.DataExchange, sourceObject []byte) (model.DataExchange, error) {
	metricJSON := sourceObject

	expandedQuery, err := c.queryWithContext(transformedObject)
	if err != nil {
		signals.Logger.Error(err, "Unable to generate query for metrics transformation")
		return transformedObject, err
	}

	if expandedQuery != "" {
		metricJSON, err = c.callAPI(ctx, expandedQuery)
		if err != nil {
			signals.Logger.Error(err, "Unable to call API for metrics transformation")
			return transformedObject, err
		}
	}

	metricJSON, err = c.jsonataExpression.EvalBytes(metricJSON)
	if err != nil {
		signals.Logger.Error(err, "Unable to evaluate template")
		return transformedObject, err
	}

	var extraKeys model.DataExchange
	json.Unmarshal(metricJSON, &extraKeys)
	for k, v := range extraKeys {
		transformedObject[k] = v
	}

	return transformedObject, nil
}
