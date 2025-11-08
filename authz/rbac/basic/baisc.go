package basic

import (
	"os"
	"path/filepath"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/authz/rbac"
	"github.com/forbearing/gst/config"
	"github.com/forbearing/gst/database"
	modelauthz "github.com/forbearing/gst/internal/model/authz"
	"github.com/forbearing/gst/logger"
)

const (
	Root = "root"
)

var adminRole = "admin"

var defaultAdmins = []string{
	"admin",
	"root",
}

var modelData = []byte(`
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act, eft

[role_definition]
g = _, _

[policy_effect]
#e = priority(p.eft) || some(where (p.eft == allow))
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, "admin") || (g(r.sub, p.sub) && keyMatch3(r.obj, p.obj) && r.act == p.act)
`)

func Init() (err error) {
	if !config.App.Auth.RBACEnable {
		return nil
	}

	filename := filepath.Join(config.Tempdir(), "casbin_model.conf")
	if err = os.WriteFile(filename, modelData, 0o600); err != nil {
		return errors.Wrapf(err, "failed to write model file %s", filename)
	}
	// NOTE: gormadapter.NewAdapterByDBWithCustomTable creates the Casbin policy table with an auto-incrementing primary key.
	if rbac.Adapter, err = gormadapter.NewAdapterByDBWithCustomTable(database.DB, new(modelauthz.CasbinRule)); err != nil {
		return errors.Wrap(err, "failed to create casbin adapter")
	}
	if rbac.Enforcer, err = casbin.NewEnforcer(filename, rbac.Adapter); err != nil {
		return errors.Wrap(err, "failed to create casbin enforcer")
	}

	rbac.Enforcer.SetLogger(logger.Casbin)
	rbac.Enforcer.EnableLog(true)
	rbac.Enforcer.EnableAutoSave(true)
	rbac.Enforcer.EnableAutoNotifyDispatcher(true)
	rbac.Enforcer.EnableAutoNotifyWatcher(true)
	rbac.Enforcer.EnableEnforce(true)

	for _, user := range defaultAdmins {
		_, _ = rbac.Enforcer.AddGroupingPolicy(user, adminRole)
	}

	return rbac.Enforcer.LoadPolicy()
}
