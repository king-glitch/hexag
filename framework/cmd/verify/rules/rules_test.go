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
	templateDir, err := filepath.Abs("../../../../template/internal")
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

func TestVerifier_Enums(t *testing.T) {
	tmpDir := t.TempDir()
	portsDir := filepath.Join(tmpDir, "internal", "ports")
	require.NoError(t, os.MkdirAll(portsDir, 0755))

	content := `package ports

type Status string

const (
	StatusActive  Status = "active"
	StatusPending Status = "pending"
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
	RoleGuest Role = "guest"
)

func (r Role) IsValid() bool {
	return r == RoleAdmin || r == RoleUser
}
`
	filePath := filepath.Join(portsDir, "enum.go")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	verifier := NewVerifier()
	err := verifier.VerifyPath(tmpDir)
	require.NoError(t, err)

	violations := verifier.Violations()
	// Expected:
	// 1. Status does NOT implement IsValid() bool -> violation!
	// 2. Role implements IsValid() bool, but misses RoleGuest -> violation!
	require.Len(t, violations, 2)
	assert.Equal(t, "Enums", violations[0].Category)
	assert.Equal(t, "Enums", violations[1].Category)
	descJoined := violations[0].Description + " " + violations[1].Description
	assert.Contains(t, descJoined, "Status")
	assert.Contains(t, descJoined, "RoleGuest")
}

func TestVerifier_OneofValidation(t *testing.T) {
	tmpDir := t.TempDir()
	routesDir := filepath.Join(tmpDir, "internal", "adapters", "endpoint", "fiber", "routes")
	require.NoError(t, os.MkdirAll(routesDir, 0755))

	content := `package routes

type CreateUserRequest struct {
	Name   string ` + "`json:\"name\" validate:\"required\"`" + `
	Status string ` + "`json:\"status\" validate:\"required,oneof=active pending\"`" + `
}
`
	filePath := filepath.Join(routesDir, "user.go")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	verifier := NewVerifier()
	err := verifier.VerifyPath(tmpDir)
	require.NoError(t, err)

	violations := verifier.Violations()
	// Expected:
	// oneof= in request struct -> violation!
	require.Len(t, violations, 1)
	assert.Equal(t, "Validation", violations[0].Category)
	assert.Contains(t, violations[0].Description, "Status")
	assert.Contains(t, violations[0].Description, "oneof=")
}

func TestVerifier_TestFileExclusion(t *testing.T) {
	tmpDir := t.TempDir()
	pkgDir := filepath.Join(tmpDir, "internal", "services", "user")
	require.NoError(t, os.MkdirAll(pkgDir, 0755))

	// Test files (ending in _test.go) are excluded from verification
	testFile := filepath.Join(pkgDir, "live_party_test.go")
	require.NoError(t, os.WriteFile(testFile, []byte("package user\n"), 0644))

	verifier := NewVerifier()
	err := verifier.VerifyPath(tmpDir)
	require.NoError(t, err)

	assert.Empty(t, verifier.Violations(), "test files should be excluded from verification")
}

func TestVerifier_DepsStructAndFieldAccess(t *testing.T) {
	tmpDir := t.TempDir()
	runnerDir := filepath.Join(tmpDir, "internal", "runner")
	require.NoError(t, os.MkdirAll(runnerDir, 0755))

	content := `package runner

import (
	"context"
	"time"
	"example/internal/ports"
)

type Deps struct {
	AccountService       ports.GameAccountService
	CreditService        ports.CreditService
	ConnectionRepository ports.BotConnectionRepository
	RunRepository        ports.BotRunRepository
	ClientVersion        string
	HeartbeatEvery       time.Duration
}

type Runner struct {
	deps Deps
}

func (r Runner) Execute(ctx context.Context) error {
	// Direct field access to repository on s.deps
	_ = r.deps.ConnectionRepository

	// Direct field access to service on s.deps
	_ = r.deps.CreditService

	// Direct field access on local deps variable
	var d Deps
	_ = d.ConnectionRepository

	// Abbreviated repo field access
	_ = r.deps.ConnectionRepo

	// Getter method call without Get prefix
	_ = r.deps.ConnectionRepository()

	return nil
}
`
	filePath := filepath.Join(runnerDir, "runner.go")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	verifier := NewVerifier()
	err := verifier.VerifyPath(tmpDir)
	require.NoError(t, err)

	violations := verifier.Violations()
	// Violations expected:
	// 1. Deps.AccountService exported service field
	// 2. Deps.CreditService exported service field
	// 3. Deps.ConnectionRepository exported repo field
	// 4. Deps.RunRepository exported repo field
	// 5. r.deps.ConnectionRepository direct field access
	// 6. r.deps.CreditService direct field access
	// 7. d.ConnectionRepository direct field access
	// 8. r.deps.ConnectionRepo abbreviated direct field access
	// 9. r.deps.ConnectionRepository() missing Get prefix
	assert.GreaterOrEqual(t, len(violations), 8)

	foundExportedService := false
	foundExportedRepo := false
	foundDirectRepoAccess := false
	foundDirectServiceAccess := false
	foundMissingGetCall := false

	for _, v := range violations {
		if v.Category == "Sibling Services" && assert.ObjectsAreEqual(v.Contract, "Service interface fields on structs must be unexported lowercase acronyms (e.g. 'cs ports.CreditService').") {
			foundExportedService = true
		}
		if v.Category == "Dependencies" && assert.ObjectsAreEqual(v.Contract, "Repository fields on structs must be unexported; direct external access to repository struct fields is forbidden.") {
			foundExportedRepo = true
		}
		if v.Category == "Dependencies" && assert.ObjectsAreEqual(v.Contract, "Interface-based dependencies use full-word getter methods; direct struct field access to repository fields is forbidden.") {
			foundDirectRepoAccess = true
		}
		if v.Category == "Sibling Services" && assert.ObjectsAreEqual(v.Contract, "Injected sibling services must be stored in unexported acronym fields (e.g. 's.cs') on service structs or accessed via getter methods (e.g. 'GetCreditService()'); direct struct field access to service full names is forbidden.") {
			foundDirectServiceAccess = true
		}
		if v.Category == "Dependencies" && assert.ObjectsAreEqual(v.Contract, "Interface-based dependencies use full-word getter methods starting with 'Get' (e.g. GetConnectionRepository()).") {
			foundMissingGetCall = true
		}
	}

	assert.True(t, foundExportedService, "should flag exported service field in Deps")
	assert.True(t, foundExportedRepo, "should flag exported repo field in Deps")
	assert.True(t, foundDirectRepoAccess, "should flag direct repo field access")
	assert.True(t, foundDirectServiceAccess, "should flag direct service field access")
	assert.True(t, foundMissingGetCall, "should flag getter call without Get prefix")
}

