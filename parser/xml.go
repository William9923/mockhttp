package parser

import (
	"github.com/clbanning/mxj"
)

func ParseXML(xmlText string) (map[string]interface{}, error) {
	data, err := mxj.NewMapXml([]byte(xmlText))
	if err != nil {
		return nil, err
	}
	return data.Old(), nil
}
