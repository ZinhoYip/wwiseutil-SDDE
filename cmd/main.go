package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"wwiseutil/bnk"
	"wwiseutil/pck"
)

func main() {
	log.SetFlags(0)

	// Define flags in the original style
	var filepathFlag, outputFlag, targetFlag string
	flag.StringVar(&filepathFlag, "f", "", "(shorthand for -filepath)")
	flag.StringVar(&filepathFlag, "filepath", "", "The path to the source .bnk or .pck file.")
	flag.StringVar(&outputFlag, "o", "", "(shorthand for -output)")
	flag.StringVar(&outputFlag, "output", "", "Output directory for unpacking or output file for repacking.")
	flag.StringVar(&targetFlag, "t", "", "(shorthand for -target)")
	flag.StringVar(&targetFlag, "target", "", "Directory containing replacement files.")

	var unpackFlag, replaceFlag, verboseFlag bool
	flag.BoolVar(&unpackFlag, "u", false, "(shorthand for -unpack)")
	flag.BoolVar(&unpackFlag, "unpack", false, "Unpack a .bnk or .pck into separate files.")
	flag.BoolVar(&replaceFlag, "r", false, "(shorthand for -replace)")
	flag.BoolVar(&replaceFlag, "replace", false, "Replace files in a source .pck or .bnk.")
	flag.BoolVar(&verboseFlag, "v", false, "(shorthand for -verbose)")
	flag.BoolVar(&verboseFlag, "verbose", false, "Show additional information about the parsed file.")

	flag.Parse()

	if filepathFlag == "" {
		log.Println("Error: -filepath (-f) is a required argument.")
		flag.Usage()
		return
	}

	if unpackFlag {
		if outputFlag == "" {
			log.Println("Error: -output (-o) is required for unpacking.")
			flag.Usage()
			return
		}
		handleUnpack(filepathFlag, outputFlag, verboseFlag)
	} else if replaceFlag {
		if outputFlag == "" {
			log.Println("Error: -output (-o) is required for replacing.")
			flag.Usage()
			return
		}
		if targetFlag == "" {
			log.Println("Error: -target (-t) is required for replacing.")
			flag.Usage()
			return
		}
		handleReplace(filepathFlag, outputFlag, targetFlag, verboseFlag)
	} else {
		log.Println("No operation specified. Use -unpack or -replace.")
		flag.Usage()
	}
}

func handleUnpack(inputFile, outputDir string, verbose bool) {
	ext := strings.ToLower(filepath.Ext(inputFile))

	switch ext {
	case ".pck", ".npck":
		log.Printf("Unpacking PCK file: %s", inputFile)
		f, err := pck.Open(inputFile)
		if err != nil {
			log.Fatalf("Error opening PCK file: %v", err)
		}
		defer f.Close()

		if verbose {
			timestamp := time.Now().Format(time.RFC3339Nano)
			verboseOutput := f.String()
			finalOutput := fmt.Sprintf("Log generated at: %s\n\n%s", timestamp, verboseOutput)

			log.Println(finalOutput)
			logFile, err := os.Create("log.txt")
			if err != nil {
				log.Printf("Warning: could not create log.txt: %v", err)
			} else {
				defer logFile.Close()
				_, err := logFile.WriteString(finalOutput)
				if err != nil {
					log.Printf("Warning: failed to write to log.txt: %v", err)
				}
			}
		}

		if err := f.UnpackTo(outputDir); err != nil {
			log.Fatalf("Error unpacking PCK file: %v", err)
		}
		log.Printf("Successfully unpacked files to: %s", outputDir)

	case ".bnk", ".nbnk":
		log.Printf("Unpacking BNK file: %s", inputFile)
		f, err := bnk.Open(inputFile)
		if err != nil {
			log.Fatalf("Error opening BNK file: %v", err)
		}
		defer f.Close()

		if verbose {
			log.Println(f.String())
		}

		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.Fatalf("Error creating output directory: %v", err)
		}

		for _, wem := range f.Wems() {
			wemName := fmt.Sprintf("%d.wem", wem.Descriptor.WemId)
			outPath := filepath.Join(outputDir, wemName)
			outFile, err := os.Create(outPath)
			if err != nil {
				log.Printf("Failed to create file %s: %v", outPath, err)
				continue
			}
			_, err = outFile.ReadFrom(wem.Reader)
			outFile.Close()
			if err != nil {
				log.Printf("Failed to write wem %s: %v", wemName, err)
			}
		}
		log.Printf("Successfully unpacked WEM files to: %s", outputDir)

	default:
		log.Fatalf("Unsupported file type: %s", ext)
	}
}

