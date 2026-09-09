package paths

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/m8urnett/PingGrid/internal/toolkit/errors"
)

var windowsEnvironmentVariable = regexp.MustCompile(`%([^%]+)%`)

// NormalizeOptions controls path normalization without losing facts that
// callers may need for validation before filesystem access.
type NormalizeOptions struct {
	MakeAbsolute    bool
	ResolveSymlinks bool
}

// NormalizedPath contains both the supplied spelling and the normalized path.
type NormalizedPath struct {
	Supplied    string
	Expanded    string
	Path        string
	WasRelative bool
}

// Normalize resolves absolute vs relative paths, expands environment variables,
// and cleans up directory separators.
// It returns the normalized path and an error if normalization fails.
func Normalize(path string) (string, error) {
	result, err := NormalizeDetailed(path, NormalizeOptions{
		MakeAbsolute:    true,
		ResolveSymlinks: true,
	})
	if err != nil {
		return "", err
	}
	return result.Path, nil
}

// NormalizeDetailed normalizes a path while preserving whether the accepted
// input was relative. It expands both portable $NAME/${NAME} variables and
// Windows-native %NAME% variables.
func NormalizeDetailed(path string, options NormalizeOptions) (NormalizedPath, error) {
	supplied := path
	if path == "" {
		return NormalizedPath{}, errors.New(
			errors.ExitInput,
			"PATH_INVALID",
			"path cannot be empty",
			path,
			"Provide a valid path",
			nil,
		)
	}

	path = strings.TrimSpace(path)
	path = strings.Trim(path, `"`)
	path = strings.Trim(path, `'`)

	if path == "" {
		return NormalizedPath{}, errors.New(
			errors.ExitInput,
			"PATH_INVALID",
			"path cannot be empty after trimming quotes and whitespace",
			path,
			"Provide a valid path",
			nil,
		)
	}

	expanded, err := expandWindowsEnvironment(path)
	if err != nil {
		return NormalizedPath{}, err
	}
	expanded, err = expandPortableEnvironment(expanded)
	if err != nil {
		return NormalizedPath{}, err
	}

	clean := filepath.Clean(expanded)
	wasRelative := !filepath.IsAbs(clean)
	normalized := clean
	if options.MakeAbsolute {
		normalized, err = filepath.Abs(clean)
	}
	if err != nil {
		return NormalizedPath{}, errors.New(
			errors.ExitInput,
			"PATH_NORMALIZATION_FAILED",
			"failed to resolve absolute path",
			path,
			"",
			err,
		)
	}

	if options.ResolveSymlinks {
		if evaluated, evalErr := filepath.EvalSymlinks(normalized); evalErr == nil {
			normalized = evaluated
		}
	}

	return NormalizedPath{
		Supplied:    supplied,
		Expanded:    expanded,
		Path:        normalized,
		WasRelative: wasRelative,
	}, nil
}

func expandWindowsEnvironment(path string) (string, error) {
	var unresolved string
	expanded := windowsEnvironmentVariable.ReplaceAllStringFunc(path, func(token string) string {
		name := token[1 : len(token)-1]
		value, ok := os.LookupEnv(name)
		if !ok {
			unresolved = token
			return token
		}
		return value
	})
	if unresolved != "" {
		return "", errors.New(
			errors.ExitInput,
			"PATH_ENVVAR_UNRESOLVED",
			"path contains an unresolved environment variable",
			unresolved,
			"Define the variable or remove it from the path",
			nil,
		)
	}
	return expanded, nil
}

func expandPortableEnvironment(path string) (string, error) {
	var unresolved string
	expanded := os.Expand(path, func(name string) string {
		value, ok := os.LookupEnv(name)
		if !ok && unresolved == "" {
			unresolved = name
		}
		return value
	})
	if unresolved != "" {
		return "", errors.New(errors.ExitInput, "PATH_ENVVAR_UNRESOLVED", "path contains an unresolved environment variable", "$"+unresolved, "Define the variable or remove it from the path", nil)
	}
	return expanded, nil
}

// Expand expands a glob pattern and returns deterministic, deduplicated matches.
// If the glob does not match anything, it returns an error.
func Expand(pattern string) ([]string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, errors.New(
			errors.ExitInput,
			"GLOB_EXPANSION_FAILED",
			"invalid glob pattern",
			pattern,
			"Check the pattern syntax",
			err,
		)
	}

	if len(matches) == 0 {
		return nil, errors.New(
			errors.ExitInput,
			"GLOB_NO_MATCH",
			"glob did not match any files",
			pattern,
			"Ensure the files exist and the pattern is correct",
			nil,
		)
	}

	// Deduplicate matches
	seen := make(map[string]bool)
	var deduped []string
	for _, m := range matches {
		if !seen[m] {
			seen[m] = true
			deduped = append(deduped, m)
		}
	}

	return deduped, nil
}

// SafeWrite atomically replaces a file, creating a backup first if it exists.
func SafeWrite(targetPath string, data []byte, skipBackup bool) error {
	norm, err := Normalize(targetPath)
	if err != nil {
		return err
	}

	info, err := os.Lstat(norm)
	if err == nil {
		// File exists
		if !info.Mode().IsRegular() {
			return errors.New(
				errors.ExitProcessing,
				"PATH_UNSUPPORTED",
				"target is not a regular file (may be a symlink or directory)",
				norm,
				"Provide a regular file path",
				nil,
			)
		}

		if !skipBackup {
			backupPath := norm + ".bak"
			if _, err := os.Stat(backupPath); err == nil {
				return errors.New(
					errors.ExitProcessing,
					"BACKUP_EXISTS",
					"backup file already exists",
					backupPath,
					"Remove or rename the existing backup",
					nil,
				)
			}

			originalData, err := os.ReadFile(norm)
			if err != nil {
				return errors.New(
					errors.ExitProcessing,
					"BACKUP_CREATE_FAILED",
					"failed to read original file for backup",
					norm,
					"",
					err,
				)
			}

			err = os.WriteFile(backupPath, originalData, info.Mode().Perm())
			if err != nil {
				return errors.New(
					errors.ExitProcessing,
					"BACKUP_CREATE_FAILED",
					"failed to write backup file",
					backupPath,
					"",
					err,
				)
			}
		}
	} else if !os.IsNotExist(err) {
		return errors.New(
			errors.ExitProcessing,
			"PATH_UNREADABLE",
			"unable to check target file",
			norm,
			"",
			err,
		)
	}

	tmpPath := norm + ".tmp"
	err = os.WriteFile(tmpPath, data, 0644)
	if err != nil {
		return errors.New(
			errors.ExitProcessing,
			"OUTPUT_WRITE_FAILED",
			"failed to write temporary file",
			tmpPath,
			"",
			err,
		)
	}

	err = os.Rename(tmpPath, norm)
	if err != nil {
		if cleanupErr := os.Remove(tmpPath); cleanupErr != nil {
			err = fmt.Errorf("%w; temporary-file cleanup also failed: %v", err, cleanupErr)
		}
		return errors.New(
			errors.ExitProcessing,
			"ATOMIC_REPLACE_FAILED",
			"failed to replace target with new data",
			norm,
			"",
			err,
		)
	}

	return nil
}
