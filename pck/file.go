// Package pck implements access to the Wwise File Package file format.
package pck

import (
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
	io.ReaderAt
	io.ReadSeeker
	io.Closer
}

// NewFile creates a new File for accessing the special Wwise File Package format.
// It requires the size of the 'Unknown' header field to be determined beforehand.
func NewFile(r readerAtSeeker, unknownSize int) (*File, error) {
	pck := new(File)
	pck.closer = r

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
	for i := uint32(0); i < wemCount; i++ {
		idx := new(FileIndex)
		if err := binary.Read(r, binary.LittleEndian, idx); err != nil {
			return nil, fmt.Errorf("reading wem index %d: %w", i, err)
		}
		pck.WemIndexes = append(pck.WemIndexes, idx)
	}

	// Create readers for BNK file data
	for _, idx := range pck.BnkIndexes {
		reader := io.NewSectionReader(r, int64(idx.Offset), int64(idx.Length))
		pck.Bnks = append(pck.Bnks, &EmbeddedFile{
			Index:  idx,
			Reader: reader,
			Name:   fmt.Sprintf("%d.bnk", idx.ID),
		})
	}

	// Create readers for WEM file data
	for _, idx := range pck.WemIndexes {
		reader := io.NewSectionReader(r, int64(idx.Offset), int64(idx.Length))
		pck.Wems = append(pck.Wems, &EmbeddedFile{
			Index:  idx,
			Reader: reader,
			Name:   fmt.Sprintf("%d.wem", idx.ID),
		})
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
			outFile.Close()
			return err
		}
		if _, err := io.Copy(outFile, bnk.Reader); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()
	}

	// Unpack WEMs
	wemDir := filepath.Join(outputDir, "wem")
	if err := os.MkdirAll(wemDir, 0755); err != nil {
		return err
	}
	for _, wem := range pck.Wems {
		outFile, err := os.Create(filepath.Join(wemDir, wem.Name))
		if err != nil {
			outFile.Close()
			return err
		}
		if _, err := io.Copy(outFile, wem.Reader); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()
	}
	return nil
}

