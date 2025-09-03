// Package pck implements access to the Wwise File Package file format.
package pck

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// A File represents an open Wwise File Package.
// This version is modified to support a special PCK format that contains both BNK and WEM files.
type File struct {
	closer     io.Closer
	reader     readerAtSeeker
	Header     *Header
	BnkIndexes []*FileIndex
	WemIndexes []*FileIndex
	Bnks       []*EmbeddedFile
	Wems       []*EmbeddedFile
}

// Header represents a single Wwise File Package header.
// The structure is adapted for variability.
type Header struct {
	Identifier             [4]byte
	HeaderAndIndexesLength uint32 // Length from this field's end to the end of all indexes.
	Unknown                []byte   // Variable length unknown section, determined by filename
}

// FileIndex represents the 24-byte structure for both BNK and WEM file indexes.
type FileIndex struct {
	ID       uint32
	Type     uint32
	Length   uint32
	Unknown1 uint32
	Offset   uint32 // Absolute offset from the beginning of the file
	Unknown2 uint32
}

// EmbeddedFile represents a file (BNK or WEM) stored within the PCK.
type EmbeddedFile struct {
	Index  *FileIndex
	Reader io.Reader
	Name   string
}

// readerAtSeeker is an interface that groups io.ReaderAt and io.ReadSeeker.
// os.File implements this.
type readerAtSeeker interface {
	io.Reader
	io.ReaderAt
	io.Seeker
	io.Closer
}

// NewFile creates a new File for accessing the special Wwise File Package format.
// It requires the size of the 'Unknown' header field to be determined beforehand.
func NewFile(r readerAtSeeker, unknownSize int) (*File, error) {
	pck := new(File)
	pck.closer = r
	pck.reader = r

	// Read Header
	hdr := new(Header)
	if err := binary.Read(r, binary.LittleEndian, &hdr.Identifier); err != nil {
		return nil, fmt.Errorf("reading header identifier: %w", err)
	}
	if err := binary.Read(r, binary.LittleEndian, &hdr.HeaderAndIndexesLength); err != nil {
		return nil, fmt.Errorf("reading header and indexes length: %w", err)
	}
	hdr.Unknown = make([]byte, unknownSize)
	if _, err := io.ReadFull(r, hdr.Unknown); err != nil {
		return nil, fmt.Errorf("reading header unknown data (size %d): %w", unknownSize, err)
	}
	pck.Header = hdr

	// Read BNK counts and indexes
	var bnkCount uint32
	if err := binary.Read(r, binary.LittleEndian, &bnkCount); err != nil {
		return nil, fmt.Errorf("reading bnk count: %w", err)
	}
	pck.BnkIndexes = make([]*FileIndex, 0, bnkCount)
	for i := uint32(0); i < bnkCount; i++ {
		idx := new(FileIndex)
		if err := binary.Read(r, binary.LittleEndian, idx); err != nil {
			return nil, fmt.Errorf("reading bnk index %d: %w", i, err)
		}
		pck.BnkIndexes = append(pck.BnkIndexes, idx)
	}

	// Read WEM counts and indexes
	var wemCount uint32
	if err := binary.Read(r, binary.LittleEndian, &wemCount); err != nil {
		return nil, fmt.Errorf("reading wem count: %w", err)
	}
	pck.WemIndexes = make([]*FileIndex, 0, wemCount)
	for i := uint32(0); i < wemCount; i++ {
		idx := new(FileIndex)
		if err := binary.Read(r, binary.LittleEndian, idx); err != nil {
			return nil, fmt.Errorf("reading wem index %d: %w", i, err)
		}
		pck.WemIndexes = append(pck.WemIndexes, idx)
	}

	// Create readers for BNK file data
	pck.Bnks = make([]*EmbeddedFile, len(pck.BnkIndexes))
	for i, idx := range pck.BnkIndexes {
		pck.Bnks[i] = &EmbeddedFile{
			Index:  idx,
			Reader: io.NewSectionReader(r, int64(idx.Offset), int64(idx.Length)),
			Name:   fmt.Sprintf("%d.bnk", idx.ID),
		}
	}

	// Create readers for WEM file data
	pck.Wems = make([]*EmbeddedFile, len(pck.WemIndexes))
	for i, idx := range pck.WemIndexes {
		pck.Wems[i] = &EmbeddedFile{
			Index:  idx,
			Reader: io.NewSectionReader(r, int64(idx.Offset), int64(idx.Length)),
			Name:   fmt.Sprintf("%d.wem", idx.ID),
		}
	}

	return pck, nil
}

