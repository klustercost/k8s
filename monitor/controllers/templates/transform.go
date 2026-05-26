package controller

import (
	"context"
	"encoding/json"
	"os"

	"github.com/jsonata-go/jsonata"
	"k8s.io/klog/v2"

	model "klustercost/monitor/pkg/model"
)

type Transform struct {
	jsonataInstance   jsonata.JSONataInstance
	logger            klog.Logger
	labelsTransform   jsonata.Expression
	metricsTransforms []metricsTransform
}

func NewTransform(ctx context.Context, path string) *Transform {
	logger := klog.FromContext(ctx)

	instance, err := jsonata.OpenLatest()
	if err != nil {
		logger.Error(err, "Unable to create JSONata instance")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	var metricsTransforms []metricsTransform

	metricsTransforms, err = NewMetricsTransformsFromFile(path+"fact.json", instance)
	if err != nil {
		logger.Error(err, "Unable to read metrics transforms")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	transform, err := getTransform(path+"object.jsonata", instance)
	if err != nil {
		logger.Error(err, "Unable to read template transform")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	return &Transform{
		jsonataInstance:   instance,
		logger:            logger,
		labelsTransform:   transform,
		metricsTransforms: metricsTransforms,
	}
}

func getTransform(path string, jsonataInstance jsonata.JSONataInstance) (jsonata.Expression, error) {
	transform, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	expression, err := jsonataInstance.Compile(string(transform), false)
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
	transformedJSON, err := c.labelsTransform.Evaluate(sourceJSON, nil)
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
