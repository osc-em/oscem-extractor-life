package metadataparser

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/osc-em/oscem-extractor-life/internal/configuration"

	"golang.org/x/exp/mmap"
)

// XML PART
// Definitions of the xml structure
type MicroscopeImage struct {
	XMLName    xml.Name   `xml:"MicroscopeImage"`
	Name       string     `xml:"name"`
	UniqueID   string     `xml:"uniqueID"`
	CustomData CustomData `xml:"CustomData"`
}

// for key-value
type CustomData struct {
	KeyValues []KeyValue `xml:"KeyValueOfstringanyType"`
}

type KeyValue struct {
	Key   string `xml:"Key"`
	Value string `xml:"Value"`
}

// For tag-value
type Element struct {
	XMLName  xml.Name
	Content  string    `xml:",chardata"`
	Children []Element `xml:",any"`
}

func parseElement(element Element, path string, leafNodes map[string]string) {
	currentPath := path
	if currentPath != "" {
		currentPath += "." + element.XMLName.Local
	} else {
		currentPath = element.XMLName.Local
	}

	trimmedContent := strings.TrimSpace(element.Content)
	if len(element.Children) == 0 && trimmedContent != "" {
		leafNodes[currentPath] = trimmedContent
	}
	for _, child := range element.Children {
		parseElement(child, currentPath, leafNodes)
	}
}

func process_xml(input string) (map[string]string, error) {
	// just here to catch some error messages would work just fine without
	if strings.Contains(input, "BatchPositionsList") {
		return nil, nil
	}
	reader, err := mmap.Open(input)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	xmlData := make([]byte, reader.Len())
	_, err = reader.ReadAt(xmlData, 0)
	if err != nil {
		return nil, err
	}

	var root Element
	err = xml.Unmarshal(xmlData, &root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error unmarshaling XML:", err)
		return nil, err
	}

	leafNodes := make(map[string]string)
	parseElement(root, "", leafNodes)

	var image MicroscopeImage
	err = xml.Unmarshal(xmlData, &image)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error unmarshaling XML:", err)
		return nil, err
	}
	leafNodes["MicroscopeImage.Name"] = image.Name
	leafNodes["MicroscopeImage.UniqueID"] = image.UniqueID
	for _, kv := range image.CustomData.KeyValues {
		leafNodes[kv.Key] = kv.Value
	}
	return (leafNodes), err
}
func untuple(dict map[string]string, key string, match string) map[string]string {
	xcheck, xexist := dict[key+"_x_max"]
	ycheck, yexist := dict[key+"_y_max"]
	xcheck_min, xmexist := dict[key+"_x_min"]
	ycheck_min, ymexist := dict[key+"_y_min"]
	if !xexist && !yexist && !xmexist && !ymexist {
		dict[key+"_x_max"] = strings.Split(match, " ")[0]
		dict[key+"_y_max"] = strings.Split(match, " ")[1]
		dict[key+"_x_min"] = strings.Split(match, " ")[0]
		dict[key+"_y_min"] = strings.Split(match, " ")[1]
	} else {
		xtest_max, _ := strconv.ParseFloat(strings.TrimSpace(xcheck), 64)
		ytest_max, _ := strconv.ParseFloat(strings.TrimSpace(ycheck), 64)
		xtest_min, _ := strconv.ParseFloat(strings.TrimSpace(xcheck_min), 64)
		ytest_min, _ := strconv.ParseFloat(strings.TrimSpace(ycheck_min), 64)
		x_new, _ := strconv.ParseFloat(strings.TrimSpace(strings.Split(match, " ")[0]), 64)
		y_new, _ := strconv.ParseFloat(strings.TrimSpace(strings.Split(match, " ")[1]), 64)
		dict[key+"_x_max"] = strconv.FormatFloat(max(xtest_max, x_new), 'f', 16, 64)
		dict[key+"_y_max"] = strconv.FormatFloat(max(ytest_max, y_new), 'f', 16, 64)
		dict[key+"_x_min"] = strconv.FormatFloat(min(xtest_min, x_new), 'f', 16, 64)
		dict[key+"_y_min"] = strconv.FormatFloat(min(ytest_min, y_new), 'f', 16, 64)
	}
	return dict
}

