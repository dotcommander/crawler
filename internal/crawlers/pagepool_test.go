package crawlers

import (
	"testing"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
)

func TestPagePoolReleaseClosesPageAfterPoolClose(t *testing.T) {
	t.Parallel()

	closed := false
	pool := &PagePool{
		closed: true,
		closePage: func(playwright.Page) {
			closed = true
		},
	}

	pool.Release(nil)

	assert.True(t, closed)
}
