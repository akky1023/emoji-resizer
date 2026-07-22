package main

import (
	"bufio"
	"fmt"
	"os"
)

var version = "devel"

func main() {
	os.Args = preprocessConfigArgs(os.Args)

	opts, args, err := parseOptions()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if opts.showVersion {
		fmt.Printf("emoji-resizer %s\n", version)
		return
	}

	reader := bufio.NewReader(os.Stdin)
	if opts.zipMode && !opts.checkMode {
		opts.category, opts.license = promptCategoryAndLicense(reader, opts.cfgCategory, opts.cfgLicense)
	}

	filesToProcess := collectFilesToProcess(args, opts.recursive, opts.outDir, opts.absOutDir, opts.checkMode)
	if len(filesToProcess) == 0 {
		if opts.checkMode {
			fmt.Println("OK")
			os.Exit(0)
		}
		fmt.Println("No supported image files found to process.")
		return
	}

	if opts.filenameOption {
		for _, file := range filesToProcess {
			fnOpt := parseFilenameOption(file)
			if fnOpt.HasInvalidOptionPos {
				fmt.Fprintf(os.Stderr, "Error: invalid filename option position in %s: option must be placed at the end before extension (e.g. name@alias.option.ext)\n", file)
				os.Exit(1)
			}
			if fnOpt.HasR && fnOpt.HasS {
				fmt.Fprintf(os.Stderr, "Error: invalid filename option in %s: 'r' and 's' cannot be specified together\n", file)
				os.Exit(1)
			}
		}
	}

	if opts.checkMode {
		executeCheckMode(filesToProcess, opts.zipMode, opts.namePrefix, opts.nameSuffix, reader, opts.filenameOption)
		return
	}

	printStartMessage(len(filesToProcess), opts)

	absTopLevelInDir, topLevelOutDir := resolveDirectoryPaths(args, opts.outDir, opts.absOutDir)

	successCount, failureCount, dirZips, allZipItems, allEmojiEntries := processBatchImages(filesToProcess, opts, reader, absTopLevelInDir)

	if opts.zipMode && successCount > 0 {
		if err := createZipArchives(dirZips, allZipItems, allEmojiEntries, opts.recursive, topLevelOutDir, opts.category); err != nil {
			fmt.Printf("Failed to create ZIP archives: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("Finished. Successfully processed %d/%d files.\n", successCount, len(filesToProcess))
	if failureCount > 0 {
		os.Exit(1)
	}
}
