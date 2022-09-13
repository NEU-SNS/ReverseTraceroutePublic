package plcontroller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendAlert(t *testing.T) {
	err := SendEmail([] string {"kevinmylinh.vermeulen@gmail.com"}, "Test", "This is a test message")
	assert.Equal(t, err, nil)
}