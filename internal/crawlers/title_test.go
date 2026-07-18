package crawlers

import (
	"errors"
	"testing"

	"github.com/go-rod/rod"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePlaywrightTitleReader struct {
	title string
	err   error
}

func (f fakePlaywrightTitleReader) Title() (string, error) {
	return f.title, f.err
}

type missingRodTitleFinder struct{}

func (missingRodTitleFinder) Has(string) (bool, *rod.Element, error) {
	return false, nil, nil
}

type failingRodTitleFinder struct{}

func (failingRodTitleFinder) Has(string) (bool, *rod.Element, error) {
	return false, nil, errors.New("title lookup failed")
}

func TestSetPlaywrightTitlePopulatesCrawlResult(t *testing.T) {
	t.Parallel()

	result := &CrawlResult{}
	err := setPlaywrightTitle(fakePlaywrightTitleReader{title: "Example title"}, result)
	require.NoError(t, err)
	assert.Equal(t, "Example title", result.Title)
}

func TestReadRodTitleReturnsImmediatelyWhenMissing(t *testing.T) {
	t.Parallel()

	title, err := readRodTitle(missingRodTitleFinder{})
	require.NoError(t, err)
	assert.Empty(t, title)
}

func TestSetPlaywrightTitleLeavesResultUsableOnError(t *testing.T) {
	t.Parallel()

	result := &CrawlResult{Success: true}
	err := setPlaywrightTitle(fakePlaywrightTitleReader{err: errors.New("title lookup failed")}, result)
	require.Error(t, err)
	assert.True(t, result.Success)
	assert.Empty(t, result.Title)
}

func TestSetRodTitleLeavesResultUsableOnError(t *testing.T) {
	t.Parallel()

	result := &CrawlResult{Success: true}
	setRodTitle(failingRodTitleFinder{}, result)
	assert.True(t, result.Success)
	assert.Empty(t, result.Title)
}
