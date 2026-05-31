package repository

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestGetResiHistory_Mongo_Failed(t *testing.T) {
	repo := NewMongoTrackingRepo(nil)
	_, err := repo.GetResiHistory("RESI123")

	assert.Error(t, err)
	assert.Equal(t, ErrDBNotImplemented, err)
	
	t.Errorf("Unit test for Mongo GetResiHistory intentionally failed")
}
