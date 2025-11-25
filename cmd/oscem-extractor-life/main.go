package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/osc-em/oscem-extractor-life/internal/configuration"
	"github.com/osc-em/oscem-extractor-life/internal/metadataparser"

	conversion "github.com/osc-em/oscem-converter-extracted"
)

func main() {
	//for benchmarking
	/*f, err := os.Create("trace.out")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := trace.Start(f); err != nil {
		panic(err)
	}
	defer trace.Stop()*/

	create_zip := flag.Bool("z", false, "Toggle whether to make a zip archive of all xml files - default: false")
	write_full_metadata := flag.Bool("f", false, "Toggle whether the full metadata is also written out in addition to the OSCEM schema conform one- default: false")
	reset_config_file := flag.Bool("c", false, "If you want to reset your config file")
	output_file_path := flag.String("o", "", "Provide target output path and name for your metadata file, leave empty to write to current working directory")
	input_folder_path := flag.String("i", "", "Provide target input folder - will take first positional argument if --i is missing")
	cs_value := flag.String("cs", "", "Provide CS value here, if you dont want to use configs")
	gain_flip_rotate := flag.String("gain_flip_rotate", "", "Provide whether and how to flip the gain ref here, if you dont want to use configs")
	epu_folder := flag.String("epu", "", "Provide the path to the mirrored EPU folder containing all the xmls of the datacollections here, if you dont want to use configs")
	metadataFolder := flag.String("folder_filter", "", "If the system deviates from standard EPU naming conventions, a regex for the folder name with the metadata files can be provided.")
	print_to_stdout := flag.Bool("cli_out", false, "If you want the results also as a stdout")
	flag.Parse()
	posArgs := flag.Args()

	// allow for reconfiguration of the config
	if *reset_config_file {
		current, err := configuration.Getconfig()
		var grid map[string]string
		if err != nil {
			fmt.Fprintln(os.Stderr, " No prior config obtainable", err)
		}
		_ = json.Unmarshal(current, &grid)
		fmt.Println("current config:\n", grid)
		configuration.Changeconfig()
		return
	}
	var directory string
	// Check that there are arguments
	if len(posArgs) == 0 && *input_folder_path == "" {
		fmt.Println("No arguments; correct minimum arguments: ./oscem-extractor-life <directory>")
		return
	} else if *input_folder_path != "" {
		directory = *input_folder_path
	} else {
		directory = posArgs[0]
	}

	current, err := configuration.Getconfig()
	var grid map[string]string
	if err == nil && *cs_value == "" && *gain_flip_rotate == "" && *epu_folder == "" {
		_ = json.Unmarshal(current, &grid)
		*cs_value = grid["cs"]
		*gain_flip_rotate = grid["gainref_flip_rotate"]
		*epu_folder = grid["MPCPATH"]
	}

	data, err := metadataparser.ReadMetadata(directory, *create_zip, *write_full_metadata, *epu_folder, *metadataFolder)
	if err != nil {
		fmt.Fprintln(os.Stderr, "The extraction went wrong due to", err)
		os.Exit(1)
	}
	out, err1 := conversion.Convert(data, "", *cs_value, *gain_flip_rotate, *output_file_path)
	if err1 != nil {
		fmt.Fprintln(os.Stderr, "The extraction went wrong due to", err1)
		os.Exit(1)
	}
	if *print_to_stdout {
		fmt.Printf("%s", string(out))
	}
}
