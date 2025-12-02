package metadataparser

import (
	"encoding/json"
	"os"
	"testing"

	conversion "github.com/osc-em/oscem-converter-extracted"

	"github.com/stretchr/testify/assert"
)

func TestReaderTableDriven(t *testing.T) {
	readJSONFile := func(filepath string) string {
		data, err := os.ReadFile(filepath)
		if err != nil {
			t.Fatalf("Failed to read expected data file %s: %v", filepath, err)
		}
		return string(data)
	}
	testsFolder := "../../tests"
	//reader
	targetXML := readJSONFile(testsFolder + "/xml_full.json")
	targetMdoc := readJSONFile(testsFolder + "/mdocs_full.json")
	targetCombine := readJSONFile(testsFolder + "/combine_full.json")
	targetmdocspa := readJSONFile(testsFolder + "/mdocspa_full.json")
	targetdepth := readJSONFile(testsFolder + "/depthcheck_full.json")
	//converter
	target2XML := readJSONFile(testsFolder + "/xml_correct.json")
	target2Mdoc := readJSONFile(testsFolder + "/mdocs_correct.json")
	target2Combine := readJSONFile(testsFolder + "/combine_correct.json")
	target2mdocspa := readJSONFile(testsFolder + "/mdocspa_correct.json")
	target2depth := readJSONFile(testsFolder + "/depthcheck_correct.json")

	tests := []struct {
		name                string
		directory           string
		create_zip          bool
		write_full_metadata bool
		wantData            string // reader only
		wantErr             bool
		wantData2           string // e2e
		cs_value            string
		gain_flip_rotate    string
		epu_folder          string
		metadataFolder      string
		print_to_stdout     bool
	}{
		{
			name:                "xmls",
			directory:           testsFolder + "/xml",
			create_zip:          false,
			write_full_metadata: false,
			wantData:            targetXML,
			wantErr:             false,
			wantData2:           target2XML,
			cs_value:            "2.7",
			gain_flip_rotate:    "none",
			epu_folder:          "",
			metadataFolder:      "",
			print_to_stdout:     false,
		},
		{
			name:                "mdocs",
			directory:           testsFolder + "/mdocs",
			create_zip:          false,
			write_full_metadata: false,
			wantData:            targetMdoc,
			wantErr:             false,
			wantData2:           target2Mdoc,
			cs_value:            "2.7",
			gain_flip_rotate:    "none",
			epu_folder:          "",
			metadataFolder:      "",
			print_to_stdout:     false,
		},
		{
			name:                "Both",
			directory:           testsFolder + "/combine",
			create_zip:          false,
			write_full_metadata: false,
			wantData:            targetCombine,
			wantErr:             false,
			wantData2:           target2Combine,
			cs_value:            "2.7",
			gain_flip_rotate:    "none",
			epu_folder:          "",
			metadataFolder:      "",
			print_to_stdout:     false,
		},
		{
			name:                "mdocspa",
			directory:           testsFolder + "/mdocspa",
			create_zip:          false,
			write_full_metadata: false,
			wantData:            targetmdocspa,
			wantErr:             false,
			wantData2:           target2mdocspa,
			cs_value:            "2.7",
			gain_flip_rotate:    "none",
			epu_folder:          "",
			metadataFolder:      "",
			print_to_stdout:     false,
		},
		{
			name:                "depthcheck",
			directory:           testsFolder + "/depthcheck",
			create_zip:          false,
			write_full_metadata: false,
			wantData:            targetdepth,
			wantErr:             false,
			wantData2:           target2depth,
			cs_value:            "2.7",
			gain_flip_rotate:    "none",
			epu_folder:          "",
			metadataFolder:      "",
			print_to_stdout:     false,
		},
		{
			name:                "metadataFolder",
			directory:           testsFolder + "/empty",
			create_zip:          false,
			write_full_metadata: false,
			wantData:            targetXML,
			wantErr:             false,
			wantData2:           target2XML,
			cs_value:            "2.7",
			gain_flip_rotate:    "none",
			epu_folder:          "",
			metadataFolder:      "myfoldername",
			print_to_stdout:     false,
		},
		{
			name:                "metadataFolderRegex",
			directory:           testsFolder + "/empty",
			create_zip:          false,
			write_full_metadata: false,
			wantData:            targetXML,
			wantErr:             false,
			wantData2:           target2XML,
			cs_value:            "2.7",
			gain_flip_rotate:    "none",
			epu_folder:          "",
			metadataFolder:      "^m.+foldername$",
			print_to_stdout:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := ReadMetadata(tt.directory, tt.create_zip, tt.write_full_metadata, tt.epu_folder, tt.metadataFolder)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Reader() error = %v, wantErr %v", err, tt.wantErr)
			}
			// rerun the json marshalling to ensure no issues with whitespaces etc
			var jsonData map[string]string
			if err := json.Unmarshal(data, &jsonData); err != nil {
				t.Fatalf("Failed to unmarshal returned data: %v", err)
			}
			var jsonDataclean map[string]string
			if err := json.Unmarshal([]byte(tt.wantData), &jsonDataclean); err != nil {
				t.Fatalf("Failed to unmarshal returned data: %v", err)
			}
			// exclude irrelevant keys:
			excludeKeys := []string{"MicroscopeImage.UniqueID", "MicroscopeImage.uniqueID", "MicroscopeImage.microscopeData.core.Guid", "ImageFile", "MinMaxMean", "SubFramePath", "[T"}
			cleaned_json := preprocessMap(jsonData, excludeKeys)
			cleaned_target := preprocessMap(jsonDataclean, excludeKeys)
			actualDataBytes, err := json.Marshal(cleaned_json)
			if err != nil {
				t.Fatalf("Failed to re-marshal returned data: %v", err)
			}
			targetDataBytes, err := json.Marshal(cleaned_target)
			if err != nil {
				t.Fatalf("Failed to re-marshal returned data: %v", err)
			}

			assert.JSONEqf(t, string(targetDataBytes), string(actualDataBytes), "Mismatch in test case %s", tt.name)

			outputFilePath := os.TempDir() + "/" + tt.name

			data2, err2 := conversion.Convert(data, "", tt.cs_value, tt.gain_flip_rotate, outputFilePath)

			if (err2 != nil) != tt.wantErr {
				t.Fatalf("Reader() error = %v, wantErr %v", err2, tt.wantErr)
			}
			var jsonData2 interface{}
			if err := json.Unmarshal(data2, &jsonData2); err != nil {
				t.Fatalf("Failed to unmarshal returned data: %v", err)
			}

			actualDataBytes2, err := json.Marshal(jsonData2)
			if err != nil {
				t.Fatalf("Failed to re-marshal returned data: %v", err)
			}

			assert.JSONEqf(t, tt.wantData2, string(actualDataBytes2), "Mismatch in test case %s", tt.name)
		})
	}
}
func filterMap(input map[string]string, excludeKeys []string) map[string]string {
	result := make(map[string]string)
	excludeSet := make(map[string]struct{}, len(excludeKeys))
	for _, key := range excludeKeys {
		excludeSet[key] = struct{}{}
	}
	for k, v := range input {
		if _, found := excludeSet[k]; !found {
			result[k] = v
		}
	}
	return result
}
func preprocessMap(input map[string]string, excludeKeys []string) map[string]string {
	return filterMap(input, excludeKeys)
}
