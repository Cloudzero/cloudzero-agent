package compress

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

// File compresses a single file into a .tar.gz archive.
// It is a pure function and does not handle locking.
func File(srcFilePath string) (*string, error) {
	// Get file info to retrieve the file name and size
	info, err := os.Stat(srcFilePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, nil
	}

	// Read the file to be compressed
	srcFile, err := os.Open(srcFilePath)
	if err != nil {
		// may not be an error in a scaled case
		log.Warn().Err(err).Msgf("failed to open file: %s", srcFilePath)
		return nil, nil
	}
	defer srcFile.Close()

	// Create the destination file
	destFilePath := srcFilePath + ".tgz"
	destFile, err := os.Create(destFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination archive file: %w", err)
	}
	defer destFile.Close()

	// Create a gzip writer
	gzipWriter := gzip.NewWriter(destFile)
	defer gzipWriter.Close()

	// Create a tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Create a tar header
	header := &tar.Header{
		Name: filepath.Base(srcFilePath),
		Size: info.Size(),
		Mode: int64(info.Mode()),
	}

	// Write the header to the tar archive
	if err := tarWriter.WriteHeader(header); err != nil {
		_ = os.Remove(destFilePath)
		return nil, fmt.Errorf("failed to write tar header: %w", err)
	}

	// Copy the file data into the tar archive
	if _, err := io.Copy(tarWriter, srcFile); err != nil {
		_ = os.Remove(destFilePath)
		return nil, fmt.Errorf("failed to copy file data to tar archive: %w", err)
	}

	return &destFilePath, nil
}
