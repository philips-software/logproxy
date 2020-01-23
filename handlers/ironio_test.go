package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIronToRFC5424(t *testing.T) {
	testPayload := "severity=INFO, task_id: 5e299d0af210cc00097e9883, code_name: loafoe/iron-test, project_id: 5e20da41d748ad000ace7654 -- This is a message"
	now := time.Unix(1405544146, 0)

	rfc := ironToRFC5424(now, testPayload)

	assert.Equal(t, "<14>1 2014-07-16T22:55:46+02:00 - - - - - severity=INFO, task_id: 5e299d0af210cc00097e9883, code_name: loafoe/iron-test, project_id: 5e20da41d748ad000ace7654 -- This is a message", rfc)
}
