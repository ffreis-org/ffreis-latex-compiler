// Package article loads LaTeX article sources from a articles-style
// repository. Each article lives in articles/<slug>/ and is described by a
// sidecar meta.yaml whose schema is deliberately a superset of the
// the posts repo frontmatter schema, so a compiled article can be promoted into
// the blog without reshaping its metadata.
package article

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// MainTeXName is the entrypoint file every article must provide.
const MainTeXName = "main.tex"

// MetaName is the sidecar metadata file every article must provide.
const MetaName = "meta.yaml"

// ImagesDir is the per-article directory holding figures referenced from the
// LaTeX source and copied alongside the generated post.
const ImagesDir = "images"

// errArticleFmt is the shared format string used when wrapping errors that
// identify an article by slug.
const errArticleFmt = "article %q: %w"

// Meta is the parsed contents of an article's meta.yaml. Title and Date are
// required; everything else is optional. Field names mirror the the posts repo
// frontmatter so promotion is a straight copy.
type Meta struct {
	Title              string   `yaml:"title"`
	Date               string   `yaml:"date"`
	Slug               string   `yaml:"slug"`
	Summary            string   `yaml:"summary"`
	Tags               []string `yaml:"tags"`
	CanonicalURL       string   `yaml:"canonical_url"`
	Thumbnail          string   `yaml:"thumbnail"`
	AvailableLanguages []string `yaml:"available_languages"`
	// PostSlug optionally overrides the slug used when promoting to
	// the posts repo. When empty the article's directory slug is used.
	PostSlug string `yaml:"post_slug"`
}

// Article is a fully resolved article source on disk.
type Article struct {
	Slug    string // directory name (canonical slug)
	Dir     string // absolute path to articles/<slug>/
	MainTeX string // absolute path to main.tex
	Meta    Meta
	Images  []string // basenames present under images/ (sorted)
}

// PostSlug returns the slug to use when promoting to the posts repo.
func (a *Article) PostSlug() string {
	if a.Meta.PostSlug != "" {
		return a.Meta.PostSlug
	}
	return a.Slug
}

// Load reads and parses a single article at articlesRoot/articles/<slug>/.
func Load(articlesRoot, slug string) (*Article, error) {
	dir := filepath.Join(articlesRoot, "articles", slug)
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf(errArticleFmt, slug, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("article %q: %s is not a directory", slug, dir)
	}

	mainTeX := filepath.Join(dir, MainTeXName)
	if _, err := os.Stat(mainTeX); err != nil {
		return nil, fmt.Errorf("article %q: %s: %w", slug, MainTeXName, err)
	}

	meta, err := loadMeta(filepath.Join(dir, MetaName))
	if err != nil {
		return nil, fmt.Errorf(errArticleFmt, slug, err)
	}

	images, err := listImages(filepath.Join(dir, ImagesDir))
	if err != nil {
		return nil, fmt.Errorf(errArticleFmt, slug, err)
	}

	return &Article{
		Slug:    slug,
		Dir:     dir,
		MainTeX: mainTeX,
		Meta:    meta,
		Images:  images,
	}, nil
}

// LoadAll reads every article under articlesRoot/articles/, sorted by slug.
func LoadAll(articlesRoot string) ([]*Article, error) {
	base := filepath.Join(articlesRoot, "articles")
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, fmt.Errorf("read articles dir: %w", err)
	}
	var slugs []string
	for _, e := range entries {
		if e.IsDir() {
			slugs = append(slugs, e.Name())
		}
	}
	sort.Strings(slugs)

	articles := make([]*Article, 0, len(slugs))
	for _, slug := range slugs {
		a, err := Load(articlesRoot, slug)
		if err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}
	return articles, nil
}

func loadMeta(path string) (Meta, error) {
	var m Meta
	raw, err := os.ReadFile(path) //nolint:gosec // path is derived from a trusted articles root
	if err != nil {
		return m, fmt.Errorf("%s: %w", MetaName, err)
	}
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return m, fmt.Errorf("%s: invalid YAML: %w", MetaName, err)
	}
	return m, nil
}

func listImages(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // images/ is optional
		}
		return nil, fmt.Errorf("read %s: %w", ImagesDir, err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}
