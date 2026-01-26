package dns

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSame_Ints(t *testing.T) {
	assert.True(t, Same([]int{1, 2, 3}, []int{3, 2, 1}))
	assert.True(t, Same([]int{1, 1, 2}, []int{2, 1, 1}))
	assert.False(t, Same([]int{1, 2, 3}, []int{1, 2}))
	assert.False(t, Same([]int{1, 2, 2}, []int{2, 1, 1}))
	assert.True(t, Same([]int{}, []int{}))
}

func TestSame_Strings(t *testing.T) {
	assert.True(t, Same([]string{"a", "b", "c"}, []string{"c", "a", "b"}))
	assert.False(t, Same([]string{"a", "b"}, []string{"a", "b", "b"}))
	assert.True(t, Same([]string{"x", "x", "y"}, []string{"y", "x", "x"}))
	assert.False(t, Same([]string{"a"}, []string{"b"}))
}

func TestSame_Bool(t *testing.T) {
	assert.True(t, Same([]bool{true, false, true}, []bool{false, true, true}))
	assert.False(t, Same([]bool{true, false}, []bool{true, true}))
}
