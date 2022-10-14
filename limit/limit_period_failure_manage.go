package limit

import (
	"context"
	"sync"
	"time"
)

const unsupportedPeriodFailureLimitKind = "__unsupported_period_failure_limit_kind__"

// PeriodFailureLimitDriver driver interface
type PeriodFailureLimitDriver interface {
	// CheckErr requests a permit state.
	// same as Check
	CheckErr(ctx context.Context, key string, err error) (PeriodFailureLimitState, error)
	// Check requests a permit.
	Check(ctx context.Context, key string, success bool) (PeriodFailureLimitState, error)
	// SetQuotaFull set a permit over quota.
	SetQuotaFull(ctx context.Context, key string) error
	// Del delete a permit
	Del(ctx context.Context, key string) error
	// TTL get key ttl
	// if key not exist, time = -1.
	// if key exist, but not set expire time, t = -2
	TTL(ctx context.Context, key string) (time.Duration, error)
	// GetInt get current failure count
	GetInt(ctx context.Context, key string) (int, bool, error)
}

// PeriodFailureLimitManage manager limit period failure
type PeriodFailureLimitManage struct {
	mu     sync.RWMutex
	driver map[string]PeriodFailureLimitDriver
}

// NewPeriodFailureLimitManage new a instance
func NewPeriodFailureLimitManage() *PeriodFailureLimitManage {
	return &PeriodFailureLimitManage{
		driver: map[string]PeriodFailureLimitDriver{
			unsupportedPeriodFailureLimitKind: new(UnsupportedPeriodFailureLimitDriver),
		},
	}
}

// NewPeriodFailureLimitManageWithDriver new a instance with driver
func NewPeriodFailureLimitManageWithDriver(drivers map[string]PeriodFailureLimitDriver) *PeriodFailureLimitManage {
	p := &PeriodFailureLimitManage{
		driver: map[string]PeriodFailureLimitDriver{
			unsupportedPeriodFailureLimitKind: new(UnsupportedPeriodFailureLimitDriver),
		},
	}
	for kind, drive := range drivers {
		p.driver[kind] = drive
	}
	return p
}

// PeriodFailureLimitManage register a PeriodFailureLimitDriver with kind.
func (p *PeriodFailureLimitManage) RegisterDriver(kind string, d PeriodFailureLimitDriver) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.driver[kind]
	if ok {
		return ErrDuplicateDriver
	}
	p.driver[kind] = d
	return nil
}

// AcquireDriver acquire driver. if driver not exist. it will return UnsupportedPeriodFailureLimitDriver.
func (p *PeriodFailureLimitManage) AcquireDriver(kind string) PeriodFailureLimitDriver {
	p.mu.RLock()
	defer p.mu.RUnlock()
	d, ok := p.driver[kind]
	if ok {
		return d
	}
	return p.driver[unsupportedPeriodFailureLimitKind]
}

// UnsupportedPeriodFailureLimitDriver unsupported limit period failure driver
type UnsupportedPeriodFailureLimitDriver struct{}

func (UnsupportedPeriodFailureLimitDriver) CheckErr(ctx context.Context, key string, err error) (PeriodFailureLimitState, error) {
	return PeriodFailureLimitStsUnknown, ErrUnsupportedDriver
}
func (UnsupportedPeriodFailureLimitDriver) Check(context.Context, string, bool) (PeriodFailureLimitState, error) {
	return PeriodFailureLimitStsUnknown, ErrUnsupportedDriver
}
func (UnsupportedPeriodFailureLimitDriver) SetQuotaFull(context.Context, string) error {
	return ErrUnsupportedDriver
}
func (UnsupportedPeriodFailureLimitDriver) Del(context.Context, string) error {
	return ErrUnsupportedDriver
}
func (UnsupportedPeriodFailureLimitDriver) TTL(context.Context, string) (time.Duration, error) {
	return 0, ErrUnsupportedDriver
}
func (UnsupportedPeriodFailureLimitDriver) GetInt(context.Context, string) (int, bool, error) {
	return 0, false, ErrUnsupportedDriver
}
