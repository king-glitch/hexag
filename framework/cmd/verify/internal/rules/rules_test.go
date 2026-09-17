package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifier_TemplatePasses(t *testing.T) {
	// Verify that the template directory in hexag passes all rules cleanly
	templateDir, err := filepath.Abs("../../../../../template/internal")
	require.NoError(t, err)

	verifier := NewVerifier()
	err = verifier.VerifyPath(templateDir)
	require.NoError(t, err)

	for _, v := range verifier.Violations() {
		t.Logf("Unexpected violation in template: %s", v.String())
	}
	assert.Empty(t, verifier.Violations(), "template/internal must have 0 violations")
	assert.Greater(t, verifier.FileCount(), 0, "should have checked template files")
}

func TestVerifier_FileNames(t *testing.T) {
	tmpDir := t.TempDir()
	internalDir := filepath.Join(tmpDir, "internal", "services", "user")
	require.NoError(t, os.MkdirAll(internalDir, 0755))

	// Underscore in non-test file
	badFile1 := filepath.Join(internalDir, "user_service.go")
	require.NoError(t, os.WriteFile(badFile1, []byte("package user\n"), 0644))

	// Kebab-case with role suffix
	badFile2 := filepath.Join(internalDir, "user-handler.go")
	require.NoError(t, os.WriteFile(badFile2, []byte("package user\n"), 0644))

	// Uppercase filename
	badFile3 := filepath.Join(internalDir, "User.go")
	require.NoError(t, os.WriteFile(badFile3, []byte("package user\n"), 0644))

	// Allowed test file
	goodTest := filepath.Join(internalDir, "service_test.go")
	require.NoError(t, os.WriteFile(goodTest, []byte("package user\n"), 0644))

	// Allowed role file
	goodFile := filepath.Join(internalDir, "service.go")
	require.NoError(t, os.WriteFile(goodFile, []byte("package user\n"), 0644))

	verifier := NewVerifier()
	err := verifier.VerifyPath(tmpDir)
	require.NoError(t, err)

	violations := verifier.Violations()
	assert.Len(t, violations, 3)

	cats := make([]string, len(violations))
	for i, v := range violations {
		cats[i] = v.Category
	}
	assert.Equal(t, []string{"File Names", "File Names", "File Names"}, cats)
}

func TestVerifier_ServiceStructAndInjections(t *testing.T) {
	tmpDir := t.TempDir()
	serviceDir := filepath.Join(tmpDir, "internal", "services", "order")
	require.NoError(t, os.MkdirAll(serviceDir, 0755))

	content := `package order

import (
	"context"
	servicebase "github.com/king-glitch/hexag/framework/api/service/base"
	hexports "github.com/king-glitch/hexag/framework/ports"
	"example/internal/ports"
)

type Service struct {
	servicebase.ServiceBase
	userRepo     ports.UserRepository
	orderRepo    ports.OrderRepository
	userService  ports.UserService
	txRunner     hexports.TransactionRunner
}

func (s Service) WithUser(u ports.UserService) Service {
	return s
}

func (s Service) DoSomething(ctx context.Context) error {
	if s.us == nil {
		return nil
	}
	return nil
}
`
	filePath := filepath.Join(serviceDir, "service.go")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	verifier := NewVerifier()
	err := verifier.VerifyPath(tmpDir)
	require.NoError(t, err)

	violations := verifier.Violations()
	// Violations expected:
	// 1. Storing TransactionRunner on service struct
	// 2. Repository named 'userRepo' instead of 'repository'
	// 3. Repository named 'orderRepo' instead of 'repository'
	// 4. Sibling service named 'userService' instead of 'us'
	// 5. Multiple repositories on service struct
	// 6. Setter injection WithUser
	// 7. Nil check on s.us
	assert.GreaterOrEqual(t, len(violations), 5)
}

func TestVerifier_Collections(t *testing.T) {
	tmpDir := t.TempDir()
	dbDir := filepath.Join(tmpDir, "internal", "adapters", "database", "mongo", "user")
	require.NoError(t, os.MkdirAll(dbDir, 0755))

	content := `package user

import (
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const CollectionName = "users"

func New(db *mongo.Database) {
	col := db.Collection("users")
	_ = col
}
`
	filePath := filepath.Join(dbDir, "repository.go")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	verifier := NewVerifier()
	err := verifier.VerifyPath(tmpDir)
	require.NoError(t, err)

	violations := verifier.Violations()
	// Expected:
	// 1. Hardcoded collection name string "users"
	// 2. CollectionName constant definition
	assert.GreaterOrEqual(t, len(violations), 2)
}

func TestVerifier_Sentinels(t *testing.T) {
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "internal", "services", "user")
	require.NoError(t, os.MkdirAll(pkgDir, 0755))

	content := `package user

import (
	"errors"
	"example/internal/ports"
)

func Check(err error) {
	if err == ports.ErrNotFound {
		return
	}
	if err != ports.ErrUserExists {
		return
	}
	switch err {
	case ports.ErrNotFound:
		return
	}
}
`
	filePath := filepath.Join(pkgDir, "service.go")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	verifier := NewVerifier()
	err := verifier.VerifyPath(tmpDir)
	require.NoError(t, err)

	violations := verifier.Violations()
	// Expected:
	// 1. err == ports.ErrNotFound
	// 2. err != ports.ErrUserExists
	// 3. switch err { case ports.ErrNotFound: }
	assert.Len(t, violations, 3)
}

func TestVerifier_HTTPHandlers(t *testing.T) {
	tmpDir := t.TempDir()
	endpointDir := filepath.Join(tmpDir, "internal", "adapters", "endpoint", "fiber", "routes")
	require.NoError(t, os.MkdirAll(endpointDir, 0755))

	content := `package routes

import (
	"time"
	"github.com/gofiber/fiber/v3"
)

type EmptyResponse struct{}

type UserHandler struct{}

func (h UserHandler) unexported(c fiber.Ctx) error {
	_ = time.Now()
	page := c.Query("page")
	_ = page
	return nil
}
`
	filePath := filepath.Join(endpointDir, "user.go")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	verifier := NewVerifier()
	err := verifier.VerifyPath(tmpDir)
	require.NoError(t, err)

	violations := verifier.Violations()
	// Expected:
	// 1. EmptyResponse struct
	// 2. unexported handler method
	// 3. time.Now() in handler
	// 4. c.Query("page")
	assert.Len(t, violations, 4)
}
