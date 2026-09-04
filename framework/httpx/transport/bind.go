package transport

import (
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gofiber/fiber/v3"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/bson"

	"github.com/king-glitch/hexag/framework/ports"
)

var commonDateLayouts = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// Bind binds query parameters on GET/DELETE/HEAD requests, and JSON body + query parameters on other methods.
func Bind(ctx fiber.Ctx, out any) error {
	method := ctx.Method()
	if method == fiber.MethodGet || method == fiber.MethodDelete || method == fiber.MethodHead {
		return BindQuery(ctx, out)
	}

	if len(ctx.Body()) > 0 {
		if err := ctx.Bind().Body(out); err != nil {
			return ports.NewServiceError(
				ports.ServiceErrorCodeValidation,
				errors.Wrap(err, "failed to parse request body"),
			)
		}
	}

	return BindQuery(ctx, out)
}

// BindQuery binds URL query parameters into the struct pointed to by out.
// Supports primitive types, pointers, slices, time.Time, and bson.ObjectID.
func BindQuery(ctx fiber.Ctx, out any) error {
	if out == nil {
		return ports.NewServiceError(ports.ServiceErrorCodeValidation, errors.New("bind target cannot be nil"))
	}

	rv := reflect.ValueOf(out)
	if rv.Kind() != reflect.Pointer || rv.Elem().Kind() != reflect.Struct {
		return ports.NewServiceError(ports.ServiceErrorCodeValidation, errors.New("bind target must be a pointer to a struct"))
	}

	elem := rv.Elem()
	elemType := elem.Type()

	queryMap := make(map[string][]string)
	for k, v := range ctx.Request().URI().QueryArgs().All() {
		key := strings.ToLower(string(k))
		for _, part := range strings.Split(string(v), ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				queryMap[key] = append(queryMap[key], part)
			}
		}
	}

	objectIDType := reflect.TypeOf(bson.ObjectID{})
	timeType := reflect.TypeOf(time.Time{})

	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		fieldVal := elem.Field(i)

		if !fieldVal.CanSet() {
			continue
		}

		candidates := candidateKeys(field)
		var values []string
		for _, candidate := range candidates {
			if vals, ok := queryMap[candidate]; ok && len(vals) > 0 {
				values = vals
				break
			}
		}

		if len(values) == 0 {
			if def := field.Tag.Get("default"); def != "" {
				values = []string{def}
			} else {
				continue
			}
		}

		firstVal := values[0]
		targetType := field.Type

		// 1. bson.ObjectID
		if targetType == objectIDType {
			id, err := bson.ObjectIDFromHex(firstVal)
			if err != nil {
				return ports.NewServiceError(
					ports.ServiceErrorCodeValidation,
					errors.Errorf("invalid ObjectID for %s", field.Name),
				).AddError(candidates[0], "must be a valid 24-character hexadecimal ObjectID", err)
			}
			fieldVal.Set(reflect.ValueOf(id))
			continue
		}

		// 2. *bson.ObjectID
		if targetType == reflect.PointerTo(objectIDType) {
			id, err := bson.ObjectIDFromHex(firstVal)
			if err != nil {
				return ports.NewServiceError(
					ports.ServiceErrorCodeValidation,
					errors.Errorf("invalid ObjectID for %s", field.Name),
				).AddError(candidates[0], "must be a valid 24-character hexadecimal ObjectID", err)
			}
			fieldVal.Set(reflect.ValueOf(&id))
			continue
		}

		// 3. []bson.ObjectID
		if targetType == reflect.SliceOf(objectIDType) {
			ids := make([]bson.ObjectID, 0, len(values))
			for _, v := range values {
				id, err := bson.ObjectIDFromHex(v)
				if err != nil {
					return ports.NewServiceError(
						ports.ServiceErrorCodeValidation,
						errors.Errorf("invalid ObjectID in slice for %s", field.Name),
					).AddError(candidates[0], "elements must be valid 24-character hexadecimal ObjectIDs", err)
				}
				ids = append(ids, id)
			}
			fieldVal.Set(reflect.ValueOf(ids))
			continue
		}

		// 4. time.Time
		if targetType == timeType {
			parsedTime, err := parseDateValue(firstVal)
			if err != nil {
				return ports.NewServiceError(
					ports.ServiceErrorCodeValidation,
					errors.Errorf("invalid date for %s", field.Name),
				).AddError(candidates[0], "must be an RFC3339 timestamp or YYYY-MM-DD date", err)
			}
			fieldVal.Set(reflect.ValueOf(parsedTime))
			continue
		}

		// 5. *time.Time
		if targetType == reflect.PointerTo(timeType) {
			parsedTime, err := parseDateValue(firstVal)
			if err != nil {
				return ports.NewServiceError(
					ports.ServiceErrorCodeValidation,
					errors.Errorf("invalid date for %s", field.Name),
				).AddError(candidates[0], "must be an RFC3339 timestamp or YYYY-MM-DD date", err)
			}
			fieldVal.Set(reflect.ValueOf(&parsedTime))
			continue
		}

		// 6. Slices (e.g. []string, []AuditAction, etc.)
		if targetType.Kind() == reflect.Slice {
			elemKind := targetType.Elem().Kind()
			if elemKind == reflect.String {
				sliceVal := reflect.MakeSlice(targetType, len(values), len(values))
				for idx, v := range values {
					sliceVal.Index(idx).Set(reflect.ValueOf(v).Convert(targetType.Elem()))
				}
				fieldVal.Set(sliceVal)
				continue
			}
		}

		// 7. Strings and custom string types
		if targetType.Kind() == reflect.String {
			fieldVal.Set(reflect.ValueOf(firstVal).Convert(targetType))
			continue
		}
		if targetType.Kind() == reflect.Pointer && targetType.Elem().Kind() == reflect.String {
			converted := reflect.ValueOf(firstVal).Convert(targetType.Elem())
			ptr := reflect.New(targetType.Elem())
			ptr.Elem().Set(converted)
			fieldVal.Set(ptr)
			continue
		}

		// 8. Booleans
		if targetType.Kind() == reflect.Bool {
			b, err := strconv.ParseBool(firstVal)
			if err == nil {
				fieldVal.SetBool(b)
			}
			continue
		}
		if targetType.Kind() == reflect.Pointer && targetType.Elem().Kind() == reflect.Bool {
			b, err := strconv.ParseBool(firstVal)
			if err == nil {
				fieldVal.Set(reflect.ValueOf(&b))
			}
			continue
		}

		// 9. Integers
		if isIntKind(targetType.Kind()) {
			n, err := strconv.ParseInt(firstVal, 10, 64)
			if err == nil {
				fieldVal.SetInt(n)
			}
			continue
		}
		if targetType.Kind() == reflect.Pointer && isIntKind(targetType.Elem().Kind()) {
			n, err := strconv.ParseInt(firstVal, 10, 64)
			if err == nil {
				ptr := reflect.New(targetType.Elem())
				ptr.Elem().SetInt(n)
				fieldVal.Set(ptr)
			}
			continue
		}

		// 10. Floats
		if targetType.Kind() == reflect.Float32 || targetType.Kind() == reflect.Float64 {
			f, err := strconv.ParseFloat(firstVal, 64)
			if err == nil {
				fieldVal.SetFloat(f)
			}
		}
	}

	if ctx.App() != nil && ctx.App().Config().StructValidator != nil {
		if err := ctx.App().Config().StructValidator.Validate(out); err != nil {
			var serr *ports.ServiceError
			if errors.As(err, &serr) {
				return serr
			}
			return ports.NewServiceError(ports.ServiceErrorCodeValidation, err)
		}
	}

	return nil
}

