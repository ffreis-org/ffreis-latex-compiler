// Package promote stages a compiled article into a posts-repo checkout: it
// copies dist/<slug>/index.md and images/ into <posts>/posts/<post-slug>/ and
// re-runs the post validation so a broken post is never left behind. Opening a
// pull request is layered on top in the command (promotecmd); this core only
// touches the local filesystem.
package promote

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/article"
	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/fsutil"
	"github.com/FelipeFuhr/ffreis-latex-compiler/internal/posts"
)

// indexMD is the filename of the compiled Markdown artifact that promote copies
// into the posts repo.
const indexMD = "index.md"

// Options configures a promote run.
type Options struct {
	ArticlesRoot string // to resolve the article's post_slug
	OutRoot      string // build output root (dist)
	PostsDir     string // posts-repo checkout root (contains posts/)
	Slug         string // article slug (directory under dist/ and articles/)
	DryRun       bool   // when true, do not write — only report what would happen
}

// Outcome reports what a promote run did (or would do, for a dry run).
type Outcome struct {
	PostSlug    string
	TargetDir   string   // <posts>/posts/<post-slug>
	CopiedFiles []string // relative to TargetDir
	Validation  posts.Result
}

// Run stages the compiled article into the posts checkout. It returns an error
// for missing inputs or when the staged post fails validation.
func Run(opts Options) (*Outcome, error) {
	a, err := article.Load(opts.ArticlesRoot, opts.Slug)
	if err != nil {
		return nil, err
	}
	postSlug := a.PostSlug()

	srcDir := filepath.Join(opts.OutRoot, opts.Slug)
	srcIndex := filepath.Join(srcDir, indexMD)
	if _, err := os.Stat(srcIndex); err != nil {
		return nil, fmt.Errorf("no compiled Markdown for %q — run build with -formats md first: %w", opts.Slug, err)
	}

	postsBase := filepath.Join(opts.PostsDir, "posts")
	targetDir := filepath.Join(postsBase, postSlug)

	out := &Outcome{PostSlug: postSlug, TargetDir: targetDir}

	if opts.DryRun {
		out.CopiedFiles = plannedFiles(srcDir)
		return out, nil
	}

	if err := os.MkdirAll(targetDir, 0o750); err != nil {
		return nil, err
	}
	if err := fsutil.CopyFile(srcIndex, filepath.Join(targetDir, indexMD)); err != nil {
		return nil, err
	}
	if err := fsutil.CopyDir(filepath.Join(srcDir, "images"), filepath.Join(targetDir, "images")); err != nil {
		return nil, err
	}
	out.CopiedFiles = plannedFiles(srcDir)

	res := posts.Validate(postsBase, postSlug)
	out.Validation = res
	if !res.OK() {
		return out, fmt.Errorf("staged post %q failed validation (%d error(s))", postSlug, len(res.Errors))
	}
	return out, nil
}

// plannedFiles lists the artifacts promote copies, relative to the target dir.
func plannedFiles(srcDir string) []string {
	files := []string{indexMD}
	imagesDir := filepath.Join(srcDir, "images")
	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		return files
	}
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, filepath.Join("images", e.Name()))
		}
	}
	return files
}
