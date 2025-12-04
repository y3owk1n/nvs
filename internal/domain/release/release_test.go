package release_test

import (
	"testing"
	"time"

	"github.com/y3owk1n/nvs/internal/domain/release"
)

func TestNewRelease(t *testing.T) {
	publishedAt := time.Date(2024, 12, 4, 10, 0, 0, 0, time.UTC)
	assets := []release.Asset{
		release.NewAsset("nvim-linux64.tar.gz", "https://example.com/nvim.tar.gz", 12345678),
	}

	r := release.New("v0.10.0", false, "abc123", publishedAt, assets)

	if got := r.TagName(); got != "v0.10.0" {
		t.Errorf("TagName() = %v, want v0.10.0", got)
	}

	if got := r.Prerelease(); got != false {
		t.Errorf("Prerelease() = %v, want false", got)
	}

	if got := r.CommitHash(); got != "abc123" {
		t.Errorf("CommitHash() = %v, want abc123", got)
	}

	if got := r.PublishedAt(); !got.Equal(publishedAt) {
		t.Errorf("PublishedAt() = %v, want %v", got, publishedAt)
	}

	if got := r.Assets(); len(got) != 1 {
		t.Errorf("Assets() length = %v, want 1", len(got))
	}
}

func TestNewAsset(t *testing.T) {
	a := release.NewAsset("nvim-linux64.tar.gz", "https://example.com/nvim.tar.gz", 12345678)

	if got := a.Name(); got != "nvim-linux64.tar.gz" {
		t.Errorf("Name() = %v, want nvim-linux64.tar.gz", got)
	}

	if got := a.DownloadURL(); got != "https://example.com/nvim.tar.gz" {
		t.Errorf("DownloadURL() = %v, want https://example.com/nvim.tar.gz", got)
	}

	if got := a.Size(); got != 12345678 {
		t.Errorf("Size() = %v, want 12345678", got)
	}
}
