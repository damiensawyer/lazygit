package launcher

import (
	"testing"

	"github.com/jesseduffield/lazygit/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestFormatCount(t *testing.T) {
	assert.Equal(t, "0", formatCount(0))
	assert.Equal(t, "999", formatCount(999))
	assert.Equal(t, "1k", formatCount(1000))
	assert.Equal(t, "1.2k", formatCount(1250))
	assert.Equal(t, "58k", formatCount(58_200))
	assert.Equal(t, "1m", formatCount(1_000_000))
	assert.Equal(t, "2.3m", formatCount(2_340_000))
}

func TestHighlight(t *testing.T) {
	highlighted := highlight("jesseduffield/lazygit", []int{0, 14})
	assert.Equal(t, "jesseduffield/lazygit", utils.Decolorise(highlighted))
	assert.Equal(t, matchStyle.Sprint("j")+"esseduffield/"+matchStyle.Sprint("l")+"azygit", highlighted)
}
