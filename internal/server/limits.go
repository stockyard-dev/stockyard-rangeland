package server

import "github.com/stockyard-dev/stockyard-rangeland/internal/license"

type Limits struct {
	MaxSites            int  // 0 = unlimited
	MaxPageviewsMonth   int  // 0 = unlimited
	RetentionDays       int
	RealTimeVisitors    bool
	ExportData          bool
	CustomDashboard     bool
	APIAccess           bool
}

var freeLimits = Limits{
	MaxSites:          1,
	MaxPageviewsMonth: 10000,
	RetentionDays:     7,
	RealTimeVisitors:  true, // free hook
	ExportData:        false,
	CustomDashboard:   false,
	APIAccess:         false,
}

var proLimits = Limits{
	MaxSites:          0,
	MaxPageviewsMonth: 0,
	RetentionDays:     365,
	RealTimeVisitors:  true,
	ExportData:        true,
	CustomDashboard:   true,
	APIAccess:         true,
}

func LimitsFor(info *license.Info) Limits {
	if info != nil && info.IsPro() {
		return proLimits
	}
	return freeLimits
}

func LimitReached(limit, current int) bool {
	if limit == 0 {
		return false
	}
	return current >= limit
}