// MDOC Part
func process_mdoc(input string) (map[string]string, error) {
	var count float64 = 0.00
	re := regexp.MustCompile(`(.+?)\s*=\s*(.+)`)
	mdocFile, err := os.Open(input)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Your file didnt open", err)
		return nil, err
	}
	defer mdocFile.Close()
	scanner := bufio.NewScanner(mdocFile)
	mdoc_results := make(map[string]string)

	for scanner.Scan() {
		// Look for special case
		//TiltAxis Angle
		tiltaxis := strings.Contains(scanner.Text(), "TiltAxisAngle")    // Tomo 5
		tiltaxis2 := strings.Contains(scanner.Text(), "Tilt axis angle") // SerialEM
		if tiltaxis {
			blabb_split := strings.Split(re.FindStringSubmatch(scanner.Text())[2], "=")[1]
			mdoc_results["TiltAxisAngle"] = (strings.TrimSpace(strings.Split(blabb_split, "  ")[0])) // this is bound to fail at some point if they dont keep their weird double space seperation logic
		}
		if tiltaxis2 {
			blab_split := strings.Split(re.FindStringSubmatch(scanner.Text())[2], ",")[0]
			mdoc_results["TiltAxisAngle"] = (strings.TrimSpace(strings.Split(blab_split, "=")[1]))
		}
		// general search and update for min/max values
		match := re.FindStringSubmatch(scanner.Text())
		//Detect which camera was used -- will only work with SerialEM properties update / script usage
		cam := strings.Contains(scanner.Text(), "CameraIndex")
		if cam {
			if strings.TrimSpace(match[2]) == "0" {
				mdoc_results["CameraUsed"] = mdoc_results["Camera0"]

			} else if strings.TrimSpace(match[2]) == "1" {
				mdoc_results["CameraUsed"] = mdoc_results["Camera1"]
			}
		}

		// Quick check incase the image dimesions are only present in the header
		image := strings.Contains(scanner.Text(), "ImageSize")
		if image {
			mdoc_results["ImageDimensions_X"] = strings.Split(match[2], " ")[0]
			mdoc_results["ImageDimensions_Y"] = strings.Split(match[2], " ")[1]
		}
		if match != nil {
			if strings.TrimSpace(match[1]) == "[ZValue" {
				count++
			}
			value, exists := mdoc_results[match[1]]
			if !exists {
				mdoc_results[match[1]] = match[2]
				// grab the first occurence of a tuple as well
				_, err := strconv.ParseFloat(strings.TrimSpace(match[2]), 64)
				gate := len(strings.Split(match[2], " "))
				if err != nil && gate > 1 {
					beamshift := strings.Contains(scanner.Text(), "Beamshift") // check for correct syntax only present in newer versions of SerialEM
					imageShift := strings.Contains(scanner.Text(), "ImageShift")
					stagepos := strings.Contains(scanner.Text(), "StagePosition")
					if beamshift || imageShift || stagepos {
						mdoc_results = untuple(mdoc_results, match[1], match[2])
					}
					continue
				}
			} else if value == match[2] {
				// Grab some Tuples
				energy := strings.Contains(scanner.Text(), "FilterSlitAndLoss")
				if energy {
					energytest, _ := strconv.ParseFloat(strings.TrimSpace(strings.Split(match[2], " ")[0]), 64)
					if energytest > float64(0.00) {
						mdoc_results["EnergyFilterUsed"] = "true"
						mdoc_results["EnergyFilterSlitWidth"] = strings.Split(match[2], " ")[0]
					}
				}
				continue
			} else if value != match[2] {
				test, err := strconv.ParseFloat(strings.TrimSpace(mdoc_results[match[1]]), 64)
				if err != nil {
					// Grab the remaining Tuples
					beamshift := strings.Contains(scanner.Text(), "Beamshift") // check for correct syntax only present in newer versions of SerialEM
					imageShift := strings.Contains(scanner.Text(), "ImageShift")
					stagepos := strings.Contains(scanner.Text(), "StagePosition")
					if beamshift || imageShift || stagepos {
						mdoc_results = untuple(mdoc_results, match[1], match[2])
					}
					continue
				} else {
					new, _ := strconv.ParseFloat(strings.TrimSpace(match[2]), 64)
					keymin, existmin := mdoc_results[match[1]+"_min"]
					keymax, existmax := mdoc_results[match[1]+"_max"]
					if !existmin {
						mdoc_results[match[1]+"_min"] = strconv.FormatFloat(min(test, new), 'f', 16, 64)
					} else {
						oldmin, _ := strconv.ParseFloat(strings.TrimSpace(keymin), 64)
						mdoc_results[match[1]+"_min"] = strconv.FormatFloat(min(new, oldmin), 'f', 16, 64)
					}
					if !existmax {
						mdoc_results[match[1]+"_max"] = strconv.FormatFloat(max(test, new), 'f', 16, 64)
					} else {
						oldmax, _ := strconv.ParseFloat(strings.TrimSpace(keymax), 64)
						mdoc_results[match[1]+"_max"] = strconv.FormatFloat(max(new, oldmax), 'f', 16, 64)
					}
				}
			}
		}

	}
	// Numberoftilts
	mdoc_results["NumberOfTilts"] = strconv.FormatFloat(count, 'f', 16, 64)

	// get tiltangle at the end if applicable
	_, existtilt := mdoc_results["TiltAngle"]
	if existtilt && count != 0.00 {
		tiltmax, err := strconv.ParseFloat(strings.TrimSpace(mdoc_results["TiltAngle_max"]), 64)
		if err != nil {
			fmt.Println("Tilt angle increment calculation failed")
		}
		tiltmin, err := strconv.ParseFloat(strings.TrimSpace(mdoc_results["TiltAngle_min"]), 64)
		if err != nil {
			fmt.Println("Tilt angle increment calculation failed")
		}
		mdoc_results["Tilt_increment"] = strconv.FormatFloat(math.Abs(tiltmax-tiltmin)/count, 'f', 16, 64)
	}
	// Software used
	T, T_exist := mdoc_results["[T"]
	if T_exist {
		if strings.Contains(T, "TOMOGRAPHY") || strings.Contains(T, "Tomography") {
			mdoc_results["Software"] = "Tomo5"
		} else if strings.Contains(T, "SerialEM") {
			mdoc_results["Software"] = "SerialEM"
		}
	} // generalized before, if SerialEM additions/scripts were used:
	vers, vers_exist := mdoc_results["Version"]
	if vers_exist {
		mdoc_results["Software"] = vers
	}
	// Inference based things come here
	dark, darkexist := mdoc_results["DarkField"]
	if darkexist {
		te, _ := strconv.Atoi(strings.TrimSpace(dark))
		if te == 1 {
			mdoc_results["Imaging"] = "Darkfield"
		}
	}
	mag, magexist := mdoc_results["MagIndex"]
	if magexist {
		te, _ := strconv.Atoi(strings.TrimSpace(mag))
		te2, _ := strconv.Atoi(strings.TrimSpace(dark))
		if te > 0 && (te2 == 0 || !darkexist) {
			mdoc_results["Imaging"] = "Brightfield"
		}
	}
	// Currently missing Illumination modes (EMDB allowed: "Flood Beam", "Spot Scan", "Other") --
	// Problem how to differentiate Spot Scan ; most cryoEM cases definitely Flood Beam
	// Could do "Flood Beam" as baseline and add a catch later; dont know if anyone uses serialEM for spotscan anyways
	EMMode, modeexist := mdoc_results["EMmode"]
	if modeexist {
		te, _ := strconv.Atoi(strings.TrimSpace(EMMode))
		if te == 0 {
			mdoc_results["EMMode"] = "TEM"
		} else if te == 1 {
			mdoc_results["EMMode"] = "EFTEM"
		} else if te == 2 {
			mdoc_results["EMMode"] = "STEM"
		} else if te == 3 {
			mdoc_results["Imaging"] = "Diffraction"
		}
	}
	// Cleanup before return
	for key := range mdoc_results {
		_, upexist := mdoc_results[key+"_max"]
		_, dwnexist := mdoc_results[key+"_min"]
		_, xmxexist := mdoc_results[key+"_x_max"] // remove original tuple form for imageshift and stageshift
		if upexist || dwnexist || xmxexist {
			delete(mdoc_results, key)
		}
	}
	return mdoc_results, err
}