// WriteTo writes the entire PCK file to a writer.
func (pck *File) WriteTo(w io.Writer) (int64, error) {
	var written int64

	// Write Header
	if err := binary.Write(w, binary.LittleEndian, pck.Header.Identifier); err != nil {
		return written, err
	}
	written += 4
	if err := binary.Write(w, binary.LittleEndian, pck.Header.HeaderAndIndexesLength); err != nil {
		return written, err
	}
	written += 4
	n, err := w.Write(pck.Header.Unknown)
	if err != nil {
		return written, err
	}
	written += int64(n)

	// Write BNK Count and Indexes
	bnkCount := uint32(len(pck.BnkIndexes))
	if err := binary.Write(w, binary.LittleEndian, bnkCount); err != nil {
		return written, err
	}
	written += 4
	for _, idx := range pck.BnkIndexes {
		if err := binary.Write(w, binary.LittleEndian, idx); err != nil {
			return written, err
		}
		written += 24
	}

	// Write WEM Count and Indexes
	wemCount := uint32(len(pck.WemIndexes))
	if err := binary.Write(w, binary.LittleEndian, wemCount); err != nil {
		return written, err
	}
	written += 4
	for _, idx := range pck.WemIndexes {
		if err := binary.Write(w, binary.LittleEndian, idx); err != nil {
			return written, err
		}
		written += 24
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
	fmt.Fprintf(b, "% -7s | % -10s | % -15s | % -10s\n", "Index", "ID", "Offset", "Length")
	for i, idx := range pck.BnkIndexes {
		fmt.Fprintf(b, "% -7d | % -10d | % -15d | % -10d\n", i+1, idx.ID, idx.Offset, idx.Length)
	}

	b.WriteString("\n--- WEM Files ---\n")
	fmt.Fprintf(b, "% -7s | % -10s | % -15s | % -10s\n", "Index", "ID", "Offset", "Length")
	for i, idx := range pck.WemIndexes {
		fmt.Fprintf(b, "% -7d | % -10d | % -15d | % -10d\n", i+1, idx.ID, idx.Offset, idx.Length)
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

// Repack rebuilds the PCK file with replacement files.
func Repack(inputFile string, outputFile string, replacements []*ReplacementFile) (int64, error) {
	// Open and parse the original file
	pckFile, err := Open(inputFile)
	if err != nil {
		return 0, fmt.Errorf("opening original file for repack: %w", err)
	}
	defer pckFile.Close()

	// Read replacement file data into memory
	for _, r := range replacements {
		data, err := os.ReadFile(r.Path)
		if err != nil {
			return 0, fmt.Errorf("reading replacement file %s: %w", r.Path, err)
		}
		r.Data = data
	}

	// Create new lists of embedded files, replacing where necessary
	newBnks := []*EmbeddedFile{}
	for _, bnk := range pckFile.Bnks {
		replaced := false
		for _, r := range replacements {
			if r.Type == "bnk" && r.ID == bnk.Index.ID {
				newBnks = append(newBnks, &EmbeddedFile{
					Index: &FileIndex{
						ID:       bnk.Index.ID,
						Type:     bnk.Index.Type,
						Length:   uint32(len(r.Data)),
						Unknown1: bnk.Index.Unknown1,
						// Offset will be recalculated
						Unknown2: bnk.Index.Unknown2,
					},
					Reader: strings.NewReader(string(r.Data)),
					Name:   bnk.Name,
				})
				replaced = true
				break
			}
		}
		if !replaced {
			buf, _ := io.ReadAll(bnk.Reader)
			bnk.Reader = strings.NewReader(string(buf))
			newBnks = append(newBnks, bnk)
		}
	}
	pckFile.Bnks = newBnks
	pckFile.BnkIndexes = getIndexes(newBnks)

	newWems := []*EmbeddedFile{}
	for _, wem := range pckFile.Wems {
		replaced := false
		for _, r := range replacements {
			if r.Type == "wem" && r.ID == wem.Index.ID {
				newWems = append(newWems, &EmbeddedFile{
					Index: &FileIndex{
						ID:       wem.Index.ID,
						Type:     wem.Index.Type,
						Length:   uint32(len(r.Data)),
						Unknown1: wem.Index.Unknown1,
						// Offset will be recalculated
						Unknown2: wem.Index.Unknown2,
					},
					Reader: strings.NewReader(string(r.Data)),
					Name:   wem.Name,
				})
				replaced = true
				break
			}
		}
		if !replaced {
			buf, _ := io.ReadAll(wem.Reader)
			wem.Reader = strings.NewReader(string(buf))
			newWems = append(newWems, wem)
		}
	}
	pckFile.Wems = newWems
	pckFile.WemIndexes = getIndexes(newWems)

	// === CORE FIXES START HERE ===

	// 1. Calculate the correct starting offset for the data area
	dataAreaStartOffset := uint32(4+4+len(pckFile.Header.Unknown)) + 4 + uint32(len(pckFile.BnkIndexes)*24) + 4 + uint32(len(pckFile.WemIndexes)*24)

	// 2. Recalculate all offsets as ABSOLUTE offsets from the start of the file
	currentOffset := dataAreaStartOffset
	for _, bnk := range pckFile.Bnks {
		bnk.Index.Offset = currentOffset
		currentOffset += bnk.Index.Length
	}
	for _, wem := range pckFile.Wems {
		wem.Index.Offset = currentOffset
		currentOffset += wem.Index.Length
	}

	// 3. Recalculate the HeaderAndIndexesLength field
	// This is the length from after the field itself to the end of all indexes.
	pckFile.Header.HeaderAndIndexesLength = dataAreaStartOffset - 8 // Subtract the first 8 bytes (Identifier + field itself)

	// === CORE FIXES END HERE ===

	// Create and write to the output file
	outFile, err := os.Create(outputFile)
	if err != nil {
		return 0, fmt.Errorf("creating output file: %w", err)
	}
	defer outFile.Close()

	return pckFile.WriteTo(outFile)
}

func getIndexes(files []*EmbeddedFile) []*FileIndex {
	indexes := make([]*FileIndex, len(files))
	for i, f := range files {
		indexes[i] = f.Index
	}
	return indexes
}