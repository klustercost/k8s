package controller

import (
	"context"
	"encoding/json"
	"os"

	jsonata "github.com/blues/jsonata-go"
	"k8s.io/klog/v2"

	model "klustercost/monitor/pkg/model"
)

type Transform struct {
	logger            klog.Logger
	labelsTransform   *jsonata.Expr
	metricsTransforms []metricsTransform
}

func NewTransform(ctx context.Context, path string) *Transform {
	logger := klog.FromContext(ctx)

	var metricsTransforms []metricsTransform

	metricsTransforms, err := NewMetricsTransformsFromFile(path + "metrics.json")
	if err != nil {
		logger.Error(err, "Unable to read metrics transforms")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	transform, err := getTransform(path + "labels.jsonata")
	if err != nil {
		logger.Error(err, "Unable to read template transform")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	return &Transform{
		logger:            logger,
		labelsTransform:   transform,
		metricsTransforms: metricsTransforms,
	}
}

func getTransform(path string) (*jsonata.Expr, error) {
	transform, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	expression, err := jsonata.Compile(string(transform))
	if err != nil {
		return nil, err
	}
	return expression, nil
}

func (c *Transform) addMetrics(ctx context.Context, keyValues model.DataExchange, sourceJSON []byte) (model.DataExchange, error) {
	for _, transform := range c.metricsTransforms {
		keyValues, err := transform.AddMetrics(ctx, keyValues, sourceJSON)
		if err != nil {
			return keyValues, err
		}
	}
	return keyValues, nil
}

func (c *Transform) getTransformedObject(sourceJSON []byte) (model.DataExchange, error) {
	var transformedObject model.DataExchange
	transformedJSON, err := c.labelsTransform.EvalBytes(sourceJSON)
	if err != nil {
		c.logger.Error(err, "Unable to evaluate template")
		return nil, err
	}
	err = json.Unmarshal(transformedJSON, &transformedObject)
	if err != nil {
		c.logger.Error(err, "Unable to unmarshal transformed JSON to object")
		return nil, err
	}
	return transformedObject, nil
}

func (c *Transform) Transform(ctx context.Context, source any) ([]byte, error) {
	sourceJSON, err := json.Marshal(source)
	if err != nil {
		c.logger.Error(err, "Unable to marshal source to JSON")
		return nil, err
	}
	transformedObject, err := c.getTransformedObject(sourceJSON)
	if err != nil {
		c.logger.Error(err, "Unable to get transformed object")
		return nil, err
	}
	transformedObject, err = c.addMetrics(ctx, transformedObject, sourceJSON)
	if err != nil {
		c.logger.Error(err, "Unable to add metrics to object")
		return nil, err
	}
	return json.Marshal(transformedObject)
}
