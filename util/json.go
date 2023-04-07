package util

import (
	"log"

	"github.com/goccy/go-json"
)

type JsonNode struct {
	data     interface{}
	IsNumber bool
	IsString bool
	IsBool   bool
	IsArray  bool
	IsObject bool
	IsNull   bool
}

func NewJsonNode(input []byte) (*JsonNode, error) {
	var data interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		return nil, err
	}

	return getJsonNode(data), nil
}

func (n *JsonNode) Number() float64 {
	if n.IsNumber {
		return n.data.(float64)
	} else {
		log.Printf("call to Number() for non-number")
		return 0
	}
}

func (n *JsonNode) String() string {
	if n.IsString {
		return n.data.(string)
	} else {
		log.Printf("call to String() for non-string")
		return ""
	}
}

func (n *JsonNode) Bool() bool {
	if n.IsBool {
		return n.data.(bool)
	} else {
		log.Printf("call to Bool() for non-bool")
		return false
	}
}

func (n *JsonNode) Array() []*JsonNode {
	if n.IsArray {
		arr := n.data.([]interface{})
		nodes := make([]*JsonNode, len(arr))

		for idx, data := range arr {
			nodes[idx] = getJsonNode(data)
		}

		return nodes
	} else {
		log.Printf("call to Array() for non-array")
		return nil
	}
}

func (n *JsonNode) Get(key string) *JsonNode {
	if n.IsObject {
		obj := n.data.(map[string]interface{})
		return getJsonNode(obj[key])
	} else {
		log.Printf("cannot call Get() on non-object")
		return nil
	}
}

func (n *JsonNode) Index(index int) *JsonNode {
	if n.IsArray {
		arr := n.data.([]interface{})
		return getJsonNode(arr[index])
	} else {
		log.Printf("cannot call Index() on non-array")
		return nil
	}
}

func getJsonNode(data interface{}) *JsonNode {
	node := &JsonNode{data: data}
	switch data.(type) {
	case bool:
		node.IsBool = true
	case float64:
		node.IsNumber = true
	case string:
		node.IsString = true
	case []interface{}:
		node.IsArray = true
	case map[string]interface{}:
		node.IsObject = true
	default:
		node.IsNull = true
	}

	return node
}
