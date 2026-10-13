package publication

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Publish runs one generation-atomic publication into outRoot. The compiler
// and Finalize both write into a fresh staging sibling of outRoot — seeded as
// a hardlink clone of the currently published generation when one exists, so
// manifest cleanup sees the previous generation and co-located non-manifest
// files (for example a shared SPA build) survive promotion. Only after the
// complete generation validates is it promoted onto outRoot; every failure
// before or during promotion leaves the previous generation byte-for-byte
// untouched. On success the promoted generation's manifest drives stale-content
// removal as before (performed inside staging before the swap).
//
// Promotion swaps directory entries: the previous generation is moved aside,
// staging takes its place, and the previous generation is deleted only after
// the swap succeeds. If the second rename fails the previous generation is
// moved back, so a failed promotion never leaves outRoot missing or partial.
func Publish(ctx context.Context, compiler Compiler, input Input, read ReadModel, media MediaSource, outRoot string) (BuildReport, error) {
	outRoot = filepath.Clean(outRoot)
	parent := filepath.Dir(outRoot)
	staging, err := os.MkdirTemp(parent, "."+filepath.Base(outRoot)+".staging-")
	if err != nil {
		return BuildReport{}, fmt.Errorf("create staging directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(staging) }()

	if err := cloneGeneration(outRoot, staging); err != nil {
		return BuildReport{}, fmt.Errorf("stage previous generation: %w", err)
	}

	writer := &FileArtifactWriter{Root: staging}
	report, err := compiler.Compile(ctx, input, read, media, writer)
	if err != nil {
		_ = writer.Abort()
		return report, err
	}
	// Validate and reconcile against the previous generation's manifest while
	// still inside staging; stale paths are removed from the staged hardlink
	// clone, never from the live generation.
	removed, err := writer.Finalize()
	if err != nil {
		_ = writer.Abort()
		return report, fmt.Errorf("finalize staged generation: %w", err)
	}
	report.Removed = len(removed)

	if err := promote(staging, outRoot); err != nil {
		return report, fmt.Errorf("promote staged generation: %w", err)
	}
	return report, nil
}

// cloneGeneration hardlinks every regular file under src into dst (recursing
// through directories, skipping other file types). Linked files share inodes,
// but every write path is temp-file-plus-rename and removals only unlink, so
// the source generation is never mutated. A missing source is the first
// publish: dst stays an empty staging root.
func cloneGeneration(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) && path == src {
				return nil
			}
			return err
		}
		relative, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, relative)
		switch {
		case entry.IsDir():
			return os.MkdirAll(target, 0o755)
		case entry.Type().IsRegular():
			return os.Link(path, target)
		default:
			return nil
		}
	})
}

// promote swaps the staged generation into outRoot. The previous generation
// (if any) is moved to a sibling holding name and only removed once the
// staging directory has taken its place; a failed second rename is rolled
// back by moving the previous generation home.
func promote(staging, outRoot string) error {
	if _, err := os.Lstat(outRoot); os.IsNotExist(err) {
		return os.Rename(staging, outRoot)
	}
	previous := staging + ".previous"
	if err := os.Rename(outRoot, previous); err != nil {
		return fmt.Errorf("set aside previous generation: %w", err)
	}
	if err := os.Rename(staging, outRoot); err != nil {
		if restore := os.Rename(previous, outRoot); restore != nil {
			return fmt.Errorf("promote staged generation: %w (restoring previous generation also failed: %w)", err, restore)
		}
		return err
	}
	return os.RemoveAll(previous)
}

// stagingPrefix marks the staging (and failed-promotion previous-generation)
// siblings Publish creates next to an output root; they are always removed on
// completion.
func stagingPrefix(outputRoot string) string {
	return "." + filepath.Base(filepath.Clean(outputRoot)) + ".staging-"
}
