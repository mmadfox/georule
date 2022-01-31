package geojson

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/geojson"

	"github.com/uber/h3-go"
)

func TestIndex(t *testing.T) {
	testCases := []struct {
		filename string
		cells    int
		level    int
	}{
		{
			filename: "feature_collection_1",
			cells:    44,
			level:    6,
		},
		{
			filename: "feature_collection_2",
			cells:    2,
			level:    6,
		},
		{
			filename: "feature_collection_3",
			cells:    11,
			level:    2,
		},
	}
	for _, tc := range testCases {
		data, err := loadData("./testdata/" + tc.filename + ".json")
		assert.NoError(t, err)
		object, err := geojson.Parse(data, geojson.DefaultParseOptions)
		assert.NoError(t, err)
		cells := EnsureIndex(object, tc.level)
		assert.Equal(t, tc.cells, len(cells))

		if !testing.Short() {
			fmt.Printf("dataset: %s\n", tc.filename)
			printCells(cells)
		}
	}
}

func loadData(filename string) (string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func printCells(cells []h3.H3Index) {
	for i := 0; i < len(cells); i++ {
		b := h3.ToGeoBoundary(cells[i])
		for _, b := range b {
			fmt.Println(b.Longitude, ",", b.Latitude)
		}
	}
}