// MERGE and datetimechecks
func merge_to_dataset_level(listofcontents []map[string]string) map[string]string {
	overallmap := make(map[string]string)
	timeformats := []string{
		"02-Jan-06  15:04:05",
		"02-Jan-2006  15:04:05",
		"2006-Jan-02  15:04:05",
		time.RFC3339Nano,
	}
	dose_avg := 0.0
	for item := range listofcontents {
		for key := range listofcontents[item] {
			value, exists := overallmap[key]
			valuenew := (listofcontents[item])[key]
			// get dose average
			if strings.Contains(key, "DoseOnCamera") || strings.Contains(key, "ExposureDose") {
				convtest, err := strconv.ParseFloat(strings.TrimSpace(valuenew), 64)
				if err == nil {
					dose_avg += convtest
				}
			}
			if !exists {
				overallmap[key] = valuenew
			} else if value == valuenew {
				continue
			} else if value != valuenew {
				if strings.Contains(key, "DateTime") {
					for _, datetime := range timeformats {
						timecheck, err1 := time.Parse(datetime, value)
						timechecknew, err := time.Parse(datetime, valuenew)
						if err == nil && err1 == nil {
							_, existstart := overallmap[key+"_start"]
							_, existend := overallmap[key+"_end"]
							if !existstart {
								if timecheck.After(timechecknew) {
									overallmap[key+"_start"] = timechecknew.Format(time.RFC3339)
								} else {
									overallmap[key+"_start"] = timecheck.Format(time.RFC3339)
								}
							} else {
								timecheckold, _ := time.Parse(time.RFC3339, overallmap[key+"_start"])
								if timecheckold.After(timechecknew) {
									overallmap[key+"_start"] = timechecknew.Format(time.RFC3339)
								}
							}
							if !existend {
								if timecheck.Before(timechecknew) {
									overallmap[key+"_end"] = timechecknew.Format(time.RFC3339)
								} else {
									overallmap[key+"_end"] = timecheck.Format(time.RFC3339)
								}
							} else {
								timecheckold, _ := time.Parse(time.RFC3339, overallmap[key+"_end"])
								if timecheckold.Before(timechecknew) {
									overallmap[key+"_end"] = timechecknew.Format(time.RFC3339)
								}
							}
						}
					}
				}
				test, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
				if err == nil {
					new, _ := strconv.ParseFloat(strings.TrimSpace(valuenew), 64)
					keymin, existmin := overallmap[key+"_min"]
					keymax, existmax := overallmap[key+"_max"]
					if !existmin {
						overallmap[key+"_min"] = strconv.FormatFloat(min(test, new), 'f', 16, 64)
					} else {
						oldmin, _ := strconv.ParseFloat(strings.TrimSpace(keymin), 64)
						overallmap[key+"_min"] = strconv.FormatFloat(min(new, oldmin), 'f', 16, 64)
					}
					if !existmax {
						overallmap[key+"_max"] = strconv.FormatFloat(max(test, new), 'f', 16, 64)
					} else {
						oldmax, _ := strconv.ParseFloat(strings.TrimSpace(keymax), 64)
						overallmap[key+"_max"] = strconv.FormatFloat(max(new, oldmax), 'f', 16, 64)
					}
				}
			}
		}
	}
	for key := range overallmap {
		_, upexist := overallmap[key+"_max"]
		_, dwnexist := overallmap[key+"_min"]
		_, startexist := overallmap[key+"_start"]
		_, endexist := overallmap[key+"_end"]
		if upexist || dwnexist || startexist || endexist {
			delete(overallmap, key)
		}
	}
	overallmap["NumberOfMovies"] = strconv.Itoa(len(listofcontents))
	overallmap["DoseAverage"] = strconv.FormatFloat(dose_avg/float64(len(listofcontents)), 'f', 16, 64)
	return overallmap
}

