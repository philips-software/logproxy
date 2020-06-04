package proto

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

type testServer struct {
}

func TestInternals(t *testing.T) {
	s, err := _Filter_Filter_Handler(&testServer{}, context.Background(), func(i interface{}) error {
		return nil
	}, interc)
	assert.Nil(t, err)
	assert.Nil(t, s)

}

func interc(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	return nil, nil
}
