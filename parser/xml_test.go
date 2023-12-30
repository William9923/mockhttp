package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const xmlStr = `
        <?xml version="1.0" encoding="UTF-8"?>
        <bookstore>
            <book category="cooking">
                <title lang="en">Everyday Italian</title>
                <author>Giada De Laurentiis</author>
                <year>2005</year>
                <price>30.00</price>
            </book>
            <book category="children">
                <title lang="en">Harry Potter</title>
                <author>J.K. Rowling</author>
                <year>2005</year>
                <price>29.99</price>
            </book>
        </bookstore>
    `

const invalidXmlStr = `
        <?xml version="1.0" encoding="UTF-8"?>
        <bookstore>
            <book category="cooking">
                <title lang="en">Everyday Italian</title>
                <author>Giada De Laurentiis</author>
                <year>2005</year>
                <price>30.00</price>
            </book>
            <book category="children">
        </bookstore>
    `

func Test_ParseXML(t *testing.T) {
	t.Run("parse correct xml", func(t *testing.T) {
		res, err := ParseXML(xmlStr)
		expected := map[string]interface{}{
			"bookstore": map[string]interface{}{
				"book": []interface{}{
					map[string]interface{}{
						"-category": "cooking",
						"author":    "Giada De Laurentiis",
						"price":     "30.00",
						"title": map[string]interface{}{
							"#text": "Everyday Italian",
							"-lang": "en",
						},
						"year": "2005",
					},
					map[string]interface{}{
						"-category": "children",
						"author":    "J.K. Rowling",
						"price":     "29.99",
						"title": map[string]interface{}{
							"#text": "Harry Potter",
							"-lang": "en",
						},
						"year": "2005",
					},
				},
			},
		}

		assert.Nil(t, err, "should not error")
		assert.Equal(t, expected, res)
	})

	t.Run("parse invalid json", func(t *testing.T) {
		_, err := ParseXML(invalidXmlStr)
		assert.NotNil(t, err, "should err")
	})
}