type xmlResult struct {
	filePath string
	content  map[string]string
}

type mdocResult struct {
	content map[string]string
}

func readin(jobs <-chan string, results chan<- interface{}, wg *sync.WaitGroup, progresstracker *ProgressTracker) {
	defer wg.Done()
	for filePath := range jobs {
		switch filepath.Ext(filePath) {
		case ".xml":
			xmlContent, err := process_xml(filePath)
			if err == nil {
				results <- xmlResult{filePath: filePath, content: xmlContent}
			} else {
				fmt.Fprintln(os.Stderr, "Import of", filePath, "failed")
			}
		case ".mdoc":
			mdocContent, err := process_mdoc(filePath)
			if err == nil {
				results <- mdocResult{content: mdocContent}
			} else {
				fmt.Fprintln(os.Stderr, "Import of", filePath, "failed")
			}
		default:
			fmt.Fprintf(os.Stderr, "Unknown file type: %s\n", filePath)
		}
		atomic.AddInt64(&progresstracker.Completed, 1)
	}
}

func zipFiles(files []string) error {
	archive, err := os.Create("xmls.zip")
	if err != nil {
		return err
	}
	defer archive.Close()
	writer := zip.NewWriter(archive)
	defer writer.Close()
	for _, file := range files {
		err = addFileToZip(writer, file)
		if err != nil {
			return err
		}
	}
	return nil
}

