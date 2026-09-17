package generator

import (
	"strings"
	"testing"

	"github.com/king-glitch/hexag/framework/cmd/mongogen/internal/parser"
	"github.com/stretchr/testify/require"
)

func TestGenerateModel_StructTypeAndNoBlankLines(t *testing.T) {
	gen, err := NewGenerator()
	require.NoError(t, err)

	meta := parser.StructMeta{
		PackageName: "models",
		StructName:  "UserModel",
		VarName:     "User",
		Fields: []parser.FieldMeta{
			{GoName: "ID", GoType: "bson.ObjectID", BsonKey: "_id", FieldType: "Field[bson.ObjectID]"},
			{GoName: "Username", GoType: "string", BsonKey: "username", FieldType: "Field[string]"},
			{GoName: "Email", GoType: "string", BsonKey: "email", FieldType: "Field[string]"},
		},
	}

	content, err := gen.GenerateModel(meta, "models", false)
	require.NoError(t, err)

	code := string(content)

	// Verify struct type exists and is named UserModelType
	require.Contains(t, code, "type UserModelType struct {")
	require.Contains(t, code, "var UserModel = UserModelType{")
	require.Contains(t, code, "var User = UserModel")

	// Verify no blank lines between field definitions
	expectedStructDef := `type UserModelType struct {
	ID       mongo.Field[bson.ObjectID]
	Username mongo.Field[string]
	Email    mongo.Field[string]
}`
	require.Contains(t, code, expectedStructDef)

	// Verify no blank lines between field initializations
	expectedVarInit := `var UserModel = UserModelType{
	ID:       mongo.NewField[bson.ObjectID]("_id"),
	Username: mongo.NewField[string]("username"),
	Email:    mongo.NewField[string]("email"),
}`
	require.Contains(t, code, expectedVarInit)

	// Verify standalone mode
	standaloneContent, err := gen.GenerateModel(meta, "models", true)
	require.NoError(t, err)

	standaloneCode := string(standaloneContent)
	require.Contains(t, standaloneCode, "type UserModelType struct {")
	require.Contains(t, standaloneCode, "var UserModel = UserModelType{")
	require.Contains(t, standaloneCode, "ID:       Field[bson.ObjectID]{key: \"_id\"},")
	require.False(t, strings.Contains(standaloneCode, "\n\n\tUsername:"), "should not have blank lines between fields")
}
