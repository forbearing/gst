package iam

type Config struct {
	EnableTenant bool // default disable tenant.
}

func Register(...Config) {
}
