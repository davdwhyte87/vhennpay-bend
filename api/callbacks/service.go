package callbacks

import "vhennpay-bend/dao"

// Service represents the Callbacks Service
type Service struct {
	factoryDAO *dao.FactoryDAO
}

// NewCallbacksService returns a new callbacks service
func NewCallbacksService(factoryDAO *dao.FactoryDAO) *Service {
	return &Service{factoryDAO: factoryDAO}
}
