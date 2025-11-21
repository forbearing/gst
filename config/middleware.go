package config

const (
	MIDDLEWARE_ENABLE_JWT_AUTH = "MIDDLEWARE_ENABLE_JWT_AUTH" //nolint:staticcheck
	MIDDLEWARE_ENABLE_AUTHZ    = "MIDDLEWARE_ENABLE_AUTHZ"    //nolint:staticcheck
)

type Middleware struct {
	EnableJwtAuth bool `json:"enable_jwt_auth" mapstructure:"enable_jwt_auth" ini:"enable_jwt_auth" yaml:"enable_jwt_auth"`
	EnableAuthz   bool `json:"enable_authz" mapstructure:"enable_authz" ini:"enable_authz" yaml:"enable_authz"`
}

func (*Middleware) setDefault() {
	cv.SetDefault("middleware.enable_jwt_auth", false)
	cv.SetDefault("middleware.enable_authz", false)
}
