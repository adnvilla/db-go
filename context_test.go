package dbgo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestSetAndGetFromContext(t *testing.T) {
	origConn := conn
	defer func() { conn = origConn }()
	conn = DBConn{}

	db := &gorm.DB{}
	ctx := SetFromContext(context.Background(), db)
	result := GetFromContext(ctx)

	assert.Equal(t, db, result)
}

func TestGetFromContext_FallsBackToGlobalConn(t *testing.T) {
	origConn := conn
	defer func() { conn = origConn }()

	globalDB := &gorm.DB{}
	conn = DBConn{Instance: globalDB}

	result := GetFromContext(context.Background())
	assert.Equal(t, globalDB, result)
}

func TestGetFromContext_ReturnsNilWhenNothingAvailable(t *testing.T) {
	origConn := conn
	defer func() { conn = origConn }()
	conn = DBConn{}

	result := GetFromContext(context.Background())
	assert.Nil(t, result)
}

func TestGetFromContext_ContextOverridesGlobal(t *testing.T) {
	origConn := conn
	defer func() { conn = origConn }()

	globalDB := &gorm.DB{Config: &gorm.Config{SkipDefaultTransaction: true}}
	conn = DBConn{Instance: globalDB}

	contextDB := &gorm.DB{Config: &gorm.Config{SkipDefaultTransaction: false}}
	ctx := SetFromContext(context.Background(), contextDB)

	result := GetFromContext(ctx)
	assert.Same(t, contextDB, result, "context DB should take precedence over global")
	assert.NotSame(t, globalDB, result)
}

func TestSetFromContext_PreservesExistingValues(t *testing.T) {
	type otherKey struct{}
	ctx := context.WithValue(context.Background(), otherKey{}, "existing-value")

	db := &gorm.DB{}
	ctx = SetFromContext(ctx, db)

	assert.Equal(t, "existing-value", ctx.Value(otherKey{}))
	assert.Equal(t, db, GetFromContext(ctx))
}
