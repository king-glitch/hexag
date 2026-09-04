package transport

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/pkg/errors"

	"github.com/king-glitch/hexag/framework/ports"
)

func ParamInt(ctx fiber.Ctx, name string) (int, error) {
	value, err := strconv.Atoi(ctx.Params(name))
	if err != nil {
		return 0, ports.NewServiceError(
			ports.ServiceErrorCodeValidation,
			errors.Wrapf(err, "invalid %s path param", name),
		).AddError(name, "this field must be a valid integer", err)
	}

	return value, nil
}

// dateRangeLayouts are tried in order so that a caller can pass either a full
// timestamp or the plain "2026-08-01" a date input produces.
var dateRangeLayouts = []string{time.RFC3339, "2006-01-02"}

func parseDateParam(ctx fiber.Ctx, name string) (*time.Time, error) {
	raw := strings.TrimSpace(ctx.Query(name))
	if raw == "" {
		return nil, nil
	}

	for _, layout := range dateRangeLayouts {
		parsed, err := time.Parse(layout, raw)
		if err != nil {
			continue
		}

		return &parsed, nil
	}

	return nil, ports.NewServiceError(
		ports.ServiceErrorCodeValidation,
		errors.Errorf("invalid %s query param", name),
	).AddError(name, "this field must be an RFC3339 timestamp or a YYYY-MM-DD date", nil)
}

// ParseDateRange reads a half-open [from, to) range from "from"/"to" query
// params. Both bounds are optional.
func ParseDateRange(ctx fiber.Ctx) (*time.Time, *time.Time, error) {
	from, err := parseDateParam(ctx, "from")
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse from query param")
	}

	to, err := parseDateParam(ctx, "to")
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse to query param")
	}

	return from, to, nil
}

type Params struct {
	Page   int `json:"page"`
	Amount int `json:"amount"`

	Queries map[string]string `json:"queries"`
	Sorts   map[string]string `json:"sorts"`
}

func (p Params) SafePage() int {
	if p.Page < 1 {
		return 1
	}
	return p.Page
}

func (p Params) SafeAmount() int {
	if p.Amount < 1 {
		return 10
	}
	return p.Amount
}

func (p Params) Offset() int {
	return (p.SafePage() - 1) * p.SafeAmount()
}

func ParamsToPortsParams(p Params) ports.PaginationParams {
	return ports.PaginationParams{
		Page:    p.Page,
		Amount:  p.Amount,
		Queries: p.Queries,
		Sorts:   p.Sorts,
	}
}

var reservedParamKeys = map[string]bool{
	"page":   true,
	"amount": true,
	"sorts":  true,
}

func ParseParams(ctx fiber.Ctx) Params {
	params := Params{
		Page:    1,
		Amount:  10,
		Queries: make(map[string]string),
		Sorts:   make(map[string]string),
	}

	valuesByKey := make(map[string][]string)

	for k, v := range ctx.Request().URI().QueryArgs().All() {
		key := string(k)
		for _, part := range strings.Split(string(v), ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				valuesByKey[key] = append(valuesByKey[key], trimmed)
			}
		}
	}

	for key, values := range valuesByKey {
		if reservedParamKeys[key] {
			continue
		}
		params.Queries[key] = strings.Join(values, ",")
	}

	if page := ctx.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			params.Page = p
		}
	}

	if amount := ctx.Query("amount"); amount != "" {
		if a, err := strconv.Atoi(amount); err == nil {
			params.Amount = a
		}
	}

	params.Page = params.SafePage()
	params.Amount = params.SafeAmount()

	if sorts := ctx.Query("sorts"); sorts != "" {
		for _, sort := range strings.Split(sorts, ",") {
			parts := strings.Split(sort, ":")
			if len(parts) == 2 {
				params.Sorts[parts[0]] = parts[1]
			} else if len(parts) == 1 {
				params.Sorts[parts[0]] = "desc"
			}
		}
	}

	return params
}

// ParsePaginationParamsContext parses pagination parameters directly from fiber.Ctx into ports.PaginationParams.
// If defaultAmount is provided and the amount query parameter is omitted, defaultAmount[0] is used.
func ParsePaginationParamsContext(ctx fiber.Ctx, defaultAmount ...int) ports.PaginationParams {
	p := ParseParams(ctx)
	if len(defaultAmount) > 0 && ctx.Query("amount") == "" {
		p.Amount = defaultAmount[0]
	}
	return ParamsToPortsParams(p)
}

// ParsePaginationParams is an alias for ParsePaginationParamsContext.
func ParsePaginationParams(ctx fiber.Ctx, defaultAmount ...int) ports.PaginationParams {
	return ParsePaginationParamsContext(ctx, defaultAmount...)
}

// RequestTime returns the request's server time (in UTC).
// If a server time was set in locals, it returns that value;
// otherwise it falls back to time.Now().UTC().
func RequestTime(ctx fiber.Ctx) time.Time {
	if val, ok := ctx.Locals("server_time").(time.Time); ok {
		return val
	}
	return time.Now().UTC()
}
