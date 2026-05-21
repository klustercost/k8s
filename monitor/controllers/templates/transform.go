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
	jsonataInstance jsonata.JSONataInstance
	logger          klog.Logger
	objectTransform jsonata.Expression
	factTransforms  []factColumn
}

func NewTransform(ctx context.Context, path string) *Transform {
	logger := klog.FromContext(ctx)

	instance, err := jsonata.OpenLatest()
	if err != nil {
		logger.Error(err, "Unable to create JSONata instance")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	var factTransforms []factColumn

	factTransforms, err = NewFactsFromFile(path+"fact.json", instance)
	if err != nil {
		logger.Error(err, "Unable to read fact transforms")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	transform, err := getTransform(path+"object.jsonata", instance)
	if err != nil {
		logger.Error(err, "Unable to read template transform")
		klog.FlushAndExit(klog.ExitFlushTimeout, 1)
	}

	return &Transform{
		jsonataInstance: instance,
		logger:          logger,
		objectTransform: transform,
		factTransforms:  factTransforms,
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

func (c *Transform) addFacts(ctx context.Context, object map[string]interface{}) (map[string]interface{}, error) {
	for _, transform := range c.factTransforms {
		object, err := transform.AddFact(ctx, object)
		if err != nil {
			return object, err
		}
	}
	return object, nil
}

func (c *Transform) Transform(ctx context.Context, source any) ([]byte, error) {
	data, err := json.Marshal(source)
	if err != nil {
		c.logger.Error(err, "Unable to marshal source to JSON")
		return nil, err
	}

	object, err := c.objectTransform.Evaluate(data, nil)
	if err != nil {
		c.logger.Error(err, "Unable to evaluate template")
		return nil, err
	}
	var keyVal model.DataExchange
	err = json.Unmarshal(object, &keyVal)
	if err != nil {
		c.logger.Error(err, "Unable to unmarshal fact transforms")
		return nil, err
	}
	keyVal, err = c.addFacts(ctx, keyVal)
	if err != nil {
		c.logger.Error(err, "Unable to add facts to object")
		return nil, err
	}
	return json.Marshal(keyVal)
}
