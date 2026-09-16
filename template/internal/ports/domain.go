//go:generate go run github.com/king-glitch/hexag/framework/cmd/mongogen -file=domain.go -out=../adapters/database/mongo/models -pkg=models

package ports

import (
	modelbase "github.com/king-glitch/hexag/framework/api/model/base"
	hexports "github.com/king-glitch/hexag/framework/ports"
)

// ExampleModel is a starter entity — rename it (and the example/ repository,
// service, and route package) to your first real domain model, or delete it
// once you've added your own.
type ExampleModel struct {
	modelbase.ModelBase `bson:",inline" json:",inline"`

	Name string `json:"name" bson:"name"`
}

func (m ExampleModel) CollectionName() string {
	return "example"
}

func (m ExampleModel) WithBase(base hexports.ModelBase) ExampleModel {
	m.ModelBase = m.ModelBase.WithBase(base)
	return m
}

func (m ExampleModel) MarshalJSON() ([]byte, error) {
	type alias ExampleModel
	return modelbase.MarshalOmitBase(m.ModelBase, alias(m))
}
