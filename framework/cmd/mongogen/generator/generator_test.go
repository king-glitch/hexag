package generator

import (
	"strings"
	"testing"

	"github.com/king-glitch/hexag/framework/cmd/mongogen/parser"
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

	// Verify struct type exists and is named UserModel
	require.Contains(t, code, "type UserModel struct {")
	require.Contains(t, code, "var User = UserModel{")

	// Verify no blank lines between field definitions
	expectedStructDef := `type UserModel struct {
	ID       mongo.Field[bson.ObjectID]
	Username mongo.Field[string]
	Email    mongo.Field[string]
}`
	require.Contains(t, code, expectedStructDef)

	// Verify no blank lines between field initializations
	expectedVarInit := `var User = UserModel{
	ID:       mongo.NewField[bson.ObjectID]("_id"),
	Username: mongo.NewField[string]("username"),
	Email:    mongo.NewField[string]("email"),
}`
	require.Contains(t, code, expectedVarInit)

	// Verify standalone mode
	standaloneContent, err := gen.GenerateModel(meta, "models", true)
	require.NoError(t, err)

	standaloneCode := string(standaloneContent)
	require.Contains(t, standaloneCode, "type UserModel struct {")
	require.Contains(t, standaloneCode, "var User = UserModel{")
	require.Contains(t, standaloneCode, "ID:       Field[bson.ObjectID]{key: \"_id\"},")
	require.False(t, strings.Contains(standaloneCode, "\n\n\tUsername:"), "should not have blank lines between fields")
}

func TestGenerateModel_NestedStructType(t *testing.T) {
	gen, err := NewGenerator()
	require.NoError(t, err)

	meta := parser.StructMeta{
		PackageName: "models",
		StructName:  "BotConnectionModel",
		VarName:     "BotConnection",
		Fields: []parser.FieldMeta{
			{GoName: "ID", GoType: "bson.ObjectID", BsonKey: "_id", FieldType: "Field[bson.ObjectID]"},
			{
				GoName:               "Stats",
				GoType:               "ports.BotConnectionStats",
				BsonKey:              "stats",
				FieldType:            "Field[ports.BotConnectionStats]",
				NestedStructTypeName: "BotConnectionStatsModel",
				SubFields: []parser.FieldMeta{
					{GoName: "Kills", GoType: "int", BsonKey: "stats.kills", FieldType: "Field[int]"},
					{GoName: "Deaths", GoType: "int", BsonKey: "stats.deaths", FieldType: "Field[int]"},
				},
			},
		},
	}

	content, err := gen.GenerateModel(meta, "models", false)
	require.NoError(t, err)

	code := string(content)

	// Verify nested struct type is declared separately (not anonymous)
	expectedNestedType := `type BotConnectionStatsModel struct {
	mongo.Field[ports.BotConnectionStats]
	Kills  mongo.Field[int]
	Deaths mongo.Field[int]
}`
	require.Contains(t, code, expectedNestedType)

	// Verify parent struct references the named nested struct type
	expectedParentDef := `type BotConnectionModel struct {
	ID    mongo.Field[bson.ObjectID]
	Stats BotConnectionStatsModel
}`
	require.Contains(t, code, expectedParentDef)

	// Verify initialization uses the named nested struct type
	expectedInit := `var BotConnection = BotConnectionModel{
	ID: mongo.NewField[bson.ObjectID]("_id"),
	Stats: BotConnectionStatsModel{
		Field:  mongo.NewField[ports.BotConnectionStats]("stats"),
		Kills:  mongo.NewField[int]("stats.kills"),
		Deaths: mongo.NewField[int]("stats.deaths"),
	},
}`
	require.Contains(t, code, expectedInit)
}


