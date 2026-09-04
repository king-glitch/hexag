//go:generate go run github.com/king-glitch/hexag/framework/cmd/mongogen -file=domain.go -out=../adapters/database/mongo/models -pkg=models

package ports

import (
	hexports "github.com/king-glitch/hexag/framework/ports"
)

// ExampleModel is a starter entity — rename it (and the example/ repository,
// service, and route package) to your first real domain model, or delete it
// once you've added your own.
type ExampleModel struct {
	hexports.ModelBase `bson:",inline" json:",inline"`

	Name string `json:"name" bson:"name"`
}

func (m ExampleModel) CollectionName() string {
	return "example"
}

func (m ExampleModel) WithBase(base hexports.ModelBase) ExampleModel {
	m.ModelBase = base
	return m
}

func (m ExampleModel) MarshalJSON() ([]byte, error) {
	type alias ExampleModel
	return hexports.MarshalOmitBase(m.ModelBase, alias(m))
}
