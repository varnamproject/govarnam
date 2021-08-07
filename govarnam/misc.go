package govarnam

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

// GetAllSchemePaths get available IDs' location as a string array
func GetAllSchemePaths() ([]string, error) {
	vstsDir, err := findVSTDir()

	if err != nil {
		return nil, err
	}

	var schemeIDs []string

	filepath.WalkDir(vstsDir, func(s string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if filepath.Ext(d.Name()) == ".vst" {
			schemeIDs = append(schemeIDs, s)
		}
		return nil
	})

	return schemeIDs, nil
}

// GetAllSchemeDetails get information of all schemes available
func GetAllSchemeDetails() ([]SchemeDetails, error) {
	schemePaths, err := GetAllSchemePaths()

	if err != nil {
		return nil, err
	}

	var schemeDetails []SchemeDetails

	for _, vstPath := range schemePaths {
		fmt.Println(vstPath)

		varnam := Varnam{}
		varnam.InitVST(vstPath)

		schemeDetails = append(schemeDetails, varnam.SchemeDetails)

		varnam.Close()
	}

	return schemeDetails, nil
}