// Open opens the File at the specified path and prepares it for use.
// It determines the header's 'Unknown' field size based on the filename.
func Open(path string) (*File, error) {
	var unknownSize int
	lowerPath := strings.ToLower(path)

	if strings.HasSuffix(lowerPath, "sfx.pck") {
		unknownSize = 36
	} else if strings.HasSuffix(lowerPath, "english(us).pck") {
		unknownSize = 68
	} else {
		return nil, fmt.Errorf("unsupported pck file: %s - unknown header size", filepath.Base(path))
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	pck, err := NewFile(f, unknownSize)
	if err != nil {
		f.Close()
		return nil, err
	}
	return pck, nil
}

// Close closes the File.
func (pck *File) Close() error {
	if pck.closer != nil {
		return pck.closer.Close()
	}
	return nil
}

// UnpackTo extracts all BNK and WEM files to a specified directory.
func (pck *File) UnpackTo(outputDir string) error {
	// Unpack BNKs
	bnkDir := filepath.Join(outputDir, "bnk")
	if err := os.MkdirAll(bnkDir, 0755); err != nil {
		return err
	}
	for _, bnk := range pck.Bnks {
		outFile, err := os.Create(filepath.Join(bnkDir, bnk.Name))
		if err != nil {
			return err // No need to close if creation failed
		}
		defer outFile.Close()

		if _, err := io.Copy(outFile, bnk.Reader); err != nil {
			return err
		}
	}

	// Unpack WEMs
	wemDir := filepath.Join(outputDir, "wem")
	if err := os.MkdirAll(wemDir, 0755); err != nil {
		return err
	}
	for _, wem := range pck.Wems {
		outFile, err := os.Create(filepath.Join(wemDir, wem.Name))
		if err != nil {
			return err
		}
		defer outFile.Close()

		if _, err := io.Copy(outFile, wem.Reader); err != nil {
			return err
		}
	}
	return nil
}

// WriteTo writes the entire PCK file to a writer.
func (pck *File) WriteTo(w io.Writer) (int64, error) {
	var written int64

	// Use a buffered writer for efficiency
	bufWriter := bufio.NewWriter(w)

	// Write Header
	if err := binary.Write(bufWriter, binary.LittleEndian, pck.Header.Identifier); err != nil {
		return written, err
	}
	written += 4
	if err := binary.Write(bufWriter, binary.LittleEndian, pck.Header.HeaderAndIndexesLength); err != nil {
		return written, err
	}
	written += 4
	n, err := bufWriter.Write(pck.Header.Unknown)
	if err != nil {
		return written, err
	}
	written += int64(n)

	// Write BNK Count and Indexes
	bnkCount := uint32(len(pck.BnkIndexes))
	if err := binary.Write(bufWriter, binary.LittleEndian, bnkCount); err != nil {
		return written, err
	}
	written += 4
	for _, idx := range pck.BnkIndexes {
		if err := binary.Write(bufWriter, binary.LittleEndian, idx); err != nil {
			return written, err
		}
		written += 24
	}

	// Write WEM Count and Indexes
	wemCount := uint32(len(pck.WemIndexes))
	if err := binary.Write(bufWriter, binary.LittleEndian, wemCount); err != nil {
		return written, err
	}
	written += 4
	for _, idx := range pck.WemIndexes {
		if err := binary.Write(bufWriter, binary.LittleEndian, idx); err != nil {
			return written, err
		}
		written += 24
	}

	// Flush header/index data
	if err := bufWriter.Flush(); err != nil {
		return written, err
	}

	// Write Data
	for _, bnk := range pck.Bnks {
		if r, ok := bnk.Reader.(io.ReadSeeker); ok {
			r.Seek(0, io.SeekStart)
		}
		n, err := io.Copy(w, bnk.Reader)
		if err != nil {
			return written, err
		}
		written += n
	}
	for _, wem := range pck.Wems {
		if r, ok := wem.Reader.(io.ReadSeeker); ok {
			r.Seek(0, io.SeekStart)
		}
		n, err := io.Copy(w, wem.Reader)
		if err != nil {
			return written, err
		}
		written += n
	}

	return written, nil
}

func (pck *File) String() string {
	b := new(strings.Builder)
	fmt.Fprintf(b, "PCK File (Hybrid BNK/WEM Format)\n")
	fmt.Fprintf(b, "BNK Count: %d\n", len(pck.BnkIndexes))
	fmt.Fprintf(b, "WEM Count: %d\n\n", len(pck.WemIndexes))

	b.WriteString("--- BNK Files ---\n")
	fmt.Fprintf(b, "%-7s | %-10s | %-15s | %-10s\n", "Index", "ID", "Offset", "Length")
	for i, idx := range pck.BnkIndexes {
		fmt.Fprintf(b, "%-7d | %-10d | %-15d | %-10d\n", i+1, idx.ID, idx.Offset, idx.Length)
	}

	b.WriteString("\n--- WEM Files ---\n")
	fmt.Fprintf(b, "%-7s | %-10s | %-15s | %-10s\n", "Index", "ID", "Offset", "Length")
	for i, idx := range pck.WemIndexes {
		fmt.Fprintf(b, "%-7d | %-10d | %-15d | %-10d\n", i+1, idx.ID, idx.Offset, idx.Length)
	}

	return b.String()
}

// ReplacementFile defines a file to be used for replacement.
type ReplacementFile struct {
	ID   uint32
	Path string
	Data []byte
	Type string // "bnk" or "wem"
}

// Repack rebuilds the PCK file with replacement files in a memory-efficient way.
func Repack(inputFile string, outputFile string, replacements []*ReplacementFile) (int64, error) {
	// Open the original file
	pckFile, err := Open(inputFile)
	if err != nil {
		return 0, fmt.Errorf("opening original file for repack: %w", err)
	}
	defer pckFile.Close()

	// Create the output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		return 0, fmt.Errorf("creating output file: %w", err)
	}
	defer outFile.Close()

	// Create a map for quick lookup of replacements
	replacementMap := make(map[string]map[uint32]*ReplacementFile)
	replacementMap["bnk"] = make(map[uint32]*ReplacementFile)
	replacementMap["wem"] = make(map[uint32]*ReplacementFile)

	for _, r := range replacements {
		data, err := os.ReadFile(r.Path)
		if err != nil {
			return 0, fmt.Errorf("reading replacement file %s: %w", r.Path, err)
		}
		r.Data = data
		replacementMap[r.Type][r.ID] = r
	}

	// Create new index slices
	newBnkIndexes := make([]*FileIndex, len(pckFile.BnkIndexes))
	newWemIndexes := make([]*FileIndex, len(pckFile.WemIndexes))

	// Copy original indexes and update lengths for replaced files
	for i, idx := range pckFile.BnkIndexes {
		newIdx := *idx // Make a copy
		if r, ok := replacementMap["bnk"][idx.ID]; ok {
			newIdx.Length = uint32(len(r.Data))
		}
		newBnkIndexes[i] = &newIdx
	}
	for i, idx := range pckFile.WemIndexes {
		newIdx := *idx // Make a copy
		if r, ok := replacementMap["wem"][idx.ID]; ok {
			newIdx.Length = uint32(len(r.Data))
		}
		newWemIndexes[i] = &newIdx
	}

	// === Recalculate Offsets and Header Length ===
	headerSize := uint32(4 + 4 + len(pckFile.Header.Unknown))
	bnkIndexSize := uint32(len(newBnkIndexes) * 24)
	wemIndexSize := uint32(len(newWemIndexes) * 24)
	dataAreaStartOffset := headerSize + 4 + bnkIndexSize + 4 + wemIndexSize

	pckFile.Header.HeaderAndIndexesLength = dataAreaStartOffset - 8 // Subtract Identifier and the field itself

	currentOffset := dataAreaStartOffset
	for _, idx := range newBnkIndexes {
		idx.Offset = currentOffset
		currentOffset += idx.Length
	}
	for _, idx := range newWemIndexes {
		idx.Offset = currentOffset
		currentOffset += idx.Length
	}

	// === Write the new PCK file ===
	var written int64
	bufWriter := bufio.NewWriter(outFile)

	// 1. Write Header
	if err := binary.Write(bufWriter, binary.LittleEndian, pckFile.Header.Identifier); err != nil {
		return written, err
	}
	if err := binary.Write(bufWriter, binary.LittleEndian, pckFile.Header.HeaderAndIndexesLength); err != nil {
		return written, err
	}
	if _, err := bufWriter.Write(pckFile.Header.Unknown); err != nil {
		return written, err
	}

	// 2. Write BNK Indexes
	if err := binary.Write(bufWriter, binary.LittleEndian, uint32(len(newBnkIndexes))); err != nil {
		return written, err
	}
	for _, idx := range newBnkIndexes {
		if err := binary.Write(bufWriter, binary.LittleEndian, idx); err != nil {
			return written, err
		}
	}

	// 3. Write WEM Indexes
	if err := binary.Write(bufWriter, binary.LittleEndian, uint32(len(newWemIndexes))); err != nil {
		return written, err
	}
	for _, idx := range newWemIndexes {
		if err := binary.Write(bufWriter, binary.LittleEndian, idx); err != nil {
			return written, err
		}
	}

	// Flush header/index data to ensure it's written before data blocks
	if err := bufWriter.Flush(); err != nil {
		return written, err
	}
	written = int64(dataAreaStartOffset)


	// 4. Write Data Blocks
	// BNKs
	for i, idx := range pckFile.BnkIndexes {
		var n int64
		var err error
		if r, ok := replacementMap["bnk"][idx.ID]; ok {
			// Write replacement data
			nW, errWrite := outFile.Write(r.Data)
			n = int64(nW)
			err = errWrite
		} else {
			// Copy original data
			pckFile.reader.Seek(int64(idx.Offset), io.SeekStart)
			n, err = io.CopyN(outFile, pckFile.reader, int64(newBnkIndexes[i].Length))
		}
		if err != nil {
			return written, fmt.Errorf("writing bnk ID %d: %w", idx.ID, err)
		}
		written += n
	}

	// WEMs
	for i, idx := range pckFile.WemIndexes {
		var n int64
		var err error
		if r, ok := replacementMap["wem"][idx.ID]; ok {
			// Write replacement data
			nW, errWrite := outFile.Write(r.Data)
			n = int64(nW)
			err = errWrite
		} else {
			// Copy original data
			pckFile.reader.Seek(int64(idx.Offset), io.SeekStart)
			n, err = io.CopyN(outFile, pckFile.reader, int64(newWemIndexes[i].Length))
		}
		if err != nil {
			return written, fmt.Errorf("writing wem ID %d: %w", idx.ID, err)
		}
		written += n
	}

	return written, nil
}