func TestVerifier_ConstructorParameters(t *testing.T) {
	tmpDir := t.TempDir()
	serviceDir := filepath.Join(tmpDir, "internal", "services", "user")
	require.NoError(t, os.MkdirAll(serviceDir, 0755))

	content := `package user

import (
	"example/internal/ports"
)

type Service struct {
	repository ports.UserRepository
	cs         ports.CreditService
}

func NewService(
	ctx ports.ServiceContext,
	repo ports.UserRepository,
	creditService ports.CreditService,
) ports.UserService {
	return Service{
		repository: repo,
		cs:         creditService,
	}
}
`
	filePath := filepath.Join(serviceDir, "service.go")
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0644))

	verifier := NewVerifier()
	err := verifier.VerifyPath(tmpDir)
	require.NoError(t, err)

	violations := verifier.Violations()
	// Expected:
	// 1. Parameter 'repo' instead of 'repository'
	// 2. Parameter 'creditService' instead of 'cs'
	require.Len(t, violations, 2)
	assert.Contains(t, violations[0].Description, "repo")
	assert.Contains(t, violations[1].Description, "creditService")
}

func TestVerifier_HandlerServiceAcronym(t *testing.T) {
	tmpDir := t.TempDir()
	routesDir := filepath.Join(tmpDir, "internal", "adapters", "endpoint", "fiber", "routes")
	require.NoError(t, os.MkdirAll(routesDir, 0755))

	// Invalid handler: uses 'connectionService' field instead of 'bcs' and calls 'h.connectionService'
	invalidContent := `package routes

import (
	"example/internal/ports"
	"github.com/gofiber/fiber/v3"
)

type ConnectionHandler struct {
	connectionService ports.BotConnectionService
}

func NewConnectionHandler(connectionService ports.BotConnectionService) ConnectionHandler {
	return ConnectionHandler{connectionService: connectionService}
}

func (h ConnectionHandler) Connect(c fiber.Ctx) error {
	return h.connectionService.Connect(c.RequestCtx())
}
`
	filePath := filepath.Join(routesDir, "connection.go")
	require.NoError(t, os.WriteFile(filePath, []byte(invalidContent), 0644))

	verifier := NewVerifier()
	err := verifier.VerifyPath(tmpDir)
	require.NoError(t, err)

	violations := verifier.Violations()
	// Expected:
	// 1. Service field 'connectionService' on struct 'ConnectionHandler' must be acronym 'bcs'
	// 2. Direct struct field access to service 'connectionService' is forbidden
	require.Len(t, violations, 2)
	assert.Equal(t, "Sibling Services", violations[0].Category)
	assert.Contains(t, violations[0].Description, "connectionService")
	assert.Contains(t, violations[0].Description, "bcs")
	assert.Equal(t, "Sibling Services", violations[1].Category)
	assert.Contains(t, violations[1].Description, "connectionService")

	// Now test valid handler: uses 'bcs ports.BotConnectionService' and 'h.bcs.Connect(...)'
	validContent := `package routes

import (
	"example/internal/ports"
	"github.com/gofiber/fiber/v3"
)

type ConnectionHandler struct {
	bcs ports.BotConnectionService
}

func NewConnectionHandler(bcs ports.BotConnectionService) ConnectionHandler {
	return ConnectionHandler{bcs: bcs}
}

func (h ConnectionHandler) Connect(c fiber.Ctx) error {
	return h.bcs.Connect(c.RequestCtx())
}
`
	require.NoError(t, os.WriteFile(filePath, []byte(validContent), 0644))

	verifierValid := NewVerifier()
	err = verifierValid.VerifyPath(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, verifierValid.Violations())
}