func addFileToZip(writer *zip.Writer, file string) error {
	op, err := os.Open(file)
	if err != nil {
		return err
	}
	defer op.Close()
	test := strings.Split(file, string(filepath.Separator))
	name := test[len(test)-1]
	wr, err := writer.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(wr, op)
	if err != nil {
		return err
	}
	return nil
}

func findDataFolders(inputDir string, dataFolders []string, folderFlag string) ([]string, error) {

	foldersRegex := "Data|Batch"
	if folderFlag != "" {
		foldersRegex = foldersRegex + "|" + folderFlag
	}
	foldersRegexCompiled, _ := regexp.Compile(foldersRegex)
	err := filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}

		if foldersRegexCompiled.MatchString(info.Name()) {
			dataFolders = append(dataFolders, path)
		}

		return nil
	})

	return dataFolders, err
}

// minicheck against hidden files
func isHidden(name string) bool {
	return len(name) > 0 && name[0] == '.'
}

type ProgressTracker struct {
	Total     int64
	Completed int64
}

func startProgressReporter(progressTracker *ProgressTracker) {
	for {
		completed := atomic.LoadInt64(&progressTracker.Completed)
		total := atomic.LoadInt64(&progressTracker.Total)
		progress := float64(completed) / float64(total) * 100
		fmt.Printf("\rProgress: %.2f%%", progress)
		if completed >= total {
			fmt.Println("")
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func collectAllFiles(directories []string) ([]string, error) {
	var allFiles []string
	for _, dir := range directories {
		files, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			if !file.IsDir() && !isHidden(file.Name()) && (filepath.Ext(file.Name()) == ".xml" || filepath.Ext(file.Name()) == ".mdoc") {
				allFiles = append(allFiles, filepath.Join(dir, file.Name()))
			}
		}
	}
	return allFiles, nil
}

func ReadMetadata(topLevelDirectory string, create_zip bool, write_full_metadata bool, epu_folder string, metadataFolderRegex string) ([]byte, error) {

	// Check if the provided directory exists
	fileInfo, err := os.Stat(topLevelDirectory)
	if os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: Directory '%s' does not exist.\n", topLevelDirectory)
		return nil, err
	}

	// Check if the provided path is a directory
	if !fileInfo.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: '%s' is not a directory.\n", topLevelDirectory)
		return nil, err
	}
	// this part is to make sure there is no confusion on the instrument computer search when running on the Athena server folder with "./"
	directory_safe, _ := filepath.Abs(topLevelDirectory)
	correct := strings.Split(directory_safe, string(filepath.Separator))
	target := correct[len(correct)-1]
	var dataFolders []string
	dataFolders, err = findDataFolders(topLevelDirectory, dataFolders, metadataFolderRegex)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Folder search failed - is this the correct directory?", err)
		return nil, err
	}
	var parallel string
	if epu_folder == "" {
		var getmpc map[string]string
		config, err := configuration.Getconfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Warning: No path config available, we suggest using either --epu or the config to provide the path where EPU mirrors the datasets and stores xmls")
		} else {
			errun := json.Unmarshal(config, &getmpc)
			if errun != nil {
				fmt.Fprintln(os.Stderr, "Your config was unretrievable, make sure it is set and accessible or use the param flags")
			}
			parallel = getmpc["MPCPATH"]
		}
	} else {
		parallel = epu_folder
	}

	if parallel != "" {
		dataFolders, err = findDataFolders(parallel+target, dataFolders, metadataFolderRegex)
		if err != nil {
			fmt.Fprintln(os.Stderr, "There should be a folder on your instrument control computer with the same name - something went wrong here", err)
			return nil, err
		}
	}
	dataFolders = append(dataFolders, topLevelDirectory)
	progress := &ProgressTracker{}

	allfiles, err := collectAllFiles(dataFolders)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not collect files:", dataFolders, err)
	}
	progress.Total = int64(len(allfiles))
	fmt.Printf("Total number of files to process: %d\n", progress.Total)
	go startProgressReporter(progress)

	jobs := make(chan string, len(allfiles))
	for _, filePath := range allfiles {
		jobs <- filePath
	}
	close(jobs)

	var mdoc_files []map[string]string
	var xml_files []map[string]string
	var listxml []string

	var wg sync.WaitGroup
	numWorkers := 16
	results := make(chan interface{}, len(allfiles))
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go readin(jobs, results, &wg, progress)
	}

	wg.Wait()
	close(results)
	for result := range results {
		switch res := result.(type) {
		case xmlResult:
			xml_files = append(xml_files, res.content)
			listxml = append(listxml, res.filePath)
		case mdocResult:
			mdoc_files = append(mdoc_files, res.content)
		}
	}
	// whether to generate zip of xmls
	if create_zip && listxml != nil {
		zipFiles(listxml)
	}

	var out map[string]string
	if mdoc_files != nil && xml_files == nil {
		out = merge_to_dataset_level(mdoc_files)
	} else if xml_files != nil && mdoc_files == nil {
		out = merge_to_dataset_level(xml_files)
	} else if xml_files != nil && mdoc_files != nil {
		out = merge_to_dataset_level(xml_files)
		b := merge_to_dataset_level(mdoc_files)
		for x, y := range b {
			out[x] = y
		}
	} else {
		fmt.Println("Something went wrong, nothing was read out")
		return nil, err
	}

	jsonData, err := json.MarshalIndent(out, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error marshaling to JSON:", err)
		return nil, err
	}
	var nameout string
	if len(correct) > 0 {
		nameout = target + "_full.json"
	} else {
		fmt.Fprintln(os.Stderr, "Name generation failed, returning to default")
		nameout = "Dataset_out.json"
	}
	if write_full_metadata {
		err = os.WriteFile(nameout, jsonData, 0644)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error writing JSON to file:", err)
		}
		fmt.Println("Extracted full data has been written to ", nameout)
	}
	return jsonData, nil
}
