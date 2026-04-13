package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/major-technology/cli/errors"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the major CLI to the latest version",
	Long:  `Automatically detects your installation method (brew or direct install) and updates to the latest version.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate(cmd)
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command) error {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#87D7FF"))

	stepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#87D7FF"))

	successStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00FF00"))

	cmd.Println(titleStyle.Render("🔄 Updating Major CLI..."))
	cmd.Println()

	// Detect installation method
	installMethod := detectInstallMethod()

	cmd.Println(stepStyle.Render(fmt.Sprintf("▸ Detected installation method: %s", installMethod)))

	switch installMethod {
	case "brew":
		return updateViaBrew(cmd, stepStyle, successStyle)
	case "direct":
		return updateViaDirect(cmd, stepStyle, successStyle)
	default:
		return fmt.Errorf("could not detect installation method")
	}
}

func detectInstallMethod() string {
	// Check if installed via brew (macOS/Linux only)
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		if _, err := exec.LookPath("brew"); err == nil {
			brewListCmd := exec.Command("brew", "list", "major")
			if err := brewListCmd.Run(); err == nil {
				return "brew"
			}
		}
	}

	// Otherwise assume direct install
	return "direct"
}

func updateViaBrew(cmd *cobra.Command, stepStyle, successStyle lipgloss.Style) error {
	cmd.Println(stepStyle.Render("▸ Updating via Homebrew..."))

	// Update brew first
	updateCmd := exec.Command("brew", "update")
	updateCmd.Stdout = os.Stdout
	updateCmd.Stderr = os.Stderr
	if err := updateCmd.Run(); err != nil {
		return errors.WrapError("failed to update Homebrew", err)
	}

	// Upgrade major
	upgradeCmd := exec.Command("brew", "upgrade", "major")
	upgradeCmd.Stdout = os.Stdout
	upgradeCmd.Stderr = os.Stderr

	if err := upgradeCmd.Run(); err != nil {
		// Check if it's already up to date
		if strings.Contains(err.Error(), "already installed") {
			cmd.Println()
			cmd.Println(successStyle.Render("✓ Major CLI is already up to date!"))
			return nil
		}
		return errors.WrapError("failed to upgrade major", err)
	}

	cmd.Println()
	cmd.Println(successStyle.Render("✓ Successfully updated Major CLI!"))
	return nil
}

func updateViaDirect(cmd *cobra.Command, stepStyle, successStyle lipgloss.Style) error {
	cmd.Println(stepStyle.Render("▸ Downloading latest version..."))

	// On Unix with bash available, use the install script for backwards compat
	if runtime.GOOS != "windows" {
		if _, err := exec.LookPath("bash"); err == nil {
			installScriptURL := "https://raw.githubusercontent.com/major-technology/cli/main/install.sh"
			curlCmd := exec.Command("bash", "-c", fmt.Sprintf("curl -fsSL %s | bash", installScriptURL))
			curlCmd.Stdout = os.Stdout
			curlCmd.Stderr = os.Stderr
			curlCmd.Stdin = os.Stdin

			if err := curlCmd.Run(); err != nil {
				return errors.WrapError("failed to download and install update", err)
			}

			cmd.Println()
			cmd.Println(successStyle.Render("✓ Successfully updated Major CLI!"))
			return nil
		}
	}

	// Go-native update: download binary directly from S3.
	// Used on Windows and as a fallback on Unix without bash.
	if err := updateViaDirectDownload(cmd, stepStyle); err != nil {
		return err
	}

	cmd.Println()
	cmd.Println(successStyle.Render("✓ Successfully updated Major CLI!"))
	return nil
}

func updateViaDirectDownload(cmd *cobra.Command, stepStyle lipgloss.Style) error {
	s3Bucket := "https://major-cli-releases.s3.us-west-1.amazonaws.com"

	// Get latest version
	resp, err := http.Get(s3Bucket + "/latest-version")
	if err != nil {
		return errors.WrapError("failed to check latest version", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to fetch latest-version: status %d", resp.StatusCode)
	}

	versionBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.WrapError("failed to read version", err)
	}
	version := strings.TrimSpace(string(versionBytes))
	if version == "" {
		return fmt.Errorf("empty version from latest-version")
	}

	// Determine OS and arch
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}

	assetName := fmt.Sprintf("major_%s_%s_%s.%s", version, goos, goarch, ext)
	checksumName := fmt.Sprintf("major_%s_checksums.txt", version)
	downloadURL := fmt.Sprintf("%s/%s/%s", s3Bucket, version, assetName)
	checksumURL := fmt.Sprintf("%s/%s/%s", s3Bucket, version, checksumName)

	cmd.Println(stepStyle.Render(fmt.Sprintf("▸ Downloading major v%s...", version)))

	tmpDir, err := os.MkdirTemp("", "major-update-*")
	if err != nil {
		return errors.WrapError("failed to create temp directory", err)
	}
	defer os.RemoveAll(tmpDir)

	// Download asset
	assetPath := filepath.Join(tmpDir, assetName)
	if err := downloadFile(downloadURL, assetPath); err != nil {
		return errors.WrapError("failed to download binary", err)
	}

	// Download and verify checksum
	cmd.Println(stepStyle.Render("▸ Verifying checksum..."))
	checksumPath := filepath.Join(tmpDir, "checksums.txt")
	if err := downloadFile(checksumURL, checksumPath); err != nil {
		return errors.WrapError("failed to download checksums", err)
	}

	if err := verifyChecksum(assetPath, assetName, checksumPath); err != nil {
		return errors.WrapError("checksum verification failed", err)
	}

	// Extract
	binaryName := "major"
	if goos == "windows" {
		binaryName = "major.exe"
		if err := extractZip(assetPath, tmpDir); err != nil {
			return errors.WrapError("failed to extract zip", err)
		}
	} else {
		if err := extractTarGz(assetPath, tmpDir); err != nil {
			return errors.WrapError("failed to extract tar.gz", err)
		}
	}

	// Determine install location
	exe, err := os.Executable()
	if err != nil {
		return errors.WrapError("failed to get executable path", err)
	}
	exe, _ = filepath.EvalSymlinks(exe)

	// Pre-promotion health check: run the new binary to verify it works
	// and reports the expected version before swapping it in.
	srcPath := filepath.Join(tmpDir, binaryName)
	cmd.Println(stepStyle.Render("▸ Verifying new binary..."))
	if err := healthCheckBinary(srcPath, version); err != nil {
		return errors.WrapError("new binary failed health check", err)
	}

	// Replace the binary
	if err := replaceBinary(exe, srcPath); err != nil {
		return errors.WrapError("failed to replace binary", err)
	}

	return nil
}

func downloadFile(url, dst string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download %s failed with status %d", url, resp.StatusCode)
	}

	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func verifyChecksum(assetPath, assetName, checksumPath string) error {
	data, err := os.ReadFile(checksumPath)
	if err != nil {
		return err
	}

	var expected string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, assetName) {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				expected = parts[0]
			}
			break
		}
	}

	if expected == "" {
		return fmt.Errorf("no checksum found for %s", assetName)
	}

	f, err := os.Open(assetPath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actual := fmt.Sprintf("%x", h.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("expected %s, got %s", expected, actual)
	}

	return nil
}

func healthCheckBinary(path, expectedVersion string) error {
	// Make sure it's executable on Unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0755); err != nil {
			return err
		}
	}

	out, err := exec.Command(path, "--version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("binary failed to execute: %w (output: %s)", err, string(out))
	}

	if !strings.Contains(string(out), expectedVersion) {
		return fmt.Errorf("version mismatch: expected %s, got %q", expectedVersion, strings.TrimSpace(string(out)))
	}

	return nil
}

func extractTarGz(src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		outPath := filepath.Join(dst, filepath.Base(header.Name))
		outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
		if err != nil {
			return err
		}
		if _, err := io.Copy(outFile, tr); err != nil {
			outFile.Close()
			return err
		}
		outFile.Close()
	}
	return nil
}

func extractZip(src, dst string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		outPath := filepath.Join(dst, filepath.Base(f.Name))
		outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		if _, err := io.Copy(outFile, rc); err != nil {
			outFile.Close()
			rc.Close()
			return err
		}
		outFile.Close()
		rc.Close()
	}
	return nil
}

func replaceBinary(dst, src string) error {
	if runtime.GOOS == "windows" {
		return replaceBinaryWindows(dst, src)
	}

	// Unix: write to temp file next to dst, then atomic rename.
	tmpDst := dst + ".new"
	if err := copyFile(src, tmpDst, 0755); err != nil {
		return err
	}
	return os.Rename(tmpDst, dst)
}

func replaceBinaryWindows(dst, src string) error {
	// On Windows, a running .exe can be renamed but not overwritten.
	// Strategy:
	//   1. Copy new binary to dst.new
	//   2. Rename running dst -> dst.old
	//   3. Rename dst.new -> dst
	// If step 3 fails, rename dst.old back to dst.
	// The .old file is left behind (can't delete a running exe); cleaned on next update.

	newPath := dst + ".new"
	oldPath := dst + ".old"

	// Clean up artifacts from any prior update
	os.Remove(newPath)
	os.Remove(oldPath)

	// Step 1: write new binary to .new
	if err := copyFile(src, newPath, 0755); err != nil {
		return fmt.Errorf("failed to write new binary: %w", err)
	}

	// Step 2: move running binary out of the way
	if err := os.Rename(dst, oldPath); err != nil {
		os.Remove(newPath)
		return fmt.Errorf("failed to rename running binary: %w", err)
	}

	// Step 3: move new binary into place
	if err := os.Rename(newPath, dst); err != nil {
		// Rollback: restore original
		os.Rename(oldPath, dst)
		return fmt.Errorf("failed to install new binary: %w", err)
	}

	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
