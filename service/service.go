package service

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/forbearing/gst/logger"
	"github.com/forbearing/gst/model"
	"github.com/forbearing/gst/types"
	"github.com/forbearing/gst/types/consts"
	"go.uber.org/zap"
)

var (
	serviceMap = make(map[string]any)
	mu         sync.Mutex

	_ types.Service[*model.User, any, any] = (*Base[*model.User, any, any])(nil)
)

var (
	ErrNotFoundService   = errors.New("no service instant matches the give Model interface, skip processing service layer")
	ErrNotFoundServiceID = errors.New("not found service id in assetIdMap")
)

func serviceKey[M types.Model](phase consts.Phase) string {
	typ := reflect.TypeOf(*new(M)).Elem()
	key := fmt.Sprintf("%s|%s|%s", typ.PkgPath(), typ.String(), phase)
	return key
}

// Register service instance.
//
// If Register called in "init" function, logger.Service is nil
// the service.Logger will be set in "Init".
//
// If Register called is not "init" function, such as "Init", logger.Service is not nil,
// set the service.Logger directly.
func Register[S types.Service[M, REQ, RSP], M types.Model, REQ types.Request, RSP types.Response](phase consts.Phase) {
	mu.Lock()
	defer mu.Unlock()
	key := serviceKey[M](phase)
	val := reflect.New(reflect.TypeOf(*new(S)).Elem()).Interface()
	setLogger(val)
	serviceMap[key] = val
}

func setLogger(s any) {
	// Check logger.Service is nil to avoid panic "panic: reflect: call of reflect.Value.Set on zero Value"
	// in statement "fieldLogger.Set(reflect.ValueOf(logger.Service))".
	if logger.Service == nil {
		return
	}
	typ := reflect.TypeOf(s).Elem()
	val := reflect.ValueOf(s).Elem()
	for i := range typ.NumField() {
		switch strings.ToLower(typ.Field(i).Name) {
		case "logger": // service object has itself types.Logger
			if val.Field(i).IsZero() {
				val.Field(i).Set(reflect.ValueOf(logger.Service))
			}
		case "base": // service object's types.Logger extends from 'base' struct.
			fieldLogger := val.Field(i).FieldByName("Logger")
			if fieldLogger.IsZero() {
				fieldLogger.Set(reflect.ValueOf(logger.Service))
			}
		}
	}
}

func Init() error {
	// Init all service types.Logger
	for _, s := range serviceMap {
		setLogger(s)
	}

	return nil
}

func Factory[M types.Model, REQ types.Request, RSP types.Response]() *factory[M, REQ, RSP] {
	return &factory[M, REQ, RSP]{}
}

// factory is a service factory used to product service instance.
// The service instance should registered by function `Register()` in init()
//
// The service defined by user should be unexported (structure name is lowercase).
// service instance are only returns by the `factory`.
type factory[M types.Model, REQ types.Request, RSP types.Response] struct{}

func (f factory[M, REQ, RSP]) Service(phase consts.Phase) types.Service[M, REQ, RSP] {
	svc, ok := serviceMap[serviceKey[M](phase)]
	if !ok {
		logger.Service.Debugz(ErrNotFoundService.Error(), zap.String("model", serviceKey[M](phase)))
		return new(Base[M, REQ, RSP])
	}
	return svc.(types.Service[M, REQ, RSP]) //nolint:errcheck
}

type Base[M types.Model, REQ types.Request, RSP types.Response] struct{ types.Logger }

func (Base[M, REQ, RSP]) Create(*types.ServiceContext, REQ) (RSP, error) { return *new(RSP), nil }
func (Base[M, REQ, RSP]) Delete(*types.ServiceContext, REQ) (RSP, error) { return *new(RSP), nil }
func (Base[M, REQ, RSP]) Update(*types.ServiceContext, REQ) (RSP, error) { return *new(RSP), nil }
func (Base[M, REQ, RSP]) Patch(*types.ServiceContext, REQ) (RSP, error)  { return *new(RSP), nil }
func (Base[M, REQ, RSP]) List(*types.ServiceContext, REQ) (RSP, error)   { return *new(RSP), nil }
func (Base[M, REQ, RSP]) Get(*types.ServiceContext, REQ) (RSP, error)    { return *new(RSP), nil }

func (Base[M, REQ, RSP]) CreateMany(*types.ServiceContext, REQ) (RSP, error) { return *new(RSP), nil }
func (Base[M, REQ, RSP]) DeleteMany(*types.ServiceContext, REQ) (RSP, error) { return *new(RSP), nil }
func (Base[M, REQ, RSP]) UpdateMany(*types.ServiceContext, REQ) (RSP, error) { return *new(RSP), nil }
func (Base[M, REQ, RSP]) PatchMany(*types.ServiceContext, REQ) (RSP, error)  { return *new(RSP), nil }

func (Base[M, REQ, RSP]) CreateBefore(*types.ServiceContext, M) error  { return nil }
func (Base[M, REQ, RSP]) CreateAfter(*types.ServiceContext, M) error   { return nil }
func (Base[M, REQ, RSP]) DeleteBefore(*types.ServiceContext, M) error  { return nil }
func (Base[M, REQ, RSP]) DeleteAfter(*types.ServiceContext, M) error   { return nil }
func (Base[M, REQ, RSP]) UpdateBefore(*types.ServiceContext, M) error  { return nil }
func (Base[M, REQ, RSP]) UpdateAfter(*types.ServiceContext, M) error   { return nil }
func (Base[M, REQ, RSP]) PatchBefore(*types.ServiceContext, M) error   { return nil }
func (Base[M, REQ, RSP]) PatchAfter(*types.ServiceContext, M) error    { return nil }
func (Base[M, REQ, RSP]) ListBefore(*types.ServiceContext, *[]M) error { return nil }
func (Base[M, REQ, RSP]) ListAfter(*types.ServiceContext, *[]M) error  { return nil }
func (Base[M, REQ, RSP]) GetBefore(*types.ServiceContext, M) error     { return nil }
func (Base[M, REQ, RSP]) GetAfter(*types.ServiceContext, M) error      { return nil }

func (Base[M, REQ, RSP]) CreateManyBefore(*types.ServiceContext, ...M) error { return nil }
func (Base[M, REQ, RSP]) CreateManyAfter(*types.ServiceContext, ...M) error  { return nil }
func (Base[M, REQ, RSP]) DeleteManyBefore(*types.ServiceContext, ...M) error { return nil }
func (Base[M, REQ, RSP]) DeleteManyAfter(*types.ServiceContext, ...M) error  { return nil }
func (Base[M, REQ, RSP]) UpdateManyBefore(*types.ServiceContext, ...M) error { return nil }
func (Base[M, REQ, RSP]) UpdateManyAfter(*types.ServiceContext, ...M) error  { return nil }
func (Base[M, REQ, RSP]) PatchManyBefore(*types.ServiceContext, ...M) error  { return nil }
func (Base[M, REQ, RSP]) PatchManyAfter(*types.ServiceContext, ...M) error   { return nil }

func (Base[M, REQ, RSP]) Import(*types.ServiceContext, io.Reader) ([]M, error) {
	return make([]M, 0), nil
}

func (Base[M, REQ, RSP]) Export(*types.ServiceContext, ...M) ([]byte, error) {
	return make([]byte, 0), nil
}

func (Base[M, REQ, RSP]) Filter(_ *types.ServiceContext, m M) M    { return m }
func (Base[M, REQ, RSP]) FilterRaw(_ *types.ServiceContext) string { return "" }