func handleReplace(inputFile, outputFile, targetDir string, verbose bool) {
	ext := strings.ToLower(filepath.Ext(inputFile))
	if ext != ".pck" && ext != ".npck" {
		log.Fatalf("Replacing is only supported for the special .pck format.")
	}

	// Open the source PCK to get the ID mappings from indexes
	srcPck, err := pck.Open(inputFile)
	if err != nil {
		log.Fatalf("Error opening source PCK: %v", err)
	}
	defer srcPck.Close()

	if verbose {
		log.Println("Source file structure:")
			timestamp := time.Now().Format(time.RFC3339Nano)
			verboseOutput := srcPck.String()
			finalOutput := fmt.Sprintf("Log generated at: %s\n\n%s", timestamp, verboseOutput)

			log.Println(finalOutput)
			logFile, err := os.Create("log.txt")
			if err != nil {
				log.Printf("Warning: could not create log.txt: %v", err)
			} else {
				defer logFile.Close()
				_, err := logFile.WriteString(finalOutput)
				if err != nil {
					log.Printf("Warning: failed to write to log.txt: %v", err)
				}
			}
	}

	// Find replacement files
	replacements, err := findReplacementFiles(targetDir, srcPck)
	if err != nil {
		log.Fatalf("Error finding replacement files: %v", err)
	}

	if len(replacements) == 0 {
		log.Println("No valid replacement files found in target directory. Nothing to do.")
		return
	}

	var replacementNames []string
	for _, r := range replacements {
		replacementNames = append(replacementNames, filepath.Base(r.Path))
	}

	log.Printf("Using %d replacement file(s): %s", len(replacements), strings.Join(replacementNames, ", "))

	bytesWritten, err := pck.Repack(inputFile, outputFile, replacements)
	if err != nil {
		log.Fatalf("Error during repack: %v", err)
	}

	log.Println("Repack completed successfully!")
	log.Printf("Output file written to: %s", outputFile)
	log.Printf("Wrote %d bytes in total", bytesWritten)
}

func findReplacementFiles(targetDir string, srcPck *pck.File) ([]*pck.ReplacementFile, error) {
	var replacements []*pck.ReplacementFile

	// Scan for BNK replacements
	bnkTargetDir := filepath.Join(targetDir, "bnk")
	if _, err := os.Stat(bnkTargetDir); !os.IsNotExist(err) {
		err := filepath.WalkDir(bnkTargetDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				base := filepath.Base(path)
				ext := filepath.Ext(base)
				indexStr := strings.TrimSuffix(base, ext)
				index, err := strconv.Atoi(indexStr)
				if err != nil {
					log.Printf("Warning: could not parse index from filename %s, skipping.", base)
					return nil
				}

				if index < 1 || index > len(srcPck.BnkIndexes) {
					log.Printf("Warning: index %d from filename %s is out of bounds for BNK files (1-%d), skipping.", index, base, len(srcPck.BnkIndexes))
					return nil
				}
				// Convert 1-based user index to 0-based slice index
				id := srcPck.BnkIndexes[index-1].ID
				replacements = append(replacements, &pck.ReplacementFile{ID: id, Path: path, Type: "bnk"})
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error scanning bnk target directory: %w", err)
		}
	}

	// Scan for WEM replacements
	wemTargetDir := filepath.Join(targetDir, "wem")
	if _, err := os.Stat(wemTargetDir); !os.IsNotExist(err) {
		err := filepath.WalkDir(wemTargetDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !d.IsDir() {
				base := filepath.Base(path)
				ext := filepath.Ext(base)
				indexStr := strings.TrimSuffix(base, ext)
				index, err := strconv.Atoi(indexStr)
				if err != nil {
					log.Printf("Warning: could not parse index from filename %s, skipping.", base)
					return nil
				}

				if index < 1 || index > len(srcPck.WemIndexes) {
					log.Printf("Warning: index %d from filename %s is out of bounds for WEM files (1-%d), skipping.", index, base, len(srcPck.WemIndexes))
					return nil
				}
				// Convert 1-based user index to 0-based slice index
				id := srcPck.WemIndexes[index-1].ID
				replacements = append(replacements, &pck.ReplacementFile{ID: id, Path: path, Type: "wem"})
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error scanning wem target directory: %w", err)
		}
	}

	return replacements, nil
}