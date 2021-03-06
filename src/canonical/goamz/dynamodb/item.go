package dynamodb

import simplejson "github.com/bitly/go-simplejson"
import (
	"errors"
	"fmt"
)

func (t *Table) GetItem(hashKey string, rangeKey string) (map[string]*Attribute, error) {
	q := NewQuery(t)
	q.AddKey(t, hashKey, rangeKey)

	jsonResponse, err := t.Server.queryServer(target("GetItem"), q)

	if err != nil {
		return nil, err
	}

	json, err := simplejson.NewJson(jsonResponse)

	if err != nil {
		return nil, err
	}

	item, err := json.Get("Item").Map()

	if err != nil {
		message := fmt.Sprintf("Unexpected response %s", jsonResponse)
		return nil, errors.New(message)
	}

	return parseAttributes(item), nil

}

func (t *Table) PutItem(hashKey string, rangeKey string, attributes []Attribute) (bool, error) {

	if len(attributes) == 0 {
		return false, errors.New("At least one attribute is required.")
	}

	q := NewQuery(t)
	
	keys := t.Key.Clone(hashKey, rangeKey)
	attributes = append(attributes, keys...)

	q.AddItem(attributes)

	jsonResponse, err := t.Server.queryServer(target("PutItem"), q)

	if err != nil {
		return false, err
	}

	json, err := simplejson.NewJson(jsonResponse)

	if err != nil {
		return false, err
	}

	units := json.Get("ConsumedCapacityUnits")

	if units == nil {
		message := fmt.Sprintf("Unexpected response %s", jsonResponse)
		return false, errors.New(message)
	}

	return true, nil
}

func (t *Table) AddItem(hashKey string, rangeKey string, attributes []Attribute) (bool, error) {

	if len(attributes) == 0 {
		return false, errors.New("At least one attribute is required.")
	}

	q := NewQuery(t)
	q.AddKey(t, hashKey, rangeKey)
	q.AddUpdates(attributes, "ADD")

	jsonResponse, err := t.Server.queryServer(target("UpdateItem"), q)

	if err != nil {
		return false, err
	}

	json, err := simplejson.NewJson(jsonResponse)

	if err != nil {
		return false, err
	}

	units := json.Get("ConsumedCapacityUnits")

	if units == nil {
		message := fmt.Sprintf("Unexpected response %s", jsonResponse)
		return false, errors.New(message)
	}

	return true, nil
	
}

func parseAttributes(s map[string]interface{}) map[string]*Attribute {
	results := map[string]*Attribute{}

	for key, value := range s {
		if v, ok := value.(map[string]interface{}); ok {
			if val, ok := v[TYPE_STRING].(string); ok {
				attr := &Attribute{
					TYPE_STRING,
					key,
					val,
				}
				results[key] = attr
			}
		} else {
			fmt.Printf("type assertion to map[string] interface{} failed for : %s\n ", value)
		}

	}

	return results
}
