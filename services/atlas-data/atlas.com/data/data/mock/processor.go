package mock

import (
	"atlas-data/data"
)

type ProcessorMock struct {
	ProcessDataFunc      func() error
	InstructWorkerFunc   func(workerName string, path string) error
	StartWorkerFunc      func(name string, path string) error
	RegisterAllDataFunc  func(rootDir string, wzFileName string, rf data.RegisterFunc) data.Worker
	RegisterFileDataFunc func(rootDir string, wzFileName string, rf data.RegisterFunc) data.Worker
}

var _ data.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ProcessData() error {
	if m.ProcessDataFunc != nil {
		return m.ProcessDataFunc()
	}
	return nil
}

func (m *ProcessorMock) InstructWorker(workerName string, path string) error {
	if m.InstructWorkerFunc != nil {
		return m.InstructWorkerFunc(workerName, path)
	}
	return nil
}

func (m *ProcessorMock) StartWorker(name string, path string) error {
	if m.StartWorkerFunc != nil {
		return m.StartWorkerFunc(name, path)
	}
	return nil
}

func (m *ProcessorMock) RegisterAllData(rootDir string, wzFileName string, rf data.RegisterFunc) data.Worker {
	if m.RegisterAllDataFunc != nil {
		return m.RegisterAllDataFunc(rootDir, wzFileName, rf)
	}
	return func() error { return nil }
}

func (m *ProcessorMock) RegisterFileData(rootDir string, wzFileName string, rf data.RegisterFunc) data.Worker {
	if m.RegisterFileDataFunc != nil {
		return m.RegisterFileDataFunc(rootDir, wzFileName, rf)
	}
	return func() error { return nil }
}
