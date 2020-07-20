package cgroups

import "github.com/lipeining/godocker/configs"

type Manager struct {
	config *configs.Config
}

func NewManager(config *configs.Config) (Manager, error) {
	return Manager{config}, nil
}

func (m *Manager) Apply(pid int) error {
	return nil
}
