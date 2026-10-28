package pg_cdi

import (
	"encoding/json"

    "github.com/go-playground/validator/v10"
	"common/models/extra_parameters"
)

var validate *validator.Validate

func init() {
    validate = validator.New(validator.WithRequiredStructEnabled())
}

// ParseExtraParameter parses JSON string from extra_parameter field
func ParseExtraParameter(extraParamStr string) (*extra_parameters.PgCDIExtraParameters, error) {

	var param extra_parameters.PgCDIExtraParameters
	if err := json.Unmarshal([]byte(extraParamStr), &param); err != nil {
		return nil, err
	}

	// validation
    if err := validate.Struct(&param); err != nil {
		return nil, err
	}

	return &param, nil
}
