package main

import (
	"encoding/json"
	"fmt"

	"github.com/davecgh/go-spew/spew"
)

type TerraformOutput struct {
	// Name      string `json:"name" db:"name"`
	Sensitive bool   `json:"sensitive" db:"sensitive"`
	Type      string `json:"type" db:"type"`
	Value     string `json:"value" db:"value"`
}

func main() {
	raw_outputs := "{ \"bar\":{ \"sensitive\":false, \"type\":\"string\", \"value\":\"foo\" }, \"foo\":{ \"sensitive\":true, \"type\":\"string\", \"value\":\"bar\" } }"

	var data map[string]TerraformOutput

	err := json.Unmarshal([]byte(raw_outputs), &data)
	if err != nil {
		fmt.Println(err)
	}

	spew.Dump(data)

}
