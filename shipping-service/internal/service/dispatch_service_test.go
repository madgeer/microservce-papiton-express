package service_test

import (
	"context"
	"testing"

	"github.com/madgeer/papiton-express/shipping-service/internal/service"
)

func TestAutoDispatchPickUp_ShouldReturnCourier(t *testing.T) {

	dispatchSvc := service.NewDispatchService(nil, nil, nil, nil)
	ctx := context.Background()

	result, err := dispatchSvc.AutoDispatchPickUp(
		ctx,
		"ORD-001",
		"Bandung",
	)

	if err != nil {
		t.Fatalf("tidak boleh error")
	}

	if result == nil {
		t.Fatalf("dispatch tidak boleh nil")
	}
}