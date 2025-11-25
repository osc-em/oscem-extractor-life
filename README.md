# Life Sciences Metadata extractor

Extract metadata from common life-science electron microscopy data in
[OSC-EM](https://github.com/osc-em) format.

## Input formats

- SerialEM
- Thermo Fisher EPU
- TOMO5

## Installation

Binaries for Mac, Linux, and Windows can be downloaded from our
[releases](https://github.com/osc-em/oscem-extractor-life/releases) page.

You can run with docker using the latest image like:

```sh
docker run --rm -v $PWD:/data -w /data oscem-life:latest -o out.json tutorial
```

Alternately, you can compile from source by running:

```sh
cd cmd/oscem-extractor-life
go build -o oscem-extractor-life .
```

### MacOS

The release executables for MacOS are not signed. You may get a warning that MacOS
cannot verify the developer or check the binary for malicious software. If downloaded
directly from Github this executable should be safe to run. You can bypass the warning
by running the command:

```sh
xattr -d com.apple.quarantine oscem-extractor-life
```

### SerialEM

**!!! Requires SerialEM 4.2.0 or newer !!!**

SerialEM requires some additional configuration to ensure that all required information
is available in the mdoc files.

1. Add instrument properties to `SerialEMproperties.txt`. See the
   [example](SerialEM_Scripts/SerialEMproperties_GlobalAutodocEntry_Example.txt). Update
   values to reflect your instrument parameters.
2. The two scripts are provided in `SerialEM_Scripts/` for SPA and Tomography datasets.
   One of these should be run after each image collection (the lowest tick mark on the
   SerialEM automization script selection). Otherwise SerialEM output will lack a few
   required fields for the schema.

### EPU and TOMO5

Some instrument data is not available in EPU output. This is normally set in a
configuration file, but can also be added at the command line using parameters.

A wizard is available to walk through creating the configuration file. Run it using

```sh
oscem-extractor-life --c
```

The configuration file is saved in the following locations depending on your platform:

- Unix: `$XDG_CONFIG_HOME/oscem-extractor-life/oscem-extractor-life.conf` (usually
  `$HOME/.config/oscem-extractor-life/oscem-extractor-life.conf`)
- MacOS: `$HOME/Library/Application Support/oscem-extractor-life/oscem-extractor-life.conf`
- Windows: `%AppData%\oscem-extractor-life\oscem-extractor-life.conf`

Config values can also be set using the command line flags:

| Config property    | CLI Option           | Required | Description                                                   |
| ------------------ | -------------------- | -------- | ------------------------------------------------------------- |
| CS                 | `--cs`               | yes      | the CS value of the instrument                                |
| Gainref_FlipRotate | `--gain_flip_rotate` | yes      | the orientation of the gain_reference relative to actual data |
| MPCPATH            | `--epu`              |          | Path to EPU metadata directory                                |

EPU writes its metadata files in a different directory than its actual data (TOMO5 also
keeps some additional info that is processed by the oscem-extractor-life there). It
generates another set of folders, usually on the microscope controlling computer, that
mirror its OffloadData folders in directory structure. Within them it stores some
related information, including the metadata xml files. If `--epu` is defined as a flag
or in the config, the oscem-extractor-life will directly grab those when the user points
it at a OffloadData directory.
*NOTE: This requires you to mount the microscope computer directory for EPU on the
machine you are running oscem-extractor-life on, as those are most likely NOT the same.
The extractor will work regardless if pointed to the xmls/mdocs directly, this is just
for convenience.*

### Suggestions

Use the associated OpenEM [tool](https://github.com/SwissOpenEM/epu_dataset_merger)
which will allow you to automate moving all the required metadata (and some more useful
data thats in the EPUs internal folders) into your datasets. Can be automated via e.g.
crontab.

## Usage

The reader should be called with the path to a folder containing the xml (EPU/TOMO5) or
mdoc (SerialEM) files.

```sh
./oscem-extractor-life -o tutorial_oscem.json tutorial/
```

For testing, try the associated [tutorial](tutorial/) folder; an example of how the
output should look like is provided in the same folder (tutorial_correct.json). For
first time use, disregard the warnings about config/flags those are for use directly
with EPU or the OpenEM Ingestor.

The reader runs on a directory containing the microscope's additional information files
for each micrograph (.mdoc or .xml for SerialEM and EPU, respectively). It generates a
JSON file following the OSC-EM schema with metadata for the whole dataset. For
usage with EPU, pointing to the top level directory is enough; it will search for the
data folders and extract the info from there.

Using `-z` you can also obtain a zip file of the xml files associated with your data
collection. This can be useful for archiving or for later analysis.

To include additional metadata not supported by the OSC-EM schema, use the `-f` flag.
This will include all available dataset-level metadata.

Using the --folder flag you can add a custom folder name that contains your xmls/mdocs
(no further nesting!). This is mainly meant for cases where local facilities deviate
from TFS folder structures when making data available to users.

## SciCat Ingestor integration

This tool is a compatible metadata extractor for use with the [SciCat Web
Ingestor](https://github.com/SwissOpenEM/Ingestor). It can be installed automatically by
including the following in your ingestor configuration file:

```yaml
MetadataExtractors:
  - Name: LS
    GithubOrg: SwissOpenEM
    GithubProject: oscem-extractor-life
    Version: v0.3.0
    Executable: oscem-extractor-life
    Checksum: 805fd036f2c83284b2cd70f2e7f3fafbe17bc750d2156f604c1505f7d5791d75
    ChecksumAlg: sha256
    CommandLineTemplate: "-i '{{.SourceFolder}}' -o '{{.OutputFile}}'"
    Methods:
      - Name: Single Particle
        Schema: oscem_schemas.schema.json
      - Name: Cellular Tomography
        Schema: oscem_cellular_tomo.json
      - Name: Tomography
        Schema: oscem_tomo.json
      - Name: EnvironmentalTomography
        Schema: oscem_env_tomo.json
```

This will automatically download and install the oscem-extractor-life with the
specified version.

## Schema

Output is compatible with [OSCEM schemas](https://github.com/osc-em/oscem-schemas).

Specific schema used to generate standard schema conform output (works for SPA and
Tomography):
<https://github.com/osc-em/oscem-schemas/blob/linkml_yaml/src/oscem_schemas/schema/oscem_schemas_tomo.yaml>
with LinkML gen-golang