func isIntKind(k reflect.Kind) bool {
	return k == reflect.Int || k == reflect.Int8 || k == reflect.Int16 || k == reflect.Int32 || k == reflect.Int64
}

func parseDateValue(val string) (time.Time, error) {
	for _, layout := range commonDateLayouts {
		if t, err := time.Parse(layout, val); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.Errorf("unable to parse date %q", val)
}

func candidateKeys(field reflect.StructField) []string {
	keys := make([]string, 0, 4)
	seen := make(map[string]bool)

	addKey := func(k string) {
		k = strings.TrimSpace(strings.ToLower(k))
		if k == "" || k == "-" || seen[k] {
			return
		}
		seen[k] = true
		keys = append(keys, k)

		// Singular / plural variants
		if strings.HasSuffix(k, "s") && len(k) > 1 {
			singular := strings.TrimSuffix(k, "s")
			if !seen[singular] {
				seen[singular] = true
				keys = append(keys, singular)
			}
		} else {
			plural := k + "s"
			if !seen[plural] {
				seen[plural] = true
				keys = append(keys, plural)
			}
		}
	}

	if tag := field.Tag.Get("query"); tag != "" {
		addKey(strings.Split(tag, ",")[0])
	}
	if tag := field.Tag.Get("json"); tag != "" {
		addKey(strings.Split(tag, ",")[0])
	}
	if tag := field.Tag.Get("form"); tag != "" {
		addKey(strings.Split(tag, ",")[0])
	}

	snake := toSnakeCase(field.Name)
	addKey(snake)
	addKey(field.Name)

	return keys
}

func toSnakeCase(str string) string {
	var b strings.Builder
	runes := []rune(str)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if unicode.IsLower(prev) || nextIsLower {
				b.WriteRune('_')
			}
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}
