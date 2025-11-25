package configuration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func Getconfig() ([]byte, error) {

	path, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	destination := filepath.Join("oscem-extractor-life", "oscem-extractor-life.conf")
	configFilePath := filepath.Join(path, destination)

	_, err1 := os.Stat(configFilePath)
	if err1 != nil {
		return nil, err1
	}
	content, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func Changeconfig() {

	path, err := os.UserConfigDir()
	if err != nil {
		fmt.Println("Couldn't reach config directory", err)
	}
	destination := filepath.Join("oscem-extractor-life", "oscem-extractor-life.conf")
	configFilePath := filepath.Join(path, destination)
	configdir := filepath.Join(path, "oscem-extractor-life")
	_, err1 := os.Stat(configdir)
	if err1 != nil {
		if os.IsNotExist(err1) {
			os.Mkdir(configdir, 0755)
		} else {
			fmt.Printf("Error accessing %q: %v\n", configdir, err1)
		}
	}
	fmt.Println("What is your instruments spherical aberration (CS)?")
	reader := bufio.NewReader(os.Stdin)
	input1, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading input:", err)
		return
	}
	fmt.Println("And what is the rotation or flipping that needs to be done when importing the gain reference to e.g cryosparc?")
	reader2 := bufio.NewReader(os.Stdin)
	input2, err := reader2.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading input:", err)
		return
	}
	fmt.Println("If available what is the path where EPU mirrors your data output and dumps its metadata .xmls (typically this is on the microscope computer). If you dont know/ cant reach that folder leave this empty. For optimal usage of this tool this is however required.")
	reader3 := bufio.NewReader(os.Stdin)
	input3, err := reader3.ReadString('\n')
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading input:", err)
		return
	}
	configmap := make(map[string]string)
	configmap["CS"] = strings.TrimSpace(input1)
	configmap["Gainref_FlipRotate"] = strings.TrimSpace(input2)
	configmap["MPCPATH"] = strings.TrimSpace(input3)
	config, err := json.MarshalIndent(configmap, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error generating config:", err)
		return
	}

	err = os.WriteFile(configFilePath, config, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error generating config:", err)
	}
	fmt.Println("Generated config at", configFilePath)
}
