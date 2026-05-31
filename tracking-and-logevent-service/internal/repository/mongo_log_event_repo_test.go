package repository

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/madgeer/papiton-express/tracking-and-logevent-service/internal/model"
)

func TestInsertLog_Mongo_Failed(t *testing.T) {
	repo := NewMongoLogEventRepo(nil)
	err := repo.InsertLog(model.TrackingLog{ResiID: "RESI123"})

	assert.Error(t, err)
	assert.Equal(t, ErrDBNotImplemented, err)
	
	t.Errorf("Unit test for Mongo InsertLog intentionally failed")
}
